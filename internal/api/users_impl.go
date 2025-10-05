package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	db "github.com/pinazu/internal/db"
)

const USER_RESOURCE = "User"

// List all users
// (GET /v1/users)
func (s *Server) ListUsers(ctx context.Context, request ListUsersRequestObject) (ListUsersResponseObject, error) {
	users, err := s.queries.GetUsers(ctx)
	if err != nil {
		return nil, err
	}
	return ListUsers200JSONResponse(UserList{Users: users}), nil
}

// Create a new user
// (POST /v1/users)
func (s *Server) CreateUser(ctx context.Context, request CreateUserRequestObject) (CreateUserResponseObject, error) {
	// Validate required fields
	if request.Body.Name == "" {
		return CreateUser400JSONResponse{Message: "user name is required"}, nil
	}
	if request.Body.Email == "" {
		return CreateUser400JSONResponse{Message: "email is too long"}, nil
	}
	if request.Body.PasswordHash == "" {
		return CreateUser400JSONResponse{Message: "password is required"}, nil
	}

	// Validate field length constraints
	if len(request.Body.Name) > 255 {
		return CreateUser400JSONResponse{Message: "name is too long"}, nil
	}
	if len(string(request.Body.Email)) > 255 {
		return CreateUser400JSONResponse{Message: "email is too long"}, nil
	}
	// Convert additional_info to JsonRaw if provided
	var additionalInfoJSON db.JsonRaw
	if request.Body.AdditionalInfo != nil {
		var err error
		additionalInfoJSON, err = db.NewJsonRaw(*request.Body.AdditionalInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal additional_info JSON: %v", err)
		}
	} else {
		var err error
		additionalInfoJSON, err = db.NewJsonRaw(map[string]any{})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal empty additional_info JSON: %v", err)
		}
	}

	params := db.CreateUserParams{
		Name:           request.Body.Name,
		Email:          string(request.Body.Email),
		AdditionalInfo: additionalInfoJSON,
		PasswordHash:   request.Body.PasswordHash,
	}

	// Set optional provider field
	if request.Body.ProviderName != nil {
		params.ProviderName = *request.Body.ProviderName
	} else {
		params.ProviderName = db.ProviderNameLocal
	}

	user, err := s.queries.CreateUser(ctx, params)
	if err != nil {
		// Check for unique constraint violations (409 Conflict)
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" { // unique violation
				if strings.Contains(pgErr.Detail, "name") {
					return CreateUser409JSONResponse{Resource: USER_RESOURCE, Id: uuid.Nil, Message: "Username already exists"}, nil
				}
				if strings.Contains(pgErr.Detail, "email") {
					return CreateUser409JSONResponse{Resource: USER_RESOURCE, Id: uuid.Nil, Message: "Email already exists"}, nil
				}
				return CreateUser409JSONResponse{Resource: USER_RESOURCE, Id: uuid.Nil, Message: "User already exists"}, nil
			}
		}
		return nil, err
	}

	return CreateUser201JSONResponse(user), nil
}

// Get user by ID
// (GET /v1/users/{user_id})
func (s *Server) GetUser(ctx context.Context, request GetUserRequestObject) (GetUserResponseObject, error) {
	user, err := s.queries.GetUserByID(ctx, request.UserId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return GetUser404JSONResponse{Message: "User not found", Resource: USER_RESOURCE, Id: request.UserId}, nil
		}
		return nil, err
	}

	return GetUser200JSONResponse(user), nil
}

// Update user
// (PUT /v1/users/{user_id})
func (s *Server) UpdateUser(ctx context.Context, request UpdateUserRequestObject) (UpdateUserResponseObject, error) {
	// Validate field length constraints if provided
	if request.Body.Username != nil && len(*request.Body.Username) > 255 {
		return UpdateUser400JSONResponse{Message: "username is too long"}, nil
	}
	if request.Body.Email != nil && len(*request.Body.Email) > 255 {
		return UpdateUser400JSONResponse{Message: "email is too long"}, nil
	}

	// Get current user to preserve existing values for optional fields
	currentUser, err := s.queries.GetUserByID(ctx, request.UserId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return UpdateUser404JSONResponse{Message: "User not found", Resource: USER_RESOURCE, Id: request.UserId}, nil
		}
		return nil, err
	}

	// Start with current values
	params := db.UpdateUserParams{
		ID:             request.UserId,
		Name:           currentUser.Name,
		Email:          currentUser.Email,
		AdditionalInfo: currentUser.AdditionalInfo,
		ProviderName:   currentUser.ProviderName,
	}

	// Update only provided fields
	if request.Body.Username != nil {
		params.Name = *request.Body.Username
	}
	if request.Body.Email != nil {
		params.Email = string(*request.Body.Email)
	}
	if request.Body.AdditionalInfo != nil {
		additionalInfoJSON, err := db.NewJsonRaw(request.Body.AdditionalInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal additional_info JSON: %v", err)
		}
		params.AdditionalInfo = additionalInfoJSON
	}
	if request.Body.ProviderName != nil && *request.Body.ProviderName != db.ProviderNameNil {
		params.ProviderName = *request.Body.ProviderName
	}

	user, err := s.queries.UpdateUser(ctx, params)
	if err != nil {
		// Check for unique constraint violations (409 Conflict)
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" { // unique violation
				if strings.Contains(pgErr.Detail, "name") {
					return UpdateUser409JSONResponse{Resource: USER_RESOURCE, Id: request.UserId, Message: "Username already exists"}, nil
				}
				if strings.Contains(pgErr.Detail, "email") {
					return UpdateUser409JSONResponse{Resource: USER_RESOURCE, Id: request.UserId, Message: "Email already exists"}, nil
				}
				return UpdateUser409JSONResponse{Resource: USER_RESOURCE, Id: request.UserId, Message: "User data conflicts"}, nil
			}
		}
		return nil, err
	}

	return UpdateUser200JSONResponse(user), nil
}

