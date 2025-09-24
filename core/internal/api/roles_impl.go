package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pinazu/core/internal/db"
)

const ROLE_RESOURCE = "Role"

// List all roles
// (GET /v1/roles)
func (s *Server) ListRoles(ctx context.Context, request ListRolesRequestObject) (ListRolesResponseObject, error) {
	roles, err := s.queries.GetAllRoles(ctx)
	if err != nil {
		return nil, err
	}
	return ListRoles200JSONResponse(RoleList{Roles: roles}), nil
}

// Create a new role
// (POST /v1/roles)
func (s *Server) CreateRole(ctx context.Context, request CreateRoleRequestObject) (CreateRoleResponseObject, error) {
	// Validate required fields
	if request.Body.Name == "" {
		return CreateRole400JSONResponse{Message: "role name is required"}, nil
	}

	// Validate field length constraints
	if len(request.Body.Name) > 255 {
		return CreateRole400JSONResponse{Message: "role name exceeds maximum length of 255 characters"}, nil
	}

	params := db.CreateRoleParams{
		Name: request.Body.Name,
	}

	// Set optional fields
	if request.Body.Description != nil {
		params.Description = *request.Body.Description
	} else {
		params.Description = pgtype.Text{Valid: false}
	}

	if request.Body.IsSystemRole != nil {
		params.IsSystem = *request.Body.IsSystemRole
	} else {
		params.IsSystem = pgtype.Bool{Bool: false, Valid: true}
	}

	role, err := s.queries.CreateRole(ctx, params)
	if err != nil {
		return nil, err
	}

	return CreateRole201JSONResponse(role), nil
}

// Get role by ID
// (GET /v1/roles/{role_id})
func (s *Server) GetRole(ctx context.Context, request GetRoleRequestObject) (GetRoleResponseObject, error) {
	role, err := s.queries.GetRoleByID(ctx, request.RoleId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetRole404JSONResponse{Message: "Role not found", Resource: ROLE_RESOURCE, Id: request.RoleId}, nil
		}
		return nil, err
	}

	return GetRole200JSONResponse(role), nil
}

// Update role
// (PUT /v1/roles/{role_id})
func (s *Server) UpdateRole(ctx context.Context, request UpdateRoleRequestObject) (UpdateRoleResponseObject, error) {
	// Validate field length constraints if provided
	if request.Body.Name != nil && len(*request.Body.Name) > 255 {
		return UpdateRole400JSONResponse{Message: "role name exceeds maximum length of 255 characters"}, nil
	}

	// Get current role to preserve existing values for optional fields
	currentRole, err := s.queries.GetRoleByID(ctx, request.RoleId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return UpdateRole404JSONResponse{Message: "Role not found", Resource: ROLE_RESOURCE, Id: request.RoleId}, nil
		}
		return nil, err
	}

	// Start with current values
	params := db.UpdateRoleParams{
		ID:          request.RoleId,
		Name:        currentRole.Name,
		Description: currentRole.Description,
		IsSystem:    currentRole.IsSystem,
	}

	// Update only provided fields
	if request.Body.Name != nil {
		params.Name = *request.Body.Name
	}
	if request.Body.Description != nil {
		params.Description = *request.Body.Description
	}
	if request.Body.IsSystemRole != nil {
		params.IsSystem = *request.Body.IsSystemRole
	}

	role, err := s.queries.UpdateRole(ctx, params)
	if err != nil {
		return nil, err
	}

	return UpdateRole200JSONResponse(role), nil
}

// List permissions for role mapping
// (GET /v1/roles/{role_id}/permissions)
func (s Server) ListPermissionsForRole(ctx context.Context, request ListPermissionsForRoleRequestObject) (ListPermissionsForRoleResponseObject, error) {
	// First check if the role exists
	_, err := s.queries.GetRoleByID(ctx, request.RoleId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ListPermissionsForRole404JSONResponse{Message: "Role not found", Resource: ROLE_RESOURCE, Id: request.RoleId}, nil
		}
		return nil, err
	}

	mappings, err := s.queries.ListPermissionForRole(ctx, request.RoleId)
	if err != nil {
		return nil, err
	}

	return ListPermissionsForRole200JSONResponse(mappings), nil
}

