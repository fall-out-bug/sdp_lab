package artifact

import (
	"reflect"
	"testing"
)

func TestTransitionControllerAllowsValidTransition(t *testing.T) {
	bus := NewBusService()
	issueID := "sdp_dev-2aq.14.3.valid"

	_, err := bus.Ingest(IngestRequest{
		IssueID:       issueID,
		ArtifactID:    "intent-001",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2026-02-20T20:00:00Z",
		Payload:       map[string]any{"trigger": "agent"},
	})
	if err != nil {
		t.Fatalf("ingest intent-brief: %v", err)
	}

	controller := NewDefaultTransitionController(bus)
	decision := controller.EvaluateTransition(TransitionRequest{
		IssueID:   issueID,
		FromPhase: "intake",
		ToPhase:   "plan",
		GateSignals: map[string]GateSignalStatus{
			"intake:issue-scoped":  GateSignalStatusPass,
			"intake:risk-assessed": GateSignalStatusPass,
		},
	})

	if !decision.Allowed {
		t.Fatalf("expected transition allowed, got decision %+v", decision)
	}
	if len(decision.ReasonCodes) != 0 {
		t.Fatalf("expected no denial reasons, got %#v", decision.ReasonCodes)
	}
	if len(decision.GateDecisions) != 2 {
		t.Fatalf("expected two gate decisions, got %#v", decision.GateDecisions)
	}
	for _, gate := range decision.GateDecisions {
		if !gate.Passed {
			t.Fatalf("expected gate %s to pass, got %+v", gate.SignalID, gate)
		}
	}
}

func TestTransitionControllerDeniesMissingArtifactsAndGateStatus(t *testing.T) {
	bus := NewBusService()
	issueID := "sdp_dev-2aq.14.3.denied"

	controller := NewDefaultTransitionController(bus)
	decision := controller.EvaluateTransition(TransitionRequest{
		IssueID:   issueID,
		FromPhase: "intake",
		ToPhase:   "plan",
		GateSignals: map[string]GateSignalStatus{
			"intake:issue-scoped": GateSignalStatusFail,
		},
	})

	if decision.Allowed {
		t.Fatalf("expected transition denied, got decision %+v", decision)
	}
	if !reflect.DeepEqual(decision.ReasonCodes, []string{
		"transition-intake-to-plan-missing-gate-signal",
		"transition-intake-to-plan-gate-not-passed",
		"transition-intake-to-plan-missing-artifact",
		"transition-intake-to-plan-missing-provenance-key",
	}) {
		t.Fatalf("unexpected denial reasons: got %#v", decision.ReasonCodes)
	}

	if len(decision.GateDecisions) != 2 {
		t.Fatalf("expected two gate decisions, got %#v", decision.GateDecisions)
	}
	if decision.GateDecisions[0].SignalID != "intake:issue-scoped" || decision.GateDecisions[0].Status != GateSignalStatusFail || decision.GateDecisions[0].Passed {
		t.Fatalf("unexpected first gate decision: %+v", decision.GateDecisions[0])
	}
	if decision.GateDecisions[1].SignalID != "intake:risk-assessed" || decision.GateDecisions[1].Status != GateSignalStatusMissing || decision.GateDecisions[1].Passed {
		t.Fatalf("unexpected second gate decision: %+v", decision.GateDecisions[1])
	}
}

func TestTransitionControllerReasonCodesAreDeterministic(t *testing.T) {
	bus := NewBusService()
	issueID := "sdp_dev-2aq.14.3.deterministic"

	controller := NewDefaultTransitionController(bus)
	first := controller.EvaluateTransition(TransitionRequest{
		IssueID:   issueID,
		FromPhase: "verify",
		ToPhase:   "review",
		GateSignals: map[string]GateSignalStatus{
			"verify:tests-passed":           GateSignalStatusPass,
			"verify:boundary-ok":            GateSignalStatusMissing,
			"verify:evidence-contract-pass": GateSignalStatusFail,
		},
	})
	second := controller.EvaluateTransition(TransitionRequest{
		IssueID:   issueID,
		FromPhase: "verify",
		ToPhase:   "review",
		GateSignals: map[string]GateSignalStatus{
			"verify:evidence-contract-pass": GateSignalStatusFail,
			"verify:tests-passed":           GateSignalStatusPass,
			"verify:boundary-ok":            GateSignalStatusMissing,
		},
	})

	want := []string{
		"transition-verify-to-review-missing-gate-signal",
		"transition-verify-to-review-gate-not-passed",
		"transition-verify-to-review-missing-artifact",
		"transition-verify-to-review-missing-provenance-key",
	}
	if !reflect.DeepEqual(first.ReasonCodes, want) {
		t.Fatalf("unexpected first denial reasons: got %#v want %#v", first.ReasonCodes, want)
	}
	if !reflect.DeepEqual(second.ReasonCodes, want) {
		t.Fatalf("unexpected second denial reasons: got %#v want %#v", second.ReasonCodes, want)
	}
	if !reflect.DeepEqual(first.ReasonCodes, second.ReasonCodes) {
		t.Fatalf("reason codes are not deterministic: first=%#v second=%#v", first.ReasonCodes, second.ReasonCodes)
	}
}

func TestTransitionControllerGateDecisionTraceIncludesUnknownStatus(t *testing.T) {
	bus := NewBusService()
	issueID := "sdp_dev-2aq.14.2.trace"

	_, err := bus.Ingest(IngestRequest{
		IssueID:       issueID,
		ArtifactID:    "intent-002",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2026-02-20T20:05:00Z",
		Payload:       map[string]any{"trigger": "manual"},
	})
	if err != nil {
		t.Fatalf("ingest intent-brief: %v", err)
	}

	controller := NewDefaultTransitionController(bus)
	decision := controller.EvaluateTransition(TransitionRequest{
		IssueID:   issueID,
		FromPhase: "intake",
		ToPhase:   "plan",
		GateSignals: map[string]GateSignalStatus{
			"intake:issue-scoped":  GateSignalStatusPass,
			"intake:risk-assessed": GateSignalStatus("unknown-status"),
		},
	})

	if decision.Allowed {
		t.Fatalf("expected transition denied when unknown status is present, got %+v", decision)
	}
	if !reflect.DeepEqual(decision.ReasonCodes, []string{"transition-intake-to-plan-gate-not-passed"}) {
		t.Fatalf("unexpected denial reasons for unknown status: %#v", decision.ReasonCodes)
	}
	if len(decision.GateDecisions) != 2 {
		t.Fatalf("expected two gate decisions, got %#v", decision.GateDecisions)
	}
	if decision.GateDecisions[1].SignalID != "intake:risk-assessed" || decision.GateDecisions[1].Status != GateSignalStatus("unknown-status") || decision.GateDecisions[1].Passed {
		t.Fatalf("unexpected gate trace for unknown status: %+v", decision.GateDecisions[1])
	}
}
