package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	// JetStreamService wraps JetStream functionality for stream and consumer management
	JetStreamService struct {
		nc     *nats.Conn
		js     jetstream.JetStream
		ctx    context.Context
		logger hclog.Logger
	}


	// ConsumerConfig holds configuration for creating JetStream consumers
	ConsumerConfig struct {
		Name        string
		StreamName  string
		Subject     string
		Description string
		AckWait     time.Duration
		MaxDeliver  int
		FilterBy    string
	}
)

// NewJetStreamService creates a new JetStream service instance
func NewJetStreamService(ctx context.Context, nc *nats.Conn, logger hclog.Logger) (*JetStreamService, error) {
	if nc == nil {
		return nil, fmt.Errorf("nats connection is required")
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &JetStreamService{
		nc:     nc,
		js:     js,
		ctx:    ctx,
		logger: logger.Named("jetstream"),
	}, nil
}

// CreateStreamConfigWithDefaults creates a jetstream.StreamConfig with values from JetStreamConfig
func CreateStreamConfigWithDefaults(name string, subjects []string, description string, jsConfig *JetStreamConfig) jetstream.StreamConfig {
	config := jetstream.StreamConfig{
		Name:        name,
		Subjects:    subjects,
		Description: description,
		Storage:     jetstream.FileStorage,
		Retention:   jetstream.WorkQueuePolicy,
		Discard:     jetstream.DiscardOld,
	}

	// jsConfig should never be nil as it comes from GetJetStreamConfig() which loads from config file
	// The config file should always have the jetstream section with proper defaults
	if jsConfig != nil {
		config.MaxMsgs = jsConfig.MaxMsgs
		config.MaxBytes = jsConfig.MaxBytes
		config.MaxAge = time.Duration(jsConfig.MaxAgeSeconds) * time.Second
		config.Replicas = jsConfig.Replicas
	}

	return config
}

// CreateOrUpdateStream creates or updates a JetStream stream
func (jss *JetStreamService) CreateOrUpdateStream(config jetstream.StreamConfig) (jetstream.Stream, error) {
	jss.logger.Info("Creating or updating stream", "name", config.Name, "subjects", config.Subjects)

	stream, err := jss.js.CreateOrUpdateStream(jss.ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create or update stream: %w", err)
	}

	jss.logger.Info("Stream created/updated successfully", "name", config.Name)
	return stream, nil
}

// CreateOrUpdateConsumer creates or updates a JetStream consumer
func (jss *JetStreamService) CreateOrUpdateConsumer(config ConsumerConfig, jsConfig *JetStreamConfig) (jetstream.Consumer, error) {
	jss.logger.Info("Creating or updating consumer", "name", config.Name, "stream", config.StreamName)

	stream, err := jss.js.Stream(jss.ctx, config.StreamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream %s: %w", config.StreamName, err)
	}

	consumerConfig := jetstream.ConsumerConfig{
		Name:        config.Name,
		Description: config.Description,
		AckWait:     config.AckWait,
		MaxDeliver:  config.MaxDeliver,
		AckPolicy:   jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	}

	// Set filter subject if provided
	if config.FilterBy != "" {
		consumerConfig.FilterSubject = config.FilterBy
	}

	// Set defaults if not provided
	if consumerConfig.AckWait <= 0 {
		consumerConfig.AckWait = 30 * time.Second
	}
	if consumerConfig.MaxDeliver <= 0 {
		// Use configuration value or default to 3
		if jsConfig != nil && jsConfig.MaxDeliver > 0 {
			consumerConfig.MaxDeliver = jsConfig.MaxDeliver
		} else {
			consumerConfig.MaxDeliver = 3
		}
	}

	consumer, err := stream.CreateOrUpdateConsumer(jss.ctx, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create or update consumer: %w", err)
	}

	jss.logger.Info("Consumer created/updated successfully", "name", config.Name)
	return consumer, nil
}

// ConsumeMessages starts consuming messages from a consumer with a handler function
func (jss *JetStreamService) ConsumeMessages(consumerName, streamName string, handler func(jetstream.Msg) error) error {
	stream, err := jss.js.Stream(jss.ctx, streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	consumer, err := stream.Consumer(jss.ctx, consumerName)
	if err != nil {
		return fmt.Errorf("failed to get consumer %s: %w", consumerName, err)
	}

	// Wrap the handler to match MessageHandler signature
	messageHandler := func(msg jetstream.Msg) {
		if err := handler(msg); err != nil {
			jss.logger.Error("Error processing message", "error", err, "subject", msg.Subject())
			// In case of error, we can choose to NAK the message
			msg.Nak()
		}
	}

	// Consume messages with context
	consumeCtx, err := consumer.Consume(messageHandler)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	jss.logger.Info("Started consuming messages", "consumer", consumerName, "stream", streamName)

	// Wait for context cancellation
	go func() {
		<-jss.ctx.Done()
		jss.logger.Info("Stopping message consumption", "consumer", consumerName)
		consumeCtx.Stop()
	}()

	return nil
}

// PublishMessage publishes a message to a subject
func (jss *JetStreamService) PublishMessage(subject string, data []byte) (*jetstream.PubAck, error) {
	ack, err := jss.js.Publish(jss.ctx, subject, data)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message to %s: %w", subject, err)
	}

	jss.logger.Debug("Message published", "subject", subject, "stream", ack.Stream, "sequence", ack.Sequence)
	return ack, nil
}

// DeleteStream deletes a JetStream stream
func (jss *JetStreamService) DeleteStream(streamName string) error {
	err := jss.js.DeleteStream(jss.ctx, streamName)
	if err != nil {
		return fmt.Errorf("failed to delete stream %s: %w", streamName, err)
	}

	jss.logger.Info("Stream deleted successfully", "name", streamName)
	return nil
}

// DeleteConsumer deletes a JetStream consumer
func (jss *JetStreamService) DeleteConsumer(streamName, consumerName string) error {
	stream, err := jss.js.Stream(jss.ctx, streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	err = stream.DeleteConsumer(jss.ctx, consumerName)
	if err != nil {
		return fmt.Errorf("failed to delete consumer %s: %w", consumerName, err)
	}

	jss.logger.Info("Consumer deleted successfully", "name", consumerName)
	return nil
}

// GetStreamInfo returns information about a stream
func (jss *JetStreamService) GetStreamInfo(streamName string) (*jetstream.StreamInfo, error) {
	stream, err := jss.js.Stream(jss.ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	info, err := stream.Info(jss.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	return info, nil
}

// GetConsumerInfo returns information about a consumer
func (jss *JetStreamService) GetConsumerInfo(streamName, consumerName string) (*jetstream.ConsumerInfo, error) {
	stream, err := jss.js.Stream(jss.ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	consumer, err := stream.Consumer(jss.ctx, consumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer %s: %w", consumerName, err)
	}

	info, err := consumer.Info(jss.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}

	return info, nil
}