package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	db "github.com/pinazu/internal/db"
)

const MESSAGE_RESOURCE = "Message"

// List all messages in a thread
// (GET /v1/threads/{thread_id}/messages)
func (s *Server) ListMessages(ctx context.Context, request ListMessagesRequestObject) (ListMessagesResponseObject, error) {
	// Check if the thread exists
	// TODO: should be replaced with the actual user ID from the context or authentication system
	userId, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}

	checkParams := db.GetThreadByIDParams{
		UserID: userId,
		ID:     request.ThreadId,
	}

	_, err = s.queries.GetThreadByID(ctx, checkParams)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ListMessages404JSONResponse{Message: "Thread for messages not found", Resource: MESSAGE_RESOURCE, Id: request.ThreadId}, nil
		}
		return nil, err
	}

	messages, err := s.queries.GetMessages(ctx, request.ThreadId)
	if err != nil {
		return nil, err
	}
	return ListMessages200JSONResponse(MessageList{Messages: messages}), nil
}

// Create a new message in a thread
// (POST /v1/threads/{thread_id}/messages)
func (s *Server) CreateMessage(ctx context.Context, request CreateMessageRequestObject) (CreateMessageResponseObject, error) {
	// Check if the thread exists
	// TODO: should be replaced with the actual user ID from the context or authentication system
	userId, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}

	checkParams := db.GetThreadByIDParams{
		UserID: userId,
		ID:     request.ThreadId,
	}

	_, err = s.queries.GetThreadByID(ctx, checkParams)
	if err != nil {
		if err == pgx.ErrNoRows {
			return CreateMessage404JSONResponse{Message: "Thread not found", Resource: THREAD_RESOURCE, Id: request.ThreadId}, nil
		}
		return nil, err
	}

	// Validate required fields
	if request.Body.Message == nil {
		return CreateMessage400JSONResponse{Message: "message is required"}, nil
	}
	if request.Body.SenderId == uuid.Nil {
		return CreateMessage400JSONResponse{Message: "sender_id is required"}, nil
	}
	if request.Body.RecipientId == uuid.Nil {
		return CreateMessage400JSONResponse{Message: "recipient_id is required"}, nil
	}

	// Convert the message JSON to db.JsonRaw
	messageJSON, err := db.NewJsonRaw(request.Body.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message JSON: %v", err)
	}

	params := db.CreateUserMessageParams{
		ThreadID:    request.ThreadId,
		Message:     messageJSON,
		SenderID:    request.Body.SenderId,
		RecipientID: request.Body.RecipientId,
	}

	message, err := s.queries.CreateUserMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	return CreateMessage201JSONResponse(message), nil
}

// Delete message
// (DELETE /v1/threads/{thread_id}/messages/{message_id})
func (s *Server) DeleteMessage(ctx context.Context, request DeleteMessageRequestObject) (DeleteMessageResponseObject, error) {
	// Check if message exists first
	_, err := s.queries.GetMessageByID(ctx, request.MessageId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeleteMessage404JSONResponse{Message: "Message not found", Resource: MESSAGE_RESOURCE, Id: request.MessageId}, nil
		}
		return nil, err
	}

	err = s.queries.DeleteMessage(ctx, request.MessageId)
	if err != nil {
		return nil, err
	}

	return DeleteMessage204Response{}, nil
}

// Get message by ID
// (GET /v1/threads/{thread_id}/messages/{message_id})
func (s *Server) GetMessage(ctx context.Context, request GetMessageRequestObject) (GetMessageResponseObject, error) {
	message, err := s.queries.GetMessageByID(ctx, request.MessageId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetMessage404JSONResponse{Message: "Message not found", Resource: MESSAGE_RESOURCE, Id: request.MessageId}, nil
		}
		return nil, err
	}

	return GetMessage200JSONResponse(message), nil
}

// Update message
// (PUT /v1/threads/{thread_id}/messages/{message_id})
func (s *Server) UpdateMessage(ctx context.Context, request UpdateMessageRequestObject) (UpdateMessageResponseObject, error) {
	// Validate required fields
	if request.Body.Message == nil {
		return UpdateMessage400JSONResponse{Message: "message is required"}, nil
	}

	// Check if message exists first
	_, err := s.queries.GetMessageByID(ctx, request.MessageId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return UpdateMessage404JSONResponse{Message: "Message not found", Resource: MESSAGE_RESOURCE, Id: request.MessageId}, nil
		}
		return nil, err
	}

	// Convert the message JSON to db.JsonRaw
	messageJSON, err := db.NewJsonRaw(request.Body.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message JSON: %v", err)
	}

	params := db.UpdateMessageParams{
		ID:      request.MessageId,
		Message: messageJSON,
	}

	message, err := s.queries.UpdateMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	return UpdateMessage200JSONResponse(message), nil
}
