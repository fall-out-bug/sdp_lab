package dispatch

import (
	"context"
	"math"
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

// stubMicroRouter is a test double for MicroRouter.
type stubMicroRouter struct {
	hint      string
	confident bool
	calls     int
}

func (s *stubMicroRouter) SuggestCapability(_ context.Context, _, _ string) (string, bool) {
	s.calls++
	return s.hint, s.confident
}

// TestRouter_ColdStart_MicroRouterNil verifies that when MicroRouter=nil, applyColdStart
// behaves identically to the pre-F147 implementation (capability-heuristic by default).
func TestRouter_ColdStart_MicroRouterNil(t *testing.T) {
	diverseProfile := makeProfile("diverse-harness", "prov-a", "model-a", map[string]CapabilityScore{
		"feature:go": {TestPassRate: 0.90},
		"bugfix:go":  {TestPassRate: 0.80},
	})
	emptyProfile := makeProfile("empty-harness", "prov-b", "model-b", nil)

	router := &Router{
		Profiles:          []*CapabilityProfile{emptyProfile, diverseProfile},
		ColdStartStrategy: ColdStartCapabilityHeuristic,
		MicroRouter:       nil, // explicitly nil — backward-compat
	}

	task := TaskClassification{TaskType: "unknown-type", Language: "go"}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Harness != "diverse-harness" {
		t.Errorf("expected diverse-harness, got %q", dec.Harness)
	}
	if !dec.ColdStart {
		t.Error("expected ColdStart=true")
	}
	// Score should reflect the capability heuristic (not a micro override of 0.9).
	// diverse profile avgPassRate = (0.90+0.80)/2 = 0.85
	const wantScore = 0.85
	if math.Abs(dec.Score-wantScore) > 1e-9 {
		t.Errorf("expected score ~0.85 (capability heuristic), got %v", dec.Score)
	}
}

// TestRouter_ColdStart_MicroRouterConfident verifies that when MicroRouter returns
// a confident hint and a profile HasCapability, that profile gets score 0.9.
func TestRouter_ColdStart_MicroRouterConfident(t *testing.T) {
	goProfile := makeProfile("go-harness", "prov-a", "model-a", map[string]CapabilityScore{
		"go-backend:go": {TestPassRate: 0.0}, // score=0 → cold start
	})
	otherProfile := makeProfile("other-harness", "prov-b", "model-b", nil)

	stub := &stubMicroRouter{hint: "go-backend", confident: true}
	router := &Router{
		Profiles:    []*CapabilityProfile{otherProfile, goProfile},
		MicroRouter: stub,
	}

	task := TaskClassification{
		TaskType:    "unknown",
		Language:    "go",
		Title:       "fix nil pointer in Go handler",
		Description: "Go handler panics",
	}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Harness != "go-harness" {
		t.Errorf("expected go-harness, got %q", dec.Harness)
	}
	if dec.Score != 0.9 {
		t.Errorf("expected score 0.9 (micro boost), got %v", dec.Score)
	}
	if stub.calls != 1 {
		t.Errorf("expected SuggestCapability called once, got %d", stub.calls)
	}
}

// TestRouter_ColdStart_MicroRouterUnsure verifies that when MicroRouter returns
// confident=false, the router falls through to the configured ColdStartStrategy.
func TestRouter_ColdStart_MicroRouterUnsure(t *testing.T) {
	p1 := makeProfile("primary", "prov-a", "model-a", nil)
	p2 := makeProfile("secondary", "prov-b", "model-b", nil)

	stub := &stubMicroRouter{hint: "", confident: false}
	router := &Router{
		Profiles:          []*CapabilityProfile{p1, p2},
		ColdStartStrategy: ColdStartFallbackChain,
		MicroRouter:       stub,
	}

	task := TaskClassification{TaskType: "unclear", Language: "go"}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// fallback chain → first profile wins
	if dec.Harness != "primary" {
		t.Errorf("expected primary (fallback chain), got %q", dec.Harness)
	}
	if stub.calls != 1 {
		t.Errorf("expected SuggestCapability called once, got %d", stub.calls)
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
