package main

import (
	"encoding/json"
	"fmt"
	"os"

	"sdp_dev/internal/orchestrate"
)

func runNextAction(cp *orchestrate.Checkpoint, workstreams []string, projectRoot string, jsonOutput bool) {
	action, err := orchestrate.ComputeNextAction(cp, workstreams, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(action); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Println(orchestrate.FormatNextAction(action))
}
