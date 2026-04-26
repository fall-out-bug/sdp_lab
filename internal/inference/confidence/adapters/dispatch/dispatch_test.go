package dispatch_test

import (
	"context"
	"strings"
	"testing"

	"sdp_dev/internal/inference/confidence"
	"sdp_dev/internal/inference/confidence/adapters/dispatch"
)

func TestNewRequiresHarnesses(t *testing.T) {
	if _, err := dispatch.New(dispatch.Options{}); err == nil {
		t.Error("expected error when AllowedHarnesses empty")
	}
}

func TestVerifyAllowedHarnessOK(t *testing.T) {
	checker, err := dispatch.New(dispatch.Options{
		AllowedHarnesses: []string{"claude-code", "opencode"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	d := dispatch.Decision{Harness: "claude-code", Agent: "implementer", Confidence: 0.9}
	raw := `{"harness":"claude-code","agent":"implementer","self_score":0.85}`
	res, err := dispatch.Verify(context.Background(), checker, d, raw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if res.Status != confidence.StatusOK {
		t.Errorf("Status = %q, want OK; reasons=%v", res.Status, res.Reasons)
	}
	if got := res.SubScores["self_check"]; got != 0.85 {
		t.Errorf("self_check subscore = %v, want 0.85", got)
	}
}

func TestVerifyDisallowedHarnessFails(t *testing.T) {
	checker, _ := dispatch.New(dispatch.Options{AllowedHarnesses: []string{"opencode"}})
	d := dispatch.Decision{Harness: "rogue-tool", Agent: "x", Confidence: 1.0}
	raw := `{"harness":"rogue-tool"}`
	res, _ := dispatch.Verify(context.Background(), checker, d, raw)
	joined := strings.Join(res.Reasons, "|")
	if !strings.Contains(joined, "harness-allowed") {
		t.Errorf("Reasons missing harness-allowed violation: %v", res.Reasons)
	}
}

func TestVerifyMissingSelfScoreLiteNeutral(t *testing.T) {
	checker, _ := dispatch.New(dispatch.Options{AllowedHarnesses: []string{"claude-code"}})
	d := dispatch.Decision{Harness: "claude-code", Agent: "x", Confidence: 0.9}
	raw := `{"harness":"claude-code"}` // no self_score
	res, _ := dispatch.Verify(context.Background(), checker, d, raw)
	if got := res.SubScores["self_check"]; got != 0.5 {
		t.Errorf("self_check subscore = %v, want 0.5 (neutral on missing annotation)", got)
	}
}

func TestVerifyLooseAnnotationParsed(t *testing.T) {
	checker, _ := dispatch.New(dispatch.Options{AllowedHarnesses: []string{"claude-code"}})
	d := dispatch.Decision{Harness: "claude-code", Agent: "x", Confidence: 0.9}
	raw := "Decision: claude-code/implementer\nself_score: 0.72"
	res, _ := dispatch.Verify(context.Background(), checker, d, raw)
	if got := res.SubScores["self_check"]; got != 0.72 {
		t.Errorf("self_check subscore = %v, want 0.72", got)
	}
}

func TestVerifySelfConfidenceOutOfRange(t *testing.T) {
	checker, _ := dispatch.New(dispatch.Options{AllowedHarnesses: []string{"claude-code"}})
	d := dispatch.Decision{Harness: "claude-code", Agent: "x", Confidence: 1.5}
	raw := `{"self_score":0.9}`
	res, _ := dispatch.Verify(context.Background(), checker, d, raw)
	joined := strings.Join(res.Reasons, "|")
	if !strings.Contains(joined, "self-confidence-range") {
		t.Errorf("Reasons missing self-confidence-range violation: %v", res.Reasons)
	}
}

func TestAllowedAgentsConstraintOff(t *testing.T) {
	// Empty AllowedAgents → invariant always passes.
	checker, _ := dispatch.New(dispatch.Options{AllowedHarnesses: []string{"claude-code"}})
	d := dispatch.Decision{Harness: "claude-code", Agent: "anything", Confidence: 0.9}
	raw := `{"self_score":0.9}`
	res, _ := dispatch.Verify(context.Background(), checker, d, raw)
	if res.Status != confidence.StatusOK {
		t.Errorf("Status = %q, want OK; reasons=%v", res.Status, res.Reasons)
	}
}

func TestAllowedAgentsConstraintOn(t *testing.T) {
	checker, _ := dispatch.New(dispatch.Options{
		AllowedHarnesses: []string{"claude-code"},
		AllowedAgents:    []string{"implementer", "reviewer"},
	})
	d := dispatch.Decision{Harness: "claude-code", Agent: "rogue", Confidence: 0.9}
	raw := `{"self_score":0.9}`
	res, _ := dispatch.Verify(context.Background(), checker, d, raw)
	joined := strings.Join(res.Reasons, "|")
	if !strings.Contains(joined, "agent-allowed") {
		t.Errorf("Reasons missing agent-allowed violation: %v", res.Reasons)
	}
}
