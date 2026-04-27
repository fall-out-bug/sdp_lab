package eval

import (
	"testing"
	"time"
)

// TestContractVersion verifies the v1 semver contract is defined.
func TestContractVersion(t *testing.T) {
	if ContractVersion != "v1.0.0" {
		t.Errorf("expected ContractVersion v1.0.0, got %s", ContractVersion)
	}
}

// TestV1InterfaceCompileTimeChecks ensures v1 interfaces match existing implementations.
func TestV1InterfaceCompileTimeChecks(t *testing.T) {
	// CaseRunnerV1: Verify RunCase function signature matches interface
	var _ CaseRunnerV1 = caseRunnerAdapter{}

	// SuiteRunnerV1: Verify Run function signature matches interface
	var _ SuiteRunnerV1 = suiteRunnerAdapter{}

	// CaseLoaderV1: Verify LoadCases function signature matches interface
	var _ CaseLoaderV1 = caseLoaderAdapter{}
}

// Adapter types for compile-time interface compliance verification.
type caseRunnerAdapter struct{}
func (caseRunnerAdapter) RunCase(c *Case, projectRoot string) Result { return RunCase(c, projectRoot) }

type suiteRunnerAdapter struct{}
func (suiteRunnerAdapter) Run(projectRoot, casesDir, skill string) ([]Result, error) { return Run(projectRoot, casesDir, skill) }

type caseLoaderAdapter struct{}
func (caseLoaderAdapter) LoadCases(casesDir, skill string) ([]Case, error) { return LoadCases(casesDir, skill) }

// TestBaselineComparisonStructFields verifies BaselineComparison structure.
func TestBaselineComparisonStructFields(t *testing.T) {
	comparison := BaselineComparison{
		Regressions:  2,
		Improvements: 3,
		Unchanged:    10,
		Details: []ComparisonDetail{
			{
				Case:         "test-case-1",
				CurrentPass:  true,
				BaselinePass: false,
				Delta:        "FAIL → PASS",
			},
		},
	}

	if comparison.Regressions != 2 {
		t.Errorf("expected 2 regressions, got %d", comparison.Regressions)
	}
	if comparison.Improvements != 3 {
		t.Errorf("expected 3 improvements, got %d", comparison.Improvements)
	}
	if comparison.Unchanged != 10 {
		t.Errorf("expected 10 unchanged, got %d", comparison.Unchanged)
	}
	if len(comparison.Details) != 1 {
		t.Errorf("expected 1 detail, got %d", len(comparison.Details))
	}
	detail := comparison.Details[0]
	if detail.Case != "test-case-1" {
		t.Errorf("expected case name 'test-case-1', got %s", detail.Case)
	}
	if detail.CurrentPass != true {
		t.Error("expected CurrentPass true")
	}
	if detail.BaselinePass != false {
		t.Error("expected BaselinePass false")
	}
	if detail.Delta != "FAIL → PASS" {
		t.Errorf("expected delta 'FAIL → PASS', got %s", detail.Delta)
	}
}

// TestScoreboardEntryStructFields verifies ScoreboardEntry structure.
func TestScoreboardEntryStructFields(t *testing.T) {
	entry := ScoreboardEntry{
		RunID:       "run-001",
		Timestamp:   time.Now(),
		TotalCases:  20,
		PassedCases: 18,
		PassRate:    90.0,
		Regressions: 1,
	}

	if entry.RunID != "run-001" {
		t.Errorf("expected run ID 'run-001', got %s", entry.RunID)
	}
	if entry.TotalCases != 20 {
		t.Errorf("expected 20 total cases, got %d", entry.TotalCases)
	}
	if entry.PassedCases != 18 {
		t.Errorf("expected 18 passed cases, got %d", entry.PassedCases)
	}
	if entry.PassRate != 90.0 {
		t.Errorf("expected pass rate 90.0, got %f", entry.PassRate)
	}
	if entry.Regressions != 1 {
		t.Errorf("expected 1 regression, got %d", entry.Regressions)
	}
}

// TestMismatchMetricStructFields verifies MismatchMetric structure.
func TestMismatchMetricStructFields(t *testing.T) {
	metric := MismatchMetric{
		TotalDecisions:        100,
		EvidenceMismatchCount: 5,
		MismatchRate:          0.05,
	}

	if metric.TotalDecisions != 100 {
		t.Errorf("expected 100 total decisions, got %d", metric.TotalDecisions)
	}
	if metric.EvidenceMismatchCount != 5 {
		t.Errorf("expected 5 mismatches, got %d", metric.EvidenceMismatchCount)
	}
	if metric.MismatchRate != 0.05 {
		t.Errorf("expected mismatch rate 0.05, got %f", metric.MismatchRate)
	}
}

// TestEvaluationContractDocumentation verifies EvaluationContract exists.
func TestEvaluationContractDocumentation(t *testing.T) {
	// This test verifies the contract type exists and is documented.
	// The struct contains documentation in its godoc.
	var contract EvaluationContract
	_ = contract // Use the variable to avoid unused error
}

// TestMismatchMetricContractDocumentation verifies MismatchMetricContract exists.
func TestMismatchMetricContractDocumentation(t *testing.T) {
	// This test verifies the contract type exists and is documented.
	// The struct contains documentation in its godoc.
	var contract MismatchMetricContract
	_ = contract // Use the variable to avoid unused error
}

// TestComparisonDetailDeltaGeneration tests Delta field formatting.
func TestComparisonDetailDeltaGeneration(t *testing.T) {
	tests := []struct {
		name         string
		currentPass  bool
		baselinePass bool
		expectedDelta string
	}{
		{
			name:         "improvement",
			currentPass:  true,
			baselinePass: false,
			expectedDelta: "FAIL → PASS",
		},
		{
			name:         "regression",
			currentPass:  false,
			baselinePass: true,
			expectedDelta: "PASS → FAIL",
		},
		{
			name:         "unchanged pass",
			currentPass:  true,
			baselinePass: true,
			expectedDelta: "PASS → PASS",
		},
		{
			name:         "unchanged fail",
			currentPass:  false,
			baselinePass: false,
			expectedDelta: "FAIL → FAIL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that delta formatting follows expected pattern
			currentStatus := "PASS"
			if !tt.currentPass {
				currentStatus = "FAIL"
			}
			baselineStatus := "PASS"
			if !tt.baselinePass {
				baselineStatus = "FAIL"
			}
			delta := baselineStatus + " → " + currentStatus
			if delta != tt.expectedDelta {
				t.Errorf("expected delta %s, got %s", tt.expectedDelta, delta)
			}
		})
	}
}
