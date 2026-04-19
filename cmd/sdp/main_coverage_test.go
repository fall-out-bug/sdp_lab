package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/coveragegate"
)

// writeTestCoverprofile creates a temporary coverprofile file for testing.
func writeTestCoverprofile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cov.out")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- Filter logic tests ---

func TestCoverageFilterBelowThreshold(t *testing.T) {
	funcs := []coveragegate.CoverageFunc{
		{File: "a.go", Function: "a.go", Statements: 10, Coverage: 50.0},
		{File: "b.go", Function: "b.go", Statements: 10, Coverage: 90.0},
		{File: "c.go", Function: "c.go", Statements: 10, Coverage: 80.0},
	}
	gaps := filterBelowThreshold(funcs, 80.0)
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if gaps[0].File != "a.go" {
		t.Errorf("gap file = %q, want %q", gaps[0].File, "a.go")
	}
	if gaps[0].ThresholdGap != 30.0 {
		t.Errorf("threshold gap = %.1f, want 30.0", gaps[0].ThresholdGap)
	}
}

func TestCoverageFilterAllAboveThreshold(t *testing.T) {
	funcs := []coveragegate.CoverageFunc{
		{File: "a.go", Function: "a.go", Statements: 10, Coverage: 90.0},
		{File: "b.go", Function: "b.go", Statements: 10, Coverage: 100.0},
	}
	gaps := filterBelowThreshold(funcs, 80.0)
	if len(gaps) != 0 {
		t.Fatalf("expected 0 gaps, got %d", len(gaps))
	}
}

func TestCoverageFilterExactThreshold(t *testing.T) {
	funcs := []coveragegate.CoverageFunc{
		{File: "a.go", Function: "a.go", Statements: 10, Coverage: 80.0},
	}
	gaps := filterBelowThreshold(funcs, 80.0)
	if len(gaps) != 0 {
		t.Fatalf("coverage at exact threshold should not be a gap, got %d gaps", len(gaps))
	}
}

func TestCoverageFilterAllBelow(t *testing.T) {
	funcs := []coveragegate.CoverageFunc{
		{File: "a.go", Function: "a.go", Statements: 10, Coverage: 10.0},
		{File: "b.go", Function: "b.go", Statements: 10, Coverage: 20.0},
	}
	gaps := filterBelowThreshold(funcs, 80.0)
	if len(gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d", len(gaps))
	}
}

// --- Text formatting tests ---

func TestCoverageFormatText(t *testing.T) {
	gaps := []coverageGap{
		{File: "internal/foo/bar.go", Function: "internal/foo/bar.go", Coverage: 45.5, ThresholdGap: 34.5},
		{File: "cmd/main.go", Function: "cmd/main.go", Coverage: 72.0, ThresholdGap: 8.0},
	}
	out := formatGapsText(gaps, 80.0)
	if !strings.Contains(out, "internal/foo/bar.go") {
		t.Error("text output should contain file path")
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
	if !strings.Contains(out, "All files above threshold") {
		t.Errorf("expected 'All files above threshold' message, got: %s", out)
	}
}

// --- JSON formatting tests ---

func TestCoverageFormatJSON(t *testing.T) {
	gaps := []coverageGap{
		{File: "a.go", Function: "a.go", Coverage: 50.0, ThresholdGap: 30.0},
		{File: "b.go", Function: "b.go", Coverage: 70.0, ThresholdGap: 10.0},
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
	if parsed.Gaps[0].Coverage != 50.0 {
		t.Errorf("parsed.Gaps[0].Coverage = %.1f, want 50.0", parsed.Gaps[0].Coverage)
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

// --- Integration test: parse coverprofile + filter ---

func TestCoverageParseAndFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping coverprofile parse integration in short mode")
	}
	coverprofile := "mode: set\n" +
		"internal/foo/bar.go:10.1,12.16 5 1\n" +
		"internal/foo/bar.go:15.1,18.20 10 0\n" +
		"internal/baz/qux.go:5.1,8.10 8 8\n"

	path := writeTestCoverprofile(t, coverprofile)

	funcs, _, _, err := parseCoverprofileForScan(path)
	if err != nil {
		t.Fatalf("parseCoverprofileForScan: %v", err)
	}
	if len(funcs) != 2 {
		t.Fatalf("expected 2 file entries, got %d", len(funcs))
	}

	// bar.go: 5/15 covered = 33.3%
	// qux.go: 8/8 covered = 100.0%
	gaps := filterBelowThreshold(funcs, 80.0)
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if !strings.Contains(gaps[0].File, "bar.go") {
		t.Errorf("gap file should contain bar.go, got %q", gaps[0].File)
	}
	// C-07: verify coverage percentage is approximately 33.3%
	expectedCov := 100.0 * 5.0 / 15.0 // 33.3...
	if math.Abs(gaps[0].Coverage-expectedCov) > 0.2 {
		t.Errorf("gap coverage = %.1f%%, expected ~%.1f%%", gaps[0].Coverage, expectedCov)
	}
}

func TestCoverageParseEmptyCoverprofile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping coverprofile parse integration in short mode")
	}
	path := writeTestCoverprofile(t, "mode: set\n")
	_, _, _, err := parseCoverprofileForScan(path)
	if err == nil {
		t.Fatal("expected error for empty coverprofile, got nil")
	}
	if !strings.Contains(err.Error(), "no statements") {
		t.Errorf("error should mention 'no statements', got: %v", err)
	}
}

func TestCoverageParseMissingFile(t *testing.T) {
	_, _, _, err := parseCoverprofileForScan("/nonexistent/path/cov.out")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
