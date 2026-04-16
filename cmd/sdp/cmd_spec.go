package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/spec"
)

func runSpec(args []string) {
	fs := flag.NewFlagSet("spec", flag.ExitOnError)
	format := fs.String("format", "json", "output format: json, text")
	output := fs.String("output", "", "write .sdp/specs/ to this directory")
	category := fs.String("category", "", "filter: api, rules, or empty for all")
	enrich := fs.Bool("enrich", false, "opt-in: attempt LLM enrichment (stub)")
	diff := fs.Bool("diff", false, "compare two spec snapshots")
	_ = fs.Parse(args)

	// Diff mode: compare two JSON snapshots
	if *diff {
		if fs.NArg() != 2 {
			fmt.Fprintln(os.Stderr, "usage: sdp spec --diff <old.json> <new.json>")
			os.Exit(2)
		}
		runDiff(fs.Arg(0), fs.Arg(1), *format)
		return
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp spec [--format json|text] [--category api|rules|invariants|sla] [--output DIR] [--enrich] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	switch *format {
	case "json", "text":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}

	report, err := spec.RunWithOptions(repoPath, spec.RunOptions{Enrich: *enrich})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: spec failed: %v\n", err)
		os.Exit(1)
	}

	// Apply category filter
	switch *category {
	case "api":
		report.BusinessRules = spec.BusinessRules{}
		report.Invariants = spec.Invariants{}
		report.SLAParameters = spec.SLAParameters{}
	case "rules":
		report.APIContracts = spec.APIContracts{}
		report.Invariants = spec.Invariants{}
		report.SLAParameters = spec.SLAParameters{}
	case "invariants":
		report.APIContracts = spec.APIContracts{}
		report.BusinessRules = spec.BusinessRules{}
		report.SLAParameters = spec.SLAParameters{}
	case "sla":
		report.APIContracts = spec.APIContracts{}
		report.BusinessRules = spec.BusinessRules{}
		report.Invariants = spec.Invariants{}
	case "":
		// no filter
	default:
		fmt.Fprintf(os.Stderr, "error: unknown category %q (use api, rules, invariants, sla, or leave empty)\n", *category)
		os.Exit(2)
	}

	// Write artifact if requested
	if *output != "" {
		path, werr := spec.WriteArtifact(*output, report)
		if werr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write artifact: %v\n", werr)
		} else {
			fmt.Fprintf(os.Stderr, "artifact: %s\n", path)
		}
	}

	// Format output
	switch *format {
	case "json":
		data, jerr := json.MarshalIndent(report, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(data))
	case "text":
		fmt.Print(spec.FormatText(report))
	}
}

// runDiff executes the diff sub-mode.
func runDiff(oldPath, newPath, format string) {
	diff, err := spec.DiffSpecs(oldPath, newPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: diff failed: %v\n", err)
		os.Exit(1)
	}
	switch format {
	case "json":
		data, jerr := json.MarshalIndent(diff, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(data))
	case "text":
		fmt.Print(formatDiffText(diff))
	}
}

// formatDiffText produces a human-readable diff summary.
func formatDiffText(d *spec.SpecDiff) string {
	total := d.Summary.Added + d.Summary.Removed + d.Summary.Modified
	result := fmt.Sprintf("Spec Diff: %s → %s\n", d.OldSnapshot, d.NewSnapshot)
	result += fmt.Sprintf("Generated:  %s\n\n", d.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	result += fmt.Sprintf("Changes:    %d total (added: %d, removed: %d, modified: %d)\n",
		total, d.Summary.Added, d.Summary.Removed, d.Summary.Modified)
	printChanges := func(label string, changes []spec.Change) {
		if len(changes) == 0 {
			return
		}
		result += fmt.Sprintf("\n%s:\n", label)
		for _, c := range changes {
			switch c.Category {
			case "added":
				result += fmt.Sprintf("  + %s: %s\n", c.Key, c.New)
			case "removed":
				result += fmt.Sprintf("  - %s: %s\n", c.Key, c.Old)
			case "modified":
				result += fmt.Sprintf("  ~ %s: %s → %s (%s)\n", c.Key, c.Old, c.New, c.Detail)
			}
		}
	}
	printChanges("API Changes", d.APIChanges)
	printChanges("Rule Changes", d.RuleChanges)
	printChanges("Invariant Changes", d.InvChanges)
	printChanges("SLA Changes", d.SLAChanges)
	if total == 0 {
		result += "\nNo changes detected.\n"
	}
	return result
}
