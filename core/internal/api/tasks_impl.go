package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go"
	"github.com/pinazu/core/internal/db"
	"github.com/pinazu/core/internal/service"
)

const TASK_RESOURCE = "Task"

func (s *Server) CreateTask(ctx context.Context, req CreateTaskRequestObject) (CreateTaskResponseObject, error) {
	// TODO: should be replaced with the actual user ID from the context or authentication system
	userId, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}

	// Validate required fields
	if req.Body.ThreadId == uuid.Nil {
		return CreateTask400JSONResponse{Message: "thread_id is required"}, nil
	}

	// Validate max_request_loop
	if req.Body.MaxRequestLoop != nil && *req.Body.MaxRequestLoop < 0 {
		return CreateTask400JSONResponse{Message: "max_request_loop cannot be negative"}, nil
	}

	// Check if thread exists
	p := db.GetThreadByIDParams{UserID: userId, ID: req.Body.ThreadId}
	_, err = s.queries.GetThreadByID(ctx, p)
	if err != nil {
		if err == pgx.ErrNoRows {
			return CreateTask404JSONResponse{Resource: "Thread", Id: req.Body.ThreadId, Message: fmt.Sprintf("Thread with ID %s not found", req.Body.ThreadId)}, nil
		}
		return nil, fmt.Errorf("failed to validate thread: %w", err)
	}

	// Validate request
	addInfo, err := db.NewJsonRaw(req.Body.AdditionalInfo)
	if err != nil {
		return CreateTask400JSONResponse{Message: "invalid additional_info"}, nil
	}

	maxRequestLoop := int32(20) // Default value
	if req.Body.MaxRequestLoop != nil {
		maxRequestLoop = int32(*req.Body.MaxRequestLoop)
	}

	params := &db.CreateTaskParams{
		ThreadID:       req.Body.ThreadId,
		MaxRequestLoop: maxRequestLoop,
		AdditionalInfo: addInfo,
		CreatedBy:      uuid.MustParse("550e8400-c95b-4444-6666-446655440000"), // TODO: Get from authentication context
	}

	task, err := s.queries.CreateTask(ctx, *params)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	return CreateTask201JSONResponse(task), nil
}

