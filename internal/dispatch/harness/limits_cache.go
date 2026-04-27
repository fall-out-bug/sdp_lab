package harness

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LimitsCache manages cached rate-limit information for providers.
// It maintains a background poller that calls Provider.CheckLimits periodically,
// and allows updates from HTTP response headers which take priority over poller data.
type LimitsCache struct {
	ttl      time.Duration
	pollers  map[string]*time.Ticker
	cache    sync.Map // map[string]*Limits
	expiry   sync.Map // map[string]time.Time — expiry time of header-derived data
	stopOnce sync.Once
	stopChan chan struct{}
}

// NewLimitsCache creates a new LimitsCache with the given TTL (default 30s).
func NewLimitsCache(ttl time.Duration) *LimitsCache {
	if ttl == 0 {
		ttl = 30 * time.Second
	}
	return &LimitsCache{
		ttl:      ttl,
		pollers:  make(map[string]*time.Ticker),
		stopChan: make(chan struct{}),
	}
}

// Start launches a background poller for each provider that calls CheckLimits periodically.
// The poller interval is ttl/2 to ensure we get fresh data before the cache expires.
func (c *LimitsCache) Start(ctx context.Context, providers []Provider) {
	interval := c.ttl / 2
	if interval == 0 {
		interval = 15 * time.Second
	}

	for _, p := range providers {
		pName := p.Name()
		ticker := time.NewTicker(interval)
		c.pollers[pName] = ticker

		go func(name string, provider Provider) {
			// Initial check
			if limits, err := provider.CheckLimits(ctx); err == nil && limits != nil {
				// Only store if no header data exists
				if _, hasExpiry := c.expiry.Load(name); !hasExpiry {
					c.cache.Store(name, limits)
				}
			}

			for {
				select {
				case <-ticker.C:
					if limits, err := provider.CheckLimits(ctx); err == nil && limits != nil {
						// Only update cache if header-derived data has expired
						expiryVal, hasExpiry := c.expiry.Load(name)
						if !hasExpiry {
							// No header data, safe to update
							c.cache.Store(name, limits)
						} else if time.Now().UTC().After(expiryVal.(time.Time)) {
							// Header data expired, update from poller
							c.cache.Store(name, limits)
							c.expiry.Delete(name)
						}
					}
				case <-c.stopChan:
					return
				case <-ctx.Done():
					return
				}
			}
		}(pName, p)
	}
}

// Stop cleanly shuts down all pollers and prevents further updates.
func (c *LimitsCache) Stop() {
	c.stopOnce.Do(func() {
		// Stop all ticker pollers
		for _, ticker := range c.pollers {
			ticker.Stop()
		}
		// Signal all goroutines to exit
		close(c.stopChan)
	})
}

// Get returns the cached Limits for the given provider name, or nil if not found.
// The cache is thread-safe and returns the most recent data available (header-derived or poller).
func (c *LimitsCache) Get(name string) *Limits {
	val, ok := c.cache.Load(name)
	if !ok {
		return nil
	}
	return val.(*Limits)
}

// UpdateFromHeaders updates the cache with limits extracted from HTTP response headers.
// Header-derived limits take priority over poller data for the duration of the TTL.
// This is called after each harness invocation completes.
//
// Supported header families (case-insensitive):
// - x-ratelimit-remaining-requests / x-ratelimit-limit-requests
// - anthropic-ratelimit-requests-remaining / anthropic-ratelimit-requests-limit
func (c *LimitsCache) UpdateFromHeaders(name string, hdrs http.Header) {
	remaining, limit := parseRateLimitHeaders(hdrs)
	if remaining < 0 || limit < 0 {
		// No recognized headers found
		return
	}

	limits := &Limits{
		Total:     limit,
		Used:      limit - remaining,
		Window:    "1h", // default window; header parsing may refine this
		Source:    "headers/" + name,
		CheckedAt: time.Now().UTC(),
	}

	c.cache.Store(name, limits)
	// Mark when this header-derived data expires
	c.expiry.Store(name, time.Now().UTC().Add(c.ttl))
}

// parseRateLimitHeaders extracts remaining and limit from standard rate-limit headers.
// Returns -1 for both if no recognized headers are found.
func parseRateLimitHeaders(hdrs http.Header) (remaining, limit int) {
	remaining, limit = -1, -1

	// Try x-ratelimit-remaining-requests / x-ratelimit-limit-requests
	if remStr := headerValue(hdrs, "x-ratelimit-remaining-requests"); remStr != "" {
		remaining = parseIntHeader(remStr)
	}
	if limStr := headerValue(hdrs, "x-ratelimit-limit-requests"); limStr != "" {
		limit = parseIntHeader(limStr)
	}

	// If found, return early
	if remaining >= 0 && limit >= 0 {
		return
	}

	// Try anthropic-ratelimit-requests-remaining / anthropic-ratelimit-requests-limit
	if remStr := headerValue(hdrs, "anthropic-ratelimit-requests-remaining"); remStr != "" {
		remaining = parseIntHeader(remStr)
	}
	if limStr := headerValue(hdrs, "anthropic-ratelimit-requests-limit"); limStr != "" {
		limit = parseIntHeader(limStr)
	}

	return
}

// headerValue returns the first value of a header (case-insensitive).
func headerValue(hdrs http.Header, name string) string {
	// http.Header is case-insensitive; Get handles it
	return hdrs.Get(name)
}

// parseIntHeader safely parses an integer from a header value string.
func parseIntHeader(val string) int {
	val = strings.TrimSpace(val)
	n, err := strconv.Atoi(val)
	if err != nil {
		return -1
	}
	return n
}
