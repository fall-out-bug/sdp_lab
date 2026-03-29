package gate

import (
	"fmt"
	"time"
)

// WaitForGate polls a gate until it is resolved or the timeout elapses.
// Returns the resolved gate or an error on timeout.
// If the gate is already resolved, it returns immediately.
func WaitForGate(mgr *BeadsGateManager, gateID string, pollInterval, timeout time.Duration) (*Gate, error) {
	deadline := time.Now().Add(timeout)

	for {
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

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("wait timeout for gate %s after %v", gateID, timeout)
		}

		time.Sleep(pollInterval)
	}
}
