package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/scout"
)

func runScout(args []string) {
	fs := flag.NewFlagSet("scout", flag.ExitOnError)
	format := fs.String("format", "json", "output format: json, text, card")
	output := fs.String("output", "", "write output to directory as .sdp/scout.json")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp scout [--format json|text|card] [--output DIR] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	// Scout implementation (phases) is delivered in WS-02 and WS-03.
	// This stub validates the CLI interface and ProjectCard contract.
	fmt.Fprintf(os.Stderr, "scout: %s (format=%s output=%s)\n", repoPath, *format, *output)

	// Emit empty card as contract proof. Full implementation in 00-120-02/03.
	card := scout.ProjectCard{
		Version:    "1.0.0",
		Identity:   scout.Identity{Name: repoPath},
		Health:     scout.HealthSignals{
			CommitFrequency:  scout.Unknown,
			Staleness:        scout.Unknown,
			TestCoverageHint: scout.Unknown,
			ComplexityHint:   scout.Unknown,
		},
	}

	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
