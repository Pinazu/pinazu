package agents

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/google/uuid"
	"github.com/pinazu/core/internal/db"
	"github.com/pinazu/core/internal/service"
)

// handleBedrockRequest handles requests for Bedrock models
func (as *AgentService) handleBedrockRequest(m []anthropic.MessageParam, spec *AgentSpecs, header *service.EventHeaders, meta *service.EventMetadata) (*anthropic.MessageParam, string, error) {
	// Fetch and convert tools for this agent
	var tools []types.Tool
	if len(spec.ToolRefs) > 0 {
		var err error
		tools, err = as.fetchBedrockTools(spec.ToolRefs)
		if err != nil {
			as.log.Error("Failed to convert tools to Bedrock format", "error", err)
			return nil, "", fmt.Errorf("failed to convert tools to Bedrock format: %w", err)
		}

		as.log.Debug("Loaded tools for agent", "tool_count", len(tools), "tool_names", func() []string {
			names := make([]string, len(tools))
			for i, t := range tools {
				if toolSpec, ok := t.(*types.ToolMemberToolSpec); ok && toolSpec.Value.Name != nil {
					names[i] = *toolSpec.Value.Name
				}
			}
			return names
		}())
	}

	// Check for output format and add prefill message
	if len(spec.Model.ResponseFormat) > 0 {
		// Add the prefill assistant response to start JSON output
		prefillMsg := anthropic.NewAssistantMessage(anthropic.NewTextBlock("{"))
		m = append(m, prefillMsg)
	}

	// Convert Anthropic messages to Bedrock format
	bedrockMessages := make([]types.Message, len(m))
	for i, msg := range m {
		converted, err := convertFromAnthropicToBedrock(msg)
		if err != nil {
			as.log.Error("Failed to convert Anthropic message to Bedrock format", "error", err, "index", i)
			return nil, "", fmt.Errorf("failed to convert message at index %d: %w", i, err)
		}
		bedrockMessages[i] = converted
	}

	// Initialize variables to accumulate content
	var (
		stop                        types.StopReason
		response                    types.Message
		accumulatedTextContent      strings.Builder
		accumulatedReasoningContent strings.Builder
		content                     []types.ContentBlock
	)

	if spec.Model.Stream {
		// Initialize state tracking for streaming event normalization
		as.contentBlockStartSent = make(map[int64]bool)

		params := &bedrockruntime.ConverseStreamInput{
			ModelId:  aws.String(spec.Model.ModelID),
			Messages: bedrockMessages,
			InferenceConfig: &types.InferenceConfiguration{
				MaxTokens: aws.Int32(int32(spec.Model.MaxTokens)),
			},
			System: getBedrockSystemPrompt(spec),
		}

		// Include tools if available
		if len(tools) > 0 {
			params.ToolConfig = &types.ToolConfiguration{
				Tools: tools,
			}
		}

		// Conditionally include Temperature if provided by user
		if spec.Model.Temperature != 0 {
			params.InferenceConfig.Temperature = aws.Float32(float32(spec.Model.Temperature))
		}

		// Conditionally include TopP if provided by user
		if spec.Model.TopP != 0 {
			params.InferenceConfig.TopP = aws.Float32(float32(spec.Model.TopP))
		}

		// Conditionally include Thinking configuration if enabled
		if spec.Model.Thinking.Enabled {
			params.AdditionalModelRequestFields = document.NewLazyDocument(map[string]any{
				"thinking": map[string]any{
					"type":          "enabled",
					"budget_tokens": spec.Model.Thinking.BudgetToken,
				},
			})
		}

		// Handle tool choice if specified
		if spec.ToolChoice != (ToolChoice{}) {
			switch spec.ToolChoice.Type {
			case "auto":
				params.ToolConfig.ToolChoice = &types.ToolChoiceMemberAuto{
					Value: types.AutoToolChoice{},
				}
			case "any":
				params.ToolConfig.ToolChoice = &types.ToolChoiceMemberAny{
					Value: types.AnyToolChoice{},
				}
			case "tool":
				params.ToolConfig.ToolChoice = &types.ToolChoiceMemberTool{
					Value: types.SpecificToolChoice{
						Name: aws.String(spec.ToolChoice.Name),
					},
				}
			}
		}

		response, err := as.bc.ConverseStream(as.ctx, params)
		if err != nil {
			as.log.Error("Error calling Bedrock Converse Stream API", "error", err)
			return nil, "", err
		}

		as.log.Debug("Streaming response from Bedrock API")
		stream := response.GetStream()
		for event := range stream.Events() {
			// Publish the streaming event to websocket client
			as.publishBedrockStreamEvent(event, header, meta)

			switch v := event.(type) {
			case *types.ConverseStreamOutputMemberMessageStart:
			case *types.ConverseStreamOutputMemberContentBlockStart:
			case *types.ConverseStreamOutputMemberContentBlockDelta:
				if v.Value.Delta != nil {
					switch delta := v.Value.Delta.(type) {
					case *types.ContentBlockDeltaMemberText:
						accumulatedTextContent.WriteString(delta.Value)
					case *types.ContentBlockDeltaMemberReasoningContent:
						switch reasoningDelta := delta.Value.(type) {
						case *types.ReasoningContentBlockDeltaMemberText:
							if reasoningDelta.Value != "" {
								accumulatedReasoningContent.WriteString(reasoningDelta.Value)
							}
						case *types.ReasoningContentBlockDeltaMemberSignature:
							// Handle signature - theo AWS docs, signature cần được preserve cho multi-turn conversations
							as.log.Debug("Received reasoning signature", "signature", reasoningDelta.Value)
							// Signature được sử dụng để verify reasoning text, không cần accumulate
						case *types.ReasoningContentBlockDeltaMemberRedactedContent:
							// Handle redacted content - content được encrypt bởi model provider vì lý do an toàn
							as.log.Debug("Received redacted reasoning content")
							// Redacted content không ảnh hưởng đến quality của response
						default:
							as.log.Warn("Unexpected reasoning content delta type", "type", fmt.Sprintf("%T", reasoningDelta))
						}
					}
				}
			case *types.ConverseStreamOutputMemberContentBlockStop:
				// Add completed content blocks to the response
				if accumulatedReasoningContent.Len() > 0 {
					reasoningContent := &types.ReasoningContentBlockMemberReasoningText{
						Value: types.ReasoningTextBlock{
							Text: aws.String(accumulatedReasoningContent.String()),
						},
					}
					content = append(content, &types.ContentBlockMemberReasoningContent{
						Value: reasoningContent,
					})
					accumulatedReasoningContent.Reset()
				}
				if accumulatedTextContent.Len() > 0 {
					content = append(content, &types.ContentBlockMemberText{
						Value: accumulatedTextContent.String(),
					})
					accumulatedTextContent.Reset()
				}
			case *types.ConverseStreamOutputMemberMessageStop:
				stop = v.Value.StopReason
				as.log.Debug("Message stopped", "stop_reason", stop)
			case *types.ConverseStreamOutputMemberMetadata:
				if v.Value.Usage != nil {
					as.log.Info("Bedrock usage metrics",
						"input_tokens", *v.Value.Usage.InputTokens,
						"output_tokens", *v.Value.Usage.OutputTokens,
						"total_tokens", *v.Value.Usage.TotalTokens,
					)
				}
				if v.Value.Metrics != nil {
					as.log.Info("Bedrock latency metrics", "latency_ms", *v.Value.Metrics.LatencyMs)
				}
			}
		}

		if err := stream.Err(); err != nil {
			as.log.Error("Error streaming response from Bedrock", "error", err)
			return nil, "", err
		}

		// Clean up state tracking to prevent memory leaks
		as.contentBlockStartSent = nil

		// No need to create content blocks here as they are handled in ContentBlockStop events
	} else {
		params := &bedrockruntime.ConverseInput{
			ModelId:  aws.String(spec.Model.ModelID),
			Messages: bedrockMessages,
			InferenceConfig: &types.InferenceConfiguration{
				MaxTokens: aws.Int32(int32(spec.Model.MaxTokens)),
			},
			System: getBedrockSystemPrompt(spec),
		}

		// Include tools if available
		if len(tools) > 0 {
			params.ToolConfig = &types.ToolConfiguration{
				Tools: tools,
			}
		}

		// Conditionally include Temperature if provided by user
		if spec.Model.Temperature != 0 {
			params.InferenceConfig.Temperature = aws.Float32(float32(spec.Model.Temperature))
		}

		// Conditionally include TopP if provided by user
		if spec.Model.TopP != 0 {
			params.InferenceConfig.TopP = aws.Float32(float32(spec.Model.TopP))
		}

		// Conditionally include Thinking configuration if enabled
		if spec.Model.Thinking.Enabled {
			params.AdditionalModelRequestFields = document.NewLazyDocument(map[string]any{
				"thinking": map[string]any{
					"type":          "enabled",
					"budget_tokens": spec.Model.Thinking.BudgetToken,
				},
			})
		}

		// Handle tool choice if specified
		if spec.ToolChoice != (ToolChoice{}) {
			switch spec.ToolChoice.Type {
			case "auto":
				params.ToolConfig.ToolChoice = &types.ToolChoiceMemberAuto{
					Value: types.AutoToolChoice{},
				}
			case "any":
				params.ToolConfig.ToolChoice = &types.ToolChoiceMemberAny{
					Value: types.AnyToolChoice{},
				}
			case "tool":
				params.ToolConfig.ToolChoice = &types.ToolChoiceMemberTool{
					Value: types.SpecificToolChoice{
						Name: aws.String(spec.ToolChoice.Name),
					},
				}
			}
		}

		resp, err := as.bc.Converse(as.ctx, params)
		if err != nil {
			as.log.Error("Error calling Bedrock Converse API", "error", err)
			return nil, "", err
		}

		// Extract message content and stop reason from response
		stop = resp.StopReason
		if msgOutput, ok := resp.Output.(*types.ConverseOutputMemberMessage); ok {
			content = msgOutput.Value.Content
		}

		// Log usage metrics if available
		if resp.Usage != nil {
			as.log.Info("Bedrock usage metrics",
				"input_tokens", *resp.Usage.InputTokens,
				"output_tokens", *resp.Usage.OutputTokens,
				"total_tokens", *resp.Usage.TotalTokens,
			)
		}
	}

	// Create response message with accumulated content
	response = types.Message{
		Role:    types.ConversationRoleAssistant,
		Content: content,
	}

	// Convert Bedrock response back to Anthropic format
	anthropicResponse, err := convertFromBedrockToAnthropic(response)
	if err != nil {
		as.log.Error("Failed to convert Bedrock response to Anthropic format", "error", err)
		return nil, "", fmt.Errorf("failed to convert bedrock response: %w", err)
	}

	return &anthropicResponse, string(stop), nil
}

