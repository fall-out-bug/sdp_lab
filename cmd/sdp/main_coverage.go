package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"sdp_dev/internal/executil"
	"sdp_dev/internal/testwriter"
)

// coverageGap represents a function below the coverage threshold.
// The output matches testwriter.CoverageGap so that coverage-scan output
// feeds directly into testwriter.
type coverageGap struct {
	File         string  `json:"file"`
	Function     string  `json:"function"`
	Coverage     float64 `json:"coverage"`
	Line         int     `json:"line"`
	ThresholdGap float64 `json:"threshold_gap"`
}

// coverageJSONOutput wraps JSON output with total coverage metadata.
type coverageJSONOutput struct {
	TotalCoverage float64       `json:"total_coverage"`
	Gaps          []coverageGap `json:"gaps"`
}

// validatePackagePattern rejects obviously invalid package patterns.
// It prevents flag injection (inputs starting with "-") and rejects bare
// words that are not valid Go package patterns.
func validatePackagePattern(pkg string) error {
	if pkg == "" {
		return fmt.Errorf("package pattern must not be empty")
	}
	if strings.HasPrefix(pkg, "-") {
		return fmt.Errorf("package pattern must not start with '-': %q", pkg)
	}
	// Valid patterns:
	//   - relative: ./..., ../...
	//   - absolute/module path: contains / or . (e.g. github.com/..., sdp_dev/...)
	// Reject bare words like "single" which are likely mistakes.
	if !(strings.HasPrefix(pkg, "./") ||
		strings.HasPrefix(pkg, "../") ||
		strings.Contains(pkg, "/") ||
		strings.Contains(pkg, ".")) {
		return fmt.Errorf("package pattern must be a relative path (./...), parent path (../...), or module path (github.com/...), got %q", pkg)
	}
	return nil
}

// runCoverFunc runs `go tool cover -func=<coverprofile>` in the given directory
// and returns the raw output. This provides per-function granularity with line
// numbers, which is the format testwriter.ParseCoverGaps expects.
func runCoverFunc(ctx context.Context, runner executil.CommandRunner, dir, coverprofile string) (string, error) {
	out, err := runner.CombinedOutput(ctx, dir, "go", "tool", "cover", "-func="+coverprofile)
	if err != nil {
		return "", fmt.Errorf("go tool cover -func: %w\n%s", err, out)
	}
	return string(out), nil
}

// runGoTest runs `go test -coverprofile=<path> <pkg>` in the given directory.
func runGoTest(ctx context.Context, runner executil.CommandRunner, dir, coverprofile, pkg string) error {
	out, err := runner.CombinedOutput(ctx, dir, "go", "test", "-coverprofile="+coverprofile, pkg)
	if err != nil {
		return fmt.Errorf("go test failed: %w\n%s", err, out)
	}
	return nil
}

// parseTotalCoverage extracts the total coverage percentage from `go tool cover -func` output.
func parseTotalCoverage(funcOutput string) (float64, error) {
	for _, line := range strings.Split(funcOutput, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "total:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		pctStr := fields[len(fields)-1]
		pctStr = strings.TrimSuffix(pctStr, "%")
		var pct float64
		if _, err := fmt.Sscanf(pctStr, "%f", &pct); err != nil {
			return 0, fmt.Errorf("parse total coverage from %q: %w", line, err)
		}
		return pct, nil
	}
	return 0, fmt.Errorf("no total: line found in go tool cover -func output")
}

// filterGapsFromFuncOutput parses go tool cover -func output and returns
// functions below the threshold as coverageGap structs with line numbers.
func filterGapsFromFuncOutput(funcOutput string, threshold float64) ([]coverageGap, float64, error) {
	// Parse gaps using testwriter.ParseCoverGaps (fail-closed on empty)
	gaps, err := testwriter.ParseCoverGaps(funcOutput, threshold)
	if err != nil {
		return nil, 0, fmt.Errorf("parse function coverage: %w", err)
	}

	// Extract total coverage
	totalPct, err := parseTotalCoverage(funcOutput)
	if err != nil {
		return nil, 0, fmt.Errorf("extract total coverage: %w", err)
	}

	// Convert testwriter.CoverageGap to our coverageGap with ThresholdGap
	result := make([]coverageGap, len(gaps))
	for i, g := range gaps {
		result[i] = coverageGap{
			File:         g.File,
			Function:     g.Function,
			Coverage:     g.Coverage,
			Line:         g.Line,
			ThresholdGap: threshold - g.Coverage,
		}
	}

	return result, totalPct, nil
}

// formatGapsText renders coverage gaps as a human-readable table.
func formatGapsText(gaps []coverageGap, threshold float64) string {
	if len(gaps) == 0 {
		return fmt.Sprintf("coverage-scan: All functions above threshold (%.1f%%)\n", threshold)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "coverage-scan: %d function(s) below %.1f%% threshold\n\n", len(gaps), threshold)
	fmt.Fprintf(&b, "  %-50s %-20s %10s %10s\n", "FILE", "FUNCTION", "COVERAGE", "GAP")
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 95))
	for _, g := range gaps {
		fmt.Fprintf(&b, "  %-50s %-20s %9.1f%% %9.1fpp\n", g.File, g.Function, g.Coverage, g.ThresholdGap)
	}
	return b.String()
}

