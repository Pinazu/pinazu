package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/pinazu/core/internal/db"
	"github.com/pinazu/core/internal/service"
	"github.com/pinazu/core/internal/utils"
)

type (
	// Handler handles WebSocket connections and messages
	Handler struct {
		log     hclog.Logger
		nc      *nats.Conn
		queries *db.Queries
		wsMap   *utils.SyncMap[uuid.UUID, *websocket.Conn]
		resMap  *utils.SyncMap[uuid.UUID, chan *nats.Msg]
		ctx     context.Context
	}

	// HandlerRequestMessage represents the structure of the message sent from the client
	HandlerRequestMessage struct {
		AgentID  uuid.UUID    `json:"agent_id"`
		ThreadId *uuid.UUID   `json:"thread_id"`
		Messages []db.JsonRaw `json:"messages"`
	}
)

func NewHandler(ctx context.Context, dbPool *pgxpool.Pool, nc *nats.Conn, wsMap *utils.SyncMap[uuid.UUID, *websocket.Conn], log hclog.Logger) *Handler {
	return &Handler{
		log:     log,
		wsMap:   wsMap,
		nc:      nc,
		queries: db.New(dbPool),
		resMap:  utils.NewSyncMap[uuid.UUID, chan *nats.Msg](),
		ctx:     ctx,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Disable origin check for development
		OnPingReceived: func(ctx context.Context, payload []byte) bool {
			h.log.Debug("Ping received", "payload", string(payload))
			return true // Return true to send a pong response
		},
	})
	if err != nil {
		h.log.Error("Failed to accept connection", "error", err)
		return
	}

	// Generate unique connection ID
	connectionID := uuid.New()
	h.wsMap.Store(connectionID, conn)
	h.log.Debug("Stored new ws connection: ", "connection_id", connectionID)

	// TODO: Extract userID from request (authentication/authorization)
	// TODO: should be replaced with the actual user ID from the context or authentication system
	userID, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		h.log.Error("Invalid UUID format for user ID", "error", err)
		return
	}

	// Create a buffered channel for responses with buffer size of 100 to handle bursts
	responseChan := make(chan *nats.Msg, 100)

	// Check if channel already exists. If so delete the previous
	if existingChan, exists := h.resMap.Load(userID); exists {
		close(existingChan)
		h.resMap.Delete(userID)
	}
	h.resMap.Store(userID, responseChan)

	// Subscribe to the user's response subjects using ChanSubscribe
	event := service.WebsocketResponseEventMessage{}
	sub, err := h.nc.ChanSubscribe(event.SubjectWithUser(userID).String(), responseChan)
	if err != nil {
		h.log.Error("Failed to subscribe to response channel", "user_id", userID, "error", err)
		return
	}

	// Subscribe to task lifecycle events
	taskEvent := service.WebsocketTaskLifecycleEventMessage{}
	taskSub, err := h.nc.ChanSubscribe(taskEvent.SubjectWithUser(userID).String(), responseChan)
	if err != nil {
		h.log.Error("Failed to subscribe to task lifecycle channel", "user_id", userID, "error", err)
		return
	}

	// Start goroutine to handle messages from the channel and forward to WebSocket
	// Use request context so goroutine dies when WebSocket connection closes
	ctx := r.Context()
	go h.handleUserMessages(ctx, responseChan)

	// Ensure cleanup on exit
	defer func() {

		// Unsubscribe from NATS
		if err := sub.Unsubscribe(); err != nil {
			h.log.Error("Failed to unsubscribe", "user_id", userID, "error", err)
		}
		if err := taskSub.Unsubscribe(); err != nil {
			h.log.Error("Failed to unsubscribe from task lifecycle", "user_id", userID, "error", err)
		}

		// Close WebSocket connection
		conn.Close(websocket.StatusNormalClosure, "Connection closed")
		h.wsMap.Delete(connectionID)

		// Clean up user response channel
		if resChan, exists := h.resMap.Load(userID); exists {
			close(resChan)
			h.resMap.Delete(userID)
		}

		h.log.Debug("Connection cleanup completed", "connection_id", connectionID, "user_id", userID)
	}()

	h.log.Info("Websocket connection established", "connection_id", connectionID, "user_id", userID)

	// Handle incoming messages - use simple blocking read for proper ping/pong handling
	for {
		msgType, msg, err := conn.Read(ctx)

		if websocket.CloseStatus(err) != -1 {
			h.log.Debug("Connection closed by client", "connection_id", connectionID, "error", err)
			return
		}
		if err != nil {
			h.log.Error("Failed to read message", "connection_id", connectionID, "error", err)
			if err := conn.Write(ctx, websocket.MessageText, []byte(`{"error":"Failed to read message"}`)); err != nil {
				h.log.Error("Failed to send read message error message", "connection_id", connectionID, "error", err)
			}
			return
		}

		h.log.Debug("Received message", "connection_id", connectionID, "type", msgType, "data", string(msg))

		// Handle different message types
		switch msgType {
		case websocket.MessageText:
			// Parse client message for text messages
			var msgStruct map[string]any
			if err := json.Unmarshal(msg, &msgStruct); err != nil {
				h.log.Error("Failed to parse client message", "connection_id", connectionID, "error", err)
				if err := conn.Write(ctx, websocket.MessageText, []byte(`{"error":"Failed to parse message"}`)); err != nil {
					h.log.Error("Failed to send parse client error message", "connection_id", connectionID, "error", err)
				}
				continue
			}
			// Check if the message is a ping
			if msgStruct["type"] == "ping" {
				// Handle ping message
				if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"pong"}`)); err != nil {
					h.log.Error("Failed to send pong message", "connection_id", connectionID, "error", err)
				}
				h.log.Debug("Sent pong response", "connection_id", connectionID)
				continue
			}
			//Try cast type to WebscoketHandlerRequestMessage
			var websocketHandlerRequestMsg HandlerRequestMessage
			if err := json.Unmarshal(msg, &websocketHandlerRequestMsg); err != nil {
				h.log.Error("Failed to parse client message", "connection_id", connectionID, "error", err)
				if err := conn.Write(ctx, websocket.MessageText, []byte(`{"error":"Invalid message format"}`)); err != nil {
					h.log.Error("Failed to send invalid format error message", "connection_id", connectionID, "error", err)
				}
				continue
			}
			// Process the text message (existing logic)
			if err := h.processTextMessage(connectionID, userID, websocketHandlerRequestMsg); err != nil {
				h.log.Error("Failed to process text message", "connection_id", connectionID, "error", err)
				if err := conn.Write(ctx, websocket.MessageText, []byte(`{"error":"Failed to process message"}`)); err != nil {
					h.log.Error("Failed to send process error message", "connection_id", connectionID, "error", err)
				}
				continue
			}
		case websocket.MessageBinary:
			// For now, we don't handle binary messages
			continue
		default:
			// Other message types (ping/pong are handled automatically by the websocket library)
			continue
		}
	}
}

// processTextMessage send the recieved message from Websocket to NATS with appropriate subject
func (h *Handler) processTextMessage(connectionID, userId uuid.UUID, websocketHandlerRequestMsg HandlerRequestMessage) error {
	// Create the event using the service layer
	event := service.NewEvent(&service.TaskExecuteEventMessage{
		AgentId:     websocketHandlerRequestMsg.AgentID,
		RecipientId: userId,
		Messages:    websocketHandlerRequestMsg.Messages,
	}, &service.EventHeaders{
		UserID:       userId,
		ThreadID:     websocketHandlerRequestMsg.ThreadId,
		ConnectionID: &connectionID,
	}, &service.EventMetadata{
		TraceID:   "", // Get from traceId set.
		Timestamp: time.Now().UTC(),
	})

	// Publish using the service layer method
	err := event.Publish(h.nc)
	if err != nil {
		return err
	}

	h.log.Debug("Message published to task service", "connection_id", connectionID)
	return nil
}

// handleUserMessages processes NATS messages for a specific user in a dedicated goroutine
// This ensures each user has their own processing pipeline for high throughput and non-blocking operations
func (h *Handler) handleUserMessages(ctx context.Context, msgChan chan *nats.Msg) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				h.log.Debug("Message channel closed, stopping user message handler")
				return
			}

			// Process message in non-blocking way
			if err := h.forwardMessageToWebSocket(msg); err != nil {
				h.log.Error("Failed to forward message to WebSocket", "error", err)
				// Continue processing other messages even if one fails
				continue
			}
		}
	}
}

// forwardMessageToWebSocket handles forwarding a single NATS message to the WebSocket connection
func (h *Handler) forwardMessageToWebSocket(msg *nats.Msg) error {
	// Determine event type based on NATS subject
	if strings.Contains(msg.Subject, "task.lifecycle") {
		return h.forwardTaskLifecycle(msg.Data)
	} else if strings.Contains(msg.Subject, "ws.response") {
		return h.forwardWebSocketResponse(msg.Data)
	}

	h.log.Warn("Unknown WebSocket event subject", "subject", msg.Subject)
	return nil
}

// forwardWebSocketResponse handles AI streaming response events
func (h *Handler) forwardWebSocketResponse(data []byte) error {
	// Parse the event
	event, err := service.ParseEvent[*service.WebsocketResponseEventMessage](data)

	// Get the WebSocket connection
	ws, ok := h.wsMap.Load(*event.H.ConnectionID)
	if !ok {
		// Connection might have been closed, this is not necessarily an error
		h.log.Debug("WebSocket connection not found, skipping message",
			"connection_id", event.H.ConnectionID,
			"user_id", event.H.UserID,
		)
		return nil // Return nil to continue processing other messages
	}

	var responseData []byte

	// Check if the event contains an error
	if err != nil {
		h.log.Debug("Received error event from NATS",
			"connection_id", *event.H.ConnectionID,
			"user_id", event.H.UserID,
			"error", err.Error(),
		)
		// Create simple error response for WebSocket client
		var err error
		responseData, err = json.Marshal(map[string]string{"error": event.Err.Error})
		if err != nil {
			return fmt.Errorf("failed to marshal error response: %w", err)
		}
	} else {
		// Forward the original message data for successful responses
		responseData = data
	}

	// Send response to WebSocket client with timeout to prevent blocking
	writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.Write(writeCtx, websocket.MessageText, responseData); err != nil {
		// WebSocket write failed - connection might be closed
		return fmt.Errorf("failed to write to websocket: %w", err)
	}

	h.log.Debug("Successfully forwarded AI response to WebSocket",
		"connection_id", *event.H.ConnectionID,
		"user_id", event.H.UserID,
		"has_error", event.Err != nil,
	)
	return nil
}

// forwardTaskLifecycle handles task lifecycle events
func (h *Handler) forwardTaskLifecycle(data []byte) error {
	// Parse the event
	event, err := service.ParseEvent[*service.WebsocketTaskLifecycleEventMessage](data)

	// Get the WebSocket connection
	ws, ok := h.wsMap.Load(*event.H.ConnectionID)
	if !ok {
		// Connection might have been closed, this is not necessarily an error
		h.log.Debug("WebSocket connection not found for task event, skipping message",
			"connection_id", event.H.ConnectionID,
			"user_id", event.H.UserID,
		)
		return nil
	}

	var responseData []byte

	// Check if the event contains an error
	if err != nil {
		h.log.Debug("Received task lifecycle error event",
			"connection_id", *event.H.ConnectionID,
			"user_id", event.H.UserID,
			"error", event.Err.Error,
		)
		// Create error response for WebSocket client
		responseData, err = json.Marshal(map[string]any{
			"type":  "task_error",
			"error": event.Err.Error,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal task error response: %w", err)
		}
	} else {
		// Forward the task lifecycle event
		responseData = data
	}

	// Send response to WebSocket client with timeout to prevent blocking
	writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.Write(writeCtx, websocket.MessageText, responseData); err != nil {
		// WebSocket write failed - connection might be closed
		return fmt.Errorf("failed to write task lifecycle to websocket: %w", err)
	}

	h.log.Debug("Successfully forwarded task lifecycle to WebSocket",
		"connection_id", *event.H.ConnectionID,
		"user_id", event.H.UserID,
		"event_type", event.Msg.Type,
		"has_error", event.Err != nil,
	)
	return nil
}
