package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/evidence"
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

	if err := resetCheckpoint(*feature, *dryRun, *yes, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// resetCheckpoint contains the core logic, extracted for testability.
// Returns error instead of calling os.Exit.
func resetCheckpoint(feature string, dryRun, autoConfirm bool, stdin io.Reader, stdout, stderr io.Writer) error {
	featureID := strings.ToUpper(feature)
	if !strings.HasPrefix(featureID, "F") {
		featureID = "F" + featureID
	}

	if err := sdputil.ValidateFeatureID(featureID); err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	projectRoot, err := orchestrate.FindProjectRoot(wd)
	if err != nil {
		return fmt.Errorf("project root: %w", err)
	}

	cpDir := filepath.Join(projectRoot, ".sdp", "checkpoints")
	// Validate cpDir stays within project root
	if err := evidence.ValidatePath(cpDir, projectRoot); err != nil {
		return fmt.Errorf("checkpoint dir: %w", err)
	}

	cpPath := filepath.Join(cpDir, featureID+".json")
	// Validate final path stays within checkpoint dir
	if err := evidence.ValidatePath(cpPath, cpDir); err != nil {
		return fmt.Errorf("checkpoint path: %w", err)
	}

	info, err := os.Stat(cpPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no checkpoint found for %s", featureID)
		}
		return fmt.Errorf("stat: %w", err)
	}

	cp, loadErr := orchestrate.LoadCheckpoint(cpDir, featureID)
	if loadErr != nil {
		fmt.Fprintf(stderr, "warning: checkpoint corrupted: %v\n", loadErr)
	}

	fmt.Fprintf(stdout, "Reset checkpoint for %s\n", featureID)
	fmt.Fprintf(stdout, "  File: %s (%d bytes)\n", cpPath, info.Size())
	if cp != nil {
		fmt.Fprintf(stdout, "  Current phase: %s\n", cp.Phase)
		fmt.Fprintf(stdout, "  Branch: %s\n", cp.Branch)
		done := 0
		for _, ws := range cp.Workstreams {
			if ws.Status == "done" {
				done++
			}
		}
		fmt.Fprintf(stdout, "  Workstream progress: %d/%d done\n", done, len(cp.Workstreams))
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Workstream files in docs/workstreams/backlog/ will be PRESERVED.")
	fmt.Fprintln(stdout, "Only the checkpoint file (.sdp/checkpoints/) will be removed.")

	if dryRun {
		fmt.Fprintln(stdout, "\n[dry-run] No changes made.")
		return nil
	}

	if !autoConfirm {
		fmt.Fprintf(stdout, "\nProceed with reset? [y/N] ")
		scanner := bufio.NewScanner(stdin)
		if !scanner.Scan() {
			return errors.New("aborted")
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			return errors.New("aborted")
		}
	}

	if err := os.Remove(cpPath); err != nil {
		return fmt.Errorf("failed to remove checkpoint: %w", err)
	}

	fmt.Fprintf(stdout, "Checkpoint for %s reset successfully.\n", featureID)
	fmt.Fprintf(stdout, "Next step: sdp-orchestrate --feature %s to create fresh checkpoint.\n", featureID)
	return nil
}
