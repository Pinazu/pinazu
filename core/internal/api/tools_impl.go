package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pinazu/core/internal/db"
)

// List all tools
// (GET /v1/tools)
func (s *Server) ListTools(ctx context.Context, request ListToolsRequestObject) (ListToolsResponseObject, error) {
	tools, err := s.queries.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	return ListTools200JSONResponse{
		Tools: tools,
	}, nil
}

// Create a new tool
// (POST /v1/tools)
func (s *Server) CreateTool(ctx context.Context, request CreateToolRequestObject) (CreateToolResponseObject, error) {
	// TODO: should be replaced with the actual user ID from the context or authentication system
	createdBy, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid default UUID format: %v", err)
	}

	if request.Body == nil {
		return CreateTool400JSONResponse{Message: "body is required"}, nil
	}

	if request.Body.Name == "" {
		return CreateTool400JSONResponse{Message: "name is required"}, nil
	}

	// Try to parse the tool type from the union
	// Validate the inner configuration
	if err := request.Body.Config.Validate(); err != nil {
		return CreateTool400JSONResponse{Message: err.Error()}, nil
	}

	// Create the base tool
	createToolParams := db.CreateToolParams{
		Name:        request.Body.Name,
		Description: pgtype.Text{Valid: false},
		Config:      request.Body.Config,
		CreatedBy:   createdBy,
	}

	if request.Body.Description != nil && request.Body.Description.Valid {
		createToolParams.Description = *request.Body.Description
	}

	tool, err := s.queries.CreateTool(ctx, createToolParams)
	if err != nil {
		return nil, err
	}
	return CreateTool201JSONResponse(tool), nil
}

// Delete a tool
// (DELETE /v1/tools/{tool_id})
func (s *Server) DeleteTool(ctx context.Context, request DeleteToolRequestObject) (DeleteToolResponseObject, error) {
	// Check if the tool exists
	_, err := s.queries.GetToolById(ctx, request.ToolId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeleteTool404JSONResponse{
				Message:  "Tool not found",
				Resource: "Tool",
				Id:       request.ToolId,
			}, nil
		}
		return nil, err
	}
	err = s.queries.DeleteTool(ctx, request.ToolId)
	if err != nil {
		return nil, err
	}

	return DeleteTool204Response{}, nil
}

// Get a tool by ID
// (GET /v1/tools/{tool_id})
func (s *Server) GetToolById(ctx context.Context, request GetToolByIdRequestObject) (GetToolByIdResponseObject, error) {
	tool, err := s.queries.GetToolById(ctx, request.ToolId)
	if err != nil {
		// Check if the error is a not found error
		if err == pgx.ErrNoRows {
			return GetToolById404JSONResponse{
				Message:  "Tool not found",
				Resource: "Tool",
				Id:       request.ToolId,
			}, nil
		}
		return nil, err
	}
	return GetToolById200JSONResponse(tool), nil
}

// Update a tool
// (PUT /v1/tools/{tool_id})
func (s *Server) UpdateTool(ctx context.Context, request UpdateToolRequestObject) (UpdateToolResponseObject, error) {
	if request.Body == nil {
		return UpdateTool404JSONResponse{}, nil
	}

	// Get current tool to preserve existing values for optional fields
	currentToolRow, err := s.queries.GetToolById(ctx, request.ToolId)
	if err != nil {
		return UpdateTool404JSONResponse{}, nil
	}

	// Start with current values
	params := db.UpdateToolParams{
		ID:          request.ToolId,
		Description: currentToolRow.Description,
		Config:      currentToolRow.Config,
	}

	// Update only provided fields
	if request.Body.Description != nil {
		params.Description = *request.Body.Description
	}

	// Handle tool configuration updates if provided
	if request.Body.Config != nil {
		// Try to parse the tool type from the union and update configuration
		params.Config = *request.Body.Config
	}

	// Update the base tool
	tool, err := s.queries.UpdateTool(ctx, params)
	if err != nil {
		return nil, err
	}

	return UpdateTool200JSONResponse(tool), nil
}
