package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/db"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
)

// executeFlowProcess spawns and monitors a Python flow process
func (ws *WorkerService) executeFlowProcess(ctx context.Context, event *service.FlowRunExecuteEventMessage) {
	// Prepare code location (local/S3)
	workingDir, fileName, cleanup, err := ws.prepareCodeLocation(event.CodeLocation)
	if err != nil {
		ws.log.Error("Failed to prepare code location", "error", err, "flow_run_id", event.FlowRunId)
		ws.reportFlowRunStatus(event.FlowRunId, "FAILED", err.Error())
		return
	}

	// Build command
	cmd, err := ws.buildCommand(event, workingDir, fileName)
	if err != nil {
		ws.log.Error("Failed to build Python command", "error", err, "flow_run_id", event.FlowRunId)
		ws.reportFlowRunStatus(event.FlowRunId, "FAILED", err.Error())
		cleanup() // Cleanup before returning on error
		return
	}

	// Start the process
	ws.log.Info("Starting flow process",
		"flow_run_id", event.FlowRunId,
		"entrypoint", event.Entrypoint,
		"args", event.Args,
		"working_dir", workingDir,
	)
	// Setup to stream stdout and stderr to console
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		ws.log.Error("Failed to start Python process", "error", err, "flow_run_id", event.FlowRunId)
		ws.reportFlowRunStatus(event.FlowRunId, "FAILED", err.Error())
		cleanup() // Cleanup before returning on error
		return
	}

	// Monitor the process in a separate goroutine
	// Pass cleanup function to be called after process completes
	go ws.monitorProcess(ctx, cmd, event.FlowRunId, cleanup)
}

// prepareCodeLocation handles code location preparation (local files, S3 downloads)
func (ws *WorkerService) prepareCodeLocation(codeLocation string) (workingDir string, fileName string, cleanup func(), err error) {
	if codeLocation == "" {
		// No specific location, use current directory
		cwd, err := os.Getwd()
		return cwd, "flow.py", func() {}, err
	}

	if strings.HasPrefix(codeLocation, "s3://") {
		// Download from S3 to temp directory
		return ws.downloadFromS3(codeLocation)
	}

	// Local file - get absolute path and directory
	absPath, err := filepath.Abs(codeLocation)
	if err != nil {
		return "", "", func() {}, fmt.Errorf("failed to get absolute path for %s: %w", codeLocation, err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		ws.log.Warn("Code location file does not exist, using directory", "path", absPath)
	}

	return filepath.Dir(absPath), filepath.Base(absPath), func() {}, nil
}

// downloadFromS3 downloads a file from S3 to a temporary directory
func (ws *WorkerService) downloadFromS3(s3Path string) (workingDir string, fileName string, cleanup func(), err error) {
	// Parse S3 URL: s3://bucket/path/to/file.py
	parsedURL, err := url.Parse(s3Path)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid S3 URL: %w", err)
	}

	bucket := parsedURL.Host
	key := strings.TrimPrefix(parsedURL.Path, "/")

	if bucket == "" || key == "" {
		return "", "", nil, fmt.Errorf("invalid S3 URL format, expected s3://bucket/key: %s", s3Path)
	}

	// Get S3 configuration from service config
	var s3Config *service.S3Config
	if ws.config != nil && ws.config.Storage != nil && ws.config.Storage.S3 != nil {
		s3Config = ws.config.Storage.S3
	} else {
		return "", "", nil, fmt.Errorf("S3 configuration not found in service config")
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "flow-*")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cleanup = func() {
		os.RemoveAll(tempDir)
		ws.log.Debug("Cleaned up temporary directory", "path", tempDir)
	}

	// Create AWS config with appropriate credential provider and endpoint
	var configOptions []func(*config.LoadOptions) error

	// Add region
	configOptions = append(configOptions, config.WithRegion(s3Config.Region))

	// Configure credentials based on type
	switch s3Config.CredentialType {
	case "static":
		if s3Config.AccessKeyID == "" || s3Config.SecretAccessKey == "" {
			cleanup()
			return "", "", nil, fmt.Errorf("access_key_id and secret_access_key required for static credentials")
		}
		configOptions = append(configOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				s3Config.AccessKeyID,
				s3Config.SecretAccessKey,
				"", // token (empty for basic access keys)
			),
		))
	case "assume_role":
		if s3Config.AssumeRoleARN == "" {
			cleanup()
			return "", "", nil, fmt.Errorf("assume_role_arn required for assume_role credentials")
		}
		// Load default config first to get base credentials for assume role
		baseCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s3Config.Region))
		if err != nil {
			cleanup()
			return "", "", nil, fmt.Errorf("failed to load base AWS config for assume role: %w", err)
		}

		// Create STS client and assume role credentials
		stsClient := sts.NewFromConfig(baseCfg)
		sessionName := s3Config.AssumeRoleSession
		if sessionName == "" {
			sessionName = "pinazu-worker-session"
		}

		assumeRoleCreds := stscreds.NewAssumeRoleProvider(stsClient, s3Config.AssumeRoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = sessionName
		})

		configOptions = append(configOptions, config.WithCredentialsProvider(assumeRoleCreds))
	case "default", "":
		// Use default credential chain (environment, instance profile, etc.)
		// No additional credential provider needed
	default:
		cleanup()
		return "", "", nil, fmt.Errorf("unsupported credential_type: %s (supported: static, assume_role, default)", s3Config.CredentialType)
	}

	// Configure custom endpoint if provided (for MinIO/S3-compatible services)
	if s3Config.EndpointURL != "" {
		configOptions = append(configOptions, config.WithBaseEndpoint(s3Config.EndpointURL))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), configOptions...)
	if err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with path-style configuration
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = s3Config.UsePathStyle
	})

	ws.log.Info("Downloading file from S3",
		"s3_path", s3Path,
		"bucket", bucket,
		"key", key,
		"temp_dir", tempDir,
		"endpoint_url", s3Config.EndpointURL,
		"use_path_style", s3Config.UsePathStyle,
		"credential_type", s3Config.CredentialType,
		"region", s3Config.Region)

	// Download object
	resp, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	defer resp.Body.Close()

	// Extract filename and create local file
	filename := filepath.Base(key)
	if filename == "." || filename == "/" {
		// If key ends with /, use a default filename
		filename = "flow.py"
	}
	localPath := filepath.Join(tempDir, filename)

	localFile, err := os.Create(localPath)
	if err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Copy S3 content to local file
	_, err = io.Copy(localFile, resp.Body)
	if err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("failed to write file content: %w", err)
	}

	ws.log.Info("Successfully downloaded file from S3",
		"s3_path", s3Path,
		"local_path", localPath,
		"temp_dir", tempDir)

	return tempDir, filename, cleanup, nil
}

