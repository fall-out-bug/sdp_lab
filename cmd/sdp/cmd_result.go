package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/control"
)

func runResult(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp result ingest")
		os.Exit(2)
	}
	switch args[0] {
	case "ingest":
		runResultIngest(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown result subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func runResultIngest(args []string) {
	fs := flag.NewFlagSet("result-ingest", flag.ExitOnError)
	path := fs.String("input", "", "input file path (required)")
	_ = fs.Parse(args)
	if *path == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		os.Exit(2)
	}

	data, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read result packet from %s: %v\n", *path, err)
		os.Exit(1)
	}

	var packet control.ExecutorResultPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		fmt.Fprintf(os.Stderr, "error: parse result packet: %v\n", err)
		os.Exit(1)
	}

	store := openStore()
	card, err := store.IngestExecutorResult(&packet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: ingest executor result: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Ingested executor result for card %s\n", packet.ParentFeatureID)
	printJSON(card)
}
