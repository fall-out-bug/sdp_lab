package bootstrap

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"sdp_dev/internal/scout"
)

// BrownfieldResult holds the delta analysis output comparing existing project
// rules with fresh scout-derived rules.
type BrownfieldResult struct {
	ExistingRules map[string]string `json:"existing_rules"` // section -> content
	FreshScout    map[string]string `json:"fresh_scout"`    // section -> content
	Deltas        []Delta           `json:"deltas"`
}

// RunBrownfield compares existing project rules with fresh scout output and
// produces a BrownfieldResult with delta markers for every change. The result
// is deterministic: same inputs always produce identical deltas.
//
// Nil existingRules is treated as empty (all sections ADDED).
// Nil card produces empty fresh sections (all existing sections REMOVED).
func RunBrownfield(card *scout.ProjectCard, existingRules map[string]string) *BrownfieldResult {
	if existingRules == nil {
		existingRules = map[string]string{}
	}
	fresh := extractFreshSections(card)
	deltas := CompareSections(existingRules, fresh)
	return &BrownfieldResult{
		ExistingRules: existingRules,
		FreshScout:    fresh,
		Deltas:        deltas,
	}
}

// extractFreshSections builds a section->content map from a scout.ProjectCard.
// Each meaningful field of the card becomes a named section whose value is a
// human-readable string derived from the scout data.
func extractFreshSections(card *scout.ProjectCard) map[string]string {
	if card == nil {
		return map[string]string{}
	}
	sections := make(map[string]string)

	// Language section
	langs := formatLanguages(card.Identity.Languages)
	if langs != "" {
		sections["language"] = langs
	}

	// Architecture section
	arch := formatArchitecture(card.Identity, card.Scale)
	if arch != "" {
		sections["architecture"] = arch
	}

	// CI section
	ci := formatCI(card.Maturity, card.Conventions)
	if ci != "" {
		sections["ci"] = ci
	}

	// Testing section
	testing := formatTesting(card.Scale, card.Conventions)
	if testing != "" {
		sections["testing"] = testing
	}

	// Linting section
	lint := formatLinting(card.Conventions)
	if lint != "" {
		sections["linting"] = lint
	}

	// Build section
	build := formatBuild(card.Build)
	if build != "" {
		sections["build"] = build
	}

	return sections
}

// formatLanguages produces a deterministic language summary string.
func formatLanguages(languages map[string]scout.LangStats) string {
	if len(languages) == 0 {
		return ""
	}
	names := make([]string, 0, len(languages))
	for lang := range languages {
		names = append(names, lang)
	}
	sort.Strings(names)
	var parts []string
	for _, lang := range names {
		stats := languages[lang]
		parts = append(parts, fmt.Sprintf("%s(%d files, %.0f%%)", lang, stats.Files, stats.Ratio*100))
	}
	return strings.Join(parts, ", ")
}

// formatArchitecture produces an architecture summary from identity and scale.
func formatArchitecture(id scout.Identity, sc scout.Scale) string {
	var parts []string
	if id.BuildSystem != nil && *id.BuildSystem != "" {
		parts = append(parts, "build_system="+*id.BuildSystem)
	}
	if id.Monorepo {
		parts = append(parts, "monorepo=true")
	}
	if sc.Directories > 0 {
		parts = append(parts, fmt.Sprintf("dirs=%d", sc.Directories))
	}
	if sc.SourceFiles > 0 {
		parts = append(parts, fmt.Sprintf("sources=%d", sc.SourceFiles))
	}
	return strings.Join(parts, ", ")
}

// formatCI produces a CI section summary.
func formatCI(m scout.Maturity, c scout.Conventions) string {
	if !m.HasCI {
		return ""
	}
	var parts []string
	if m.CISystem != nil {
		parts = append(parts, "system="+*m.CISystem)
	}
	if c.CIWorkflow != nil {
		parts = append(parts, "config="+c.CIWorkflow.ConfigFile)
		if len(c.CIWorkflow.Steps) > 0 {
			steps := make([]string, len(c.CIWorkflow.Steps))
			copy(steps, c.CIWorkflow.Steps)
			sort.Strings(steps)
			parts = append(parts, "steps="+strings.Join(steps, ","))
		}
	}
	if len(parts) == 0 {
		return "ci_detected=true"
	}
	return strings.Join(parts, ", ")
}

// formatTesting produces a testing section summary.
func formatTesting(sc scout.Scale, c scout.Conventions) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("test_files=%d", sc.TestFiles))
	parts = append(parts, fmt.Sprintf("test_ratio=%.2f", sc.TestRatio))
	if c.TestStructure.Style != "" {
		parts = append(parts, "style="+c.TestStructure.Style)
	}
	return strings.Join(parts, ", ")
}

// formatLinting produces a linting section summary.
func formatLinting(c scout.Conventions) string {
	if c.LintConfig == nil {
		return ""
	}
	var parts []string
	parts = append(parts, "tool="+c.LintConfig.Tool)
	parts = append(parts, "config="+c.LintConfig.ConfigFile)
	if len(c.LintConfig.Rules) > 0 {
		rules := make([]string, len(c.LintConfig.Rules))
		copy(rules, c.LintConfig.Rules)
		sort.Strings(rules)
		parts = append(parts, "rules="+strings.Join(rules, ","))
	}
	return strings.Join(parts, ", ")
}

// formatBuild produces a build section summary.
func formatBuild(b scout.Build) string {
	var parts []string
	if b.PackageManager != nil {
		parts = append(parts, "pkg_mgr="+*b.PackageManager)
	}
	parts = append(parts, fmt.Sprintf("deps=%d", b.DependencyCount))
	if len(b.EntryPoints) > 0 {
		eps := make([]string, len(b.EntryPoints))
		copy(eps, b.EntryPoints)
		sort.Strings(eps)
		parts = append(parts, "entry_points="+strings.Join(eps, ","))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// MarshalBrownfieldResult serializes a BrownfieldResult to indented JSON.
func MarshalBrownfieldResult(result *BrownfieldResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

// UnmarshalBrownfieldResult deserializes a BrownfieldResult from JSON bytes.
func UnmarshalBrownfieldResult(data []byte) (*BrownfieldResult, error) {
	var result BrownfieldResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("brownfield: unmarshal: %w", err)
	}
	return &result, nil
}
