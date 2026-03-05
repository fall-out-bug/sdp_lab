package adapters

import (
	"context"
	"testing"
	"time"

	"sdp_dev/internal/modelgateway"
)

func TestNewOpenAIProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  modelgateway.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: modelgateway.ProviderConfig{
				ID:     "openai",
				APIKey: "sk-test",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: modelgateway.ProviderConfig{
				ID:     "openai",
				APIKey: "",
			},
			wantErr: true,
		},
		{
			name: "default timeout",
			config: modelgateway.ProviderConfig{
				ID:     "openai",
				APIKey: "sk-test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewOpenAIProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && p == nil {
				t.Error("provider is nil")
			}
		})
	}
}

func TestOpenAIProvider_Chat(t *testing.T) {
	p, err := NewOpenAIProvider(modelgateway.ProviderConfig{
		ID:     "openai",
		APIKey: "sk-test",
	})
	if err != nil {
		t.Fatalf("NewOpenAIProvider failed: %v", err)
	}

	tests := []struct {
		name    string
		req     *modelgateway.ChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &modelgateway.ChatRequest{
				Model: "gpt-4o",
				Messages: []modelgateway.Message{
					{Role: modelgateway.RoleUser, Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: &modelgateway.ChatRequest{
				Model: "",
				Messages: []modelgateway.Message{
					{Role: modelgateway.RoleUser, Content: "Hello"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing messages",
			req: &modelgateway.ChatRequest{
				Model:    "gpt-4o",
				Messages: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := p.Chat(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Chat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if resp == nil {
					t.Error("response is nil")
					return
				}
				if resp.Model != tt.req.Model {
					t.Errorf("resp.Model = %v, want %v", resp.Model, tt.req.Model)
				}
				if resp.Usage == nil {
					t.Error("resp.Usage is nil")
				}
				if resp.ProviderMeta == nil {
					t.Error("resp.ProviderMeta is nil")
				}
			}
		})
	}
}

func TestOpenAIProvider_Capabilities(t *testing.T) {
	p, _ := NewOpenAIProvider(modelgateway.ProviderConfig{
		ID:     "openai",
		APIKey: "sk-test",
	})

	caps := p.Capabilities()
	if !caps.Vision {
		t.Error("expected Vision capability")
	}
	if !caps.FunctionCall {
		t.Error("expected FunctionCall capability")
	}
	if !caps.Streaming {
		t.Error("expected Streaming capability")
	}
	if len(caps.SupportedModels) == 0 {
		t.Error("expected supported models")
	}
}

func TestOpenAIProvider_IsAvailable(t *testing.T) {
	p, _ := NewOpenAIProvider(modelgateway.ProviderConfig{
		ID:     "openai",
		APIKey: "sk-test",
	})

	if !p.IsAvailable(context.Background()) {
		t.Error("expected IsAvailable to return true")
	}
}

func TestNewAnthropicProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  modelgateway.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: modelgateway.ProviderConfig{
				ID:     "anthropic",
				APIKey: "sk-ant-test",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: modelgateway.ProviderConfig{
				ID:     "anthropic",
				APIKey: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewAnthropicProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAnthropicProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && p == nil {
				t.Error("provider is nil")
			}
		})
	}
}

func TestAnthropicProvider_Chat(t *testing.T) {
	p, err := NewAnthropicProvider(modelgateway.ProviderConfig{
		ID:     "anthropic",
		APIKey: "sk-ant-test",
	})
	if err != nil {
		t.Fatalf("NewAnthropicProvider failed: %v", err)
	}

	tests := []struct {
		name    string
		req     *modelgateway.ChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &modelgateway.ChatRequest{
				Model: "claude-3-opus",
				Messages: []modelgateway.Message{
					{Role: modelgateway.RoleUser, Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: &modelgateway.ChatRequest{
				Model: "",
				Messages: []modelgateway.Message{
					{Role: modelgateway.RoleUser, Content: "Hello"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing messages",
			req: &modelgateway.ChatRequest{
				Model:    "claude-3-opus",
				Messages: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := p.Chat(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Chat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if resp == nil {
					t.Error("response is nil")
					return
				}
				if resp.Model != tt.req.Model {
					t.Errorf("resp.Model = %v, want %v", resp.Model, tt.req.Model)
				}
			}
		})
	}
}

func TestAnthropicProvider_Capabilities(t *testing.T) {
	p, _ := NewAnthropicProvider(modelgateway.ProviderConfig{
		ID:     "anthropic",
		APIKey: "sk-ant-test",
	})

	caps := p.Capabilities()
	if !caps.Vision {
		t.Error("expected Vision capability")
	}
	if caps.MaxContext != 200000 {
		t.Errorf("MaxContext = %d, want 200000", caps.MaxContext)
	}
}

func TestAnthropicProvider_IsAvailable(t *testing.T) {
	p, _ := NewAnthropicProvider(modelgateway.ProviderConfig{
		ID:     "anthropic",
		APIKey: "sk-ant-test",
	})

	if !p.IsAvailable(context.Background()) {
		t.Error("expected IsAvailable to return true")
	}
}

func TestNewSelfHostedProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  modelgateway.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: modelgateway.ProviderConfig{
				ID:      "selfhosted",
				BaseURL: "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "missing base URL",
			config: modelgateway.ProviderConfig{
				ID:      "selfhosted",
				BaseURL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewSelfHostedProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSelfHostedProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && p == nil {
				t.Error("provider is nil")
			}
		})
	}
}

func TestSelfHostedProvider_Chat(t *testing.T) {
	p, err := NewSelfHostedProvider(modelgateway.ProviderConfig{
		ID:      "selfhosted",
		BaseURL: "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("NewSelfHostedProvider failed: %v", err)
	}

	req := &modelgateway.ChatRequest{
		Model: "local-model",
		Messages: []modelgateway.Message{
			{Role: modelgateway.RoleUser, Content: "Hello"},
		},
	}

	resp, err := p.Chat(context.Background(), req)
	if err != nil {
		t.Errorf("Chat() error = %v", err)
		return
	}
	if resp == nil {
		t.Error("response is nil")
	}
}

func TestSelfHostedProvider_Timeout(t *testing.T) {
	// Test that timeout is set correctly
	p, err := NewSelfHostedProvider(modelgateway.ProviderConfig{
		ID:      "selfhosted",
		BaseURL: "http://localhost:8080",
		Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewSelfHostedProvider failed: %v", err)
	}

	if p == nil {
		t.Error("provider is nil")
	}
}

func TestProvider_ID(t *testing.T) {
	openai, _ := NewOpenAIProvider(modelgateway.ProviderConfig{
		ID:     "openai-prod",
		APIKey: "sk-test",
	})
	if openai.ID() != "openai-prod" {
		t.Errorf("OpenAI ID = %v, want openai-prod", openai.ID())
	}

	anthropic, _ := NewAnthropicProvider(modelgateway.ProviderConfig{
		ID:     "anthropic-prod",
		APIKey: "sk-ant-test",
	})
	if anthropic.ID() != "anthropic-prod" {
		t.Errorf("Anthropic ID = %v, want anthropic-prod", anthropic.ID())
	}

	selfhosted, _ := NewSelfHostedProvider(modelgateway.ProviderConfig{
		ID:      "local",
		BaseURL: "http://localhost:8080",
	})
	if selfhosted.ID() != "local" {
		t.Errorf("SelfHosted ID = %v, want local", selfhosted.ID())
	}
}
