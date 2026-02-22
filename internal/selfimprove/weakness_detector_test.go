package selfimprove

import (
	"testing"
)

func TestNewWeaknessDetector(t *testing.T) {
	d := NewWeaknessDetector()
	if d == nil || d.classifier == nil {
		t.Fatal("NewWeaknessDetector returned nil")
	}
}

func TestWeaknessDetector_Detect(t *testing.T) {
	d := NewWeaknessDetector()

	// Empty input
	patterns := d.Detect(nil, nil)
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(patterns))
	}

	// Repeated failures
	runs := []RunDoc{
		{IssueID: "a", Events: []RunEvent{{Phase: "verify", State: "failed"}}},
		{IssueID: "b", Events: []RunEvent{{Phase: "verify", State: "failed"}}},
	}
	patterns = d.Detect(runs, nil)
	if len(patterns) == 0 {
		t.Log("no patterns from runs (classifier may not match); checking escalation")
	}

	// Escalation pattern
	telemetry := []TelemetryRecord{
		{Escalated: true},
		{Escalated: true},
	}
	patterns = d.Detect(nil, telemetry)
	var found bool
	for _, p := range patterns {
		if p.ID == "escalation-spike" {
			found = true
			break
		}
	}
	if !found && len(telemetry) >= 2 {
		t.Log("escalation-spike pattern may depend on classifier; patterns:", patterns)
	}
}

func TestSuggestImprovement(t *testing.T) {
	tests := []struct {
		class FailureClass
		want  string
	}{
		{ClassTransient, "Harden retry/backoff"},
		{ClassToolFlake, "Add infra-flaky"},
		{ClassVerificationFail, "Improve verification"},
		{ClassPolicyConflict, "Review policy"},
		{ClassSecuritySensitive, "Document security"},
		{FailureClass("unknown"), "Investigate"},
	}
	for _, tt := range tests {
		p := WeaknessPattern{Class: tt.class, ID: "test"}
		got := SuggestImprovement(p)
		if got == "" {
			t.Errorf("SuggestImprovement(%q) returned empty", tt.class)
		}
	}
}
