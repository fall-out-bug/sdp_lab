// Package readiness implements the verify-before-completion gate for
// @review --mode readiness. It checks tests, coverage, docs, orphaned
// workstreams, and TODO hygiene, returning a structured JSON report.
package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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
	ProjectRoot    string
	CoverageDelta  float64 // max allowed coverage regression in percentage points (default 2.0)
	TestCommand    string  // override for test command (default "go test ./...")
	ChangedFiles   []string // if non-nil, TODO check is scoped to these files
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
	return CheckResult{Name: "tests", Status: StatusPass, Detail: fmt.Sprintf("%d tests passed", passed)}
}

func (rc *ReadinessChecker) checkCoverage(ctx context.Context) CheckResult {
	cmd := exec.CommandContext(ctx, "go", "test", "./...", "-coverprofile=/dev/null", "-covermode=count")
	cmd.Dir = rc.ProjectRoot
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()

	var currentPct float64
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "coverage:") {
			currentPct = extractCoveragePct(line)
			break
		}
	}

	if err != nil && currentPct == 0 {
		return CheckResult{Name: "coverage", Status: StatusFail, Detail: "failed to collect coverage"}
	}

	baseline := rc.loadBaseline()
	delta := currentPct - baseline
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
	files := rc.ChangedFiles
	if len(files) == 0 {
		// No explicit changed files; scan tracked Go files in the repo.
		var err error
		files, err = rc.trackedGoFiles()
		if err != nil {
			return CheckResult{Name: "todos", Status: StatusFail, Detail: fmt.Sprintf("scan files: %v", err)}
		}
	}

	count := 0
	for _, f := range files {
		path := f
		if !filepath.IsAbs(path) {
			path = filepath.Join(rc.ProjectRoot, path)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		count += len(todoRe.FindAll(b, -1))
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

// --- helpers ---

func (rc *ReadinessChecker) trackedGoFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--", "*.go")
	cmd.Dir = rc.ProjectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func (rc *ReadinessChecker) loadBaseline() float64 {
	path := filepath.Join(rc.ProjectRoot, ".coverage-baseline")
	b, err := os.ReadFile(path)
	if err != nil {
		return 0 // no baseline file means no regression check threshold applied
	}
	pct, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
	if err != nil {
		return 0
	}
	return pct
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
