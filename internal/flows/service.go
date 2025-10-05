package flows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pinazu/internal/db"
	"github.com/pinazu/internal/service"
)

type FlowService struct {
	s         service.Service
	log       hclog.Logger
	wg        *sync.WaitGroup
	ctx       context.Context
	jetstream *service.JetStreamService
}

// NewService creates a new FlowService instance
func NewService(ctx context.Context, externalDependenciesConfig *service.ExternalDependenciesConfig, log hclog.Logger, wg *sync.WaitGroup) (*FlowService, error) {
	if externalDependenciesConfig == nil {
		return nil, fmt.Errorf("externalDependenciesConfig is nil")
	}

	// Create a new service instance
	config := &service.Config{
		Name:                 "flows-handler-service",
		Version:              "0.0.1",
		Description:          "Flow service for handling flow execution, flow context management, and flow completion.",
		ExternalDependencies: externalDependenciesConfig,
		ErrorHandler:         nil,
	}
	s, err := service.NewService(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create flow service: %w", err)
	}

	fs := &FlowService{s: s, log: log, wg: wg, ctx: ctx}

	// Register all event handlers
	s.RegisterHandler(service.FlowRunExecuteRequestEventSubject.String(), fs.handleFlowRunExecute)
	s.RegisterHandler("v1.svc.flow._info", nil)
	s.RegisterHandler("v1.svc.flow._stats", nil)

	// Register JetStream handlers for FlowRunStatusUpdateEvent only
	if externalDependenciesConfig.Nats != nil && externalDependenciesConfig.Nats.JetStreamDefaultConfig != nil {
		err = fs.registerStreamHandler(s, externalDependenciesConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to register JetStream stream handlers: %w", err)
		}
	}

	// Start a goroutine to wait for context cancellation and then shutdown
	go func() {
		<-ctx.Done()
		fs.log.Warn("Flow service shutting down...")
		if err := fs.s.Shutdown(); err != nil {
			fs.log.Error("Error during flow service shutdown", "error", err)
		}
		fs.wg.Done()
	}()

	return fs, nil
}

