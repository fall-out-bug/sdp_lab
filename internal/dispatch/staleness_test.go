package dispatch

import (
	"context"
	"testing"
	"time"

	"sdp_dev/internal/dispatch/harness"
)

var testCfg = DefaultStalenessConfig

func profileWithAge(updatedAt string) *CapabilityProfile {
	return &CapabilityProfile{
		Harness:   "test-harness",
		Provider:  "test-provider",
		Model:     "test-model",
		UpdatedAt: updatedAt,
	}
}

func TestCheckFreshness_Fresh(t *testing.T) {
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	p := profileWithAge(now.Add(-1 * time.Hour).Format(time.RFC3339))
	got := CheckFreshness(p, testCfg, now)
	if got != FreshnessFresh {
		t.Errorf("expected fresh, got %q", got)
	}
}

func TestCheckFreshness_Stale(t *testing.T) {
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	p := profileWithAge(now.Add(-10 * 24 * time.Hour).Format(time.RFC3339))
	got := CheckFreshness(p, testCfg, now)
	if got != FreshnessStale {
		t.Errorf("expected stale, got %q", got)
	}
}

func TestCheckFreshness_Expired(t *testing.T) {
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	p := profileWithAge(now.Add(-60 * 24 * time.Hour).Format(time.RFC3339))
	got := CheckFreshness(p, testCfg, now)
	if got != FreshnessExpired {
		t.Errorf("expected expired, got %q", got)
	}
}

func TestCheckFreshness_EmptyUpdatedAt(t *testing.T) {
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	p := profileWithAge("")
	got := CheckFreshness(p, testCfg, now)
	if got != FreshnessExpired {
		t.Errorf("expected expired for empty UpdatedAt, got %q", got)
	}
}

func TestDecayScore(t *testing.T) {
	tests := []struct {
		name      string
		score     float64
		freshness Freshness
		want      float64
	}{
		{"fresh unchanged", 0.9, FreshnessFresh, 0.9},
		{"stale halved", 0.9, FreshnessStale, 0.45},
		{"expired zero", 0.9, FreshnessExpired, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecayScore(tt.score, tt.freshness, testCfg)
			if got != tt.want {
				t.Errorf("DecayScore(%v, %v) = %v, want %v", tt.score, tt.freshness, got, tt.want)
			}
		})
	}
}

// TestRouter_WithStaleness verifies that a stale profile with higher base score
// loses to a fresh profile with lower base score after decay is applied.
func TestRouter_WithStaleness(t *testing.T) {
	now := time.Now().UTC()

	// staleProfile: high base score (0.95) but 10 days old → stale → 0.95 * 0.5 = 0.475
	staleProfile := &CapabilityProfile{
		Harness:  "stale-harness",
		Provider: "prov-a",
		Model:    "model-a",
		Capabilities: map[string]CapabilityScore{
			"feature:go": {TestPassRate: 0.95},
		},
		UpdatedAt: now.Add(-10 * 24 * time.Hour).Format(time.RFC3339),
	}

	// freshProfile: lower base score (0.70) but 1 hour old → fresh → 0.70
	freshProfile := &CapabilityProfile{
		Harness:  "fresh-harness",
		Provider: "prov-b",
		Model:    "model-b",
		Capabilities: map[string]CapabilityScore{
			"feature:go": {TestPassRate: 0.70},
		},
		UpdatedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
	}

	cfg := DefaultStalenessConfig
	router := &Router{
		Profiles:        []*CapabilityProfile{staleProfile, freshProfile},
		StalenessConfig: &cfg,
	}

	task := TaskClassification{TaskType: "feature", Language: "go"}
	limits := map[string]*harness.Limits{
		"prov-a": {Total: 100, Used: 10},
		"prov-b": {Total: 100, Used: 10},
	}

	dec, err := router.Route(context.Background(), task, limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Harness != "fresh-harness" {
		t.Errorf("expected fresh-harness to win over stale, got %q (score=%v)", dec.Harness, dec.Score)
	}
	if dec.Staleness != "fresh" {
		t.Errorf("expected staleness=fresh for winner, got %q", dec.Staleness)
	}
}
