package harness

import "testing"

func baseContract() *TaskContract {
	return &TaskContract{
		Version: "v1",
		RunID:   "run-123",
		AcceptanceCriteria: []AcceptanceCriterion{
			{ID: "AC-1", Statement: "one", Priority: "P1"},
			{ID: "AC-2", Statement: "two", Priority: "P1"},
		},
		RequiredMetrics: []RequiredMetric{
			{Name: "coverage", Direction: "at_least", Target: 80},
			{Name: "test_pass_rate", Direction: "equals", Target: "100%"},
		},
		RequiredEvidence: []string{"test_results", "build_log"},
		QualityGates: QualityGates{
			Build:     true,
			Test:      true,
			Lint:      true,
			Typecheck: true,
		},
		Constraints: Constraints{
			AllowScopeReduction:  false,
			AllowMetricReduction: false,
		},
	}
}

func fullSnapshot() *TaskSnapshot {
	return &TaskSnapshot{
		RunID: "run-123",
		Phase: "validate",
		AcceptanceCriteria: []CriterionStatus{
			{ID: "AC-1", Status: "done"},
			{ID: "AC-2", Status: "done"},
		},
		Metrics: []MetricSnapshot{
			{Name: "coverage", Value: 85},
			{Name: "test_pass_rate", Value: "100%"},
		},
		Evidence: []string{"test_results", "build_log"},
		Claims: []Claim{{
			ID:           "claim-1",
			Statement:    "tests passed",
			EvidenceRefs: []string{"test_results"},
		}},
		QualityResults: map[string]bool{
			"build":     true,
			"test":      true,
			"lint":      true,
			"typecheck": true,
		},
		ProcessReport: ProcessReport{
			ContractCoverageSummary: true,
			GateResults:             true,
			EvidenceIndex:           true,
			DecisionLog:             true,
		},
	}
}

func TestEvaluateCompliancePass(t *testing.T) {
	report := EvaluateCompliance(baseContract(), fullSnapshot())
	if report.Blocked {
		t.Fatalf("expected report not blocked, got %+v", report)
	}
	if len(report.GateResults) != 5 {
		t.Fatalf("expected 5 gate results, got %d", len(report.GateResults))
	}
}

func TestEvaluateComplianceBlocksOnRequirementDrift(t *testing.T) {
	s := fullSnapshot()
	s.AcceptanceCriteria = s.AcceptanceCriteria[:1]
	report := EvaluateCompliance(baseContract(), s)
	if !report.Blocked {
		t.Fatal("expected blocked report")
	}
	found := false
	for _, gate := range report.GateResults {
		if gate.GateID != GateRequirementIntegrity {
			continue
		}
		for _, v := range gate.Violations {
			if v.Type == DriftACDrop {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected DriftACDrop violation")
	}
}

func TestEvaluateComplianceBlocksOnUnsupportedClaim(t *testing.T) {
	s := fullSnapshot()
	s.Claims = []Claim{{ID: "claim-2", Statement: "done"}}
	report := EvaluateCompliance(baseContract(), s)
	if !report.Blocked {
		t.Fatal("expected blocked report")
	}
	found := false
	for _, gate := range report.GateResults {
		if gate.GateID != GateEvidence {
			continue
		}
		for _, v := range gate.Violations {
			if v.Type == DriftUnsupportedClaim {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected DriftUnsupportedClaim violation")
	}
}
