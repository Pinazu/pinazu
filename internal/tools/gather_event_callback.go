package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nats-io/nats.go"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/db"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
	"gopkg.in/yaml.v3"
)

// gatherEventCallback handles the tool gather result event callback
func (ts *ToolService) gatherEventCallback(msg *nats.Msg) {
	// Check if context was cancelled
	select {
	case <-ts.ctx.Done():
		ts.log.Info("Context cancelled, stopping message processing")
		return
	default:
	}

	// Parse NATS message to request struct
	req, err := service.ParseEvent[*service.ToolGatherEventMessage](msg.Data)
	if err != nil {
		ts.log.Error("Failed to unmarshal message to request", "error", err)
		return
	}

	// Log the received message
	ts.log.Info("Received and validated tool result message",
		"tool_run_id", req.Msg.ToolRunId,
		"thread_id", req.H.ThreadID,
		"connection_id", req.H.ConnectionID,
		"user_id", req.H.UserID,
		"task_id", req.H.TaskID,
	)
	ts.log.Debug("Tool result", "result", req.Msg.Content)

	// Get the database queries
	queries := db.New(ts.s.GetDB())

	// Update the Tool Run
	toolRunStatus, err := queries.GetToolRunStatusByID(ts.ctx, req.Msg.ToolRunId)
	if err != nil {
		if err == pgx.ErrNoRows {
			ts.log.Info("No tool run status found. Might be finished")
			return
		}
		ts.log.Error("Failed to get tool run status", "error", err)
		return
	}

	// Calculate duration as the difference between timestamps in seconds
	durationSeconds := req.M.Timestamp.Sub(toolRunStatus.CreatedAt.Time).Seconds()
	duration := pgtype.Float8{Float64: durationSeconds, Valid: true}

	// Store the tool result content in database for later retrieval
	status := db.ToolRunStatusSuccess
	if req.Msg.IsError {
		status = db.ToolRunStatusFailed
	}

	toolRunStatus, err = queries.UpdateToolRunStatusByID(
		ts.ctx,
		db.UpdateToolRunStatusByIDParams{
			ID:       req.Msg.ToolRunId,
			Result:   req.Msg.Content, // Store the actual result content
			Status:   status,
			Duration: duration,
		},
	)
	if err != nil {
		ts.log.Error("Failed to update tool run status", "tool_run_id", req.Msg.ToolRunId, "status", status, "error", err)
		return
	}
	ts.log.Info("Updated tool run status", "tool_run_id", req.Msg.ToolRunId, "status", status)

	// Get agent information for cache control and result formatting
	agent, err := queries.GetAgentByID(ts.ctx, toolRunStatus.AgentID)
	if err != nil {
		ts.log.Error("Failed to get agent information", "agent_id", toolRunStatus.AgentID, "error", err)
		// Continue with empty agent - no cache control will be added
		agent = db.Agent{}
	}

	// Create tool result block using helper function
	toolResultBlock, err := ts.createToolResultBlock(toolRunStatus.ID, req.Msg.Content, req.Msg.ResultType, req.Msg.IsError, agent)
	if err != nil {
		ts.log.Error("Failed to create tool result block", "error", err)
		return
	}

	// Create anthropic Message
	resultMessages := anthropic.MessageParam{
		Role: anthropic.MessageParamRoleUser,
		Content: []anthropic.ContentBlockParamUnion{
			{
				OfToolResult: toolResultBlock,
			},
		},
	}

	// Checking if is child tool
	if toolRunStatus.ParentRunID.String != "" {
		// Checking if parent tool exist
		parentToolRunStatus, err := queries.GetToolRunStatusByID(ts.ctx, toolRunStatus.ParentRunID.String)
		if err != nil {
			if err == pgx.ErrNoRows {
				ts.log.Error("Parent tool does not exist", "parent_tool_run_id", toolRunStatus.ParentRunID.String)
				// Continue to treat as normal tool run
			}
			ts.log.Error("Failed to retrieve information about parent tool run status", "parent_tool_run_id", toolRunStatus.ParentRunID.String, "error", err)
		}

		// Checking for child tool status
		isCompleted, err := queries.CheckIfAllChildToolRunStatusAreCompleted(ts.ctx, toolRunStatus.ParentRunID)
		if err != nil {
			ts.log.Error("Failed to check if all child tool run status are completed", "error", err)
			return
		}

		if !isCompleted {
			ts.log.Warn("All the child tool have not completed, skipping further processing and waiting")
			return
		}
		ts.log.Info("All tool have been completed, update the parent tool run status")

		// Replace the toolRunStatus with the parentToolRunStatus
		toolRunStatus, err = queries.UpdateToolRunStatusToSuccessByID(
			ts.ctx,
			db.UpdateToolRunStatusToSuccessByIDParams{
				ID:       parentToolRunStatus.ID,
				Duration: duration,
			},
		)
		if err != nil {
			ts.log.Error("Failed to update parent tool run status to SUCCESS", "tool_run_id", parentToolRunStatus.ID, "error", err)
			return
		}
		ts.log.Info("Update parent tool run status to SUCCESS", "tool_run_id", parentToolRunStatus.ID)

		// Get the parent tool run name
		isTempParallelToolManagement, err := queries.IsTempParallelToolManagement(ts.ctx, parentToolRunStatus.ID)
		if err != nil {
			ts.log.Error("Failed to check whether the parent tool run is a temp parallel tool management", "error", err)
			return
		}
		if isTempParallelToolManagement {
			// Get all child tool runs and aggregate their results for temp parallel management
			childToolRuns, err := queries.GetChildToolRunStatusByParentID(ts.ctx, pgtype.Text{String: parentToolRunStatus.ID, Valid: true})
			if err != nil {
				ts.log.Error("Failed to get child tool runs", "parent_id", parentToolRunStatus.ID, "error", err)
				return
			}

			// Create aggregated tool results message with multiple tool result blocks
			resultMessages = anthropic.MessageParam{
				Role:    anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{},
			}

			// Process each child tool result as separate tool result blocks
			for _, childToolRun := range childToolRuns {
				if childToolRun.Status != db.ToolRunStatusSuccess && childToolRun.Status != db.ToolRunStatusFailed {
					ts.log.Warn("Skipping incomplete child tool", "child_id", childToolRun.ID, "status", childToolRun.Status)
					continue
				}

				// Infer result type from stored content structure
				var resultType db.ResultMessageType = db.ResultMessageTypeText // Default to text
				var resultContent map[string]any
				if err := json.Unmarshal(childToolRun.Result, &resultContent); err == nil {
					if resultContent["type"] == "image" || resultContent["media_type"] != nil {
						resultType = db.ResultMessageTypeImage
					}
				}

				// Create tool result block using helper function
				isError := childToolRun.Status == db.ToolRunStatusFailed
				toolResultBlock, err := ts.createToolResultBlock(childToolRun.ID, childToolRun.Result, resultType, isError, agent)
				if err != nil {
					ts.log.Error("Failed to create child tool result block", "child_id", childToolRun.ID, "error", err)
					continue
				}

				// Add this tool result to the message
				resultMessages.Content = append(resultMessages.Content,
					anthropic.ContentBlockParamUnion{
						OfToolResult: toolResultBlock,
					},
				)
			}

			ts.log.Info("Aggregated parallel tool results", "parent_id", parentToolRunStatus.ID, "child_count", len(childToolRuns))
		} else {
			// This is batch_tool - return single tool result block with batch tool run ID
			// and content as a flat list of all child tool content blocks
			childToolRuns, err := queries.GetChildToolRunStatusByParentID(ts.ctx, pgtype.Text{String: parentToolRunStatus.ID, Valid: true})
			if err != nil {
				ts.log.Error("Failed to get child tool runs for batch_tool", "parent_id", parentToolRunStatus.ID, "error", err)
				return
			}

			// Collect all content blocks from child tools in order
			var allContentBlocks []anthropic.ToolResultBlockParamContentUnion
			var processedCount int

			for _, childToolRun := range childToolRuns {
				if childToolRun.Status != db.ToolRunStatusSuccess && childToolRun.Status != db.ToolRunStatusFailed {
					ts.log.Warn("Skipping incomplete child tool in batch", "child_id", childToolRun.ID, "status", childToolRun.Status)
					continue
				}

				// Determine result type and error status
				var resultType db.ResultMessageType = db.ResultMessageTypeText // Default to text
				isError := childToolRun.Status == db.ToolRunStatusFailed

				if childToolRun.Result != nil {
					// Infer result type from stored content structure
					var resultContent map[string]any
					if err := json.Unmarshal(childToolRun.Result, &resultContent); err == nil {
						if resultContent["type"] == "image" || resultContent["media_type"] != nil {
							resultType = db.ResultMessageTypeImage
						}
					}
				} else {
					// Handle null result - create appropriate error/success message
					var fallbackResult map[string]any
					if isError {
						fallbackResult = map[string]any{
							"error": "Tool execution failed with no result",
						}
					} else {
						fallbackResult = map[string]any{
							"text": "Tool completed successfully with no result",
						}
					}
					childToolRun.Result, _ = db.NewJsonRaw(fallbackResult)
				}

				// Use the existing createToolResultContent function to get proper content blocks
				childContentBlocks, err := ts.createToolResultContent(childToolRun.Result, resultType, isError)
				if err != nil {
					ts.log.Error("Failed to create content for child tool", "child_id", childToolRun.ID, "error", err)
					// Create fallback error content block
					errorContent, _ := db.NewJsonRaw(map[string]string{
						"error": fmt.Sprintf("Failed to process result: %v", err),
					})
					childContentBlocks, _ = ts.createToolResultContent(errorContent, db.ResultMessageTypeText, true)
				}

				// Add all content blocks from this child tool to the batch
				allContentBlocks = append(allContentBlocks, childContentBlocks...)
				processedCount++
			}

			// Create single tool result block with batch tool run ID and combined content
			batchToolResultBlock := &anthropic.ToolResultBlockParam{
				Type:      "tool_result",
				ToolUseID: parentToolRunStatus.ID,
				Content:   allContentBlocks,
				IsError:   param.Opt[bool]{Value: false}, // Batch tool itself doesn't error, individual children might
			}

			// Add cache control if appropriate
			if ts.shouldUseCacheControl(agent, allContentBlocks) {
				batchToolResultBlock.CacheControl = anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
			}

			// Create result message with single tool result block
			resultMessages = anthropic.MessageParam{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{
						OfToolResult: batchToolResultBlock,
					},
				},
			}

			ts.log.Info("Created batch_tool result", "batch_id", parentToolRunStatus.ID, "child_count", processedCount, "total_content_blocks", len(allContentBlocks))
		}
	}

	// Convert from Anthropic Message to db.JsonRaw
	messages, err := db.NewJsonRaw(resultMessages)
	if err != nil {
		ts.log.Error("Unable to create new jsonRaw for result message")
		return
	}

	// Publish event to TaskHandlerExecute
	event := service.NewEvent(
		&service.TaskExecuteEventMessage{
			AgentId:     toolRunStatus.AgentID,
			RecipientId: toolRunStatus.RecipientID,
			Messages:    []db.JsonRaw{messages},
		},
		req.H,
		&service.EventMetadata{
			TraceID:   req.M.TraceID,
			Timestamp: time.Now(),
		},
	)
	err = event.Publish(ts.s.GetNATS())
	if err != nil {
		ts.log.Error("Failed to publish to task execute event", "error", err)
	}
}

