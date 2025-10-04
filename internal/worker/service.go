package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nats-io/nats.go/jetstream"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
)

type WorkerService struct {
	s      service.Service
	js     *service.JetStreamService
	config *service.ExternalDependenciesConfig
	log    hclog.Logger
	wg     *sync.WaitGroup
	ctx    context.Context
}

// Create a new worker service instance
func NewService(ctx context.Context, externalDependenciesConfig *service.ExternalDependenciesConfig, log hclog.Logger, wg *sync.WaitGroup) (*WorkerService, error) {
	if externalDependenciesConfig == nil {
		return nil, fmt.Errorf("externalDependenciesConfig is nil")
	}

	// Create a new service instance
	config := &service.Config{
		Name:                 "worker-service",
		Version:              "0.0.1",
		Description:          "Worker service for executing flow runs via local processes",
		ExternalDependencies: externalDependenciesConfig,
		ErrorHandler:         nil,
	}
	s, err := service.NewService(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker service: %w", err)
	}

	// Create JetStream service
	js, err := service.NewJetStreamService(ctx, s.GetNATS(), log)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream service: %w", err)
	}

	ws := &WorkerService{s: s, js: js, config: externalDependenciesConfig, log: log, wg: wg, ctx: ctx}

	// Get JetStream configuration
	jsConfig := externalDependenciesConfig.Nats.GetJetStreamConfig()

	// Create or update stream
	streamConfig := service.CreateStreamConfigWithDefaults(
		"WORKER_FLOWS",
		[]string{string(service.FlowRunExecuteEventSubject)},
		"Stream for worker flow execution events",
		jsConfig,
	)

	stream, err := js.CreateOrUpdateStream(streamConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	ws.log.Info("JetStream stream created/updated", "name", stream.CachedInfo().Config.Name)

	// Create consumer configuration
	consumerConfig := service.ConsumerConfig{
		Name:        "worker-flow-consumer",
		StreamName:  "WORKER_FLOWS",
		Subject:     string(service.FlowRunExecuteEventSubject),
		Description: "Consumer for worker flow execution events",
		AckWait:     30 * time.Second,
		MaxDeliver:  3,
	}

	consumer, err := js.CreateOrUpdateConsumer(consumerConfig, jsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	ws.log.Info("JetStream consumer created/updated", "name", consumer.CachedInfo().Name)

	// Start consuming messages
	if err := js.ConsumeMessages("worker-flow-consumer", "WORKER_FLOWS", ws.handleFlowRunExecute); err != nil {
		return nil, fmt.Errorf("failed to start consuming messages: %w", err)
	}

	ws.log.Info("Started consuming JetStream messages",
		"subject", string(service.FlowRunExecuteEventSubject),
		"stream", "WORKER_FLOWS",
		"consumer", "worker-flow-consumer",
	)

	// Log cache configuration status
	ws.logCacheConfiguration()

	// Keep regular handlers for info and stats
	s.RegisterHandler("v1.svc.worker._info", nil)
	s.RegisterHandler("v1.svc.worker._stats", nil)

	// Start a goroutine to wait for context cancellation and then shutdown
	go func() {
		<-ctx.Done()
		ws.log.Warn("Worker service shutting down...")

		if err := ws.s.Shutdown(); err != nil {
			ws.log.Error("Error during worker service shutdown", "error", err)
		}
		ws.wg.Done()
	}()

	return ws, nil
}

// logCacheConfiguration logs the current cache configuration for flows
func (ws *WorkerService) logCacheConfiguration() {
	if ws.config == nil {
		ws.log.Info("Flow cache configuration: No configuration specified, using default behavior (S3 if available, otherwise memory)")
		return
	}

	if ws.config.Cache == nil {
		// Check if S3 is available for backward compatibility
		if ws.config.Storage != nil && ws.config.Storage.S3 != nil {
			ws.log.Info("Flow cache configuration: No cache config specified, using S3 cache (backward compatibility)",
				"s3_endpoint", ws.config.Storage.S3.EndpointURL,
				"default_bucket", "flow-cache-bucket",
			)
		} else {
			ws.log.Info("Flow cache configuration: No cache config specified, using memory cache (S3 not available)")
		}
		return
	}

	cacheConfig := ws.config.Cache
	switch cacheConfig.Type {
	case service.CacheTypeS3:
		if ws.config.Storage != nil && ws.config.Storage.S3 != nil {
			ws.log.Info("Flow cache configuration: S3 caching enabled",
				"cache_type", cacheConfig.Type,
				"cache_bucket", cacheConfig.Bucket,
				"s3_endpoint", ws.config.Storage.S3.EndpointURL,
				"s3_region", ws.config.Storage.S3.Region,
				"s3_use_path_style", ws.config.Storage.S3.UsePathStyle,
			)
		} else {
			ws.log.Warn("Flow cache configuration: S3 caching requested but S3 storage not configured - flows will use memory cache",
				"cache_type", cacheConfig.Type,
				"cache_bucket", cacheConfig.Bucket,
			)
		}
	case service.CacheTypeMemory:
		ws.log.Info("Flow cache configuration: Memory caching enabled",
			"cache_type", cacheConfig.Type,
			"note", "Cache will be cleared when each flow completes",
		)
	default:
		ws.log.Error("Flow cache configuration: Invalid cache type",
			"cache_type", cacheConfig.Type,
			"valid_types", fmt.Sprintf("%s, %s", service.CacheTypeMemory, service.CacheTypeS3),
		)
	}
}

// handleFlowRunExecute handles incoming FlowRunExecuteEvent messages from the Orchestrator
func (ws *WorkerService) handleFlowRunExecute(msg jetstream.Msg) error {
	startTime := time.Now().UTC()

	// Extract JetStream metadata
	metadata, err := msg.Metadata()
	deliveryCount := uint64(0)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	if err == nil && metadata != nil {
		deliveryCount = metadata.NumDelivered
		timestamp = metadata.Timestamp.Format(time.RFC3339)
	}

	ws.log.Debug("Processing JetStream message",
		"subject", msg.Subject(),
		"size_bytes", len(msg.Data()),
		"delivery_count", deliveryCount,
		"timestamp", timestamp,
	)

	// Check if context was cancelled
	select {
	case <-ws.ctx.Done():
		ws.log.Info("Context cancelled, stopping message processing",
			"processing_time_ms", time.Since(startTime).Milliseconds(),
		)
		// Don't acknowledge the message if we're shutting down
		return nil
	default:
	}

	_, span := ws.s.GetTracer().Start(ws.ctx, "handleFlowRunExecute")
	defer span.End()

	// Parse JetStream message to request struct
	req, err := service.ParseEvent[*service.FlowRunExecuteEventMessage](msg.Data())
	if err != nil {
		ws.log.Error("Failed to parse FlowRunExecuteEvent",
			"error", err,
			"subject", msg.Subject(),
			"delivery_count", deliveryCount,
			"processing_time_ms", time.Since(startTime).Milliseconds(),
		)

		// Negative acknowledge - this will trigger a retry
		if nakErr := msg.Nak(); nakErr != nil {
			ws.log.Error("Failed to NAK message after parse error", "error", nakErr)
		} else {
			ws.log.Warn("Successfully NAK'd message due to parse error - message will be retried",
				"delivery_count", deliveryCount,
				"max_deliver", 3,
			)
		}
		return nil
	}

	// Log message details with JetStream context
	ws.log.Info("Processing flow run execute event from JetStream",
		"flow_run_id", req.Msg.FlowRunId,
		"thread_id", req.H.ThreadID,
		"connection_id", req.H.ConnectionID,
		"user_id", req.H.UserID,
		"delivery_count", deliveryCount,
		"is_retry", deliveryCount > 1,
		"engine", req.Msg.Engine,
		"code_location", req.Msg.CodeLocation,
	)

	// Report PENDING status
	ws.reportFlowRunStatus(req.Msg.FlowRunId, "PENDING")

	// Start the flow process execution in a separate goroutine
	// and handle the acknowledgment based on the result
	go func() {
		processingStartTime := time.Now().UTC()

		defer func() {
			processingDuration := time.Since(processingStartTime)
			totalDuration := time.Since(startTime)

			// Acknowledge the message after processing is complete
			if ackErr := msg.Ack(); ackErr != nil {
				ws.log.Error("Failed to ACK message after flow processing",
					"flow_run_id", req.Msg.FlowRunId,
					"error", ackErr,
					"processing_time_ms", processingDuration.Milliseconds(),
					"total_time_ms", totalDuration.Milliseconds(),
				)
			} else {
				ws.log.Info("Successfully acknowledged message after flow processing",
					"flow_run_id", req.Msg.FlowRunId,
					"delivery_count", deliveryCount,
					"processing_time_ms", processingDuration.Milliseconds(),
					"total_time_ms", totalDuration.Milliseconds(),
				)
			}
		}()

		// Execute the flow process
		ws.executeFlowProcess(ws.ctx, req.Msg)
	}()

	return nil
}
