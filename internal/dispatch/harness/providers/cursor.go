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

// cursorModelList represents a cached list of Cursor models with its creation timestamp.
type cursorModelList struct {
	models    []string
	timestamp time.Time
}

// CursorProvider implements harness.Provider for Cursor models.
// It fetches models via `cursor agent --list-models` subprocess and caches them.
type CursorProvider struct {
	cache     *harness.LimitsCache
	cmdRunner cmdRunnerFunc
	nowFn     func() time.Time
	// cache atomic for model list with 10min TTL
	modelCache atomic.Pointer[cursorModelList]
}

// cmdRunnerFunc is the signature for the command runner function.
// Tests override this to inject a fake runner; production uses exec.CommandContext.
type cmdRunnerFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// NewCursorProvider creates a new CursorProvider.
// cache is optional (can be nil). cmdRunner and nowFn default to production implementations.
func NewCursorProvider(
	cache *harness.LimitsCache,
	cmdRunner cmdRunnerFunc,
	nowFn func() time.Time,
) *CursorProvider {
	if cmdRunner == nil {
		cmdRunner = defaultCmdRunner
	}
	if nowFn == nil {
		nowFn = time.Now
	}

	return &CursorProvider{
		cache:     cache,
		cmdRunner: cmdRunner,
		nowFn:     nowFn,
	}
}

// defaultCmdRunner is the production command runner using exec.CommandContext.
func defaultCmdRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// Name returns the canonical provider name.
func (p *CursorProvider) Name() string {
	return "cursor"
}

// Models returns the list of available Cursor models.
// It runs `cursor agent --list-models`, parses the output, and caches it for 10 minutes.
func (p *CursorProvider) Models() []string {
	// Check cache first
	cached := p.modelCache.Load()
	if cached != nil {
		ttl := 10 * time.Minute
		if p.nowFn().Sub(cached.timestamp) < ttl {
			return cached.models
		}
	}

	// Cache miss or expired; fetch from subprocess
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := p.cmdRunner(ctx, "cursor", "agent", "--list-models")
	if err != nil {
		slog.Warn("cursor agent --list-models failed", "error", err)
		return []string{}
	}

	models := parseListModelsOutput(string(output))

	// Store in cache
	p.modelCache.Store(&cursorModelList{
		models:    models,
		timestamp: p.nowFn(),
	})

	return models
}

// parseListModelsOutput parses the output of `cursor agent --list-models`.
// Expected format:
//
//	Available models:
//
//	<id> - <description>
//	<id> - <description>
//	...
//
// Returns a slice of model IDs, skipping empty lines and the header.
func parseListModelsOutput(output string) []string {
	var models []string
	lines := strings.Split(output, "\n")
	seenHeader := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Identify and skip header line
		if line == "Available models:" {
			seenHeader = true
			continue
		}

		// Only process lines after header
		if !seenHeader {
			continue
		}

		// Parse <id> - <description> format
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) == 2 {
			modelID := strings.TrimSpace(parts[0])
			if modelID != "" {
				models = append(models, modelID)
			}
		}
	}

	return models
}

// CheckLimits returns rate-limit information for Cursor.
// Cursor CLI does not expose rate-limit endpoints, so this returns a stub with Source="cursor-cli".
func (p *CursorProvider) CheckLimits(ctx context.Context) (*harness.Limits, error) {
	return &harness.Limits{
		Total:     0,
		Used:      0,
		Source:    "cursor-cli",
		CheckedAt: time.Now().UTC(),
	}, nil
}