func getBedrockSystemPrompt(spec *AgentSpecs) []types.SystemContentBlock {
	systemText := spec.System
	if systemText == "" {
		systemText = "You are a helpful assistant."
	}

	// Add schema instruction if response format is specified
	if len(spec.Model.ResponseFormat) > 0 {
		schemaBytes, err := json.Marshal(spec.Model.ResponseFormat)
		if err == nil {
			schemaInstruction := fmt.Sprintf("\n\nYou must respond with valid JSON that matches this exact schema:\n%s\n\n", string(schemaBytes))
			systemText += schemaInstruction
		}
	}

	return []types.SystemContentBlock{
		&types.SystemContentBlockMemberText{
			Value: systemText,
		},
	}
}

// convertFromAnthropicToBedrock converts an Anthropic message to Bedrock format
func convertFromAnthropicToBedrock(input anthropic.MessageParam) (types.Message, error) {
	// Convert role
	var role types.ConversationRole
	switch input.Role {
	case "user":
		role = types.ConversationRoleUser
	case "assistant":
		role = types.ConversationRoleAssistant
	default:
		return types.Message{}, fmt.Errorf("unsupported role: %s", input.Role)
	}

	// Convert content blocks
	var bedrockContent []types.ContentBlock
	for _, contentBlock := range input.Content {
		// Handle text blocks
		if textBlock := contentBlock.OfText; textBlock != nil {
			bedrockContent = append(bedrockContent, &types.ContentBlockMemberText{
				Value: textBlock.Text,
			})
			continue
		}

		// Handle thinking blocks
		if thinkingBlock := contentBlock.OfThinking; thinkingBlock != nil {
			// Map thinking to reasoning content
			reasoningContent := &types.ReasoningContentBlockMemberReasoningText{
				Value: types.ReasoningTextBlock{
					Text: aws.String(thinkingBlock.Thinking),
				},
			}
			bedrockContent = append(bedrockContent, &types.ContentBlockMemberReasoningContent{
				Value: reasoningContent,
			})
			continue
		}

		// Handle tool use blocks
		if toolUseBlock := contentBlock.OfToolUse; toolUseBlock != nil {
			// Convert the input to document.Interface
			inputDoc := document.NewLazyDocument(toolUseBlock.Input)
			bedrockContent = append(bedrockContent, &types.ContentBlockMemberToolUse{
				Value: types.ToolUseBlock{
					ToolUseId: aws.String(toolUseBlock.ID),
					Name:      aws.String(toolUseBlock.Name),
					Input:     inputDoc,
				},
			})
			continue
		}

		// Handle tool result blocks
		if toolResultBlock := contentBlock.OfToolResult; toolResultBlock != nil {
			var toolResultContent []types.ToolResultContentBlock
			for _, content := range toolResultBlock.Content {
				if textContent := content.OfText; textContent != nil {
					toolResultContent = append(toolResultContent, &types.ToolResultContentBlockMemberText{
						Value: textContent.Text,
					})
				}
			}

			bedrockContent = append(bedrockContent, &types.ContentBlockMemberToolResult{
				Value: types.ToolResultBlock{
					ToolUseId: aws.String(toolResultBlock.ToolUseID),
					Content:   toolResultContent,
					Status:    types.ToolResultStatusSuccess, // Default to success
				},
			})
			continue
		}

	}

	return types.Message{
		Role:    role,
		Content: bedrockContent,
	}, nil
}

