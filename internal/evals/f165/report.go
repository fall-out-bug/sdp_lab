package f165

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Report struct {
	Meta    ReportMeta   `json:"meta"`
	Cases   []ReportCase `json:"cases"`
	Summary string       `json:"summary"`
}

type ReportMeta struct {
	FeatureID     string `json:"feature_id"`
	ReportVersion string `json:"report_version"`
	GeneratedAt   string `json:"generated_at"`
	Disclaimer    string `json:"disclaimer"`
	TotalCases    int    `json:"total_cases"`
	BlockedCount  int    `json:"blocked_count"`
	CleanCount    int    `json:"clean_count"`
	ResidualCount int    `json:"residual_count"`
}

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

func GenerateReport(testdataDir string) (Report, error) {
	matches, err := filepath.Glob(filepath.Join(testdataDir, "*.yaml"))
	if err != nil {
		return Report{}, fmt.Errorf("glob fixtures: %w", err)
	}
	var cases []ReportCase
	var blocked, clean, residual int
	for _, p := range matches {
		data, err := os.ReadFile(p)
		if err != nil {
			return Report{}, fmt.Errorf("read %s: %w", p, err)
		}
		var fixture Case
		if err := yaml.Unmarshal(data, &fixture); err != nil {
			return Report{}, fmt.Errorf("unmarshal %s: %w", p, err)
		}
		naive := NewUnsafeDemoRunner().RunCase(fixture)
		defended := DefendCase(fixture)
		if defended.BlockedReason != "" && !IsValidBlockedReason(defended.BlockedReason) {
			return Report{}, fmt.Errorf("invalid blocked_reason %q in case %s", defended.BlockedReason, fixture.CaseID)
		}
		if !IsValidVector(fixture.Vector) {
			return Report{}, fmt.Errorf("invalid vector %q in case %s", fixture.Vector, fixture.CaseID)
		}
		if fixture.ResidualRiskCategory != "" && !IsValidResidualRiskCategory(fixture.ResidualRiskCategory) {
			return Report{}, fmt.Errorf("invalid residual_risk_category %q in case %s", fixture.ResidualRiskCategory, fixture.CaseID)
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
	sort.Slice(cases, func(i, j int) bool {
		return cases[i].CaseID < cases[j].CaseID
	})
	summary := fmt.Sprintf(
		"F165 evaluated %d task-data vectors. Defenses blocked %d, confirmed %d clean, and classified %d as residual risk. "+
			"This is an advisory demo report, not a delivery gate verdict.",
		len(cases), blocked, clean, residual,
	)
	return Report{
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
	}, nil
}

func RenderReportJSON(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func RenderReportText(report Report) string {
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

func describeDefenseTrigger(defended ValidateResult, fixture Case) string {
	if defended.Verdict == "blocked" {
		return defended.BlockedReason
	}
	if defended.Verdict == "residual_risk" {
		return "unsupported_surface"
	}
	return "clean"
}
