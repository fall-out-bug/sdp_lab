package strataudit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// ModelRuntime is the provider-neutral StratAudit runtime contract.
// Host harnesses can inject their own implementation; the CLI resolves one from config.
type ModelRuntime interface {
	Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error)
	Embed(ctx context.Context, texts []string, model string) ([][]float32, error)
}

// FunctionalRuntime is a lightweight adapter for tests and harness-side injection.
type FunctionalRuntime struct {
	ChatFunc  func(ctx context.Context, req LLMRequest) (*LLMResponse, error)
	EmbedFunc func(ctx context.Context, texts []string, model string) ([][]float32, error)
}

func (r FunctionalRuntime) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	if r.ChatFunc == nil {
		return nil, fmt.Errorf("functional runtime: ChatFunc is nil")
	}
	return r.ChatFunc(ctx, req)
}

func (r FunctionalRuntime) Embed(ctx context.Context, texts []string, model string) ([][]float32, error) {
	if r.EmbedFunc == nil {
		return nil, fmt.Errorf("functional runtime: EmbedFunc is nil")
	}
	return r.EmbedFunc(ctx, texts, model)
}

func normalizeRuntimeProvider(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	p = strings.ReplaceAll(p, "-", "_")
	switch p {
	case "", "openrouter":
		return "openrouter"
	case "openai_compatible", "compatible":
		return "openai_compatible"
	case "host", "injected":
		return "host"
	default:
		return p
	}
}

// ResolveRuntime creates the CLI runtime from config.
// Providers that depend on host-owned models must be injected by the caller.
func (c *Config) ResolveRuntime() (ModelRuntime, error) {
	if c == nil {
		return nil, fmt.Errorf("resolve runtime: nil config")
	}

	provider := normalizeRuntimeProvider(c.Runtime.Provider)
	switch provider {
	case "host":
		return nil, fmt.Errorf("runtime.provider=%q requires injected runtime; sdp-strataudit CLI cannot create a host-native runtime", c.Runtime.Provider)
	case "openrouter", "openai_compatible":
	default:
		return nil, fmt.Errorf("unsupported runtime.provider=%q", c.Runtime.Provider)
	}

	if strings.TrimSpace(c.Runtime.BaseURL) == "" {
		return nil, fmt.Errorf("runtime.base_url must be set for provider %q", provider)
	}
	if strings.TrimSpace(c.Runtime.APIKeyEnv) == "" {
		return nil, fmt.Errorf("runtime.api_key_env must be set for provider %q", provider)
	}

	apiKey := os.Getenv(c.Runtime.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("runtime API key env %s not set", c.Runtime.APIKeyEnv)
	}

	client := NewLLMClient(apiKey, c.Runtime.BaseURL)
	client.SetRateLimit(c.LLM.RequestsPerMin)
	client.SetRetryConfig(c.LLM.MaxRetries, time.Duration(c.LLM.RetryBaseDelayMs)*time.Millisecond)
	return client, nil
}