// convertFromBedrockToAnthropic converts a Bedrock message to Anthropic format
func convertFromBedrockToAnthropic(input types.Message) (anthropic.MessageParam, error) {
	// Convert role
	var role string
	switch input.Role {
	case types.ConversationRoleUser:
		role = "user"
	case types.ConversationRoleAssistant:
		role = "assistant"
	default:
		return anthropic.MessageParam{}, fmt.Errorf("unsupported bedrock role: %s", input.Role)
	}

	// Convert content blocks
	var anthropicContent []anthropic.ContentBlockParamUnion
	for _, contentBlock := range input.Content {
		// Handle text blocks
		if textBlock, ok := contentBlock.(*types.ContentBlockMemberText); ok {
			textContent := anthropic.NewTextBlock(textBlock.Value)
			// Set the Type field for consistency
			if textContent.OfText != nil {
				textContent.OfText.Type = "text"
			}
			anthropicContent = append(anthropicContent, textContent)
			continue
		}

		// Handle reasoning content blocks (map to thinking)
		if reasoningBlock, ok := contentBlock.(*types.ContentBlockMemberReasoningContent); ok {
			if textReasoning, ok := reasoningBlock.Value.(*types.ReasoningContentBlockMemberReasoningText); ok {
				if textReasoning.Value.Text != nil {
					thinkingContent := anthropic.NewThinkingBlock("", *textReasoning.Value.Text)
					// Set the Type field for consistency
					if thinkingContent.OfThinking != nil {
						thinkingContent.OfThinking.Type = "thinking"
					}
					anthropicContent = append(anthropicContent, thinkingContent)
				}
			}
			continue
		}

		// Handle tool use blocks
		if toolUseBlock, ok := contentBlock.(*types.ContentBlockMemberToolUse); ok {
			toolID := ""
			toolName := ""
			if toolUseBlock.Value.ToolUseId != nil {
				toolID = *toolUseBlock.Value.ToolUseId
			}
			if toolUseBlock.Value.Name != nil {
				toolName = *toolUseBlock.Value.Name
			}

			anthropicContent = append(anthropicContent, anthropic.NewToolUseBlock(
				toolID,
				toolUseBlock.Value.Input,
				toolName,
			))
			continue
		}

		// Handle tool result blocks
		if toolResultBlock, ok := contentBlock.(*types.ContentBlockMemberToolResult); ok {
			toolUseID := ""
			if toolResultBlock.Value.ToolUseId != nil {
				toolUseID = *toolResultBlock.Value.ToolUseId
			}

			var resultContent []anthropic.ToolResultBlockParamContentUnion
			for _, content := range toolResultBlock.Value.Content {
				if textContent, ok := content.(*types.ToolResultContentBlockMemberText); ok {
					resultContent = append(resultContent, anthropic.ToolResultBlockParamContentUnion{
						OfText: &anthropic.TextBlockParam{
							Type: "text",
							Text: textContent.Value,
						},
					})
				}
			}

			isError := toolResultBlock.Value.Status == types.ToolResultStatusError
			anthropicContent = append(anthropicContent, anthropic.ContentBlockParamUnion{
				OfToolResult: &anthropic.ToolResultBlockParam{
					Type:      "tool_result",
					ToolUseID: toolUseID,
					Content:   resultContent,
					IsError:   anthropic.Bool(isError),
				},
			})
			continue
		}

	}

	return anthropic.MessageParam{
		Role:    anthropic.MessageParamRole(role),
		Content: anthropicContent,
	}, nil
}

