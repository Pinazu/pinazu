package tasks

import (
	"encoding/json"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nats-io/nats.go"
	"github.com/pinazu/internal/db"
	"github.com/pinazu/internal/service"
)

// finishEventCallback handles the task finish event callback
func (ts *TaskService) finishEventCallback(msg *nats.Msg) {
	// Check if context was cancelled
	select {
	case <-ts.ctx.Done():
		ts.log.Info("Context cancelled, stopping message processing")
		return
	default:
	}

	// Parse NATS message to request struct
	req, err := service.ParseEvent[*service.TaskFinishEventMessage](msg.Data)
	if err != nil {
		if req == nil {
			ts.log.Error("Failed to parse task finish message", "error", err)
			return
		}
		ts.errorEventCallback(req)
		return
	}

	// Log the received message
	ts.log.Info("Received and validated task finish message",
		"agent_id", req.Msg.AgentId,
		"thread_id", req.H.ThreadID,
		"connection_id", req.H.ConnectionID,
		"user_id", req.H.UserID,
	)
	ts.log.Debug("Task finish response", "message", string(req.Msg.Response))

	// Get the database queries
	queries := db.New(ts.s.GetDB())

	// Create each new message into the database
	_, err = queries.CreateAgentMessage(ts.ctx, db.CreateAgentMessageParams{
		ThreadID:    *req.H.ThreadID,
		Message:     req.Msg.Response,
		StopReason:  pgtype.Text{String: "end_turn", Valid: true},
		SenderID:    req.Msg.AgentId,
		Citations:   req.Msg.Citations,
		RecipientID: req.Msg.RecipientId,
	})
	if err != nil {
		// Check if this is a foreign key constraint violation (thread was deleted)
		if strings.Contains(err.Error(), "fk_thread_message") || strings.Contains(err.Error(), "foreign key constraint") {
			ts.log.Warn("Cannot create completion message: thread was deleted", "thread_id", *req.H.ThreadID, "error", err)
			// Continue processing to ensure proper SSE stream closure
		} else {
			ts.log.Error("Failed to create message to the database", "error", err)
			return
		}
	}

	// // TODO: Waiting for approve about the message payload
	messageContent := anthropic.MessageParam{}
	if err := json.Unmarshal(req.Msg.Response, &messageContent); err != nil {
		ts.log.Error("Failed to unmarshal message content", "error", err)
		return
	}

	// Convert message back to json.Raw
	messageRaw, err := json.Marshal(messageContent.Content)
	if err != nil {
		ts.log.Error("Failed to marshal message content", "error", err)
		return
	}

	// Check if this task is a sub task
	taskInfo, err := queries.GetTaskById(ts.ctx, *req.H.TaskID)
	if err != nil {
		ts.log.Error("Failed to retrieve information about task", "error", err)
		service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		return
	}

	if !taskInfo.ParentTaskID.Valid {
		// If not sub task, update task status to FINISHED
		err = queries.UpdateTaskRunStatusByTaskID(ts.ctx, db.UpdateTaskRunStatusByTaskIDParams{
			TaskID: *req.H.TaskID,
			Status: db.TaskRunStatusFinished,
		})
		if err != nil {
			ts.log.Error("Failed to update main task run status to FINISHED", "error", err)
			service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
			return
		}
		ts.log.Info("Main task marked as FINISHED", "task_id", *req.H.TaskID)

		// Send stop event
		taskLifecycleMsg := &service.WebsocketTaskLifecycleEventMessage{
			Type:     "task_stop",
			ThreadId: *req.H.ThreadID,
			TaskId:   *req.H.TaskID,
		}
		taskStopEvent := service.NewEvent(taskLifecycleMsg, req.H, req.M)
		err = taskStopEvent.PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		if err != nil {
			ts.log.Error("Failed to publish task stop event", "error", err)
			service.NewErrorEvent[*service.WebsocketTaskLifecycleEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
			return
		}
		ts.log.Info("Task finished")
		return // End here if not sub task
	}

	// If sub task, send stop sub start event
	taskLifecycleMsg := &service.WebsocketTaskLifecycleEventMessage{
		Type:     "sub_task_stop",
		ThreadId: *req.H.ThreadID,
		TaskId:   *req.H.TaskID,
	}
	taskStopEvent := service.NewEvent(taskLifecycleMsg, req.H, req.M)
	err = taskStopEvent.PublishWithUser(ts.s.GetNATS(), req.H.UserID)
	if err != nil {
		ts.log.Error("Failed to publish task stop event", "error", err)
		service.NewErrorEvent[*service.WebsocketTaskLifecycleEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		return
	}

	// Set the parent task run status back to RUNNING now that sub-agent is complete
	err = queries.UpdateTaskRunStatusByTaskID(ts.ctx, db.UpdateTaskRunStatusByTaskIDParams{
		TaskID: taskInfo.ParentTaskID.String,
		Status: db.TaskRunStatusRunning,
	})
	if err != nil {
		ts.log.Error("Failed to update parent task run status back to RUNNING", "error", err)
		service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		return
	}
	ts.log.Info("Set parent task run status back to RUNNING after sub-agent completion", "parent_task_id", taskInfo.ParentTaskID.String, "sub_task_id", *req.H.TaskID)

	// Update sub task status to FINISHED since it completed successfully
	err = queries.UpdateTaskRunStatusByTaskID(ts.ctx, db.UpdateTaskRunStatusByTaskIDParams{
		TaskID: *req.H.TaskID,
		Status: db.TaskRunStatusFinished,
	})
	if err != nil {
		ts.log.Error("Failed to update sub task run status to FINISHED", "error", err)
		service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		return
	}
	ts.log.Info("Sub task marked as FINISHED", "sub_task_id", *req.H.TaskID)

	// Create a new header with an old task id
	oldTaskIDHeader := &service.EventHeaders{
		UserID:       req.H.UserID,
		ThreadID:     req.H.ThreadID,
		ConnectionID: req.H.ConnectionID,
		TaskID:       &taskInfo.ParentTaskID.String,
	}

	// Publish messages to tool handler
	event := service.NewEvent(&service.ToolGatherEventMessage{
		ToolRunId:  *req.H.TaskID,
		Content:    messageRaw,
		ResultType: db.ResultMessageTypeText,
		IsError:    false,
	}, oldTaskIDHeader, req.M)
	err = event.Publish(ts.s.GetNATS())
	ts.log.Debug("Published event to Tools Handler", "tool_run_id", *req.H.TaskID)
	if err != nil {
		ts.log.Error("Failed to publish event to Tools Handler", "error", err)
		service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
	}
}

func (ts *TaskService) errorEventCallback(req *service.Event[*service.TaskFinishEventMessage]) {
	// Get the database queries
	queries := db.New(ts.s.GetDB())

	// Get the task
	taskRun, err := queries.GetTaskRunByTaskID(ts.ctx, *req.H.TaskID)
	if err != nil {
		ts.log.Error("Failed to get task by ID", "error", err)
		return
	}

	// Save the updated task
	if err := queries.UpdateTaskRunStatus(ts.ctx, db.UpdateTaskRunStatusParams{
		Status:    db.TaskRunStatusFailed,
		TaskRunID: taskRun[0].TaskRunID,
	}); err != nil {
		ts.log.Error("Failed to update task", "error", err)
		return
	}

	ts.log.Debug("Task marked as failed", "task_id", *req.H.TaskID)
}
