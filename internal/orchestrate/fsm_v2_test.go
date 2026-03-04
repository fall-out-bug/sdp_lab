package orchestrate

import (
	"context"
	"errors"
	"testing"
	"time"

	"sdp_dev/internal/adapters/sdk"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StatePending, "pending"},
		{StateValidated, "validated"},
		{StateAssigned, "assigned"},
		{StateExecuted, "executed"},
		{StateReviewed, "reviewed"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("State.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStateIsTerminal(t *testing.T) {
	tests := []struct {
		state    State
		expected bool
	}{
		{StatePending, false},
		{StateValidated, false},
		{StateCompleted, true},
		{StateFailed, true},
		{StateRolledBack, true},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.expected {
				t.Errorf("State.IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewFSMV2(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "00-071-02",
		FeatureID:    "F071",
		BeadsID:      "sdplab-dwm",
	}

	fsm := NewFSMV2(ctx, fsmCtx)

	if fsm.CurrentState() != StatePending {
		t.Errorf("NewFSMV2() initial state = %v, want %v", fsm.CurrentState(), StatePending)
	}

	if fsm.Context().WorkstreamID != "00-071-02" {
		t.Errorf("NewFSMV2() context workstream = %v, want 00-071-02", fsm.Context().WorkstreamID)
	}
}

func TestFSMV2_Transition_ValidPath(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "00-071-02",
		FeatureID:    "F071",
		BeadsID:      "sdplab-dwm",
		Actor:        &sdk.Actor{Type: "agent", ID: "test-agent"},
		SessionID:    "session-123",
	}

	fsm := NewFSMV2(ctx, fsmCtx)

	err := fsm.Validate(ctx)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if fsm.CurrentState() != StateValidated {
		t.Errorf("after Validate(), state = %v, want %v", fsm.CurrentState(), StateValidated)
	}

	err = fsm.Assign(ctx)
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}
	if fsm.CurrentState() != StateAssigned {
		t.Errorf("after Assign(), state = %v, want %v", fsm.CurrentState(), StateAssigned)
	}

	err = fsm.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if fsm.CurrentState() != StateExecuted {
		t.Errorf("after Execute(), state = %v, want %v", fsm.CurrentState(), StateExecuted)
	}

	err = fsm.Review(ctx)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if fsm.CurrentState() != StateReviewed {
		t.Errorf("after Review(), state = %v, want %v", fsm.CurrentState(), StateReviewed)
	}

	err = fsm.Complete(ctx)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if fsm.CurrentState() != StateCompleted {
		t.Errorf("after Complete(), state = %v, want %v", fsm.CurrentState(), StateCompleted)
	}
}

func TestFSMV2_Transition_InvalidTransition(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "00-071-02",
		FeatureID:    "F071",
	}

	fsm := NewFSMV2(ctx, fsmCtx)

	err := fsm.Transition(ctx, StateCompleted)
	if err == nil {
		t.Fatal("Transition() expected error for invalid transition, got nil")
	}

	var te *TransitionError
	if !errors.As(err, &te) {
		t.Fatalf("Transition() error type = %T, want *TransitionError", err)
	}

	if te.Code != "INVALID_TRANSITION" {
		t.Errorf("TransitionError.Code = %v, want INVALID_TRANSITION", te.Code)
	}
}

func TestFSMV2_Transition_FromTerminalState(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "00-071-02",
		FeatureID:    "F071",
	}

	fsm := NewFSMV2(ctx, fsmCtx)

	err := fsm.Fail(ctx, "test failure")
	if err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if fsm.CurrentState() != StateFailed {
		t.Fatalf("after Fail(), state = %v, want %v", fsm.CurrentState(), StateFailed)
	}

	err = fsm.Transition(ctx, StateValidated)
	if err == nil {
		t.Fatal("Transition() from terminal state expected error, got nil")
	}

	var te *TransitionError
	if !errors.As(err, &te) {
		t.Fatalf("Transition() error type = %T, want *TransitionError", err)
	}

	if te.Code != "TERMINAL_STATE" {
		t.Errorf("TransitionError.Code = %v, want TERMINAL_STATE", te.Code)
	}
}

func TestFSMV2_Transition_WithPolicyCheck(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "00-071-02",
		FeatureID:    "F071",
	}

	mockDM := &mockDecisionMaker{
		decision: &sdk.RuntimeDecision{
			Decision: "allow",
			Reason:   sdk.DecisionReason{Code: "OK", Message: "allowed"},
		},
	}

	fsm := NewFSMV2(ctx, fsmCtx, WithDecisionMaker(mockDM))

	err := fsm.Validate(ctx)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !mockDM.called {
		t.Error("DecisionMaker.MakeDecision() was not called")
	}
}

