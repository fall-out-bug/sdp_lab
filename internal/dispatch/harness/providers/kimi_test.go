package providers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

func TestKimiProvider_Name(t *testing.T) {
	p := NewKimiProvider(nil)
	if p.Name() != "kimi" {
		t.Errorf("expected Name()='kimi', got %q", p.Name())
	}
}

func TestKimiProvider_Models(t *testing.T) {
	p := NewKimiProvider(nil)
	models := p.Models()

	// AC 1: Models() non-empty
	if len(models) == 0 {
		t.Fatal("Models() returned empty slice")
	}

	// AC 1: Contains ≥ 4 Moonshot/Kimi models
	expectedModels := map[string]bool{
		"kimi-k1.5":        true,
		"kimi-k2":          true,
		"moonshot-v1-8k":   true,
		"moonshot-v1-32k":  true,
		"moonshot-v1-128k": true,
	}

	found := make(map[string]bool)
	for _, model := range models {
		if _, ok := expectedModels[model]; ok {
			found[model] = true
		}
	}

	if len(found) < 4 {
		t.Errorf("expected at least 4 Moonshot/Kimi models, found %d: %v", len(found), models)
	}

	// Verify all expected models are present
	for expected := range expectedModels {
		if !found[expected] {
			t.Errorf("expected model %q not found in Models()", expected)
		}
	}
}

func TestKimiProvider_CheckLimits_NilCache(t *testing.T) {
	p := NewKimiProvider(nil)

	ctx := context.Background()
	limits, err := p.CheckLimits(ctx)

	if err != nil {
		t.Fatalf("CheckLimits with nil cache returned error: %v", err)
	}

	if limits == nil {
		t.Fatal("CheckLimits returned nil limits")
	}

	// AC 3: With nil cache, Source should be "kimi-config"
	if limits.Source != "kimi-config" {
		t.Errorf("expected Source='kimi-config', got %q", limits.Source)
	}

	// Verify CheckedAt is set and approximately now
	if limits.CheckedAt.IsZero() {
		t.Fatal("CheckedAt should not be zero")
	}

	now := time.Now().UTC()
	if limits.CheckedAt.After(now.Add(time.Second)) {
		t.Errorf("CheckedAt should be approximately now, got %v", limits.CheckedAt)
	}
}

func TestKimiProvider_CheckLimits_WithCache(t *testing.T) {
	// Create a cache and prime it with test limits
	cache := harness.NewLimitsCache(30 * time.Second)

	// Simulate receiving headers (Moonshot uses x-ratelimit family per design)
	headers := make(http.Header)
	headers.Set("x-ratelimit-remaining-requests", "450")
	headers.Set("x-ratelimit-limit-requests", "500")

	cache.UpdateFromHeaders("kimi", headers)

	p := NewKimiProvider(cache)

	ctx := context.Background()
	limits, err := p.CheckLimits(ctx)

	if err != nil {
		t.Fatalf("CheckLimits with cache returned error: %v", err)
	}

	if limits == nil {
		t.Fatal("CheckLimits returned nil limits")
	}

	// AC 4: Should return cached value from headers
	if limits.Total != 500 {
		t.Errorf("expected Total=500 from cache, got %d", limits.Total)
	}

	if limits.Used != 50 { // limit - remaining
		t.Errorf("expected Used=50 (500-450), got %d", limits.Used)
	}

	// Source should indicate it came from headers
	expectedSourcePrefix := "headers/"
	if len(limits.Source) < len(expectedSourcePrefix) || limits.Source[:len(expectedSourcePrefix)] != expectedSourcePrefix {
		t.Errorf("expected Source to start with %q, got %q", expectedSourcePrefix, limits.Source)
	}
}
