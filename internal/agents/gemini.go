package agents

import (
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/db"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
	"google.golang.org/genai"
)

// handleGeminiRequest handles requests for Gemini models
func (as *AgentService) handleGeminiRequest(m []anthropic.MessageParam, spec *AgentSpecs, header *service.EventHeaders, meta *service.EventMetadata) (*anthropic.MessageParam, string, error) {
	// Check if Gemini client is available
	if as.gc == nil {
		return nil, "", fmt.Errorf("gemini client is not initialized - API key may be missing")
	}

	// Convert Anthropic messages to Gemini format
	geminiMessages := make([]genai.Content, len(m))
	for i, msg := range m {
		converted, err := convertFromAnthropicToGemini(msg)
		if err != nil {
			as.log.Error("Failed to convert Anthropic message to Gemini format", "error", err, "index", i)
			return nil, "", fmt.Errorf("failed to convert message at index %d: %w", i, err)
		}
		geminiMessages[i] = converted
	}

	// Initialize variables to accumulate content
	var (
		finishReason               genai.FinishReason
		response                   genai.Content
		accumulatedTextContent     strings.Builder
		accumulatedThinkingContent strings.Builder
		parts                      []*genai.Part
	)

	if spec.Model.Stream {
		// Initialize state tracking for streaming event normalization
		as.contentBlockStartSent = make(map[int64]bool)
	}
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: spec.System},
			},
		},
		MaxOutputTokens: int32(spec.Model.MaxTokens),
		Temperature:     aws.Float32(float32(spec.Model.Temperature)),
		TopP:            aws.Float32(float32(spec.Model.TopP)),
		TopK:            aws.Float32(float32(spec.Model.TopK)),
		ThinkingConfig:  getGeminiThinkingConfig(spec),
	}

	// Convert []genai.Content to []*genai.Content
	contentPointers := make([]*genai.Content, len(geminiMessages))
	for i := range geminiMessages {
		contentPointers[i] = &geminiMessages[i]
	}

	// Check for empty input condition (this causes "empty input" API error)
	totalParts := 0
	for _, cp := range contentPointers {
		if cp != nil {
			totalParts += len(cp.Parts)
		}
	}
	
	if totalParts == 0 {
		as.log.Error("❌ GEMINI ERROR: No content parts found in messages - this will cause empty input API error")
		return nil, "", fmt.Errorf("empty input: no content parts found in messages")
	}

	if spec.Model.Stream {
		stream := as.gc.Models.GenerateContentStream(as.ctx, spec.Model.ModelID, contentPointers, config)

		for chunk, err := range stream {
			if err != nil {
				as.log.Error("Error streaming response from Gemini", 
					"error", err,
					"error_type", fmt.Sprintf("%T", err))
				return nil, "", err
			}

			// Publish the streaming event to websocket client
			as.publishGeminiStreamEvent(chunk, header, meta)

			// Accumulate content from streaming response
			if len(chunk.Candidates) > 0 {
				candidate := chunk.Candidates[0]

				// Get finish reason from the last chunk
				if candidate.FinishReason != "" {
					finishReason = candidate.FinishReason
				}

				// Accumulate text and thinking content separately
				if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
					for _, part := range candidate.Content.Parts {
						if part.Text != "" {
							if part.Thought {
								// This is thinking content
								accumulatedThinkingContent.WriteString(part.Text)
							} else {
								// This is regular text content
								accumulatedTextContent.WriteString(part.Text)
							}
						}
					}
				}
			}
		}

		// Clean up state tracking to prevent memory leaks
		as.contentBlockStartSent = nil
	} else {
		resp, err := as.gc.Models.GenerateContent(as.ctx, spec.Model.ModelID, contentPointers, config)
		if err != nil {
			as.log.Error("Error in non-streaming response from Gemini",
				"error", err,
				"error_type", fmt.Sprintf("%T", err),
				"model_id", spec.Model.ModelID)
			return nil, "", fmt.Errorf("failed to response from gemini: %w", err)
		}

		// Extract content and finish reason from non-streaming response
		if len(resp.Candidates) > 0 {
			candidate := resp.Candidates[0]
			finishReason = candidate.FinishReason

			if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						if part.Thought {
							// This is thinking content
							accumulatedThinkingContent.WriteString(part.Text)
						} else {
							// This is regular text content
							accumulatedTextContent.WriteString(part.Text)
						}
					}
				}
			}
		}

		// Log usage metrics if available
		if resp.UsageMetadata != nil {
			as.log.Info("Gemini usage metrics",
				"input_tokens", resp.UsageMetadata.PromptTokenCount,
				"output_tokens", resp.UsageMetadata.CandidatesTokenCount,
				"total_tokens", resp.UsageMetadata.TotalTokenCount,
			)
		}
	}

	// Create response content with accumulated thinking and text
	if accumulatedThinkingContent.Len() > 0 {
		parts = append(parts, &genai.Part{
			Text:    accumulatedThinkingContent.String(),
			Thought: true,
		})
	}
	if accumulatedTextContent.Len() > 0 {
		parts = append(parts, &genai.Part{
			Text:    accumulatedTextContent.String(),
			Thought: false,
		})
	}

	response = genai.Content{
		Role:  "model",
		Parts: parts,
	}

	// Convert Gemini response back to Anthropic format
	anthropicResponse, err := convertFromGeminiToAnthropic(response)
	if err != nil {
		as.log.Error("Failed to convert Gemini response to Anthropic format", "error", err)
		return nil, "", fmt.Errorf("failed to convert gemini response: %w", err)
	}

	// Map finish reasons to stop reasons
	var stop string
	switch finishReason {
	case genai.FinishReasonStop:
		as.log.Info("Gemini Agent stopped with STOP")
		stop = "end_turn"
	case genai.FinishReasonMaxTokens:
		as.log.Info("Gemini Agent stopped with MAX_TOKENS")
		stop = "max_tokens"
	case genai.FinishReasonSafety:
		as.log.Info("Gemini Agent stopped with SAFETY")
		stop = "stop_sequence"
	case genai.FinishReasonRecitation:
		as.log.Info("Gemini Agent stopped with RECITATION")
		stop = "stop_sequence"
	case genai.FinishReasonOther:
		as.log.Info("Gemini Agent stopped with OTHER")
		stop = "stop_sequence"
	case genai.FinishReasonBlocklist:
		as.log.Info("Gemini Agent stopped with BLOCKLIST")
		stop = "stop_sequence"  
	case genai.FinishReasonProhibitedContent:
		as.log.Info("Gemini Agent stopped with PROHIBITED_CONTENT")
		stop = "stop_sequence"
	case genai.FinishReasonMalformedFunctionCall:
		as.log.Info("Gemini Agent stopped with MALFORMED_FUNCTION_CALL")
		stop = "stop_sequence"
	case genai.FinishReasonUnspecified:
		as.log.Warn("Gemini Agent stopped with FINISH_REASON_UNSPECIFIED")
		stop = "stop_sequence"
	default:
		as.log.Warn("Gemini Agent stopped with unknown reason", "finish_reason", finishReason)
		stop = "stop_sequence"
	}

	return &anthropicResponse, stop, nil
}

