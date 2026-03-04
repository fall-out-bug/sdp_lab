package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscrepancyType represents a category of discrepancy between attestations.
type DiscrepancyType string

const (
	DiscrepancyFileScope      DiscrepancyType = "file_scope_drift"
	DiscrepancyTestResult     DiscrepancyType = "test_result_mismatch"
	DiscrepancyCoverage       DiscrepancyType = "coverage_mismatch"
	DiscrepancyMissingAgent   DiscrepancyType = "missing_agent_attestation"
	DiscrepancyMissingCI      DiscrepancyType = "missing_ci_attestation"
	DiscrepancyBoundary       DiscrepancyType = "boundary_violation"
	DiscrepancyCommitMismatch DiscrepancyType = "commit_mismatch"
)

// Discrepancy represents a single detected discrepancy.
type Discrepancy struct {
	Type        DiscrepancyType `json:"type"`
	Severity    string          `json:"severity"` // critical, high, medium, low
	Description string          `json:"description"`
	AgentValue  interface{}     `json:"agent_value,omitempty"`
	CIValue     interface{}     `json:"ci_value,omitempty"`
}

// DiscrepancyReport contains the full comparison result.
type DiscrepancyReport struct {
	OK            bool          `json:"ok"`
	RunID         string        `json:"run_id"`
	AgentFile     string        `json:"agent_file,omitempty"`
	CIFile        string        `json:"ci_file,omitempty"`
	Discrepancies []Discrepancy `json:"discrepancies,omitempty"`
	Summary       string        `json:"summary"`
}

// CompareOptions configures the comparison behavior.
type CompareOptions struct {
	CoverageThreshold float64 // Minimum coverage difference to flag (default: 5.0)
	EvidenceDir       string  // Directory containing evidence files (default: .sdp/evidence)
}

// CompareAttestations compares agent and CI attestations for a given run.
func CompareAttestations(runID string, opts CompareOptions) (DiscrepancyReport, error) {
	if opts.EvidenceDir == "" {
		opts.EvidenceDir = ".sdp/evidence"
	}
	if opts.CoverageThreshold == 0 {
		opts.CoverageThreshold = 5.0
	}

	report := DiscrepancyReport{
		OK:    true,
		RunID: runID,
	}

	// Find agent attestation (not CI)
	agentPath, err := findAttestation(opts.EvidenceDir, runID, "run-")
	if err != nil {
		return report, fmt.Errorf("find agent attestation: %w", err)
	}
	report.AgentFile = agentPath

	// Find CI attestation
	ciPath, err := findAttestation(opts.EvidenceDir, runID, "ci-auto-")
	if err != nil {
		return report, fmt.Errorf("find CI attestation: %w", err)
	}
	report.CIFile = ciPath

	// Load agent attestation
	var agentStmt CodingWorkflowStatement
	if agentPath != "" {
		var err error
		agentStmt, err = ReadAttestation(agentPath)
		if err != nil {
			return report, fmt.Errorf("read agent attestation: %w", err)
		}
	} else {
		report.Discrepancies = append(report.Discrepancies, Discrepancy{
			Type:        DiscrepancyMissingAgent,
			Severity:    "high",
			Description: "Agent attestation not found for run",
		})
		report.OK = false
	}

	// Load CI attestation
	var ciStmt CodingWorkflowStatement
	if ciPath != "" {
		var err error
		ciStmt, err = ReadAttestation(ciPath)
		if err != nil {
			return report, fmt.Errorf("read CI attestation: %w", err)
		}
	} else {
		report.Discrepancies = append(report.Discrepancies, Discrepancy{
			Type:        DiscrepancyMissingCI,
			Severity:    "medium",
			Description: "CI attestation not found for run",
		})
		// Missing CI is not critical - CI may not have run yet
	}

	// If both exist, compare them
	if agentPath != "" && ciPath != "" {
		report.Discrepancies = append(report.Discrepancies, compareFileScope(agentStmt, ciStmt)...)
		report.Discrepancies = append(report.Discrepancies, compareTestResults(agentStmt, ciStmt)...)
		report.Discrepancies = append(report.Discrepancies, compareCoverage(agentStmt, ciStmt, opts.CoverageThreshold)...)
		report.Discrepancies = append(report.Discrepancies, compareBoundary(agentStmt, ciStmt)...)
		report.Discrepancies = append(report.Discrepancies, compareCommits(agentStmt, ciStmt)...)
	}

	// Determine overall status
	for _, d := range report.Discrepancies {
		if d.Severity == "critical" || d.Severity == "high" {
			report.OK = false
			break
		}
	}

	// Generate summary
	report.Summary = generateSummary(report)

	return report, nil
}

