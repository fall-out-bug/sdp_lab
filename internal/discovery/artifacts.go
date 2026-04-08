package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session holds all pipeline state for one discovery run.
type Session struct {
	Slug       string
	Date       string
	Frame      *FrameResult
	Hypothesis *HypothesisResult
	Scan       *ScanResult
	Validation *ValidationResult
	Experiment *ExperimentBrief
}

// NewSession creates a new Session with today's date and a slug derived from idea.
func NewSession(idea string) *Session {
	return &Session{
		Slug: slugify(idea),
		Date: time.Now().Format("2006-01-02"),
	}
}

// slugify converts a string to a URL-safe slug (max 40 chars).
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-':
			b.WriteRune('-')
		}
	}
	slug := b.String()
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return strings.Trim(slug, "-")
}

// WriteArtifacts writes frame, hypothesis, and scan markdown files to dir.
func WriteArtifacts(dir string, s *Session) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	prefix := filepath.Join(dir, s.Date+"-"+s.Slug)

	if s.Frame != nil {
		if err := writeFrame(prefix+"-frame.md", s.Frame); err != nil {
			return err
		}
	}
	if s.Hypothesis != nil {
		if err := writeHypothesis(prefix+"-hypothesis.md", s.Hypothesis); err != nil {
			return err
		}
	}
	if s.Scan != nil {
		if err := writeScan(prefix+"-scan.md", s.Scan); err != nil {
			return err
		}
	}
	if s.Validation != nil {
		if err := writeValidation(prefix+"-validation.md", s.Validation); err != nil {
			return err
		}
	}
	if s.Experiment != nil {
		if err := writeExperiment(prefix+"-experiment.md", s.Experiment); err != nil {
			return err
		}
	}
	return nil
}

