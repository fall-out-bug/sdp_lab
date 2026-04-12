package architect

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"sdp_dev/internal/llmclient"
)

// LLMConfig holds configuration for the LLM HTTP client.
type LLMConfig struct {
	// BaseURL is the OpenAI-compatible API endpoint.
	// Default: "https://openrouter.ai/api/v1"
	BaseURL string

	// APIKey is the authentication key. Sourced from OPENROUTER_API_KEY env
	// if empty.
	APIKey string

	// Model is the model identifier to use (e.g. "openai/gpt-4o-mini").
	Model string

	// MaxTokens caps the response length. Default: 4096.
	MaxTokens int

	// Timeout is the per-request HTTP timeout. Default: 120s.
	Timeout time.Duration

	// RetryConfig controls exponential backoff retries. Default: 3 retries.
	Retry RetryConfig
}

// DefaultLLMConfig returns a Config with production defaults.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		BaseURL:   "https://openrouter.ai/api/v1",
		APIKey:    os.Getenv("OPENROUTER_API_KEY"),
		Model:     "openai/gpt-4o-mini",
		MaxTokens: 4096,
		Timeout:   120 * time.Second,
		Retry:     NewRetryConfig(),
	}
}

// TokenUsage tracks token consumption from a single API call.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMClient is an HTTP client for OpenAI-compatible chat completion APIs.
type LLMClient struct {
	cfg    LLMConfig
	inner  *llmclient.Client
	filter *SecurityFilter
	cb     *CircuitBreaker
}

// NewLLMClient creates a client with the given config and security filter.
func NewLLMClient(cfg LLMConfig, sf *SecurityFilter) *LLMClient {
	return &LLMClient{
		cfg:    cfg,
		inner:  llmclient.New(cfg.APIKey, cfg.BaseURL),
		filter: sf,
		cb:     NewCircuitBreaker(),
	}
}

// Complete sends a chat completion request with retry and circuit-breaker
// protection. It returns the assistant message content and token usage.
func (c *LLMClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (content string, usage TokenUsage, err error) {
	if !c.cb.Allow() {
		return "", TokenUsage{}, fmt.Errorf("llm: circuit breaker open for provider %s", c.cfg.BaseURL)
	}

	req := llmclient.ChatRequest{
		Model: c.cfg.Model,
		Messages: []llmclient.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   c.cfg.MaxTokens,
		Temperature: 0.2, // low temperature for deterministic structured output
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retry.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.backoffDelay(attempt)
			select {
			case <-ctx.Done():
				return "", TokenUsage{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		content, usage, err = c.callLLM(ctx, req)
		if err == nil {
			c.cb.RecordSuccess()
			return content, usage, nil
		}
		lastErr = err

		// Non-retriable errors: stop immediately.
		if !isRetriable(err) {
			c.cb.RecordFailure()
			return "", TokenUsage{}, err
		}
	}

	c.cb.RecordFailure()
	return "", TokenUsage{}, fmt.Errorf("llm: %d retries exhausted: %w", c.cfg.Retry.MaxRetries, lastErr)
}

// callLLM delegates the HTTP call to llmclient.Client.Chat and classifies errors.
func (c *LLMClient) callLLM(ctx context.Context, req llmclient.ChatRequest) (_ string, usage TokenUsage, _ error) {
	resp, err := c.inner.Chat(ctx, req)
	if err != nil {
		errStr := err.Error()
		// Network errors and 5xx server errors are retriable.
		if strings.Contains(errStr, "http:") {
			return "", TokenUsage{}, &retriableError{err: err}
		}
		// Status 5xx from llmclient: "status 500: ..."
		if strings.Contains(errStr, "status 5") {
			scrubbed := scrubSecretsJSON(truncate(errStr, 200))
			return "", TokenUsage{}, &retriableError{err: fmt.Errorf("server error: %s", scrubbed)}
		}
		// 4xx and other errors are not retriable.
		scrubbed := scrubSecretsJSON(truncate(errStr, 200))
		return "", TokenUsage{}, fmt.Errorf("client error: %s", scrubbed)
	}

	return resp.Content, TokenUsage{
		PromptTokens:     resp.InputTokens,
		CompletionTokens: resp.OutputTokens,
		TotalTokens:      resp.InputTokens + resp.OutputTokens,
	}, nil
}

// backoffDelay computes an exponential backoff with jitter.
func (c *LLMClient) backoffDelay(attempt int) time.Duration {
	base := float64(c.cfg.Retry.BaseDelay)
	delay := base * math.Pow(2, float64(attempt-1))
	// Add jitter: randomize between 0.8x and 1.2x (±20% per spec).
	jitter := 0.8 + 0.4*rand.Float64() //nolint:gosec // no crypto need for jitter
	delay *= jitter
	max := float64(c.cfg.Retry.MaxDelay)
	if delay > max {
		delay = max
	}
	return time.Duration(delay)
}

// retriableError wraps errors that should trigger a retry.
type retriableError struct {
	err error
}

func (e *retriableError) Error() string { return e.err.Error() }
func (e *retriableError) Unwrap() error { return e.err }

// isRetriable returns true for network errors and 5xx responses.
func isRetriable(err error) bool {
	var re *retriableError
	return isError(err, re)
}

// isError checks if err or any wrapped error matches the target type.
func isError(err error, _ interface{}) bool {
	// Simple type assertion check for retriableError.
	for {
		if _, ok := err.(*retriableError); ok {
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			return false
		}
	}
}

// truncate shortens a string to maxLen runes.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// defaultSecurityFilter is the package-level singleton used by scrubSecretsJSON
// to avoid creating a new SecurityFilter on every call.
var defaultSecurityFilter *SecurityFilter

func init() {
	defaultSecurityFilter = NewSecurityFilter()
}

// scrubSecretsJSON removes detected secret patterns from a string while
// preserving JSON structural characters. It scrubs values only, never keys.
// This is used for LLM output sanitization.
func scrubSecretsJSON(content string) string {
	// JSON-aware scrubbing: scan for secrets and replace only the matched
	// regions, preserving JSON structure around them.
	result := content
	matches := defaultSecurityFilter.ScanForSecrets(content)
	// Replace from end to preserve positions.
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		result = result[:m.Position] + "[REDACTED_" + m.Type + "]" + result[m.Position+m.Length:]
	}
	return result
}
