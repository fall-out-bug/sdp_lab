package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/docsync"
	"sdp_dev/internal/orchestrate"
)

func main() {
	mode := flag.String("mode", "check", "Mode: check, fix, or changelog")
	format := flag.String("format", "text", "Output format: text or json")
	jsonOutput := flag.Bool("json", false, "Output in JSON format (shorthand for --format json)")
	strict := flag.Bool("strict", false, "Treat documentation consistency findings as errors")
	since := flag.String("since", "", "Git range for changelog update (default: HEAD~1..HEAD)")
	projectRoot := flag.String("project-root", "", "Project root (auto-detected if empty)")
	flag.Parse()

	// --json takes precedence over --format
	if *jsonOutput {
		*format = "json"
	}

	root := *projectRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: get cwd: %v\n", err)
			os.Exit(1)
		}
		root, err = orchestrate.FindProjectRoot(wd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: find project root: %v\n", err)
			os.Exit(1)
		}
	}

	switch *mode {
	case "check":
		report, err := docsync.CheckConsistency(root, *strict)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: documentation consistency check failed: %v\n", err)
			os.Exit(1)
		}
		emitReport(report, *format)
		if report.HasErrors() {
			os.Exit(2)
		}
	case "fix":
		fixReport, err := docsync.FixConsistency(root, *strict)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: documentation fix failed: %v\n", err)
			os.Exit(1)
		}
		emitFixReport(fixReport, *format)
		if len(fixReport.Unresolved) > 0 {
			os.Exit(2)
		}
	case "changelog":
		path, err := docsync.UpdateChangelog(root, *since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: update changelog: %v\n", err)
			os.Exit(1)
		}
		if path == "" {
			fmt.Println("No commits found for range; changelog unchanged")
			return
		}
		fmt.Printf("Updated changelog: %s\n", path)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown mode %q (expected check, fix, or changelog)\n", *mode)
		os.Exit(1)
	}
}

func emitReport(report docsync.ConsistencyReport, format string) {
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "error: encode output: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(report.Issues) == 0 {
		fmt.Println("OK: documentation consistency passed")
		return
	}

	errors := 0
	warnings := 0
	for _, issue := range report.Issues {
		switch issue.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		}
	}

	fmt.Printf("Documentation consistency: %d error(s), %d warning(s)\n", errors, warnings)
	for _, issue := range report.Issues {
		if issue.File != "" {
			fmt.Printf("- [%s] %s: %s\n", issue.Severity, issue.File, issue.Message)
		} else {
			fmt.Printf("- [%s] %s\n", issue.Severity, issue.Message)
		}
	}
}

func emitFixReport(report docsync.FixReport, format string) {
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "error: encode output: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(report.Fixed) == 0 && len(report.Unresolved) == 0 {
		fmt.Println("OK: nothing to fix, documentation is consistent")
		return
	}

	fmt.Printf("Fix report: %d fix(es) applied, %d unresolved issue(s)\n", len(report.Fixed), len(report.Unresolved))
	for _, fix := range report.Fixed {
		fmt.Printf("  FIXED [%s] %s: %s -> %s\n", fix.Fix, fix.File, fix.Before, fix.After)
	}
	for _, issue := range report.Unresolved {
		if issue.File != "" {
			fmt.Printf("  UNRESOLVED [%s] %s: %s\n", issue.Severity, issue.File, issue.Message)
		} else {
			fmt.Printf("  UNRESOLVED [%s] %s\n", issue.Severity, issue.Message)
		}
	}
}