// handleFlowRunExecute handles the flow run execution event
func (fs *FlowService) handleFlowRunExecute(msg *nats.Msg) {
	select {
	case <-fs.ctx.Done():
		fs.log.Info("Context cancelled, stopping message processing")
		return
	default:
	}

	_, span := fs.s.GetTracer().Start(fs.ctx, "handleFlowRunExecute")
	defer span.End()

	data, err := service.ParseEvent[*service.FlowRunExecuteRequestEventMessage](msg.Data)
	if err != nil {
		fs.log.Error("Failed to parse flow run execute request", "error", err)
		service.NewErrorEvent[*service.FlowRunExecuteResponseEventMessage](data.H, data.M, err).Respond(msg)
		return
	}

	req := data.Msg
	queries := db.New(fs.s.GetDB())

	// Check if flow exists
	flow, err := queries.GetFlowById(fs.ctx, req.FlowId)
	if err != nil {
		fs.log.Error("Failed to get flow", "error", err)
		service.NewErrorEvent[*service.FlowRunExecuteResponseEventMessage](data.H, data.M, err).Respond(msg)
		return
	}
	// Set default engine if not provided
	engine := req.Engine
	if engine == "" {
		engine = flow.Engine
	}

	// Generate UUID v7 for the flow run ID
	// Get from req
	flowRunID := uuid.Must(uuid.NewV7())
	if req.FlowRunId != nil {
		flowRunID = *req.FlowRunId
	}

	// Create flow run in database - store only the parameters
	parametersJsonRaw, err := db.NewJsonRaw(req.Parameters)
	if err != nil {
		fs.log.Error("Failed to create JsonRaw from parameters", "error", err)
		service.NewErrorEvent[*service.FlowRunExecuteResponseEventMessage](data.H, data.M, err).Respond(msg)
		return
	}

	// Initialize empty JSON objects for task statuses and results
	taskStatuses, _ := db.NewJsonRaw(map[string]interface{}{})
	successResults, _ := db.NewJsonRaw(map[string]interface{}{})

	flowRunParams := db.CreateFlowRunParams{
		FlowRunID:          flowRunID,
		FlowID:             req.FlowId,
		Parameters:         parametersJsonRaw,
		Status:             "SCHEDULED",
		Engine:             engine,
		TaskStatuses:       taskStatuses,
		SuccessTaskResults: successResults,
		MaxRetries:         pgtype.Int4{Int32: 0, Valid: true}, // Default to 0 retries
	}

	flowRun, err := queries.CreateFlowRun(fs.ctx, flowRunParams)
	if err != nil {
		fs.log.Error("Failed to create flow run", "error", err)
		service.NewErrorEvent[*service.FlowRunExecuteResponseEventMessage](data.H, data.M, err).Respond(msg)
		return
	}

	// Get args from additional_info or use defaults
	args := []string{}
	// Fallback to request args if no args in additional_info
	// Get code location and entrypoint from flow table, fallback to request
	// Publish flow execute event for Worker
	executeEvent := service.Event[*service.FlowRunExecuteEventMessage]{
		H: data.H,
		Msg: &service.FlowRunExecuteEventMessage{
			FlowRunId:          flowRunID,
			Parameters:         req.Parameters,
			Engine:             engine,
			CodeLocation:       flow.CodeLocation.String,
			Entrypoint:         flow.Entrypoint.String,
			Args:               args,
			SuccessTaskResults: make(map[string]string),
			EventTimestamp:     time.Now().UTC(),
		},
		M: &service.EventMetadata{
			TraceID:   data.M.TraceID,
			Timestamp: time.Now().UTC(),
		},
	}

	fs.log.Info("Publishing flow execute event to Worker",
		"subject", service.FlowRunExecuteEventSubject.String(),
		"flow_run_id", flowRunID,
		"code_location", flow.CodeLocation.String,
		"entrypoint", flow.Entrypoint.String,
		"args", args,
		"parameters", req.Parameters)

	// Publish to JetStream instead of regular NATS since worker consumes from JetStream
	err = executeEvent.Publish(fs.s.GetNATS())
	if err != nil {
		fs.log.Error("Failed to publish flow execute event to JetStream", "error", err, "flow_run_id", flowRunID)
		service.NewErrorEvent[*service.FlowRunExecuteResponseEventMessage](data.H, data.M, err).Respond(msg)
		return
	}

	fs.log.Info("Successfully published flow execute event to Worker", "flow_run_id", flowRunID, "subject", service.FlowRunExecuteEventSubject.String())
	// Create response event
	response := service.Event[*service.FlowRunExecuteResponseEventMessage]{
		H: data.H,
		Msg: &service.FlowRunExecuteResponseEventMessage{
			FlowRun: flowRun,
		},
		M: data.M,
	}
	response.Respond(msg)
}

