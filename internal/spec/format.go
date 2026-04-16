package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteArtifact writes the SpecReport as JSON to the output directory.
func WriteArtifact(outputDir string, report *SpecReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("spec: create output dir: %w", err)
	}
	path := filepath.Join(outputDir, "spec.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("spec: marshal report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("spec: write report: %w", err)
	}
	return path, nil
}

// FormatText returns a human-readable summary of a SpecReport.
func FormatText(report *SpecReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Spec Report: %s\n", report.Repo)
	fmt.Fprintf(&b, "Generated:   %s\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Duration:    %dms\n\n", report.DurationMs)
	fmt.Fprintf(&b, "API Contracts:\n  Endpoints: %d\n", report.APIContracts.Total)
	for _, ep := range report.APIContracts.HTTPEndpoints {
		fmt.Fprintf(&b, "    %-6s %-30s %s\n", ep.Method, ep.Path, ep.Handler)
	}
	fmt.Fprintf(&b, "\nBusiness Rules:\n  Rules: %d\n", report.BusinessRules.Total)
	cats := map[string]int{}
	for _, r := range report.BusinessRules.Validations {
		cats[r.Category]++
	}
	for cat, n := range cats {
		fmt.Fprintf(&b, "    %-20s %d\n", cat, n)
	}
	fmt.Fprintf(&b, "\nCoverage:\n  Files scanned:    %d\n  Files with specs: %d\n  Spec density:     %.1f%%\n",
		report.Coverage.FilesScanned, report.Coverage.FilesWithSpecs, report.Coverage.SpecDensity*100)
	return b.String()
}