// buildCommand constructs the command with parameters using flexible entrypoint and args
func (ws *WorkerService) buildCommand(event *service.FlowRunExecuteEventMessage, workingDir string, fileName string) (*exec.Cmd, error) {
	// Start with the provided args
	args := make([]string, len(event.Args))
	copy(args, event.Args)
	args = append(args, fileName) // Append the file name to args

	// Add regular parameters
	for key, value := range event.Parameters {
		args = append(args, fmt.Sprintf("--%s=%v", key, value))
	}

	// Add success_task_results if present (for retry scenarios)
	if len(event.SuccessTaskResults) > 0 {
		resultsJSON, err := json.Marshal(event.SuccessTaskResults)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal success_task_results: %w", err)
		}
		args = append(args, "--success-task-results", string(resultsJSON))
	}

	cmd := exec.Command(event.Entrypoint, args...)
	cmd.Dir = workingDir

	// Get NATS URL from NATS connection
	natsURL := "nats://localhost:4222" // default
	if ws.s.GetNATS() != nil && len(ws.s.GetNATS().Servers()) > 0 {
		natsURL = ws.s.GetNATS().Servers()[0]
	}
	// Log out the args
	ws.log.Info("Building command for flow process",
		"flow_run_id", event.FlowRunId,
		"entrypoint", event.Entrypoint,
		"args", strings.Join(args, " "),
		"working_dir", workingDir,
		"nats_url", natsURL,
	)

	// Set environment variables
	envVars := []string{
		fmt.Sprintf("FLOW_RUN_ID=%s", event.FlowRunId),
		fmt.Sprintf("NATS_URL=%s", natsURL),
	}

	// Add cache configuration based on config settings
	if ws.config != nil && ws.config.Cache != nil {
		cacheConfig := ws.config.Cache

		// Set cache type based on configuration
		switch cacheConfig.Type {
		case service.CacheTypeS3:
			// Enable S3 caching - validation ensures S3 storage is configured
			envVars = append(envVars,
				"ENABLE_S3_CACHING=true",
				fmt.Sprintf("CACHE_BUCKET=%s", cacheConfig.Bucket),
			)

			// Also pass S3 configuration for the Python CacheManager
			s3Config := ws.config.Storage.S3
			envVars = append(envVars,
				fmt.Sprintf("AWS_ENDPOINT_URL=%s", s3Config.EndpointURL),
				fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", s3Config.AccessKeyID),
				fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", s3Config.SecretAccessKey),
				fmt.Sprintf("AWS_DEFAULT_REGION=%s", s3Config.Region),
			)

			// Set path style for MinIO
			if s3Config.UsePathStyle {
				envVars = append(envVars, "AWS_S3_USE_PATH_STYLE=true")
			}

			ws.log.Info("Flow execution: Using S3 cache",
				"flow_run_id", event.FlowRunId,
				"cache_type", "s3",
				"bucket", cacheConfig.Bucket,
				"endpoint", s3Config.EndpointURL)

		case service.CacheTypeMemory:
			// Explicitly disable S3 caching for memory cache
			envVars = append(envVars, "ENABLE_S3_CACHING=false")
			ws.log.Info("Flow execution: Using memory cache",
				"flow_run_id", event.FlowRunId,
				"cache_type", "memory",
				"note", "Cache will be cleared when flow completes")

		default:
			// This should not happen due to validation, but handle it gracefully
			ws.log.Error("Invalid cache type configured", "type", cacheConfig.Type)
			return nil, fmt.Errorf("invalid cache type: %s. Valid options are: %s, %s",
				cacheConfig.Type, service.CacheTypeMemory, service.CacheTypeS3)
		}
	} else {
		// No cache config specified - fall back to old behavior for backward compatibility
		if ws.config != nil && ws.config.Storage != nil && ws.config.Storage.S3 != nil {
			envVars = append(envVars,
				"ENABLE_S3_CACHING=true",
				"CACHE_BUCKET=flow-cache-bucket", // default bucket name
			)
			s3Config := ws.config.Storage.S3
			envVars = append(envVars,
				fmt.Sprintf("AWS_ENDPOINT_URL=%s", s3Config.EndpointURL),
				fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", s3Config.AccessKeyID),
				fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", s3Config.SecretAccessKey),
				fmt.Sprintf("AWS_DEFAULT_REGION=%s", s3Config.Region),
			)
			if s3Config.UsePathStyle {
				envVars = append(envVars, "AWS_S3_USE_PATH_STYLE=true")
			}
			ws.log.Info("Flow execution: Using S3 cache (backward compatibility)",
				"flow_run_id", event.FlowRunId,
				"cache_type", "s3_default",
				"bucket", "flow-cache-bucket",
				"endpoint", s3Config.EndpointURL,
				"note", "No cache config specified, using S3 because storage is configured")
		} else {
			envVars = append(envVars, "ENABLE_S3_CACHING=false")
			ws.log.Info("Flow execution: Using memory cache (backward compatibility)",
				"flow_run_id", event.FlowRunId,
				"cache_type", "memory_default",
				"note", "No cache config specified and S3 not available")
		}
	}

	cmd.Env = append(os.Environ(), envVars...)
	return cmd, nil
}

