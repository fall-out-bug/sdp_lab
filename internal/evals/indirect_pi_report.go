package evals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// IndirectPIReport is the automation-safe report for F165 demos.
// Typed fields are consumable by downstream automation.
// Free-form narrative is explicitly non-authoritative.
type IndirectPIReport struct {
	// Meta contains report metadata.
	Meta ReportMeta `json:"meta"`

	// Cases contains one entry per fixture case.
	Cases []ReportCase `json:"cases"`

	// Summary is a concise operator-facing narrative. It is non-authoritative.
	Summary string `json:"summary"`
}

// ReportMeta carries report-level metadata.
type ReportMeta struct {
	FeatureID       string `json:"feature_id"`
	ReportVersion   string `json:"report_version"`
	GeneratedAt     string `json:"generated_at"`
	Disclaimer      string `json:"disclaimer"`
	TotalCases      int    `json:"total_cases"`
	BlockedCount    int    `json:"blocked_count"`
	CleanCount      int    `json:"clean_count"`
	ResidualCount   int    `json:"residual_count"`
}

// ReportCase is one row in the F165 report.
type ReportCase struct {
	CaseID               string `json:"case_id"`
	Vector               string `json:"vector"`
	NaiveVerdict         string `json:"naive_verdict"`
	DefendedVerdict      string `json:"defended_verdict"`
	BlockedReason        string `json:"blocked_reason,omitempty"`
	TrustedEvidenceRef   string `json:"trusted_evidence_ref,omitempty"`
	ResidualRiskCategory string `json:"residual_risk_category,omitempty"`
	DefenseTrigger       string `json:"defense_trigger,omitempty"`
}

// GenerateIndirectPIReport loads all fixtures in testdataDir and produces
// a deterministic report comparing naive and defended outcomes.
func GenerateIndirectPIReport(testdataDir string) (IndirectPIReport, error) {
	matches, err := filepath.Glob(filepath.Join(testdataDir, "*.yaml"))
	if err != nil {
		return IndirectPIReport{}, fmt.Errorf("glob fixtures: %w", err)
	}

	var cases []ReportCase
	var blocked, clean, residual int

	for _, p := range matches {
		data, err := os.ReadFile(p)
		if err != nil {
			return IndirectPIReport{}, fmt.Errorf("read %s: %w", p, err)
		}
		var fixture IndirectPICase
		if err := yamlUnmarshal(data, &fixture); err != nil {
			return IndirectPIReport{}, fmt.Errorf("unmarshal %s: %w", p, err)
		}

		// Naive path
		naive := NewUnsafeDemoRunner().RunCase(fixture)

		// Defended path
		norm := Normalize(fixture.UntrustedArtifact)
		parsed := Parse(norm, fixture.Vector)
		wrapped := Wrap(parsed, norm)
		defended := Validate(wrapped, fixture.TrustedStateSnapshot, fixture.ExpectedUnsafeResult.UnsafeAction, fixture.ExpectedUnsafeResult.UnsafeClaim)

		// Validate closed sets before emitting.
		if defended.BlockedReason != "" && !IsValidBlockedReason(defended.BlockedReason) {
			return IndirectPIReport{}, fmt.Errorf("invalid blocked_reason %q in case %s", defended.BlockedReason, fixture.CaseID)
		}
		if fixture.ResidualRiskCategory != "" && !IsValidResidualRiskCategory(fixture.ResidualRiskCategory) {
			return IndirectPIReport{}, fmt.Errorf("invalid residual_risk_category %q in case %s", fixture.ResidualRiskCategory, fixture.CaseID)
		}

		rc := ReportCase{
			CaseID:               fixture.CaseID,
			Vector:               fixture.Vector,
			NaiveVerdict:         naive.Verdict,
			DefendedVerdict:      defended.Verdict,
			BlockedReason:        defended.BlockedReason,
			TrustedEvidenceRef:   defended.TrustedEvidenceRef,
			ResidualRiskCategory: fixture.ResidualRiskCategory,
			DefenseTrigger:       describeDefenseTrigger(defended, fixture),
		}
		cases = append(cases, rc)

		switch defended.Verdict {
		case "blocked":
			blocked++
		case "clean":
			clean++
		case "residual_risk":
			residual++
		}
	}

	summary := fmt.Sprintf(
		"F165 evaluated %d task-data vectors. Defenses blocked %d, confirmed %d clean, and classified %d as residual risk. "+
		"This is an advisory demo report, not a delivery gate verdict.",
		len(cases), blocked, clean, residual,
	)

	report := IndirectPIReport{
		Meta: ReportMeta{
			FeatureID:     "F165",
			ReportVersion: "v1",
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
			Disclaimer:    "F165 verdicts are advisory demo/eval verdicts. They do not authorize Beads, Git, PR, CI, or filesystem mutation. Do not map to ci:pass, qa:pass, review:pass, or merge readiness without a separate decision record.",
			TotalCases:    len(cases),
			BlockedCount:  blocked,
			CleanCount:    clean,
			ResidualCount: residual,
		},
		Cases:   cases,
		Summary: summary,
	}
	return report, nil
}

// RenderReportJSON emits the report as indented JSON.
func RenderReportJSON(report IndirectPIReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// RenderReportText emits a concise operator-facing text summary.
func RenderReportText(report IndirectPIReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "F165 Indirect Prompt Injection Report\n")
	fmt.Fprintf(&b, "=====================================\n")
	fmt.Fprintf(&b, "Generated: %s\n", report.Meta.GeneratedAt)
	fmt.Fprintf(&b, "Disclaimer: %s\n\n", report.Meta.Disclaimer)
	for _, c := range report.Cases {
		fmt.Fprintf(&b, "  %s | vector=%s | naive=%s | defended=%s", c.CaseID, c.Vector, c.NaiveVerdict, c.DefendedVerdict)
		if c.BlockedReason != "" {
			fmt.Fprintf(&b, " | blocked_reason=%s", c.BlockedReason)
		}
		if c.TrustedEvidenceRef != "" {
			fmt.Fprintf(&b, " | evidence=%s", c.TrustedEvidenceRef)
		}
		if c.ResidualRiskCategory != "" && c.ResidualRiskCategory != ResidualRiskNone {
			fmt.Fprintf(&b, " | residual=%s", c.ResidualRiskCategory)
		}
		if c.DefenseTrigger != "" {
			fmt.Fprintf(&b, " | trigger=%s", c.DefenseTrigger)
		}
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "\nSummary: %s\n", report.Summary)
	fmt.Fprintf(&b, "=====================================\n")
	return b.String()
}

func describeDefenseTrigger(defended IndirectPIValidateResult, fixture IndirectPICase) string {
	if defended.Verdict == "blocked" {
		return defended.BlockedReason
	}
	if defended.Verdict == "residual_risk" {
		return "unsupported_surface"
	}
	return "clean"
}

func yamlUnmarshal(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}
