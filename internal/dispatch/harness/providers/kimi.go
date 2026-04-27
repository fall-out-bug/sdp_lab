package providers

import (
	"context"
	"time"

	"sdp_dev/internal/dispatch/harness"
)

// KimiProvider is a metadata-only Provider for Moonshot models.
// It provides model catalog and delegates CheckLimits to LimitsCache
// (which parses x-ratelimit headers per Moonshot API spec).
type KimiProvider struct {
	cache *harness.LimitsCache
}

// NewKimiProvider creates a new KimiProvider with optional LimitsCache.
func NewKimiProvider(cache *harness.LimitsCache) *KimiProvider {
	return &KimiProvider{cache: cache}
}

// Name returns the provider name.
func (p *KimiProvider) Name() string {
	return "kimi"
}

// Models returns the Moonshot/Kimi model catalog.
func (p *KimiProvider) Models() []string {
	return []string{
		"kimi-k1.5",
		"kimi-k2",
		"moonshot-v1-8k",
		"moonshot-v1-32k",
		"moonshot-v1-128k",
	}
}

// CheckLimits returns rate-limit information.
// If cache is non-nil, delegates to cache.Get("kimi").
// If cache is nil or returns nil, returns a stub with Source="kimi-config".
func (p *KimiProvider) CheckLimits(ctx context.Context) (*harness.Limits, error) {
	if p.cache != nil {
		if limits := p.cache.Get("kimi"); limits != nil {
			return limits, nil
		}
	}

	// Fallback: return stub with config source
	return &harness.Limits{
		Total:     0,
		Used:      0,
		Window:    "",
		Source:    "kimi-config",
		CheckedAt: time.Now().UTC(),
	}, nil
}
