package tasks

import (
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nats-io/nats.go"
	"github.com/pinazu/internal/db"
	"github.com/pinazu/internal/service"
)

// executeEventCallback handles the task execute request event callback
func (ts *TaskService) executeEventCallback(msg *nats.Msg) {
	// Check if context was cancelled
	select {
	case <-ts.ctx.Done():
		ts.log.Info("Context cancelled, stopping message processing")
		return
	default:
	}

	// Parse and validate the request
	ts.log.Debug("Recieved task execute event message", "subject", msg.Subject, "data", string(msg.Data))
	fmt.Printf("RECIVED: %s\n", string(msg.Data)) // Remove when in production

	// Parse NATS message to request struct
	req, err := service.ParseEvent[*service.TaskExecuteEventMessage](msg.Data)
	if err != nil {
		ts.log.Error("Failed to unmarshal message to request", "error", err)
		service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		return
	}

	// Log the received message
	ts.log.Debug("Received and validated task execute message",
		"agent_id", req.Msg.AgentId,
		"thread_id", req.H.ThreadID,
		"connection_id", req.H.ConnectionID,
		"user_id", req.H.UserID,
	)
	ts.log.Info("Execute message", "messages", req.Msg.Messages)

	// Ensure thread exists (create if needed)
	if err := ts.ensureThreadExists(req); err != nil {
		return
	}

	// Check if this is a new task (TaskID is nil) before processing
	isNewTask := req.H.TaskID == nil

	// Process message operations sequentially, task operations concurrently
	senderRecipientMessages, err := ts.processMessageOperations(req)
	if err != nil {
		return
	}

	// Use the retrieved messages directly (they already include the newly inserted messages)
	sendMessages := senderRecipientMessages

	// Only send task start event for NEW tasks, not existing ones
	if isNewTask {
		taskLifecycleMsg := &service.WebsocketTaskLifecycleEventMessage{
			Type:     "task_start",
			ThreadId: *req.H.ThreadID,
			TaskId:   *req.H.TaskID,
		}

		taskStartEvent := service.NewEvent(taskLifecycleMsg, req.H, req.M)
		err = taskStartEvent.PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		if err != nil {
			ts.log.Error("Failed to publish task start event", "error", err)
			service.NewErrorEvent[*service.WebsocketTaskLifecycleEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
			return
		}
		ts.log.Info("Published task_start event for new task", "task_id", *req.H.TaskID)
	} else {
		ts.log.Info("Continuing existing task, skipping task_start event", "task_id", *req.H.TaskID)
	}

	// Send agent invoke event with complete message history
	ts.log.Info("Publishing messages to agent", "agent_id", req.Msg.AgentId)
	invokeEvent := service.NewEvent(&service.AgentInvokeEventMessage{
		AgentId:     req.Msg.AgentId,
		Messages:    sendMessages,
		RecipientId: req.Msg.RecipientId,
	}, req.H, req.M)
	err = invokeEvent.Publish(ts.s.GetNATS())
	if err != nil {
		ts.log.Error("Failed to publish agent invoke event", "error", err)
		service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		return
	}

	ts.log.Info("Successfully processed task execution with concurrent operations",
		"thread_id", *req.H.ThreadID,
		"total_messages", len(sendMessages),
	)
}

// ensureThreadExists creates a new thread if one doesn't exist in the request
func (ts *TaskService) ensureThreadExists(req *service.Event[*service.TaskExecuteEventMessage]) error {
	if req.H.ThreadID != nil {
		return nil
	}

	// Get database queries
	queries := db.New(ts.s.GetDB())

	ts.log.Info("ThreadId is nil, creating new thread")
	now := time.Now()
	thread, err := queries.CreateThread(ts.ctx, db.CreateThreadParams{
		Title:     "Thread_" + req.H.UserID.String() + req.H.ConnectionID.String(),
		UserID:    req.H.UserID,
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		ts.log.Error("Failed to create new thread", "error", err)
		service.NewErrorEvent[*service.WebsocketResponseEventMessage](req.H, req.M, err).PublishWithUser(ts.s.GetNATS(), req.H.UserID)
		return err
	}
	// Update the request with the new thread ID
	req.H.ThreadID = &thread.ID
	ts.log.Info("Created new temporary thread", "thread_id", thread.ID)

	return nil
}

// processMessageOperations handles message operations sequentially and task operations concurrently
func (ts *TaskService) processMessageOperations(req *service.Event[*service.TaskExecuteEventMessage]) ([]db.JsonRaw, error) {
	queries := db.New(ts.s.GetDB())

	// Handle task runs concurrently (no race condition here)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go ts.manageTaskRuns(req, queries, &wg, errChan)

	// Handle messages sequentially to avoid race condition
	// 1. Insert new user messages FIRST
	for i, message := range req.Msg.Messages {
		_, err := queries.CreateUserMessage(ts.ctx, db.CreateUserMessageParams{
			ThreadID:    *req.H.ThreadID,
			Message:     message,
			SenderID:    req.Msg.RecipientId, // User is sender
			RecipientID: req.Msg.AgentId,     // Agent is recipient
		})
		if err != nil {
			return nil, fmt.Errorf("failed to insert message %d: %w", i, err)
		}
	}

	// 2. Get sender-recipient message history (now includes newly inserted messages)
	messages, err := queries.GetSenderRecipientMessages(ts.ctx, db.GetSenderRecipientMessagesParams{
		ThreadID:    *req.H.ThreadID,
		SenderID:    req.Msg.AgentId,
		RecipientID: req.Msg.RecipientId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get sender-recipient messages: %w", err)
	}

	// Wait for task operations to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for result := range errChan {
		if result != nil {
			ts.log.Error("Task operation failed", "error", result)
			return nil, result
		}
	}

	ts.log.Info("Message operations completed", "message_count", len(messages))
	return messages, nil
}

// manageTaskRuns handles task and task run creation/management
func (ts *TaskService) manageTaskRuns(req *service.Event[*service.TaskExecuteEventMessage], queries *db.Queries, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()
	// If no tasks exist, create one. If already have 1, update the task runs
	var taskRun db.TasksRun
	var task db.Task
	var err error

	if req.H.TaskID == nil {
		task, err = queries.CreateTask(ts.ctx, db.CreateTaskParams{
			ThreadID:       *req.H.ThreadID,
			MaxRequestLoop: 20,           // Default max loops
			AdditionalInfo: []byte("{}"), // Empty JSON
			CreatedBy:      req.H.UserID,
		})
		if err != nil {
			errChan <- fmt.Errorf("failed to create task: %w", err)
			return
		}
		ts.log.Info("Created new task", "task_id", task.ID, "thread_id", *req.H.ThreadID)
		// Update the task_run_id
		req.H.TaskID = &task.ID
		taskRun, err = queries.CreateTasksRun(ts.ctx, task.ID)
		if err != nil {
			errChan <- fmt.Errorf("failed to create task run: %w", err)
			return
		}
		ts.log.Info("Created new task run", "task_run_id", taskRun.TaskRunID, "task_id", task.ID)
	} else {
		// Get the task run
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			taskRun, err = queries.GetCurrentTaskRunByTaskID(ts.ctx, *req.H.TaskID)
			if err != nil {
				errChan <- fmt.Errorf("failed to get task runs: %w", err)
				return
			}
		}()
		// Get the task
		go func() {
			defer wg.Done()
			task, err = queries.GetTaskById(ts.ctx, *req.H.TaskID)
			if err != nil {
				errChan <- fmt.Errorf("failed to get task by id: %w", err)
				return
			}
			ts.log.Info("Retrieved existing task", "task_id", task.ID, "current_loops", taskRun.CurrentLoops, "max_loops", task.MaxRequestLoop)
		}()
		wg.Wait()
	}

	// Check if max loops reached before incrementing
	if taskRun.CurrentLoops >= task.MaxRequestLoop {
		// If max loops reached, mark task as PENDING waiting for user input
		defer wg.Done()
		err := queries.UpdateTaskRunStatus(ts.ctx, db.UpdateTaskRunStatusParams{
			TaskRunID: taskRun.TaskRunID,
			Status:    db.TaskRunStatusPending,
		})
		if err != nil {
			errChan <- fmt.Errorf("failed to update task run status to PAUSE: %w", err)
		}
		err = queries.IncrementTaskRunLoops(ts.ctx, taskRun.TaskRunID)
		if err != nil {
			errChan <- fmt.Errorf("failed to increment task run loops: %w", err)
		}
		ts.log.Warn("Task run paused due to max loops reached", "task_run_id", taskRun.TaskRunID, "current_loops", taskRun.CurrentLoops, "max_loops", task.MaxRequestLoop)
		// TODO UPDATE THIS TO ADD WAITING FOR USER FEEDBACK. CURRENTLY SKIP
	} else {
		err := queries.IncrementTaskRunLoops(ts.ctx, taskRun.TaskRunID)
		if err != nil {
			errChan <- fmt.Errorf("failed to increment task run loops: %w", err)
		}
		// Update the task_run_status to RUNNING
		err = queries.UpdateTaskRunStatus(ts.ctx, db.UpdateTaskRunStatusParams{
			TaskRunID: taskRun.TaskRunID,
			Status:    db.TaskRunStatusRunning,
		})
		if err != nil {
			errChan <- fmt.Errorf("failed to update task run status to RUNNING: %w", err)
		}
		ts.log.Info("Updated task run status to RUNNING", "task_run_id", taskRun.TaskRunID)
	}
}