// Add permission to role
// (POST /v1/roles/{role_id}/permissions)
func (s Server) AddPermissionToRole(ctx context.Context, request AddPermissionToRoleRequestObject) (AddPermissionToRoleResponseObject, error) {
	// First check if the role exists
	_, err := s.queries.GetRoleByID(ctx, request.RoleId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AddPermissionToRole404JSONResponse{Message: "Role not found", Resource: ROLE_RESOURCE, Id: request.RoleId}, nil
		}
		return nil, err
	}

	// Validate field
	if request.Body.PermissionId == uuid.Nil {
		return AddPermissionToRole400JSONResponse{Message: "permission_id is required"}, nil
	}

	// Check if the permission is exists
	_, err = s.queries.GetPermissionByID(ctx, request.Body.PermissionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AddPermissionToRole404JSONResponse{Message: "Permission not found", Resource: PERMISSION_RESOURCE, Id: request.Body.PermissionId}, nil
		}
		return nil, err
	}

	// Check if the permission already exists for this role
	checkParams := db.CheckPermissionExistsForRoleParams{
		RoleID:       request.RoleId,
		PermissionID: request.Body.PermissionId,
	}

	exists, err := s.queries.CheckPermissionExistsForRole(ctx, checkParams)
	if err != nil {
		return nil, err
	}

	if exists {
		return AddPermissionToRole409JSONResponse{Message: "Permission already exists in role"}, nil
	}

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

	params := db.AddPermissionToRoleParams{
		RoleID:       request.RoleId,
		PermissionID: request.Body.PermissionId,
		AssignedBy:   assignedBy,
	}

	mapping, err := s.queries.AddPermissionToRole(ctx, params)
	if err != nil {
		return nil, err
	}

	return AddPermissionToRole201JSONResponse(mapping), nil
}

// Remove permission from role
// (DELETE /v1/roles/{role_id}/permissions/{permission_id})
func (s Server) RemovePermissionFromRole(ctx context.Context, request RemovePermissionFromRoleRequestObject) (RemovePermissionFromRoleResponseObject, error) {
	// First check if the role exists
	_, err := s.queries.GetRoleByID(ctx, request.RoleId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return RemovePermissionFromRole404JSONResponse{Message: "Role not found", Resource: ROLE_RESOURCE, Id: request.RoleId}, nil
		}
		return nil, err
	}

	// Check if the permission exists
	_, err = s.queries.GetPermissionByID(ctx, request.PermissionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return RemovePermissionFromRole404JSONResponse{Message: "Permission not found", Resource: PERMISSION_RESOURCE, Id: request.PermissionId}, nil
		}
		return nil, err
	}

	params := db.DeletePermissionFromRoleParams{
		RoleID:       request.RoleId,
		PermissionID: request.PermissionId,
	}

	err = s.queries.DeletePermissionFromRole(ctx, params)
	if err != nil {
		return nil, err
	}

	return RemovePermissionFromRole204Response{}, nil
}

// Delete role
// (DELETE /v1/roles/{role_id})
func (s *Server) DeleteRole(ctx context.Context, request DeleteRoleRequestObject) (DeleteRoleResponseObject, error) {
	// First check if the role exists
	_, err := s.queries.GetRoleByID(ctx, request.RoleId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeleteRole404JSONResponse{Message: "Role not found", Resource: ROLE_RESOURCE, Id: request.RoleId}, nil
		}
		return nil, err
	}

	// Delete the role
	err = s.queries.DeleteRole(ctx, request.RoleId)
	if err != nil {
		return nil, err
	}

	return DeleteRole204Response{}, nil
}
