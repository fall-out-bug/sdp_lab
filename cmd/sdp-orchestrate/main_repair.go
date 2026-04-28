package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
)

func runRepair(projectRoot, featureID, cpPath string) {
	fmt.Printf("This will overwrite the checkpoint for %s with the last committed version from git.\n", featureID)
	fmt.Print("Continue? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Fprintln(os.Stderr, "Aborted.")
		os.Exit(1)
	}

	cp, err := orchestrate.RepairCheckpoint(projectRoot, cpPath, featureID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Could not recover checkpoint. You may need to manually reconstruct it.")
		os.Exit(3)
	}
	fmt.Printf("Checkpoint repaired for %s (phase: %s)\n", cp.FeatureID, cp.Phase)
	fmt.Println("Recovered from git history and re-saved with fresh integrity hash.")
}