func getGeminiThinkingConfig(spec *AgentSpecs) *genai.ThinkingConfig {
	if spec.Model.Thinking.Enabled {
		return &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  aws.Int32(int32(spec.Model.Thinking.BudgetToken)),
		}
	}
	return nil
}

// convertFromAnthropicToGemini converts an Anthropic message to Gemini format
func convertFromAnthropicToGemini(input anthropic.MessageParam) (genai.Content, error) {
	// Convert role
	var role string
	switch input.Role {
	case "user":
		role = "user"
	case "assistant":
		role = "model"
	default:
		return genai.Content{}, fmt.Errorf("unsupported role: %s", input.Role)
	}

	// Convert content blocks
	var geminiParts []*genai.Part
	for _, contentBlock := range input.Content {
		// Handle text blocks
		if textBlock := contentBlock.OfText; textBlock != nil {
			geminiParts = append(geminiParts, &genai.Part{
				Text:    textBlock.Text,
				Thought: false,
			})
			continue
		}

		// Handle thinking blocks
		if thinkingBlock := contentBlock.OfThinking; thinkingBlock != nil {
			geminiParts = append(geminiParts, &genai.Part{
				Text:    thinkingBlock.Thinking,
				Thought: true,
			})
			continue
		}

		// Handle tool use blocks - convert to text for now
		if toolUseBlock := contentBlock.OfToolUse; toolUseBlock != nil {
			// Convert tool use to text representation for Gemini
			toolText := fmt.Sprintf("Tool use: %s with ID: %s", toolUseBlock.Name, toolUseBlock.ID)
			geminiParts = append(geminiParts, &genai.Part{
				Text:    toolText,
				Thought: false,
			})
			continue
		}

		// Handle tool result blocks - convert to text for now
		if toolResultBlock := contentBlock.OfToolResult; toolResultBlock != nil {
			var resultText strings.Builder
			resultText.WriteString(fmt.Sprintf("Tool result for ID: %s\n", toolResultBlock.ToolUseID))
			for _, content := range toolResultBlock.Content {
				if textContent := content.OfText; textContent != nil {
					resultText.WriteString(textContent.Text)
					resultText.WriteString("\n")
				}
			}
			geminiParts = append(geminiParts, &genai.Part{
				Text:    resultText.String(),
				Thought: false,
			})
			continue
		}
	}

	// Log warning if no parts created (this causes empty input error)
	if len(geminiParts) == 0 {
		fmt.Printf("❌ GEMINI WARNING: No content parts created from input - this will cause empty input error\n")
	}

	return genai.Content{
		Role:  role,
		Parts: geminiParts,
	}, nil
}

