package websocket

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/db"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/utils"
)

// Common test setup for database queries
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db_pool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		t.Fatalf("Failed to connect to the database: %v", err)
	}
	return db_pool
}

// Common test setup for nats connection
func setupTestNats(t *testing.T) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	return nc
}

// Common test setup for logger
func setupTestLogger(t *testing.T) hclog.Logger {
	t.Helper()
	logger := hclog.New(nil)
	return logger
}

func TestWebsocketHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	dbPool := setupTestDB(t)
	defer dbPool.Close()

	nc := setupTestNats(t)
	defer nc.Close()

	log := setupTestLogger(t)

	ctx := context.Background()
	handler := NewHandler(ctx, dbPool, nc, utils.NewSyncMap[uuid.UUID, *websocket.Conn](), log)

	tests := []struct {
		name          string
		clientMessage HandlerRequestMessage
		expectNATSMsg bool
		expectError   bool
		description   string
	}{
		{
			name:        "valid_websocket_message",
			description: "Should accept valid websocket message and publish to NATS",
			clientMessage: HandlerRequestMessage{
				AgentID:  uuid.New(),
				ThreadId: &[]uuid.UUID{uuid.New()}[0],
				Messages: []db.JsonRaw{
					db.JsonRaw(`{"role": "user", "content": "Hello"}`),
				},
			},
			expectNATSMsg: true,
			expectError:   false,
		},
		{
			name:        "invalid_messages",
			description: "Should handle invalid messages gracefully",
			clientMessage: HandlerRequestMessage{
				AgentID:  uuid.New(),
				ThreadId: &[]uuid.UUID{uuid.New()}[0],
				Messages: []db.JsonRaw{
					db.JsonRaw(`{"role": "user"}`), // Invalid JSON
				},
			},
			expectNATSMsg: false,
			expectError:   false, // Handler continues on invalid messages
		},
		{
			name:        "invalid_agent_id",
			description: "Should handle invalid agent ID gracefully",
			clientMessage: HandlerRequestMessage{
				AgentID:  uuid.Nil, // Invalid UUID
				ThreadId: &[]uuid.UUID{uuid.New()}[0],
				Messages: []db.JsonRaw{
					db.JsonRaw(`{"role": "user", "content": "Hello"}`),
				},
			},
			expectNATSMsg: false,
			expectError:   false, // Handler continues on invalid messages
		},
		{
			name:        "invalid_thread_id",
			description: "Should handle invalid thread ID gracefully",
			clientMessage: HandlerRequestMessage{
				AgentID:  uuid.New(),
				ThreadId: &[]uuid.UUID{uuid.Nil}[0], // Invalid UUID
				Messages: []db.JsonRaw{
					db.JsonRaw(`{"role": "user", "content": "Hello"}`),
				},
			},
			expectNATSMsg: false,
			expectError:   false, // Handler continues on invalid messages
		},
		{
			name:        "empty_messages",
			description: "Should handle empty messages array",
			clientMessage: HandlerRequestMessage{
				AgentID:  uuid.New(),
				ThreadId: &[]uuid.UUID{uuid.New()}[0],
				Messages: []db.JsonRaw{}, // Empty messages
			},
			expectNATSMsg: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(handler)
			defer server.Close()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
			// Convert HTTP URL to WebSocket URL

			// Set up NATS subscription to capture published messages
			var receivedMsg *service.Event[*service.TaskExecuteEventMessage]
			if tt.expectNATSMsg {
				sub, err := nc.Subscribe(service.TaskExecuteEventSubject.String(), func(msg *nats.Msg) {
					var event service.Event[*service.TaskExecuteEventMessage]
					if err := json.Unmarshal(msg.Data, &event); err == nil {
						receivedMsg = &event
					}
				})
				if err != nil {
					t.Fatalf("Failed to subscribe to NATS: %v", err)
				}
				defer sub.Unsubscribe()
			}

			// Connect to WebSocket
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(ctx, wsURL, nil)

			if err != nil {
				t.Fatalf("Failed to connect to WebSocket: %v", err)
			}
			defer conn.Close(websocket.StatusNormalClosure, "test complete")

			// Send test message
			msgBytes, err := json.Marshal(tt.clientMessage)
			if err != nil {
				t.Fatalf("Failed to marshal test message: %v", err)
			}

			err = conn.Write(ctx, websocket.MessageText, msgBytes)
			if err != nil {
				if tt.expectError {
					return // Expected error occurred
				}
				t.Fatalf("Failed to write message: %v", err)
			}

			// Give some time for message processing
			time.Sleep(100 * time.Millisecond)

			// Verify NATS message was published if expected
			if tt.expectNATSMsg {
				if receivedMsg == nil {
					t.Errorf("Expected NATS message but none received")
					return
				}

				// Verify the published message structure
				if receivedMsg.Msg.AgentId != tt.clientMessage.AgentID {
					t.Errorf("Expected AgentID %v, got %v", tt.clientMessage.AgentID, receivedMsg.Msg.AgentId)
				}

				if receivedMsg.H.ThreadID == nil || *receivedMsg.H.ThreadID != *tt.clientMessage.ThreadId {
					t.Errorf("Expected ThreadID %v, got %v", tt.clientMessage.ThreadId, receivedMsg.H.ThreadID)
				}

				if len(receivedMsg.Msg.Messages) != len(tt.clientMessage.Messages) {
					t.Errorf("Expected %d messages, got %d", len(tt.clientMessage.Messages), len(receivedMsg.Msg.Messages))
				}

				if receivedMsg.H.ConnectionID == nil {
					t.Errorf("Expected ConnectionID to be set")
				}

				if receivedMsg.H.UserID == uuid.Nil {
					t.Errorf("Expected UserID to be set")
				}
			}
		})
	}
}

