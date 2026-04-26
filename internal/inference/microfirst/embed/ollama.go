package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaEmbedder calls Ollama POST /api/embeddings to get text embeddings.
type OllamaEmbedder struct {
	BaseURL    string // e.g. "http://localhost:11434"
	Model      string // e.g. "bge-small-en-v1.5"
	HTTPClient *http.Client
}

// NewOllamaEmbedder returns a new embedder with default model bge-small-en-v1.5.
func NewOllamaEmbedder(baseURL string) *OllamaEmbedder {
	return &OllamaEmbedder{
		BaseURL:    baseURL,
		Model:      "bge-small-en-v1.5",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed returns the embedding vector for text.
// POST /api/embeddings body: {"model": "...", "prompt": "..."}
// Response: {"embedding": [0.1, 0.2, ...]}
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	reqBody := embedRequest{
		Model:  e.Model,
		Prompt: text,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("embed: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+"/api/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("embed: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("embed: http status %d", resp.StatusCode)
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("embed: decode response: %w", err)
	}

	return result.Embedding, nil
}
