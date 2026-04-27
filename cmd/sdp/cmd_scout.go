package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fall-out-bug/sdp_lab/internal/scout"
)

func runScout(args []string) {
	fs := flag.NewFlagSet("scout", flag.ExitOnError)
	format := fs.String("format", "json", "output format: json, text, card")
	output := fs.String("output", "", "write .sdp/scout.json to this directory")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp scout [--format json|text|card] [--output DIR] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	// Validate format flag
	switch *format {
	case "json", "text", "card":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json, text, or card)\n", *format)
		os.Exit(2)
	}

	card, err := scout.Run(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: scout failed: %v\n", err)
		os.Exit(1)
	}

	// Write artifact if requested
	if *output != "" {
		path, werr := scout.WriteArtifact(*output, card)
		if werr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write artifact: %v\n", werr)
		} else {
			fmt.Fprintf(os.Stderr, "artifact: %s\n", path)
		}
	}

	// Format output
	var out string
	switch *format {
	case "json":
		var jerr error
		out, jerr = scout.FormatJSON(card)
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
	case "text":
		out = scout.FormatText(card)
	case "card":
		out = scout.FormatCard(card)
	}

	fmt.Print(out)
}
