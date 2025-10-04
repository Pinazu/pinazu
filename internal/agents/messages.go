package agents

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/openai/openai-go"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/db"
	"google.golang.org/genai"
)

// Generic messages wrapper
type Messages[T any] struct {
	Messages []T `json:"messages"`
}

// Provider-specific message types using generics
type (
	AnthropicMessages = Messages[anthropic.MessageParam]
	GoogleMessages    = Messages[genai.Content]
	OpenAIMessages    = Messages[openai.ChatCompletionMessageParamUnion]
	BedrockMessages   = Messages[types.Message]
)

// Generic parse function for multiple messages
func ParseMessages[T any](data []db.JsonRaw) ([]T, error) {
	messages := make([]T, len(data))
	for i, rawMsg := range data {
		var msg T
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message at index %d: %w", i, err)
		}
		messages[i] = msg
	}
	return messages, nil
}

// Generic parse function for a single message
func ParseMessage[T any](data db.JsonRaw) (T, error) {
	var msg T
	if err := json.Unmarshal(data, &msg); err != nil {
		return msg, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return msg, nil
}
