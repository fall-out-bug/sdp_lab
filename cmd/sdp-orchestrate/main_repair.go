package main

import (
	"fmt"
	"os"

	"sdp_dev/internal/orchestrate"
)

func runRepair(projectRoot, featureID, cpPath string) {
	cp, err := orchestrate.RepairCheckpoint(projectRoot, cpPath, featureID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Could not recover checkpoint. You may need to manually reconstruct it.")
		os.Exit(3)
	}
	fmt.Printf("Checkpoint repaired for %s (phase: %s)\n", cp.FeatureID, cp.Phase)
	fmt.Println("Recovered from git history and re-saved with fresh integrity hash.")
}
