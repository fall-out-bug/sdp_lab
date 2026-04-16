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
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp spec [--format json|text] [--category api|rules|invariants|sla] [--output DIR] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	switch *format {
	case "json", "text":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}

	report, err := spec.Run(repoPath)
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
