package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/orchestrate"
	"sdp_dev/internal/workstream"
)

func main() {
	format := flag.String("format", "text", "Output format: text or json")
	strictBeads := flag.Bool("strict-beads", false, "Require concrete Beads IDs (sdplab-<id>)")
	strict := flag.Bool("strict", false, "Treat legacy/protocol drift findings as errors")
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

	report, err := workstream.ValidateProtocol(root, *strictBeads, *strict)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: validate protocol: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "error: encode output: %v\n", err)
			os.Exit(1)
		}
	default:
		printText(report)
	}

	if report.HasErrors() {
		os.Exit(2)
	}
}

func printText(report workstream.ValidationReport) {
	if len(report.Issues) == 0 {
		fmt.Println("OK: protocol validation passed")
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

	fmt.Printf("Protocol validation: %d error(s), %d warning(s)\n", errors, warnings)
	for _, issue := range report.Issues {
		if issue.File != "" {
			fmt.Printf("- [%s] %s: %s\n", issue.Severity, issue.File, issue.Message)
		} else {
			fmt.Printf("- [%s] %s\n", issue.Severity, issue.Message)
		}
	}
}
