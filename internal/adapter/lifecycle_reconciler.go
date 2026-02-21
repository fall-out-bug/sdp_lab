package adapter

import (
	"fmt"
	"strings"
)

// CRDPhase represents kubeopencode Task status.phase.
type CRDPhase string

const (
	PhasePending   CRDPhase = "Pending"
	PhaseRunning   CRDPhase = "Running"
	PhaseSucceeded CRDPhase = "Succeeded"
	PhaseFailed    CRDPhase = "Failed"
)

// FSMState is the SDP protocol flow state.
type FSMState string

const (
	FSMOpen        FSMState = "open"
	FSMInProgress  FSMState = "in_progress"
	FSMReview      FSMState = "review"
	FSMVerified    FSMState = "verified"
	FSMDone        FSMState = "done"
	FSMBlocked     FSMState = "blocked"
	FSMEscalated   FSMState = "escalated"
	FSMCancelled   FSMState = "cancelled"
)

// LifecycleReconciler maps CRD status/events to SDP FSM transitions.
type LifecycleReconciler struct{}

// NewLifecycleReconciler returns a new reconciler.
func NewLifecycleReconciler() *LifecycleReconciler {
	return &LifecycleReconciler{}
}

// ReconcilePhase maps CRD phase to target FSM state.
// Implements the mapping from docs/KUBEOPENCODE_SDP_ADAPTER_ARCHITECTURE.md.
func (r *LifecycleReconciler) ReconcilePhase(currentFSM FSMState, crdPhase CRDPhase, failureReason string) (FSMState, string, error) {
	switch crdPhase {
	case PhasePending:
		if currentFSM == FSMOpen {
			return FSMInProgress, "claim", nil
		}
		return currentFSM, "", nil
	case PhaseRunning:
		return FSMInProgress, "heartbeat", nil
	case PhaseSucceeded:
		if currentFSM == FSMInProgress {
			return FSMReview, "verification_candidate", nil
		}
		if currentFSM == FSMReview {
			return FSMVerified, "policy_pass", nil
		}
		return currentFSM, "", nil
	case PhaseFailed:
		reason := strings.ToLower(failureReason)
		if strings.Contains(reason, "retry") || strings.Contains(reason, "transient") {
			return FSMBlocked, "retry_budget", nil
		}
		return FSMEscalated, "terminal_failure", nil
	default:
		return currentFSM, "", fmt.Errorf("unknown CRD phase: %s", crdPhase)
	}
}

// DenialReason maps adapter denial to taxonomy.
func DenialReason(code string) string {
	switch code {
	case "policy_denied":
		return "policy_denied"
	case "verification_failed":
		return "verification_failed"
	case "dependency_blocked":
		return "dependency_blocked"
	case "runtime_failed":
		return "runtime_failed"
	default:
		return "unknown"
	}
}
