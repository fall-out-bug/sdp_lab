package modelgateway

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockProvider struct {
	id           ProviderID
	capabilities ModelCapabilities
	available    bool
	chatFunc     func(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

func (m *mockProvider) ID() ProviderID {
	return m.id
}

func (m *mockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &ChatResponse{
		ID:      "test-id",
		Model:   req.Model,
		Created: time.Now(),
		Message: Message{Role: RoleAssistant, Content: "test response"},
	}, nil
}

func (m *mockProvider) Capabilities() ModelCapabilities {
	return m.capabilities
}

func (m *mockProvider) IsAvailable(ctx context.Context) bool {
	return m.available
}

func (m *mockProvider) ValidateRequest(req *ChatRequest) error {
	if req.Model == "" {
		return &ProviderError{Code: "MISSING_MODEL", Type: ErrorTypeInvalidInput}
	}
	return nil
}

func TestProviderRegistry(t *testing.T) {
	r := NewProviderRegistry()

	p := &mockProvider{id: "test-provider", available: true}
	r.Register(p)

	got, ok := r.Get("test-provider")
	if !ok {
		t.Fatal("expected provider to be found")
	}
	if got.ID() != "test-provider" {
		t.Errorf("expected test-provider, got %s", got.ID())
	}
}

func TestProviderRegistryList(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(&mockProvider{id: "p1"})
	r.Register(&mockProvider{id: "p2"})

	ids := r.List()
	if len(ids) != 2 {
		t.Errorf("expected 2 providers, got %d", len(ids))
	}
}

func TestProviderRegistryCapabilities(t *testing.T) {
	r := NewProviderRegistry()
	caps := ModelCapabilities{Vision: true, MaxContext: 1000}
	r.Register(&mockProvider{id: "p1", capabilities: caps})

	got, ok := r.Capabilities("p1")
	if !ok {
		t.Fatal("expected capabilities")
	}
	if !got.Vision {
		t.Error("expected vision capability")
	}
}

func TestProviderErrorIsRetryable(t *testing.T) {
	tests := []struct {
		err       *ProviderError
		retryable bool
	}{
		{&ProviderError{Type: ErrorTypeRateLimit, Retryable: true}, true},
		{&ProviderError{Type: ErrorTypeTimeout, Retryable: true}, true},
		{&ProviderError{Type: ErrorTypeAuth, Retryable: false}, false},
		{&ProviderError{Type: ErrorTypeInvalidInput, Retryable: false}, false},
	}

	for _, tt := range tests {
		if tt.err.IsRetryable() != tt.retryable {
			t.Errorf("error type %s: expected retryable=%v", tt.err.Type, tt.retryable)
		}
	}
}

func TestModelRouter(t *testing.T) {
	r := NewProviderRegistry()
	p1 := &mockProvider{id: "openai", available: true}
	p2 := &mockProvider{id: "anthropic", available: true}
	r.Register(p1)
	r.Register(p2)

	router := NewModelRouter(r, RouterConfig{
		DefaultProvider: "openai",
	})

	req := &ChatRequest{Model: "gpt-4o", Messages: []Message{{Role: RoleUser, Content: "hello"}}}
	got, err := router.Route(req)
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if got.ID() != "openai" {
		t.Errorf("expected openai, got %s", got.ID())
	}
}

func TestModelRouterWithRules(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(&mockProvider{id: "openai"})
	r.Register(&mockProvider{id: "anthropic"})

	router := NewModelRouter(r, RouterConfig{
		DefaultProvider: "openai",
		RoutingRules: []RoutingRule{
			{Match: RoutingMatch{ModelPrefix: "claude"}, Target: "anthropic"},
		},
	})

	req := &ChatRequest{Model: "claude-3-opus", Messages: []Message{{Role: RoleUser}}}
	got, err := router.Route(req)
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if got.ID() != "anthropic" {
		t.Errorf("expected anthropic for claude model, got %s", got.ID())
	}
}

func TestModelRouterChat(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(&mockProvider{
		id: "openai",
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{
				ID:      "resp-1",
				Model:   req.Model,
				Message: Message{Role: RoleAssistant, Content: "hello back"},
			}, nil
		},
	})

	router := NewModelRouter(r, RouterConfig{DefaultProvider: "openai"})

	req := &ChatRequest{Model: "gpt-4o", Messages: []Message{{Role: RoleUser, Content: "hello"}}}
	resp, err := router.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	if resp.Message.Content != "hello back" {
		t.Errorf("unexpected response: %s", resp.Message.Content)
	}
}

func TestModelRouterChatWithFallback(t *testing.T) {
	r := NewProviderRegistry()

	callCount := 0
	r.Register(&mockProvider{
		id: "primary",
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			callCount++
			return nil, &ProviderError{Code: "RATE_LIMIT", Type: ErrorTypeRateLimit, Retryable: true}
		},
	})
	r.Register(&mockProvider{
		id: "fallback",
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			callCount++
			return &ChatResponse{ID: "fallback-resp", Model: req.Model}, nil
		},
	})

	router := NewModelRouter(r, RouterConfig{
		DefaultProvider: "primary",
		FallbackOrder:   []ProviderID{"fallback"},
	})

	req := &ChatRequest{Model: "test", Messages: []Message{{Role: RoleUser}}}
	resp, err := router.ChatWithFallback(context.Background(), req)
	if err != nil {
		t.Fatalf("expected fallback to succeed: %v", err)
	}
	if resp.ID != "fallback-resp" {
		t.Errorf("expected fallback response, got %s", resp.ID)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls (primary + fallback), got %d", callCount)
	}
}

