package executor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"sdp_dev/internal/control"
)

// RunOrchestrateLoop runs continuous orchestration cycles until context is cancelled.
// Each cycle: orchestrate once → dispatch next → if dispatched, bridge execute → repeat.
// Sleeps between cycles. Returns when ctx is cancelled or max cycles reached.
func RunOrchestrateLoop(ctx context.Context, store *control.Store, bridge *ExecutorBridge, projectRoot string, interval time.Duration, maxCycles int) error {
	if store == nil {
		return fmt.Errorf("nil control store")
	}
	if interval < 0 {
		return fmt.Errorf("interval must be >= 0")
	}

	logger := slog.Default().With("component", "executor.loop", "project_root", projectRoot)
	cycles := 0

	for {
		if err := ctx.Err(); err != nil {
			logger.Info("orchestrate loop stopped", "reason", "context_cancelled", "cycles", cycles, "error", err)
			return err
		}
		if maxCycles > 0 && cycles >= maxCycles {
			logger.Info("orchestrate loop stopped", "reason", "max_cycles_reached", "cycles", cycles)
			return nil
		}

		cycles++
		result, err := store.OrchestrateOnce()
		if err != nil {
			logger.Error("orchestrate cycle failed", "cycle", cycles, "error", err)
			return err
		}
		if result == nil {
			logger.Info("orchestrate cycle completed", "cycle", cycles, "action", "no_action", "message", "nil result from orchestrate once")
		} else {
			logger.Info("orchestrate cycle completed", "cycle", cycles, "action", result.Action, "message", result.Message)
		}

		if result != nil && result.Action == "dispatched" && bridge != nil && result.DispatchedCard != nil {
			bridge.ProjectRoot = projectRoot
			execResult, err := bridge.DispatchAndRun(ctx, result.DispatchedCard.ProjectID, result.DispatchedCard.ID)
			if err != nil {
				logger.Error("dispatch bridge execution failed", "cycle", cycles, "project_id", result.DispatchedCard.ProjectID, "card_id", result.DispatchedCard.ID, "error", err)
				return err
			}
			logger.Info("dispatch bridge execution completed", "cycle", cycles, "project_id", result.DispatchedCard.ProjectID, "card_id", result.DispatchedCard.ID, "status", execResult.Status)
		}

		if maxCycles > 0 && cycles >= maxCycles {
			logger.Info("orchestrate loop stopped", "reason", "max_cycles_reached", "cycles", cycles)
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
			logger.Info("orchestrate loop stopped", "reason", "context_cancelled", "cycles", cycles, "error", ctx.Err())
			return ctx.Err()
		case <-timer.C:
		}
	}
}
