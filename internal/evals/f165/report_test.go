package f165

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateReport_StructuredCases(t *testing.T) {
	report, err := GenerateReport(testdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	if report.Meta.FeatureID != "F165" {
		t.Errorf("feature_id = %q, want F165", report.Meta.FeatureID)
	}
	if report.Meta.Disclaimer == "" {
		t.Error("disclaimer is required")
	}
	if len(report.Cases) != report.Meta.TotalCases {
		t.Errorf("case count mismatch: len=%d, total=%d", len(report.Cases), report.Meta.TotalCases)
	}
}

func TestGenerateReport_BlockedReasonValidation(t *testing.T) {
	report, err := GenerateReport(testdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	for _, c := range report.Cases {
		if c.BlockedReason != "" && !IsValidBlockedReason(c.BlockedReason) {
			t.Errorf("case %s: invalid blocked_reason %q", c.CaseID, c.BlockedReason)
		}
	}
}

func TestGenerateReport_NarrativeNonAuthoritative(t *testing.T) {
	report, err := GenerateReport(testdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	lower := strings.ToLower(report.Meta.Disclaimer)
	for _, phrase := range []string{"do not authorize", "ci:pass", "qa:pass", "review:pass", "merge readiness"} {
		if !strings.Contains(lower, phrase) {
			t.Errorf("disclaimer missing required phrase: %q", phrase)
		}
	}
}

func TestGenerateReport_CountsMatch(t *testing.T) {
	report, err := GenerateReport(testdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	var blocked, clean, residual int
	for _, c := range report.Cases {
		switch c.DefendedVerdict {
		case "blocked":
			blocked++
		case "clean":
			clean++
		case "residual_risk":
			residual++
		}
	}
	if report.Meta.BlockedCount != blocked || report.Meta.CleanCount != clean || report.Meta.ResidualCount != residual {
		t.Errorf("counts mismatch: blocked=%d/%d clean=%d/%d residual=%d/%d",
			report.Meta.BlockedCount, blocked, report.Meta.CleanCount, clean, report.Meta.ResidualCount, residual)
	}
}

func TestRenderReportText_AvoidsExploitLanguage(t *testing.T) {
	report, _ := GenerateReport(testdataDir)
	text := RenderReportText(report)
	lower := strings.ToLower(text)
	badWords := []string{"exploit", "attack payload", "0day", "shellcode", "backdoor"}
	for _, w := range badWords {
		if strings.Contains(lower, w) {
			t.Errorf("text report contains exploit-style word %q", w)
		}
	}
}

func TestGenerateReport_InvalidVectorRejected(t *testing.T) {
	// Create a temporary testdata dir with one invalid fixture.
	dir := t.TempDir()
	invalidFixture := `case_id: F165-BAD-001
vector: unknown_vector
trusted_operator_request: test
trusted_state_snapshot: {}
untrusted_artifact: test
expected_unsafe_result:
  unsafe_action: test
  unsafe_claim: test
expected_defended_result:
  verdict: clean
evidence_expectation: test
residual_risk_category: none
`
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(invalidFixture), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := GenerateReport(dir)
	if err == nil {
		t.Fatal("expected error for invalid vector, got nil")
	}
}

func TestWrap_BoundaryNotInNarrative(t *testing.T) {
	// Ensure the boundary marker does not appear in fixtures.
	matches, _ := filepath.Glob(filepath.Join(testdataDir, "*.yaml"))
	for _, p := range matches {
		data, _ := os.ReadFile(p)
		if strings.Contains(string(data), "---UNTRUSTED-NARRATIVE-BOUNDARY---") {
			t.Errorf("fixture %s contains boundary marker", filepath.Base(p))
		}
	}
}
