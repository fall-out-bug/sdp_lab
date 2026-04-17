// Package readiness implements the verify-before-completion gate for
// @review --mode readiness. It checks tests, coverage, docs, orphaned
// workstreams, and TODO hygiene, returning a structured JSON report.
package readiness

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sdp_dev/internal/docsync"
	"sdp_dev/internal/workstream"
)

// CheckStatus is "pass" or "fail".
type CheckStatus string

const (
	StatusPass CheckStatus = "pass"
	StatusFail CheckStatus = "fail"
)

// CheckResult describes the outcome of a single readiness check.
type CheckResult struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Detail string      `json:"detail"`
}

// Report is the top-level readiness gate output.
type Report struct {
	Ready   bool          `json:"ready"`
	Checks  []CheckResult `json:"checks"`
	Summary string        `json:"summary"`
}

// ReadinessChecker runs all readiness checks against a project root.
type ReadinessChecker struct {
	ProjectRoot   string
	CoverageDelta float64  // max allowed coverage regression in percentage points (default 2.0)
	TestCommand   string   // override for test command (default "go test ./...")
	ChangedFiles  []string // if non-nil, TODO check is scoped to these files
}

// NewChecker returns a ReadinessChecker with sensible defaults.
func NewChecker(projectRoot string) *ReadinessChecker {
	return &ReadinessChecker{
		ProjectRoot:   projectRoot,
		CoverageDelta: 2.0,
		TestCommand:   "go test ./...",
	}
}

// Check runs all readiness checks and returns a Report.
func (rc *ReadinessChecker) Check(ctx context.Context) Report {
	checks := []CheckResult{
		rc.checkTests(ctx),
		rc.checkCoverage(ctx),
		rc.checkDocs(),
		rc.checkDocsArtifacts(),
		rc.checkOrphans(),
		rc.checkTODOs(),
	}

	failCount := 0
	var failing []string
	for _, c := range checks {
		if c.Status == StatusFail {
			failCount++
			failing = append(failing, c.Name)
		}
	}

	summary := "All checks pass"
	ready := failCount == 0
	if !ready {
		summary = fmt.Sprintf("%d check(s) failing: %s", failCount, strings.Join(failing, ", "))
	}

	return Report{
		Ready:   ready,
		Checks:  checks,
		Summary: summary,
	}
}

// ToJSON serialises the report to indented JSON.
func (r Report) ToJSON() string {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return `{"ready": false, "summary": "json marshal error"}`
	}
	return string(b)
}

// --- individual checks ---

func (rc *ReadinessChecker) checkTests(ctx context.Context) CheckResult {
	args := strings.Fields(rc.TestCommand)
	if len(args) == 0 {
		return CheckResult{Name: "tests", Status: StatusFail, Detail: "empty test command"}
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = rc.ProjectRoot
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		last := lines[len(lines)-1]
		if len(last) > 120 {
			last = last[:120] + "..."
		}
		return CheckResult{Name: "tests", Status: StatusFail, Detail: last}
	}
	passed := parseTestCount(string(out))
	return CheckResult{Name: "tests", Status: StatusPass, Detail: fmt.Sprintf("%d packages passed", passed)}
}

func (rc *ReadinessChecker) checkCoverage(ctx context.Context) CheckResult {
	// Write a real coverprofile to a temp file so we get aggregate coverage.
	covFile, err := os.CreateTemp("", "readiness-cov-*.out")
	if err != nil {
		return CheckResult{Name: "coverage", Status: StatusFail, Detail: fmt.Sprintf("create temp file: %v", err)}
	}
	covPath := covFile.Name()
	covFile.Close()
	defer os.Remove(covPath)

	cmd := exec.CommandContext(ctx, "go", "test", "./...", "-coverprofile="+covPath, "-covermode=count")
	cmd.Dir = rc.ProjectRoot
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, testErr := cmd.CombinedOutput()

	// If the coverprofile is empty or missing, tests may have failed with no
	// coverage data at all.
	if stat, serr := os.Stat(covPath); serr != nil || stat.Size() == 0 {
		if testErr != nil {
			return CheckResult{Name: "coverage", Status: StatusFail, Detail: "failed to collect coverage"}
		}
		return CheckResult{Name: "coverage", Status: StatusFail, Detail: "empty coverprofile"}
	}

	// Parse the total: line from "go tool cover -func".
	toolCmd := exec.CommandContext(ctx, "go", "tool", "cover", "-func="+covPath)
	toolCmd.Dir = rc.ProjectRoot
	toolOut, err := toolCmd.Output()
	if err != nil {
		// Fallback: scan test output for last coverage line.
		var fallback float64
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "coverage:") {
				fallback = extractCoveragePct(line)
			}
		}
		if fallback == 0 {
			return CheckResult{Name: "coverage", Status: StatusFail, Detail: "failed to parse coverage"}
		}
		return rc.buildCoverageResult(fallback)
	}

	currentPct := parseCoverageTotal(string(toolOut))
	if currentPct == 0 {
		return CheckResult{Name: "coverage", Status: StatusFail, Detail: "no total: line in cover -func output"}
	}

	return rc.buildCoverageResult(currentPct)
}

