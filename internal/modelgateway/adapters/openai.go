package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/modelgateway"
)

type OpenAIProvider struct {
	config       modelgateway.ProviderConfig
	capabilities modelgateway.ModelCapabilities
}

func NewOpenAIProvider(config modelgateway.ProviderConfig) (modelgateway.Provider, error) {
	if config.APIKey == "" {
		return nil, &modelgateway.ProviderError{
			Code:       "MISSING_API_KEY",
			Message:    "OpenAI API key is required",
			ProviderID: config.ID,
			Type:       modelgateway.ErrorTypeAuth,
		}
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &OpenAIProvider{
		config: config,
		capabilities: modelgateway.ModelCapabilities{
			Vision:       true,
			FunctionCall: true,
			Streaming:    true,
			MaxContext:   128000,
			SupportedModels: []modelgateway.ModelID{
				"gpt-4o",
				"gpt-4o-mini",
				"gpt-4-turbo",
				"gpt-3.5-turbo",
			},
		},
	}, nil
}

func (p *OpenAIProvider) ID() modelgateway.ProviderID {
	return p.config.ID
}

func (p *OpenAIProvider) Chat(ctx context.Context, req *modelgateway.ChatRequest) (*modelgateway.ChatResponse, error) {
	start := time.Now()

	if err := p.ValidateRequest(req); err != nil {
		return nil, err
	}

	resp := &modelgateway.ChatResponse{
		ID:      fmt.Sprintf("openai-%d", time.Now().UnixNano()),
		Model:   req.Model,
		Created: time.Now(),
		Message: modelgateway.Message{
			Role:    modelgateway.RoleAssistant,
			Content: "[MOCK] OpenAI response",
		},
		Usage: &modelgateway.TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		FinishReason: "stop",
		ProviderMeta: &modelgateway.ProviderMeta{
			ProviderID:   p.config.ID,
			ModelName:    string(req.Model),
			Capabilities: p.capabilities,
			Latency:      time.Since(start),
		},
	}

	return resp, nil
}

func (p *OpenAIProvider) Capabilities() modelgateway.ModelCapabilities {
	return p.capabilities
}

func (p *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	return true
}

func (p *OpenAIProvider) ValidateRequest(req *modelgateway.ChatRequest) error {
	if req.Model == "" {
		return &modelgateway.ProviderError{
			Code:       "MISSING_MODEL",
			Message:    "model is required",
			ProviderID: p.config.ID,
			Type:       modelgateway.ErrorTypeInvalidInput,
		}
	}
	if len(req.Messages) == 0 {
		return &modelgateway.ProviderError{
			Code:       "MISSING_MESSAGES",
			Message:    "messages are required",
			ProviderID: p.config.ID,
			Type:       modelgateway.ErrorTypeInvalidInput,
		}
	}
	return nil
}
