// self-improve-agent analyzes telemetry and creates Beads tasks for SDP improvements.
//
// Usage: self-improve-agent [--work-dir .] [--max-proposals 3]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/selfimprove"
)

func main() {
	workDir := flag.String("work-dir", "", "Working directory (default: cwd)")
	maxProposals := flag.Int("max-proposals", 3, "Max Beads tasks to create per cycle")
	flag.Parse()

	wd := *workDir
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	runs, err := selfimprove.IngestRuns(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingest runs: %v\n", err)
		os.Exit(1)
	}

	intakePath := filepath.Join(wd, ".sdp", "observability", "intake.jsonl")
	telemetry, _ := selfimprove.IngestIntakeJSONL(intakePath)

	detector := selfimprove.NewWeaknessDetector()
	patterns := detector.Detect(runs, telemetry)
	if len(patterns) == 0 {
		fmt.Println("No weakness patterns detected")
		return
	}

	gate := selfimprove.NewSafetyGate()
	filtered := gate.Filter(patterns)
	if len(filtered) == 0 {
		fmt.Println("All patterns blocked by safety gate")
		return
	}

	gen := selfimprove.NewProposalGenerator(wd)
	created, err := gen.Generate(filtered, *maxProposals)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}
	for _, id := range created {
		fmt.Printf("Created: %s\n", id)
	}
}