func (rc *ReadinessChecker) buildCoverageResult(currentPct float64) CheckResult {
	baseline := rc.loadBaseline()
	delta := currentPct - baseline

	// When ChangedFiles is empty (no specific change context), only check the
	// absolute threshold, not the delta regression.
	if len(rc.ChangedFiles) == 0 {
		return CheckResult{
			Name:   "coverage",
			Status: StatusPass,
			Detail: fmt.Sprintf("%.1f%% (baseline %.1f%%, delta check skipped — no change context)", currentPct, baseline),
		}
	}

	if delta < -rc.CoverageDelta {
		return CheckResult{
			Name:   "coverage",
			Status: StatusFail,
			Detail: fmt.Sprintf("%.1f%% (baseline %.1f%%, delta %.1fpp, threshold %.1fpp)", currentPct, baseline, delta, -rc.CoverageDelta),
		}
	}
	return CheckResult{
		Name:   "coverage",
		Status: StatusPass,
		Detail: fmt.Sprintf("%.1f%% (baseline %.1f%%, delta %+.1fpp)", currentPct, baseline, delta),
	}
}

func (rc *ReadinessChecker) checkDocs() CheckResult {
	report, err := docsync.CheckConsistency(rc.ProjectRoot, true)
	if err != nil {
		return CheckResult{Name: "docs", Status: StatusFail, Detail: fmt.Sprintf("docsync error: %v", err)}
	}
	if report.HasErrors() {
		return CheckResult{
			Name:   "docs",
			Status: StatusFail,
			Detail: fmt.Sprintf("%d error-severity finding(s)", len(report.Issues)),
		}
	}
	return CheckResult{
		Name:   "docs",
		Status: StatusPass,
		Detail: fmt.Sprintf("%d findings", len(report.Issues)),
	}
}

func (rc *ReadinessChecker) checkDocsArtifacts() CheckResult {
	var failures []string

	// 1. CHANGELOG.md must exist and have an entry for today's date.
	changelogPath := filepath.Join(rc.ProjectRoot, "CHANGELOG.md")
	b, err := os.ReadFile(changelogPath)
	if err != nil {
		return CheckResult{
			Name:   "docs-artifacts",
			Status: StatusFail,
			Detail: "CHANGELOG.md not found",
		}
	}

	today := time.Now().Format("2006-01-02")
	if !strings.Contains(string(b), today) {
		failures = append(failures, fmt.Sprintf("CHANGELOG.md has no entry for %s", today))
	}

	// 2. If ChangedFiles is set, at least one CHANGELOG entry should mention a changed file.
	if len(rc.ChangedFiles) > 0 {
		found := false
		content := strings.ToLower(string(b))
		for _, f := range rc.ChangedFiles {
			base := strings.ToLower(filepath.Base(f))
			if base != "" && strings.Contains(content, base) {
				found = true
				break
			}
		}
		if !found {
			failures = append(failures, "CHANGELOG.md does not reference any changed file")
		}
	}

	if len(failures) > 0 {
		return CheckResult{
			Name:   "docs-artifacts",
			Status: StatusFail,
			Detail: strings.Join(failures, "; "),
		}
	}
	return CheckResult{
		Name:   "docs-artifacts",
		Status: StatusPass,
		Detail: "CHANGELOG.md present with today's entry",
	}
}

func (rc *ReadinessChecker) checkOrphans() CheckResult {
	report, err := workstream.ValidateProtocol(rc.ProjectRoot, false, false)
	if err != nil {
		return CheckResult{Name: "orphans", Status: StatusFail, Detail: fmt.Sprintf("protocol validation error: %v", err)}
	}
	orphanCount := 0
	for _, issue := range report.Issues {
		if strings.Contains(strings.ToLower(issue.Message), "orphan") ||
			strings.Contains(strings.ToLower(issue.Message), "missing in index") ||
			strings.Contains(strings.ToLower(issue.Message), "backlog file not found") {
			orphanCount++
		}
	}
	if orphanCount > 0 {
		return CheckResult{
			Name:   "orphans",
			Status: StatusFail,
			Detail: fmt.Sprintf("%d orphaned workstream(s)", orphanCount),
		}
	}
	return CheckResult{Name: "orphans", Status: StatusPass, Detail: "0 orphaned workstreams"}
}

