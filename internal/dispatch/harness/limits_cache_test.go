package harness

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name   string
	limits *Limits
	calls  atomic.Int32
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Models() []string {
	return []string{"mock-model"}
}

func (m *mockProvider) CheckLimits(ctx context.Context) (*Limits, error) {
	m.calls.Add(1)
	return &Limits{
		Total:     1000,
		Used:      int(m.calls.Load() * 10),
		Window:    "1h",
		Source:    "poller",
		CheckedAt: time.Now().UTC(),
	}, nil
}

// TestLimitsCache_Poller_Updates verifies that Start launches pollers that update the cache.
func TestLimitsCache_Poller_Updates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	provider := &mockProvider{name: "test-provider"}
	ttl := 100 * time.Millisecond // short TTL for fast test

	cache := NewLimitsCache(ttl)
	defer cache.Stop()

	cache.Start(ctx, []Provider{provider})

	// Allow poller to run at least once
	time.Sleep(150 * time.Millisecond)

	limits := cache.Get("test-provider")
	if limits == nil {
		t.Fatalf("expected limits, got nil")
	}
	if limits.Source != "poller" {
		t.Errorf("expected Source=poller, got %q", limits.Source)
	}
	if limits.Used <= 0 {
		t.Errorf("expected Used > 0 after poller call, got %d", limits.Used)
	}
}

// TestLimitsCache_Get_Latest verifies that Get returns the most recent cached limits.
func TestLimitsCache_Get_Latest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	provider := &mockProvider{name: "test-provider"}
	ttl := 50 * time.Millisecond

	cache := NewLimitsCache(ttl)
	defer cache.Stop()

	cache.Start(ctx, []Provider{provider})

	// Allow first poller run
	time.Sleep(100 * time.Millisecond)
	limits1 := cache.Get("test-provider")

	// Allow second poller run
	time.Sleep(100 * time.Millisecond)
	limits2 := cache.Get("test-provider")

	if limits1 == nil || limits2 == nil {
		t.Fatal("expected non-nil limits")
	}

	// Second call should have higher Used value
	if limits2.Used <= limits1.Used {
		t.Errorf("expected limits2.Used > limits1.Used, got %d vs %d", limits2.Used, limits1.Used)
	}
}

// TestLimitsCache_UpdateFromHeaders_Priority verifies that UpdateFromHeaders takes priority over poller.
func TestLimitsCache_UpdateFromHeaders_Priority(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	provider := &mockProvider{name: "test-provider"}
	ttl := 100 * time.Millisecond

	cache := NewLimitsCache(ttl)
	defer cache.Stop()

	cache.Start(ctx, []Provider{provider})

	// Allow poller to update
	time.Sleep(150 * time.Millisecond)

	// Update from headers (should override poller data)
	headers := make(http.Header)
	headers.Add("x-ratelimit-limit-requests", "5000")
	headers.Add("x-ratelimit-remaining-requests", "4900")
	cache.UpdateFromHeaders("test-provider", headers)

	limits := cache.Get("test-provider")
	if limits == nil {
		t.Fatalf("expected limits, got nil")
	}

	if limits.Total != 5000 {
		t.Errorf("expected Total=5000 from headers, got %d", limits.Total)
	}
	if limits.Used != 100 { // 5000 - 4900
		t.Errorf("expected Used=100 from headers, got %d", limits.Used)
	}
	if limits.Source != "headers/test-provider" {
		t.Errorf("expected Source=headers/test-provider, got %q", limits.Source)
	}
}

// TestLimitsCache_Concurrent_ThreadSafe verifies that concurrent Get/UpdateFromHeaders are thread-safe.
func TestLimitsCache_Concurrent_ThreadSafe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-running concurrent test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	provider := &mockProvider{name: "concurrent-provider"}
	cache := NewLimitsCache(50 * time.Millisecond)
	defer cache.Stop()

	cache.Start(ctx, []Provider{provider})

	// Run 100 concurrent goroutines mixing Get and UpdateFromHeaders
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				// Even IDs: Get
				_ = cache.Get("concurrent-provider")
			} else {
				// Odd IDs: UpdateFromHeaders
				headers := http.Header{
					"x-ratelimit-limit-requests":     {"1000"},
					"x-ratelimit-remaining-requests": {"900"},
				}
				cache.UpdateFromHeaders("concurrent-provider", headers)
			}
		}(i)
	}

	wg.Wait()
	// If we reach here without panic (detected by race detector), test passes
}