func writeFrame(path string, f *FrameResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Frame\n\n")
	fmt.Fprintf(&b, "**Raw idea:** %s\n\n", f.RawIdea)
	fmt.Fprintf(&b, "## Problem Statement\n\n%s\n\n", f.ProblemStatement)
	fmt.Fprintf(&b, "## Jobs to Be Done\n\n")
	for _, j := range f.Jobs {
		fmt.Fprintf(&b, "- %s\n", j)
	}
	fmt.Fprintf(&b, "\n**Appetite:** %s\n\n**Scope:** %s\n", f.Appetite, f.Scope)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeHypothesis(path string, h *HypothesisResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Hypothesis\n\n")
	fmt.Fprintf(&b, "**Raw idea:** %s\n\n", h.RawIdea)
	fmt.Fprintf(&b, "## Test Card (Strategyzer)\n\n")
	fmt.Fprintf(&b, "**We believe** %s\n\n", h.WeBelieve)
	fmt.Fprintf(&b, "**To verify this**, we will %s\n\n", h.ToVerify)
	fmt.Fprintf(&b, "**We'll measure** %s\n\n", h.WeMeasure)
	fmt.Fprintf(&b, "**We are right if** %s\n\n", h.WeAreRightIf)
	fmt.Fprintf(&b, "## Assumptions (RAT-Ranked)\n\n")
	fmt.Fprintf(&b, "| Rank | Assumption | Risk | Uncertainty | RAT Score |\n")
	fmt.Fprintf(&b, "|------|-----------|------|-------------|----------|\n")
	for _, a := range h.Assumptions {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %.0f |\n",
			a.RATRank, a.Statement, a.RiskLevel, a.Uncertainty, a.RATScore)
	}
	if len(h.Assumptions) > 0 {
		fmt.Fprintf(&b, "\n**Riskiest assumption (rank 1):** %s\n", h.Assumptions[0].Statement)
	}
	if len(h.Requirements) > 0 {
		fmt.Fprintf(&b, "\n## Requirements\n\n")
		for _, r := range h.Requirements {
			fmt.Fprintf(&b, "- %s\n", r)
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeScan(path string, r *ScanResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Scan\n\n")
	fmt.Fprintf(&b, "**Whitespace:** %s\n\n", r.Whitespace)
	fmt.Fprintf(&b, "## Landscape\n\n")
	fmt.Fprintf(&b, "| Tool | Disposition | Coverage | Flagged |\n")
	fmt.Fprintf(&b, "|---|---|---|---|\n")
	for _, item := range r.Items {
		flagged := "—"
		if item.DepthFlag != nil && item.DepthFlag.Flagged {
			flagged = "⚠️ " + item.DepthFlag.Reason
		}
		fmt.Fprintf(&b, "| %s | %s | %.2f | %s |\n",
			item.Name, item.Disposition, item.CoverageScore, flagged)
	}
	// Append raw JSON for downstream use
	raw, _ := json.MarshalIndent(r, "", "  ")
	fmt.Fprintf(&b, "\n```json\n%s\n```\n", raw)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeValidation(path string, v *ValidationResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Validation\n\n")

	verdictIcon := map[FinalVerdict]string{
		VerdictGO:    "✅",
		VerdictPIVOT: "🔄",
		VerdictKILL:  "❌",
	}
	icon := verdictIcon[v.FinalVerdict]
	if icon == "" {
		icon = "❓"
	}
	fmt.Fprintf(&b, "## %s Final Verdict: %s\n\n", icon, v.FinalVerdict)
	fmt.Fprintf(&b, "%s\n\n", v.VerdictReason)

	if v.PivotSuggestion != "" {
		fmt.Fprintf(&b, "**Pivot suggestion:** %s\n\n", v.PivotSuggestion)
	}
	if v.KillReason != "" {
		fmt.Fprintf(&b, "**Kill reason:** %s\n\n", v.KillReason)
	}
	if v.NeedsExperiment {
		fmt.Fprintf(&b, "> ⚠️ **Needs experiment:** one or more claims have insufficient desk-research data — Phase 4b recommended.\n\n")
	}

	fmt.Fprintf(&b, "## Claim Validation\n\n")
	for _, c := range v.Claims {
		verdictStr := string(c.Verdict)
		fmt.Fprintf(&b, "### Rank %d — %s (confidence %.0f%%)\n\n", c.RATRank, verdictStr, c.Confidence*100)
		fmt.Fprintf(&b, "**Claim:** %s\n\n", c.Claim)
		if c.Notes != "" {
			fmt.Fprintf(&b, "**Notes:** %s\n\n", c.Notes)
		}
		if len(c.Evidence) > 0 {
			fmt.Fprintf(&b, "| Direction | Evidence | Estimate? |\n")
			fmt.Fprintf(&b, "|-----------|----------|-----------|\n")
			for _, e := range c.Evidence {
				est := "no"
				if e.IsEstimate {
					est = "yes"
				}
				stmt := e.Statement
				if e.SourceURL != "" {
					stmt = fmt.Sprintf("[%s](%s)", e.Statement, e.SourceURL)
				}
				fmt.Fprintf(&b, "| %s | %s | %s |\n", strings.ToUpper(e.Direction), stmt, est)
			}
			fmt.Fprintf(&b, "\n")
		}
	}

	fmt.Fprintf(&b, "---\n\n*Cost: $%.5f*\n", v.CostUSD)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeExperiment(path string, e *ExperimentBrief) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Experiment Brief\n\n")
	fmt.Fprintf(&b, "## Format: %s\n\n", e.Format)
	fmt.Fprintf(&b, "**Objective:** %s\n\n", e.Objective)
	fmt.Fprintf(&b, "**Hypothesis:** %s\n\n", e.Hypothesis)
	fmt.Fprintf(&b, "**Success metric:** %s\n\n", e.SuccessMetric)
	fmt.Fprintf(&b, "**Time box:** %d days\n\n", e.TimeBoxDays)
	if e.RawClaim != "" {
		fmt.Fprintf(&b, "**Testing claim:** %s\n\n", e.RawClaim)
	}
	if len(e.SetupSteps) > 0 {
		fmt.Fprintf(&b, "## Setup Steps\n\n")
		for i, s := range e.SetupSteps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, s)
		}
		fmt.Fprintf(&b, "\n")
	}
	if len(e.RequiredTools) > 0 {
		fmt.Fprintf(&b, "## Required Tools\n\n")
		for _, t := range e.RequiredTools {
			fmt.Fprintf(&b, "- %s\n", t)
		}
		fmt.Fprintf(&b, "\n")
	}
	fmt.Fprintf(&b, "---\n\n*Cost: $%.5f*\n", e.CostUSD)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
