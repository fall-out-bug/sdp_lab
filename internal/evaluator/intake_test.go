package evaluator

import "testing"

func TestBaselineScopeSortedAndComplete(t *testing.T) {
	scope := BaselineScope()
	if len(scope) != 4 {
		t.Fatalf("expected 4 baseline scope components, got %d", len(scope))
	}

	seen := map[string]struct{}{}
	for i, component := range scope {
		if component.ID == "" || component.Description == "" || component.OwnerSignal == "" || component.EvidenceClass == "" {
			t.Fatalf("scope component %d has empty required field: %+v", i, component)
		}
		if _, exists := seen[component.ID]; exists {
			t.Fatalf("duplicate scope component id %q", component.ID)
		}
		seen[component.ID] = struct{}{}
		if i > 0 && scope[i-1].ID > component.ID {
			t.Fatalf("scope components are not sorted: %q before %q", scope[i-1].ID, component.ID)
		}
	}
}

func TestEvaluateHappyPathOperabilityPass(t *testing.T) {
	checks := HappyPathOperabilityChecks()
	signals := make(map[string]bool, len(checks))
	for _, check := range checks {
		signals[check.ID] = true
	}

	result := EvaluateHappyPathOperability(signals)
	if !result.Passed {
		t.Fatalf("expected pass, got missing checks: %v", result.MissingCheckIDs)
	}
	if len(result.MissingCheckIDs) != 0 {
		t.Fatalf("expected no missing checks, got %v", result.MissingCheckIDs)
	}
}

func TestEvaluateHappyPathOperabilityReportsMissingInDeterministicOrder(t *testing.T) {
	result := EvaluateHappyPathOperability(map[string]bool{
		"issue-selected":              true,
		"dependencies-clear":          false,
		"scope-baseline-defined":      false,
		"gate-command-declared":       true,
		"callback-contract-available": false,
	})

	if result.Passed {
		t.Fatalf("expected fail when checks are missing")
	}
	want := []string{"dependencies-clear", "scope-baseline-defined", "callback-contract-available"}
	if len(result.MissingCheckIDs) != len(want) {
		t.Fatalf("missing checks length mismatch: got %v want %v", result.MissingCheckIDs, want)
	}
	for i := range want {
		if result.MissingCheckIDs[i] != want[i] {
			t.Fatalf("missing checks mismatch at %d: got %q want %q", i, result.MissingCheckIDs[i], want[i])
		}
	}
}
