package tasks

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nats-io/nats.go"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/db"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
)

// handoffEventCallback handles the task handoff to sub agent event callback
func (ts *TaskService) handoffEventCallback(msg *nats.Msg) {
	// Check if context was cancelled
	select {
	case <-ts.ctx.Done():
		ts.log.Info("Context cancelled, stopping message processing")
		return
	default:
	}

	ts.log.Debug("Recieved task handoff event message", "subject", msg.Subject, "data", string(msg.Data))
	fmt.Printf("RECIVED: %s\n", string(msg.Data)) // Remove when in production

	// Parse NATS message to request struct
	req, err := service.ParseEvent[*service.TaskHandoffEventMessage](msg.Data)
	if err != nil {
		ts.log.Error("Failed to unmarshal message to request", "error", err)
		// Send error result back to the tool handler
		return
	}

	// Log the received message
	ts.log.Debug("Received and validated task handoff message",
		"agent_id", req.Msg.AgentID,
		"agent_handoff_to_id", req.Msg.AgentHandoffToID,
		"thread_id", req.H.ThreadID,
		"connection_id", req.H.ConnectionID,
		"user_id", req.H.UserID,
	)
	ts.log.Info("Execute message", "messages", req.Msg.Messages)

	// Ensure the agent_id is exist
	queries := db.New(ts.s.GetDB())
	_, err = queries.GetAgentByID(ts.ctx, req.Msg.AgentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			ts.log.Error("Agent not found", "agent_id", req.Msg.AgentID)
			// Send error result back to the tool handler
			return
		}
		ts.log.Error("Failed to get agent by ID", "error", err)
		// Send error result back to the tool handler
		return
	}

	// Create new task for this agent.
	handoffTask, err := queries.CreateTaskWithID(ts.ctx, db.CreateTaskWithIDParams{
		ID:             req.Msg.ToolRunId,
		ThreadID:       *req.H.ThreadID,
		MaxRequestLoop: 20,           // Default max loops
		AdditionalInfo: []byte("{}"), // Empty JSON
		CreatedBy:      req.H.UserID,
		ParentTaskID:   pgtype.Text{String: *req.H.TaskID, Valid: true},
	})
	if err != nil {
		ts.log.Error("Failed to create sub task", "error", err)
		// Send error result back to the tool handler
		return
	}

	// Create task_run for the sub-agent task
	_, err = queries.CreateTasksRun(ts.ctx, handoffTask.ID)
	if err != nil {
		ts.log.Error("Failed to create task run for sub task", "error", err)
		return
	}

	// Set the parent task run status to PENDING while waiting for sub-agent
	err = queries.UpdateTaskRunStatusByTaskID(ts.ctx, db.UpdateTaskRunStatusByTaskIDParams{
		TaskID: *req.H.TaskID,
		Status: db.TaskRunStatusPending,
	})
	if err != nil {
		ts.log.Error("Failed to update parent task run status to PENDING", "error", err)
		return
	}
	ts.log.Info("Set parent task run status to PENDING while waiting for sub-agent", "parent_task_id", *req.H.TaskID, "sub_task_id", handoffTask.ID)

	// Insert new parent agent messages into the database
	for i, message := range req.Msg.Messages {
		_, err := queries.CreateUserMessage(ts.ctx, db.CreateUserMessageParams{
			ThreadID:    *req.H.ThreadID,
			Message:     message,
			SenderID:    req.Msg.AgentID,          // Main Agent is sender
			RecipientID: req.Msg.AgentHandoffToID, // Handoff To Agent is recipient
		})
		if err != nil {
			ts.log.Error("Failed to insert handoffs message %d: %w", i, err)
			// Send error result back to the tool handler
			return
		}
	}

	// Get sender-recipient message history (now includes newly inserted messages)
	messages, err := queries.GetSenderRecipientMessages(ts.ctx, db.GetSenderRecipientMessagesParams{
		ThreadID:    *req.H.ThreadID,
		SenderID:    req.Msg.AgentID,
		RecipientID: req.Msg.AgentHandoffToID,
	})
	if err != nil {
		ts.log.Error("Failed to get sender-recipient messages: %w", err)
		// Send error result back to the tool handler
		return
	}

	// Create a new header for handoffs task
	newHandoffTaskHeader := &service.EventHeaders{
		UserID:       req.H.UserID,
		ThreadID:     req.H.ThreadID,
		TaskID:       &handoffTask.ID,
		ConnectionID: req.H.ConnectionID,
	}

	// Send sub task start event for new sub task
	taskLifecycleMsg := &service.WebsocketTaskLifecycleEventMessage{
		Type:     "sub_task_start",
		ThreadId: *req.H.ThreadID,
		TaskId:   handoffTask.ID,
	}
	taskStartEvent := service.NewEvent(taskLifecycleMsg, newHandoffTaskHeader, req.M)
	err = taskStartEvent.PublishWithUser(ts.s.GetNATS(), req.H.UserID)
	if err != nil {
		ts.log.Error("Failed to publish task start event", "error", err)
		service.NewErrorEvent[*service.WebsocketTaskLifecycleEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		// Send error result back to the tool handler
		return
	}
	ts.log.Info("Published task_start event for new task", "task_id", *req.H.TaskID)

	// Send agent invoke event with complete message history
	ts.log.Info("Publishing messages to agent", "agent_id", req.Msg.AgentHandoffToID)
	invokeEvent := service.NewEvent(&service.AgentInvokeEventMessage{
		AgentId:     req.Msg.AgentHandoffToID,
		Messages:    messages,
		RecipientId: req.Msg.AgentID, // The user recieved this message is the parent agent
	}, newHandoffTaskHeader, req.M)
	err = invokeEvent.Publish(ts.s.GetNATS())
	if err != nil {
		ts.log.Error("Failed to publish agent invoke event", "error", err)
		service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		return
	}

	ts.log.Info("Successfully processed task execution with concurrent operations",
		"thread_id", *req.H.ThreadID,
		"total_messages", len(messages),
	)
}
