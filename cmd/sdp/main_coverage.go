package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/coveragegate"
	"sdp_dev/internal/executil"
)

// coverageItem mirrors coveragegate.CoverageFunc for internal use in the
// coverage-scan command (avoids exporting the type from coveragegate while
// keeping the command self-contained).
type coverageItem struct {
	File       string  `json:"file"`
	Function   string  `json:"function"`
	Statements int     `json:"statements"`
	Coverage   float64 `json:"coverage"`
}

// coverageGap represents a file/function below the coverage threshold.
type coverageGap struct {
	File         string  `json:"file"`
	Function     string  `json:"function"`
	Coverage     float64 `json:"coverage"`
	ThresholdGap float64 `json:"threshold_gap"`
}

// parseCoverprofileForScan wraps coveragegate.ParseCoverprofile and converts
// the result to coverageItem slices for use in the coverage-scan command.
func parseCoverprofileForScan(path string) ([]coverageItem, int, int, error) {
	funcs, totalStmts, coveredStmts, err := coveragegate.ParseCoverprofile(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("coverage-scan: parse coverprofile: %w", err)
	}
	if totalStmts == 0 {
		return nil, 0, 0, fmt.Errorf("coverage-scan: coverprofile contains no statements")
	}
	items := make([]coverageItem, len(funcs))
	for i, f := range funcs {
		items[i] = coverageItem{
			File:       f.File,
			Function:   f.Function,
			Statements: f.Statements,
			Coverage:   f.Coverage,
		}
	}
	return items, totalStmts, coveredStmts, nil
}

// filterBelowThreshold returns items whose coverage is strictly below the
// given threshold, along with the gap (threshold - coverage).
func filterBelowThreshold(items []coverageItem, threshold float64) []coverageGap {
	var gaps []coverageGap
	for _, item := range items {
		if item.Coverage < threshold {
			gaps = append(gaps, coverageGap{
				File:         item.File,
				Function:     item.Function,
				Coverage:     item.Coverage,
				ThresholdGap: threshold - item.Coverage,
			})
		}
	}
	return gaps
}

// formatGapsText renders coverage gaps as a human-readable table.
func formatGapsText(gaps []coverageGap, threshold float64) string {
	if len(gaps) == 0 {
		return fmt.Sprintf("coverage-scan: All files above threshold (%.1f%%)\n", threshold)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "coverage-scan: %d file(s) below %.1f%% threshold\n\n", len(gaps), threshold)
	fmt.Fprintf(&b, "  %-50s %10s %10s\n", "FILE", "COVERAGE", "GAP")
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 72))
	for _, g := range gaps {
		fmt.Fprintf(&b, "  %-50s %9.1f%% %9.1fpp\n", g.File, g.Coverage, g.ThresholdGap)
	}
	return b.String()
}

// formatGapsJSON renders coverage gaps as a JSON array.
func formatGapsJSON(gaps []coverageGap) string {
	if gaps == nil {
		gaps = []coverageGap{}
	}
	out, err := json.MarshalIndent(gaps, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(out) + "\n"
}

func runCoverageScan(args []string) {
	fs := flag.NewFlagSet("coverage-scan", flag.ExitOnError)
	path := fs.String("path", ".", "target repository path")
	threshold := fs.Float64("threshold", 80.0, "minimum coverage percentage")
	format := fs.String("format", "text", "output format: text, json")
	shortMode := fs.Bool("short", false, "skip tests, parse existing coverprofile only")
	coverprofilePath := fs.String("coverprofile", "", "path to existing coverprofile file (overrides running go test)")
	_ = fs.Parse(args)

	// Validate format
	switch *format {
	case "text", "json":
	default:
		fmt.Fprintf(os.Stderr, "coverage-scan: unknown format %q (use text or json)\n", *format)
		os.Exit(2)
	}

	// Validate threshold
	if *threshold < 0 || *threshold > 100 {
		fmt.Fprintf(os.Stderr, "coverage-scan: threshold must be between 0 and 100, got %.1f\n", *threshold)
		os.Exit(2)
	}

	// Validate path
	info, err := os.Stat(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-scan: %v\n", err)
		os.Exit(2)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "coverage-scan: %q is not a directory\n", *path)
		os.Exit(2)
	}

	// Determine coverprofile file path
	covPath := *coverprofilePath
	if covPath == "" {
		covPath = filepath.Join(os.TempDir(), "sdp-cov.out")
	}

	// Run go test unless short mode or existing coverprofile provided
	if !*shortMode && *coverprofilePath == "" {
		runner := executil.GetDefaultRunner()
		ctx := context.Background()
		_, err := runner.CombinedOutput(ctx, *path,
			"go", "test", "-coverprofile="+covPath, "./...")
		if err != nil {
			fmt.Fprintf(os.Stderr, "coverage-scan: go test failed: %v\n", err)
			os.Exit(2)
		}
	}

	// Parse coverprofile
	items, totalStmts, coveredStmts, err := parseCoverprofileForScan(covPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-scan: %v\n", err)
		os.Exit(2)
	}

	// Filter below threshold
	gaps := filterBelowThreshold(items, *threshold)

	// Output
	switch *format {
	case "json":
		fmt.Print(formatGapsJSON(gaps))
	case "text":
		totalPct := float64(coveredStmts) / float64(totalStmts) * 100.0
		fmt.Fprintf(os.Stderr, "coverage-scan: total coverage %.1f%% (%d/%d statements)\n", totalPct, coveredStmts, totalStmts)
		fmt.Print(formatGapsText(gaps, *threshold))
	}

	// Exit code
	if len(gaps) > 0 {
		os.Exit(1)
	}
}
