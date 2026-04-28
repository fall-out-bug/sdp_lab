package providers

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
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

// defaultOllamaCmdRunner is the production command runner using exec.CommandContext.
func defaultOllamaCmdRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// NewOllamaProvider creates a new OllamaProvider pointing to the given host.
// If host is empty, defaults to "http://localhost:11434".
// Uses the default production command runner (exec.CommandContext).
func NewOllamaProvider(host string) *OllamaProvider {
	if host == "" {
		host = "http://localhost:11434"
	}
	return NewOllamaProviderWithRunner(host, defaultOllamaCmdRunner)
}

// NewOllamaProviderWithRunner creates a new OllamaProvider with a custom command runner.
// This is used for testing to inject mock runners.
func NewOllamaProviderWithRunner(
	host string,
	runner func(ctx context.Context, name string, args ...string) ([]byte, error),
) *OllamaProvider {
	if host == "" {
		host = "http://localhost:11434"
	}
	if runner == nil {
		runner = defaultOllamaCmdRunner
	}
	return &OllamaProvider{
		host:      host,
		cmdRunner: runner,
	}
}

// Name returns the canonical provider name.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Models returns the list of available Ollama models.
// Results are cached with a 5-minute TTL. Parses `ollama list` output
// skipping the header line and extracting the first whitespace-separated column.
// Wraps subprocess call with 5-second timeout.
func (p *OllamaProvider) Models() []string {
	const cacheTTL = 5 * time.Minute

	// Check if cache exists and is fresh
	if cached := p.cache.Load(); cached != nil {
		if time.Since(cached.cachedAt) < cacheTTL {
			return cached.models
		}
	}

	// Cache miss or expired; fetch fresh list with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := p.cmdRunner(ctx, "ollama", "list")
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
