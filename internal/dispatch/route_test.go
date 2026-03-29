package dispatch

import (
	"context"
	"testing"

	"sdp_dev/internal/dispatch/harness"
)

func makeProfile(harnessName, provider, model string, caps map[string]CapabilityScore) *CapabilityProfile {
	return &CapabilityProfile{
		Harness:      harnessName,
		Provider:     provider,
		Model:        model,
		Capabilities: caps,
	}
}

func featureCaps(score float64) map[string]CapabilityScore {
	return map[string]CapabilityScore{
		"feature:go": {TestPassRate: score},
	}
}

// TestRouter_Route verifies that when limits are fine, the higher-scored profile wins.
func TestRouter_Route(t *testing.T) {
	claudeProfile := makeProfile("claude-harness", "anthropic", "claude-3", featureCaps(0.92))
	codexProfile := makeProfile("codex-harness", "openai", "codex", featureCaps(0.85))

	router := &Router{
		Profiles: []*CapabilityProfile{claudeProfile, codexProfile},
	}

	task := TaskClassification{TaskType: "feature", Language: "go"}
	limits := map[string]*harness.Limits{
		"anthropic": {Total: 100, Used: 10}, // 10% used → factor 1.0
		"openai":    {Total: 100, Used: 10}, // 10% used → factor 1.0
	}

	dec, err := router.Route(context.Background(), task, limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Harness != "claude-harness" {
		t.Errorf("expected claude-harness, got %q", dec.Harness)
	}
	if dec.Provider != "anthropic" {
		t.Errorf("expected provider anthropic, got %q", dec.Provider)
	}
	if dec.Score != 0.92 {
		t.Errorf("expected score 0.92, got %v", dec.Score)
	}
	if len(dec.Alternatives) != 1 {
		t.Errorf("expected 1 alternative, got %d", len(dec.Alternatives))
	}
}

// TestRouter_Route_LimitsOverride verifies that high usage degrades anthropic score
// enough that codex wins (claude cap=0.92 * factor=0.5 = 0.46 < codex 0.85 * 1.0).
func TestRouter_Route_LimitsOverride(t *testing.T) {
	claudeProfile := makeProfile("claude-harness", "anthropic", "claude-3", featureCaps(0.92))
	codexProfile := makeProfile("codex-harness", "openai", "codex", featureCaps(0.85))

	router := &Router{
		Profiles: []*CapabilityProfile{claudeProfile, codexProfile},
	}

	task := TaskClassification{TaskType: "feature", Language: "go"}
	limits := map[string]*harness.Limits{
		"anthropic": {Total: 100, Used: 85}, // 85% used → factor 0.5
		"openai":    {Total: 100, Used: 10}, // 10% used → factor 1.0
	}

	dec, err := router.Route(context.Background(), task, limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Harness != "codex-harness" {
		t.Errorf("expected codex-harness to win, got %q", dec.Harness)
	}
}

// TestRouter_Route_NoProfiles verifies that an empty profile list returns an error.
func TestRouter_Route_NoProfiles(t *testing.T) {
	router := &Router{Profiles: []*CapabilityProfile{}}

	task := TaskClassification{TaskType: "feature", Language: "go"}
	_, err := router.Route(context.Background(), task, nil)
	if err == nil {
		t.Fatal("expected error for no profiles, got nil")
	}
}

// TestRouter_ColdStart_CapabilityHeuristic verifies that a profile with diverse
// capabilities gets a higher cold-start score than an empty profile.
func TestRouter_ColdStart_CapabilityHeuristic(t *testing.T) {
	// diverseProfile has capabilities for other task types but NOT refactor:go
	diverseProfile := makeProfile("diverse-harness", "provider-a", "model-a", map[string]CapabilityScore{
		"feature:go":  {TestPassRate: 0.90},
		"bugfix:go":   {TestPassRate: 0.80},
		"feature:py":  {TestPassRate: 0.70},
	})
	// emptyProfile has zero capabilities
	emptyProfile := makeProfile("empty-harness", "provider-b", "model-b", nil)

	router := &Router{
		Profiles:          []*CapabilityProfile{emptyProfile, diverseProfile},
		ColdStartStrategy: ColdStartCapabilityHeuristic,
	}

	task := TaskClassification{TaskType: "refactor", Language: "go"}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Harness != "diverse-harness" {
		t.Errorf("expected diverse-harness to win, got %q", dec.Harness)
	}
	if !dec.ColdStart {
		t.Error("expected ColdStart=true")
	}
	if dec.Score <= 0.5 {
		t.Errorf("expected diverse profile score > 0.5, got %v", dec.Score)
	}
	// Verify reason mentions cold-start strategy
	if !contains(dec.Reason, "cold-start: capability-heuristic") {
		t.Errorf("expected reason to mention cold-start strategy, got %q", dec.Reason)
	}
}

// TestRouter_ColdStart_RoundRobin verifies that sequential calls cycle through profiles.
func TestRouter_ColdStart_RoundRobin(t *testing.T) {
	p1 := makeProfile("harness-a", "prov-a", "model-a", nil)
	p2 := makeProfile("harness-b", "prov-b", "model-b", nil)
	p3 := makeProfile("harness-c", "prov-c", "model-c", nil)

	router := &Router{
		Profiles:          []*CapabilityProfile{p1, p2, p3},
		ColdStartStrategy: ColdStartRoundRobin,
	}

	task := TaskClassification{TaskType: "unknown", Language: "go"}
	results := make([]string, 6)
	for i := 0; i < 6; i++ {
		dec, err := router.Route(context.Background(), task, nil)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		results[i] = dec.Harness
		if !dec.ColdStart {
			t.Errorf("call %d: expected ColdStart=true", i)
		}
	}
	// Should cycle: a, b, c, a, b, c
	expected := []string{"harness-a", "harness-b", "harness-c", "harness-a", "harness-b", "harness-c"}
	for i, want := range expected {
		if results[i] != want {
			t.Errorf("call %d: expected %q, got %q", i, want, results[i])
		}
	}
}

// TestRouter_ColdStart_FallbackChain verifies first profile always wins.
func TestRouter_ColdStart_FallbackChain(t *testing.T) {
	p1 := makeProfile("primary", "prov-a", "model-a", nil)
	p2 := makeProfile("secondary", "prov-b", "model-b", nil)

	router := &Router{
		Profiles:          []*CapabilityProfile{p1, p2},
		ColdStartStrategy: ColdStartFallbackChain,
	}

	task := TaskClassification{TaskType: "unknown", Language: "rust"}
	for i := 0; i < 3; i++ {
		dec, err := router.Route(context.Background(), task, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dec.Harness != "primary" {
			t.Errorf("call %d: expected primary, got %q", i, dec.Harness)
		}
		if !dec.ColdStart {
			t.Errorf("call %d: expected ColdStart=true", i)
		}
	}
}

// TestRouter_ColdStart_PartialData verifies that when at least one profile has data
// for the task type, cold start is NOT triggered.
func TestRouter_ColdStart_PartialData(t *testing.T) {
	knownProfile := makeProfile("known-harness", "prov-a", "model-a", map[string]CapabilityScore{
		"feature:go": {TestPassRate: 0.75},
	})
	unknownProfile := makeProfile("unknown-harness", "prov-b", "model-b", nil)

	router := &Router{
		Profiles:          []*CapabilityProfile{unknownProfile, knownProfile},
		ColdStartStrategy: ColdStartRoundRobin, // should NOT be used
	}

	task := TaskClassification{TaskType: "feature", Language: "go"}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Harness != "known-harness" {
		t.Errorf("expected known-harness, got %q", dec.Harness)
	}
	if dec.ColdStart {
		t.Error("expected ColdStart=false when partial data exists")
	}
}

// TestRouter_ColdStart_DefaultStrategy verifies that empty strategy defaults to capability-heuristic.
func TestRouter_ColdStart_DefaultStrategy(t *testing.T) {
	p := makeProfile("solo", "prov", "model", map[string]CapabilityScore{
		"feature:go": {TestPassRate: 0.80},
	})
	router := &Router{
		Profiles: []*CapabilityProfile{p},
		// ColdStartStrategy not set — should default to capability-heuristic
	}

	task := TaskClassification{TaskType: "refactor", Language: "go"}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.ColdStart {
		t.Error("expected ColdStart=true")
	}
	if !contains(dec.Reason, "cold-start: capability-heuristic") {
		t.Errorf("expected default strategy in reason, got %q", dec.Reason)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
