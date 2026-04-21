package orchestrate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Exit codes for orchestrator (CI contract).
const (
	ExitSuccess    = 0
	ExitFailure    = 1
	ExitNeedsHuman = 2
	ExitCorrupted  = 3
)

func failf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// ProgressInfo tracks per-workstream progress for display.
type ProgressInfo struct {
	Done  int
	Total int
	WSID  string
	Phase string
}

// FormatProgress returns a progress line like "[3/7] building 00-042-03".
func FormatProgress(p ProgressInfo) string {
	return fmt.Sprintf("[%d/%d] %s %s", p.Done+1, p.Total, p.Phase, p.WSID)
}

func countDone(cp *Checkpoint) int {
	done := 0
	for _, ws := range cp.Workstreams {
		if ws.Status == "done" {
			done++
		}
	}
	return done
}

func printProgress(cp *Checkpoint, phase, wsID string) {
	total := len(cp.Workstreams)
	done := countDone(cp)
	info := ProgressInfo{Done: done, Total: total, WSID: wsID, Phase: phase}
	fmt.Fprintf(os.Stderr, "%s\n", FormatProgress(info))
}

func printPhaseProgress(phase, featureID string) {
	fmt.Fprintf(os.Stderr, "[phase] %s %s\n", phase, featureID)
}

