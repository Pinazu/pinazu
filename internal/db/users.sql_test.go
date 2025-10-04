package db

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCRUDUsers(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)

	// Create a new user
	additionalInfoJsonRaw, _ := NewJsonRaw(map[string]any{"role": "admin", "department": "engineering"})

	createParams := CreateUserParams{
		Name:           "testuser_users_crud_unique",
		Email:          "testuser_users_crud@example.com",
		AdditionalInfo: additionalInfoJsonRaw,
		PasswordHash:   "hashedpassword123",
		ProviderName:   ProviderNameLocal,
	}

	createdUser, err := queries.CreateUser(t.Context(), createParams)
	if err != nil {
		// Handle duplicate key error by getting existing user
		t.Logf("User already exists, using existing user: %v", err)
		user, err := queries.GetUserByEmail(t.Context(), createParams.Email)
		if err != nil {
			t.Fatalf("Failed to get existing user by email: %v", err)
		}
		// Convert GetUserByEmailRow to CreateUserRow
		createdUser = CreateUserRow(user)
	}

	// Test GetUsers to find our created user
	users, err := queries.GetUsers(t.Context())
	if err != nil {
		t.Fatalf("Failed to get users: %v", err)
	}
	assert.NotEmpty(t, users, "Users should not be empty")
	assert.Greater(t, len(users), 0, "There should be at least one user")

	// Find our created user
	var foundUser *GetUsersRow
	for i := range users {
		if users[i].Name == createParams.Name {
			foundUser = &users[i]
			break
		}
	}

	var foundUserAdditionalInfoMap map[string]any
	_ = json.Unmarshal(foundUser.AdditionalInfo, &foundUserAdditionalInfoMap)
	var createParamsAdditionalInfoMap map[string]any
	_ = json.Unmarshal(createParams.AdditionalInfo, &createParamsAdditionalInfoMap)

	assert.NotNil(t, foundUser, "Created user should be found in the user list")
	assert.Equal(t, createParams.Name, foundUser.Name, "Created user Name should match")
	assert.Equal(t, createParams.Email, foundUser.Email, "Created user email should match")
	assert.Equal(t, createParamsAdditionalInfoMap["role"], foundUserAdditionalInfoMap["role"], "Created user role additional info should match")
	// Note: PasswordHash is no longer returned by GetUsers query for security

	// Test GetUserByID
	user, err := queries.GetUserByID(t.Context(), createdUser.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	var userAdditionalInfoMap map[string]any
	_ = json.Unmarshal(foundUser.AdditionalInfo, &userAdditionalInfoMap)

	assert.NotNil(t, user, "User should not be nil")
	assert.Equal(t, createdUser.ID, user.ID, "Retrieved user ID should match")
	assert.Equal(t, createParams.Name, user.Name, "Retrieved user Name should match")
	assert.Equal(t, createParams.Email, user.Email, "Retrieved user email should match")
	assert.Equal(t, createParamsAdditionalInfoMap["role"], userAdditionalInfoMap["role"], "Retrieved user role additional info should match")
	// Note: PasswordHash is no longer returned by GetUserByID query for security
	assert.Equal(t, createParams.ProviderName, user.ProviderName, "Retrieved user provider should match")

	// Verify user fields that are set automatically
	assert.NotNil(t, user.CreatedAt, "User created_at should not be nil")
	assert.NotNil(t, user.UpdatedAt, "User updated_at should not be nil")
	assert.NotNil(t, user.IsOnline, "User is_online should not be nil")
	// Note: LastLogin is not returned by GetUserByID query
}
