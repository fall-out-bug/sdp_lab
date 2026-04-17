package readiness

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckReturnsAllChecks(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)
	report := rc.Check(context.Background())

	if len(report.Checks) != 6 {
		t.Fatalf("expected 6 checks, got %d", len(report.Checks))
	}
	expected := []string{"tests", "coverage", "docs", "docs-artifacts", "orphans", "todos"}
	for i, name := range expected {
		if report.Checks[i].Name != name {
			t.Errorf("check[%d]: expected name %q, got %q", i, name, report.Checks[i].Name)
		}
	}
}

func TestReportSummaryAllPass(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)
	rc.TestCommand = "echo PASS" // override so test check passes

	report := rc.Check(context.Background())

	// With echo PASS the test check will report pass (exit 0).
	// Docs/orphans may fail because scaffold is minimal, so we just
	// verify the summary format is correct for the general case.
	if report.Ready {
		if report.Summary != "All checks pass" {
			t.Errorf("expected 'All checks pass' summary, got %q", report.Summary)
		}
	} else {
		if !strings.Contains(report.Summary, "failing") {
			t.Errorf("failing summary should mention 'failing', got %q", report.Summary)
		}
	}
}

func TestReportToJSON(t *testing.T) {
	report := Report{
		Ready:   true,
		Checks:  []CheckResult{{Name: "tests", Status: StatusPass, Detail: "1 packages passed"}},
		Summary: "All checks pass",
	}
	raw := report.ToJSON()

	var parsed Report
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("ToJSON produced invalid JSON: %v\n%s", err, raw)
	}
	if !parsed.Ready {
		t.Error("expected ready=true after round-trip")
	}
	if len(parsed.Checks) != 1 || parsed.Checks[0].Name != "tests" {
		t.Errorf("unexpected checks after round-trip: %+v", parsed.Checks)
	}
}

func TestCheckTestsPass(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)
	rc.TestCommand = "echo ok"

	result := rc.checkTests(context.Background())
	if result.Status != StatusPass {
		t.Errorf("expected pass for echo command, got %q: %s", result.Status, result.Detail)
	}
}

func TestCheckTestsFail(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)
	rc.TestCommand = "false" // exits non-zero

	result := rc.checkTests(context.Background())
	if result.Status != StatusFail {
		t.Errorf("expected fail for false command, got %q", result.Status)
	}
}

func TestCheckDocsConsistency(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)

	result := rc.checkDocs()
	// The scaffold has no docs content so docsync will report errors.
	if result.Name != "docs" {
		t.Errorf("expected check name 'docs', got %q", result.Name)
	}
}

func TestCheckOrphans(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)

	result := rc.checkOrphans()
	if result.Name != "orphans" {
		t.Errorf("expected check name 'orphans', got %q", result.Name)
	}
}

func TestCheckTODOsPass(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)
	rc.ChangedFiles = []string{filepath.Join(root, "clean.go")}

	// Write a file with no TODOs.
	if err := os.WriteFile(filepath.Join(root, "clean.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write clean.go: %v", err)
	}

	result := rc.checkTODOs()
	if result.Status != StatusPass {
		t.Errorf("expected pass with no TODOs, got %q: %s", result.Status, result.Detail)
	}
}

func TestCheckTODOsFail(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)
	rc.ChangedFiles = []string{filepath.Join(root, "dirty.go")}

	if err := os.WriteFile(filepath.Join(root, "dirty.go"), []byte("package main\n// TODO fix this\n"), 0o644); err != nil {
		t.Fatalf("write dirty.go: %v", err)
	}

	result := rc.checkTODOs()
	if result.Status != StatusFail {
		t.Errorf("expected fail with TODO, got %q", result.Status)
	}
}

