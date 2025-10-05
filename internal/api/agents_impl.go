package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pinazu/internal/db"
)

const AGENT_RESOURCE = "Agent"

// List all agents
// (GET /v1/agents)
func (s *Server) ListAgents(ctx context.Context, request ListAgentsRequestObject) (ListAgentsResponseObject, error) {
	agents, err := s.queries.GetAgents(ctx)
	if err != nil {
		return nil, err
	}
	return ListAgents200JSONResponse(AgentList{Agents: agents}), nil
}

// Create a new agent
// (POST /v1/agents)
func (s *Server) CreateAgent(ctx context.Context, request CreateAgentRequestObject) (CreateAgentResponseObject, error) {
	now := time.Now()

	// TODO: should be replaced with the actual user ID from the context or authentication system
	// Parse string to UUID
	createdBy, err := uuid.Parse("550e8400-c95b-4444-6666-446655440000")
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}
	// Required Agent Name
	if request.Body.Name == "" {
		return CreateAgent400JSONResponse{Message: "agent_name is required"}, nil
	}
	// Check length of Agent Name
	if len(request.Body.Name) > 255 {
		return CreateAgent400JSONResponse{Message: "agent_name must be less than 255 characters"}, nil
	}

	// Convert CreateAgentRequest to CreateAgentParams
	params := db.CreateAgentParams{
		Name:        request.Body.Name,
		Description: pgtype.Text{String: "", Valid: false},
		Specs:       pgtype.Text{String: "", Valid: false},
		CreatedBy:   createdBy,
		CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
	}

	// Set optional fields if provided
	if request.Body.Description != nil {
		params.Description = pgtype.Text{String: *request.Body.Description, Valid: true}
	}
	if request.Body.Specs != nil {
		params.Specs = pgtype.Text{String: *request.Body.Specs, Valid: true}
	}

	agent, err := s.queries.CreateAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	return CreateAgent201JSONResponse(agent), nil
}

// Delete agent
// (DELETE /v1/agents/{agent_id})
func (s *Server) DeleteAgent(ctx context.Context, request DeleteAgentRequestObject) (DeleteAgentResponseObject, error) {

	// Check if agent exists
	_, err := s.queries.GetAgentByID(ctx, request.AgentId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeleteAgent404JSONResponse{Message: "Agent not found", Resource: AGENT_RESOURCE, Id: request.AgentId}, nil
		}
		return nil, err
	}
	err = s.queries.DeleteAgent(ctx, request.AgentId)
	if err != nil {
		return nil, err
	}

	return DeleteAgent204Response{}, nil
}

// Get agent by ID
// (GET /v1/agents/{agent_id})
func (s *Server) GetAgent(ctx context.Context, request GetAgentRequestObject) (GetAgentResponseObject, error) {
	agent, err := s.queries.GetAgentByID(ctx, request.AgentId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetAgent404JSONResponse{Message: "Agent not found", Resource: AGENT_RESOURCE, Id: request.AgentId}, nil
		}
		return nil, err
	}

	return GetAgent200JSONResponse(agent), nil
}

// Update agent
// (PUT /v1/agents/{agent_id})
func (s *Server) UpdateAgent(ctx context.Context, request UpdateAgentRequestObject) (UpdateAgentResponseObject, error) {
	// Get current agent to preserve existing values for optional fields
	currentAgent, err := s.queries.GetAgentByID(ctx, request.AgentId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return UpdateAgent404JSONResponse{Message: "Agent not found", Resource: AGENT_RESOURCE, Id: request.AgentId}, nil
		}
		return nil, err
	}

	// Start with current values
	params := db.UpdateAgentParams{
		ID:          request.AgentId,
		Name:        currentAgent.Name,
		Description: currentAgent.Description,
		Specs:       currentAgent.Specs,
	}

	// Update only provided fields
	if request.Body.Name != nil {
		params.Name = *request.Body.Name
	}
	if request.Body.Description != nil {
		params.Description = pgtype.Text{String: *request.Body.Description, Valid: true}
	}
	if request.Body.Specs != nil {
		params.Specs = pgtype.Text{String: *request.Body.Specs, Valid: true}
	}

	agent, err := s.queries.UpdateAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	return UpdateAgent200JSONResponse(agent), nil
}

