package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/executil"
)

// --- Mock runner for CLI integration tests ---

// mockRunner implements executil.CommandRunner for testing.
type mockRunner struct {
	outputFunc         map[string]outputResult
	combinedOutputFunc map[string]outputResult
}

type outputResult struct {
	data []byte
	err  error
}

func (m *mockRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	for k, v := range m.outputFunc {
		if strings.Contains(key, k) {
			return v.data, v.err
		}
	}
	return nil, fmt.Errorf("mock: unexpected Output call: %s", key)
}

func (m *mockRunner) CombinedOutput(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	for k, v := range m.combinedOutputFunc {
		if strings.Contains(key, k) {
			return v.data, v.err
		}
	}
	return nil, nil
}

func (m *mockRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	return nil
}

// scanResult captures the output and exit code from a runCoverageScanWithWriters call.
type scanResult struct {
	exitCode int
	stdout   string
	stderr   string
}

// runScanTest is a helper that runs runCoverageScanWithWriters with captured output.
func runScanTest(args []string, runner executil.CommandRunner) scanResult {
	var stdout, stderr bytes.Buffer
	code := runCoverageScanWithWriters(args, &stdout, &stderr, runner)
	return scanResult{
		exitCode: code,
		stdout:   stdout.String(),
		stderr:   stderr.String(),
	}
}

// --- Package validation tests (Finding 3) ---

func TestValidatePackagePattern(t *testing.T) {
	tests := []struct {
		pkg     string
		wantErr bool
	}{
		{"./...", false},
		{"./internal/foo/...", false},
		{"../...", false},
		{"github.com/user/repo/...", false},
		{"sdp_dev/internal/foo", false},
		{"-v", true},
		{"--help", true},
		{"-race -coverprofile=/tmp/evil", true},
		{"", true},
		{"single", true}, // bare word, not a valid pattern
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			err := validatePackagePattern(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePackagePattern(%q) error = %v, wantErr %v", tt.pkg, err, tt.wantErr)
			}
		})
	}
}

// --- Filter and formatting tests ---

func TestCoverageFilterBelowThreshold(t *testing.T) {
	output := `internal/foo/bar.go:Bar 100.0%
internal/foo/bar.go:Uncovered 0.0%
internal/baz/qux.go:Qux 50.0%
total:                                                  (statements) 50.0%
`
	gaps, totalPct, err := filterGapsFromFuncOutput(output, 80.0)
	if err != nil {
		t.Fatalf("filterGapsFromFuncOutput: %v", err)
	}
	if len(gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d", len(gaps))
	}
	if totalPct != 50.0 {
		t.Errorf("totalPct = %.1f, want 50.0", totalPct)
	}
	if gaps[0].Function != "Uncovered" {
		t.Errorf("gap[0].Function = %q, want %q", gaps[0].Function, "Uncovered")
	}
	if gaps[1].Function != "Qux" {
		t.Errorf("gap[1].Function = %q, want %q", gaps[1].Function, "Qux")
	}
	if gaps[0].ThresholdGap != 80.0 {
		t.Errorf("gap[0].ThresholdGap = %.1f, want 80.0", gaps[0].ThresholdGap)
	}
}

func TestCoverageFilterAllAboveThreshold(t *testing.T) {
	output := `internal/foo/bar.go:Bar 100.0%
internal/foo/bar.go:Also 90.0%
total:                                                  95.0%
`
	gaps, _, err := filterGapsFromFuncOutput(output, 80.0)
	if err != nil {
		t.Fatalf("filterGapsFromFuncOutput: %v", err)
	}
	if len(gaps) != 0 {
		t.Fatalf("expected 0 gaps, got %d", len(gaps))
	}
}

func TestCoverageFilterExactThreshold(t *testing.T) {
	output := `internal/foo/bar.go:Bar 80.0%
total:                                                  80.0%
`
	gaps, _, err := filterGapsFromFuncOutput(output, 80.0)
	if err != nil {
		t.Fatalf("filterGapsFromFuncOutput: %v", err)
	}
	if len(gaps) != 0 {
		t.Fatalf("coverage at exact threshold should not be a gap, got %d gaps", len(gaps))
	}
}

func TestCoverageFilterWithLineNumbers(t *testing.T) {
	output := `internal/foo/bar.go:25:  Bar 100.0%
internal/foo/bar.go:40:  Uncovered 0.0%
total:                                                  50.0%
`
	gaps, _, err := filterGapsFromFuncOutput(output, 80.0)
	if err != nil {
		t.Fatalf("filterGapsFromFuncOutput: %v", err)
	}
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if gaps[0].Function != "Uncovered" {
		t.Errorf("Function = %q, want %q", gaps[0].Function, "Uncovered")
	}
	if gaps[0].Line != 40 {
		t.Errorf("Line = %d, want 40", gaps[0].Line)
	}
}

