package planner

import "testing"

func TestBuildConstraintEnvelopeDefaults(t *testing.T) {
	out, err := BuildConstraintEnvelope(PlanningInput{FeatureText: "Add parallel swarm", Repo: "fall-out-bug/sdp_private", Boundaries: []string{"internal/", "cmd/"}})
	if err != nil {
		t.Fatalf("build envelope: %v", err)
	}
	if out.RiskClass != "medium" || out.Lane != "commit" || out.Model != "glm-5" {
		t.Fatalf("unexpected defaults: %#v", out)
	}
	if len(out.Boundaries) != 2 {
		t.Fatalf("unexpected boundaries: %#v", out.Boundaries)
	}
}

func TestBuildConstraintEnvelopeRequiresInputs(t *testing.T) {
	if _, err := BuildConstraintEnvelope(PlanningInput{Repo: "fall-out-bug/sdp_private"}); err == nil {
		t.Fatal("expected feature_text validation error")
	}
	if _, err := BuildConstraintEnvelope(PlanningInput{FeatureText: "x"}); err == nil {
		t.Fatal("expected repo validation error")
	}
}
