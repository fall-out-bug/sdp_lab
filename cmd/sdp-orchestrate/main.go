package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"sdp_dev/internal/ciloop"
	"sdp_dev/internal/evidence"
	"sdp_dev/internal/orchestrate"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	feature := flag.String("feature", "", "Feature ID (e.g. F016)")
	nextAction := flag.Bool("next-action", false, "Output next action as JSON")
	advance := flag.Bool("advance", false, "Advance to next phase after current action")
	result := flag.String("result", "", "Result for advance (e.g. commit hash for build phase)")
	resume := flag.Bool("resume", false, "Resume from existing checkpoint")
	checkpointDir := flag.String("checkpoint-dir", ".sdp/checkpoints", "Checkpoint directory")
	runsDir := flag.String("runs-dir", ".sdp/runs", "Runs directory")
	runtime := flag.String("runtime", "", "Runtime for LLM phases: opencode (invokes opencode run as subprocess)")
	hydrate := flag.Bool("hydrate", false, "Gather context and write .sdp/context-packet.json (before LLM invocation)")
	ws := flag.String("ws", "", "Workstream ID for --hydrate (default: current build ws from next-action)")
	index := flag.Bool("index", false, "Generate INDEX table for feature workstreams (print to stdout)")
	status := flag.Bool("status", false, "Output status: pending WS, open beads, next action")
	jsonOutput := flag.Bool("json", false, "Output in JSON format (default is human-readable)")
	format := flag.String("format", "", "Output format: json or text (overrides --json if both set)")
	repair := flag.Bool("repair", false, "Repair corrupted checkpoint from git history")
	autonomous := flag.Bool("autonomous", false, "Run batch-mode pull-FSM loop (no external runtime)")
	acceptGates := flag.String("accept-gates", "", "Comma-separated list of human gates to auto-accept (e.g. review,pr,ci-loop,qa)")
	autonomousDryRun := flag.Bool("dry-run", false, "With --autonomous: print action sequence without execution")
	autonomousForce := flag.Bool("force", false, "With --autonomous: override MVP safety and allow non-dry-run without executor backend")
	maxIterations := flag.Int("max-iterations", orchestrate.DefaultMaxIterations, "Max iterations for --autonomous loop")
	flag.Parse()

	// --format takes precedence over --json
	if *format != "" {
		switch *format {
		case "json":
			*jsonOutput = true
		case "text":
			*jsonOutput = false
		}
	}

	if *feature == "" {
		fmt.Fprintln(os.Stderr, "error: --feature is required")
		flag.Usage()
		os.Exit(1)
	}

	featureID := strings.ToUpper(*feature)
	if !strings.HasPrefix(featureID, "F") {
		featureID = "F" + featureID
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

	if *index {
		runIndex(projectRoot, featureID, *checkpointDir)
		return
	}
	if *repair {
		runRepair(projectRoot, featureID, filepath.Join(projectRoot, *checkpointDir))
		return
	}
	workstreams, err := orchestrate.DiscoverWorkstreams(projectRoot, featureID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cpPath := filepath.Join(projectRoot, *checkpointDir)
	runsPath := filepath.Join(projectRoot, *runsDir)
	if err := evidence.ValidatePath(cpPath, projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "checkpoint-dir: %v\n", err)
		os.Exit(1)
	}
	if err := evidence.ValidatePath(runsPath, projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "runs-dir: %v\n", err)
		os.Exit(1)
	}

	// Remove orphan .tmp files from previous runs
	ciloop.RemoveOrphanTmpFiles(cpPath, runsPath, filepath.Join(projectRoot, ".sdp"))

	cp, err := orchestrate.LoadCheckpoint(cpPath, featureID)
	if err != nil {
		if errors.Is(err, orchestrate.ErrCheckpointCorrupted) {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Checkpoint corrupted. Run --repair to recover.\n")
			os.Exit(orchestrate.ExitCorrupted)
		}
		if *resume || !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(orchestrate.ExitFailure)
		}
		branch, err := orchestrate.CurrentBranch(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		cp = orchestrate.CreateInitialCheckpoint(featureID, branch, workstreams)
		cp.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := os.MkdirAll(cpPath, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := orchestrate.SaveCheckpoint(cpPath, cp); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := orchestrate.EnsureRunFile(runsPath, featureID, cp.Branch); err != nil {
			fmt.Fprintf(os.Stderr, "error: ensure run file: %v\n", err)
			os.Exit(1)
		}
	}

	if *status {
		runStatus(projectRoot, featureID, cp, workstreams, *jsonOutput)
		return
	}
	if *nextAction {
		runNextAction(cp, workstreams, projectRoot, *jsonOutput)
		return
	}
	if *hydrate {
		runHydrate(projectRoot, featureID, *ws, cp, workstreams)
		return
	}
	if *runtime == "opencode" {
		if err := orchestrate.RunOpenCodeLoop(projectRoot, featureID, cpPath, runsPath, cp, workstreams); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *autonomous {
		var gates []string
		if *acceptGates != "" {
			gates = strings.Split(*acceptGates, ",")
			for i := range gates {
				gates[i] = strings.TrimSpace(gates[i])
			}
		}
		config := orchestrate.AutonomousConfig{
			MaxIterations: *maxIterations,
			AcceptGates:   gates,
			DryRun:        *autonomousDryRun,
			Force:         *autonomousForce,
		}
		if err := orchestrate.RunAutonomous(ctx, config, projectRoot, featureID, cpPath, cp, workstreams); err != nil {
			os.Exit(orchestrate.ExitCode(err))
		}
		return
	}
	if *advance {
		runAdvance(projectRoot, featureID, cpPath, runsPath, *result, false, cp, workstreams)
		return
	}

	action, err := orchestrate.ComputeNextAction(cp, workstreams, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	switch action.Action {
	case "build":
		fmt.Printf("INVOKE: @build %s\n", action.WSID)
	case "review":
		cpFilePath := filepath.Join(cpPath, featureID+".json")
		hookEnv := orchestrate.HookEnv{FeatureID: action.Feature, Phase: "review", CheckpointPath: cpFilePath}
		if err := orchestrate.RunHooks(ctx, projectRoot, "review", "pre", hookEnv, func(msg string) { fmt.Fprintln(os.Stderr, msg) }); err != nil {
			fmt.Fprintf(os.Stderr, "error: pre-review hook: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("INVOKE: @review %s\n", action.Feature)
	case "pr":
		fmt.Println("INVOKE: git push && gh pr create")
	case "ci-loop":
		fmt.Printf("INVOKE: sdp-ci-loop --pr %d --feature %s\n", action.PR, action.Feature)
	case "done":
		fmt.Println("CI GREEN - @oneshot complete")
	}
}

func runIndex(projectRoot, featureID, checkpointDir string) {
	cpPath := filepath.Join(projectRoot, checkpointDir)
	cp, err := orchestrate.LoadCheckpoint(cpPath, featureID)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	rows, err := orchestrate.GenerateIndexTable(projectRoot, featureID, cp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(orchestrate.FormatIndexTable(rows))
}