func TestCoverageFilterEmptyInput(t *testing.T) {
	_, _, err := filterGapsFromFuncOutput("", 80.0)
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
	if !strings.Contains(err.Error(), "no valid coverage data") {
		t.Errorf("error should mention 'no valid coverage data', got: %v", err)
	}
}

// --- Text formatting tests ---

func TestCoverageFormatText(t *testing.T) {
	gaps := []coverageGap{
		{File: "internal/foo/bar.go", Function: "BarFunc", Coverage: 45.5, Line: 10, ThresholdGap: 34.5},
		{File: "cmd/main.go", Function: "MainFunc", Coverage: 72.0, Line: 5, ThresholdGap: 8.0},
	}
	out := formatGapsText(gaps, 80.0)
	if !strings.Contains(out, "internal/foo/bar.go") {
		t.Error("text output should contain file path")
	}
	if !strings.Contains(out, "BarFunc") {
		t.Error("text output should contain function name")
	}
	if !strings.Contains(out, "45.5%") {
		t.Error("text output should contain coverage percentage")
	}
	if !strings.Contains(out, "80.0% threshold") {
		t.Errorf("text output should contain threshold, got: %s", out)
	}
}

func TestCoverageFormatTextNoGaps(t *testing.T) {
	gaps := []coverageGap{}
	out := formatGapsText(gaps, 80.0)
	if !strings.Contains(out, "All functions above threshold") {
		t.Errorf("expected 'All functions above threshold' message, got: %s", out)
	}
}

// --- JSON formatting tests ---

func TestCoverageFormatJSON(t *testing.T) {
	gaps := []coverageGap{
		{File: "a.go", Function: "FuncA", Coverage: 50.0, Line: 10, ThresholdGap: 30.0},
		{File: "b.go", Function: "FuncB", Coverage: 70.0, Line: 20, ThresholdGap: 10.0},
	}
	out := formatGapsJSON(gaps, 55.0)

	var parsed coverageJSONOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if len(parsed.Gaps) != 2 {
		t.Fatalf("expected 2 items in JSON gaps, got %d", len(parsed.Gaps))
	}
	if parsed.Gaps[0].File != "a.go" {
		t.Errorf("parsed.Gaps[0].File = %q, want %q", parsed.Gaps[0].File, "a.go")
	}
	if parsed.Gaps[0].Function != "FuncA" {
		t.Errorf("parsed.Gaps[0].Function = %q, want %q", parsed.Gaps[0].Function, "FuncA")
	}
	if parsed.Gaps[0].Coverage != 50.0 {
		t.Errorf("parsed.Gaps[0].Coverage = %.1f, want 50.0", parsed.Gaps[0].Coverage)
	}
	if parsed.Gaps[0].Line != 10 {
		t.Errorf("parsed.Gaps[0].Line = %d, want 10", parsed.Gaps[0].Line)
	}
	if parsed.Gaps[0].ThresholdGap != 30.0 {
		t.Errorf("parsed.Gaps[0].ThresholdGap = %.1f, want 30.0", parsed.Gaps[0].ThresholdGap)
	}
	if parsed.TotalCoverage != 55.0 {
		t.Errorf("parsed.TotalCoverage = %.1f, want 55.0", parsed.TotalCoverage)
	}
}

func TestCoverageFormatJSONEmpty(t *testing.T) {
	gaps := []coverageGap{}
	out := formatGapsJSON(gaps, 90.0)
	var parsed coverageJSONOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if len(parsed.Gaps) != 0 {
		t.Errorf("empty gaps JSON should have 0 gaps, got %d", len(parsed.Gaps))
	}
	if parsed.TotalCoverage != 90.0 {
		t.Errorf("TotalCoverage = %.1f, want 90.0", parsed.TotalCoverage)
	}
}

// --- parseTotalCoverage tests ---

func TestParseTotalCoverage(t *testing.T) {
	output := `internal/foo/bar.go:Bar 100.0%
internal/foo/bar.go:Uncovered 0.0%
total:                                                  (statements) 50.0%
`
	pct, err := parseTotalCoverage(output)
	if err != nil {
		t.Fatalf("parseTotalCoverage: %v", err)
	}
	if pct != 50.0 {
		t.Errorf("total coverage = %.1f, want 50.0", pct)
	}
}

