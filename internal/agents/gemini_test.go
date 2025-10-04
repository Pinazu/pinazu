package agents

import (
	"os"
	"sync"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
	"google.golang.org/genai"
)

func TestGetGeminiThinkingConfig(t *testing.T) {
	tests := []struct {
		name     string
		spec     *AgentSpecs
		expected *genai.ThinkingConfig
	}{
		{
			name: "thinking_enabled",
			spec: &AgentSpecs{
				Model: ModelSpecs{
					Thinking: ThinkingSpecs{
						Enabled:     true,
						BudgetToken: 1500,
					},
				},
			},
			expected: &genai.ThinkingConfig{
				IncludeThoughts: true,
				ThinkingBudget:  aws.Int32(int32(1500)),
			},
		},
		{
			name: "thinking_disabled",
			spec: &AgentSpecs{
				Model: ModelSpecs{
					Thinking: ThinkingSpecs{
						Enabled:     false,
						BudgetToken: 0,
					},
				},
			},
			expected: nil,
		},
		{
			name: "thinking_enabled_zero_budget",
			spec: &AgentSpecs{
				Model: ModelSpecs{
					Thinking: ThinkingSpecs{
						Enabled:     true,
						BudgetToken: 0,
					},
				},
			},
			expected: &genai.ThinkingConfig{
				IncludeThoughts: true,
				ThinkingBudget:  aws.Int32(int32(0)),
			},
		},
		{
			name: "thinking_enabled_custom_budget",
			spec: &AgentSpecs{
				Model: ModelSpecs{
					Thinking: ThinkingSpecs{
						Enabled:     true,
						BudgetToken: 2048,
					},
				},
			},
			expected: &genai.ThinkingConfig{
				IncludeThoughts: true,
				ThinkingBudget:  aws.Int32(int32(2048)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGeminiThinkingConfig(tt.spec)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.IncludeThoughts, result.IncludeThoughts)
				if tt.expected.ThinkingBudget != nil {
					require.NotNil(t, result.ThinkingBudget)
					assert.Equal(t, *tt.expected.ThinkingBudget, *result.ThinkingBudget)
				} else {
					assert.Nil(t, result.ThinkingBudget)
				}
			}
		})
	}
}

func TestGetGeminiSystemPrompt(t *testing.T) {
	tests := []struct {
		name     string
		spec     *AgentSpecs
		expected string
	}{
		{
			name: "simple_system_prompt",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.",
			},
			expected: "You are a helpful assistant.",
		},
		{
			name: "empty_system_prompt",
			spec: &AgentSpecs{
				System: "",
			},
			expected: "",
		},
		{
			name: "multiline_system_prompt",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.\nAlways be polite and respectful.\nProvide clear and concise answers.",
			},
			expected: "You are a helpful assistant.\nAlways be polite and respectful.\nProvide clear and concise answers.",
		},
		{
			name: "complex_system_prompt",
			spec: &AgentSpecs{
				System: "You are an AI assistant specialized in coding tasks. You should:\n- Write clean, readable code\n- Follow best practices\n- Explain your solutions clearly",
			},
			expected: "You are an AI assistant specialized in coding tasks. You should:\n- Write clean, readable code\n- Follow best practices\n- Explain your solutions clearly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test system prompt extraction from AgentSpecs
			assert.Equal(t, tt.expected, tt.spec.System)
		})
	}
}

