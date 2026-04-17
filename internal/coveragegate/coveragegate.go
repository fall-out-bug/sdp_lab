// Package coveragegate implements a coverage enforcement gate for CI.
// It parses Go coverprofile output, compares total coverage against a
// baseline, and reports whether the change exceeds an allowed threshold.
package coveragegate

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CoverageReport holds the result of a coverage gate check.
type CoverageReport struct {
	TotalStatements int     `json:"total_statements"`
	CoveredStatements int   `json:"covered_statements"`
	TotalCoverage   float64 `json:"total_coverage"`
	Baseline        float64 `json:"baseline"`
	Delta           float64 `json:"delta"`
	Threshold       float64 `json:"threshold"`
	Passed          bool    `json:"passed"`
	Message         string  `json:"message"`
}

// CoverageFunc holds per-function coverage data parsed from coverprofile.
type CoverageFunc struct {
	File       string  `json:"file"`
	Function   string  `json:"function"`
	Statements int     `json:"statements"`
	Coverage   float64 `json:"coverage"`
}

// DefaultThreshold is the maximum allowed drop in percentage points from baseline.
const DefaultThreshold = 2.0

// DefaultMetricsPath is the path to the coverage baseline file relative to project root.
const DefaultMetricsPath = ".sdp/metrics/coverage.txt"

// CheckCoverage runs the coverage gate check against a coverprofile file.
// projectRoot is the root of the repository (for reading/writing baseline).
// coverprofilePath is the path to the Go coverprofile output (cov.out).
func CheckCoverage(projectRoot, coverprofilePath string) (CoverageReport, error) {
	return CheckCoverageWithThreshold(projectRoot, coverprofilePath, DefaultThreshold)
}

// CheckCoverageWithThreshold runs the coverage gate with a custom threshold.
func CheckCoverageWithThreshold(projectRoot, coverprofilePath string, threshold float64) (CoverageReport, error) {
	if threshold < 0 {
		return CoverageReport{}, fmt.Errorf("coveragegate: threshold must be >= 0, got %.2f", threshold)
	}

	funcs, totalStmts, coveredStmts, err := ParseCoverprofile(coverprofilePath)
	if err != nil {
		return CoverageReport{}, fmt.Errorf("coveragegate: parse coverprofile: %w", err)
	}

	if totalStmts == 0 {
		return CoverageReport{}, fmt.Errorf("coveragegate: coverprofile contains no statements")
	}

	totalPct := float64(coveredStmts) / float64(totalStmts) * 100.0

	baseline, err := ReadBaseline(projectRoot)
	if err != nil {
		return CoverageReport{}, fmt.Errorf("coveragegate: read baseline: %w", err)
	}

	delta := totalPct - baseline
	passed := delta >= -threshold

	report := CoverageReport{
		TotalStatements: totalStmts,
		CoveredStatements: coveredStmts,
		TotalCoverage:   roundTo1(totalPct),
		Baseline:        baseline,
		Delta:           roundTo1(delta),
		Threshold:       threshold,
		Passed:          passed,
	}

	// Keep a reference to parsed functions (for potential future use)
	_ = funcs

	if passed {
		report.Message = fmt.Sprintf(
			"coverage gate PASSED: %.1f%% (baseline %.1f%%, delta %.1fpp, threshold -%.1fpp)",
			totalPct, baseline, delta, threshold,
		)
	} else {
		report.Message = fmt.Sprintf(
			"coverage gate FAILED: %.1f%% (baseline %.1f%%, delta %.1fpp exceeds threshold -%.1fpp)",
			totalPct, baseline, delta, threshold,
		)
	}

	return report, nil
}

// ParseCoverprofile reads a Go coverprofile file and returns per-function
// coverage data along with aggregate totals. It replicates the output format
// of `go tool cover -func=cov.out` but in pure Go (no shell-out).
func ParseCoverprofile(path string) ([]CoverageFunc, int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// Accumulate statement counts per file.
	// coverprofile lines look like:
	//   path/file.go:startLine.startCol,endLine.endCol numstmt count
	// Example:
	//   internal/foo/bar.go:10.1,12.16 3 1
	type fileKey struct {
		file string
	}
	type fileAccum struct {
		totalStmts   int
		coveredStmts int
	}

	accums := make(map[fileKey]*fileAccum)
	var order []fileKey // preserve insertion order for deterministic output

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip mode line and empty lines.
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		// Format: "file:range numstmts count"
		// Fields are space-separated; file:range is one token.
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Parse file path from file:range token.
		fileAndRange := parts[0]
		lastColon := strings.LastIndex(fileAndRange, ":")
		if lastColon < 0 {
			continue
		}
		filePath := fileAndRange[:lastColon]

		numStmts, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		count, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		key := fileKey{file: filePath}
		acc, exists := accums[key]
		if !exists {
			acc = &fileAccum{}
			accums[key] = acc
			order = append(order, key)
		}
		acc.totalStmts += numStmts
		if count > 0 {
			acc.coveredStmts += numStmts
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("scan coverprofile: %w", err)
	}

	var result []CoverageFunc
	totalStmts := 0
	coveredStmts := 0

	for _, key := range order {
		acc := accums[key]
		totalStmts += acc.totalStmts
		coveredStmts += acc.coveredStmts

		var pct float64
		if acc.totalStmts > 0 {
			pct = float64(acc.coveredStmts) / float64(acc.totalStmts) * 100.0
		}

		result = append(result, CoverageFunc{
			File:       key.file,
			Function:   key.file,
			Statements: acc.totalStmts,
			Coverage:   roundTo1(pct),
		})
	}

	return result, totalStmts, coveredStmts, nil
}

// ParseFuncOutput parses the output of `go tool cover -func=cov.out` and
// extracts the total coverage percentage. Each line looks like:
//
//	path/file.go:function   percentage
//	total:                  (statements)   percentage%
func ParseFuncOutput(output string) (float64, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "total:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		last := fields[len(fields)-1]
		last = strings.TrimSuffix(last, "%")
		pct, err := strconv.ParseFloat(last, 64)
		if err != nil {
			return 0, fmt.Errorf("parse total coverage percentage from %q: %w", line, err)
		}
		return pct, nil
	}
	return 0, fmt.Errorf("no total: line found in cover -func output")
}

// ReadBaseline reads the coverage baseline from the metrics file.
// If the file does not exist, it returns 0.0 with no error (first run).
func ReadBaseline(projectRoot string) (float64, error) {
	path := filepath.Join(projectRoot, DefaultMetricsPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0.0, nil
		}
		return 0, fmt.Errorf("read baseline %s: %w", path, err)
	}

	line := strings.TrimSpace(string(data))
	if line == "" {
		return 0.0, nil
	}

	pct, err := strconv.ParseFloat(line, 64)
	if err != nil {
		return 0, fmt.Errorf("parse baseline percentage from %q: %w", line, err)
	}

	return pct, nil
}

// WriteBaseline writes the current coverage percentage to the metrics file.
// It creates the metrics directory if it doesn't exist.
func WriteBaseline(projectRoot string, pct float64) error {
	dir := filepath.Join(projectRoot, filepath.Dir(DefaultMetricsPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create metrics dir %s: %w", dir, err)
	}

	path := filepath.Join(projectRoot, DefaultMetricsPath)
	content := fmt.Sprintf("%.1f\n", pct)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write baseline %s: %w", path, err)
	}

	return nil
}

// roundTo1 rounds a float64 to one decimal place.
func roundTo1(v float64) float64 {
	return math.Round(v*10) / 10
}