func TestFSMV2_Transition_PolicyDenied(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "00-071-02",
		FeatureID:    "F071",
	}

	mockDM := &mockDecisionMaker{
		decision: &sdk.RuntimeDecision{
			Decision: "deny",
			Reason:   sdk.DecisionReason{Code: "FORBIDDEN", Message: "not allowed"},
		},
	}

	fsm := NewFSMV2(ctx, fsmCtx, WithDecisionMaker(mockDM))

	err := fsm.Validate(ctx)
	if err == nil {
		t.Fatal("Validate() expected error for policy denial, got nil")
	}

	var te *TransitionError
	if !errors.As(err, &te) {
		t.Fatalf("Validate() error type = %T, want *TransitionError", err)
	}

	if te.Code != "POLICY_DENIED" {
		t.Errorf("TransitionError.Code = %v, want POLICY_DENIED", te.Code)
	}
}

func TestFSMV2_Transition_WithHooks(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "00-071-02",
		FeatureID:    "F071",
	}

	beforeCalled := false
	afterCalled := false

	hook := &testHook{
		beforeFunc: func(ctx context.Context, from, to State, fsmCtx *FSMContext) error {
			beforeCalled = true
			return nil
		},
		afterFunc: func(ctx context.Context, from, to State, fsmCtx *FSMContext, err error) error {
			afterCalled = true
			return nil
		},
	}

	fsm := NewFSMV2(ctx, fsmCtx, WithHooks(hook))

	_ = fsm.Validate(ctx)

	if !beforeCalled {
		t.Error("BeforeTransition hook was not called")
	}
	if !afterCalled {
		t.Error("AfterTransition hook was not called")
	}
}

func TestFSMV2_Transition_ValidationFailure(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "",
		FeatureID:    "F071",
	}

	fsm := NewFSMV2(ctx, fsmCtx)

	err := fsm.Validate(ctx)
	if err == nil {
		t.Fatal("Validate() expected error for missing workstream_id, got nil")
	}

	var te *TransitionError
	if !errors.As(err, &te) {
		t.Fatalf("Validate() error type = %T, want *TransitionError", err)
	}

	if te.Code != "VALIDATION_FAILED" {
		t.Errorf("TransitionError.Code = %v, want VALIDATION_FAILED", te.Code)
	}
}

func TestFSMV2_GetCheckpoints(t *testing.T) {
	ctx := context.Background()
	fsmCtx := &FSMContext{
		WorkstreamID: "00-071-02",
		FeatureID:    "F071",
		Actor:        &sdk.Actor{Type: "agent", ID: "test-agent"},
		SessionID:    "session-123",
	}

	fsm := NewFSMV2(ctx, fsmCtx)

	_ = fsm.Validate(ctx)
	_ = fsm.Assign(ctx)
	_ = fsm.Execute(ctx)
	_ = fsm.Review(ctx)
	_ = fsm.Complete(ctx)

	checkpoints := fsm.GetCheckpoints()
	if len(checkpoints) == 0 {
		t.Error("GetCheckpoints() returned empty, expected checkpoints")
	}
}

func TestFSMV2_GetState_ReturnsDefensiveCopy(t *testing.T) {
	ctx := context.Background()
	fsm := NewFSMV2(ctx, &FSMContext{WorkstreamID: "00-071-02", FeatureID: "F071"})

	fsm.mu.Lock()
	fsm.state.LastError = &TransitionError{Code: "X", Message: "orig"}
	fsm.state.Checkpoints = []CheckpointRecord{{
		Name:    "cp1",
		Result:  "passed",
		Details: map[string]interface{}{"k": "v"},
	}}
	fsm.mu.Unlock()

	copyState := fsm.GetState()
	if copyState == nil {
		t.Fatal("expected state copy")
	}

	copyState.State = StateFailed
	copyState.LastError.Message = "changed"
	copyState.Checkpoints[0].Details["k"] = "mutated"

	fresh := fsm.GetState()
	if fresh.State != StatePending {
		t.Fatalf("internal state mutated: got %s want %s", fresh.State, StatePending)
	}
	if fresh.LastError == nil || fresh.LastError.Message != "orig" {
		t.Fatalf("last error mutated: got %+v", fresh.LastError)
	}
	if got := fresh.Checkpoints[0].Details["k"]; got != "v" {
		t.Fatalf("checkpoint details mutated: got %v want v", got)
	}
}

