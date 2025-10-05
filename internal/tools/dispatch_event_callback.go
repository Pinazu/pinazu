package tools

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nats-io/nats.go"
	"github.com/pinazu/internal/agents"
	"github.com/pinazu/internal/db"
	"github.com/pinazu/internal/service"
)

// dispatchEventCallback handles the tool dispatch tool use event callback
func (ts *ToolService) dispatchEventCallback(msg *nats.Msg) {
	// Check if context was cancelled
	select {
	case <-ts.ctx.Done():
		ts.log.Info("Context cancelled, stopping message processing")
		return
	default:
	}

	// Parse NATS message to request struct
	req, err := service.ParseEvent[*service.ToolDispatchEventMessage](msg.Data)
	if err != nil {
		ts.log.Error("Failed to unmarshal message to request", "error", err)
		return
	}

	// Log the received message
	ts.log.Info("Received and validated tool dispatch message",
		"agent_id", req.Msg.AgentId,
		"thread_id", req.H.ThreadID,
		"connection_id", req.H.ConnectionID,
		"user_id", req.H.UserID,
	)
	ts.log.Debug("Tool use message", "message", string(req.Msg.Message))

	// Get the database queries
	queries := db.New(ts.s.GetDB())

	// Add tool request message to the database
	_, err = queries.CreateAgentMessage(ts.ctx, db.CreateAgentMessageParams{
		ThreadID:    *req.H.ThreadID,
		Message:     req.Msg.Message,
		SenderID:    req.Msg.AgentId,
		RecipientID: req.Msg.RecipientId,
		StopReason:  pgtype.Text{String: "tool_use", Valid: true},
	})
	if err != nil {
		ts.log.Error("Failed to add tool use message to the database", "error", err)
		return
	}

	switch req.Msg.Provider {
	case db.ProviderModelAnthropic:
		ts.handleAnthropicToolUse(req, queries)
	case db.ProviderModelBedrockAnthropic:
		ts.handleAnthropicToolUse(req, queries)
	case db.ProviderModelBedrock:
		ts.log.Error("Bedrock model provider not yet supported")
	case db.ProviderModelGoogle:
		ts.log.Error("Google model provider not yet supported")
	case db.ProviderModelOpenAI:
		ts.log.Error("OpenAI model provider not yet supported")
	default:
		ts.log.Error("Unsupported model provider", "model_provider", req.Msg.Provider)
		return
	}
}