// publishBedrockStreamEvent publishes Bedrock stream events to WebSocket clients after converting to Anthropic format
func (as *AgentService) publishBedrockStreamEvent(event types.ConverseStreamOutput, header *service.EventHeaders, meta *service.EventMetadata) {
	// Convert Bedrock stream event to Anthropic format (may return multiple events)
	anthropicEvents := convertBedrockStreamEventToAnthropic(event, as.contentBlockStartSent)
	if len(anthropicEvents) == 0 {
		// Skip events that don't map to Anthropic format
		return
	}

	// Publish all events in sequence
	for _, anthropicEvent := range anthropicEvents {
		// Convert to WebsocketResponseEventMessage
		wsEvent := ToWebsocketResponseEventMessage(*anthropicEvent, db.ProviderModelBedrock)

		// Publish to NATS for WebSocket handler
		newEvent := service.NewEvent(wsEvent, header, &service.EventMetadata{
			TraceID:   meta.TraceID,
			Timestamp: time.Now().UTC(),
		})

		// Publish using the new PublishWithUser method for WebSocket events
		if err := newEvent.PublishWithUser(as.s.GetNATS(), header.UserID); err != nil {
			as.log.Error("Failed to publish websocket event", "error", err)
			return
		}
	}
}

// convertBedrockStreamEventToAnthropic converts Bedrock stream events to Anthropic format
// Returns slice of events to allow injecting synthetic content_block_start events
func convertBedrockStreamEventToAnthropic(event types.ConverseStreamOutput, state map[int64]bool) []*anthropic.MessageStreamEventUnion {
	var events []*anthropic.MessageStreamEventUnion

	switch v := event.(type) {
	case *types.ConverseStreamOutputMemberMessageStart:
		events = append(events, &anthropic.MessageStreamEventUnion{
			Type: "message_start",
			Message: anthropic.Message{
				ID:    "bedrock-message-id",
				Type:  "message",
				Role:  "assistant",
				Model: "",
			},
		})

	case *types.ConverseStreamOutputMemberContentBlockStart:
		var index int64 = 0
		if v.Value.ContentBlockIndex != nil {
			index = int64(*v.Value.ContentBlockIndex)
		}

		// Mark that content_block_start has been sent for this index
		state[index] = true

		// Create a simple event since we don't have exact ContentBlock structure
		events = append(events, &anthropic.MessageStreamEventUnion{
			Type:  "content_block_start",
			Index: index,
		})

	case *types.ConverseStreamOutputMemberContentBlockDelta:
		var index int64 = 0
		if v.Value.ContentBlockIndex != nil {
			index = int64(*v.Value.ContentBlockIndex)
		}

		// Inject synthetic content_block_start if not already sent
		if !state[index] {
			syntheticStart := &anthropic.MessageStreamEventUnion{
				Type:  "content_block_start",
				Index: index,
			}
			events = append(events, syntheticStart)
			state[index] = true // Mark as sent
		}

		if v.Value.Delta != nil {
			switch delta := v.Value.Delta.(type) {
			case *types.ContentBlockDeltaMemberText:
				// Create a delta event with actual text content
				deltaEvent := &anthropic.MessageStreamEventUnion{
					Type:  "content_block_delta",
					Index: index,
				}
				// Set delta fields directly
				deltaEvent.Delta.Type = "text_delta"
				deltaEvent.Delta.Text = delta.Value
				events = append(events, deltaEvent)

			case *types.ContentBlockDeltaMemberReasoningContent:
				// Handle all types of reasoning content deltas
				switch reasoningDelta := delta.Value.(type) {
				case *types.ReasoningContentBlockDeltaMemberText:
					if reasoningDelta != nil {
						// Create a delta event with actual thinking content
						deltaEvent := &anthropic.MessageStreamEventUnion{
							Type:  "content_block_delta",
							Index: index,
						}
						// Set delta fields directly
						deltaEvent.Delta.Type = "thinking_delta"
						deltaEvent.Delta.Thinking = reasoningDelta.Value
						events = append(events, deltaEvent)
					}
				case *types.ReasoningContentBlockDeltaMemberSignature:
					// Signature deltas are handled but don't generate stream events
					// as they are used internally for verification
				case *types.ReasoningContentBlockDeltaMemberRedactedContent:
					// Redacted content deltas are handled but don't generate stream events
					// as the content is encrypted for safety reasons
				}
			}
		}

	case *types.ConverseStreamOutputMemberContentBlockStop:
		var index int64 = 0
		if v.Value.ContentBlockIndex != nil {
			index = int64(*v.Value.ContentBlockIndex)
		}
		events = append(events, &anthropic.MessageStreamEventUnion{
			Type:  "content_block_stop",
			Index: index,
		})

	case *types.ConverseStreamOutputMemberMessageStop:
		events = append(events, &anthropic.MessageStreamEventUnion{
			Type: "message_delta",
		})

	case *types.ConverseStreamOutputMemberMetadata:
		events = append(events, &anthropic.MessageStreamEventUnion{
			Type: "message_stop",
		})
	}

	return events
}

