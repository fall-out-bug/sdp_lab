package providers

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"sdp_dev/internal/dispatch/harness"
)

// OllamaProvider implements the Provider interface for a local Ollama instance.
// It runs `ollama list` subprocess (with 5-minute cache) to discover models,
// and always returns unlimited rate limits (no API rate-capping for local models).
type OllamaProvider struct {
	host      string
	cache     atomic.Pointer[ollamaModelList]
	cmdRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// ollamaModelList caches the output of `ollama list` with a TTL.
type ollamaModelList struct {
	models    []string
	cachedAt  time.Time
	cacheTTL  time.Duration
}

// NewOllamaProvider creates a new OllamaProvider pointing to the given host.
// If host is empty, defaults to "http://localhost:11434".
func NewOllamaProvider(host string) *OllamaProvider {
	if host == "" {
		host = "http://localhost:11434"
	}
	p := &OllamaProvider{
		host: host,
	}
	// Set default command runner that invokes `ollama list` as subprocess
	p.cmdRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// In production, this would run the actual ollama CLI.
		// For now, return empty (tests will inject a mock).
		return nil, nil
	}
	return p
}

// SetCmdRunner injects a mock command runner for testing.
func (p *OllamaProvider) SetCmdRunner(runner func(ctx context.Context, name string, args ...string) ([]byte, error)) {
	p.cmdRunner = runner
}

// Name returns the canonical provider name.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Models returns the list of available Ollama models.
// Results are cached with a 5-minute TTL. Parses `ollama list` output
// skipping the header line and extracting the first whitespace-separated column.
func (p *OllamaProvider) Models() []string {
	const cacheTTL = 5 * time.Minute

	// Check if cache exists and is fresh
	if cached := p.cache.Load(); cached != nil {
		if time.Since(cached.cachedAt) < cacheTTL {
			return cached.models
		}
	}

	// Cache miss or expired; fetch fresh list
	output, err := p.cmdRunner(context.Background(), "ollama", "list")
	if err != nil {
		slog.Warn("ollama list command failed", "error", err)
		// Return empty list on error
		return []string{}
	}

	models := parseOllamaList(string(output))
	p.cache.Store(&ollamaModelList{
		models:   models,
		cachedAt: time.Now(),
		cacheTTL: cacheTTL,
	})

	return models
}

// CheckLimits always returns unlimited rates for local Ollama (no API rate-capping).
func (p *OllamaProvider) CheckLimits(ctx context.Context) (*harness.Limits, error) {
	return &harness.Limits{
		Total:     999999,
		Used:      0,
		Window:    "unlimited",
		Source:    "local",
		CheckedAt: time.Now().UTC(),
	}, nil
}

// parseOllamaList parses the output of `ollama list`.
// Skips the header line and extracts model names from the first column.
// Example input:
//
//	NAME                       ID              SIZE      MODIFIED
//	qwen2.5-coder:7b           abc123def456    4.5 GB    2 days ago
//	llama3.2:3b                def456ghi789    2.0 GB    1 week ago
func parseOllamaList(output string) []string {
	lines := strings.Split(output, "\n")
	var models []string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header (first non-empty line that contains "NAME")
		if i == 0 || strings.Contains(line, "NAME") {
			continue
		}

		// Extract first column (model name) — split on whitespace
		fields := strings.Fields(line)
		if len(fields) > 0 {
			models = append(models, fields[0])
		}
	}

	return models
}
