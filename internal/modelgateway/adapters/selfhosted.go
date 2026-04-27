package adapters

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/modelgateway"
)

type SelfHostedProvider struct {
	config       modelgateway.ProviderConfig
	capabilities modelgateway.ModelCapabilities
	httpClient   *http.Client
}

func NewSelfHostedProvider(config modelgateway.ProviderConfig) (modelgateway.Provider, error) {
	if config.BaseURL == "" {
		return nil, &modelgateway.ProviderError{
			Code:       "MISSING_BASE_URL",
			Message:    "base URL is required for self-hosted provider",
			ProviderID: config.ID,
			Type:       modelgateway.ErrorTypeInvalidInput,
		}
	}

	if config.Timeout == 0 {
		config.Timeout = 120 * time.Second
	}

	return &SelfHostedProvider{
		config: config,
		capabilities: modelgateway.ModelCapabilities{
			Vision:       false,
			FunctionCall: false,
			Streaming:    true,
			MaxContext:   32768,
			SupportedModels: []modelgateway.ModelID{
				"llama-3",
				"mistral-7b",
				"codellama-34b",
			},
		},
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

func (p *SelfHostedProvider) ID() modelgateway.ProviderID {
	return p.config.ID
}

func (p *SelfHostedProvider) Chat(ctx context.Context, req *modelgateway.ChatRequest) (*modelgateway.ChatResponse, error) {
	start := time.Now()

	if err := p.ValidateRequest(req); err != nil {
		return nil, err
	}

	resp := &modelgateway.ChatResponse{
		ID:      fmt.Sprintf("selfhosted-%d", time.Now().UnixNano()),
		Model:   req.Model,
		Created: time.Now(),
		Message: modelgateway.Message{
			Role:    modelgateway.RoleAssistant,
			Content: "[MOCK] Self-hosted model response",
		},
		Usage: &modelgateway.TokenUsage{
			PromptTokens:     80,
			CompletionTokens: 40,
			TotalTokens:      120,
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

func (p *SelfHostedProvider) Capabilities() modelgateway.ModelCapabilities {
	return p.capabilities
}

func (p *SelfHostedProvider) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == 200
}

func (p *SelfHostedProvider) ValidateRequest(req *modelgateway.ChatRequest) error {
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
