package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareAttestations_MissingAgent(t *testing.T) {
	dir := t.TempDir()

	// Create only CI attestation
	ciStmt := testStatement()
	ciPath := filepath.Join(dir, "ci-auto-test.json")
	b, _ := json.Marshal(ciStmt)
	os.WriteFile(ciPath, b, 0o644)

	report, err := CompareAttestations("test", CompareOptions{EvidenceDir: dir})
	if err != nil {
		t.Fatalf("CompareAttestations error: %v", err)
	}

	if report.OK {
		t.Error("expected report.OK = false for missing agent attestation")
	}

	found := false
	for _, d := range report.Discrepancies {
		if d.Type == DiscrepancyMissingAgent {
			found = true
			if d.Severity != "high" {
				t.Errorf("missing agent severity = %q, want high", d.Severity)
			}
		}
	}
	if !found {
		t.Error("expected DiscrepancyMissingAgent in report")
	}
}

func TestCompareAttestations_MissingCI(t *testing.T) {
	dir := t.TempDir()

	// Create only agent attestation
	agentStmt := testStatement()
	agentPath := filepath.Join(dir, "run-test.json")
	b, _ := json.Marshal(agentStmt)
	os.WriteFile(agentPath, b, 0o644)

	report, err := CompareAttestations("test", CompareOptions{EvidenceDir: dir})
	if err != nil {
		t.Fatalf("CompareAttestations error: %v", err)
	}

	// Missing CI is medium severity, not critical
	found := false
	for _, d := range report.Discrepancies {
		if d.Type == DiscrepancyMissingCI {
			found = true
		}
	}
	if !found {
		t.Error("expected DiscrepancyMissingCI in report")
	}
}

