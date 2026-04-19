package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"sdp_dev/internal/delta"
	"sdp_dev/internal/gate"
)

func runPhase(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp phase <plan|review|eval> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Phase commands:")
		fmt.Fprintln(os.Stderr, "  sdp phase plan [--feature-id ID] [--ws-id ID] [--run-id ID] [--strict]")
		fmt.Fprintln(os.Stderr, "  sdp phase review [--feature-id ID] [--ws-id ID] [--run-id ID] [--strict]")
		fmt.Fprintln(os.Stderr, "  sdp phase eval [--feature-id ID] [--ws-id ID] [--run-id ID] [--strict]")
		os.Exit(2)
	}

	switch args[0] {
	case "plan":
		runPhaseCommand("plan", args[1:], gate.GateTypePlan, "Approve plan delta?", []string{"approve", "reject", "defer"})
	case "review":
		runPhaseCommand("review", args[1:], gate.GateTypeReview, "Approve review delta?", []string{"approve", "reject", "request-changes"})
	case "eval":
		runPhaseCommand("eval", args[1:], gate.GateTypeEval, "Approve eval results?", []string{"approve", "reject", "retry"})
	default:
		fmt.Fprintf(os.Stderr, "unknown phase: %s\n", args[0])
		os.Exit(2)
	}
}

// phaseFlags holds common flags for phase commands.
type phaseFlags struct {
	featureID string
	wsID      string
	runID     string
	strict    bool
}

// parsePhaseFlags parses common phase flags from args.
func parsePhaseFlags(fs *flag.FlagSet) *phaseFlags {
	f := &phaseFlags{}
	fs.StringVar(&f.featureID, "feature-id", "", "Feature ID (e.g., F134)")
	fs.StringVar(&f.wsID, "ws-id", "", "Workstream ID (optional)")
	fs.StringVar(&f.runID, "run-id", "", "Run ID (auto-generated if not provided)")
	fs.BoolVar(&f.strict, "strict", false, "Require evidence enforcement")
	return f
}

// validatePhaseFlags checks that required flags are set.
func validatePhaseFlags(f *phaseFlags, phaseName string) {
	if f.featureID == "" {
		fmt.Fprintf(os.Stderr, "error: --feature-id is required for phase %s\n", phaseName)
		os.Exit(2)
	}
}

// generateRunID creates a run ID if not provided.
func generateRunID(runID string) string {
	if runID != "" {
		return runID
	}
	return fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), time.Now().UnixNano()%1000)
}

func runPhaseCommand(phase string, args []string, gateType gate.GateType, question string, options []string) {
	fs := flag.NewFlagSet(fmt.Sprintf("phase %s", phase), flag.ExitOnError)
	f := parsePhaseFlags(fs)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
		os.Exit(2)
	}
	validatePhaseFlags(f, phase)

	runID := generateRunID(f.runID)

	d := delta.NewDelta(phase,
		delta.WithFeatureID(f.featureID),
		delta.WithWorkstreamID(f.wsID),
		delta.WithRunID(runID),
	)

	fmt.Printf("Phase: %s\n", phase)
	fmt.Printf("   Feature:  %s\n", f.featureID)
	if f.wsID != "" {
		fmt.Printf("   Workstream: %s\n", f.wsID)
	}
	fmt.Printf("   Run ID:   %s\n", runID)
	fmt.Printf("   Strict:   %v\n", f.strict)
	fmt.Printf("\n")

	g := &gate.Gate{
		ID:        fmt.Sprintf("%s-%s-%s", phase, f.featureID, runID),
		Question:  question,
		Context:   fmt.Sprintf("%s phase for feature %s", phase, f.featureID),
		Options:   options,
		Type:      gateType,
		CreatedAt: time.Now(),
	}

	if d.HasChanges() {
		fmt.Printf("Delta: %d blocks, %d files\n", d.TotalBlocks(), d.TotalFiles())
	}

	fmt.Printf("Gate: %s\n", g.ID)
	fmt.Printf("   Question: %s\n", g.Question)
	fmt.Printf("   Status:   %s\n", g.Status())

	if !f.strict {
		fmt.Printf("\nAuto-approving gate (non-strict mode)\n")
		now := time.Now()
		g.Answer = "approve"
		g.Answerer = "sdp-phase"
		g.ResolvedAt = &now
	} else {
		fmt.Printf("\nGate awaiting human decision (strict mode)\n")
		fmt.Printf("   Options: %v\n", g.Options)
		fmt.Printf("   To resolve: sdp approve %s\n", g.ID)
	}

	fmt.Printf("\nTrace record: run-%s.json\n", runID)

	if g.IsBlocking() {
		os.Exit(1)
	}
}
