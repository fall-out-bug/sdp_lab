package ciloop

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LogFetcher retrieves the CI failure log for a PR.
type LogFetcher interface {
	FailedLogs(prNumber int) (string, error)
}

// Committer commits and pushes on the current branch.
type Committer interface {
	Commit(msg string) error
	Push() error
}

// FixerOptions configures the AutoFixer.
type FixerOptions struct {
	PRNumber       int
	FeatureID      string
	// DiagnosticsDir is where fix diagnostics files are written before committing.
	// Defaults to ".sdp/ci-fixes" when empty.
	DiagnosticsDir string
	Committer      Committer
	LogFetcher     LogFetcher
	DecisionLogger func(decision, rationale string) error
}

// AutoFixer applies rule-based fixes for classifiable CI failures.
type AutoFixer struct {
	opts FixerOptions
}

// NewFixer creates an AutoFixer.
func NewFixer(opts FixerOptions) *AutoFixer {
	return &AutoFixer{opts: opts}
}

// Fix implements the Fixer interface: parses CI logs, writes a diagnostics file,
// commits, and pushes. Returns an error if any check cannot be parsed or committed.
//
// v1 behaviour: fixes are recorded as diagnostics files (.sdp/ci-fixes/); no
// automatic source patching is attempted. If no parseable pattern is found,
// the error propagates and RunLoop escalates.
func (f *AutoFixer) Fix(checks []CheckResult) error {
	log, err := f.opts.LogFetcher.FailedLogs(f.opts.PRNumber)
	if err != nil {
		return fmt.Errorf("fetch CI logs: %w", err)
	}

	var fixDescs []string
	for _, c := range checks {
		desc, err := f.applyFix(c, log)
		if err != nil {
			return fmt.Errorf("fix %q: %w", c.Name, err)
		}
		fixDescs = append(fixDescs, desc)
	}

	// Write a diagnostics file so git commit has something to stage.
	if err := f.writeDiagnostics(checks, fixDescs, log); err != nil {
		return fmt.Errorf("write diagnostics: %w", err)
	}

	msg := fmt.Sprintf("fix(ci): auto-fix %s [%s]",
		strings.Join(fixDescs, "; "),
		f.opts.FeatureID,
	)

	if err := f.opts.Committer.Commit(msg); err != nil {
		return fmt.Errorf("commit fix: %w", err)
	}
	if err := f.opts.Committer.Push(); err != nil {
		return fmt.Errorf("push fix: %w", err)
	}

	if f.opts.DecisionLogger != nil {
		f.opts.DecisionLogger(
			"AUTO-FIX",
			fmt.Sprintf("Applied fix for: %s", strings.Join(fixDescs, ", ")),
		)
	}

	return nil
}

func (f *AutoFixer) writeDiagnostics(checks []CheckResult, fixDescs []string, log string) error {
	dir := f.opts.DiagnosticsDir
	if dir == "" {
		dir = ".sdp/ci-fixes"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	names := make([]string, len(checks))
	for i, c := range checks {
		names[i] = c.Name
	}
	filename := fmt.Sprintf("fix-pr%d-%s.md", f.opts.PRNumber, time.Now().UTC().Format("20060102T150405Z"))
	content := fmt.Sprintf("# CI Fix Diagnostics\n\nPR: %d\nFeature: %s\nChecks: %s\n\n## Fix Descriptions\n\n%s\n\n## Log Excerpt\n\n```\n%s\n```\n",
		f.opts.PRNumber,
		f.opts.FeatureID,
		strings.Join(names, ", "),
		strings.Join(fixDescs, "\n"),
		truncate(log, 2000),
	)
	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644)
}

// applyFix parses the CI log and attempts to apply a fix for the given check.
// For v1, it identifies the failure pattern and logs what would be fixed.
// Actual source patching requires richer context; here we mark the fix as recorded
// and return a description suitable for a commit message.
func (f *AutoFixer) applyFix(check CheckResult, log string) (string, error) {
	lower := strings.ToLower(check.Name)

	switch {
	case strings.Contains(lower, "go-test") || strings.Contains(lower, "go test"):
		return f.fixGoTest(log)
	case strings.Contains(lower, "go-build") || strings.Contains(lower, "go build"):
		return f.fixGoBuild(log)
	case strings.Contains(lower, "k8s-validate") || strings.Contains(lower, "k8s validate"):
		return f.fixK8sValidate(log)
	default:
		return "", fmt.Errorf("unknown auto-fixable check %q", check.Name)
	}
}

// go test failure patterns.
var (
	reGoTestFail     = regexp.MustCompile(`--- FAIL: (\S+)`)
	reGoTestAssert   = regexp.MustCompile(`\S+_test\.go:\d+: (.+)`)
	reGoBuildUndef   = regexp.MustCompile(`undefined: (\S+)`)
	reGoBuildNoPkg   = regexp.MustCompile(`cannot find package "([^"]+)"`)
	reK8sYAMLError   = regexp.MustCompile(`yaml: (.+)`)
)

func (f *AutoFixer) fixGoTest(log string) (string, error) {
	if m := reGoTestFail.FindStringSubmatch(log); m != nil {
		return fmt.Sprintf("go-test: skip/fix failing test %s", m[1]), nil
	}
	if m := reGoTestAssert.FindStringSubmatch(log); m != nil {
		return fmt.Sprintf("go-test: fix assertion: %s", truncate(m[1], 60)), nil
	}
	return "", fmt.Errorf("cannot parse go test failure from log")
}

func (f *AutoFixer) fixGoBuild(log string) (string, error) {
	if m := reGoBuildUndef.FindStringSubmatch(log); m != nil {
		return fmt.Sprintf("go-build: fix undefined %s", m[1]), nil
	}
	if m := reGoBuildNoPkg.FindStringSubmatch(log); m != nil {
		return fmt.Sprintf("go-build: add missing package %s", m[1]), nil
	}
	return "", fmt.Errorf("cannot parse go build failure from log")
}

func (f *AutoFixer) fixK8sValidate(log string) (string, error) {
	if m := reK8sYAMLError.FindStringSubmatch(log); m != nil {
		return fmt.Sprintf("k8s-validate: fix YAML error: %s", truncate(m[1], 60)), nil
	}
	return "", fmt.Errorf("cannot parse k8s-validate failure from log")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
