package tasks

import (
	"github.com/nats-io/nats.go"
	"github.com/pinazu/core/internal/service"
)

// cancelEventCallback handles the task cancel event callback
func (ts *TaskService) cancelEventCallback(msg *nats.Msg) {
	// Check if context was cancelled
	select {
	case <-ts.ctx.Done():
		ts.log.Info("Context cancelled, stopping message processing")
		return
	default:
	}

	// Parse NATS message to request struct
	req, err := service.ParseEvent[*service.TaskCancelEventMessage](msg.Data)
	if err != nil {
		ts.log.Error("Failed to unmarshal message to request", "error", err)
		return
	}

	// Handle the callback logic here
	ts.log.Info("Received and validated task cancel message",
		"thread_id", req.H.ThreadID,
		"connection_id", req.H.ConnectionID,
		"user_id", req.H.UserID,
	)
}
