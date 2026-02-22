package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const openRouterEndpoint = "https://openrouter.ai/api/v1/chat/completions"
const defaultTimeout = 2 * time.Minute

// OpenRouterMessage is a chat message for the OpenRouter API.
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterRequest is the request body for chat completions.
type OpenRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenRouterMessage  `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

// OpenRouterChoice represents a response choice.
type OpenRouterChoice struct {
	Message struct {
		Content string `json:"content"`
		Role    string `json:"role"`
	} `json:"message"`
}

// OpenRouterUsage holds token usage from the API response.
type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenRouterResponse is the chat completion response.
type OpenRouterResponse struct {
	Choices []OpenRouterChoice `json:"choices"`
	Usage   *OpenRouterUsage   `json:"usage,omitempty"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// OpenRouterClient is a client for the OpenRouter chat completions API.
type OpenRouterClient struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewOpenRouterClient returns a client using OPENROUTER_API_KEY from env.
func NewOpenRouterClient() *OpenRouterClient {
	key := os.Getenv("OPENROUTER_API_KEY")
	return &OpenRouterClient{
		APIKey: key,
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// ChatResult holds the response content and usage.
type ChatResult struct {
	Content         string
	PromptTokens    int
	CompletionTokens int
}

// Chat sends a chat completion request and returns the assistant message content.
func (c *OpenRouterClient) Chat(ctx context.Context, model string, messages []OpenRouterMessage) (string, error) {
	content, _, err := c.ChatWithUsage(ctx, model, messages)
	return content, err
}

// ChatWithUsage returns content and token usage for telemetry.
func (c *OpenRouterClient) ChatWithUsage(ctx context.Context, model string, messages []OpenRouterMessage) (string, *ChatResult, error) {
	if c.APIKey == "" {
		return "", nil, fmt.Errorf("OPENROUTER_API_KEY not set")
	}
	if model == "" {
		model = "anthropic/claude-sonnet-4"
	}

	reqBody := OpenRouterRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: 4096,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var out OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", nil, fmt.Errorf("decode response: %w", err)
	}
	if out.Error != nil {
		return "", nil, fmt.Errorf("openrouter error: %s", out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("openrouter status %d", resp.StatusCode)
	}
	if len(out.Choices) == 0 {
		return "", nil, fmt.Errorf("openrouter: no choices in response")
	}
	content := out.Choices[0].Message.Content
	result := &ChatResult{Content: content}
	if out.Usage != nil {
		result.PromptTokens = out.Usage.PromptTokens
		result.CompletionTokens = out.Usage.CompletionTokens
	}
	return content, result, nil
}

// Complete is a convenience method for a single user prompt.
func (c *OpenRouterClient) Complete(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	msgs := []OpenRouterMessage{}
	if systemPrompt != "" {
		msgs = append(msgs, OpenRouterMessage{Role: "system", Content: systemPrompt})
	}
	msgs = append(msgs, OpenRouterMessage{Role: "user", Content: userPrompt})
	return c.Chat(ctx, model, msgs)
}
