// Package llmclient is the single HTTP client for all OpenRouter LLM calls in SDP.
// All packages (discovery, architect, strataudit, agentloop/livegw) import this package.
// Never create a separate HTTP client for LLM calls elsewhere.
package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message is a chat message with a role and text content.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Tool describes a tool the model may call.
type Tool struct {
	Type     string       `json:"type"` // always "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction is the function spec within a Tool.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema object
}

// ChatRequest is the request body for both Chat and Stream.
// Set Stream: true for SSE streaming (used by LiveGateway).
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ChatResponse is the response from a non-streaming Chat call.
type ChatResponse struct {
	Content      string
	FinishReason string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// StreamEvent is a single event from a streaming call.
// Type is one of: "text_delta", "tool_call", "finish", "error".
type StreamEvent struct {
	Type string         // "text_delta" | "tool_call" | "finish" | "error"
	Text string         // set when Type == "text_delta"
	Tool *ToolCallChunk // set when Type == "tool_call"
	Err  error          // set when Type == "error"
}

// ToolCallChunk is a finalized tool call emitted when finish_reason == "tool_calls".
// ID is always non-empty: if the provider returned empty, llmclient generates a UUID.
type ToolCallChunk struct {
	ID        string // tool call ID (never empty)
	Name      string // function name
	Arguments string // accumulated JSON arguments string
}

// Client is the LLM HTTP client. Create with New().
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New creates a new Client. apiKey and baseURL are required at construction;
// calls will return errors if apiKey is empty.
func New(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Chat sends a single non-streaming request and returns the full response.
// Use for discovery, architect, strataudit — any package needing request-response semantics.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if c.apiKey == "" {
		return nil, errors.New("llmclient: API key is required")
	}

	req.Stream = false
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, raw)
	}

	var out struct {
		Choices []struct {
			Message      Message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int     `json:"prompt_tokens"`
			CompletionTokens int     `json:"completion_tokens"`
			Cost             float64 `json:"cost"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	return &ChatResponse{
		Content:      out.Choices[0].Message.Content,
		FinishReason: out.Choices[0].FinishReason,
		InputTokens:  out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
		CostUSD:      out.Usage.Cost,
	}, nil
}

func (c *Client) setHeaders(r *http.Request) {
	r.Header.Set("Authorization", "Bearer "+c.apiKey)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("HTTP-Referer", "https://github.com/fall-out-bug/sdp_lab")
	r.Header.Set("X-Title", "SDP")
}
