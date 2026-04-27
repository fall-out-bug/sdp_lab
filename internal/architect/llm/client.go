package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
	"github.com/fall-out-bug/sdp_lab/internal/llmclient"
)

// Provider represents an LLM provider (local or cloud).
type Provider string

const (
	ProviderLocal    Provider = "local"    // Ollama, LM Studio
	ProviderOpenAI   Provider = "openai"   // OpenAI API
	ProviderAnthropic Provider = "anthropic" // Anthropic Claude
	ProviderOpenRouter Provider = "openrouter" // OpenRouter aggregator
)

// Config holds LLM client configuration.
type Config struct {
	// Provider is the LLM provider to use.
	Provider Provider

	// BaseURL is the API endpoint for cloud providers or local models.
	// For local: http://localhost:11434/v1 (Ollama)
	// For OpenRouter: https://openrouter.ai/api/v1
	BaseURL string

	// APIKey is the authentication key (ignored for local providers).
	APIKey string

	// Model is the model identifier (e.g., "openai/gpt-4o-mini", "llama3.2").
	Model string

	// MaxTokens caps the response length. Default: 4096.
	MaxTokens int

	// Timeout is the per-request timeout. Default: 120s.
	Timeout time.Duration

	// Retry controls exponential backoff retries.
	Retry architect.RetryConfig

	// CircuitBreaker thresholds.
	FailureThreshold int
	CooldownPeriod   time.Duration
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Provider:         ProviderOpenRouter,
		BaseURL:          "https://openrouter.ai/api/v1",
		APIKey:           "",
		Model:            "openai/gpt-4o-mini",
		MaxTokens:        4096,
		Timeout:          120 * time.Second,
		Retry:            architect.NewRetryConfig(),
		FailureThreshold: 5,
		CooldownPeriod:   30 * time.Second,
	}
}

// Client is the LLM client abstraction supporting multiple providers.
type Client struct {
	cfg    Config
	inner  *llmclient.Client
	cb     *architect.CircuitBreaker
	filter *architect.SecurityFilter
}

// NewClient creates a new LLM client with the given configuration.
func NewClient(cfg Config, filter *architect.SecurityFilter) *Client {
	inner := llmclient.New(cfg.APIKey, cfg.BaseURL)
	return &Client{
		cfg:    cfg,
		inner:  inner,
		cb:     architect.NewCircuitBreaker(),
		filter: filter,
	}
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64 // 0.0 = deterministic, 1.0 = creative
	MaxTokens    int
}

// ChatResponse represents the response from a chat completion.
type ChatResponse struct {
	Content      string
	Usage        TokenUsage
	Model        string
	FinishReason string
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Complete sends a chat completion request with retry and circuit-breaker protection.
func (c *Client) Complete(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if !c.cb.Allow() {
		return nil, fmt.Errorf("llm: circuit breaker open for provider %s", c.cfg.Provider)
	}

	// Build request
	llmReq := llmclient.ChatRequest{
		Model: c.cfg.Model,
		Messages: []llmclient.Message{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserPrompt},
		},
		MaxTokens:   c.maxTokens(req.MaxTokens),
		Temperature: c.temperature(req.Temperature),
		// Note: No tools/function calling - per spec
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retry.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.backoffDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := c.call(ctx, llmReq)
		if err == nil {
			c.cb.RecordSuccess()
			return resp, nil
		}
		lastErr = err

		if !c.isRetriable(err) {
			c.cb.RecordFailure()
			return nil, err
		}
	}

	c.cb.RecordFailure()
	return nil, fmt.Errorf("llm: %d retries exhausted: %w", c.cfg.Retry.MaxRetries, lastErr)
}

// call executes the HTTP request.
func (c *Client) call(ctx context.Context, req llmclient.ChatRequest) (*ChatResponse, error) {
	resp, err := c.inner.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{
		Content: resp.Content,
		Usage: TokenUsage{
			PromptTokens:     resp.InputTokens,
			CompletionTokens: resp.OutputTokens,
			TotalTokens:      resp.InputTokens + resp.OutputTokens,
		},
		Model:        c.cfg.Model,
		FinishReason: resp.FinishReason,
	}, nil
}

// maxTokens returns the effective max tokens (request takes precedence over config).
func (c *Client) maxTokens(requestMax int) int {
	if requestMax > 0 {
		return requestMax
	}
	return c.cfg.MaxTokens
}

// temperature returns the effective temperature (request takes precedence over default).
func (c *Client) temperature(requestTemp float64) float64 {
	if requestTemp > 0 {
		return requestTemp
	}
	return 0.2 // Default: low temperature for structured output
}

// backoffDelay computes exponential backoff with jitter.
func (c *Client) backoffDelay(attempt int) time.Duration {
	base := float64(c.cfg.Retry.BaseDelay)
	delay := base * pow(2, float64(attempt-1))
	// Jitter: ±20%
	jitter := 0.8 + 0.4*c.randomFloat()
	delay *= jitter
	max := float64(c.cfg.Retry.MaxDelay)
	if delay > max {
		delay = max
	}
	return time.Duration(delay)
}

// isRetriable returns true for network errors and 5xx responses.
func (c *Client) isRetriable(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Network errors
	if contains(errStr, "http:") || contains(errStr, "timeout") || contains(errStr, "connection refused") {
		return true
	}
	// 5xx server errors
	if contains(errStr, "status 5") {
		return true
	}
	return false
}

// randomFloat returns a random float in [0, 1).
func (c *Client) randomFloat() float64 {
	// Simple pseudo-random for jitter (not security-sensitive)
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// pow computes base^exp for small integer exponents.
func pow(base float64, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

// indexOf finds the index of a substring.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
