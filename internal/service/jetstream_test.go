package service

import (
	"testing"
	"time"
	
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

func TestCreateStreamConfigWithDefaults(t *testing.T) {
	t.Run("with custom jetstream config", func(t *testing.T) {
		jsConfig := &JetStreamConfig{
			MaxMsgs:       2000000,
			MaxBytes:      2 * 1024 * 1024 * 1024, // 2GB
			MaxAgeSeconds: 172800,                  // 48 hours (172800 seconds)
			Replicas:      3,                       // 3 replicas
		}

		config := CreateStreamConfigWithDefaults(
			"TEST_STREAM",
			[]string{"test.subject.>"},
			"Test stream",
			jsConfig,
		)

		assert.Equal(t, "TEST_STREAM", config.Name)
		assert.Equal(t, []string{"test.subject.>"}, config.Subjects)
		assert.Equal(t, int64(2000000), config.MaxMsgs)
		assert.Equal(t, int64(2*1024*1024*1024), config.MaxBytes)
		assert.Equal(t, 172800*time.Second, config.MaxAge)
		assert.Equal(t, 3, config.Replicas)
		assert.Equal(t, jetstream.FileStorage, config.Storage)
		assert.Equal(t, jetstream.WorkQueuePolicy, config.Retention)
		assert.Equal(t, jetstream.DiscardOld, config.Discard)
	})

	t.Run("with nil jetstream config (should have default NATS values)", func(t *testing.T) {
		config := CreateStreamConfigWithDefaults(
			"DEFAULT_STREAM",
			[]string{"default.>"},
			"Default stream",
			nil,
		)

		// With nil config, stream-specific values should be zero, but NATS defaults should be set
		assert.Equal(t, int64(0), config.MaxMsgs)
		assert.Equal(t, int64(0), config.MaxBytes)
		assert.Equal(t, time.Duration(0), config.MaxAge)
		assert.Equal(t, 0, config.Replicas)
		
		// But NATS defaults should be set
		assert.Equal(t, jetstream.FileStorage, config.Storage)
		assert.Equal(t, jetstream.WorkQueuePolicy, config.Retention)
		assert.Equal(t, jetstream.DiscardOld, config.Discard)
	})
}

func TestNatsConfig_GetJetStreamConfig(t *testing.T) {
	t.Run("with existing jetstream config", func(t *testing.T) {
		natsConfig := &NatsConfig{
			URL: "nats://localhost:4222",
			JetStreamDefaultConfig: &JetStreamConfig{
				MaxMsgs:       500000,
				MaxBytes:      512 * 1024 * 1024, // 512MB
				MaxAgeSeconds: 43200,              // 12 hours (43200 seconds)
				Replicas:      2,                  // 2 replicas
			},
		}

		jsConfig := natsConfig.GetJetStreamConfig()

		assert.Equal(t, int64(500000), jsConfig.MaxMsgs)
		assert.Equal(t, int64(512*1024*1024), jsConfig.MaxBytes)
		assert.Equal(t, 43200, jsConfig.MaxAgeSeconds)
		assert.Equal(t, 2, jsConfig.Replicas)
	})

	t.Run("with nil jetstream config (returns nil)", func(t *testing.T) {
		natsConfig := &NatsConfig{
			URL: "nats://localhost:4222",
			JetStreamDefaultConfig: nil, // No JetStream config
		}

		jsConfig := natsConfig.GetJetStreamConfig()

		assert.Nil(t, jsConfig)
	})

	t.Run("with nil nats config (returns nil)", func(t *testing.T) {
		var natsConfig *NatsConfig = nil

		jsConfig := natsConfig.GetJetStreamConfig()

		assert.Nil(t, jsConfig)
	})
}