// List role for user mapping
// (GET /v1/users/{user_id}/roles)
func (s Server) ListRoleForUser(ctx context.Context, request ListRoleForUserRequestObject) (ListRoleForUserResponseObject, error) {
	// Check if user exists first
	_, err := s.queries.GetUserByID(ctx, request.UserId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ListRoleForUser404JSONResponse{Message: "User not found", Resource: USER_RESOURCE, Id: request.UserId}, nil
		}
		return nil, err
	}

	mappings, err := s.queries.ListRolesForUser(ctx, request.UserId)
	if err != nil {
		return nil, err
	}

	return ListRoleForUser200JSONResponse(mappings), nil
}

// Add role to user
// (POST /v1/users/{user_id}/roles)
func (s Server) AddRoleToUser(ctx context.Context, request AddRoleToUserRequestObject) (AddRoleToUserResponseObject, error) {
	// Validate required fields
	if request.Body.RoleId == uuid.Nil {
		return AddRoleToUser400JSONResponse{Message: "role_id is required"}, nil
	}

	// Check if user exists
	_, err := s.queries.GetUserByID(ctx, request.UserId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AddRoleToUser404JSONResponse{Message: "User not found", Resource: USER_RESOURCE, Id: request.UserId}, nil
		}
		return nil, err
	}

	// Check if role exists
	_, err = s.queries.GetRoleByID(ctx, request.Body.RoleId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AddRoleToUser404JSONResponse{Message: "Role not found", Resource: ROLE_RESOURCE, Id: request.Body.RoleId}, nil
		}
		return nil, err
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
			return AddRoleToUser400JSONResponse{Message: "assigned_by is required"}, nil
		}
	}

	params := db.AddRoleToUserParams{
		UserID:     request.UserId,
		RoleID:     request.Body.RoleId,
		AssignedBy: assignedBy,
	}

	mapping, err := s.queries.AddRoleToUser(ctx, params)
	if err != nil {
		// Check for unique constraint violations (409 Conflict)
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" { // unique violation
				// Check for any reference to the user_role_mapping constraint or table
				if strings.Contains(pgErr.Detail, "uk_user_role_mapping") ||
					strings.Contains(pgErr.Detail, "user_role_mapping") ||
					strings.Contains(pgErr.ConstraintName, "uk_user_role_mapping") {
					return AddRoleToUser409JSONResponse{Resource: "UserRoleMapping", Id: request.UserId, Message: "Role already assigned to user"}, nil
				}
			}
		}
		return nil, err
	}

	return AddRoleToUser201JSONResponse(mapping), nil
}

// Remove role from user
// (DELETE /v1/users/{user_id}/roles/{role_id})
func (s Server) RemoveRoleFromUser(ctx context.Context, request RemoveRoleFromUserRequestObject) (RemoveRoleFromUserResponseObject, error) {
	// Check if user exists
	_, err := s.queries.GetUserByID(ctx, request.UserId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return RemoveRoleFromUser404JSONResponse{Message: "User not found", Resource: USER_RESOURCE, Id: request.UserId}, nil
		}
		return nil, err
	}

	// Check if role exists
	_, err = s.queries.GetRoleByID(ctx, request.RoleId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return RemoveRoleFromUser404JSONResponse{Message: "Role not found", Resource: ROLE_RESOURCE, Id: request.RoleId}, nil
		}
		return nil, err
	}

	// Check if role exists inside

	params := db.RemoveRoleFromUserParams{
		UserID: request.UserId,
		RoleID: request.RoleId,
	}

	err = s.queries.RemoveRoleFromUser(ctx, params)
	if err != nil {
		return nil, err
	}

	return RemoveRoleFromUser204Response{}, nil
}

// Delete user
// (DELETE /v1/users/{user_id})
func (s *Server) DeleteUser(ctx context.Context, request DeleteUserRequestObject) (DeleteUserResponseObject, error) {
	// Check if user exists first
	_, err := s.queries.GetUserByID(ctx, request.UserId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DeleteUser404JSONResponse{Message: "User not found", Resource: USER_RESOURCE, Id: request.UserId}, nil
		}
		return nil, err
	}

	err = s.queries.DeleteUser(ctx, request.UserId)
	if err != nil {
		return nil, err
	}

	return DeleteUser204Response{}, nil
}
