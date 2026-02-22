package evaluator

import "testing"

func TestDefaultDeepThinkingSwarmPlanDeterministicAndComplete(t *testing.T) {
	plan := DefaultDeepThinkingSwarmPlan()

	if plan.ContractVersion != DeepThinkingSwarmPlanContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", plan.ContractVersion, DeepThinkingSwarmPlanContractVersion)
	}
	if plan.Cadence != "weekly-or-change-triggered" {
		t.Fatalf("unexpected cadence: %s", plan.Cadence)
	}
	if len(plan.TriggerSignals) != 5 {
		t.Fatalf("expected 5 trigger signals, got %d", len(plan.TriggerSignals))
	}
	if len(plan.Roles) != 5 {
		t.Fatalf("expected 5 persona roles, got %d", len(plan.Roles))
	}
	if len(plan.Phases) != 5 {
		t.Fatalf("expected 5 phases, got %d", len(plan.Phases))
	}

	for i, role := range plan.Roles {
		if role.ID == "" || role.DecisionLens == "" || role.PrimaryQuestion == "" || role.EscalationTarget == "" {
			t.Fatalf("role %d has empty required field: %+v", i, role)
		}
		if len(role.RequiredEvidence) == 0 {
			t.Fatalf("role %s missing required evidence", role.ID)
		}
		if i > 0 && plan.Roles[i-1].ID > role.ID {
			t.Fatalf("roles are not sorted: %q before %q", plan.Roles[i-1].ID, role.ID)
		}
	}

	for i, phase := range plan.Phases {
		if phase.ID == "" || phase.Objective == "" {
			t.Fatalf("phase %d has empty required field: %+v", i, phase)
		}
		if len(phase.RequiredSignals) == 0 || len(phase.OutputArtifacts) == 0 {
			t.Fatalf("phase %s missing signals/artifacts", phase.ID)
		}
		if i > 0 && plan.Phases[i-1].ID > phase.ID {
			t.Fatalf("phases are not sorted: %q before %q", plan.Phases[i-1].ID, phase.ID)
		}
	}
}

func TestEvaluateSwarmPlanReadinessPass(t *testing.T) {
	plan := DefaultDeepThinkingSwarmPlan()

	triggerSignals := map[string]bool{}
	for _, signal := range plan.TriggerSignals {
		triggerSignals[signal] = true
	}

	roleSignals := map[string]bool{}
	for _, role := range plan.Roles {
		roleSignals[role.ID] = true
	}

	phaseSignals := map[string]bool{}
	for _, phase := range plan.Phases {
		phaseSignals[phase.ID] = true
	}

	result := EvaluateSwarmPlanReadiness(triggerSignals, roleSignals, phaseSignals)
	if !result.Passed {
		t.Fatalf("expected readiness pass, got missing triggers=%v roles=%v phases=%v", result.MissingTriggerSignals, result.MissingRoleIDs, result.MissingPhaseIDs)
	}
	if len(result.MissingTriggerSignals) != 0 || len(result.MissingRoleIDs) != 0 || len(result.MissingPhaseIDs) != 0 {
		t.Fatalf("expected empty missing lists, got %+v", result)
	}
}

func TestEvaluateSwarmPlanReadinessReportsDeterministicMissing(t *testing.T) {
	result := EvaluateSwarmPlanReadiness(
		map[string]bool{
			"callback-contract-available": false,
			"dependencies-clear":          true,
			"gate-command-declared":       true,
			"issue-selected":              true,
			"scope-baseline-defined":      false,
		},
		map[string]bool{
			"dx-expert":          true,
			"product-strategist": false,
			"security-reviewer":  true,
			"sre":                false,
			"systems-architect":  true,
		},
		map[string]bool{
			"adversarial-review":  true,
			"consensus-synthesis": false,
			"framing":             true,
			"persona-analysis":    false,
			"publish-handoff":     true,
		},
	)

	if result.Passed {
		t.Fatal("expected readiness failure when required signals are missing")
	}

	wantTriggers := []string{"callback-contract-available", "scope-baseline-defined"}
	if len(result.MissingTriggerSignals) != len(wantTriggers) {
		t.Fatalf("missing trigger count mismatch: got=%v want=%v", result.MissingTriggerSignals, wantTriggers)
	}
	for i := range wantTriggers {
		if result.MissingTriggerSignals[i] != wantTriggers[i] {
			t.Fatalf("missing trigger mismatch at %d: got=%q want=%q", i, result.MissingTriggerSignals[i], wantTriggers[i])
		}
	}

	wantRoles := []string{"product-strategist", "sre"}
	if len(result.MissingRoleIDs) != len(wantRoles) {
		t.Fatalf("missing role count mismatch: got=%v want=%v", result.MissingRoleIDs, wantRoles)
	}
	for i := range wantRoles {
		if result.MissingRoleIDs[i] != wantRoles[i] {
			t.Fatalf("missing role mismatch at %d: got=%q want=%q", i, result.MissingRoleIDs[i], wantRoles[i])
		}
	}

	wantPhases := []string{"consensus-synthesis", "persona-analysis"}
	if len(result.MissingPhaseIDs) != len(wantPhases) {
		t.Fatalf("missing phase count mismatch: got=%v want=%v", result.MissingPhaseIDs, wantPhases)
	}
	for i := range wantPhases {
		if result.MissingPhaseIDs[i] != wantPhases[i] {
			t.Fatalf("missing phase mismatch at %d: got=%q want=%q", i, result.MissingPhaseIDs[i], wantPhases[i])
		}
	}
}
