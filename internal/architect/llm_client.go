package architect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"
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

// chatRequest is the JSON body sent to /v1/chat/completions.
// No tool use / function calling parameters are included.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// chatMessage is a single message in the chat request.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the restricted JSON schema expected from the API.
// Unknown fields are silently discarded (no strict schema enforcement
// on extra fields, but we only read known fields).
type chatResponse struct {
	ID      string `json:"id,omitempty"`
	Choices []struct {
		Index   int `json:"index,omitempty"`
		Message struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens,omitempty"`
		CompletionTokens int `json:"completion_tokens,omitempty"`
		TotalTokens      int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

// LLMClient is an HTTP client for OpenAI-compatible chat completion APIs.
type LLMClient struct {
	cfg    LLMConfig
	http   *http.Client
	filter *SecurityFilter
	cb     *CircuitBreaker
}

// NewLLMClient creates a client with the given config and security filter.
func NewLLMClient(cfg LLMConfig, sf *SecurityFilter) *LLMClient {
	return &LLMClient{
		cfg:    cfg,
		http:   &http.Client{Timeout: cfg.Timeout},
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

	reqBody := chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   c.cfg.MaxTokens,
		Temperature: 0.2, // low temperature for deterministic structured output
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("llm: marshal request: %w", err)
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

		content, usage, err = c.doRequest(ctx, body)
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

// doRequest executes a single HTTP request to the completions endpoint.
func (c *LLMClient) doRequest(ctx context.Context, body []byte) (_ string, usage TokenUsage, _ error) {
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("llm: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", TokenUsage{}, &retriableError{err: fmt.Errorf("http do: %w", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
	if err != nil {
		return "", TokenUsage{}, &retriableError{err: fmt.Errorf("read body: %w", err)}
	}

	if resp.StatusCode >= 500 {
		return "", TokenUsage{}, &retriableError{err: fmt.Errorf("server error %d: %s", resp.StatusCode, truncate(string(respBody), 200))}
	}
	if resp.StatusCode >= 400 {
		return "", TokenUsage{}, fmt.Errorf("client error %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", TokenUsage{}, fmt.Errorf("llm: decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", TokenUsage{}, fmt.Errorf("llm: empty choices in response")
	}

	usage = TokenUsage{
		PromptTokens:     chatResp.Usage.PromptTokens,
		CompletionTokens: chatResp.Usage.CompletionTokens,
		TotalTokens:      chatResp.Usage.TotalTokens,
	}

	return chatResp.Choices[0].Message.Content, usage, nil
}

// backoffDelay computes an exponential backoff with jitter.
func (c *LLMClient) backoffDelay(attempt int) time.Duration {
	base := float64(c.cfg.Retry.BaseDelay)
	delay := base * math.Pow(2, float64(attempt-1))
	// Add jitter: randomize between 0.5x and 1.5x.
	jitter := 0.5 + rand.Float64() //nolint:gosec // no crypto need for jitter
	delay *= jitter
	max := float64(c.cfg.Retry.MaxDelay)
	if delay > max {
		delay = max
	}
	return time.Duration(delay)
}

// TokenUsage tracks token consumption from a single API call.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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

// scrubSecretsJSON removes detected secret patterns from a string while
// preserving JSON structural characters. It scrubs values only, never keys.
// This is used for LLM output sanitization.
func scrubSecretsJSON(content string) string {
	sf := NewSecurityFilter()

	// JSON-aware scrubbing: scan for secrets and replace only the matched
	// regions, preserving JSON structure around them.
	result := content
	matches := sf.ScanForSecrets(content)
	// Replace from end to preserve positions.
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		result = result[:m.Position] + "[REDACTED_" + m.Type + "]" + result[m.Position+m.Length:]
	}
	return result
}
