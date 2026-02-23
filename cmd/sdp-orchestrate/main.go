package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"sdp_dev/internal/orchestrate"
)

func main() {
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
	skipGuard := flag.Bool("skip-guard", false, "Skip scope guard check on advance (escape hatch)")
	flag.Parse()

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

	workstreams, err := orchestrate.DiscoverWorkstreams(projectRoot, featureID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cpPath := filepath.Join(projectRoot, *checkpointDir)
	runsPath := filepath.Join(projectRoot, *runsDir)

	cp, err := orchestrate.LoadCheckpoint(cpPath, featureID)
	if err != nil {
		if *resume || !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		branch, err := orchestrate.CurrentBranch()
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

	if *nextAction {
		action, err := orchestrate.ComputeNextAction(cp, workstreams, projectRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(action); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *hydrate {
		action, err := orchestrate.ComputeNextAction(cp, workstreams, projectRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if action.Action == "review" {
			if _, err := orchestrate.HydrateForReview(projectRoot, featureID, cp, workstreams); err != nil {
				fmt.Fprintf(os.Stderr, "error: hydrate: %v\n", err)
				os.Exit(1)
			}
		} else {
			wsID := *ws
			if wsID == "" && action.Action == "build" {
				wsID = action.WSID
			}
			if wsID == "" {
				fmt.Fprintf(os.Stderr, "error: cannot hydrate: action=%s, specify --ws\n", action.Action)
				os.Exit(1)
			}
			if _, err := orchestrate.Hydrate(projectRoot, featureID, wsID, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: hydrate: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Println("Wrote .sdp/context-packet.json")
		return
	}

	if *runtime == "opencode" {
		orchestrate.RunOpenCodeLoop(projectRoot, featureID, cpPath, runsPath, cp, workstreams)
		return
	}

	if *advance {
		advanceCtx, advanceStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer advanceStop()
		if cp.Phase == orchestrate.PhaseBuild && *result != "" && !*skipGuard {
			wsID := orchestrate.CurrentBuildWS(cp)
			if wsID != "" {
				if err := orchestrate.RunGuardCheck(projectRoot, wsID); err != nil {
					var scopeErr *orchestrate.ScopeViolationError
					if errors.As(err, &scopeErr) {
						fmt.Fprintf(os.Stderr, "SCOPE VIOLATION: %s\n", err)
						if createErr := orchestrate.CreateScopeEscalationBead(scopeErr.WSID, scopeErr.Violations); createErr != nil {
							fmt.Fprintf(os.Stderr, "warning: bd create failed: %v\n", createErr)
						}
					}
					fmt.Fprintf(os.Stderr, "error: advance blocked by scope guard: %v\n", err)
					os.Exit(1)
				}
			}
		}
		if err := orchestrate.Advance(cp, workstreams, *result); err != nil {
			fmt.Fprintf(os.Stderr, "error: advance: %v\n", err)
			os.Exit(1)
		}
		if err := orchestrate.SaveCheckpoint(cpPath, cp); err != nil {
			fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
			os.Exit(1)
		}
		if cp.Phase == orchestrate.PhasePR {
			if err := orchestrate.AdvancePRPhase(advanceCtx, projectRoot, featureID, cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}
		if cp.Phase == orchestrate.PhaseCI {
			if err := orchestrate.AdvanceCIPhase(advanceCtx, featureID, cpPath, runsPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}
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
		fmt.Printf("INVOKE: @review %s\n", action.Feature)
	case "pr":
		fmt.Println("INVOKE: git push && gh pr create")
	case "ci-loop":
		fmt.Printf("INVOKE: sdp-ci-loop --pr %d --feature %s\n", action.PR, action.Feature)
	case "done":
		fmt.Println("CI GREEN - @oneshot complete")
	}
}
