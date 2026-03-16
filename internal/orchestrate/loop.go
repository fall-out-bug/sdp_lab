package orchestrate

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func failf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// RunOpenCodeLoop drives the full workflow using opencode as the inner loop.
func RunOpenCodeLoop(projectRoot, featureID, cpPath, runsPath string, cp *Checkpoint, workstreams []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			_ = SaveCheckpoint(cpPath, cp) // best-effort so resume does not re-run last phase
			slog.Warn("shutdown", "error", ctx.Err())
			return ctx.Err()
		default:
		}

		action, err := ComputeNextAction(cp, workstreams, projectRoot)
		if err != nil {
			return failf("error: %v", err)
		}
		switch action.Action {
		case "build":
			cpFilePath := filepath.Join(cpPath, featureID+".json")
			hookEnv := HookEnv{WSID: action.WSID, FeatureID: featureID, Phase: "build", CheckpointPath: cpFilePath}
			if err := RunHooks(ctx, projectRoot, "build", "pre", hookEnv, func(msg string) { slog.Info("hook", "msg", msg) }); err != nil {
				return failf("error: pre-build hook: %v", err)
			}
			if _, err := Hydrate(projectRoot, featureID, action.WSID, cp); err != nil {
				slog.Error("hydration failed", "error", err, "ws", action.WSID)
				return err
			}
			phaseCtx, cancel := context.WithTimeout(ctx, buildPhaseTimeout)
			commit, err := RunBuildPhase(phaseCtx, projectRoot, action.Feature, action.WSID, nil)
			cancel()
			if err != nil {
				slog.Error("opencode build failed", "error", err, "ws", action.WSID)
				return err
			}
			pending := 0
			for _, ws := range cp.Workstreams {
				if ws.Status != "done" {
					pending++
				}
			}
			if pending == 1 {
				if err := RunHooks(ctx, projectRoot, "build", "post", hookEnv, func(msg string) { slog.Info("hook", "msg", msg) }); err != nil {
					return failf("error: post-build hook: %v", err)
				}
			}
			if report, err := EnforceContractGate(projectRoot, featureID); err != nil {
				if report != nil {
					slog.Error("contract gate blocked", "phase", report.Phase)
				}
				return failf("error: contract gate blocked: %v", err)
			}
			if err := Advance(cp, workstreams, commit); err != nil {
				return failf("error: advance: %v", err)
			}
			if strings.TrimSpace(cp.PRURL) == "" && strings.TrimSpace(commit) != "" {
				if err := EnsureDraftPR(ctx, projectRoot, featureID, cp); err != nil {
					return failf("error: ensure draft PR: %v", err)
				}
			}
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				return failf("error: save checkpoint: %v", err)
			}
		case "review":
			if blocked, findings, err := HasBlockingFindings(ctx, action.Feature); err == nil && blocked {
				targetWS, findingIDs, rerouteErr := RedirectToBuildForBlockingFindings(cp, PhaseReview, findings)
				if rerouteErr != nil {
					return failf("error: reroute review blockers: %v", rerouteErr)
				}
				if _, verdictErr := WriteReviewVerdict(projectRoot, cp, buildBlockedReviewVerdict(cp, "blocking findings require workstream resolution before review rerun", findingIDs)); verdictErr != nil {
					return failf("error: write blocked review verdict: %v", verdictErr)
				}
				if err := SaveCheckpoint(cpPath, cp); err != nil {
					return failf("error: save checkpoint: %v", err)
				}
				slog.Info("rerouted review to build due to blocking findings", "feature", action.Feature, "ws_id", targetWS, "count", len(findings))
				continue
			}
			cpFilePath := filepath.Join(cpPath, featureID+".json")
			hookEnv := HookEnv{FeatureID: action.Feature, Phase: "review", CheckpointPath: cpFilePath}
			if err := RunHooks(ctx, projectRoot, "review", "pre", hookEnv, func(msg string) { slog.Info("hook", "msg", msg) }); err != nil {
				return failf("error: pre-review hook: %v", err)
			}
			if _, err := HydrateForReview(projectRoot, action.Feature, cp, workstreams); err != nil {
				slog.Error("hydration failed", "error", err, "feature", action.Feature)
				return err
			}
			phaseCtx, cancel := context.WithTimeout(ctx, reviewPhaseTimeout)
			approved, reviewOutput, err := RunReviewPhase(phaseCtx, projectRoot, action.Feature, nil)
			cancel()
			if err != nil || !approved {
				findingID, findingErr := EmitReviewFailureFinding(ctx, projectRoot, cp, reviewOutput, err)
				if findingErr != nil {
					slog.Warn("review finding emission failed", "error", findingErr, "feature", action.Feature)
				}
				if _, verdictErr := WriteReviewVerdict(projectRoot, cp, buildChangesRequestedReviewVerdict(cp, firstNonEmpty(strings.TrimSpace(reviewOutput), "review not approved"), findingID)); verdictErr != nil {
					slog.Warn("review verdict write failed", "error", verdictErr, "feature", action.Feature)
				}
				_ = SaveCheckpoint(cpPath, cp)
				slog.Error("opencode review failed", "error", err, "approved", approved, "feature", action.Feature)
				if err != nil {
					return err
				}
				return fmt.Errorf("opencode review not approved")
			}
			if err := RunHooks(ctx, projectRoot, "review", "post", hookEnv, func(msg string) { slog.Info("hook", "msg", msg) }); err != nil {
				return failf("error: post-review hook: %v", err)
			}
			if _, verdictErr := WriteReviewVerdict(projectRoot, cp, buildApprovedReviewVerdict(cp, strings.TrimSpace(reviewOutput))); verdictErr != nil {
				return failf("error: write review verdict: %v", verdictErr)
			}
			if report, err := EnforceContractGate(projectRoot, featureID); err != nil {
				if report != nil {
					slog.Error("contract gate blocked", "phase", report.Phase)
				}
				return failf("error: contract gate blocked: %v", err)
			}
			if err := Advance(cp, workstreams, ""); err != nil {
				return failf("error: advance: %v", err)
			}
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				return failf("error: save checkpoint: %v", err)
			}
		case "pr":
			if report, err := EnforceContractGate(projectRoot, featureID); err != nil {
				if report != nil {
					slog.Error("contract gate blocked", "phase", report.Phase)
				}
				return failf("error: contract gate blocked: %v", err)
			}
			if err := AdvancePRPhase(ctx, projectRoot, featureID, cpPath, cp); err != nil {
				return failf("error: %v", err)
			}
		case "ci-loop":
			if report, err := EnforceContractGate(projectRoot, featureID); err != nil {
				if report != nil {
					slog.Error("contract gate blocked", "phase", report.Phase)
				}
				return failf("error: contract gate blocked: %v", err)
			}
			if err := AdvanceCIPhase(ctx, projectRoot, featureID, cpPath, runsPath, cp); err != nil {
				return failf("error: %v", err)
			}
		case "qa":
			if blocked, findings, err := HasBlockingFindings(ctx, action.Feature); err == nil && blocked {
				targetWS, findingIDs, rerouteErr := RedirectToBuildForBlockingFindings(cp, PhaseQA, findings)
				if rerouteErr != nil {
					return failf("error: reroute qa blockers: %v", rerouteErr)
				}
				if _, verdictErr := WriteQAVerdict(projectRoot, cp, buildBlockedQAVerdict(cp, "blocking findings require workstream resolution before QA rerun", cp.QA.EvidenceRef, findingIDs)); verdictErr != nil {
					return failf("error: write blocked qa verdict: %v", verdictErr)
				}
				if err := SaveCheckpoint(cpPath, cp); err != nil {
					return failf("error: save checkpoint: %v", err)
				}
				slog.Info("rerouted qa to build due to blocking findings", "feature", action.Feature, "ws_id", targetWS, "count", len(findings))
				continue
			}
			phaseCtx, cancel := context.WithTimeout(ctx, qaPhaseTimeout)
			passed, qaOutput, err := RunQAPhase(phaseCtx, projectRoot, action.Feature, nil)
			cancel()
			if err != nil || !passed {
				qaInput := buildQAFailureFindingInput(cp, qaOutput, err)
				findingID, findingErr := EmitQAFailureFinding(ctx, projectRoot, cp, qaInput)
				if findingErr != nil {
					slog.Warn("qa finding emission failed", "error", findingErr, "feature", action.Feature)
				}
				if _, verdictErr := WriteQAVerdict(projectRoot, cp, buildFailedQAVerdict(cp, firstNonEmpty(strings.TrimSpace(qaOutput), "qa not passed"), cp.QA.EvidenceRef, findingID)); verdictErr != nil {
					slog.Warn("qa verdict write failed", "error", verdictErr, "feature", action.Feature)
				}
				_ = SaveCheckpoint(cpPath, cp)
				slog.Error("opencode qa failed", "error", err, "passed", passed, "feature", action.Feature)
				if err != nil {
					return err
				}
				return fmt.Errorf("opencode qa not passed")
			}
			if cp.QA == nil {
				cp.QA = &QAStatus{Iteration: 0}
			}
			cp.QA.Iteration++
			cp.QA.Status = "passed"
			if _, verdictErr := WriteQAVerdict(projectRoot, cp, buildPassedQAVerdict(cp, strings.TrimSpace(qaOutput), cp.QA.EvidenceRef)); verdictErr != nil {
				return failf("error: write qa verdict: %v", verdictErr)
			}
			if err := Advance(cp, workstreams, ""); err != nil {
				return failf("error: advance: %v", err)
			}
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				return failf("error: save checkpoint: %v", err)
			}
		case "done":
			slog.Info("oneshot complete", "feature", featureID)
			fmt.Println("CI GREEN - @oneshot complete")
			return nil
		}
	}
}