// createToolResultContent creates the content array for a tool result based on the result type
func (ts *ToolService) createToolResultContent(resultContent db.JsonRaw, resultType db.ResultMessageType, isError bool) ([]anthropic.ToolResultBlockParamContentUnion, error) {
	var content []anthropic.ToolResultBlockParamContentUnion

	switch resultType {
	case db.ResultMessageTypeText:
		// Unmarshal JSON the text result
		var textResult struct {
			Text     string                             `json:"text"`
			Citation []anthropic.TextCitationParamUnion `json:"citation,omitzero"`
		}
		b, _ := resultContent.MarshalJSON()

		// Handle error content differently - look for "error" field instead of "text" field
		if isError {
			var errorResult map[string]any
			if err := json.Unmarshal(b, &errorResult); err == nil {
				if errorText, ok := errorResult["error"].(string); ok {
					textResult.Text = errorText
					textResult.Citation = []anthropic.TextCitationParamUnion{}
				} else {
					// Fallback to raw JSON if no error field
					textResult.Text = string(b)
					textResult.Citation = []anthropic.TextCitationParamUnion{}
				}
			} else {
				// If parsing fails, use raw bytes
				textResult.Text = string(b)
				textResult.Citation = []anthropic.TextCitationParamUnion{}
			}
		} else {
			// Normal success case - parse as usual
			err := json.Unmarshal(b, &textResult)
			if err != nil {
				ts.log.Warn("Failed to parse text result with citation, attempting fallback parsing", "error", err)

				// Try to parse as a generic JSON object and extract the text field if it exists
				var fallbackResult map[string]any
				if fallbackErr := json.Unmarshal(b, &fallbackResult); fallbackErr == nil {
					if text, ok := fallbackResult["text"].(string); ok {
						textResult.Text = text
					} else {
						// If no text field, use the whole JSON as text
						textResult.Text = string(b)
					}
					textResult.Citation = []anthropic.TextCitationParamUnion{}
				} else {
					// If all parsing fails, use the raw bytes as text
					textResult.Text = string(b)
					textResult.Citation = []anthropic.TextCitationParamUnion{}
				}
			}
		}

		content = append(content, anthropic.ToolResultBlockParamContentUnion{
			OfText: &anthropic.TextBlockParam{
				Type:      "text",
				Text:      textResult.Text,
				Citations: textResult.Citation,
			},
		})

	case db.ResultMessageTypeImage:
		// Unmarshal JSON the image result
		var imageResult map[string]any
		err := json.Unmarshal(resultContent, &imageResult)
		if err != nil {
			return nil, fmt.Errorf("unable to read the image result: %w", err)
		}

		imageBlock := &anthropic.ImageBlockParam{
			Type:   "images",
			Source: anthropic.ImageBlockParamSourceUnion{},
		}

		switch imageResult["type"].(string) {
		case "base64":
			imageBlock.Source = anthropic.ImageBlockParamSourceUnion{
				OfBase64: &anthropic.Base64ImageSourceParam{
					Type:      "base64",
					Data:      imageResult["data"].(string),
					MediaType: imageResult["media_type"].(anthropic.Base64ImageSourceMediaType),
				},
			}
		case "url":
			imageBlock.Source = anthropic.ImageBlockParamSourceUnion{
				OfURL: &anthropic.URLImageSourceParam{
					Type: "url",
					URL:  imageResult["data"].(string),
				},
			}
		default:
			return nil, fmt.Errorf("unsupported image result type: %s", imageResult["type"].(string))
		}

		content = append(content, anthropic.ToolResultBlockParamContentUnion{
			OfImage: imageBlock,
		})
	}

	return content, nil
}

