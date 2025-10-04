package agents

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
)

// handleOpenAIRequest handles requests for OpenAI models
func (as *AgentService) handleOpenAIRequest(m []openai.ChatCompletionMessageParamUnion, spec *AgentSpecs, header *service.EventHeaders) (*openai.ChatCompletionMessageParamUnion, error) {
	params := openai.ChatCompletionNewParams{
		Model:               spec.Model.ModelID,
		Messages:            m,
		MaxCompletionTokens: param.NewOpt(spec.Model.MaxTokens),
		Temperature:         param.NewOpt(spec.Model.Temperature),
		TopP:                param.NewOpt(spec.Model.TopP),
	}

	as.log.Debug("OpenAI request", "params", params)

	return nil, nil
}
