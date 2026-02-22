package evaluator

import "testing"

func TestDefaultPeriodicComponentAuditProtocolDeterministicAndComplete(t *testing.T) {
	protocol := DefaultPeriodicComponentAuditProtocol()

	if protocol.ContractVersion != PeriodicComponentAuditProtocolContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", protocol.ContractVersion, PeriodicComponentAuditProtocolContractVersion)
	}
	if protocol.Cadence != "weekly-or-change-triggered" {
		t.Fatalf("unexpected cadence: %s", protocol.Cadence)
	}
	if protocol.EscalationPath == "" {
		t.Fatal("expected escalation path to be set")
	}
	if len(protocol.RequiredInputs) != 6 {
		t.Fatalf("expected 6 required inputs, got %d", len(protocol.RequiredInputs))
	}
	if len(protocol.DecisionCheckpoints) != 4 {
		t.Fatalf("expected 4 decision checkpoints, got %d", len(protocol.DecisionCheckpoints))
	}

	for i, checkpoint := range protocol.DecisionCheckpoints {
		if checkpoint.ID == "" || checkpoint.DecisionQuestion == "" || checkpoint.PassCondition == "" {
			t.Fatalf("checkpoint %d missing required fields: %+v", i, checkpoint)
		}
		if len(checkpoint.RequiredInputs) == 0 {
			t.Fatalf("checkpoint %s requires at least one input", checkpoint.ID)
		}
		if len(checkpoint.EscalationOnFailure) == 0 {
			t.Fatalf("checkpoint %s requires escalation targets", checkpoint.ID)
		}
		if i > 0 && protocol.DecisionCheckpoints[i-1].ID > checkpoint.ID {
			t.Fatalf("decision checkpoints are not sorted: %q before %q", protocol.DecisionCheckpoints[i-1].ID, checkpoint.ID)
		}
	}
}

func TestEvaluatePeriodicAuditDecisionPassesWhenInputsAndCheckpointsPass(t *testing.T) {
	protocol := DefaultPeriodicComponentAuditProtocol()

	providedInputs := map[string]bool{}
	for _, input := range protocol.RequiredInputs {
		providedInputs[input] = true
	}

	checkpointPass := map[string]bool{}
	for _, checkpoint := range protocol.DecisionCheckpoints {
		checkpointPass[checkpoint.ID] = true
	}

	result := EvaluatePeriodicAuditDecision(protocol, providedInputs, checkpointPass)
	if !result.Passed {
		t.Fatalf("expected pass result, got missing=%v failed=%v escalation=%v", result.MissingInputs, result.FailedCheckpointIDs, result.EscalationTargets)
	}
	if result.EscalationRequired {
		t.Fatal("expected escalation not required on passing protocol decision")
	}
	if len(result.MissingInputs) != 0 || len(result.FailedCheckpointIDs) != 0 || len(result.EscalationTargets) != 0 {
		t.Fatalf("expected empty missing/failed/escalation lists, got %+v", result)
	}
}

func TestEvaluatePeriodicAuditDecisionReportsMissingInputsAndEscalations(t *testing.T) {
	protocol := DefaultPeriodicComponentAuditProtocol()

	result := EvaluatePeriodicAuditDecision(
		protocol,
		map[string]bool{
			"component-boundary-map": false,
			"component-change-log":   true,
			"dependency-risk-delta":  true,
			"incident-signal-digest": false,
			"outcome-metric-delta":   true,
			"persona-score-report":   true,
		},
		map[string]bool{
			"decision-gate": true,
			"entry-gate":    true,
			"evidence-gate": false,
			"handoff-gate":  false,
		},
	)

	if result.Passed {
		t.Fatal("expected failure when inputs/checkpoints are missing")
	}
	if !result.EscalationRequired {
		t.Fatal("expected escalation to be required when protocol fails")
	}

	wantMissingInputs := []string{"component-boundary-map", "incident-signal-digest"}
	if len(result.MissingInputs) != len(wantMissingInputs) {
		t.Fatalf("missing input count mismatch: got=%v want=%v", result.MissingInputs, wantMissingInputs)
	}
	for i := range wantMissingInputs {
		if result.MissingInputs[i] != wantMissingInputs[i] {
			t.Fatalf("missing input mismatch at %d: got=%q want=%q", i, result.MissingInputs[i], wantMissingInputs[i])
		}
	}

	wantFailed := []string{"evidence-gate", "handoff-gate"}
	if len(result.FailedCheckpointIDs) != len(wantFailed) {
		t.Fatalf("failed checkpoint count mismatch: got=%v want=%v", result.FailedCheckpointIDs, wantFailed)
	}
	for i := range wantFailed {
		if result.FailedCheckpointIDs[i] != wantFailed[i] {
			t.Fatalf("failed checkpoint mismatch at %d: got=%q want=%q", i, result.FailedCheckpointIDs[i], wantFailed[i])
		}
	}

	wantEscalations := []string{"dx-expert", "product-strategist", "security-reviewer", "sre"}
	if len(result.EscalationTargets) != len(wantEscalations) {
		t.Fatalf("escalation target count mismatch: got=%v want=%v", result.EscalationTargets, wantEscalations)
	}
	for i := range wantEscalations {
		if result.EscalationTargets[i] != wantEscalations[i] {
			t.Fatalf("escalation mismatch at %d: got=%q want=%q", i, result.EscalationTargets[i], wantEscalations[i])
		}
	}
}
