package api

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pinazu/core/internal/db"
)

const PERMISSION_RESOURCE = "Permission"

// List all permissions
// (GET /v1/permissions)
func (s *Server) ListPermissions(ctx context.Context, request ListPermissionsRequestObject) (ListPermissionsResponseObject, error) {
	permissions, err := s.queries.GetAllPermissions(ctx)
	if err != nil {
		return nil, err
	}
	return ListPermissions200JSONResponse(PermissionList{Permissions: permissions}), nil
}

// Create a new permission
// (POST /v1/permissions)
func (s *Server) CreatePermission(ctx context.Context, request CreatePermissionRequestObject) (CreatePermissionResponseObject, error) {
	// Validate required fields
	if request.Body.Name == "" {
		return CreatePermission400JSONResponse{Message: "name is required"}, nil
	}
	if request.Body.Content == nil {
		return CreatePermission400JSONResponse{Message: "content is required"}, nil
	}

	// Validate field length constraints
	if len(request.Body.Name) > 255 {
		return CreatePermission400JSONResponse{Message: "name exceeds maximum length of 255 characters"}, nil
	}

	// Convert permission_content to JsonRaw
	permissionContentJSON, err := db.NewJsonRaw(request.Body.Content)
	if err != nil {
		return CreatePermission400JSONResponse{Message: fmt.Sprintf("invalid content JSON: %v", err)}, nil
	}

	params := db.CreatePermissionParams{
		Name:    request.Body.Name,
		Content: permissionContentJSON,
	}

	// Set optional description
	if request.Body.Description != nil {
		params.Description = *request.Body.Description
	} else {
		params.Description = pgtype.Text{Valid: false}
	}

	permission, err := s.queries.CreatePermission(ctx, params)
	if err != nil {
		return nil, err
	}

	return CreatePermission201JSONResponse(permission), nil
}

// Get permission by ID
// (GET /v1/permissions/{permission_id})
func (s *Server) GetPermission(ctx context.Context, request GetPermissionRequestObject) (GetPermissionResponseObject, error) {
	permission, err := s.queries.GetPermissionByID(ctx, request.PermissionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetPermission404JSONResponse{Message: "Permission not found", Resource: PERMISSION_RESOURCE, Id: request.PermissionId}, nil
		}
		return nil, err
	}

	return GetPermission200JSONResponse(permission), nil
}

// Update permission
// (PUT /v1/permissions/{permission_id})
func (s *Server) UpdatePermission(ctx context.Context, request UpdatePermissionRequestObject) (UpdatePermissionResponseObject, error) {
	// Validate field length constraints if provided
	if request.Body.Name != nil && len(*request.Body.Name) > 255 {
		return UpdatePermission400JSONResponse{Message: "permission name exceeds maximum length of 255 characters"}, nil
	}

	// Get current permission to preserve existing values for optional fields
	currentPermission, err := s.queries.GetPermissionByID(ctx, request.PermissionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return UpdatePermission404JSONResponse{Message: "Permission not found", Resource: PERMISSION_RESOURCE, Id: request.PermissionId}, nil
		}
		return nil, err
	}

	// Start with current values
	params := db.UpdatePermissionParams{
		ID:          request.PermissionId,
		Name:        currentPermission.Name,
		Description: currentPermission.Description,
		Content:     currentPermission.Content,
	}

	// Update only provided fields
	if request.Body.Name != nil {
		params.Name = *request.Body.Name
	}
	if request.Body.Description != nil {
		params.Description = *request.Body.Description
	}
	if request.Body.Content != nil {
		permissionContentJSON, err := db.NewJsonRaw(*request.Body.Content)
		if err != nil {
			return UpdatePermission400JSONResponse{Message: fmt.Sprintf("invalid permission_content JSON: %v", err)}, nil
		}
		params.Content = permissionContentJSON
	}

	permission, err := s.queries.UpdatePermission(ctx, params)
	if err != nil {
		return nil, err
	}

	return UpdatePermission200JSONResponse(permission), nil
}

// Delete permission
// (DELETE /v1/permissions/{permission_id})
func (s *Server) DeletePermission(ctx context.Context, request DeletePermissionRequestObject) (DeletePermissionResponseObject, error) {
	// Check if permission exists first
	_, err := s.queries.GetPermissionByID(ctx, request.PermissionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeletePermission404JSONResponse{Message: "Permission not found", Resource: PERMISSION_RESOURCE, Id: request.PermissionId}, nil
		}
		return nil, err
	}

	// Delete the permission
	err = s.queries.DeletePermission(ctx, request.PermissionId)
	if err != nil {
		return nil, err
	}

	return DeletePermission204Response{}, nil
}