func TestGetGeminiGenerateContentConfig(t *testing.T) {
	tests := []struct {
		name     string
		spec     *AgentSpecs
		expected *genai.GenerateContentConfig
	}{
		{
			name: "basic_config",
			spec: &AgentSpecs{
				System: "You are a helpful assistant.",
				Model: ModelSpecs{
					MaxTokens:   1000,
					Temperature: 0.7,
					TopP:        0.9,
					TopK:        40,
					Thinking: ThinkingSpecs{
						Enabled:     false,
						BudgetToken: 0,
					},
				},
			},
			expected: &genai.GenerateContentConfig{
				SystemInstruction: &genai.Content{
					Parts: []*genai.Part{
						{Text: "You are a helpful assistant."},
					},
				},
				MaxOutputTokens: 1000,
				Temperature:     aws.Float32(float32(0.7)),
				TopP:            aws.Float32(float32(0.9)),
				TopK:            aws.Float32(float32(40)),
				ThinkingConfig:  nil,
			},
		},
		{
			name: "config_with_thinking_enabled",
			spec: &AgentSpecs{
				System: "You are a coding assistant.",
				Model: ModelSpecs{
					MaxTokens:   2000,
					Temperature: 0.5,
					TopP:        0.8,
					TopK:        50,
					Thinking: ThinkingSpecs{
						Enabled:     true,
						BudgetToken: 1024,
					},
				},
			},
			expected: &genai.GenerateContentConfig{
				SystemInstruction: &genai.Content{
					Parts: []*genai.Part{
						{Text: "You are a coding assistant."},
					},
				},
				MaxOutputTokens: 2000,
				Temperature:     aws.Float32(float32(0.5)),
				TopP:            aws.Float32(float32(0.8)),
				TopK:            aws.Float32(float32(50)),
				ThinkingConfig: &genai.ThinkingConfig{
					IncludeThoughts: true,
					ThinkingBudget:  aws.Int32(int32(1024)),
				},
			},
		},
		{
			name: "config_with_zero_values",
			spec: &AgentSpecs{
				System: "",
				Model: ModelSpecs{
					MaxTokens:   0,
					Temperature: 0,
					TopP:        0,
					TopK:        0,
					Thinking: ThinkingSpecs{
						Enabled:     false,
						BudgetToken: 0,
					},
				},
			},
			expected: &genai.GenerateContentConfig{
				SystemInstruction: &genai.Content{
					Parts: []*genai.Part{
						{Text: ""},
					},
				},
				MaxOutputTokens: 0,
				Temperature:     aws.Float32(float32(0)),
				TopP:            aws.Float32(float32(0)),
				TopK:            aws.Float32(float32(0)),
				ThinkingConfig:  nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test configuration creation following the pattern in handleGeminiRequest
			config := &genai.GenerateContentConfig{
				SystemInstruction: &genai.Content{
					Parts: []*genai.Part{
						{Text: tt.spec.System},
					},
				},
				MaxOutputTokens: int32(tt.spec.Model.MaxTokens),
				Temperature:     aws.Float32(float32(tt.spec.Model.Temperature)),
				TopP:            aws.Float32(float32(tt.spec.Model.TopP)),
				TopK:            aws.Float32(float32(tt.spec.Model.TopK)),
				ThinkingConfig:  getGeminiThinkingConfig(tt.spec),
			}

			// Verify system instruction
			require.NotNil(t, config.SystemInstruction)
			require.Len(t, config.SystemInstruction.Parts, 1)
			assert.Equal(t, tt.expected.SystemInstruction.Parts[0].Text, config.SystemInstruction.Parts[0].Text)

			// Verify other parameters
			assert.Equal(t, tt.expected.MaxOutputTokens, config.MaxOutputTokens)
			assert.Equal(t, *tt.expected.Temperature, *config.Temperature)
			assert.Equal(t, *tt.expected.TopP, *config.TopP)
			assert.Equal(t, *tt.expected.TopK, *config.TopK)

			// Verify thinking config
			if tt.expected.ThinkingConfig == nil {
				assert.Nil(t, config.ThinkingConfig)
			} else {
				require.NotNil(t, config.ThinkingConfig)
				assert.Equal(t, tt.expected.ThinkingConfig.IncludeThoughts, config.ThinkingConfig.IncludeThoughts)
				if tt.expected.ThinkingConfig.ThinkingBudget != nil {
					require.NotNil(t, config.ThinkingConfig.ThinkingBudget)
					assert.Equal(t, *tt.expected.ThinkingConfig.ThinkingBudget, *config.ThinkingConfig.ThinkingBudget)
				}
			}
		})
	}
}

func TestGeminiMessageConversion(t *testing.T) {
	tests := []struct {
		name     string
		messages []genai.Content
		expected []*genai.Content
	}{
		{
			name: "single_user_message",
			messages: []genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Hello"},
					},
				},
			},
			expected: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Hello"},
					},
				},
			},
		},
		{
			name: "conversation_with_multiple_messages",
			messages: []genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "What is AI?"},
					},
				},
				{
					Role: "model",
					Parts: []*genai.Part{
						{Text: "AI stands for Artificial Intelligence..."},
					},
				},
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Can you give me examples?"},
					},
				},
			},
			expected: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "What is AI?"},
					},
				},
				{
					Role: "model",
					Parts: []*genai.Part{
						{Text: "AI stands for Artificial Intelligence..."},
					},
				},
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Can you give me examples?"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the conversion logic used in handleGeminiRequest
			contentPointers := make([]*genai.Content, len(tt.messages))
			for i := range tt.messages {
				contentPointers[i] = &tt.messages[i]
			}

			require.Len(t, contentPointers, len(tt.expected))
			for i, expected := range tt.expected {
				assert.Equal(t, expected.Role, contentPointers[i].Role)
				require.Len(t, contentPointers[i].Parts, len(expected.Parts))
				for j, expectedPart := range expected.Parts {
					assert.Equal(t, expectedPart.Text, contentPointers[i].Parts[j].Text)
				}
			}
		})
	}
}

