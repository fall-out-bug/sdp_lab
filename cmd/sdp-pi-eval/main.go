package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/evals"
	"github.com/fall-out-bug/sdp_lab/internal/executil"
)

func main() {
	fs := flag.NewFlagSet("sdp-pi-eval", flag.ExitOnError)
	corpusDir := fs.String("corpus", "", "Path to PI corpus YAML directory")
	feature := fs.String("feature", "", "Feature ID (e.g. F164)")
	outDir := fs.String("out", "", "Output directory for artifacts (default: .sdp/runs/pi-eval/<run-id>)")
	timeout := fs.Duration("timeout", evals.DefaultLiveModelTimeout, "Timeout per model call")
	slotsFlag := fs.String("slots", "", "Comma-separated slot overrides (slot=provider/model)")
	dryRun := fs.Bool("dry-run", false, "Load corpus and print plan without executing model calls")
	_ = fs.Parse(os.Args[1:])

	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdp-pi-eval: %v\n", err)
		os.Exit(1)
	}

	if *corpusDir == "" {
		*corpusDir = filepath.Join(projectRoot, "internal", "evals", "corpus")
	}

	if !filepath.IsAbs(*corpusDir) {
		*corpusDir = filepath.Join(projectRoot, *corpusDir)
	}

	slots := buildSlots(*slotsFlag)

	cfg := evals.LiveEvalConfig{
		ProjectRoot:  projectRoot,
		Runner:       executil.GetDefaultRunner(),
		ModelTimeout: *timeout,
		Slots:        slots,
		CorpusDir:    *corpusDir,
		OutDir:       *outDir,
		Feature:      *feature,
	}

	if *dryRun {
		printDryRun(cfg, slots)
		return
	}

	lr, err := evals.NewLiveRunner(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdp-pi-eval: %v\n", err)
		os.Exit(2)
	}

	run, err := lr.Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdp-pi-eval: %v\n", err)
		os.Exit(1)
	}

	printSummary(run, *feature)

	// Exit 0 always — this is advisory, not blocking CI
}

func buildSlots(slotsFlag string) []evals.LiveProviderSlot {
	if slotsFlag == "" {
		return nil
	}

	var slots []evals.LiveProviderSlot
	for _, part := range strings.Split(slotsFlag, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		slot := kv[0]
		pm := strings.SplitN(kv[1], "/", 2)
		if len(pm) != 2 {
			continue
		}
		slots = append(slots, evals.LiveProviderSlot{
			Slot:     slot,
			Provider: pm[0],
			Model:    pm[1],
		})
	}
	return slots
}

func printDryRun(cfg evals.LiveEvalConfig, slots []evals.LiveProviderSlot) {
	if len(slots) == 0 {
		slots = evals.DefaultLiveProviderSlots()
	}

	timeout := cfg.ModelTimeout
	if timeout == 0 {
		timeout = evals.DefaultLiveModelTimeout
	}

	fmt.Printf("## PI Live Eval Dry Run\n")
	fmt.Printf("Project: %s\n", cfg.ProjectRoot)
	fmt.Printf("Corpus: %s\n", cfg.CorpusDir)
	fmt.Printf("Feature: %s\n", cfg.Feature)
	fmt.Printf("Timeout: %s\n", timeout)
	fmt.Printf("Providers:\n")
	for _, s := range slots {
		fmt.Printf("  - %s: %s/%s\n", s.Slot, s.Provider, s.Model)
	}
	fmt.Printf("\nAdvisory: true (does NOT block CI)\n")
}

func printSummary(run *evals.LiveEvalRun, feature string) {
	fmt.Printf("\n## PI Live Eval Summary\n")
	fmt.Printf("Run: %s\n", run.RunID)
	fmt.Printf("Feature: %s\n", feature)
	fmt.Printf("Advisory: %v (this does NOT block CI)\n", run.Advisory)
	fmt.Printf("Status: %s\n", run.Status)
	fmt.Printf("Providers: %d slots\n", len(run.Slots))
	for _, s := range run.Slots {
		fmt.Printf("  - %s: %s/%s\n", s.Slot, s.Provider, s.Model)
	}
	fmt.Printf("Results: %d total evals\n", run.Summary.TotalEvals)
	fmt.Printf("  PASS: %d\n", run.Summary.PassCount)
	fmt.Printf("  FAIL: %d\n", run.Summary.FailCount)
	fmt.Printf("  DEGRADED: %d\n", run.Summary.DegradedCount)
	fmt.Printf("  ERROR: %d\n", run.Summary.ErrorCount)
	fmt.Printf("  Provider Failures: %d\n", run.Summary.ProviderFailures)
	fmt.Printf("Artifacts: %s\n", run.ArtifactDir)

	fmt.Printf("\n### Per-Case Verdicts\n")
	for _, r := range run.Results {
		fmt.Printf("  %s [%s/%s]: %s", r.CaseID, r.Provider, r.Model, r.Verdict)
		if r.Reason != "" {
			fmt.Printf(" (%s)", r.Reason)
		}
		fmt.Println()
	}

	fmt.Printf("\nAdvisory: Live evals are advisory/manual/scheduled. They are NOT required blocking PR CI.\n")
	fmt.Printf("Manifest: %s\n", filepath.Join(run.ArtifactDir, "eval-run.json"))
}
