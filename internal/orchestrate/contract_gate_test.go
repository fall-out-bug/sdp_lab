package orchestrate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/harness"
)

func TestEnforceContractGateNoContract(t *testing.T) {
	report, err := EnforceContractGate(t.TempDir(), "F123")
	if err != nil {
		t.Fatalf("expected no error when contract is absent: %v", err)
	}
	if report != nil {
		t.Fatal("expected nil report when contract is absent")
	}
}

func TestEnforceContractGateBlocked(t *testing.T) {
	root := t.TempDir()
	contractPath, snapshotPath, reportPath, err := ContractPaths(root, "F123")
	if err != nil {
		t.Fatalf("contract paths: %v", err)
	}

	contract := &harness.TaskContract{
		Version: "v1",
		RunID:   "run-1",
		AcceptanceCriteria: []harness.AcceptanceCriterion{
			{ID: "AC-1", Statement: "must be present", Priority: "P1"},
		},
		RequiredMetrics:  []harness.RequiredMetric{{Name: "coverage", Direction: "at_least", Target: 80}},
		RequiredEvidence: []string{"test_results"},
		QualityGates:     harness.QualityGates{Build: true, Test: true},
		Constraints:      harness.Constraints{AllowScopeReduction: false, AllowMetricReduction: false},
	}
	if err := harness.SaveTaskContract(contractPath, contract); err != nil {
		t.Fatalf("save contract: %v", err)
	}

	snapshot := harness.TaskSnapshot{
		RunID:              "run-1",
		Phase:              "validate",
		AcceptanceCriteria: []harness.CriterionStatus{},
		Metrics:            []harness.MetricSnapshot{},
		Evidence:           []string{},
		QualityResults:     map[string]bool{"build": false, "test": false},
		ProcessReport:      harness.ProcessReport{},
	}
	if err := writeJSON(snapshotPath, snapshot); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	report, err := EnforceContractGate(root, "F123")
	if err == nil {
		t.Fatal("expected contract gate to block")
	}
	if !errors.Is(err, ErrContractGateBlocked) {
		t.Fatalf("expected ErrContractGateBlocked, got: %v", err)
	}
	if report == nil || !report.Blocked {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected report file to be written: %v", err)
	}
}

func TestEnforceContractGatePass(t *testing.T) {
	root := t.TempDir()
	contractPath, snapshotPath, _, err := ContractPaths(root, "F123")
	if err != nil {
		t.Fatalf("contract paths: %v", err)
	}

	contract := &harness.TaskContract{
		Version: "v1",
		RunID:   "run-1",
		AcceptanceCriteria: []harness.AcceptanceCriterion{
			{ID: "AC-1", Statement: "must be present", Priority: "P1"},
		},
		RequiredMetrics:  []harness.RequiredMetric{{Name: "coverage", Direction: "at_least", Target: 80}},
		RequiredEvidence: []string{"test_results"},
		QualityGates:     harness.QualityGates{Build: true, Test: true, Lint: true, Typecheck: true},
		Constraints:      harness.Constraints{AllowScopeReduction: false, AllowMetricReduction: false},
	}
	if err := harness.SaveTaskContract(contractPath, contract); err != nil {
		t.Fatalf("save contract: %v", err)
	}

	snapshot := harness.TaskSnapshot{
		RunID: "run-1",
		Phase: "validate",
		AcceptanceCriteria: []harness.CriterionStatus{
			{ID: "AC-1", Status: "done"},
		},
		Metrics: []harness.MetricSnapshot{{Name: "coverage", Value: 85}},
		Evidence: []string{
			"test_results",
		},
		QualityResults: map[string]bool{
			"build":     true,
			"test":      true,
			"lint":      true,
			"typecheck": true,
		},
		ProcessReport: harness.ProcessReport{
			ContractCoverageSummary: true,
			GateResults:             true,
			EvidenceIndex:           true,
			DecisionLog:             true,
		},
		Claims: []harness.Claim{{
			ID:           "C1",
			Statement:    "validated",
			EvidenceRefs: []string{"test_results"},
		}},
	}
	if err := writeJSON(snapshotPath, snapshot); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	report, err := EnforceContractGate(root, "F123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if report == nil || report.Blocked {
		t.Fatalf("expected passing report, got %#v", report)
	}
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
