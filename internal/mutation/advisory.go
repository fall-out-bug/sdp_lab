// Package mutation provides advisory-only guidance on mutation testing.
// This is P2/non-blocking per design — no actual mutation testing infrastructure.
// The advisory string is intended for inclusion in CI summaries.
package mutation

import (
	"fmt"
	"math"
	"strings"
)

// GenerateAdvisory returns a human-readable advisory string about mutation
// testing readiness based on the project's current coverage percentage and
// test count.
//
// This is purely informational. No mutation testing is performed.
func GenerateAdvisory(coveragePct float64, testCount int) string {
	var b strings.Builder

	b.WriteString("## Mutation Testing Advisory\n\n")
	b.WriteString(fmt.Sprintf("Coverage: %.1f%% | Tests: %d\n\n", coveragePct, testCount))

	// Provide actionable guidance based on coverage level.
	switch {
	case coveragePct >= 80:
		b.WriteString("Coverage is strong. Consider adding mutation testing (e.g., `go-mutesting`) ")
		b.WriteString("to validate test quality beyond line coverage.\n")
		b.WriteString("Mutation testing is most valuable when coverage is high but defects still escape.\n")
	case coveragePct >= 50:
		b.WriteString("Coverage is moderate. Focus on increasing line/branch coverage first.\n")
		b.WriteString("Mutation testing yields diminishing returns when basic coverage gaps exist.\n")
		b.WriteString("Priority: cover critical paths and error branches before investing in mutation.\n")
	default:
		b.WriteString("Coverage is low. Mutation testing is not recommended at this stage.\n")
		b.WriteString("Priority: achieve at least 50% coverage on critical packages first.\n")
	}

	b.WriteString("\n---\n")
	b.WriteString("This is advisory-only (F129-10). Not a CI gate.\n")

	// Suggest a rough mutation score estimate for planning purposes.
	estimatedScore := estimateMutationScore(coveragePct, testCount)
	b.WriteString(fmt.Sprintf("Estimated mutation score if added now: ~%.0f%% (rough heuristic).\n", estimatedScore))

	return b.String()
}

// estimateMutationScore provides a very rough heuristic for what mutation
// score the project might achieve based on coverage and test count.
// This is not a real measurement — just a planning aid.
func estimateMutationScore(coveragePct float64, testCount int) float64 {
	// Base: assume mutation score correlates loosely with coverage.
	base := coveragePct * 0.7

	// More tests tends to catch more mutations, with diminishing returns.
	testBonus := math.Min(float64(testCount)/100.0, 15.0)

	score := base + testBonus
	if score > 95 {
		score = 95
	}
	if score < 0 {
		score = 0
	}
	return score
}
