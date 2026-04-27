package providers

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"sdp_dev/internal/dispatch/harness"
)

func TestOpenAIProvider_Models(t *testing.T) {
	provider := NewOpenAIProvider(nil)
	models := provider.Models()

	if len(models) == 0 {
		t.Fatal("Models() returned empty list")
	}

	expectedModels := []string{"gpt-5", "gpt-5-codex", "o1", "o1-pro", "o3", "o3-mini", "gpt-4o", "gpt-4o-mini"}
	modelMap := make(map[string]bool)
	for _, m := range models {
		modelMap[m] = true
	}

	for _, expected := range expectedModels {
		if !modelMap[expected] {
			t.Errorf("Models() missing expected model: %s", expected)
		}
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProvider(nil)
	if provider.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "openai")
	}
}

func TestOpenAIProvider_CheckLimits_NilCache(t *testing.T) {
	provider := NewOpenAIProvider(nil)
	limits, err := provider.CheckLimits(context.Background())

	if err != nil {
		t.Fatalf("CheckLimits() with nil cache returned error: %v", err)
	}

	if limits == nil {
		t.Fatal("CheckLimits() returned nil limits")
	}

	if limits.Source != "uninitialized" {
		t.Errorf("limits.Source = %q, want %q", limits.Source, "uninitialized")
	}

	if limits.CheckedAt.IsZero() {
		t.Error("limits.CheckedAt should not be zero")
	}
}

func TestOpenAIProvider_CheckLimits_WithCache(t *testing.T) {
	cache := harness.NewLimitsCache(30 * time.Second)

	// Simulate header update
	headers := make(http.Header)
	headers.Set("x-ratelimit-remaining-requests", "45")
	headers.Set("x-ratelimit-limit-requests", "100")

	cache.UpdateFromHeaders("openai", headers)

	provider := NewOpenAIProvider(cache)
	limits, err := provider.CheckLimits(context.Background())

	if err != nil {
		t.Fatalf("CheckLimits() with cache returned error: %v", err)
	}

	if limits == nil {
		t.Fatal("CheckLimits() returned nil limits")
	}

	if limits.Total != 100 {
		t.Errorf("limits.Total = %d, want 100", limits.Total)
	}

	if limits.Used != 55 { // limit - remaining = 100 - 45
		t.Errorf("limits.Used = %d, want 55", limits.Used)
	}

	if !strings.Contains(limits.Source, "openai") {
		t.Errorf("limits.Source = %q, should contain 'openai'", limits.Source)
	}
}