// convertFromGeminiToAnthropic converts a Gemini message to Anthropic format
func convertFromGeminiToAnthropic(input genai.Content) (anthropic.MessageParam, error) {
	// Convert role
	var role string
	switch input.Role {
	case "user":
		role = "user"
	case "model":
		role = "assistant"
	default:
		return anthropic.MessageParam{}, fmt.Errorf("unsupported gemini role: %s", input.Role)
	}

	// Convert content blocks
	var anthropicContent []anthropic.ContentBlockParamUnion
	for _, part := range input.Parts {
		if part.Text != "" {
			if part.Thought {
				// This is thinking content
				thinkingContent := anthropic.NewThinkingBlock("", part.Text)
				// Set the Type field for consistency
				if thinkingContent.OfThinking != nil {
					thinkingContent.OfThinking.Type = "thinking"
				}
				anthropicContent = append(anthropicContent, thinkingContent)
			} else {
				// This is regular text content
				textContent := anthropic.NewTextBlock(part.Text)
				// Set the Type field for consistency
				if textContent.OfText != nil {
					textContent.OfText.Type = "text"
				}
				anthropicContent = append(anthropicContent, textContent)
			}
		}
	}

	return anthropic.MessageParam{
		Role:    anthropic.MessageParamRole(role),
		Content: anthropicContent,
	}, nil
}

// publishGeminiStreamEvent publishes Gemini stream events to WebSocket clients after converting to Anthropic format
func (as *AgentService) publishGeminiStreamEvent(chunk *genai.GenerateContentResponse, header *service.EventHeaders, meta *service.EventMetadata) {
	// Convert Gemini stream chunk to Anthropic format (may return multiple events)
	anthropicEvents := convertGeminiStreamChunkToAnthropic(chunk, as.contentBlockStartSent)
	if len(anthropicEvents) == 0 {
		// Skip chunks that don't map to Anthropic format
		return
	}

	// Publish all events in sequence
	for _, anthropicEvent := range anthropicEvents {
		// Convert to WebsocketResponseEventMessage
		wsEvent := ToWebsocketResponseEventMessage(*anthropicEvent, db.ProviderModelGoogle)

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

// convertGeminiStreamChunkToAnthropic converts Gemini stream chunks to Anthropic format
// Returns slice of events to allow injecting synthetic content_block_start events
func convertGeminiStreamChunkToAnthropic(chunk *genai.GenerateContentResponse, state map[int64]bool) []*anthropic.MessageStreamEventUnion {
	var events []*anthropic.MessageStreamEventUnion

	// If this is the first chunk, send message_start
	if len(state) == 0 {
		events = append(events, &anthropic.MessageStreamEventUnion{
			Type: "message_start",
			Message: anthropic.Message{
				ID:    "gemini-message-id",
				Type:  "message",
				Role:  "assistant",
				Model: "",
			},
		})
	}

	// Process candidates
	if len(chunk.Candidates) > 0 {
		candidate := chunk.Candidates[0]
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			for partIndex, part := range candidate.Content.Parts {
				if part.Text != "" {
					index := int64(partIndex)
					
					// Inject synthetic content_block_start if not already sent
					if !state[index] {
						syntheticStart := &anthropic.MessageStreamEventUnion{
							Type:  "content_block_start",
							Index: index,
						}
						events = append(events, syntheticStart)
						state[index] = true // Mark as sent
					}

			// Create delta event for the text chunk
			deltaEvent := &anthropic.MessageStreamEventUnion{
				Type:  "content_block_delta",
				Index: index,
			}
			
			// Check if this is thinking content or regular text
			if part.Thought {
				// This is thinking content
				deltaEvent.Delta.Type = "thinking_delta"
				deltaEvent.Delta.Thinking = part.Text
			} else {
				// This is regular text content
				deltaEvent.Delta.Type = "text_delta"
				deltaEvent.Delta.Text = part.Text
			}
			
			events = append(events, deltaEvent)
				}
			}
		}

		// If candidate has finish reason, send stop events
		if candidate.FinishReason != "" {
			// Send content_block_stop for each active content block
			for index := range state {
				events = append(events, &anthropic.MessageStreamEventUnion{
					Type:  "content_block_stop",
					Index: index,
				})
			}

			// Send message_delta and message_stop
			events = append(events, &anthropic.MessageStreamEventUnion{
				Type: "message_delta",
			})

			events = append(events, &anthropic.MessageStreamEventUnion{
				Type: "message_stop",
			})
		}
	}

	return events
}