// fetchBedrockTools retrieves tools from database based on agent's tool_refs
func (as *AgentService) fetchBedrockTools(toolRefs []uuid.UUID) ([]types.Tool, error) {
	var bedrockTools []types.Tool

	if len(toolRefs) == 0 {
		return nil, nil
	}

	// Fetch tools from database
	queries := db.New(as.s.GetDB())
	tools, err := queries.GetToolsByIDs(as.ctx, toolRefs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tools from database: %w", err)
	}

	// Check if any tools were not found and log warnings
	if len(tools) < len(toolRefs) {
		foundToolIDs := make(map[uuid.UUID]bool)
		for _, tool := range tools {
			foundToolIDs[tool.ID] = true
		}

		for _, toolRef := range toolRefs {
			if !foundToolIDs[toolRef] {
				as.log.Warn("Tool not found in database, will not use this tool", "tool_id", toolRef)
			}
		}
	}

	// Extract tool params
	for _, tool := range tools {
		// Extract description
		description := ""
		if tool.Description.Valid {
			description = tool.Description.String
		}

		// Convert tool config to parameter schema
		var inputSchema document.Interface
		switch tool.Config.Type {
		case db.ToolTypeStandalone:
			standaloneConfig := tool.Config.GetStandalone()
			if standaloneConfig != nil {
				// Convert OpenAPI schema to document for Bedrock
				inputSchema = document.NewLazyDocument(standaloneConfig.Params)
			}
		case db.ToolTypeWorkflow:
			workflowConfig := tool.Config.GetWorkflow()
			if workflowConfig != nil {
				// Convert OpenAPI schema to document for Bedrock
				inputSchema = document.NewLazyDocument(workflowConfig.Params)
			}
		case db.ToolTypeMCP:
			// MCP tools don't have predefined schemas in the database
			// They are dynamically discovered, so we'll skip them for now
			as.log.Debug("Skipping MCP tool - dynamic schema discovery required", "tool_name", tool.Name)
			continue
		default:
			as.log.Warn("Unknown tool type", "tool_name", tool.Name, "type", tool.Config.Type)
			continue
		}

		// Create Bedrock tool parameter
		if inputSchema != nil {
			bedrockTool := &types.ToolMemberToolSpec{
				Value: types.ToolSpecification{
					Name:        aws.String(tool.Name),
					Description: aws.String(description),
					InputSchema: &types.ToolInputSchemaMemberJson{
						Value: inputSchema,
					},
				},
			}
			bedrockTools = append(bedrockTools, bedrockTool)
		}
	}

	return bedrockTools, nil
}