// shouldUseCacheControl determines if cache control should be added based on text length and agent model
func (ts *ToolService) shouldUseCacheControl(agent db.Agent, content []anthropic.ToolResultBlockParamContentUnion) bool {
	const maxTextSizeBytes = 40 * 1024 // 40kB threshold

	// Fast path: check text size first with early exit
	totalTextSize := 0
	for _, contentItem := range content {
		if contentItem.OfText != nil {
			textLen := len(contentItem.OfText.Text)
			totalTextSize += textLen

			// Early exit as soon as we exceed threshold - no need to continue counting
			if totalTextSize > maxTextSizeBytes {
				// Text is large enough, now check if model supports caching
				return ts.modelSupportsCaching(agent)
			}
		}
	}

	// Text is small, no cache control needed regardless of model
	return false
}

// modelSupportsCaching checks if the agent's model supports cache control
func (ts *ToolService) modelSupportsCaching(agent db.Agent) bool {
	if !agent.Specs.Valid {
		return false
	}

	// Parse agent spec to get model ID
	var agentSpec struct {
		Model struct {
			ModelID string `yaml:"model_id"`
		} `yaml:"model"`
	}

	if err := yaml.Unmarshal([]byte(agent.Specs.String), &agentSpec); err != nil {
		return false
	}

	// Only add CacheControl if model doesn't contain "haiku-3" or "sonnet-3-5"
	modelID := strings.ToLower(agentSpec.Model.ModelID)
	return !strings.Contains(modelID, "3-haiku") && !strings.Contains(modelID, "3-5-sonnet")
}

// createToolResultBlock creates a complete tool result block with proper content and cache control
func (ts *ToolService) createToolResultBlock(toolRunID string, resultContent db.JsonRaw, resultType db.ResultMessageType, isError bool, agent db.Agent) (*anthropic.ToolResultBlockParam, error) {
	// Create tool result content
	content, err := ts.createToolResultContent(resultContent, resultType, isError)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool result content: %w", err)
	}

	// Create tool result block
	toolResultBlock := &anthropic.ToolResultBlockParam{
		Type:      "tool_result",
		ToolUseID: toolRunID,
		Content:   content,
		IsError:   param.Opt[bool]{Value: isError},
	}

	// Add cache control if text is large enough and model supports it
	if ts.shouldUseCacheControl(agent, content) {
		toolResultBlock.CacheControl = anthropic.CacheControlEphemeralParam{
			Type: "ephemeral",
		}
	}

	return toolResultBlock, nil
}