// TestLimitsCache_Stop_NoGoroutineLeak verifies that Stop properly terminates pollers.
func TestLimitsCache_Stop_NoGoroutineLeak(t *testing.T) {
	ctx := context.Background()

	provider := &mockProvider{name: "leak-provider"}
	cache := NewLimitsCache(50 * time.Millisecond)

	cache.Start(ctx, []Provider{provider})

	// Give pollers time to start
	time.Sleep(100 * time.Millisecond)

	// Stop should cleanly shut down
	cache.Stop()

	// Verify no goroutine panic after Stop (wait a bit for any lingering access)
	time.Sleep(200 * time.Millisecond)
	// If Stop didn't clean up, we'd see goroutine panics in the test runner
}

// TestLimitsCache_TTL_Expiration verifies that header-derived limits are eventually replaced by poller data.
func TestLimitsCache_TTL_Expiration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	provider := &mockProvider{name: "ttl-provider"}
	ttl := 200 * time.Millisecond

	cache := NewLimitsCache(ttl)
	defer cache.Stop()

	cache.Start(ctx, []Provider{provider})

	// Wait for initial poller
	time.Sleep(150 * time.Millisecond)

	// Update from headers
	headers := make(http.Header)
	headers.Add("x-ratelimit-limit-requests", "9999")
	headers.Add("x-ratelimit-remaining-requests", "9998")
	cache.UpdateFromHeaders("ttl-provider", headers)

	limitsAfterHeader := cache.Get("ttl-provider")
	if limitsAfterHeader.Total != 9999 {
		t.Errorf("expected header Total=9999, got %d", limitsAfterHeader.Total)
	}

	// Wait for TTL to expire and new poller cycle
	time.Sleep(ttl + 300*time.Millisecond)

	limitsAfterExpiry := cache.Get("ttl-provider")
	// After TTL expires, poller should have updated with new Used value
	if limitsAfterExpiry.Source != "poller" {
		t.Errorf("expected Source=poller after TTL expiry, got %q", limitsAfterExpiry.Source)
	}
	if limitsAfterExpiry.Total == 9999 {
		t.Errorf("expected header cache to expire, but Total still 9999")
	}
}

// TestLimitsCache_UpdateFromHeaders_Anthropic_Headers verifies parsing of Anthropic-specific headers.
func TestLimitsCache_UpdateFromHeaders_Anthropic_Headers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	provider := &mockProvider{name: "anthropic-provider"}
	cache := NewLimitsCache(100 * time.Millisecond)
	defer cache.Stop()

	cache.Start(ctx, []Provider{provider})

	// Update with Anthropic-style headers
	headers := make(http.Header)
	headers.Add("anthropic-ratelimit-requests-limit", "100000")
	headers.Add("anthropic-ratelimit-requests-remaining", "99900")
	cache.UpdateFromHeaders("anthropic-provider", headers)

	limits := cache.Get("anthropic-provider")
	if limits == nil {
		t.Fatalf("expected limits, got nil")
	}
	if limits.Total != 100000 {
		t.Errorf("expected Total=100000 from Anthropic headers, got %d", limits.Total)
	}
	if limits.Used != 100 { // 100000 - 99900
		t.Errorf("expected Used=100 from Anthropic headers, got %d", limits.Used)
	}
}

// TestLimitsCache_Get_NonExistentProvider returns nil for unknown providers.
func TestLimitsCache_Get_NonExistentProvider(t *testing.T) {
	cache := NewLimitsCache(100 * time.Millisecond)
	defer cache.Stop()

	limits := cache.Get("nonexistent-provider")
	if limits != nil {
		t.Errorf("expected nil for nonexistent provider, got %+v", limits)
	}
}

// TestLimitsCache_UpdateFromHeaders_NoHeaders is a no-op (doesn't error).
func TestLimitsCache_UpdateFromHeaders_NoHeaders(t *testing.T) {
	cache := NewLimitsCache(100 * time.Millisecond)
	defer cache.Stop()

	// Should not panic or error
	cache.UpdateFromHeaders("some-provider", http.Header{})
}
