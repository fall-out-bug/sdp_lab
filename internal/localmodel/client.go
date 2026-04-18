// Package localmodel provides a minimal client for Ollama's HTTP API.
// Used by the dispatch tier to route low-complexity tasks to a local model.
package localmodel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Config carries all injectable dependencies for a Client.
type Config struct {
	BaseURL    string       // e.g. "http://localhost:11434"
	Model      string       // e.g. "qwen2.5-coder:7b"
	HTTPClient *http.Client // optional; nil = http.DefaultClient
}

// Client calls the Ollama /api/generate endpoint.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient validates cfg and returns a ready Client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("localmodel.NewClient: BaseURL must not be empty")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, fmt.Errorf("localmodel.NewClient: Model must not be empty")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		model:      cfg.Model,
		httpClient: hc,
	}, nil
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

// Prompt sends text to the local model and returns the generated response.
func (c *Client) Prompt(ctx context.Context, text string) (string, error) {
	body, err := json.Marshal(generateRequest{
		Model:  c.model,
		Prompt: text,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("localmodel: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("localmodel: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("localmodel: http: %w", err)
	}
	defer resp.Body.Close()

	var gr generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("localmodel: decode response: %w", err)
	}
	if gr.Error != "" {
		return "", fmt.Errorf("localmodel: ollama error: %s", gr.Error)
	}
	return gr.Response, nil
}

// Model returns the model name this client was configured with.
func (c *Client) Model() string { return c.model }
