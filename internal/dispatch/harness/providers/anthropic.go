package providers

import (
	"context"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

// AnthropicProvider implements harness.Provider for Anthropic Claude models.
type AnthropicProvider struct {
	cache *harness.LimitsCache
}

// NewAnthropicProvider creates a new AnthropicProvider with optional cache.
func NewAnthropicProvider(cache *harness.LimitsCache) *AnthropicProvider {
	return &AnthropicProvider{cache: cache}
}

// Name returns the canonical provider name.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Models returns the list of available Claude models.
func (p *AnthropicProvider) Models() []string {
	return []string{
		"claude-opus-4-7",
		"claude-sonnet-4-6",
		"claude-haiku-4-5",
		"claude-opus-4-1",
		"claude-sonnet-4-5",
		"claude-haiku-4-1",
	}
}

// CheckLimits returns rate-limit information for Anthropic.
// If cache is nil, returns an uninitialized Limits struct.
// Otherwise delegates to the cache.
func (p *AnthropicProvider) CheckLimits(ctx context.Context) (*harness.Limits, error) {
	if p.cache == nil {
		return &harness.Limits{
			Source:    "uninitialized",
			CheckedAt: time.Now().UTC(),
		}, nil
	}

	limits := p.cache.Get("anthropic")
	if limits != nil {
		return limits, nil
	}

	// Cache miss, return uninitialized
	return &harness.Limits{
		Source:    "uninitialized",
		CheckedAt: time.Now().UTC(),
	}, nil
}
