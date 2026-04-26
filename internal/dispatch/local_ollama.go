package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultOllamaBaseURL is the default endpoint for a local Ollama instance.
	DefaultOllamaBaseURL = "http://localhost:11434"
	// DefaultOllamaTimeout is the default timeout for Ollama requests.
	DefaultOllamaTimeout = 30 * time.Second
)

// OllamaClient wraps HTTP calls to a local Ollama instance.
type OllamaClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewOllamaClient creates a new OllamaClient with sensible defaults.
func NewOllamaClient(baseURL string) *OllamaClient {
	if baseURL == "" {
		baseURL = DefaultOllamaBaseURL
	}
	return &OllamaClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: DefaultOllamaTimeout,
		},
	}
}

// HealthCheck returns nil if Ollama is reachable and responding.
func (c *OllamaClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("dispatch: create health check request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("dispatch: ollama health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dispatch: ollama health check returned status %d", resp.StatusCode)
	}

	return nil
}

// Generate sends a prompt and returns the response text.
func (c *OllamaClient) Generate(ctx context.Context, model, prompt string) (string, error) {
	if model == "" {
		return "", fmt.Errorf("dispatch: ollama generate: model name required")
	}
	if prompt == "" {
		return "", fmt.Errorf("dispatch: ollama generate: prompt required")
	}

	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("dispatch: ollama generate marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("dispatch: create generate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("dispatch: ollama generate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("dispatch: ollama generate returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("dispatch: decode ollama response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("dispatch: ollama generate error: %s", result.Error)
	}

	return result.Response, nil
}

// ListModels returns the names of locally available models.
func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("dispatch: create list models request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dispatch: ollama list models failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dispatch: ollama list models returned status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("dispatch: decode ollama models response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}
