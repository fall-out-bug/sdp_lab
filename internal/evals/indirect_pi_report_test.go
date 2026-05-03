package evals

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateIndirectPIReport_StructuredCases(t *testing.T) {
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}

	if report.Meta.FeatureID != "F165" {
		t.Errorf("feature_id = %q, want F165", report.Meta.FeatureID)
	}
	if report.Meta.Disclaimer == "" {
		t.Error("disclaimer is required")
	}
	if report.Meta.TotalCases == 0 {
		t.Fatal("expected at least one case")
	}
	if len(report.Cases) != report.Meta.TotalCases {
		t.Errorf("case count mismatch: len(cases)=%d, total_cases=%d", len(report.Cases), report.Meta.TotalCases)
	}
}

func TestGenerateIndirectPIReport_BlockedReasonValidation(t *testing.T) {
	// Ensure all emitted blocked_reason values are in the closed set.
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	for _, c := range report.Cases {
		if c.BlockedReason != "" && !IsValidBlockedReason(c.BlockedReason) {
			t.Errorf("case %s: invalid blocked_reason %q", c.CaseID, c.BlockedReason)
		}
	}
}

func TestGenerateIndirectPIReport_ResidualRiskValidation(t *testing.T) {
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	for _, c := range report.Cases {
		if c.ResidualRiskCategory != "" && c.ResidualRiskCategory != ResidualRiskNone && !IsValidResidualRiskCategory(c.ResidualRiskCategory) {
			t.Errorf("case %s: invalid residual_risk_category %q", c.CaseID, c.ResidualRiskCategory)
		}
	}
}

func TestGenerateIndirectPIReport_NarrativeNonAuthoritative(t *testing.T) {
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	// The summary must contain the advisory disclaimer.
	if !strings.Contains(report.Summary, "advisory") && !strings.Contains(report.Summary, "demo") {
		t.Error("summary should contain advisory/demo language")
	}
	// The meta disclaimer must explicitly reject gate authority.
	lower := strings.ToLower(report.Meta.Disclaimer)
	for _, phrase := range []string{"do not authorize", "ci:pass", "qa:pass", "review:pass", "merge readiness"} {
		if !strings.Contains(lower, phrase) {
			t.Errorf("disclaimer missing required phrase: %q", phrase)
		}
	}
}

func TestGenerateIndirectPIReport_CountsMatch(t *testing.T) {
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
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
	if report.Meta.BlockedCount != blocked {
		t.Errorf("blocked_count = %d, want %d", report.Meta.BlockedCount, blocked)
	}
	if report.Meta.CleanCount != clean {
		t.Errorf("clean_count = %d, want %d", report.Meta.CleanCount, clean)
	}
	if report.Meta.ResidualCount != residual {
		t.Errorf("residual_count = %d, want %d", report.Meta.ResidualCount, residual)
	}
}

func TestRenderReportJSON_RoundTrip(t *testing.T) {
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}

	data, err := RenderReportJSON(report)
	if err != nil {
		t.Fatalf("render json: %v", err)
	}

	var parsed IndirectPIReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json round-trip: %v", err)
	}
	if len(parsed.Cases) != len(report.Cases) {
		t.Errorf("round-trip case count = %d, want %d", len(parsed.Cases), len(report.Cases))
	}
}

func TestRenderReportText_ContainsCases(t *testing.T) {
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}

	text := RenderReportText(report)
	if !strings.Contains(text, "F165 Indirect Prompt Injection Report") {
		t.Error("text report missing header")
	}
	for _, c := range report.Cases {
		if !strings.Contains(text, c.CaseID) {
			t.Errorf("text report missing case %s", c.CaseID)
		}
	}
}

func TestRenderReportText_AvoidsExploitLanguage(t *testing.T) {
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}

	text := RenderReportText(report)
	lower := strings.ToLower(text)
	badWords := []string{"exploit", "attack payload", "0day", "shellcode", "backdoor"}
	for _, w := range badWords {
		if strings.Contains(lower, w) {
			t.Errorf("text report contains exploit-style word %q", w)
		}
	}
}

func TestReportCase_CanBeConsumedByAutomation(t *testing.T) {
	report, err := GenerateIndirectPIReport(indirectPITestdataDir)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	for _, c := range report.Cases {
		if c.CaseID == "" {
			t.Error("case_id is required for automation")
		}
		if c.DefendedVerdict == "" {
			t.Error("defended_verdict is required for automation")
		}
		// automation may ignore narrative and consume only typed fields
		_ = c.CaseID
		_ = c.DefendedVerdict
		_ = c.BlockedReason
		_ = c.TrustedEvidenceRef
		_ = c.ResidualRiskCategory
	}
}
