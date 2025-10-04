package flows

import (
	"time"
)

// StatusUpdateConfig represents configuration for status updates
type StatusUpdateConfig struct {
	EnableRetries       bool          `json:"enable_retries"`
	MaxRetries          int           `json:"max_retries"`
	RetryBackoff        time.Duration `json:"retry_backoff"`
	EnableDeadLettering bool          `json:"enable_dead_lettering"`
	DeadLetterTopic     string        `json:"dead_letter_topic"`
	BatchSize           int           `json:"batch_size"`
	BatchTimeout        time.Duration `json:"batch_timeout"`
}

// NewStatusUpdateConfigFromServiceConfig creates a default StatusUpdateConfig
func NewDefaultStatusUpdateConfig() StatusUpdateConfig {
	return StatusUpdateConfig{
		EnableRetries:       true,
		MaxRetries:          3,
		RetryBackoff:        5 * time.Second,
		EnableDeadLettering: true,
		DeadLetterTopic:     "flows.dead_letter",
		BatchSize:           10,
		BatchTimeout:        5 * time.Second,
	}
}

// Valid status values for flow runs
const (
	FlowRunStatusPending = "PENDING"
	FlowRunStatusRunning = "RUNNING"
	FlowRunStatusSuccess = "SUCCESS"
	FlowRunStatusFailed  = "FAILED"
)

// Valid status values for task runs
const (
	TaskRunStatusRunning = "RUNNING"
	TaskRunStatusSuccess = "SUCCESS"
	TaskRunStatusFailed  = "FAILED"
)