// findAttestation searches for an attestation file matching the run ID.
func findAttestation(dir, runID, prefix string) (string, error) {
	// First try exact prefix + runID pattern
	pattern := filepath.Join(dir, prefix+runID+".json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob exact attestation pattern %q: %w", pattern, err)
	}
	if len(matches) > 0 {
		return matches[0], nil
	}

	// Then try prefix + runID as substring
	pattern = filepath.Join(dir, prefix+"*"+runID+"*.json")
	matches, err = filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob partial attestation pattern %q: %w", pattern, err)
	}
	if len(matches) > 0 {
		sort.Sort(sort.Reverse(sort.StringSlice(matches)))
		return matches[0], nil
	}

	// Finally try any file with prefix (for backwards compatibility)
	if prefix != "" {
		pattern = filepath.Join(dir, prefix+"*.json")
		matches, err = filepath.Glob(pattern)
		if err != nil {
			return "", fmt.Errorf("glob fallback attestation pattern %q: %w", pattern, err)
		}
		if len(matches) > 0 {
			sort.Sort(sort.Reverse(sort.StringSlice(matches)))
			return matches[0], nil
		}
	}

	return "", nil
}

// compareFileScope compares the changed files between attestations.
func compareFileScope(agent, ci CodingWorkflowStatement) []Discrepancy {
	var discrepancies []Discrepancy

	agentFiles := make(map[string]bool)
	for _, f := range agent.Predicate.Execution.ChangedFiles {
		agentFiles[f] = true
	}

	ciFiles := make(map[string]bool)
	for _, f := range ci.Predicate.Execution.ChangedFiles {
		ciFiles[f] = true
	}

	// Find files in agent but not in CI
	var agentOnly []string
	for f := range agentFiles {
		if !ciFiles[f] {
			agentOnly = append(agentOnly, f)
		}
	}

	// Find files in CI but not in agent
	var ciOnly []string
	for f := range ciFiles {
		if !agentFiles[f] {
			ciOnly = append(ciOnly, f)
		}
	}

	if len(agentOnly) > 0 {
		severity := "medium"
		if len(agentOnly) > 3 {
			severity = "high"
		}
		discrepancies = append(discrepancies, Discrepancy{
			Type:        DiscrepancyFileScope,
			Severity:    severity,
			Description: fmt.Sprintf("Agent claims %d files not observed by CI", len(agentOnly)),
			AgentValue:  agentOnly,
			CIValue:     []string{},
		})
	}

	if len(ciOnly) > 0 {
		severity := "medium"
		if len(ciOnly) > 3 {
			severity = "high"
		}
		discrepancies = append(discrepancies, Discrepancy{
			Type:        DiscrepancyFileScope,
			Severity:    severity,
			Description: fmt.Sprintf("CI observed %d files not claimed by agent", len(ciOnly)),
			AgentValue:  []string{},
			CIValue:     ciOnly,
		})
	}

	return discrepancies
}

// compareTestResults compares test outcomes between attestations.
func compareTestResults(agent, ci CodingWorkflowStatement) []Discrepancy {
	var discrepancies []Discrepancy

	agentPassed := countPassingTests(agent.Predicate.Verification.Tests)
	ciPassed := countPassingTests(ci.Predicate.Verification.Tests)

	if agentPassed != ciPassed {
		severity := "medium"
		if agentPassed > ciPassed {
			severity = "high" // Agent claims more tests pass than CI observed
		}

		discrepancies = append(discrepancies, Discrepancy{
			Type:        DiscrepancyTestResult,
			Severity:    severity,
			Description: fmt.Sprintf("Test result mismatch: agent=%d passing, CI=%d passing", agentPassed, ciPassed),
			AgentValue:  agentPassed,
			CIValue:     ciPassed,
		})
	}

	// Check for explicit test failures in CI
	for _, t := range ci.Predicate.Verification.Tests {
		if strings.HasPrefix(t.Status, "fail") {
			// Check if agent also reported failure
			agentFailed := false
			for _, at := range agent.Predicate.Verification.Tests {
				if at.Name == t.Name && strings.HasPrefix(at.Status, "fail") {
					agentFailed = true
					break
				}
			}
			if !agentFailed {
				discrepancies = append(discrepancies, Discrepancy{
					Type:        DiscrepancyTestResult,
					Severity:    "critical",
					Description: fmt.Sprintf("CI reports test failure '%s' not reported by agent: %s", t.Name, t.Status),
					AgentValue:  "pass",
					CIValue:     t.Status,
				})
			}
		}
	}

	return discrepancies
}

