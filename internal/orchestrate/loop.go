package orchestrate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// RunOpenCodeLoop drives the full workflow using opencode as the inner loop.
func RunOpenCodeLoop(projectRoot, featureID, cpPath, runsPath string, cp *Checkpoint, workstreams []string) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			_ = SaveCheckpoint(cpPath, cp) // best-effort so resume does not re-run last phase
			slog.Warn("shutdown", "error", ctx.Err())
			os.Exit(1)
		default:
		}

		action, err := ComputeNextAction(cp, workstreams, projectRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		switch action.Action {
		case "build":
			phaseCtx, cancel := context.WithTimeout(ctx, buildPhaseTimeout)
			commit, err := RunBuildPhase(phaseCtx, projectRoot, action.WSID)
			cancel()
			if err != nil {
				slog.Error("opencode build failed", "error", err, "ws", action.WSID)
				os.Exit(1)
			}
			if err := Advance(cp, workstreams, commit); err != nil {
				fmt.Fprintf(os.Stderr, "error: advance: %v\n", err)
				os.Exit(1)
			}
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
				os.Exit(1)
			}
		case "review":
			phaseCtx, cancel := context.WithTimeout(ctx, reviewPhaseTimeout)
			approved, err := RunReviewPhase(phaseCtx, projectRoot, action.Feature)
			cancel()
			if err != nil || !approved {
				slog.Error("opencode review failed", "error", err, "approved", approved, "feature", action.Feature)
				os.Exit(1)
			}
			if err := Advance(cp, workstreams, ""); err != nil {
				fmt.Fprintf(os.Stderr, "error: advance: %v\n", err)
				os.Exit(1)
			}
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
				os.Exit(1)
			}
		case "pr":
			if err := RunPRPhase(ctx, projectRoot, featureID, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			prNum, prURL, err := GetPRInfo()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: get PR info: %v\n", err)
				os.Exit(1)
			}
			cp.PRNumber = &prNum
			cp.PRURL = prURL
			cp.Phase = PhaseCI
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
				os.Exit(1)
			}
		case "ci-loop":
			pr := 0
			if cp.PRNumber != nil {
				pr = *cp.PRNumber
			}
			if pr == 0 {
				prNum, _, getErr := GetPRInfo()
				if getErr != nil {
					fmt.Fprintf(os.Stderr, "error: get PR info: %v\n", getErr)
					os.Exit(1)
				}
				pr = prNum
			}
			if pr > 0 {
				if err := RunCILoop(ctx, pr, featureID, cpPath, runsPath); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
			}
			cp.Phase = PhaseDone
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
				os.Exit(1)
			}
		case "done":
			slog.Info("oneshot complete", "feature", featureID)
			fmt.Println("CI GREEN - @oneshot complete")
			return
		}
	}
}
