package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCRUDTools(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)

	// First create a user who will create the tools
	createUserParams := CreateUserParams{
		Name:           "testuser_tools_crud_unique",
		Email:          "tools_crud_unique@example.com",
		AdditionalInfo: JsonRaw{},
		PasswordHash:   "hashedpassword123",
		ProviderName:   ProviderNameLocal,
	}
	createdUser, err := queries.CreateUser(t.Context(), createUserParams)
	if err != nil {
		// Handle duplicate key error by getting existing user
		t.Logf("User already exists, using existing user: %v", err)
		user, err := queries.GetUserByEmail(t.Context(), createUserParams.Email)
		require.NoError(t, err, "Failed to get existing user by email")
		// Convert GetUserByEmailRow to CreateUserRow
		createdUser = CreateUserRow(user)
	}

	// Test CreateTool - Standalone Tool
	standaloneSchema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: openapi3.Schemas{
			"input": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"string"},
					Description: "Input parameter",
				},
			},
		},
		Required: []string{"input"},
	}

	standaloneConfig := ToolConfig{
		Type: ToolTypeStandalone,
		C: &ToolConfigStandalone{
			ApiKey: stringPtr("test-api-key-123"),
			Url:    "https://api.example.com/tool",
			Params: standaloneSchema,
		},
	}

	createStandaloneParams := CreateToolParams{
		Name:        uniqueName("Test Standalone Tool"),
		Description: pgtype.Text{String: "A test standalone tool", Valid: true},
		Config:      standaloneConfig,
		CreatedBy:   createdUser.ID,
	}

	createdStandaloneTool, err := queries.CreateTool(t.Context(), createStandaloneParams)
	require.NoError(t, err, "Failed to create standalone tool")
	assert.NotEqual(t, uuid.Nil, createdStandaloneTool.ID, "Created tool should have a valid ID")
	assert.Equal(t, createStandaloneParams.Name, createdStandaloneTool.Name)
	assert.Equal(t, createStandaloneParams.Description, createdStandaloneTool.Description)
	assert.Equal(t, createStandaloneParams.Config.Type, createdStandaloneTool.Config.Type)
	assert.Equal(t, createStandaloneParams.CreatedBy, createdStandaloneTool.CreatedBy)

	// Test CreateTool - Workflow Tool
	workflowSchema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: openapi3.Schemas{
			"workflow_input": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"string"},
					Description: "Workflow input",
				},
			},
		},
		Required: []string{"workflow_input"},
	}

	workflowConfig := ToolConfig{
		Type: ToolTypeWorkflow,
		C: &ToolConfigWorkflow{
			S3Url:  "s3://test-bucket/workflow.py",
			Params: workflowSchema,
		},
	}

	createWorkflowParams := CreateToolParams{
		Name:        uniqueName("Test Workflow Tool"),
		Description: pgtype.Text{String: "A test workflow tool", Valid: true},
		Config:      workflowConfig,
		CreatedBy:   createdUser.ID,
	}

	createdWorkflowTool, err := queries.CreateTool(t.Context(), createWorkflowParams)
	require.NoError(t, err, "Failed to create workflow tool")
	assert.NotEqual(t, uuid.Nil, createdWorkflowTool.ID, "Created workflow tool should have a valid ID")
	assert.Equal(t, createWorkflowParams.Name, createdWorkflowTool.Name)
	assert.Equal(t, createWorkflowParams.Config.Type, createdWorkflowTool.Config.Type)

	// Test CreateTool - MCP Tool
	envVars := map[string]string{
		"DEBUG":       "true",
		"CONFIG_PATH": "/etc/mcp/config.yaml",
	}
	mcpConfig := ToolConfig{
		Type: ToolTypeMCP,
		C: &ToolConfigMCP{
			ApiKey:     stringPtr("mcp-api-key-456"),
			Entrypoint: "/usr/bin/python",
			Protocol:   MCPProtocolStdio,
			EnvVars:    &envVars,
		},
	}

	createMCPParams := CreateToolParams{
		Name:        uniqueName("Test MCP Tool"),
		Description: pgtype.Text{String: "A test MCP tool", Valid: true},
		Config:      mcpConfig,
		CreatedBy:   createdUser.ID,
	}

	createdMCPTool, err := queries.CreateTool(t.Context(), createMCPParams)
	require.NoError(t, err, "Failed to create MCP tool")
	assert.NotEqual(t, uuid.Nil, createdMCPTool.ID, "Created MCP tool should have a valid ID")
	assert.Equal(t, createMCPParams.Name, createdMCPTool.Name)
	assert.Equal(t, createMCPParams.Config.Type, createdMCPTool.Config.Type)

	// Test ListTools
	tools, err := queries.ListTools(t.Context())
	require.NoError(t, err, "Failed to list tools")
	assert.NotEmpty(t, tools, "Tools list should not be empty")

	// Find our created tools in the list
	var foundStandalone, foundWorkflow, foundMCP *Tool
	for i := range tools {
		switch tools[i].ID {
		case createdStandaloneTool.ID:
			foundStandalone = &tools[i]
		case createdWorkflowTool.ID:
			foundWorkflow = &tools[i]
		case createdMCPTool.ID:
			foundMCP = &tools[i]
		}
	}

	assert.NotNil(t, foundStandalone, "Standalone tool should be found in list")
	assert.NotNil(t, foundWorkflow, "Workflow tool should be found in list")
	assert.NotNil(t, foundMCP, "MCP tool should be found in list")

	if foundStandalone != nil {
		assert.Equal(t, createdStandaloneTool.Name, foundStandalone.Name)
		// Type is now part of config, not directly accessible in ListToolsRow
	}

	// Test GetToolById
	retrievedStandaloneTool, err := queries.GetToolById(t.Context(), createdStandaloneTool.ID)
	require.NoError(t, err, "Failed to get standalone tool by ID")
	assert.Equal(t, createdStandaloneTool.ID, retrievedStandaloneTool.ID)
	assert.Equal(t, createdStandaloneTool.Name, retrievedStandaloneTool.Name)
	assert.Equal(t, createdStandaloneTool.Config.Type, retrievedStandaloneTool.Config.Type)
	assert.Equal(t, createdStandaloneTool.Config.Type, retrievedStandaloneTool.Config.Type)

	retrievedWorkflowTool, err := queries.GetToolById(t.Context(), createdWorkflowTool.ID)
	require.NoError(t, err, "Failed to get workflow tool by ID")
	assert.Equal(t, createdWorkflowTool.ID, retrievedWorkflowTool.ID)
	assert.Equal(t, createdWorkflowTool.Name, retrievedWorkflowTool.Name)

	retrievedMCPTool, err := queries.GetToolById(t.Context(), createdMCPTool.ID)
	require.NoError(t, err, "Failed to get MCP tool by ID")
	assert.Equal(t, createdMCPTool.ID, retrievedMCPTool.ID)
	assert.Equal(t, createdMCPTool.Name, retrievedMCPTool.Name)

	// Test GetToolInfoByName
	retrievedByName, err := queries.GetToolInfoByName(t.Context(), createStandaloneParams.Name)
	require.NoError(t, err, "Failed to get tool by name")
	assert.Equal(t, createdStandaloneTool.ID, retrievedByName.ID)
	assert.Equal(t, createdStandaloneTool.Name, retrievedByName.Name)

	// Test UpdateTool
	updatedSchema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: openapi3.Schemas{
			"input": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"string"},
					Description: "Updated input parameter",
				},
			},
			"config": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"object"},
					Description: "Configuration object",
				},
			},
		},
		Required: []string{"input"},
	}

	updatedConfig := ToolConfig{
		Type: ToolTypeStandalone,
		C: &ToolConfigStandalone{
			ApiKey: stringPtr("updated-api-key-789"),
			Url:    "https://api-updated.example.com/tool",
			Params: updatedSchema,
		},
	}

	updateParams := UpdateToolParams{
		ID:          createdStandaloneTool.ID,
		Description: pgtype.Text{String: "Updated description for standalone tool", Valid: true},
		Config:      updatedConfig,
	}

	updatedTool, err := queries.UpdateTool(t.Context(), updateParams)
	require.NoError(t, err, "Failed to update tool")
	assert.Equal(t, createdStandaloneTool.ID, updatedTool.ID)
	assert.Equal(t, updateParams.Description, updatedTool.Description)
	assert.Equal(t, updatedConfig.Type, updatedTool.Config.Type)

	// Verify the update persisted
	retrievedUpdatedTool, err := queries.GetToolById(t.Context(), createdStandaloneTool.ID)
	require.NoError(t, err, "Failed to get updated tool")
	assert.Equal(t, updateParams.Description, retrievedUpdatedTool.Description)

	// Test DeleteTool
	err = queries.DeleteTool(t.Context(), createdStandaloneTool.ID)
	require.NoError(t, err, "Failed to delete standalone tool")

	err = queries.DeleteTool(t.Context(), createdWorkflowTool.ID)
	require.NoError(t, err, "Failed to delete workflow tool")

	err = queries.DeleteTool(t.Context(), createdMCPTool.ID)
	require.NoError(t, err, "Failed to delete MCP tool")

	// Verify tools are deleted
	_, err = queries.GetToolById(t.Context(), createdStandaloneTool.ID)
	assert.Error(t, err, "Getting deleted tool should return error")
	assert.Equal(t, pgx.ErrNoRows, err, "Should return no rows error")

	_, err = queries.GetToolById(t.Context(), createdWorkflowTool.ID)
	assert.Error(t, err, "Getting deleted workflow tool should return error")

	_, err = queries.GetToolById(t.Context(), createdMCPTool.ID)
	assert.Error(t, err, "Getting deleted MCP tool should return error")
}

