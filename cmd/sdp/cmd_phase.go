package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/delta"
	"sdp_dev/internal/gate"
)

func runPhase(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp phase <plan|review|eval> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Phase commands:")
		fmt.Fprintln(os.Stderr, "  sdp phase plan [--feature-id ID] [--evidence-path PATH] [--strict]")
		fmt.Fprintln(os.Stderr, "  sdp phase review [--feature-id ID] [--evidence-path PATH] [--strict]")
		fmt.Fprintln(os.Stderr, "  sdp phase eval [--feature-id ID] [--evidence-path PATH] [--strict]")
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
	featureID    string
	wsID         string
	runID        string
	strict       bool
	evidencePath string
}

// parsePhaseFlags parses common phase flags from args.
func parsePhaseFlags(fs *flag.FlagSet) *phaseFlags {
	f := &phaseFlags{}
	fs.StringVar(&f.featureID, "feature-id", "", "Feature ID (e.g., F134)")
	fs.StringVar(&f.wsID, "ws-id", "", "Workstream ID (optional)")
	fs.StringVar(&f.runID, "run-id", "", "Run ID (auto-generated if not provided)")
	fs.BoolVar(&f.strict, "strict", false, "Require evidence enforcement")
	fs.StringVar(&f.evidencePath, "evidence-path", "", "Path to evidence JSON file (required in strict mode)")
	return f
}

// validatePhaseFlags checks that required flags are set.
// Returns an error instead of calling os.Exit so it is testable.
func validatePhaseFlags(f *phaseFlags, phaseName string) error {
	if f.featureID == "" {
		return fmt.Errorf("--feature-id is required for phase %s", phaseName)
	}
	if f.strict && f.evidencePath == "" {
		return fmt.Errorf("--evidence-path is required when --strict is set")
	}
	return nil
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
	if err := validatePhaseFlags(f, phase); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

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
	if f.evidencePath != "" {
		fmt.Printf("   Evidence: %s\n", f.evidencePath)
	}
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

	// Resolve gate with evidence enforcement
	if f.strict {
		// Strict mode: validate evidence but do NOT auto-resolve gate.
		// The gate is persisted to gate.json and stays blocking until a human
		// edits it (sets answer + answerer + resolved_at) or beads integration is complete.
		if err := gate.ValidateEvidenceSchema(gateType, f.evidencePath); err != nil {
			fmt.Fprintf(os.Stderr, "\nGate BLOCKED (strict mode): evidence validation failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Provide valid evidence via --evidence-path.\n")
			os.Exit(1)
		}
		fmt.Printf("\nGate: AWAITING (strict mode, evidence validated)\n")
		fmt.Printf("   Evidence accepted. Gate requires human approval to proceed.\n")
		g.EvidencePath = f.evidencePath
	} else {
		// Non-strict mode
		if f.evidencePath != "" {
			// Evidence provided: validate and resolve through proper path
			if err := g.ResolveWithEvidence("approve", "sdp-phase-auto", f.evidencePath); err != nil {
				fmt.Fprintf(os.Stderr, "\nWARNING: evidence validation failed: %v\n", err)
				fmt.Fprintf(os.Stderr, "Non-strict mode: auto-approving despite invalid evidence\n")
				now := time.Now()
				g.Answer = "approve"
				g.Answerer = "sdp-phase-auto"
				g.ResolvedAt = &now
			} else {
				fmt.Printf("\nAuto-approving gate (non-strict mode, evidence validated)\n")
			}
		} else {
			// No evidence in non-strict mode: auto-approve with warning
			fmt.Printf("\nWARNING: phase gate approved without evidence (non-strict mode)\n")
			now := time.Now()
			g.Answer = "approve"
			g.Answerer = "sdp-phase-auto"
			g.ResolvedAt = &now
		}
	}

	// Persist delta artifact to .sdp/phases/<run_id>/<phase>.delta.md
	phaseDir := filepath.Join(".sdp", "phases", runID)
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: failed to create phase directory: %v\n", err)
	} else {
		deltaPath := filepath.Join(phaseDir, phase+".delta.md")
		if err := os.WriteFile(deltaPath, []byte(d.RenderMarkdown()), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to write delta artifact: %v\n", err)
		} else {
			fmt.Printf("Delta artifact: %s\n", deltaPath)
		}

		// Persist gate object to gate.json
		gatePath := filepath.Join(phaseDir, "gate.json")
		gateData, err := json.MarshalIndent(g, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to marshal gate: %v\n", err)
		} else if err := os.WriteFile(gatePath, gateData, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to write gate file: %v\n", err)
		} else {
			fmt.Printf("Gate file: %s\n", gatePath)
			if g.IsBlocking() {
				fmt.Printf("   To approve: edit gate.json (set answer, answerer, resolved_at)\n")
			}
		}
	}

	// Write trace record to .sdp/phases/<run_id>/trace.json
	writeTraceRecord(phaseDir, &traceRecord{
		Phase:        phase,
		FeatureID:    f.featureID,
		WorkstreamID: f.wsID,
		RunID:        runID,
		Strict:       f.strict,
		EvidencePath: f.evidencePath,
		GateID:       g.ID,
		Answer:       g.Answer,
		Answerer:     g.Answerer,
		Resolved:     g.ResolvedAt != nil,
		Timestamp:    time.Now().UTC(),
	})

	fmt.Printf("\nTrace record: %s\n", filepath.Join(phaseDir, "trace.json"))

	if g.IsBlocking() {
		os.Exit(1)
	}
}

// traceRecord is a minimal audit trace for a phase gate resolution.
type traceRecord struct {
	Phase        string    `json:"phase"`
	FeatureID    string    `json:"feature_id"`
	WorkstreamID string    `json:"ws_id,omitempty"`
	RunID        string    `json:"run_id"`
	Strict       bool      `json:"strict"`
	EvidencePath string    `json:"evidence_path,omitempty"`
	GateID       string    `json:"gate_id"`
	Answer       string    `json:"answer,omitempty"`
	Answerer     string    `json:"answerer,omitempty"`
	Resolved     bool      `json:"resolved"`
	Timestamp    time.Time `json:"timestamp"`
}

// writeTraceRecord writes a JSON trace record to the phase directory.
func writeTraceRecord(dir string, rec *traceRecord) {
	path := filepath.Join(dir, "trace.json")
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: failed to marshal trace record: %v\n", err)
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: failed to write trace record: %v\n", err)
	}
}