// registerStreamHandler registers JetStream stream handlers for FlowRunStatus and TaskRunStatus events
func (fs *FlowService) registerStreamHandler(s service.Service, config *service.ExternalDependenciesConfig) error {
	// Create JetStream service
	jetStreamService, err := service.NewJetStreamService(fs.ctx, s.GetNATS(), fs.log)
	if err != nil {
		return fmt.Errorf("failed to create JetStream service: %w", err)
	}

	// Store jetstream service for later use
	fs.jetstream = jetStreamService

	// Register handlers for both FlowRunStatus and TaskRunStatus events
	streamConfig := service.CreateStreamConfigWithDefaults(
		"FLOWS_STATUS",
		[]string{
			service.FlowRunStatusEventSubject.String(),
			service.FlowTaskRunStatusEventSubject.String(),
		},
		"Stream for flow and task status updates",
		config.Nats.GetJetStreamConfig(),
	)

	// Create or update the stream
	_, err = jetStreamService.CreateOrUpdateStream(streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create/update FLOWS_STATUS stream: %w", err)
	}

	// Also create the WORKER_FLOWS stream for publishing execution events
	workerStreamConfig := service.CreateStreamConfigWithDefaults(
		"WORKER_FLOWS",
		[]string{service.FlowRunExecuteEventSubject.String()},
		"Stream for worker flow execution events",
		config.Nats.GetJetStreamConfig(),
	)

	_, err = jetStreamService.CreateOrUpdateStream(workerStreamConfig)
	if err != nil {
		return fmt.Errorf("failed to create/update WORKER_FLOWS stream: %w", err)
	}

	// Create consumer for FlowRunStatus events
	flowConsumerConfig := service.ConsumerConfig{
		Name:        "flow_run_status_consumer",
		StreamName:  "FLOWS_STATUS",
		Subject:     service.FlowRunStatusEventSubject.String(),
		Description: "Consumer for FlowRunStatus update events",
		AckWait:     60 * time.Second,
		MaxDeliver:  0, // Will use config default
		FilterBy:    service.FlowRunStatusEventSubject.String(),
	}

	_, err = jetStreamService.CreateOrUpdateConsumer(flowConsumerConfig, config.Nats.GetJetStreamConfig())
	if err != nil {
		return fmt.Errorf("failed to create FlowRunStatus consumer: %w", err)
	}

	// Create consumer for TaskRunStatus events
	taskConsumerConfig := service.ConsumerConfig{
		Name:        "task_run_status_consumer",
		StreamName:  "FLOWS_STATUS",
		Subject:     service.FlowTaskRunStatusEventSubject.String(),
		Description: "Consumer for TaskRunStatus update events",
		AckWait:     60 * time.Second,
		MaxDeliver:  0, // Will use config default
		FilterBy:    service.FlowTaskRunStatusEventSubject.String(),
	}

	_, err = jetStreamService.CreateOrUpdateConsumer(taskConsumerConfig, config.Nats.GetJetStreamConfig())
	if err != nil {
		return fmt.Errorf("failed to create TaskRunStatus consumer: %w", err)
	}

	// Start consuming messages for FlowRunStatus events
	err = jetStreamService.ConsumeMessages("flow_run_status_consumer", "FLOWS_STATUS", fs.handleFlowRunStatusUpdateJS)
	if err != nil {
		return fmt.Errorf("failed to start FlowRunStatus consumer: %w", err)
	}

	// Start consuming messages for TaskRunStatus events
	err = jetStreamService.ConsumeMessages("task_run_status_consumer", "FLOWS_STATUS", fs.handleTaskRunStatusUpdateJS)
	if err != nil {
		return fmt.Errorf("failed to start TaskRunStatus consumer: %w", err)
	}

	fs.log.Info("JetStream stream handlers registered successfully for FlowRunStatus and TaskRunStatus events")
	return nil
}

// handleFlowRunStatusUpdateJS handles FlowRunStatus events via JetStream
func (fs *FlowService) handleFlowRunStatusUpdateJS(msg jetstream.Msg) error {
	// Parse the event
	eventData, err := service.ParseEvent[*service.FlowRunStatusEventMessage](msg.Data())
	if err != nil {
		fs.log.Error("Failed to parse FlowRunStatus event", "error", err)
		return err
	}

	statusMsg := eventData.Msg
	fs.log.Info("Processing FlowRunStatus update via JetStream",
		"flow_run_id", statusMsg.FlowRunId,
		"status", statusMsg.Status,
		"timestamp", statusMsg.EventTimestamp,
		"error_message", statusMsg.ErrorMessage)

	// Update flow run status in database
	queries := db.New(fs.s.GetDB())

	// Update started_at when event is consumed (first time only)
	if err := queries.UpdateFlowRunStartedAt(fs.ctx, statusMsg.FlowRunId); err != nil {
		fs.log.Error("Failed to update flow run started_at",
			"flow_run_id", statusMsg.FlowRunId,
			"error", err)
		// Continue with status update even if started_at update fails
	}

	// Use UpdateFlowRunError for FAILED status with error message
	if statusMsg.Status == "FAILED" && statusMsg.ErrorMessage != "" {
		if err := queries.UpdateFlowRunError(fs.ctx, db.UpdateFlowRunErrorParams{
			FlowRunID:    statusMsg.FlowRunId,
			ErrorMessage: pgtype.Text{String: statusMsg.ErrorMessage, Valid: true},
		}); err != nil {
			fs.log.Error("Failed to update flow run with error",
				"flow_run_id", statusMsg.FlowRunId,
				"error", err)
			return fmt.Errorf("failed to update flow run with error: %w", err)
		}
	} else {
		// Use timestamp-aware status update to track started_at when status becomes RUNNING
		if err := queries.UpdateFlowRunStatusWithTimestamps(fs.ctx, db.UpdateFlowRunStatusWithTimestampsParams{
			Status:    string(statusMsg.Status),
			FlowRunID: statusMsg.FlowRunId,
		}); err != nil {
			fs.log.Error("Failed to update flow run status",
				"flow_run_id", statusMsg.FlowRunId,
				"error", err)
			return fmt.Errorf("failed to update flow run status: %w", err)
		}
	}

	fs.log.Info("FlowRunStatus updated successfully via JetStream",
		"flow_run_id", statusMsg.FlowRunId,
		"status", statusMsg.Status,
		"error_message", statusMsg.ErrorMessage)

	// Acknowledge the message
	return msg.Ack()
}

