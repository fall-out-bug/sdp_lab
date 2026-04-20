package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/orchestrate"
	"sdp_dev/internal/sdputil"
)

func runReset(args []string) {
	fs := flag.NewFlagSet("reset", flag.ExitOnError)
	feature := fs.String("feature", "", "Feature ID to reset (e.g. F042)")
	dryRun := fs.Bool("dry-run", false, "Show what would be reset without doing it")
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.Parse(args)

	if *feature == "" {
		fmt.Fprintln(os.Stderr, "error: --feature is required")
		fmt.Fprintln(os.Stderr, "usage: sdp reset --feature F042 [--dry-run] [--yes]")
		os.Exit(1)
	}

	featureID := strings.ToUpper(*feature)
	if !strings.HasPrefix(featureID, "F") {
		featureID = "F" + featureID
	}

	// Validate feature ID format to prevent path traversal (e.g. --feature '001/../../etc')
	if err := sdputil.ValidateFeatureID(featureID); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	projectRoot, err := orchestrate.FindProjectRoot(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cpDir := filepath.Join(projectRoot, ".sdp", "checkpoints")
	cpPath := filepath.Join(cpDir, featureID+".json")

	// Check checkpoint exists
	info, err := os.Stat(cpPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "No checkpoint found for %s\n", featureID)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Load current checkpoint to show what will be reset
	cp, loadErr := orchestrate.LoadCheckpoint(cpDir, featureID)
	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "warning: checkpoint corrupted: %v\n", loadErr)
	}

	// Show plan
	fmt.Printf("Reset checkpoint for %s\n", featureID)
	fmt.Printf("  File: %s (%d bytes)\n", cpPath, info.Size())
	if cp != nil {
		fmt.Printf("  Current phase: %s\n", cp.Phase)
		fmt.Printf("  Branch: %s\n", cp.Branch)
		done := 0
		for _, ws := range cp.Workstreams {
			if ws.Status == "done" {
				done++
			}
		}
		fmt.Printf("  Workstream progress: %d/%d done\n", done, len(cp.Workstreams))
	}
	fmt.Println()
	fmt.Println("Workstream files in docs/workstreams/backlog/ will be PRESERVED.")
	fmt.Println("Only the checkpoint file (.sdp/checkpoints/) will be removed.")

	if *dryRun {
		fmt.Println("\n[dry-run] No changes made.")
		return
	}

	// Confirmation
	if !*yes {
		fmt.Printf("\nProceed with reset? [y/N] ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			fmt.Println("Aborted.")
			return
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return
		}
	}

	// Remove checkpoint file
	if err := os.Remove(cpPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to remove checkpoint: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Checkpoint for %s reset successfully.\n", featureID)
	fmt.Printf("Next step: sdp-orchestrate --feature %s to create fresh checkpoint.\n", featureID)
}
