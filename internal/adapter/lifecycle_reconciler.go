package adapter

import (
	"fmt"
	"strings"
)

// CRDPhase represents kubeopencode Task status.phase.
// kubeopencode uses "Completed"; adapter uses PhaseSucceeded for FSM mapping.
// PhaseCompleted is an alias so callers can pass kubeopencode's TaskPhaseCompleted directly.
type CRDPhase string

const (
	PhasePending   CRDPhase = "Pending"
	PhaseRunning   CRDPhase = "Running"
	PhaseSucceeded CRDPhase = "Succeeded"
	PhaseCompleted CRDPhase = "Completed" // kubeopencode uses Completed; maps to PhaseSucceeded semantics
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

// TerminalReasonCode mirrors kubeopencode Task status.terminalReason.code.
// Used for deterministic FSM mapping when Task fails.
const (
	TerminalReasonInfrastructureError = "InfrastructureError"
	TerminalReasonAgentExitNonZero   = "AgentExitNonZero"
	TerminalReasonTimeout            = "Timeout"
	TerminalReasonUserStopped        = "UserStopped"
	TerminalReasonRetryExhausted     = "RetryExhausted"
	TerminalReasonUnknown            = "Unknown"
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
	case PhaseSucceeded, PhaseCompleted:
		if currentFSM == FSMInProgress {
			return FSMReview, "verification_candidate", nil
		}
		if currentFSM == FSMReview {
			return FSMVerified, "policy_pass", nil
		}
		return currentFSM, "", nil
	case PhaseFailed:
		// Prefer TerminalReasonCode when available (from Task status.terminalReason.code)
		code := strings.TrimSpace(failureReason)
		switch code {
		case TerminalReasonRetryExhausted, TerminalReasonInfrastructureError:
			return FSMBlocked, "retry_budget", nil
		case TerminalReasonAgentExitNonZero, TerminalReasonUserStopped, TerminalReasonTimeout, TerminalReasonUnknown:
			return FSMEscalated, "terminal_failure", nil
		}
		// Fallback: legacy string matching
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
