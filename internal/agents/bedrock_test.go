package agents

import (
	"sync"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/pinazu/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBedrockSystemPrompt(t *testing.T) {
	tests := []struct {
		name     string
		spec     *AgentSpecs
		expected []types.SystemContentBlock
	}{
		{
			name: "simple_system_prompt",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.",
			},
			expected: []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{
					Value: "You are a helpful assistant.",
				},
			},
		},
		{
			name: "empty_system_prompt_gets_default",
			spec: &AgentSpecs{
				System: "",
			},
			expected: []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{
					Value: "You are a helpful assistant.",
				},
			},
		},
		{
			name: "multiline_system_prompt",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.\nAlways be polite and respectful.\nProvide clear and concise answers.",
			},
			expected: []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{
					Value: "You are a helpful assistant.\nAlways be polite and respectful.\nProvide clear and concise answers.",
				},
			},
		},
		{
			name: "system_prompt_with_structured_output",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.",
				Model: ModelSpecs{
					ResponseFormat: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{
								"type": "string",
							},
							"age": map[string]any{
								"type": "integer",
							},
						},
						"required": []string{"name", "age"},
					},
				},
			},
			expected: []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{
					Value: "You are a helpful assistant.\n\nYou must respond with valid JSON that matches this exact schema:\n{\"properties\":{\"age\":{\"type\":\"integer\"},\"name\":{\"type\":\"string\"}},\"required\":[\"name\",\"age\"],\"type\":\"object\"}\n\n",
				},
			},
		},
		{
			name: "system_prompt_with_empty_response_format",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.",
				Model: ModelSpecs{
					ResponseFormat: map[string]any{},
				},
			},
			expected: []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{
					Value: "You are a helpful assistant.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBedrockSystemPrompt(tt.spec)
			require.Len(t, result, len(tt.expected))
			for i, expected := range tt.expected {
				if expectedText, ok := expected.(*types.SystemContentBlockMemberText); ok {
					if actualText, ok := result[i].(*types.SystemContentBlockMemberText); ok {
						assert.Equal(t, expectedText.Value, actualText.Value)
					} else {
						t.Errorf("Expected SystemContentBlockMemberText, got different type")
					}
				}
			}
		})
	}
}