func TestToolsErrorCases(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)

	// Test GetToolById with non-existent ID
	nonExistentID := uuid.New()
	_, err := queries.GetToolById(t.Context(), nonExistentID)
	assert.Error(t, err, "Getting non-existent tool should return error")
	assert.Equal(t, pgx.ErrNoRows, err, "Should return no rows error")

	// Test GetToolInfoByName with non-existent name
	_, err = queries.GetToolInfoByName(t.Context(), "NonExistentToolName")
	assert.Error(t, err, "Getting tool by non-existent name should return error")
	assert.Equal(t, pgx.ErrNoRows, err, "Should return no rows error")

	// Test UpdateTool with non-existent ID
	updateParams := UpdateToolParams{
		ID:          nonExistentID,
		Description: pgtype.Text{String: "Updated description", Valid: true},
		Config: ToolConfig{
			Type: ToolTypeStandalone,
			C:    &ToolConfigStandalone{Url: "https://example.com"},
		},
	}
	_, err = queries.UpdateTool(t.Context(), updateParams)
	assert.Error(t, err, "Updating non-existent tool should return error")
	assert.Equal(t, pgx.ErrNoRows, err, "Should return no rows error")

	// Test DeleteTool with non-existent ID (should not return error)
	err = queries.DeleteTool(t.Context(), nonExistentID)
	assert.NoError(t, err, "Deleting non-existent tool should not return error")
}

