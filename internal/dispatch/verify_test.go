package dispatch

import (
	"context"
	"testing"

	"sdp_dev/internal/dispatch/harness"
)

func TestVerificationRouter_SelectsAlternative(t *testing.T) {
	buildProfile := makeProfile("claude", "anthropic", "claude-3", featureCaps(0.92))
	verifyProfile := makeProfile("opencode", "omo", "default", map[string]CapabilityScore{
		"review:go": {TestPassRate: 0.88},
	})

	vr := &VerificationRouter{Profiles: []*CapabilityProfile{buildProfile, verifyProfile}}

	dec, err := vr.RouteVerification(context.Background(), TaskClassification{
		TaskType: "review",
		Language: "go",
	}, "claude", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec == nil {
		t.Fatal("expected non-nil decision")
		return
	}
	if dec.Harness == "claude" {
		t.Errorf("verification harness should NOT be the build harness, got %q", dec.Harness)
	}
	if dec.Harness != "opencode" {
		t.Errorf("expected opencode, got %q", dec.Harness)
	}
}

func TestVerificationRouter_NoAlternative(t *testing.T) {
	// Only one profile — no alternative exists.
	buildProfile := makeProfile("claude", "anthropic", "claude-3", featureCaps(0.92))

	vr := &VerificationRouter{Profiles: []*CapabilityProfile{buildProfile}}

	dec, err := vr.RouteVerification(context.Background(), TaskClassification{
		TaskType: "review",
		Language: "go",
	}, "claude", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec != nil {
		t.Errorf("expected nil decision when no alternative, got %+v", dec)
	}
}

func TestVerificationRouter_RespectsLimits(t *testing.T) {
	buildProfile := makeProfile("claude", "anthropic", "claude-3", featureCaps(0.92))
	verifyProfile := makeProfile("opencode", "omo", "default", featureCaps(0.88))

	vr := &VerificationRouter{Profiles: []*CapabilityProfile{buildProfile, verifyProfile}}

	// Exhaust opencode provider
	limits := map[string]*harness.Limits{
		"omo": {Total: 100, Used: 100}, // 100% → AvailabilityFactor = 0.0
	}

	dec, err := vr.RouteVerification(context.Background(), TaskClassification{
		TaskType: "feature",
		Language: "go",
	}, "claude", limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// opencode is exhausted → score = 0.88 * 0.0 = 0, cold-start gives averageTestPassRate
	// which is still > 0, so it should be selected via cold-start
	if dec == nil {
		t.Fatal("expected non-nil decision (cold-start should kick in)")
		return
	}
	if dec.Harness != "opencode" {
		t.Errorf("expected opencode, got %q", dec.Harness)
	}
}

func TestVerificationRouter_MultipleAlternatives(t *testing.T) {
	buildProfile := makeProfile("claude", "anthropic", "claude-3", featureCaps(0.92))
	okProfile := makeProfile("opencode", "omo", "default", featureCaps(0.75))
	bestProfile := makeProfile("codex", "openai", "gpt-4o", featureCaps(0.90))

	vr := &VerificationRouter{Profiles: []*CapabilityProfile{buildProfile, okProfile, bestProfile}}

	dec, err := vr.RouteVerification(context.Background(), TaskClassification{
		TaskType: "feature",
		Language: "go",
	}, "claude", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec == nil {
		t.Fatal("expected non-nil decision")
		return
	}
	if dec.Harness != "codex" {
		t.Errorf("expected codex (highest score), got %q", dec.Harness)
	}
	if len(dec.Alternatives) != 1 {
		t.Errorf("expected 1 alternative, got %d", len(dec.Alternatives))
	}
}

func TestVerificationRouter_EmptyProfiles(t *testing.T) {
	vr := &VerificationRouter{Profiles: nil}

	dec, err := vr.RouteVerification(context.Background(), TaskClassification{
		TaskType: "review",
		Language: "go",
	}, "claude", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec != nil {
		t.Errorf("expected nil with empty profiles, got %+v", dec)
	}
}