func TestInvokeBedrockModel(t *testing.T) {
	log := MockServiceConfigs.CreateLogger()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	mockService, err := NewService(t.Context(), MockServiceConfigs, log, wg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	testCases := []struct {
		name     string
		messages []anthropic.MessageParam
		spec     *AgentSpecs
	}{
		{
			name: "Successful non-streaming request",
			messages: []anthropic.MessageParam{
				{
					Role: "user",
					Content: []anthropic.ContentBlockParamUnion{
						anthropic.NewTextBlock("Hi"),
					},
				},
			},
			spec: &AgentSpecs{
				Model: ModelSpecs{
					Provider:  "bedrock",
					ModelID:   "apac.amazon.nova-micro-v1:0",
					MaxTokens: 1000,
					Stream:    false,
				},
				System: "You are a helpful assistant.",
			},
		},
		{
			name: "Successful streaming request",
			messages: []anthropic.MessageParam{
				{
					Role: "user",
					Content: []anthropic.ContentBlockParamUnion{
						anthropic.NewTextBlock("Hi"),
					},
				},
			},
			spec: &AgentSpecs{
				Model: ModelSpecs{
					Provider:  "bedrock",
					ModelID:   "apac.amazon.nova-micro-v1:0",
					MaxTokens: 1000,
					Stream:    true,
				},
				System: "You are a helpful assistant.",
			},
		},
		{
			name: "Successful request with thinking enabled",
			messages: []anthropic.MessageParam{
				{
					Role: "user",
					Content: []anthropic.ContentBlockParamUnion{
						anthropic.NewTextBlock("What is 2+2? Think step by step."),
					},
				},
			},
			spec: &AgentSpecs{
				Model: ModelSpecs{
					Provider:  "bedrock",
					ModelID:   "apac.anthropic.claude-3-7-sonnet-20250219-v1:0",
					MaxTokens: 4000,
					Stream:    true,
					Thinking: ThinkingSpecs{
						Enabled:     true,
						BudgetToken: 1024,
					},
				},
				System: "You are a helpful assistant.",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg, stop, err := mockService.handleBedrockRequest(tc.messages, tc.spec, &service.EventHeaders{}, &service.EventMetadata{})

			assert.Nil(t, err, "Error should be nil for successful requests")

			// Successful case - should return a message with content
			require.NotNil(t, msg, "Response message should not be nil for successful requests")
			assert.Equal(t, "assistant", string(msg.Role), "Response should be from assistant")
			assert.Equal(t, "end_turn", stop, "Stop reason should be 'end_turn' for successful requests")
			require.NotEmpty(t, msg.Content, "Message should have content")

			// Assert content is text content and not empty
			for _, contentBlock := range msg.Content {
				if textBlock := contentBlock.OfText; textBlock != nil {
					assert.NotEmpty(t, textBlock.Text, "Text content should not be empty")
				}
			}
		})
	}
}

func TestBedrockStructuredOutputPrefillMessageAdvanced(t *testing.T) {
	tests := []struct {
		name          string
		spec          *AgentSpecs
		inputMessages []anthropic.MessageParam
		expectPrefill bool
		expectedText  string
	}{
		{
			name: "structured_output_adds_prefill",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.",
				Model: ModelSpecs{
					Provider:  "bedrock",
					ModelID:   "anthropic.claude-3-haiku-20240307-v1:0",
					MaxTokens: 1000,
					Stream:    false,
					ResponseFormat: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{
								"type": "string",
							},
						},
						"required": []string{"name"},
					},
				},
			},
			inputMessages: []anthropic.MessageParam{
				{
					Content: []anthropic.ContentBlockParamUnion{
						anthropic.NewTextBlock("Hello"),
					},
					Role: anthropic.MessageParamRoleUser,
				},
			},
			expectPrefill: true,
			expectedText:  "{",
		},
		{
			name: "no_response_format_no_prefill",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.",
				Model: ModelSpecs{
					Provider:       "bedrock",
					ModelID:        "anthropic.claude-3-haiku-20240307-v1:0",
					MaxTokens:      1000,
					Stream:         false,
					ResponseFormat: nil,
				},
			},
			inputMessages: []anthropic.MessageParam{
				{
					Content: []anthropic.ContentBlockParamUnion{
						anthropic.NewTextBlock("Hello"),
					},
					Role: anthropic.MessageParamRoleUser,
				},
			},
			expectPrefill: false,
		},
		{
			name: "empty_response_format_no_prefill",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.",
				Model: ModelSpecs{
					Provider:       "bedrock",
					ModelID:        "anthropic.claude-3-haiku-20240307-v1:0",
					MaxTokens:      1000,
					Stream:         false,
					ResponseFormat: map[string]any{},
				},
			},
			inputMessages: []anthropic.MessageParam{
				{
					Content: []anthropic.ContentBlockParamUnion{
						anthropic.NewTextBlock("Hello"),
					},
					Role: anthropic.MessageParamRoleUser,
				},
			},
			expectPrefill: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the same logic as in handleBedrockRequest for message handling
			messages := tt.inputMessages

			// Apply the same logic as in handleBedrockRequest for adding prefill
			if len(tt.spec.Model.ResponseFormat) > 0 {
				prefillMsg := anthropic.NewAssistantMessage(anthropic.NewTextBlock("{"))
				messages = append(messages, prefillMsg)
			}

			if tt.expectPrefill {
				// Verify that a prefill message was added
				assert.Greater(t, len(messages), len(tt.inputMessages), "Should have added prefill message")

				// Check the last message is assistant role with prefill content
				lastMsg := messages[len(messages)-1]
				assert.Equal(t, "assistant", string(lastMsg.Role), "Last message should be from assistant")

				// Check the content of the prefill message
				require.NotEmpty(t, lastMsg.Content, "Prefill message should have content")

				// Verify it's a text block with "{"
				textBlock := lastMsg.Content[0].OfText
				require.NotNil(t, textBlock, "Prefill message should be a text block")
				assert.Equal(t, tt.expectedText, textBlock.Text, "Prefill message should contain '{'")
			} else {
				// Verify no prefill message was added
				assert.Equal(t, len(tt.inputMessages), len(messages), "Should not have added prefill message")
			}
		})
	}
}