func TestWebsocketHandler_ServeHTTP_ConnectionHandling(t *testing.T) {
	t.Parallel()

	dbPool := setupTestDB(t)
	defer dbPool.Close()

	nc := setupTestNats(t)
	defer nc.Close()

	log := setupTestLogger(t)

	syncMap := utils.NewSyncMap[uuid.UUID, *websocket.Conn]()
	ctx := context.Background()
	handler := NewHandler(ctx, dbPool, nc, syncMap, log)

	// Create a test server
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	// Convert HTTP URL to WebSocket URL
	t.Logf("WebSocket URL: %s", wsURL)

	t.Run("connection_establishment", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
			OnPongReceived: func(ctx context.Context, payload []byte) {
				t.Logf("Client: Pong received, payload: %s", string(payload))
			},
		})

		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "test complete")
		go func() {
			for {
				// Read messages to keep the connection alive
				_, _, err := conn.Read(ctx)
				if err != nil {
					t.Logf("Connection closed: %v", err)
					return
				}
			}
		}()

		// Connection should be established successfully
		// Send a ping to verify connection is active
		err = conn.Ping(ctx)
		if err != nil {
			t.Errorf("Connection ping failed: %v", err)
		}
	})

	t.Run("connection_id_saved_to_syncmap", func(t *testing.T) {
		// Count initial connections in syncMap
		initialCount := 0
		syncMap.Range(func(key uuid.UUID, value *websocket.Conn) bool {
			initialCount++
			return true
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "test complete")

		// Give some time for the connection to be established and stored
		time.Sleep(100 * time.Millisecond)

		// Verify that a new connection was added to the syncMap
		currentCount := 0
		var foundConnectionID uuid.UUID
		var foundConn *websocket.Conn
		syncMap.Range(func(key uuid.UUID, value *websocket.Conn) bool {
			currentCount++
			foundConnectionID = key
			foundConn = value
			return true
		})

		if currentCount != initialCount+1 {
			t.Errorf("Expected syncMap to contain %d connection(s), got %d", initialCount+1, currentCount)
		}

		if foundConnectionID == uuid.Nil {
			t.Errorf("Expected valid connection ID to be stored in syncMap")
		}

		if foundConn == nil {
			t.Errorf("Expected websocket connection to be stored in syncMap")
		}

		// Verify we can load the connection by ID
		loadedConn, ok := syncMap.Load(foundConnectionID)
		if !ok {
			t.Errorf("Expected to be able to load connection by ID from syncMap")
		}

		if loadedConn != foundConn {
			t.Errorf("Expected loaded connection to match the stored connection")
		}

		t.Logf("Connection ID %v successfully stored and retrieved from syncMap", foundConnectionID)
	})

	t.Run("connection_cleanup_removes_from_syncmap", func(t *testing.T) {
		// Count initial connections in syncMap
		initialCount := 0
		syncMap.Range(func(key uuid.UUID, value *websocket.Conn) bool {
			initialCount++
			return true
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}

		// Give some time for the connection to be established and stored
		time.Sleep(100 * time.Millisecond)

		// Verify connection was added
		connectionCount := 0
		syncMap.Range(func(key uuid.UUID, value *websocket.Conn) bool {
			connectionCount++
			return true
		})

		if connectionCount != initialCount+1 {
			t.Errorf("Expected syncMap to contain %d connection(s), got %d", initialCount+1, connectionCount)
		}

		// Close the connection
		conn.Close(websocket.StatusNormalClosure, "test complete")

		// Give some time for cleanup to occur
		time.Sleep(200 * time.Millisecond)

		// Verify connection was removed from syncMap
		finalCount := 0
		syncMap.Range(func(key uuid.UUID, value *websocket.Conn) bool {
			finalCount++
			return true
		})

		if finalCount != initialCount {
			t.Errorf("Expected syncMap to contain %d connection(s) after cleanup, got %d", initialCount, finalCount)
		}

		t.Logf("Connection successfully removed from syncMap after cleanup")
	})

	t.Run("malformed_json_handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, _, err := websocket.Dial(ctx, wsURL, nil)

		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "test complete")

		// Send malformed JSON
		err = conn.Write(ctx, websocket.MessageText, []byte(`{"invalid": json`))
		if err != nil {
			t.Fatalf("Failed to write malformed JSON: %v", err)
		}
		go func() {
			for {
				// Read messages to keep the connection alive
				_, _, err := conn.Read(ctx)
				if err != nil {
					t.Logf("Connection closed: %v", err)
					return
				}
			}
		}()

		// Connection should remain open despite malformed JSON
		time.Sleep(50 * time.Millisecond)
		err = conn.Ping(ctx)
		if err != nil {
			t.Errorf("Connection should remain open after malformed JSON: %v", err)
		}
	})
}