func TestCreateToolValidation(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)

	// First create a user
	createUserParams := CreateUserParams{
		Name:           "testuser_tools_validation_unique",
		Email:          "tools_validation_unique@example.com",
		AdditionalInfo: JsonRaw{},
		PasswordHash:   "hashedpassword123",
		ProviderName:   ProviderNameLocal,
	}
	createdUser, err := queries.CreateUser(t.Context(), createUserParams)
	if err != nil {
		user, err := queries.GetUserByEmail(t.Context(), createUserParams.Email)
		require.NoError(t, err)
		// Convert GetUserByEmailRow to CreateUserRow
		createdUser = CreateUserRow(user)
	}

	// Test creating tool with duplicate name should fail
	emptySchema := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: openapi3.Schemas{},
	}

	config := ToolConfig{
		Type: ToolTypeStandalone,
		C: &ToolConfigStandalone{
			Url:    "https://example.com",
			Params: emptySchema,
		},
	}

	duplicateTestName := uniqueName("Duplicate Tool Name Test")
	createParams := CreateToolParams{
		Name:        duplicateTestName,
		Description: pgtype.Text{String: "First tool", Valid: true},
		Config:      config,
		CreatedBy:   createdUser.ID,
	}

	// Create first tool
	firstTool, err := queries.CreateTool(t.Context(), createParams)
	require.NoError(t, err, "First tool creation should succeed")

	// Try to create second tool with same name
	createParams.Name = duplicateTestName // Use the exact same name again
	createParams.Description = pgtype.Text{String: "Second tool with same name", Valid: true}
	_, err = queries.CreateTool(t.Context(), createParams)
	assert.Error(t, err, "Creating tool with duplicate name should fail")

	// Clean up
	err = queries.DeleteTool(t.Context(), firstTool.ID)
	require.NoError(t, err)
}