// monitorProcess waits for the process to complete and handles errors
func (ws *WorkerService) monitorProcess(ctx context.Context, cmd *exec.Cmd, flowRunID uuid.UUID, cleanup func()) {
	// Wait for process completion or context cancellation
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	//cleanup when function exits
	defer cleanup()

	select {
	case <-ctx.Done():
		// Context cancelled, kill the process
		ws.log.Warn("Context cancelled, terminating flow process", "flow_run_id", flowRunID)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		ws.reportFlowRunStatus(flowRunID, "FAILED", "Process cancelled due to context cancellation")

	case err := <-done:
		if err != nil {
			// Process terminated abnormally
			ws.log.Error("Flow process terminated abnormally",
				"error", err,
				"flow_run_id", flowRunID,
			)
			ws.reportFlowRunStatus(flowRunID, "FAILED", err.Error())
		} else {
			// Process completed successfully - Flow library handles RUNNING/SUCCESS reporting
			ws.log.Info("Flow process completed successfully", "flow_run_id", flowRunID)
		}
	}
}

// reportFlowRunStatus sends a FlowRunStatusEvent to the Orchestrator via JetStream
func (ws *WorkerService) reportFlowRunStatus(flowRunID uuid.UUID, status db.FlowStatus, errorMessage ...string) {
	// Convert string FlowRunId to uuid.UUID
	// Extract error message if provided
	var errMsg string
	if len(errorMessage) > 0 {
		errMsg = errorMessage[0]
	}

	// Create the event using the correct structure expected by flows service
	eventMessage := &service.FlowRunStatusEventMessage{
		FlowRunId:      flowRunID,
		Status:         status,
		EventTimestamp: time.Now().UTC(),
		ErrorMessage:   errMsg,
	}

	// Create the service event wrapper
	event := service.Event[*service.FlowRunStatusEventMessage]{
		H: &service.EventHeaders{
			UserID: uuid.New(), // TODO: Get actual user ID from context
		},
		Msg: eventMessage,
		M: &service.EventMetadata{
			Timestamp: time.Now().UTC(),
		},
	}

	// Publish the event to NATS
	err := event.Publish(ws.s.GetNATS())
	if err != nil {
		ws.log.Error("Failed to publish FlowRunStatusEvent to JetStream", "error", err,
			"subject", service.FlowRunStatusEventSubject.String(),
			"flow_run_id", flowRunID,
			"status", status)
		return
	}

	ws.log.Info("Successfully reported flow run status",
		"flow_run_id", flowRunID,
		"status", status,
		"subject", service.FlowRunStatusEventSubject.String(),
	)
}