func (s *Server) GetTask(ctx context.Context, req GetTaskRequestObject) (GetTaskResponseObject, error) {
	// Validate nil UUID
	if req.TaskId == uuid.Nil {
		return GetTask404JSONResponse{Resource: TASK_RESOURCE, Id: req.TaskId, Message: "Task ID cannot be nil"}, nil
	}

	task, err := s.queries.GetTaskById(ctx, req.TaskId.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetTask404JSONResponse{Resource: TASK_RESOURCE, Id: req.TaskId, Message: fmt.Sprintf("Task with ID %s not found", req.TaskId)}, nil
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return GetTask200JSONResponse(task), nil
}

func (s *Server) UpdateTask(ctx context.Context, req UpdateTaskRequestObject) (UpdateTaskResponseObject, error) {
	taskID := req.TaskId
	// Validate request - check if task exists
	task, err := s.queries.GetTaskById(ctx, taskID.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return UpdateTask404JSONResponse{Resource: TASK_RESOURCE, Id: taskID, Message: fmt.Sprintf("Task with ID %s not found", taskID)}, nil
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	params := &db.UpdateTaskParams{
		ID:             taskID.String(),
		MaxRequestLoop: task.MaxRequestLoop,
		AdditionalInfo: task.AdditionalInfo,
	}

	if req.Body.MaxRequestLoop != nil {
		params.MaxRequestLoop = int32(*req.Body.MaxRequestLoop)
	}
	if req.Body.AdditionalInfo != nil {
		addInfo, err := db.NewJsonRaw(req.Body.AdditionalInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal additional info: %w", err)
		}
		params.AdditionalInfo = addInfo
	}

	updatedTask, err := s.queries.UpdateTask(ctx, *params)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	return UpdateTask200JSONResponse(updatedTask), nil
}

func (s *Server) DeleteTask(ctx context.Context, req DeleteTaskRequestObject) (DeleteTaskResponseObject, error) {
	taskID := req.TaskId
	// Validate request - check if task exists
	_, err := s.queries.GetTaskById(ctx, taskID.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeleteTask404JSONResponse{Resource: TASK_RESOURCE, Id: taskID, Message: fmt.Sprintf("Task with ID %s not found", taskID)}, nil
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	err = s.queries.DeleteTask(ctx, taskID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to delete task: %w", err)
	}
	return DeleteTask204Response{}, nil
}

func (s *Server) ListTasks(ctx context.Context, req ListTasksRequestObject) (ListTasksResponseObject, error) {
	params := db.GetTasksParams{
		Limit:  10,
		Offset: 0,
	}
	var page int32 = 1
	if req.Params.PerPage != nil {
		params.Limit = *req.Params.PerPage
	}
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	params.Offset = (page - 1) * params.Limit

	tasks, err := s.queries.GetTasks(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return ListTasks200JSONResponse(TaskList{
		Tasks:      tasks,
		Page:       page,
		PerPage:    params.Limit,
		Total:      len(tasks),
		TotalPages: (len(tasks) + int(params.Limit) - 1) / int(params.Limit),
	}), nil
}

// Get all task runs for a task
// (GET /v1/tasks/{task_id}/runs)
func (s *Server) ListTaskRuns(ctx context.Context, request ListTaskRunsRequestObject) (ListTaskRunsResponseObject, error) {
	// First check if the task exists
	_, err := s.queries.GetTaskById(ctx, request.TaskId.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return ListTaskRuns404JSONResponse{Resource: "Task", Id: request.TaskId, Message: fmt.Sprintf("Task with ID %s not found", request.TaskId)}, nil
		}
		return nil, fmt.Errorf("failed to verify task exists: %w", err)
	}

	// Get task runs for the existing task
	taskRuns, err := s.queries.GetTaskRunByTaskID(ctx, request.TaskId.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get task runs: %w", err)
	}
	return ListTaskRuns200JSONResponse(taskRuns), nil
}

func (s *Server) ExecuteTask(ctx context.Context, req ExecuteTaskRequestObject) (ExecuteTaskResponseObject, error) {
	taskID := req.TaskId
	agentID := req.Body.AgentId
	currentLoops := 0
	if req.Body.CurrentLoops != nil {
		currentLoops = *req.Body.CurrentLoops
	}
	if req.Body.AgentId == uuid.Nil {
		return ExecuteTask400JSONResponse{Message: "agent_id is required"}, nil
	}

	// TODO: should be replaced with the actual user ID from the context or authentication system
	userID, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}

	// Validate task exists and get task details
	task, err := s.queries.GetTaskById(ctx, taskID.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return ExecuteTask404JSONResponse{Resource: TASK_RESOURCE, Id: taskID, Message: fmt.Sprintf("Task with ID %s not found", taskID)}, nil
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// TODO: Will need to update this as a single task can be run multiple time.
	// Check if there's already a running task run for this task
	existingTaskRun, err := s.queries.GetCurrentTaskRunByTaskID(ctx, taskID.String())
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to check for existing task runs: %w", err)
	}
	if err == nil {
		// Task run already exists and is running/scheduled - return error via 404 with custom message
		return ExecuteTask404JSONResponse{Resource: TASK_RESOURCE, Id: taskID, Message: fmt.Sprintf("Task %s already has a running task run with status %s", taskID, existingTaskRun.Status)}, nil
	}

	// Create new task run
	taskRun, err := s.queries.CreateTasksRun(ctx, taskID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create task run: %w", err)
	}

	// Update task run with current loops if provided
	if currentLoops > 0 {
		err = s.queries.UpdateTaskRunCurrentLoops(ctx, db.UpdateTaskRunCurrentLoopsParams{
			TaskRunID:    taskRun.TaskRunID,
			CurrentLoops: int32(currentLoops),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update task run current loops: %w", err)
		}
		// Refresh task run data
		taskRun, err = s.queries.GetTasksRun(ctx, taskRun.TaskRunID)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh task run: %w", err)
		}
	}

	// Get all the messages from the threads
	messages, err := s.queries.GetMessageContents(ctx, task.ThreadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages for thread %s: %w", task.ThreadID, err)
	}

	// Create a pipe for SSE streaming
	pipeReader, pipeWriter := io.Pipe()

	// Create a buffered channel for responses with buffer size of 100 to handle bursts
	responseChan := make(chan *nats.Msg, 100)

	// Subscribe to the user's response subjects using ChanSubscribe
	event := service.WebsocketResponseEventMessage{}
	sub, err := s.nc.ChanSubscribe(event.SubjectWithUser(userID).String(), responseChan)
	if err != nil {
		s.log.Error("Failed to subscribe to response channel", "user_id", userID, "error", err)
		pipeWriter.Close()
		return nil, fmt.Errorf("failed to get response streaming from model: %w", err)
	}

	// Subscribe to task lifecycle events
	taskEvent := service.WebsocketTaskLifecycleEventMessage{}
	taskSub, err := s.nc.ChanSubscribe(taskEvent.SubjectWithUser(userID).String(), responseChan)
	if err != nil {
		s.log.Error("Failed to subscribe to task lifecycle channel", "user_id", userID, "error", err)
		pipeWriter.Close()
		return nil, fmt.Errorf("failed to get response streaming from model: %w", err)
	}

	// Handle incoming messages and stream as SSE
	go func() {
		var taskStatus db.TaskRunStatus = db.TaskRunStatusFailed // Default to failed if something goes wrong

		// Set up heartbeat ticker to keep connection alive
		heartbeatTicker := time.NewTicker(30 * time.Second) // Send heartbeat every 30 seconds
		defer heartbeatTicker.Stop()

		defer func() {
			// Use a separate context for cleanup operations to avoid "context canceled" errors
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// ALWAYS update the TaskRun status when streaming ends, regardless of how it ends
			if err := s.queries.UpdateTaskRunStatus(cleanupCtx, db.UpdateTaskRunStatusParams{
				Status:    taskStatus,
				TaskRunID: taskRun.TaskRunID,
			}); err != nil {
				s.log.Error("Failed to update task run status", "task_run_id", taskRun.TaskRunID, "status", taskStatus, "error", err)
			} else {
				s.log.Debug("Updated task run status", "task_run_id", taskRun.TaskRunID, "status", taskStatus)
			}

			// Cleanup: unsubscribe from NATS and close the pipe writer
			if err := sub.Unsubscribe(); err != nil {
				s.log.Error("Failed to unsubscribe", "user_id", userID, "error", err)
			}
			if err := taskSub.Unsubscribe(); err != nil {
				s.log.Error("Failed to unsubscribe", "user_id", userID, "error", err)
			}
			pipeWriter.Close()
		}()

		for {
			select {
			case <-ctx.Done():
				s.log.Debug("Context cancelled, stopping SSE stream", "user_id", userID)
				taskStatus = db.TaskRunStatusFailed
				return
			case <-heartbeatTicker.C:
				// Send heartbeat to keep connection alive
				heartbeatEvent := fmt.Sprintf("event: heartbeat\ndata: {\"type\":\"heartbeat\",\"timestamp\":\"%s\"}\n\n",
					time.Now().UTC().Format(time.RFC3339))
				if _, err := pipeWriter.Write([]byte(heartbeatEvent)); err != nil {
					s.log.Error("Failed to write heartbeat event", "error", err)
					return
				}
				s.log.Debug("Sent heartbeat to keep SSE connection alive", "user_id", userID)
			case msg, ok := <-responseChan:
				if !ok {
					s.log.Debug("Message channel closed, stopping SSE stream", "user_id", userID)
					// Keep default taskStatus = FAILED
					return
				}

				if !strings.Contains(msg.Subject, "ws.response") {
					s.log.Debug("Received non-ws.response event, stopping SSE stream", "subject", msg.Subject)
					s.log.Debug("Message data", "data", string(msg.Data))

					// Parse task lifecycle events to determine completion status
					if strings.Contains(msg.Subject, "task.lifecycle") {
						// Parse the task lifecycle message to get the event type
						var taskLifecycleData map[string]any
						if err := json.Unmarshal(msg.Data, &taskLifecycleData); err == nil {
							if messageData, ok := taskLifecycleData["message"].(map[string]any); ok {
								if eventTypeRaw, exists := messageData["type"]; exists {
									if eventType, ok := eventTypeRaw.(string); ok {
										s.log.Debug("Received task lifecycle event", "type", eventType, "subject", msg.Subject)

										switch eventType {
										case "task_stop":
											taskStatus = db.TaskRunStatusFinished
											s.log.Debug("Task completed successfully", "task_run_id", taskRun.TaskRunID)
											return
										case "task_error", "task_failed":
											taskStatus = db.TaskRunStatusFailed
											s.log.Debug("Task failed", "task_run_id", taskRun.TaskRunID)
											return
										case "sub_task_start", "sub_task_stop":
											// Ignore since this is for sub task
										default:
											s.log.Debug("Unknown task lifecycle event type", "type", eventType)
											// Keep default taskStatus = FAILED
											return
										}
									}
								}
							}
						}
					}
				}

				// Parse the NATS message into WebsocketResponseEventMessage
				event, _ := service.ParseEvent[*service.WebsocketResponseEventMessage](msg.Data)

				// Convert the response event to JSON for SSE data
				eventData, err := json.Marshal(event.Msg)
				if err != nil {
					s.log.Error("Failed to marshal response event", "error", err)
					continue
				}

				// Format as SSE event with event type and data
				sseEvent := fmt.Sprintf("data: %s\n\n", string(eventData))

				// s.log.Debug("SSE event sent to client: %s", sseEvent)

				// Write the SSE event to the pipe
				if _, err := pipeWriter.Write([]byte(sseEvent)); err != nil {
					s.log.Error("Failed to write SSE event", "error", err)
					return
				}
			}
		}
	}()

	// Now publish the agent invoke event to trigger execution
	agentEvent := service.NewEvent(&service.AgentInvokeEventMessage{
		AgentId:     agentID,
		RecipientId: userID,
		Messages:    messages,
	}, &service.EventHeaders{
		UserID:   userID,
		ThreadID: &task.ThreadID,
		TaskID:   aws.String(taskID.String()),
	}, &service.EventMetadata{
		TraceID:   "", // TODO: Get from request context
		Timestamp: time.Now().UTC(),
	})

	// Publish using the service layer method
	err = agentEvent.Publish(s.nc)
	if err != nil {
		pipeWriter.Close()
		return nil, fmt.Errorf("failed to publish task execute event: %w", err)
	}

	// Return SSE response with proper headers
	return ExecuteTask200TexteventStreamResponse{
		Body: pipeReader,
		Headers: ExecuteTask200ResponseHeaders{
			CacheControl: "no-cache",
			Connection:   "keep-alive",
		},
	}, nil
}

func (s *Server) GetTaskRun(ctx context.Context, req GetTaskRunRequestObject) (GetTaskRunResponseObject, error) {
	taskRun, err := s.queries.GetTasksRun(ctx, req.TaskRunId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetTaskRun404JSONResponse{Resource: "TaskRun", Id: req.TaskRunId, Message: fmt.Sprintf("TaskRun with ID %s not found", req.TaskRunId)}, nil
		}
		return nil, fmt.Errorf("failed to get task run: %w", err)
	}
	return GetTaskRun200JSONResponse(taskRun), nil
}
