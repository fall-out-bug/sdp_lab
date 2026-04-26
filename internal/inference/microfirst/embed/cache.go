package embed

import (
	"context"
	"sync"
)

// CachedEmbedder wraps OllamaEmbedder with an in-memory LRU-like cache.
// On overflow (len >= maxSize), the oldest inserted entry is evicted.
type CachedEmbedder struct {
	inner   *OllamaEmbedder
	cache   map[string][]float64
	order   []string // insertion order for eviction
	maxSize int
	mu      sync.Mutex
	hits    int64
	misses  int64
}

// NewCachedEmbedder creates a CachedEmbedder backed by inner with capacity maxSize.
func NewCachedEmbedder(inner *OllamaEmbedder, maxSize int) *CachedEmbedder {
	return &CachedEmbedder{
		inner:   inner,
		cache:   make(map[string][]float64),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

// Embed returns the cached embedding for text, or fetches and caches it.
func (c *CachedEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	c.mu.Lock()
	if vec, ok := c.cache[text]; ok {
		c.hits++
		c.mu.Unlock()
		return vec, nil
	}
	c.misses++
	c.mu.Unlock()

	vec, err := c.inner.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check in case another goroutine populated it while we waited.
	if _, ok := c.cache[text]; !ok {
		if len(c.cache) >= c.maxSize && c.maxSize > 0 {
			// Evict oldest.
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.cache, oldest)
		}
		c.cache[text] = vec
		c.order = append(c.order, text)
	}

	return vec, nil
}

// Stats returns the number of cache hits and misses so far.
func (c *CachedEmbedder) Stats() (hits, misses int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses
}
