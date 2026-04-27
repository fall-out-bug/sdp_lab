package providers

import (
	"context"
	"net/http"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

func TestAnthropicProvider_Models(t *testing.T) {
	provider := NewAnthropicProvider(nil)
	models := provider.Models()

	if len(models) == 0 {
		t.Error("Models() returned empty list, want ≥ 6")
	}

	expectedModels := map[string]bool{
		"claude-opus-4-7":      true,
		"claude-sonnet-4-6":    true,
		"claude-haiku-4-5":     true,
		"claude-opus-4-1":      true,
		"claude-sonnet-4-5":    true,
		"claude-haiku-4-1":     true,
	}

	for _, model := range models {
		if !expectedModels[model] {
			t.Logf("Models() returned unexpected model: %q", model)
		}
	}

	for expected := range expectedModels {
		found := false
		for _, model := range models {
			if model == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Models() does not contain expected model %q", expected)
		}
	}
}

func TestAnthropicProvider_Name(t *testing.T) {
	provider := NewAnthropicProvider(nil)
	got := provider.Name()
	want := "anthropic"

	if got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

func TestAnthropicProvider_CheckLimits_NilCache(t *testing.T) {
	provider := NewAnthropicProvider(nil)
	ctx := context.Background()

	limits, err := provider.CheckLimits(ctx)

	if err != nil {
		t.Fatalf("CheckLimits(nil cache) failed: %v", err)
	}

	if limits == nil {
		t.Fatal("CheckLimits(nil cache) returned nil, want &Limits")
	}

	if limits.Source != "uninitialized" {
		t.Errorf("CheckLimits(nil cache) Source = %q, want %q", limits.Source, "uninitialized")
	}
}

func TestAnthropicProvider_CheckLimits_WithCache(t *testing.T) {
	cache := harness.NewLimitsCache(0)

	// Simulate UpdateFromHeaders with mock anthropic-ratelimit headers
	hdrs := http.Header{}
	hdrs.Set("anthropic-ratelimit-requests-remaining", "1000")
	hdrs.Set("anthropic-ratelimit-requests-limit", "2000")

	cache.UpdateFromHeaders("anthropic", hdrs)

	provider := NewAnthropicProvider(cache)
	ctx := context.Background()

	limits, err := provider.CheckLimits(ctx)

	if err != nil {
		t.Fatalf("CheckLimits(with cache) failed: %v", err)
	}

	if limits == nil {
		t.Fatal("CheckLimits(with cache) returned nil, want &Limits")
	}

	if limits.Total != 2000 {
		t.Errorf("CheckLimits(with cache) Total = %d, want 2000", limits.Total)
	}

	if limits.Used != 1000 {
		t.Errorf("CheckLimits(with cache) Used = %d, want 1000", limits.Used)
	}

	if limits.Source != "headers/anthropic" {
		t.Errorf("CheckLimits(with cache) Source = %q, want %q", limits.Source, "headers/anthropic")
	}
}

func TestAnthropicProvider_CheckLimits_CacheMiss(t *testing.T) {
	cache := harness.NewLimitsCache(0)
	// Do not populate cache with anthropic data

	provider := NewAnthropicProvider(cache)
	ctx := context.Background()

	limits, err := provider.CheckLimits(ctx)

	if err != nil {
		t.Fatalf("CheckLimits(cache miss) failed: %v", err)
	}

	if limits == nil {
		t.Fatal("CheckLimits(cache miss) returned nil, want &Limits")
	}

	if limits.Source != "uninitialized" {
		t.Errorf("CheckLimits(cache miss) Source = %q, want %q", limits.Source, "uninitialized")
	}
}
