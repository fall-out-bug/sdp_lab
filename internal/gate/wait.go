package gate

import (
	"context"
	"fmt"
	"time"
)

// WaitForGate polls a gate until it is resolved or the timeout elapses.
// Returns the resolved gate or an error on timeout or context cancellation.
// If the gate is already resolved, it returns immediately.
func WaitForGate(ctx context.Context, mgr *BeadsGateManager, gateID string, pollInterval, timeout time.Duration) (*Gate, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Check immediately before waiting for the first tick.
	g, err := mgr.CheckGate(gateID)
	if err != nil {
		return nil, fmt.Errorf("check gate %s: %w", gateID, err)
	}
	switch g.Status() {
	case "resolved":
		return g, nil
	case "timed_out":
		return nil, fmt.Errorf("gate %s timed out", gateID)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for gate %s: %w", gateID, ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("wait timeout for gate %s after %v", gateID, timeout)
			}

			g, err := mgr.CheckGate(gateID)
			if err != nil {
				return nil, fmt.Errorf("check gate %s: %w", gateID, err)
			}

			switch g.Status() {
			case "resolved":
				return g, nil
			case "timed_out":
				return nil, fmt.Errorf("gate %s timed out", gateID)
			}
		}
	}
}