func TestParseTotalCoverage_Missing(t *testing.T) {
	output := `internal/foo/bar.go:Bar 100.0%
`
	_, err := parseTotalCoverage(output)
	if err == nil {
		t.Fatal("expected error when total line missing")
	}
}

// --- CLI exit code tests (Finding 4) ---

func TestRunCoverageScan_InvalidFormat(t *testing.T) {
	res := runScanTest([]string{"--format=xml"}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for invalid format", res.exitCode)
	}
	if !strings.Contains(res.stderr, "unknown format") {
		t.Errorf("stderr should mention unknown format, got: %s", res.stderr)
	}
}

func TestRunCoverageScan_InvalidThreshold(t *testing.T) {
	res := runScanTest([]string{"--threshold=150"}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for out-of-range threshold", res.exitCode)
	}
}

func TestRunCoverageScan_NegativeThreshold(t *testing.T) {
	res := runScanTest([]string{"--threshold=-10"}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for negative threshold", res.exitCode)
	}
}

func TestRunCoverageScan_SkipTestWithoutCoverprofile(t *testing.T) {
	res := runScanTest([]string{"--skip-test"}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for --skip-test without --coverprofile", res.exitCode)
	}
	if !strings.Contains(res.stderr, "--skip-test requires --coverprofile") {
		t.Errorf("stderr should mention --skip-test requires --coverprofile, got: %s", res.stderr)
	}
}

func TestRunCoverageScan_InvalidPackage_FlagInjection(t *testing.T) {
	res := runScanTest([]string{"--package=-v"}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for package starting with -", res.exitCode)
	}
	if !strings.Contains(res.stderr, "must not start with '-'") {
		t.Errorf("stderr should mention flag injection, got: %s", res.stderr)
	}
}

func TestRunCoverageScan_InvalidPackage_HelpFlag(t *testing.T) {
	res := runScanTest([]string{"--package=--help"}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for --help as package", res.exitCode)
	}
}

func TestRunCoverageScan_InvalidPackage_BareWord(t *testing.T) {
	res := runScanTest([]string{"--package=single"}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for bare word package", res.exitCode)
	}
}

func TestRunCoverageScan_NonexistentPath(t *testing.T) {
	res := runScanTest([]string{"--path=/nonexistent/path"}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for nonexistent path", res.exitCode)
	}
}

func TestRunCoverageScan_NotDirectory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := runScanTest([]string{"--path=" + tmpFile}, nil)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 for file as path", res.exitCode)
	}
}

func TestRunCoverageScan_WithMockRunner_NoGaps(t *testing.T) {
	tmpDir := t.TempDir()

	funcOutput := `internal/foo/bar.go:Bar 100.0%
internal/foo/bar.go:Also 95.0%
total:                                                  97.5%
`
	runner := &mockRunner{
		combinedOutputFunc: map[string]outputResult{
			"go test":      {},
			"cover -func=": {data: []byte(funcOutput), err: nil},
		},
	}

	res := runScanTest([]string{"--path=" + tmpDir, "--format=json"}, runner)
	if res.exitCode != 0 {
		t.Errorf("exit code = %d, want 0 when no gaps found", res.exitCode)
	}
}

func TestRunCoverageScan_WithMockRunner_GapsFound(t *testing.T) {
	tmpDir := t.TempDir()

	funcOutput := `internal/foo/bar.go:Bar 100.0%
internal/foo/bar.go:Uncovered 30.0%
total:                                                  65.0%
`
	runner := &mockRunner{
		combinedOutputFunc: map[string]outputResult{
			"go test":      {},
			"cover -func=": {data: []byte(funcOutput), err: nil},
		},
	}

	res := runScanTest([]string{"--path=" + tmpDir, "--format=json"}, runner)
	if res.exitCode != 1 {
		t.Errorf("exit code = %d, want 1 when gaps found", res.exitCode)
	}
}

func TestRunCoverageScan_WithMockRunner_GoTestFails(t *testing.T) {
	tmpDir := t.TempDir()

	runner := &mockRunner{
		combinedOutputFunc: map[string]outputResult{
			"go test": {err: fmt.Errorf("go test: exit status 1")},
		},
	}

	res := runScanTest([]string{"--path=" + tmpDir}, runner)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 when go test fails", res.exitCode)
	}
	if !strings.Contains(res.stderr, "go test failed") {
		t.Errorf("stderr should mention go test failure, got: %s", res.stderr)
	}
}