func TestFSMV2_GetCheckpoints_ReturnsDefensiveCopy(t *testing.T) {
	ctx := context.Background()
	fsm := NewFSMV2(ctx, &FSMContext{WorkstreamID: "00-071-02", FeatureID: "F071"})

	fsm.mu.Lock()
	fsm.state.Checkpoints = []CheckpointRecord{{
		Name:    "cp1",
		Result:  "passed",
		Details: map[string]interface{}{"k": "v"},
	}}
	fsm.mu.Unlock()

	checkpoints := fsm.GetCheckpoints()
	if len(checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(checkpoints))
	}

	checkpoints[0].Name = "changed"
	checkpoints[0].Details["k"] = "mutated"

	fresh := fsm.GetCheckpoints()
	if fresh[0].Name != "cp1" {
		t.Fatalf("checkpoint name mutated: got %s", fresh[0].Name)
	}
	if got := fresh[0].Details["k"]; got != "v" {
		t.Fatalf("checkpoint details mutated: got %v want v", got)
	}
}

func TestGetTransition(t *testing.T) {
	trans := GetTransition(StatePending, StateValidated)
	if trans == nil {
		t.Fatal("GetTransition() returned nil for valid transition")
	}

	if trans.Name != TransitionValidate {
		t.Errorf("GetTransition().Name = %v, want %v", trans.Name, TransitionValidate)
	}

	if !trans.PolicyCheck {
		t.Error("GetTransition().PolicyCheck = false, want true")
	}
}

func TestGetTransition_Invalid(t *testing.T) {
	trans := GetTransition(StatePending, StateCompleted)
	if trans != nil {
		t.Errorf("GetTransition() returned non-nil for invalid transition: %v", trans)
	}
}

func TestGetValidTransitions(t *testing.T) {
	trans := GetValidTransitions(StatePending)
	if len(trans) != 2 {
		t.Errorf("GetValidTransitions(StatePending) returned %d transitions, want 2", len(trans))
	}

	trans = GetValidTransitions(StateCompleted)
	if trans != nil {
		t.Errorf("GetValidTransitions(StateCompleted) = %v, want nil (terminal state)", trans)
	}
}

type mockDecisionMaker struct {
	decision *sdk.RuntimeDecision
	err      error
	called   bool
}

func (m *mockDecisionMaker) MakeDecision(ctx context.Context, req *sdk.DecisionRequest, ctx2 *sdk.DecisionContext) (*sdk.RuntimeDecision, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return m.decision, nil
}

func (m *mockDecisionMaker) MakeDecisionAsync(ctx context.Context, req *sdk.DecisionRequest, ctx2 *sdk.DecisionContext) (string, error) {
	m.called = true
	return "decision-123", m.err
}

func (m *mockDecisionMaker) Close() error {
	return nil
}

type testHook struct {
	beforeFunc func(ctx context.Context, from, to State, fsmCtx *FSMContext) error
	afterFunc  func(ctx context.Context, from, to State, fsmCtx *FSMContext, err error) error
}

func (h *testHook) BeforeTransition(ctx context.Context, from, to State, fsmCtx *FSMContext) error {
	if h.beforeFunc != nil {
		return h.beforeFunc(ctx, from, to, fsmCtx)
	}
	return nil
}

func (h *testHook) AfterTransition(ctx context.Context, from, to State, fsmCtx *FSMContext, err error) error {
	if h.afterFunc != nil {
		return h.afterFunc(ctx, from, to, fsmCtx, err)
	}
	return nil
}

func TestTransitionError(t *testing.T) {
	cause := errors.New("underlying error")
	te := &TransitionError{
		Code:      "TEST_ERROR",
		Message:   "test message",
		FromState: StatePending,
		ToState:   StateValidated,
		Timestamp: time.Now(),
		Retryable: true,
		Cause:     cause,
	}

	expected := "[TEST_ERROR] pending → validated: test message"
	if got := te.Error(); got != expected {
		t.Errorf("TransitionError.Error() = %v, want %v", got, expected)
	}

	unwrapped := te.Unwrap()
	if unwrapped != cause {
		t.Errorf("TransitionError.Unwrap() = %v, want %v", unwrapped, cause)
	}
}