// formatGapsJSON renders coverage gaps as a JSON object with total coverage.
func formatGapsJSON(gaps []coverageGap, totalCoverage float64) string {
	if gaps == nil {
		gaps = []coverageGap{}
	}
	out := coverageJSONOutput{
		TotalCoverage: totalCoverage,
		Gaps:          gaps,
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "{\"total_coverage\":0,\"gaps\":[]}\n"
	}
	return string(b) + "\n"
}

// runCoverageScan executes the coverage-scan subcommand. It returns an exit
// code: 0 = no gaps, 1 = gaps found, 2 = error. This design allows testing
// without os.Exit terminating the process.
func runCoverageScan(args []string) int {
	return runCoverageScanWithWriters(args, os.Stdout, os.Stderr, executil.GetDefaultRunner())
}

// runCoverageScanWithWriters is the testable core of runCoverageScan.
// stdout and stderr allow output capture in tests; runner allows mocking subprocess calls.
func runCoverageScanWithWriters(args []string, stdout, stderr io.Writer, runner executil.CommandRunner) int {
	fs := flag.NewFlagSet("coverage-scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("path", ".", "target repository path")
	threshold := fs.Float64("threshold", 80.0, "minimum coverage percentage")
	format := fs.String("format", "text", "output format: text, json")
	skipTest := fs.Bool("skip-test", false, "skip running tests; parse existing coverprofile only (requires --coverprofile)")
	coverprofilePath := fs.String("coverprofile", "", "path to existing coverprofile file (overrides running go test)")
	pkgPattern := fs.String("package", "./...", "package pattern to test (e.g. ./internal/foo/...)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "coverage-scan: %v\n", err)
		return 2
	}

	// Validate format
	switch *format {
	case "text", "json":
	default:
		fmt.Fprintf(stderr, "coverage-scan: unknown format %q (use text or json)\n", *format)
		return 2
	}

	// Validate threshold
	if *threshold < 0 || *threshold > 100 {
		fmt.Fprintf(stderr, "coverage-scan: threshold must be between 0 and 100, got %.1f\n", *threshold)
		return 2
	}

	// Validate --package to prevent flag injection
	if err := validatePackagePattern(*pkgPattern); err != nil {
		fmt.Fprintf(stderr, "coverage-scan: %v\n", err)
		return 2
	}

	// Validate path
	info, err := os.Stat(*path)
	if err != nil {
		fmt.Fprintf(stderr, "coverage-scan: %v\n", err)
		return 2
	}
	if !info.IsDir() {
		fmt.Fprintf(stderr, "coverage-scan: %q is not a directory\n", *path)
		return 2
	}

	// When skip-test is set, a coverprofile must be provided
	if *skipTest && *coverprofilePath == "" {
		fmt.Fprintf(stderr, "coverage-scan: --skip-test requires --coverprofile to specify an existing coverprofile file\n")
		return 2
	}

	// Determine coverprofile file path
	covPath := *coverprofilePath
	autoGenerated := false
	if covPath == "" {
		tmpFile, err := os.CreateTemp("", "sdp-cov-*.out")
		if err != nil {
			fmt.Fprintf(stderr, "coverage-scan: create temp file: %v\n", err)
			return 2
		}
		covPath = tmpFile.Name()
		tmpFile.Close()
		autoGenerated = true
	}

	// Clean up auto-generated temp file
	if autoGenerated {
		defer os.Remove(covPath)
	}

	// Run go test unless skip-test is set
	ctx := context.Background()

	if !*skipTest {
		if err := runGoTest(ctx, runner, *path, covPath, *pkgPattern); err != nil {
			fmt.Fprintf(stderr, "coverage-scan: %v\n", err)
			return 2
		}
	}

	// Run go tool cover -func to get per-function coverage data
	funcOutput, err := runCoverFunc(ctx, runner, *path, covPath)
	if err != nil {
		fmt.Fprintf(stderr, "coverage-scan: %v\n", err)
		return 2
	}

	// Parse function-level gaps and total coverage
	gaps, totalPct, err := filterGapsFromFuncOutput(funcOutput, *threshold)
	if err != nil {
		fmt.Fprintf(stderr, "coverage-scan: %v\n", err)
		return 2
	}

	// Output
	switch *format {
	case "json":
		fmt.Fprint(stdout, formatGapsJSON(gaps, totalPct))
	case "text":
		fmt.Fprintf(stderr, "coverage-scan: total coverage %.1f%%\n", totalPct)
		fmt.Fprint(stdout, formatGapsText(gaps, *threshold))
	}

	// Exit code
	if len(gaps) > 0 {
		return 1
	}
	return 0
}