// List permissions for agent mapping
// (GET /v1/agents/{agent_id}/permissions)
func (s Server) ListPermissionsForAgent(ctx context.Context, request ListPermissionsForAgentRequestObject) (ListPermissionsForAgentResponseObject, error) {
	// Check if agent exists
	_, err := s.queries.GetAgentByID(ctx, request.AgentId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ListPermissionsForAgent404JSONResponse{Message: "Agent not found", Resource: AGENT_RESOURCE, Id: request.AgentId}, nil
		}
		return nil, err
	}
	mappings, err := s.queries.ListPermissionsForAgent(ctx, request.AgentId)
	if err != nil {
		return nil, err
	}

	return ListPermissionsForAgent200JSONResponse{
		PermissionMappings: mappings,
	}, nil
}

// Add permission to agent
// (POST /v1/agents/{agent_id}/permissions)
func (s Server) AddPermissionToAgent(ctx context.Context, request AddPermissionToAgentRequestObject) (AddPermissionToAgentResponseObject, error) {
	// TODO: should be replaced with the actual user ID from the context or authentication system
	var assignedBy uuid.UUID
	if request.Body.AssignedBy != nil {
		assignedBy = *request.Body.AssignedBy
	} else {
		// Default assigned_by if not provided
		var err error
		assignedBy, err = uuid.Parse("550e8400-c95b-4444-6666-446655440000")
		if err != nil {
			return nil, fmt.Errorf("invalid default UUID format: %v", err)
		}
	}

	params := db.AddAgentPermissionParams{
		AgentID:      request.AgentId,
		PermissionID: request.Body.PermissionId,
		AssignedBy:   assignedBy,
	}
	// Nil check for PermissionId
	if request.Body.PermissionId == uuid.Nil {
		return AddPermissionToAgent400JSONResponse{Message: "permission_id is required"}, nil
	}

	// Check if agent exists
	_, err := s.queries.GetAgentByID(ctx, request.AgentId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AddPermissionToAgent404JSONResponse{Message: "Agent not found", Resource: AGENT_RESOURCE, Id: request.AgentId}, nil
		}
		return nil, err
	}
	// Check if permission exists
	_, err = s.queries.GetPermissionByID(ctx, request.Body.PermissionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AddPermissionToAgent404JSONResponse{Message: "Permission not found", Resource: "Permission", Id: request.Body.PermissionId}, nil
		}
		return nil, err
	}

	mapping, err := s.queries.AddAgentPermission(ctx, params)
	if err != nil {
		s.log.Error("Failed to add permission to agent", "error", err, "agent_id", request.AgentId, "permission_id", request.Body.PermissionId)
		if db.IsConflictError(err) {
			return AddPermissionToAgent409JSONResponse{Message: "Permission already exists for this agent", Resource: AGENT_RESOURCE, Id: request.AgentId}, nil
		}
		return nil, err
	}

	return AddPermissionToAgent201JSONResponse(mapping), nil
}

// Remove permission from agent
// (DELETE /v1/agents/{agent_id}/permissions/{permission_id})
func (s Server) RemovePermissionFromAgent(ctx context.Context, request RemovePermissionFromAgentRequestObject) (RemovePermissionFromAgentResponseObject, error) {
	params := db.RemoveAgentPermissionParams{
		AgentID:      request.AgentId,
		PermissionID: request.PermissionId,
	}
	// Check if agent exists
	_, err := s.queries.GetAgentByID(ctx, request.AgentId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return RemovePermissionFromAgent404JSONResponse{Message: "Agent not found", Resource: AGENT_RESOURCE, Id: request.AgentId}, nil
		}
		return nil, err
	}
	// Check if permission exists
	_, err = s.queries.GetPermissionByID(ctx, request.PermissionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return RemovePermissionFromAgent404JSONResponse{Message: "Permission not found", Resource: "Permission", Id: request.PermissionId}, nil
		}
		return nil, err
	}

	// Remove the permission mapping

	err = s.queries.RemoveAgentPermission(ctx, params)
	if err != nil {
		return nil, err
	}

	return RemovePermissionFromAgent204Response{}, nil
}
