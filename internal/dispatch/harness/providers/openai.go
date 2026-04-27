package providers

import (
	"context"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

// OpenAIProvider represents the OpenAI model provider with metadata and rate-limit awareness.
// It is metadata-only: no API keys or HTTP calls. Rate limits are delegated to LimitsCache.
type OpenAIProvider struct {
	cache *harness.LimitsCache
}

// NewOpenAIProvider creates a new OpenAI provider with optional cache integration.
func NewOpenAIProvider(cache *harness.LimitsCache) *OpenAIProvider {
	return &OpenAIProvider{cache: cache}
}

// Name returns the canonical provider identifier.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Models returns the list of OpenAI models available through the codex CLI.
func (p *OpenAIProvider) Models() []string {
	return []string{
		"gpt-5",
		"gpt-5-codex",
		"o1",
		"o1-pro",
		"o3",
		"o3-mini",
		"gpt-4o",
		"gpt-4o-mini",
	}
}

// CheckLimits returns the current rate-limit state from cache if available,
// or a stub "uninitialized" limit if cache is nil.
// This method does not make HTTP calls; header parsing is handled by LimitsCache.UpdateFromHeaders.
func (p *OpenAIProvider) CheckLimits(ctx context.Context) (*harness.Limits, error) {
	if p.cache == nil {
		return &harness.Limits{
			Source:    "uninitialized",
			CheckedAt: time.Now().UTC(),
		}, nil
	}

	limits := p.cache.Get(p.Name())
	if limits != nil {
		return limits, nil
	}

	// Cache exists but no data for this provider yet
	return &harness.Limits{
		Source:    "uninitialized",
		CheckedAt: time.Now().UTC(),
	}, nil
}
