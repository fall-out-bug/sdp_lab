package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
	"github.com/fall-out-bug/sdp_lab/internal/workstream"
)

func main() {
	format := flag.String("format", "text", "Output format: text or json")
	strictBeads := flag.Bool("strict-beads", false, "Require numeric Beads IDs (sdplab-<number>)")
	strict := flag.Bool("strict", false, "Treat legacy/protocol drift findings as errors")
	projectRoot := flag.String("project-root", "", "Project root (auto-detected if empty)")
	lintSkills := flag.Bool("lint-skills", false, "Lint .agents/skills/*.md frontmatter and harness-neutrality (F127-08)")
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

	if *lintSkills {
		skillReport, err := workstream.ValidateSkills(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: lint skills: %v\n", err)
			os.Exit(1)
		}
		emitSkillReport(skillReport, *format)
		if skillReport.HasErrors() {
			os.Exit(2)
		}
		return
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

func emitSkillReport(report workstream.SkillLintResult, format string) {
	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "error: encode output: %v\n", err)
			os.Exit(1)
		}
	default:
		printSkillReport(report)
	}
}

func printSkillReport(report workstream.SkillLintResult) {
	if len(report.Issues) == 0 {
		fmt.Println("OK: skill lint passed")
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
	fmt.Printf("Skill lint: %d error(s), %d warning(s)\n", errors, warnings)
	for _, issue := range report.Issues {
		if issue.File != "" {
			fmt.Printf("- [%s] %s: %s\n", issue.Severity, issue.File, issue.Message)
		} else {
			fmt.Printf("- [%s] %s\n", issue.Severity, issue.Message)
		}
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
