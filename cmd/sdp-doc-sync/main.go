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
	mode := flag.String("mode", "check", "Mode: check or changelog")
	format := flag.String("format", "text", "Output format: text or json")
	strict := flag.Bool("strict", false, "Treat documentation consistency findings as errors")
	since := flag.String("since", "", "Git range for changelog update (default: HEAD~1..HEAD)")
	projectRoot := flag.String("project-root", "", "Project root (auto-detected if empty)")
	flag.Parse()

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
		fmt.Fprintf(os.Stderr, "error: unknown mode %q (expected check or changelog)\n", *mode)
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
