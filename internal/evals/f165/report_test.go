package f165

import (
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