// handleTaskRunStatusUpdateJS handles TaskRunStatus events via JetStream
func (fs *FlowService) handleTaskRunStatusUpdateJS(msg jetstream.Msg) error {
	// Parse the event
	eventData, err := service.ParseEvent[*service.FlowTaskRunStatusEventMessage](msg.Data())
	if err != nil {
		fs.log.Error("Failed to parse TaskRunStatus event", "error", err)
		return err
	}

	statusMsg := eventData.Msg
	fs.log.Info("Processing TaskRunStatus update via JetStream",
		"flow_run_id", statusMsg.FlowRunId,
		"task_name", statusMsg.TaskName,
		"status", statusMsg.Status,
		"timestamp", statusMsg.EventTimestamp)

	// Update task run status in database directly
	queries := db.New(fs.s.GetDB())
	if err := queries.UpdateFlowTaskRunStatus(fs.ctx, db.UpdateFlowTaskRunStatusParams{
		Status:    statusMsg.Status,
		FlowRunID: statusMsg.FlowRunId,
		TaskName:  statusMsg.TaskName,
	}); err != nil {
		fs.log.Error("Failed to update task run status",
			"flow_run_id", statusMsg.FlowRunId,
			"task_name", statusMsg.TaskName,
			"error", err)
		return fmt.Errorf("failed to update task run status: %w", err)
	}

	// Update result cache key if provided (for SUCCESS status)
	if statusMsg.ResultCacheKey != nil && statusMsg.Status == TaskRunStatusSuccess {
		// We need to provide a result even if it's empty for the update
		emptyResult, _ := db.NewJsonRaw(map[string]interface{}{})
		if err := queries.UpdateFlowTaskRunResult(fs.ctx, db.UpdateFlowTaskRunResultParams{
			FlowRunID:      statusMsg.FlowRunId,
			TaskName:       statusMsg.TaskName,
			Result:         emptyResult,
			ResultCacheKey: pgtype.Text{String: *statusMsg.ResultCacheKey, Valid: true},
		}); err != nil {
			fs.log.Error("Failed to update task run result cache key",
				"flow_run_id", statusMsg.FlowRunId,
				"task_name", statusMsg.TaskName,
				"error", err)
			return fmt.Errorf("failed to update task run result cache key: %w", err)
		}
		fs.log.Debug("Updated task run result cache key",
			"flow_run_id", statusMsg.FlowRunId,
			"task_name", statusMsg.TaskName,
			"result_cache_key", *statusMsg.ResultCacheKey)
	}

	fs.log.Info("TaskRunStatus updated successfully via JetStream",
		"flow_run_id", statusMsg.FlowRunId,
		"task_name", statusMsg.TaskName,
		"status", statusMsg.Status)

	// Acknowledge the message
	return msg.Ack()
}
