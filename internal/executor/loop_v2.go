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
			logger.Info("dispatching beads card", "cycle", cycles, "card_id", cardID)
			result, execErr := bridge.DispatchAndRun(ctx, "", cardID)
			if execErr != nil {
				logger.Error("serve bridge execution failed", "card_id", cardID, "error", execErr)
			} else {
				logger.Info("serve bridge completed", "card_id", cardID, "status", result.Status)
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
