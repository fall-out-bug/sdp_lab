package adapter

import (
	"testing"
)

func TestNewLifecycleReconciler(t *testing.T) {
	r := NewLifecycleReconciler()
	if r == nil {
		t.Fatal("NewLifecycleReconciler returned nil")
	}
}

func TestLifecycleReconciler_ReconcilePhase(t *testing.T) {
	r := NewLifecycleReconciler()

	// Pending + Open -> InProgress
	state, action, err := r.ReconcilePhase(FSMOpen, PhasePending, "")
	if err != nil || state != FSMInProgress || action != "claim" {
		t.Errorf("Pending+Open: got state=%s action=%s err=%v", state, action, err)
	}

	// Running -> InProgress
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseRunning, "")
	if state != FSMInProgress || action != "heartbeat" {
		t.Errorf("Running: got state=%s action=%s", state, action)
	}

	// Succeeded + InProgress -> Review
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseSucceeded, "")
	if state != FSMReview || action != "verification_candidate" {
		t.Errorf("Succeeded+InProgress: got state=%s action=%s", state, action)
	}

	// Succeeded + Review -> Verified
	state, action, _ = r.ReconcilePhase(FSMReview, PhaseSucceeded, "")
	if state != FSMVerified || action != "policy_pass" {
		t.Errorf("Succeeded+Review: got state=%s action=%s", state, action)
	}

	// PhaseCompleted (kubeopencode) maps same as PhaseSucceeded
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseCompleted, "")
	if state != FSMReview || action != "verification_candidate" {
		t.Errorf("Completed+InProgress: got state=%s action=%s", state, action)
	}

	// Failed + RetryExhausted (TerminalReasonCode) -> Blocked
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseFailed, TerminalReasonRetryExhausted)
	if state != FSMBlocked || action != "retry_budget" {
		t.Errorf("Failed+RetryExhausted: got state=%s action=%s", state, action)
	}

	// Failed + InfrastructureError (TerminalReasonCode) -> Blocked
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseFailed, TerminalReasonInfrastructureError)
	if state != FSMBlocked || action != "retry_budget" {
		t.Errorf("Failed+InfrastructureError: got state=%s action=%s", state, action)
	}

	// Failed + AgentExitNonZero (TerminalReasonCode) -> Escalated
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseFailed, TerminalReasonAgentExitNonZero)
	if state != FSMEscalated || action != "terminal_failure" {
		t.Errorf("Failed+AgentExitNonZero: got state=%s action=%s", state, action)
	}

	// Failed + UserStopped (TerminalReasonCode) -> Escalated
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseFailed, TerminalReasonUserStopped)
	if state != FSMEscalated || action != "terminal_failure" {
		t.Errorf("Failed+UserStopped: got state=%s action=%s", state, action)
	}

	// Failed + retry (legacy string) -> Blocked
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseFailed, "retry timeout")
	if state != FSMBlocked || action != "retry_budget" {
		t.Errorf("Failed+retry: got state=%s action=%s", state, action)
	}

	// Failed + other -> Escalated
	state, action, _ = r.ReconcilePhase(FSMInProgress, PhaseFailed, "unknown error")
	if state != FSMEscalated || action != "terminal_failure" {
		t.Errorf("Failed+other: got state=%s action=%s", state, action)
	}

	// Unknown phase
	_, _, err = r.ReconcilePhase(FSMOpen, CRDPhase("Unknown"), "")
	if err == nil {
		t.Error("expected error for unknown phase")
	}
}

func TestDenialReason(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"policy_denied", "policy_denied"},
		{"verification_failed", "verification_failed"},
		{"dependency_blocked", "dependency_blocked"},
		{"runtime_failed", "runtime_failed"},
		{"other", "unknown"},
	}
	for _, tt := range tests {
		got := DenialReason(tt.code)
		if got != tt.want {
			t.Errorf("DenialReason(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