func TestExtractCoveragePct(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"coverage: 78.3% of statements", 78.3},
		{"coverage: 100% of statements", 100.0},
		{"no coverage info", 0},
		{"  coverage: 0.0% of statements", 0.0},
	}
	for _, tt := range tests {
		got := extractCoveragePct(tt.input)
		if got != tt.want {
			t.Errorf("extractCoveragePct(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseCoverageTotal(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"total:                                          (statements)            78.3%\n", 78.3},
		{"total: (statements) 100%\n", 100.0},
		{"total: (statements)   0.0%\n", 0.0},
		{"no total line here", 0},
		{"some/path/file.go:3:  Func  50.0%\ntotal: (statements)   84.5%\n", 84.5},
	}
	for _, tt := range tests {
		got := parseCoverageTotal(tt.input)
		if got != tt.want {
			t.Errorf("parseCoverageTotal(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCheckDocsArtifactsNoChangelog(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)

	result := rc.checkDocsArtifacts()
	if result.Status != StatusFail {
		t.Errorf("expected fail when CHANGELOG.md missing, got %q", result.Status)
	}
	if result.Name != "docs-artifacts" {
		t.Errorf("expected name 'docs-artifacts', got %q", result.Name)
	}
}

func TestCheckDocsArtifactsStaleChangelog(t *testing.T) {
	root := scaffoldProject(t)
	// Write a CHANGELOG.md without today's date.
	if err := os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("# Changelog\n\n## 2020-01-01\n- old entry\n"), 0o644); err != nil {
		t.Fatalf("write CHANGELOG.md: %v", err)
	}
	rc := NewChecker(root)

	result := rc.checkDocsArtifacts()
	if result.Status != StatusFail {
		t.Errorf("expected fail with stale changelog, got %q: %s", result.Status, result.Detail)
	}
}

func TestCheckDocsArtifactsCurrentChangelog(t *testing.T) {
	root := scaffoldProject(t)
	today := time.Now().Format("2006-01-02")
	content := "# Changelog\n\n## " + today + "\n- Updated something\n"
	if err := os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write CHANGELOG.md: %v", err)
	}
	rc := NewChecker(root)

	result := rc.checkDocsArtifacts()
	if result.Status != StatusPass {
		t.Errorf("expected pass with current changelog, got %q: %s", result.Status, result.Detail)
	}
}

func TestCheckDocsArtifactsChangelogWithChangedFile(t *testing.T) {
	root := scaffoldProject(t)
	today := time.Now().Format("2006-01-02")
	content := "# Changelog\n\n## " + today + "\n- Updated foo.go for clarity\n"
	if err := os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write CHANGELOG.md: %v", err)
	}
	rc := NewChecker(root)
	rc.ChangedFiles = []string{"foo.go"}

	result := rc.checkDocsArtifacts()
	if result.Status != StatusPass {
		t.Errorf("expected pass with changed file in changelog, got %q: %s", result.Status, result.Detail)
	}
}

func TestCheckDocsArtifactsChangelogMissingChangedFile(t *testing.T) {
	root := scaffoldProject(t)
	today := time.Now().Format("2006-01-02")
	content := "# Changelog\n\n## " + today + "\n- Unrelated change\n"
	if err := os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write CHANGELOG.md: %v", err)
	}
	rc := NewChecker(root)
	rc.ChangedFiles = []string{"bar.go"}

	result := rc.checkDocsArtifacts()
	if result.Status != StatusFail {
		t.Errorf("expected fail when changed file not in changelog, got %q: %s", result.Status, result.Detail)
	}
}

func TestCheckTODOsSkipWithoutContext(t *testing.T) {
	root := scaffoldProject(t)
	rc := NewChecker(root)
	// ChangedFiles is nil/empty — should skip.
	result := rc.checkTODOs()
	if result.Status != StatusPass {
		t.Errorf("expected pass when no change context, got %q: %s", result.Status, result.Detail)
	}
	if !strings.Contains(result.Detail, "skipped") {
		t.Errorf("expected skip message, got %q", result.Detail)
	}
}

func TestCheckCoverageBaselineFile(t *testing.T) {
	root := scaffoldProject(t)
	if err := os.WriteFile(filepath.Join(root, ".coverage-baseline"), []byte("50.0"), 0o644); err != nil {
		t.Fatalf("write .coverage-baseline: %v", err)
	}
	rc := NewChecker(root)

	baseline := rc.loadBaseline()
	if baseline != 50.0 {
		t.Errorf("expected baseline 50.0 from local file, got %v", baseline)
	}
}

func TestCheckCoverageBaselineDefault(t *testing.T) {
	root := scaffoldProject(t)
	// No .coverage-baseline file and git show won't have .sdp/metrics/coverage.txt.
	rc := NewChecker(root)

	baseline := rc.loadBaseline()
	if baseline != defaultCoverageBaseline {
		t.Errorf("expected default baseline %.1f, got %v", defaultCoverageBaseline, baseline)
	}
}

func TestNewCheckerDefaults(t *testing.T) {
	rc := NewChecker("/tmp")
	if rc.CoverageDelta != 2.0 {
		t.Errorf("default CoverageDelta should be 2.0, got %v", rc.CoverageDelta)
	}
	if rc.TestCommand != "go test ./..." {
		t.Errorf("default TestCommand should be 'go test ./...', got %q", rc.TestCommand)
	}
}

// --- helpers ---

func scaffoldProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Minimal go.mod so go tools can function.
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test_proj\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Minimal docs structure for docsync/workstream validation.
	dirs := []string{
		filepath.Join("docs", "workstreams", "backlog"),
		filepath.Join("docs", "workstreams"),
		filepath.Join("docs", "roadmap"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Minimal files so docsync/workstream validation has something to parse.
	writeScaffoldFile(t, root, "docs/workstreams/INDEX.md", `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F999** | Test | 00-999-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-999-01 | F999 | Test | Backlog |
`)
	writeScaffoldFile(t, root, "docs/roadmap/ROADMAP.md", `# Roadmap

- **F999** — Test
`)
	writeScaffoldFile(t, root, "docs/workstreams/backlog/00-999-01.md", `---
ws_id: 00-999-01
feature_id: F999
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-999-01: Test

## Beads

- test-001: Scaffold

## Acceptance Criteria

- [ ] Test AC
`)

	// Init a git repo so git ls-files works.
	gitInit(t, root)

	return root
}

func writeScaffoldFile(t *testing.T, root, rel, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"add", "."},
		{"commit", "-m", "scaffold"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}