func (ts *ToolService) handleAnthropicToolUse(req *service.Event[*service.ToolDispatchEventMessage], queries *db.Queries) {
	// State initialization
	var standaloneToolsToExecute []service.StandaloneToolRequestEventMessage
	var workflowToolsToExecute []service.FlowRunExecuteRequestEventMessage
	var mcpToolsToExecute []service.StandaloneToolRequestEventMessage

	msg, err := agents.ParseMessage[anthropic.MessageParam](req.Msg.Message)
	if err != nil {
		ts.log.Debug("Anthropic message content", "content", req.Msg.Message)
		ts.log.Error("Failed to parse Anthropic messages", "error", err)
		return
	}

	// Check if the message is a tool use message
	if msg.Role != anthropic.MessageParamRoleAssistant {
		ts.log.Error("Message is not a tool use message", "role", msg.Role)
		return
	}
	if msg.Content == nil {
		ts.log.Error("Message content is nil")
		return
	}

	// Count tool use blocks first to determine processing strategy
	// Create a list of tool use block and iterate from it
	toolUseBlocks := make([]*anthropic.ToolUseBlockParam, 0)
	for _, blk := range msg.Content {
		if blk.OfToolUse != nil {
			toolUseBlocks = append(toolUseBlocks, blk.OfToolUse)
		}
	}
	if len(toolUseBlocks) == 0 {
		ts.log.Error("No tool use blocks found in message")
		return
	}
	ts.log.Debug("Counted tool use blocks in MessageParamRoleAssistant", "count", len(toolUseBlocks))

	// Handle nil ConnectionID (e.g., from HTTP requests without WebSocket)
	var connectionID uuid.UUID
	if req.H.ConnectionID != nil {
		connectionID = *req.H.ConnectionID
	} else {
		connectionID = uuid.Nil // Use zero UUID for HTTP requests
	}

	// Handle nil ThreadID (this should not normally be nil)
	var threadID uuid.UUID
	if req.H.ThreadID != nil {
		threadID = *req.H.ThreadID
	} else {
		ts.log.Error("ThreadID is nil in tool dispatch event")
		return
	}

	// Create a temp parent tool when process parallel, multiple tool use blocks
	var tempParallelToolManagement db.ToolRun
	if len(toolUseBlocks) > 1 {
		tempInput, _ := db.NewJsonRaw(map[string]any{})
		tempParallelToolManagement, err = queries.CreateToolRunStatus(ts.ctx, db.CreateToolRunStatusParams{
			ID:           uuid.NewString(),
			Name:         "temp_parallel_tool_management",
			Input:        tempInput,
			ConnectionID: connectionID,
			ThreadID:     threadID,
			AgentID:      req.Msg.AgentId,
			RecipientID:  req.Msg.RecipientId,
		})
		if err != nil {
			ts.log.Error("Failed to create temp parent tool status", "error", err)
			return
		}
	}

	// Iterate through toolUseBlock list
	for _, toolBlock := range toolUseBlocks {
		// blk is already a *ToolUseBlockParam, so we can access Input directly
		toolBlockInputMap, ok := toolBlock.Input.(map[string]any)
		if !ok {
			ts.log.Error("Tool input is not a map[string]any", "input_type", fmt.Sprintf("%T", toolBlock.Input))
			continue
		}

		// Get tool_type
		tool, err := queries.GetToolInfoByName(ts.ctx, toolBlock.Name)
		if err != nil {
			ts.log.Warn("Failed to get tool", "name", toolBlock.Name, "error", err)
			continue // Skip if errord
		}

		// Add tool state to the database
		toolBlockInputJson, err := db.NewJsonRaw(toolBlockInputMap)
		if err != nil {
			ts.log.Error("Failed to convert tool input to JsonRaw", "error", err)
			continue
		}

		if len(toolUseBlocks) == 1 {
			_, err = queries.CreateToolRunStatus(ts.ctx, db.CreateToolRunStatusParams{
				ID:           toolBlock.ID,
				Name:         toolBlock.Name,
				Input:        toolBlockInputJson,
				ConnectionID: connectionID,
				ThreadID:     threadID,
				AgentID:      req.Msg.AgentId,
				RecipientID:  req.Msg.RecipientId,
			})
			if err != nil {
				ts.log.Error("Failed to add tool run status to the database", "error", err)
				continue
			}
		} else if len(toolUseBlocks) > 1 && tempParallelToolManagement.ID != "" {
			_, err = queries.CreateChildToolRunStatus(ts.ctx, db.CreateChildToolRunStatusParams{
				ID:           toolBlock.ID,
				Name:         toolBlock.Name,
				Input:        toolBlockInputJson,
				ConnectionID: connectionID,
				ThreadID:     threadID,
				AgentID:      req.Msg.AgentId,
				RecipientID:  req.Msg.RecipientId,
				ParentRunID:  pgtype.Text{String: tempParallelToolManagement.ID, Valid: true},
			})
			if err != nil {
				ts.log.Error("Failed to add child tool run status to the database (non-batch)", "error", err)
				continue
			}
		} else {
			ts.log.Error("Failed to add tool run status to the database. Unexpected error")
			continue
		}

		// Process tool recursively and collect all tools to execute
		processResult := ts.processToolRecursively(toolBlock.ID, toolBlockInputMap, tool, req, queries)
		standaloneToolsToExecute = append(standaloneToolsToExecute, processResult.StandaloneTools...)
		workflowToolsToExecute = append(workflowToolsToExecute, processResult.WorkflowTools...)
		mcpToolsToExecute = append(mcpToolsToExecute, processResult.MCPTools...)
	}

	// Execute tools by type using goroutines
	var wg sync.WaitGroup
	if len(standaloneToolsToExecute) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ts.executeStandaloneTool(standaloneToolsToExecute, req.H, req.M)
		}()
	}
	if len(workflowToolsToExecute) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ts.executeWorkflowTool(workflowToolsToExecute)
		}()
	}
	if len(mcpToolsToExecute) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ts.executeMCPTool(mcpToolsToExecute)
		}()
	}
	if len(standaloneToolsToExecute) == 0 && len(workflowToolsToExecute) == 0 && len(mcpToolsToExecute) == 0 {
		ts.log.Warn("No tools to execute after processing tool use message")
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

// processToolRecursively handles tool processing with recursive batch tool support
func (ts *ToolService) processToolRecursively(toolRunID string, toolInput map[string]any, tool db.Tool, req *service.Event[*service.ToolDispatchEventMessage], queries *db.Queries) ToolProcessResult {
	result := ToolProcessResult{
		StandaloneTools: []service.StandaloneToolRequestEventMessage{},
		WorkflowTools:   []service.FlowRunExecuteRequestEventMessage{},
		MCPTools:        []service.StandaloneToolRequestEventMessage{},
	}

	// Handle special tool name cases
	switch tool.Name {
	case "batch_tool":
		ts.log.Info("Tool batch_tool is detected, processing child tools recursively")

		invocations, ok := toolInput["invocations"].([]any)
		if !ok {
			ts.log.Error("Invalid format: 'invocations' is not a list")
			return result
		}

		for _, rawChild := range invocations {
			child, ok := rawChild.(map[string]any)
			if !ok {
				ts.log.Error("Invalid format for child invocation")
				continue
			}

			// Parse JSON arguments if it's a string
			switch argVal := child["arguments"].(type) {
			case string:
				var parsed map[string]any
				if err := json.Unmarshal([]byte(argVal), &parsed); err != nil {
					ts.log.Error("Invalid JSON in child arguments", "error", err)
					continue
				}
				child["arguments"] = parsed
			case map[string]any:
				// OK
			default:
				ts.log.Error("Unsupported format for child arguments")
				continue
			}

			childToolInput := child["arguments"].(map[string]any)

			// Validate tool
			childTool, err := queries.GetToolInfoByName(ts.ctx, child["name"].(string))
			if err != nil {
				ts.log.Warn("Failed to get child tool", "name", childTool.Name, "error", err)
				continue // Skip if error
			}

			ts.log.Info("Creating state for child tool request", "name", childTool.Name, "type", childTool.Config.Type)

			// Add child tool state to the database
			inputJsonRaw, err := db.NewJsonRaw(childToolInput)
			if err != nil {
				ts.log.Error("Failed to map child arguments to JsonRaw data", "error", err)
				continue
			}

			// Handle nil ConnectionID (e.g., from HTTP requests without WebSocket)
			var connectionID uuid.UUID
			if req.H.ConnectionID != nil {
				connectionID = *req.H.ConnectionID
			} else {
				connectionID = uuid.Nil // Use zero UUID for HTTP requests
			}

			// Handle nil ThreadID (this should not normally be nil)
			var threadID uuid.UUID
			if req.H.ThreadID != nil {
				threadID = *req.H.ThreadID
			} else {
				ts.log.Error("ThreadID is nil in child tool dispatch event", "tool_name", childTool.Name)
				continue // Skip this child tool if no thread context
			}

			// Generate a new UUID for the child tool run
			childToolRunID := uuid.NewString()
			ts.log.Debug("Creating child tool run", "child_id", childToolRunID, "parent_id", toolRunID, "tool_name", childTool.Name)

			// Create a new child tool run status record with new run status UUID
			childToolRunStatus, err := queries.CreateChildToolRunStatus(ts.ctx, db.CreateChildToolRunStatusParams{
				ID:           childToolRunID,
				ConnectionID: connectionID,
				ThreadID:     threadID,
				AgentID:      req.Msg.AgentId,
				RecipientID:  req.Msg.RecipientId,
				Name:         childTool.Name,
				Input:        inputJsonRaw,
				ParentRunID:  pgtype.Text{String: toolRunID, Valid: true},
			})
			if err != nil {
				ts.log.Error("Failed to add child tool run status to the database (batch)", "error", err)
				continue
			}

			// Recursively process the child tool (this handles nested batch_tool cases)
			childResult := ts.processToolRecursively(childToolRunStatus.ID, childToolInput, childTool, req, queries)
			result.StandaloneTools = append(result.StandaloneTools, childResult.StandaloneTools...)
			result.WorkflowTools = append(result.WorkflowTools, childResult.WorkflowTools...)
			result.MCPTools = append(result.MCPTools, childResult.MCPTools...)
		}
	case "invoke_agent":
		ts.log.Info("Tool invoke_tool_agent detected, transfer message to the agent")

		// Create a new user message for the agent
		message := map[string]any{
			"role": "user",
			"content": []map[string]any{
				{"type": "text", "text": toolInput["query"].(string)},
			},
		}

		// Marshal the event message with new header and publish to task execute event
		messageJsonRaw, err := db.NewJsonRaw(message)
		if err != nil {
			ts.log.Error("Failed to marshal new created user message for invoke_agent", "error", err)
		}
		// Parse agent_id from string to UUID
		agentHandoffToID, err := uuid.Parse(toolInput["agent_id"].(string))
		if err != nil {
			ts.log.Error("Failed to parse agent_id as UUID", "agent_id", toolInput["agent_id"], "error", err)
			return result
		}

		event := service.NewEvent(&service.TaskHandoffEventMessage{
			ToolRunId:        toolRunID,
			AgentID:          req.Msg.AgentId,
			AgentHandoffToID: agentHandoffToID,
			Messages:         []db.JsonRaw{messageJsonRaw},
		}, req.H, req.M)
		err = event.Publish(ts.s.GetNATS())
		if err != nil {
			ts.log.Error("Failed to publish task handoff event for invoke_agent", "error", err)
		}

		// Don't add invoke_tool_agent to any execution list as they are handled separately
	default:
		// Regular tool - categorize by tool tydepe
		switch tool.Config.Type {
		case db.ToolTypeStandalone:
			standaloneConfig := tool.Config.GetStandalone()
			if standaloneConfig == nil {
				ts.log.Error("Failed to get standalone config for tool", "tool_name", tool.Name)
				break
			}
			result.StandaloneTools = append(result.StandaloneTools, service.StandaloneToolRequestEventMessage{
				ToolRunId:  toolRunID,
				ToolName:   tool.Name,
				ToolInput:  toolInput,
				ToolURL:    standaloneConfig.Url,
				ToolAPIKey: standaloneConfig.ApiKey,
			})
		case db.ToolTypeWorkflow:
			flowRunID := uuid.NewSHA1(uuid.Nil, []byte(toolRunID))
			result.WorkflowTools = append(result.WorkflowTools, service.FlowRunExecuteRequestEventMessage{
				FlowId:     tool.ID,
				FlowRunId:  &flowRunID,
				Parameters: toolInput,
				Engine:     "process", // Default engine
			})
		case db.ToolTypeMCP:
			result.MCPTools = append(result.MCPTools, service.StandaloneToolRequestEventMessage{
				ToolRunId: toolRunID,
				ToolName:  tool.Name,
				ToolInput: toolInput,
			})
		default:
			ts.log.Warn("Unknown tool type, defaulting to standalone",
				"tool_name", tool.Name,
				"tool_type", tool.Config.Type,
			)
		}
	}

	return result
}