func TestRunCoverageScan_WithMockRunner_CoverFuncFails(t *testing.T) {
	tmpDir := t.TempDir()

	runner := &mockRunner{
		combinedOutputFunc: map[string]outputResult{
			"go test":      {},
			"cover -func=": {data: nil, err: fmt.Errorf("cover: no profile")},
		},
	}

	res := runScanTest([]string{"--path=" + tmpDir}, runner)
	if res.exitCode != 2 {
		t.Errorf("exit code = %d, want 2 when cover -func fails", res.exitCode)
	}
	if !strings.Contains(res.stderr, "go tool cover -func") {
		t.Errorf("stderr should mention cover -func, got: %s", res.stderr)
	}
}

func TestRunCoverageScan_SkipTestWithCoverprofile(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a fake coverprofile
	covPath := filepath.Join(tmpDir, "cov.out")
	coverprofileContent := "mode: set\n" +
		"internal/foo/bar.go:10.1,12.16 5 5\n" +
		"internal/foo/bar.go:15.1,18.20 10 0\n"
	if err := os.WriteFile(covPath, []byte(coverprofileContent), 0o644); err != nil {
		t.Fatal(err)
	}

	funcOutput := `internal/foo/bar.go:10:   Bar 100.0%
internal/foo/bar.go:15:   Uncovered 0.0%
total:                                                  33.3%
`
	runner := &mockRunner{
		combinedOutputFunc: map[string]outputResult{
			"cover -func=": {data: []byte(funcOutput), err: nil},
		},
	}

	res := runScanTest([]string{
		"--path=" + tmpDir,
		"--skip-test",
		"--coverprofile=" + covPath,
		"--format=json",
	}, runner)
	if res.exitCode != 1 {
		t.Errorf("exit code = %d, want 1 when gaps found via skip-test", res.exitCode)
	}
}

func TestRunCoverageScan_JSONOutputStructure(t *testing.T) {
	tmpDir := t.TempDir()

	funcOutput := `internal/foo/bar.go:25:   Bar 100.0%
internal/foo/bar.go:40:   Uncovered 0.0%
total:                                                  50.0%
`
	runner := &mockRunner{
		combinedOutputFunc: map[string]outputResult{
			"go test":      {},
			"cover -func=": {data: []byte(funcOutput), err: nil},
		},
	}

	res := runScanTest([]string{"--path=" + tmpDir, "--format=json", "--threshold=80"}, runner)
	if res.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", res.exitCode)
	}

	var parsed coverageJSONOutput
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if parsed.TotalCoverage != 50.0 {
		t.Errorf("TotalCoverage = %.1f, want 50.0", parsed.TotalCoverage)
	}
	if len(parsed.Gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(parsed.Gaps))
	}
	g := parsed.Gaps[0]
	if g.File != "internal/foo/bar.go" {
		t.Errorf("File = %q, want %q", g.File, "internal/foo/bar.go")
	}
	if g.Function != "Uncovered" {
		t.Errorf("Function = %q, want %q", g.Function, "Uncovered")
	}
	if g.Coverage != 0.0 {
		t.Errorf("Coverage = %.1f, want 0.0", g.Coverage)
	}
	if g.Line != 40 {
		t.Errorf("Line = %d, want 40", g.Line)
	}
	if g.ThresholdGap != 80.0 {
		t.Errorf("ThresholdGap = %.1f, want 80.0", g.ThresholdGap)
	}
}

// --- Integration test: parse + filter with real function-level data ---

func TestCoverageParseAndFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping coverprofile parse integration in short mode")
	}

	funcOutput := `internal/foo/bar.go:Bar 33.3%
internal/baz/qux.go:Qux 100.0%
total:                                                  52.0%
`
	gaps, totalPct, err := filterGapsFromFuncOutput(funcOutput, 80.0)
	if err != nil {
		t.Fatalf("filterGapsFromFuncOutput: %v", err)
	}
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if !strings.Contains(gaps[0].File, "bar.go") {
		t.Errorf("gap file should contain bar.go, got %q", gaps[0].File)
	}
	if gaps[0].Function != "Bar" {
		t.Errorf("gap function = %q, want %q", gaps[0].Function, "Bar")
	}
	if gaps[0].Coverage != 33.3 {
		t.Errorf("gap coverage = %.1f%%, expected ~33.3%%", gaps[0].Coverage)
	}
	if totalPct != 52.0 {
		t.Errorf("totalPct = %.1f, want 52.0", totalPct)
	}
}