func TestToolConfigSerialization(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)

	// Create user
	createUserParams := CreateUserParams{
		Name:           "testuser_tools_serialization_unique",
		Email:          "tools_serialization_unique@example.com",
		AdditionalInfo: JsonRaw{},
		PasswordHash:   "hashedpassword123",
		ProviderName:   ProviderNameLocal,
	}
	createdUser, err := queries.CreateUser(t.Context(), createUserParams)
	if err != nil {
		user, err := queries.GetUserByEmail(t.Context(), createUserParams.Email)
		require.NoError(t, err)
		// Convert GetUserByEmailRow to CreateUserRow
		createdUser = CreateUserRow(user)
	}

	// Test complex parameter serialization
	complexSchema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: openapi3.Schemas{
			"nested_object": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"object"},
					Properties: openapi3.Schemas{
						"deep_property": &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type:        &openapi3.Types{"string"},
								Description: "A deeply nested property",
							},
						},
					},
				},
			},
			"array_property": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"array"},
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
				},
			},
		},
		Required: []string{"nested_object"},
	}

	config := ToolConfig{
		Type: ToolTypeStandalone,
		C: &ToolConfigStandalone{
			ApiKey: stringPtr("complex-serialization-key"),
			Url:    "https://api.complex-serialization.com",
			Params: complexSchema,
		},
	}

	createParams := CreateToolParams{
		Name:        uniqueName("Complex Serialization Test Tool"),
		Description: pgtype.Text{String: "Testing complex parameter serialization", Valid: true},
		Config:      config,
		CreatedBy:   createdUser.ID,
	}

	createdTool, err := queries.CreateTool(t.Context(), createParams)
	require.NoError(t, err, "Failed to create tool with complex params")

	// Retrieve and verify serialization
	retrievedTool, err := queries.GetToolById(t.Context(), createdTool.ID)
	require.NoError(t, err, "Failed to retrieve tool")

	assert.Equal(t, config.Type, retrievedTool.Config.Type)

	standaloneConfig := retrievedTool.Config.GetStandalone()
	require.NotNil(t, standaloneConfig, "Should be able to get standalone config")

	assert.Equal(t, "https://api.complex-serialization.com", standaloneConfig.Url)
	assert.Equal(t, "complex-serialization-key", *standaloneConfig.ApiKey)

	// Verify nested structure is preserved
	assert.NotNil(t, standaloneConfig.Params.Type, "Param.Type should not be nil")
	assert.Equal(t, &openapi3.Types{"object"}, standaloneConfig.Params.Type, "Type should be object")
	assert.NotNil(t, standaloneConfig.Params.Properties, "Properties should not be nil")

	nestedObjectRef, exists := standaloneConfig.Params.Properties["nested_object"]
	assert.True(t, exists, "Nested object property should exist")
	assert.NotNil(t, nestedObjectRef.Value, "Nested object value should not be nil")
	assert.NotNil(t, nestedObjectRef.Value.Type, "Nested object type should not be nil")
	assert.Equal(t, &openapi3.Types{"object"}, nestedObjectRef.Value.Type, "Nested object type should be object")

	// Clean up
	err = queries.DeleteTool(t.Context(), createdTool.ID)
	require.NoError(t, err)
}