func TestModelRouterChatWithFallbackNonRetryable(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(&mockProvider{
		id: "primary",
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return nil, &ProviderError{Code: "AUTH_ERROR", Type: ErrorTypeAuth, Retryable: false}
		},
	})
	r.Register(&mockProvider{
		id: "fallback",
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{ID: "fallback-resp"}, nil
		},
	})

	router := NewModelRouter(r, RouterConfig{
		DefaultProvider: "primary",
		FallbackOrder:   []ProviderID{"fallback"},
	})

	req := &ChatRequest{Model: "test", Messages: []Message{{Role: RoleUser}}}
	_, err := router.ChatWithFallback(context.Background(), req)
	if err == nil {
		t.Error("expected error for non-retryable failure")
	}
}

func TestProviderHealth(t *testing.T) {
	r := NewProviderRegistry()
	r.Register(&mockProvider{id: "p1", available: true})
	r.Register(&mockProvider{id: "p2", available: false})

	health := r.HealthCheck(context.Background())
	if len(health) != 2 {
		t.Errorf("expected 2 health checks, got %d", len(health))
	}

	for _, h := range health {
		if h.ProviderID == "p1" && !h.Available {
			t.Error("p1 should be available")
		}
		if h.ProviderID == "p2" && h.Available {
			t.Error("p2 should not be available")
		}
	}
}

func TestChatRequestValidation(t *testing.T) {
	p := &mockProvider{id: "test"}

	err := p.ValidateRequest(&ChatRequest{Model: "", Messages: nil})
	if err == nil {
		t.Error("expected validation error for empty model")
	}
}

func TestTokenUsage(t *testing.T) {
	usage := &TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		t.Error("token usage mismatch")
	}
}

func TestProviderMetaLatency(t *testing.T) {
	start := time.Now()
	latency := time.Since(start)

	meta := &ProviderMeta{
		ProviderID: "test",
		Latency:    latency,
	}

	if meta.Latency < 0 {
		t.Error("latency should not be negative")
	}
}

func TestIsRetryableError(t *testing.T) {
	retryable := &ProviderError{Retryable: true}
	nonRetryable := &ProviderError{Retryable: false}
	otherErr := errors.New("other")

	if !isRetryableError(retryable) {
		t.Error("retryable provider error should be retryable")
	}
	if isRetryableError(nonRetryable) {
		t.Error("non-retryable provider error should not be retryable")
	}
	if isRetryableError(otherErr) {
		t.Error("non-provider error should not be retryable")
	}
}

func TestProviderFactory(t *testing.T) {
	r := NewProviderRegistry()
	r.RegisterFactory("test", func(config ProviderConfig) (Provider, error) {
		return &mockProvider{id: config.ID}, nil
	})

	p, err := r.CreateProvider(ProviderConfig{ID: "test"})
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if p.ID() != "test" {
		t.Errorf("expected test, got %s", p.ID())
	}
}

func TestProviderFactoryNotFound(t *testing.T) {
	r := NewProviderRegistry()

	_, err := r.CreateProvider(ProviderConfig{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent factory")
	}
}