func TestFindAttestation_ThirdFallbackReturnsMatch(t *testing.T) {
	dir := t.TempDir()

	first := filepath.Join(dir, "run-alpha.json")
	second := filepath.Join(dir, "run-zeta.json")
	if err := os.WriteFile(first, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	if err := os.WriteFile(second, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write second file: %v", err)
	}

	got := findAttestation(dir, "does-not-exist", "run-")
	if got == "" {
		t.Fatal("expected fallback attestation path, got empty string")
	}

	if got != second {
		t.Fatalf("fallback path = %q, want %q", got, second)
	}
}

func TestCompareFileScope_Identical(t *testing.T) {
	agent := testStatement()
	agent.Predicate.Execution.ChangedFiles = []string{"a.go", "b.go"}

	ci := testStatement()
	ci.Predicate.Execution.ChangedFiles = []string{"a.go", "b.go"}

	discrepancies := compareFileScope(agent, ci)
	if len(discrepancies) != 0 {
		t.Errorf("expected no discrepancies for identical file sets, got %d", len(discrepancies))
	}
}

func TestCompareFileScope_AgentOnly(t *testing.T) {
	agent := testStatement()
	agent.Predicate.Execution.ChangedFiles = []string{"a.go", "b.go", "c.go"}

	ci := testStatement()
	ci.Predicate.Execution.ChangedFiles = []string{"a.go", "b.go"}

	discrepancies := compareFileScope(agent, ci)
	if len(discrepancies) == 0 {
		t.Fatal("expected discrepancy for agent-only files")
	}

	if discrepancies[0].Type != DiscrepancyFileScope {
		t.Errorf("type = %q, want %q", discrepancies[0].Type, DiscrepancyFileScope)
	}
}

func TestCompareFileScope_CIOnly(t *testing.T) {
	agent := testStatement()
	agent.Predicate.Execution.ChangedFiles = []string{"a.go"}

	ci := testStatement()
	ci.Predicate.Execution.ChangedFiles = []string{"a.go", "b.go", "c.go", "d.go", "e.go"}

	discrepancies := compareFileScope(agent, ci)
	if len(discrepancies) == 0 {
		t.Fatal("expected discrepancy for CI-only files")
	}

	// 4+ files should be high severity
	if discrepancies[0].Severity != "high" {
		t.Errorf("severity = %q, want high for 4+ CI-only files", discrepancies[0].Severity)
	}
}

func TestCompareTestResults_Mismatch(t *testing.T) {
	agent := testStatement()
	agent.Predicate.Verification.Tests = []GateResult{
		{Name: "test-a", Status: "pass"},
		{Name: "test-b", Status: "pass"},
	}

	ci := testStatement()
	ci.Predicate.Verification.Tests = []GateResult{
		{Name: "test-a", Status: "pass"},
		{Name: "test-b", Status: "fail: assertion error"},
	}

	discrepancies := compareTestResults(agent, ci)
	if len(discrepancies) == 0 {
		t.Fatal("expected discrepancy for test result mismatch")
	}

	// Should have critical severity for test failure not reported by agent
	foundCritical := false
	for _, d := range discrepancies {
		if d.Severity == "critical" {
			foundCritical = true
		}
	}
	if !foundCritical {
		t.Error("expected critical severity for unreported test failure")
	}
}

func TestCompareCoverage_BelowThreshold(t *testing.T) {
	agent := testStatement()
	agent.Predicate.Verification.Coverage = &Coverage{Value: 85.0, Threshold: 80.0}

	ci := testStatement()
	ci.Predicate.Verification.Coverage = &Coverage{Value: 82.0, Threshold: 80.0}

	// 3% difference, below default threshold of 5%
	discrepancies := compareCoverage(agent, ci, 5.0)
	if len(discrepancies) != 0 {
		t.Errorf("expected no discrepancy for coverage within threshold, got %d", len(discrepancies))
	}
}

func TestCompareCoverage_AboveThreshold(t *testing.T) {
	agent := testStatement()
	agent.Predicate.Verification.Coverage = &Coverage{Value: 90.0, Threshold: 80.0}

	ci := testStatement()
	ci.Predicate.Verification.Coverage = &Coverage{Value: 70.0, Threshold: 80.0}

	// 20% difference, above threshold
	discrepancies := compareCoverage(agent, ci, 5.0)
	if len(discrepancies) == 0 {
		t.Fatal("expected discrepancy for coverage above threshold")
	}

	// 20% difference should be high severity
	if discrepancies[0].Severity != "high" {
		t.Errorf("severity = %q, want high for 20%% coverage difference", discrepancies[0].Severity)
	}
}

func TestCompareBoundary_AgentCompliantCINot(t *testing.T) {
	agent := testStatement()
	agent.Predicate.Boundary = Boundary{
		Compliance: BoundaryCompliance{OK: true, Reason: "all files in scope"},
	}

	ci := testStatement()
	ci.Predicate.Boundary = Boundary{
		Observed: ObservedBoundary{
			OutOfBoundaryPaths: []string{"internal/secret/creds.go"},
		},
		Compliance: BoundaryCompliance{OK: false, Reason: "1 file outside declared scope"},
	}

	discrepancies := compareBoundary(agent, ci)
	if len(discrepancies) == 0 {
		t.Fatal("expected discrepancy for boundary violation not reported by agent")
	}

	if discrepancies[0].Severity != "high" {
		t.Errorf("severity = %q, want high for unreported boundary violation", discrepancies[0].Severity)
	}
}

func TestCompareCommits_HeadMismatch(t *testing.T) {
	agent := testStatement()
	agent.Predicate.Trace.Commits = []string{"abc123"}

	ci := testStatement()
	ci.Predicate.Trace.Commits = []string{"def456"}

	discrepancies := compareCommits(agent, ci)
	if len(discrepancies) == 0 {
		t.Fatal("expected discrepancy for commit mismatch")
	}

	if discrepancies[0].Severity != "high" {
		t.Errorf("severity = %q, want high for commit mismatch", discrepancies[0].Severity)
	}
}

func TestGenerateSummary_OK(t *testing.T) {
	report := DiscrepancyReport{OK: true}
	summary := generateSummary(report)
	if summary == "" {
		t.Error("summary should not be empty")
	}
}

func TestGenerateSummary_WithDiscrepancies(t *testing.T) {
	report := DiscrepancyReport{
		OK: false,
		Discrepancies: []Discrepancy{
			{Severity: "critical"},
			{Severity: "high"},
			{Severity: "medium"},
		},
	}
	summary := generateSummary(report)
	if summary == "" {
		t.Error("summary should not be empty")
	}
}

func TestWriteReadDiscrepancyReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "discrepancy.json")

	report := DiscrepancyReport{
		OK:      false,
		RunID:   "test-123",
		Summary: "1 critical, 1 high",
		Discrepancies: []Discrepancy{
			{Type: DiscrepancyTestResult, Severity: "critical", Description: "test failed"},
		},
	}

	if err := WriteDiscrepancyReport(path, report); err != nil {
		t.Fatalf("WriteDiscrepancyReport error: %v", err)
	}

	read, err := ReadDiscrepancyReport(path)
	if err != nil {
		t.Fatalf("ReadDiscrepancyReport error: %v", err)
	}

	if read.OK != report.OK {
		t.Errorf("OK = %v, want %v", read.OK, report.OK)
	}
	if read.RunID != report.RunID {
		t.Errorf("RunID = %q, want %q", read.RunID, report.RunID)
	}
	if len(read.Discrepancies) != len(report.Discrepancies) {
		t.Errorf("Discrepancies count = %d, want %d", len(read.Discrepancies), len(report.Discrepancies))
	}
}
