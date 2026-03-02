package adapters

import (
	"context"
	"fmt"
	"time"

	"sdp_dev/internal/modelgateway"
)

type AnthropicProvider struct {
	config       modelgateway.ProviderConfig
	capabilities modelgateway.ModelCapabilities
}

func NewAnthropicProvider(config modelgateway.ProviderConfig) (modelgateway.Provider, error) {
	if config.APIKey == "" {
		return nil, &modelgateway.ProviderError{
			Code:       "MISSING_API_KEY",
			Message:    "Anthropic API key is required",
			ProviderID: config.ID,
			Type:       modelgateway.ErrorTypeAuth,
		}
	}

	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	return &AnthropicProvider{
		config: config,
		capabilities: modelgateway.ModelCapabilities{
			Vision:       true,
			FunctionCall: true,
			Streaming:    true,
			MaxContext:   200000,
			SupportedModels: []modelgateway.ModelID{
				"claude-3-opus",
				"claude-3-sonnet",
				"claude-3-haiku",
				"claude-3.5-sonnet",
			},
		},
	}, nil
}

func (p *AnthropicProvider) ID() modelgateway.ProviderID {
	return p.config.ID
}

func (p *AnthropicProvider) Chat(ctx context.Context, req *modelgateway.ChatRequest) (*modelgateway.ChatResponse, error) {
	start := time.Now()

	if err := p.ValidateRequest(req); err != nil {
		return nil, err
	}

	resp := &modelgateway.ChatResponse{
		ID:      fmt.Sprintf("anthropic-%d", time.Now().UnixNano()),
		Model:   req.Model,
		Created: time.Now(),
		Message: modelgateway.Message{
			Role:    modelgateway.RoleAssistant,
			Content: "[MOCK] Anthropic response",
		},
		Usage: &modelgateway.TokenUsage{
			PromptTokens:     120,
			CompletionTokens: 60,
			TotalTokens:      180,
		},
		FinishReason: "end_turn",
		ProviderMeta: &modelgateway.ProviderMeta{
			ProviderID:   p.config.ID,
			ModelName:    string(req.Model),
			Capabilities: p.capabilities,
			Latency:      time.Since(start),
		},
	}

	return resp, nil
}

func (p *AnthropicProvider) Capabilities() modelgateway.ModelCapabilities {
	return p.capabilities
}

func (p *AnthropicProvider) IsAvailable(ctx context.Context) bool {
	return true
}

func (p *AnthropicProvider) ValidateRequest(req *modelgateway.ChatRequest) error {
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