// RunOpenCodeLoop drives the full workflow using opencode as the inner loop.
func RunOpenCodeLoop(projectRoot, featureID, cpPath, runsPath string, cp *Checkpoint, workstreams []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				slog.Error("failed to save checkpoint on shutdown", "error", err)
			}
			slog.Warn("shutdown", "error", ctx.Err())
			fmt.Fprintf(os.Stderr, "\nInterrupted. Resume with: sdp-orchestrate --feature %s --resume --runtime opencode\n", featureID)
			return ctx.Err()
		default:
		}

		action, err := ComputeNextAction(cp, workstreams, projectRoot)
		if err != nil {
			return failf("error: %v", err)
		}
		switch action.Action {
		case "build":
			startTime := time.Now()
			printProgress(cp, "building", action.WSID)
			cpFilePath := filepath.Join(cpPath, featureID+".json")
			hookEnv := HookEnv{WSID: action.WSID, FeatureID: featureID, Phase: "build", CheckpointPath: cpFilePath}
			if err := RunHooks(ctx, projectRoot, "build", "pre", hookEnv, func(msg string) { slog.Info("hook", "msg", msg) }); err != nil {
				return failf("error: pre-build hook: %v", err)
			}
			if _, err := Hydrate(projectRoot, featureID, action.WSID, cp); err != nil {
				slog.Error("hydration failed", "error", err, "ws", action.WSID)
				return fmt.Errorf("hydrate for build: %w", err)
			}
			phaseCtx, cancel := context.WithTimeout(ctx, buildPhaseTimeout)
			commit, err := RunBuildPhase(phaseCtx, projectRoot, action.Feature, action.WSID, nil)
			cancel()
			if err != nil {
				slog.Error("opencode build failed", "error", err, "ws", action.WSID)
				return fmt.Errorf("build phase: %w", err)
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
			elapsed := time.Since(startTime).Truncate(time.Second)
			fmt.Fprintf(os.Stderr, "[%d/%d] done %s (%s)\n", countDone(cp), len(cp.Workstreams), action.WSID, elapsed)
		case "review":
			printPhaseProgress("review", action.Feature)
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
				return fmt.Errorf("hydrate for review: %w", err)
			}
			phaseCtx, cancel := context.WithTimeout(ctx, reviewPhaseTimeout)
			reviewStartedAt := time.Now()
			reviewResult, reviewErr := RunReviewPhaseDetailed(phaseCtx, projectRoot, action.Feature, nil)
			cancel()
			if reviewErr != nil || reviewResult == nil || !reviewResult.Approved {
				var reviewOutput string
				var resultVerdict string
				if reviewResult != nil {
					reviewOutput = reviewResult.Output
					resultVerdict = reviewResult.Verdict
				}
				findingID, findingErr := EmitReviewFailureFinding(ctx, projectRoot, cp, reviewOutput, reviewErr)
				if findingErr != nil {
					slog.Warn("review finding emission failed", "error", findingErr, "feature", action.Feature)
				}
				adopted := false
				if resultVerdict == "PARTIALLY_APPROVED" || resultVerdict == "ESCALATED" {
					var adoptErr error
					adopted, adoptErr = adoptExistingReviewVerdictSince(projectRoot, cp, resultVerdict, reviewStartedAt)
					if adoptErr != nil {
						slog.Warn("agent-written review verdict not adopted", "error", adoptErr, "feature", action.Feature, "verdict", resultVerdict)
					}
				}
				if !adopted {
					var verdict ReviewVerdict
					switch resultVerdict {
					case "ESCALATED":
						if strings.TrimSpace(findingID) == "" {
							verdict = buildChangesRequestedReviewVerdict(cp, firstNonEmpty(strings.TrimSpace(reviewOutput), "review escalated without escalation issue"), findingID)
						} else {
							verdict = buildEscalatedReviewVerdict(cp, firstNonEmpty(strings.TrimSpace(reviewOutput), "review escalated"), findingID)
						}
					case "PARTIALLY_APPROVED":
						verdict = buildChangesRequestedReviewVerdict(cp, firstNonEmpty(strings.TrimSpace(reviewOutput), "review partially approved without structured verdict"), findingID)
					default:
						verdict = buildChangesRequestedReviewVerdict(cp, firstNonEmpty(strings.TrimSpace(reviewOutput), "review not approved"), findingID)
					}
					if _, verdictErr := WriteReviewVerdict(projectRoot, cp, verdict); verdictErr != nil {
						slog.Warn("review verdict write failed", "error", verdictErr, "feature", action.Feature)
					}
				}
				if saveErr := SaveCheckpoint(cpPath, cp); saveErr != nil {
					slog.Error("failed to save checkpoint after review failure", "error", saveErr)
				}
				slog.Error("opencode review failed", "error", reviewErr, "verdict", resultVerdict, "feature", action.Feature)
				if reviewErr != nil {
					return fmt.Errorf("review phase: %w", reviewErr)
				}
				return fmt.Errorf("opencode review %s", resultVerdict)
			}
			if err := RunHooks(ctx, projectRoot, "review", "post", hookEnv, func(msg string) { slog.Info("hook", "msg", msg) }); err != nil {
				return failf("error: post-review hook: %v", err)
			}
			adopted, adoptErr := adoptExistingReviewVerdictSince(projectRoot, cp, "APPROVED", reviewStartedAt)
			if adoptErr != nil {
				slog.Warn("agent-written approved verdict not adopted", "error", adoptErr, "feature", action.Feature)
			}
			if !adopted {
				if _, verdictErr := WriteReviewVerdict(projectRoot, cp, buildApprovedReviewVerdict(cp, strings.TrimSpace(reviewResult.Output))); verdictErr != nil {
					return failf("error: write review verdict: %v", verdictErr)
				}
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
			printPhaseProgress("pr", action.Feature)
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
			printPhaseProgress("qa", action.Feature)
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
			if _, err := HydrateForReview(projectRoot, action.Feature, cp, workstreams); err != nil {
				slog.Error("hydration failed", "error", err, "feature", action.Feature)
				return fmt.Errorf("hydrate for QA: %w", err)
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
				if saveErr := SaveCheckpoint(cpPath, cp); saveErr != nil {
					slog.Error("failed to save checkpoint after qa failure", "error", saveErr)
				}
				slog.Error("opencode qa failed", "error", err, "passed", passed, "feature", action.Feature)
				if err != nil {
					return fmt.Errorf("QA phase: %w", err)
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
