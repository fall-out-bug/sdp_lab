package executor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"sdp_dev/internal/control"
)

// RunOrchestrateLoopV2 runs the Beads-first orchestration loop.
// In beads/dual mode, queries ready queue from Beads and dispatches via ServeBridge.
func RunOrchestrateLoopV2(ctx context.Context, store *control.Store, projectRoot string, interval time.Duration, maxCycles int) error {
	if store == nil {
		return fmt.Errorf("nil control store")
	}

	logger := slog.Default().With("component", "executor.loop-v2", "project_root", projectRoot)
	bridge := NewServeBridge(store, projectRoot)

	cycles := 0

	for {
		if err := ctx.Err(); err != nil {
			logger.Info("loop stopped", "reason", "context_cancelled", "cycles", cycles)
			return err
		}
		if maxCycles > 0 && cycles >= maxCycles {
			logger.Info("loop stopped", "reason", "max_cycles", "cycles", cycles)
			return nil
		}

		cycles++

		// Try Beads-first dispatch
		cardID, err := bridge.DispatchBeads(ctx)
		if err != nil {
			logger.Warn("beads dispatch query failed, falling back to v1", "error", err)
			// Fallback: run v1 orchestrate
			result, v1Err := store.OrchestrateOnce()
			if v1Err != nil {
				logger.Error("v1 orchestrate failed", "cycle", cycles, "error", v1Err)
			} else if result != nil {
				logger.Info("v1 dispatch", "cycle", cycles, "action", result.Action)
			}
		} else if cardID != "" {
			card, loadErr := bridge.Store.LoadCard("", cardID)
			if loadErr != nil {
				logger.Error("load card before clarification failed", "card_id", cardID, "error", loadErr)
				continue
			}

			// Route discovery issues to the discovery pipeline instead of normal dispatch.
			if isDiscoveryCard(card) {
				logger.Info("routing discovery issue to pipeline", "card_id", cardID, "idea", card.Title)
				if discErr := RunDiscoveryFromCard(ctx, bridge.Store, card, projectRoot); discErr != nil {
					logger.Error("discovery pipeline failed", "card_id", cardID, "error", discErr)
				}
				continue
			}

			clarifyResult, clarifyErr := bridge.Clarify(ctx, card)
			if clarifyErr != nil {
				logger.Error("clarification failed", "card_id", cardID, "error", clarifyErr)
				continue
			}
			switch clarifyResult.Status {
			case "needs_clarification":
				logger.Info("card needs human clarification", "card_id", cardID, "questions", clarifyResult.Questions)
				if err := bridge.RecordClarification(cardID, clarifyResult); err != nil {
					logger.Error("failed to record clarification", "card_id", cardID, "error", err)
				} else if summary, sumErr := bridge.Summarize(ctx, cardID); sumErr != nil {
					logger.Warn("failed to summarize clarification", "card_id", cardID, "error", sumErr)
				} else {
					logger.Info("clarification summary", "card_id", cardID, "summary", summary.Text)
				}
				continue
			case "error":
				logger.Error("clarifier error", "card_id", cardID, "questions", clarifyResult.Questions)
				continue
			case "ready":
				if clarifyResult.Card != nil && !isAlreadyClarified(card) {
					if err := bridge.Store.SaveCard(clarifyResult.Card); err != nil {
						logger.Error("failed to persist clarified card", "card_id", cardID, "error", err)
						continue
					}
				}
			}

			// Plan step - generate implementation plan before dispatch
			card, loadErr = bridge.Store.LoadCard("", cardID)
			if loadErr != nil {
				logger.Error("load card before planning failed", "card_id", cardID, "error", loadErr)
				continue
			}
			planResult, planErr := bridge.GeneratePlan(ctx, card)
			if planErr != nil {
				logger.Error("plan generation failed", "card_id", cardID, "error", planErr)
				continue
			}
			switch planResult.Status {
			case "pending_approval":
				logger.Info("plan needs human approval", "card_id", cardID)
				if err := bridge.RecordPlan(cardID, planResult); err != nil {
					logger.Error("failed to record plan", "card_id", cardID, "error", err)
				}
				continue
			case "error":
				logger.Error("planner error", "card_id", cardID)
				continue
			case "approved":
				logger.Info("plan already approved", "card_id", cardID)
			case "generated":
				if err := bridge.RecordPlan(cardID, planResult); err != nil {
					logger.Error("failed to record generated plan", "card_id", cardID, "error", err)
				}
			}

			logger.Info("dispatching beads card", "cycle", cycles, "card_id", cardID)
			result, execErr := bridge.DispatchAndRun(ctx, "", cardID)
			if execErr != nil {
				logger.Error("serve bridge execution failed", "card_id", cardID, "error", execErr)
			} else {
				logger.Info("serve bridge completed", "card_id", cardID, "status", result.Status)

				if result.Status == control.ResultStatusSuccess {
					evalResult, evalErr := bridge.Evaluate(ctx, cardID)
					if evalErr != nil {
						logger.Error("evaluation error — pipeline blocked", "card_id", cardID, "error", evalErr)
						continue
					}

					summary, sumErr := bridge.Summarize(ctx, cardID)
					if sumErr != nil {
						logger.Warn("failed to summarize evaluation", "card_id", cardID, "error", sumErr)
					} else {
						logger.Info("evaluation summary", "card_id", cardID, "summary", summary.Text)
					}

					if evalResult.Verdict == evalVerdictFail || evalResult.Verdict == evalVerdictBlocked {
						logger.Info("evaluation blocked/failed", "card_id", cardID, "verdict", evalResult.Verdict, "score", evalResult.Score)
						if err := bridge.RecordEvalFindings(cardID, evalResult); err != nil {
							logger.Warn("failed to record evaluation findings", "card_id", cardID, "error", err)
						}
						continue
					}

					if evalResult.Verdict == evalVerdictNeedsReview {
						logger.Info("evaluation needs review — awaiting human", "card_id", cardID, "score", evalResult.Score)
						if err := bridge.RecordEvalFindings(cardID, evalResult); err != nil {
							logger.Warn("failed to record evaluation findings", "card_id", cardID, "error", err)
						}
						continue
					}

					if evalResult.Verdict == evalVerdictPass {
						if deployErr := bridge.TryDeployPhase(ctx, cardID, projectRoot); deployErr != nil {
							logger.Warn("deploy phase skipped", "card_id", cardID, "error", deployErr)
						}
					}
				}
			}
		} else {
			logger.Debug("no ready items", "cycle", cycles)
		}

		// Sleep
		if maxCycles > 0 && cycles >= maxCycles {
			logger.Info("loop stopped", "reason", "max_cycles", "cycles", cycles)
			return nil
		}
		if interval <= 0 {
			continue
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}
