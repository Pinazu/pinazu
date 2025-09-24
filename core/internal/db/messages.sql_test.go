package db

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestCRUDMessages(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)

	// First create a user for the thread
	createUserParams := CreateUserParams{
		Name:           "testuser_messages_crud_unique",
		Email:          "messages_crud_unique@example.com",
		AdditionalInfo: JsonRaw{},
		PasswordHash:   "hashedpassword123",
		ProviderName:   ProviderNameLocal,
	}

	createdUser, err := queries.CreateUser(t.Context(), createUserParams)
	if err != nil {
		// Handle duplicate key error by getting existing user
		t.Logf("User already exists, using existing user: %v", err)
		user, err := queries.GetUserByEmail(t.Context(), createUserParams.Email)
		if err != nil {
			t.Fatalf("Failed to get existing user by email: %v", err)
		}
		// Convert GetUserByEmailRow to CreateUserRow
		createdUser = CreateUserRow(user)
	}
	userID := createdUser.ID

	// Create a thread for the messages

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	createThreadParams := CreateThreadParams{
		Title:     "Test Thread",
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    userID,
	}

	createdThread, err := queries.CreateThread(t.Context(), createThreadParams)
	if err != nil {
		t.Fatalf("Failed to create test thread: %v", err)
	}

	// Create a new message
	messageJsonRaw, _ := NewJsonRaw(map[string]any{"content": "Test message", "type": "text"})

	createParams := CreateUserMessageParams{
		ThreadID:    createdThread.ID,
		Message:     messageJsonRaw,
		SenderID:    uuid.New(),
		RecipientID: uuid.New(),
	}

	createdMessage, err := queries.CreateUserMessage(t.Context(), createParams)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	var createdMessageMap map[string]any
	_ = json.Unmarshal(createdMessage.Message, &createdMessageMap)
	var createParamsMessageMap map[string]any
	_ = json.Unmarshal(createParams.Message, &createParamsMessageMap)

	assert.Equal(t, createParams.ThreadID, createdMessage.ThreadID, "Created message thread ID should match the input thread ID")
	assert.Equal(t, createParamsMessageMap["content"], createdMessageMap["content"], "Created message content should match the input content")
	assert.Equal(t, "user", string(createdMessage.SenderType), "Created message sender type should be 'user'")

	// Test GetMessageByID
	message, err := queries.GetMessageByID(t.Context(), createdMessage.ID)
	if err != nil {
		t.Fatalf("Failed to get message by ID: %v", err)
	}
	assert.NotNil(t, message, "Message should not be nil")
	assert.Equal(t, createdMessage.ID, message.ID, "Retrieved message ID should match the created message ID")
	assert.Equal(t, "user", string(message.SenderType), "Retrieved message sender type should be 'user'")

	// Test GetMessages
	messages, err := queries.GetMessages(t.Context(), createdThread.ID)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	assert.NotEmpty(t, messages, "Messages should not be empty")
	assert.Greater(t, len(messages), 0, "There should be at least one message")
	assert.Equal(t, createdMessage.ID, messages[0].ID, "First message should match created message")

	// Test GetMessageContents
	messageContents, err := queries.GetMessageContents(t.Context(), createdThread.ID)
	if err != nil {
		t.Fatalf("Failed to get message contents: %v", err)
	}
	assert.NotEmpty(t, messageContents, "Message contents should not be empty")
	assert.Greater(t, len(messageContents), 0, "There should be at least one message content")

	// Test UpdateMessage
	messageJsonRaw, _ = NewJsonRaw(map[string]any{"content": "Updated test message", "type": "text"})

	updateParams := UpdateMessageParams{
		Message: messageJsonRaw,
		ID:      createdMessage.ID,
	}
	msg, err := queries.UpdateMessage(t.Context(), updateParams)
	if err != nil {
		t.Fatalf("Failed to update message: %v", err)
	}
	assert.Equal(t, createdMessage.ID, msg.ID, "Updated message ID should match")

	// Verify update
	updatedMessage, err := queries.GetMessageByID(t.Context(), createdMessage.ID)
	if err != nil {
		t.Fatalf("Failed to get updated message: %v", err)
	}

	var updatedMessageMap map[string]any
	_ = json.Unmarshal(updatedMessage.Message, &updatedMessageMap)
	var updatedParamsMessageMap map[string]any
	_ = json.Unmarshal(updateParams.Message, &updatedParamsMessageMap)

	assert.Equal(t, updatedParamsMessageMap["content"], updatedMessageMap["content"], "Updated message content should match")

	// Test DeleteMessage
	err = queries.DeleteMessage(t.Context(), createdMessage.ID)
	if err != nil {
		t.Fatalf("Failed to delete message: %v", err)
	}

	// Verify deletion
	_, err = queries.GetMessageByID(t.Context(), createdMessage.ID)
	if err == nil {
		t.Fatalf("Expected error when getting deleted message, but got none")
	}
	assert.EqualError(t, err, pgx.ErrNoRows.Error(), "Expected no rows error when getting deleted message")
	assert.Equal(t, err, pgx.ErrNoRows, "Expected no rows error when getting deleted message")

	// Clean up thread
	err = queries.DeleteThread(t.Context(), createdThread.ID)
	if err != nil {
		t.Fatalf("Failed to clean up test thread: %v", err)
	}
}
