package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pinazu/core/internal/db"
)

const THREAD_RESOURCE = "Thread"

// List all threads
// (GET /v1/threads)
func (s *Server) ListThreads(ctx context.Context, request ListThreadsRequestObject) (ListThreadsResponseObject, error) {

	// TODO: should be replaced with the actual user ID from the context or authentication system
	// Parse string to UUID
	userId, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}

	threads, err := s.queries.GetThreads(ctx, userId)
	if err != nil {
		return nil, err
	}
	return ListThreads200JSONResponse(ThreadList{Threads: threads}), nil
}

// Create a new thread
// (POST /v1/threads)
func (s *Server) CreateThread(ctx context.Context, request CreateThreadRequestObject) (CreateThreadResponseObject, error) {
	// Validate required fields
	if request.Body.Title == "" {
		return CreateThread400JSONResponse{Message: "thread title is required"}, nil
	}
	if request.Body.UserId == uuid.Nil {
		return CreateThread400JSONResponse{Message: "user_id is required"}, nil
	}

	// Check length of thread title
	if len(request.Body.Title) > 255 {
		return CreateThread400JSONResponse{Message: "thread title must be less than 255 characters"}, nil
	}

	now := time.Now()

	params := db.CreateThreadParams{
		Title:     request.Body.Title,
		UserID:    request.Body.UserId,
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}

	thread, err := s.queries.CreateThread(ctx, params)
	if err != nil {
		return nil, err
	}

	return CreateThread201JSONResponse(thread), nil
}

// Delete thread
// (DELETE /v1/threads/{thread_id})
func (s *Server) DeleteThread(ctx context.Context, request DeleteThreadRequestObject) (DeleteThreadResponseObject, error) {
	// Check if thread exists first
	userId, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}

	params := db.GetThreadByIDParams{
		UserID: userId,
		ID:     request.ThreadId,
	}

	_, err = s.queries.GetThreadByID(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeleteThread404JSONResponse{Message: "Thread not found", Resource: THREAD_RESOURCE, Id: request.ThreadId}, nil
		}
		return nil, err
	}

	err = s.queries.DeleteThread(ctx, request.ThreadId)
	if err != nil {
		return nil, err
	}

	return DeleteThread204Response{}, nil
}

// Get thread by ID
// (GET /v1/threads/{thread_id})
func (s *Server) GetThread(ctx context.Context, request GetThreadRequestObject) (GetThreadResponseObject, error) {

	// TODO: should be replaced with the actual user ID from the context or authentication system
	// Parse string to UUID
	userId, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}

	params := db.GetThreadByIDParams{
		UserID: userId,
		ID:     request.ThreadId,
	}

	thread, err := s.queries.GetThreadByID(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetThread404JSONResponse{Message: "Thread not found", Resource: THREAD_RESOURCE, Id: request.ThreadId}, nil
		}
		return nil, err
	}

	return GetThread200JSONResponse(thread), nil
}

// Update thread title
// (PUT /v1/threads/{thread_id})
func (s *Server) UpdateThreadTitle(ctx context.Context, request UpdateThreadTitleRequestObject) (UpdateThreadTitleResponseObject, error) {
	// Validate required fields
	if request.Body.Title == "" {
		return UpdateThreadTitle400JSONResponse{Message: "title is required"}, nil
	}

	// Check length of thread title
	if len(request.Body.Title) > 255 {
		return UpdateThreadTitle400JSONResponse{Message: "title must be less than 255 characters"}, nil
	}

	// Check if thread exists first
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
			return UpdateThreadTitle404JSONResponse{Message: "Thread not found", Resource: THREAD_RESOURCE, Id: request.ThreadId}, nil
		}
		return nil, err
	}

	params := db.UpdateThreadParams{
		ID:    request.ThreadId,
		Title: request.Body.Title,
	}

	thread, err := s.queries.UpdateThread(ctx, params)
	if err != nil {
		return nil, err
	}

	return UpdateThreadTitle200JSONResponse(thread), nil
}