func TestMCPToolWithEnvironmentVariables(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)

	// Create user
	createUserParams := CreateUserParams{
		Name:           "testuser_tools_mcp_env_unique",
		Email:          "tools_mcp_env_unique@example.com",
		AdditionalInfo: JsonRaw{},
		PasswordHash:   "hashedpassword123",
		ProviderName:   ProviderNameLocal,
	}
	createdUser, err := queries.CreateUser(t.Context(), createUserParams)
	if err != nil {
		user, err := queries.GetUserByEmail(t.Context(), createUserParams.Email)
		require.NoError(t, err)
		// Convert GetUserByEmailRow to CreateUserRow
		createdUser = CreateUserRow(user)
	}

	// Test MCP tool with environment variables
	envVars := map[string]string{
		"PATH":            "/usr/local/bin:/usr/bin:/bin",
		"PYTHONPATH":      "/opt/mcp/lib",
		"MCP_CONFIG":      "/etc/mcp.yaml",
		"DEBUG":           "true",
		"LOG_LEVEL":       "info",
		"SPECIAL_CHARS":   "!@#$%^&*()_+-={}[]|\\:;\"'<>?,./",
		"UNICODE":         "こんにちは世界",
		"MULTILINE_VALUE": "line1\nline2\nline3",
	}

	config := ToolConfig{
		Type: ToolTypeMCP,
		C: &ToolConfigMCP{
			ApiKey:     stringPtr("mcp-env-test-key"),
			Entrypoint: "/usr/bin/python3",
			Protocol:   MCPProtocolGRPC,
			EnvVars:    &envVars,
		},
	}

	createParams := CreateToolParams{
		Name:        uniqueName("MCP Environment Variables Test Tool"),
		Description: pgtype.Text{String: "Testing MCP tool with complex environment variables", Valid: true},
		Config:      config,
		CreatedBy:   createdUser.ID,
	}

	createdTool, err := queries.CreateTool(t.Context(), createParams)
	require.NoError(t, err, "Failed to create MCP tool with env vars")

	// Retrieve and verify environment variables
	retrievedTool, err := queries.GetToolById(t.Context(), createdTool.ID)
	require.NoError(t, err, "Failed to retrieve MCP tool")

	mcpConfig := retrievedTool.Config.GetMCP()
	require.NotNil(t, mcpConfig, "Should be able to get MCP config")

	assert.Equal(t, "/usr/bin/python3", mcpConfig.Entrypoint)
	assert.Equal(t, MCPProtocolGRPC, mcpConfig.Protocol)
	assert.Equal(t, "mcp-env-test-key", *mcpConfig.ApiKey)

	// Verify all environment variables are preserved
	require.NotNil(t, mcpConfig.EnvVars, "EnvVars should not be nil")
	assert.Equal(t, len(envVars), len(*mcpConfig.EnvVars), "All env vars should be preserved")
	for key, expectedValue := range envVars {
		actualValue, exists := (*mcpConfig.EnvVars)[key]
		assert.True(t, exists, "Environment variable %s should exist", key)
		assert.Equal(t, expectedValue, actualValue, "Environment variable %s should have correct value", key)
	}

	// Clean up
	err = queries.DeleteTool(t.Context(), createdTool.ID)
	require.NoError(t, err)
}

func TestListToolsOrderingAndPagination(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)

	// Create user
	createUserParams := CreateUserParams{
		Name:           "testuser_tools_ordering_unique",
		Email:          "tools_ordering_unique@example.com",
		AdditionalInfo: JsonRaw{},
		PasswordHash:   "hashedpassword123",
		ProviderName:   ProviderNameLocal,
	}
	createdUser, err := queries.CreateUser(t.Context(), createUserParams)
	if err != nil {
		user, err := queries.GetUserByEmail(t.Context(), createUserParams.Email)
		require.NoError(t, err)
		// Convert GetUserByEmailRow to CreateUserRow
		createdUser = CreateUserRow(user)
	}

	// Create multiple tools with slight delays to test ordering
	var createdTools []Tool
	for i := 0; i < 3; i++ {
		emptySchema := &openapi3.Schema{
			Type:       &openapi3.Types{"object"},
			Properties: openapi3.Schemas{},
		}

		config := ToolConfig{
			Type: ToolTypeStandalone,
			C: &ToolConfigStandalone{
				Url:    "https://example.com",
				Params: emptySchema,
			},
		}

		createParams := CreateToolParams{
			Name:        uniqueName(fmt.Sprintf("Ordering Test Tool %c", rune('A'+i))),
			Description: pgtype.Text{String: "Tool for ordering test", Valid: true},
			Config:      config,
			CreatedBy:   createdUser.ID,
		}

		tool, err := queries.CreateTool(t.Context(), createParams)
		require.NoError(t, err, "Failed to create tool %d", i)
		createdTools = append(createdTools, tool)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// List tools and verify ordering (should be DESC by created_at)
	tools, err := queries.ListTools(t.Context())
	require.NoError(t, err, "Failed to list tools")

	// Find our tools in the list
	var ourTools []Tool
	for _, tool := range tools {
		for _, created := range createdTools {
			if tool.ID == created.ID {
				ourTools = append(ourTools, tool)
				break
			}
		}
	}

	assert.Equal(t, 3, len(ourTools), "Should find all our created tools")

	// Verify DESC ordering by checking that later created tools come first
	if len(ourTools) >= 2 {
		// The tools should be ordered with the most recently created first
		assert.True(t, ourTools[0].CreatedAt.Time.After(ourTools[1].CreatedAt.Time),
			"Tools should be ordered by created_at DESC")
	}

	// Clean up
	for _, tool := range createdTools {
		err = queries.DeleteTool(t.Context(), tool.ID)
		require.NoError(t, err, "Failed to delete tool %s", tool.ID)
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function to generate unique names for testing
func uniqueName(base string) string {
	return fmt.Sprintf("%s_%d_%s", base, time.Now().UnixNano(), uuid.New().String()[:8])
}
