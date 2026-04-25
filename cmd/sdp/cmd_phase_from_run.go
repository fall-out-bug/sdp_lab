package main

import (
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/promote"
)

func runPhaseFromRun(args []string) {
	fs := flag.NewFlagSet("phase from-run", flag.ExitOnError)
	featureID := fs.String("feature-id", "", "Feature ID for delta artifacts (e.g., F135-05)")
	evidenceDir := fs.String("evidence-dir", "", "Evidence directory (default: .sdp/evidence/<run_id>)")
	phaseDir := fs.String("phase-dir", "", "Output directory for Phase FSM artifacts (default: .sdp/phases/<run_id>)")
	format := fs.String("format", "text", "output format: json|text")

	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp phase from-run <run_id> [--feature-id ID] [--evidence-dir DIR] [--phase-dir DIR]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Promotes a completed vibecode run to strict Phase FSM.")
		fmt.Fprintln(os.Stderr, "Reads .sdp/evidence/<run_id>/evidence.json and generates")
		fmt.Fprintln(os.Stderr, "plan/review/eval delta artifacts + gates with pre-populated evidence.")
		os.Exit(2)
	}

	runID := fs.Arg(0)

	validateFormat(*format)

	opts := promote.PromoteOptions{
		RunID:       runID,
		FeatureID:   *featureID,
		EvidenceDir: *evidenceDir,
		PhaseDir:    *phaseDir,
	}

	result, err := promote.PromoteFromRun(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *format == "json" {
		printJSON(result)
		return
	}

	fmt.Printf("Phase: from-run (vibecode promotion)\n")
	fmt.Printf("  Run ID:     %s\n", result.RunID)
	fmt.Printf("  Feature:    %s\n", result.FeatureID)
	fmt.Printf("  Evidence:   %s\n", result.EvidenceDir)
	fmt.Printf("  Phase dir:  %s\n", result.PhaseDir)
	fmt.Println()

	fmt.Printf("Deltas: %d\n", len(result.Deltas))
	for _, d := range result.Deltas {
		fmt.Printf("  %s: %s\n", d.Phase, d.Path)
	}

	fmt.Printf("Gates: %d\n", len(result.Gates))
	for _, g := range result.Gates {
		fmt.Printf("  %s: %s (%s)\n", g.Phase, g.ID, g.Path)
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  WARNING: %s\n", e)
		}
	}

	fmt.Println()
	fmt.Println("Gates are AWAITING human approval.")
	fmt.Println("To approve: edit each <phase>.gate.json (set answer, answerer, resolved_at)")
}