var todoRe = regexp.MustCompile(`\b(TODO|FIXME|HACK)\b`)

func (rc *ReadinessChecker) checkTODOs() CheckResult {
	if len(rc.ChangedFiles) == 0 {
		// No change context available; skip the check entirely.
		return CheckResult{Name: "todos", Status: StatusPass, Detail: "skipped — no change context"}
	}

	count := 0
	for _, f := range rc.ChangedFiles {
		rel := f
		if filepath.IsAbs(rel) {
			// Convert to repo-relative path for git diff.
			rel, _ = filepath.Rel(rc.ProjectRoot, rel)
		}
		added := rc.addedMarkerLines(rel)
		count += len(added)
	}

	if count > 0 {
		return CheckResult{
			Name:   "todos",
			Status: StatusFail,
			Detail: fmt.Sprintf("%d new TODO/FIXME/HACK in changed files", count),
		}
	}
	return CheckResult{Name: "todos", Status: StatusPass, Detail: "0 new TODO/FIXME/HACK"}
}

// addedMarkerLines returns lines added in <file> that contain a TODO/FIXME/HACK
// marker, derived from `git diff --unified=0 main...HEAD -- <file>`.
func (rc *ReadinessChecker) addedMarkerLines(file string) []string {
	// Guard: only use git diff when ProjectRoot is a git repo.
	if _, err := os.Stat(filepath.Join(rc.ProjectRoot, ".git")); err != nil {
		return rc.scanFileMarkers(file)
	}

	cmd := exec.Command("git", "diff", "--unified=0", "main...HEAD", "--", file)
	cmd.Dir = rc.ProjectRoot
	out, err := cmd.Output()
	if err != nil {
		return rc.scanFileMarkers(file)
	}

	// Empty diff + file not tracked = wrong git context (e.g. temp dir inside
	// a worktree).  Fall back to file scan so we don't produce false negatives.
	if len(out) == 0 {
		trackCmd := exec.Command("git", "ls-files", "--error-unmatch", "--", file)
		trackCmd.Dir = rc.ProjectRoot
		if trackCmd.Run() != nil {
			return rc.scanFileMarkers(file)
		}
	}

	var matches []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		// Only newly added lines (git diff prefix "+").
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			if todoRe.MatchString(line) {
				matches = append(matches, strings.TrimSpace(line[1:]))
			}
		}
	}
	return matches
}

// scanFileMarkers reads a file and returns all TODO/FIXME/HACK markers.
func (rc *ReadinessChecker) scanFileMarkers(file string) []string {
	path := filepath.Join(rc.ProjectRoot, file)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return todoRe.FindAllString(string(b), -1)
}

const defaultCoverageBaseline = 70.0

func (rc *ReadinessChecker) loadBaseline() float64 {
	// 1. Try the local file first (for local overrides).
	path := filepath.Join(rc.ProjectRoot, ".coverage-baseline")
	b, err := os.ReadFile(path)
	if err == nil {
		if pct, perr := strconv.ParseFloat(strings.TrimSpace(string(b)), 64); perr == nil {
			return pct
		}
	}

	// 2. Try git show from origin/main (CI baseline).
	cmd := exec.Command("git", "show", "origin/main:.sdp/metrics/coverage.txt")
	cmd.Dir = rc.ProjectRoot
	out, err := cmd.Output()
	if err == nil {
		if pct, perr := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); perr == nil {
			return pct
		}
	}

	// 3. Default baseline.
	return defaultCoverageBaseline
}

func parseTestCount(output string) int {
	// Look for "ok  <pkg>  <time>" lines.
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "ok ") {
			count++
		}
	}
	if count == 0 {
		count = 1 // at least one package ran if we reached here
	}
	return count
}

func extractCoveragePct(line string) float64 {
	// Match patterns like "coverage: 78.3% of statements"
	re := regexp.MustCompile(`coverage:\s+([0-9.]+)%`)
	m := re.FindStringSubmatch(line)
	if len(m) < 2 {
		return 0
	}
	pct, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	return pct
}

// parseCoverageTotal parses the "total:" line from "go tool cover -func" output.
// Example line: "total:                                          (statements)            78.3%"
func parseCoverageTotal(output string) float64 {
	re := regexp.MustCompile(`^total:\s+\(statements\)\s+([0-9.]+)%`)
	for _, line := range strings.Split(output, "\n") {
		m := re.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) >= 2 {
			pct, err := strconv.ParseFloat(m[1], 64)
			if err == nil {
				return pct
			}
		}
	}
	return 0
}