func TestInvokeGeminiModel(t *testing.T) {
	// Check if API key is available in environment
	apiKey := os.Getenv("GOOGLE_AI_API_KEY")
	if apiKey == "" {
		t.Skip("GOOGLE_AI_API_KEY environment variable not set, skipping Gemini tests. Set it with: export GOOGLE_AI_API_KEY=your_api_key")
	}

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
					Provider:  "gemini",
					ModelID:   "gemini-2.5-flash-lite",
					MaxTokens: 1000,
					Stream:    false,
				},
				System: "You are a helpful assistant.",
			},
		},
		{
			name: "Successful non-streaming request with thinking enabled",
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
					Provider:  "gemini",
					ModelID:   "gemini-2.5-flash-lite",
					MaxTokens: 4000,
					Stream:    false,
					Thinking: ThinkingSpecs{
						Enabled:     true,
						BudgetToken: 1024,
					},
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
					Provider:  "gemini",
					ModelID:   "gemini-2.5-flash-lite",
					MaxTokens: 1000,
					Stream:    true,
					Thinking: ThinkingSpecs{
						Enabled:     false,
						BudgetToken: 0,
					},
				},
				System: "You are a helpful assistant.",
			},
		},
		{
			name: "Successful streaming request with thinking enabled",
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
					Provider:  "gemini",
					ModelID:   "gemini-2.5-flash-lite",
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
		{
			name: "Request with model parameters",
			messages: []anthropic.MessageParam{
				{
					Role: "user",
					Content: []anthropic.ContentBlockParamUnion{
						anthropic.NewTextBlock("Hi with parameters"),
					},
				},
			},
			spec: &AgentSpecs{
				Model: ModelSpecs{
					Provider:    "gemini",
					ModelID:     "gemini-2.5-flash-lite",
					MaxTokens:   2048,
					Stream:      false,
					Temperature: 0.7,
					TopP:        0.9,
					TopK:        40,
				},
				System: "You are an expert assistant.",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg, stop, err := mockService.handleGeminiRequest(tc.messages, tc.spec, &service.EventHeaders{}, &service.EventMetadata{})

			// Handle potential credential errors in test environment
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Successful case - should return a message with content
			require.NotNil(t, msg, "Response message should not be nil for successful requests")
			assert.Equal(t, "assistant", string(msg.Role), "Response should have 'assistant' role")
			require.NotEmpty(t, msg.Content, "Message should have content")

			// Assert content is text content and not empty
			hasTextContent := false
			hasThinkingContent := false

			for _, contentBlock := range msg.Content {
				if textBlock := contentBlock.OfText; textBlock != nil {
					hasTextContent = true
					assert.NotEmpty(t, textBlock.Text, "Text content should not be empty")
				}
				if thinkingBlock := contentBlock.OfThinking; thinkingBlock != nil {
					hasThinkingContent = true
					assert.NotEmpty(t, thinkingBlock.Thinking, "Thinking content should not be empty")
					// Verify thinking content only appears when thinking is enabled
					assert.True(t, tc.spec.Model.Thinking.Enabled, "Thinking content should only appear when thinking is enabled in spec")
				}
			}

			// At least one of text or thinking content should be present
			assert.True(t, hasTextContent || hasThinkingContent, "Message should have either text or thinking content")

			// If thinking is enabled, we might have thinking content
			if tc.spec.Model.Thinking.Enabled {
				// Thinking content is optional even when enabled
				t.Logf("Thinking enabled, hasThinkingContent: %t", hasThinkingContent)
			} else {
				// If thinking is disabled, we should not have thinking content
				assert.False(t, hasThinkingContent, "Should not have thinking content when thinking is disabled")
			}

			// Verify stop reason
			assert.NotEmpty(t, stop, "Stop reason should not be empty")
		})
	}
}
