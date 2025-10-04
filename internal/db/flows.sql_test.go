package db

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestCRUDFlows(t *testing.T) {
	t.Parallel()
	db_pool := setupTestDB(t)
	defer db_pool.Close()
	queries := New(db_pool)
	new_uuid, err := uuid.NewUUID()
	if err != nil {
		t.Fatalf("Failed to generate UUID: %v", err)
	}
	// Create a new workflow
	parameterSchemaJsonRaw, _ := NewJsonRaw(map[string]any{"param1": "value1", "param2": "value2"})

	createParams := CreateFlowParams{
		ID:               new_uuid,
		Name:             "Test Workflow",
		Description:      pgtype.Text{String: "This is a test workflow", Valid: true},
		ParametersSchema: parameterSchemaJsonRaw,
		Engine:           "test_engine",
		AdditionalInfo:   nil,
		Tags:             []string{"tag1", "tag2"},
	}

	createdWorkflow, err := queries.CreateFlow(t.Context(), createParams)
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}
	assert.Equal(t, createParams.ID, createdWorkflow.ID, "Created workflow ID should match the input ID")
	assert.Equal(t, createParams.Name, createdWorkflow.Name, "Created workflow name should match the input name")

	// Test GetWorkflows
	workflows, err := queries.GetFlows(t.Context(), GetFlowsParams{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Failed to get workflows: %v", err)
	}
	assert.NotEmpty(t, workflows, "Workflows should not be empty")
	assert.Greater(t, len(workflows), 0, "There should be at least one workflow in the database")

	// Test GetWorkflowByID
	workflow, err := queries.GetFlowById(t.Context(), new_uuid)
	if err != nil {
		t.Fatalf("Failed to get workflow by ID: %v", err)
	}
	assert.NotNil(t, workflow, "Workflow should not be nil")
	assert.Equal(t, new_uuid, workflow.ID, "Retrieved workflow ID should match the created workflow ID")

	// Additional tests for CRUD operations can be added here
	// Test UpdateWorkflow
	parameterSchemaJsonRaw, _ = NewJsonRaw(map[string]any{"param1": "new_value1", "param2": "new_value2"})

	updateParams := UpdateFlowParams{
		ID:               new_uuid,
		Name:             "Updated Test Workflow",
		Description:      pgtype.Text{String: "This is an updated test workflow", Valid: true},
		ParametersSchema: parameterSchemaJsonRaw,
		Engine:           "updated_test_engine",
		AdditionalInfo:   nil,
		Tags:             []string{"updated_tag1", "updated_tag2"},
	}
	updatedWorkflow, err := queries.UpdateFlow(t.Context(), updateParams)
	if err != nil {
		t.Fatalf("Failed to update workflow: %v", err)
	}

	var updatedWorkflowParameterSchemaMap map[string]any
	_ = json.Unmarshal(updatedWorkflow.ParametersSchema, &updatedWorkflowParameterSchemaMap)
	var updateParamsParameterSchemaMap map[string]any
	_ = json.Unmarshal(updateParams.ParametersSchema, &updateParamsParameterSchemaMap)

	assert.Equal(t, updateParams.Name, updatedWorkflow.Name, "Updated workflow name should match the input name")
	assert.Equal(t, updateParamsParameterSchemaMap["param1"], updatedWorkflowParameterSchemaMap["param1"], "Updated workflow parameter schema should match the input schema")

	// Test DeleteWorkflow
	err = queries.DeleteFlow(t.Context(), new_uuid)
	if err != nil {
		t.Fatalf("Failed to delete workflow: %v", err)
	}
	// Verify deletion
	_, err = queries.GetFlowById(t.Context(), new_uuid)
	if err == nil {
		t.Fatalf("Expected error when getting deleted workflow, but got none")
	}
	// Catch no rows in result set error, no other errors
	assert.EqualError(t, err, pgx.ErrNoRows.Error(), "Expected no rows error when getting deleted workflow")
	assert.Equal(t, err, pgx.ErrNoRows, "Expected no rows error when getting deleted workflow")
}