// compareCoverage compares coverage percentages.
func compareCoverage(agent, ci CodingWorkflowStatement, threshold float64) []Discrepancy {
	var discrepancies []Discrepancy

	if agent.Predicate.Verification.Coverage != nil && ci.Predicate.Verification.Coverage != nil {
		agentCov := agent.Predicate.Verification.Coverage.Value
		ciCov := ci.Predicate.Verification.Coverage.Value
		diff := agentCov - ciCov

		if diff < 0 {
			diff = -diff
		}

		if diff > threshold {
			severity := "low"
			if diff >= 10 {
				severity = "medium"
			}
			if diff >= 20 {
				severity = "high"
			}

			discrepancies = append(discrepancies, Discrepancy{
				Type:        DiscrepancyCoverage,
				Severity:    severity,
				Description: fmt.Sprintf("Coverage difference: agent=%.1f%%, CI=%.1f%% (diff=%.1f%%)", agentCov, ciCov, diff),
				AgentValue:  agentCov,
				CIValue:     ciCov,
			})
		}
	}

	return discrepancies
}

// compareBoundary compares scope compliance status.
func compareBoundary(agent, ci CodingWorkflowStatement) []Discrepancy {
	var discrepancies []Discrepancy

	// If CI observed out-of-boundary files but agent didn't report them
	if !ci.Predicate.Boundary.Compliance.OK && len(ci.Predicate.Boundary.Observed.OutOfBoundaryPaths) > 0 {
		// Check if agent also detected the issue
		if agent.Predicate.Boundary.Compliance.OK {
			discrepancies = append(discrepancies, Discrepancy{
				Type:        DiscrepancyBoundary,
				Severity:    "high",
				Description: fmt.Sprintf("CI detected boundary violation not reported by agent: %s", ci.Predicate.Boundary.Compliance.Reason),
				AgentValue:  "compliant",
				CIValue:     ci.Predicate.Boundary.Observed.OutOfBoundaryPaths,
			})
		}
	}

	return discrepancies
}

// compareCommits compares commit references.
func compareCommits(agent, ci CodingWorkflowStatement) []Discrepancy {
	var discrepancies []Discrepancy

	agentCommits := agent.Predicate.Trace.Commits
	ciCommits := ci.Predicate.Trace.Commits

	if len(agentCommits) > 0 && len(ciCommits) > 0 {
		// Compare head commits
		agentHead := agentCommits[0]
		ciHead := ciCommits[0]

		if agentHead != ciHead {
			discrepancies = append(discrepancies, Discrepancy{
				Type:        DiscrepancyCommitMismatch,
				Severity:    "high",
				Description: "Head commit mismatch between agent and CI attestations",
				AgentValue:  agentHead,
				CIValue:     ciHead,
			})
		}
	}

	return discrepancies
}

// countPassingTests counts tests with passing status.
func countPassingTests(tests []GateResult) int {
	count := 0
	for _, t := range tests {
		if strings.HasPrefix(t.Status, "pass") {
			count++
		}
	}
	return count
}

// generateSummary creates a human-readable summary.
func generateSummary(report DiscrepancyReport) string {
	if report.OK {
		return fmt.Sprintf("No critical discrepancies found (%d observations)", len(report.Discrepancies))
	}

	critical := 0
	high := 0
	medium := 0
	low := 0

	for _, d := range report.Discrepancies {
		switch d.Severity {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}
	}

	parts := []string{}
	if critical > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", critical))
	}
	if high > 0 {
		parts = append(parts, fmt.Sprintf("%d high", high))
	}
	if medium > 0 {
		parts = append(parts, fmt.Sprintf("%d medium", medium))
	}
	if low > 0 {
		parts = append(parts, fmt.Sprintf("%d low", low))
	}

	return fmt.Sprintf("Discrepancies found: %s", strings.Join(parts, ", "))
}

// WriteDiscrepancyReport writes a discrepancy report to a file.
func WriteDiscrepancyReport(path string, report DiscrepancyReport) error {
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

// ReadDiscrepancyReport reads a discrepancy report from a file.
func ReadDiscrepancyReport(path string) (DiscrepancyReport, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return DiscrepancyReport{}, err
	}
	var report DiscrepancyReport
	if err := json.Unmarshal(b, &report); err != nil {
		return DiscrepancyReport{}, fmt.Errorf("parse report: %w", err)
	}
	return report, nil
}
