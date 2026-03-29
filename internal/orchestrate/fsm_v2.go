package orchestrate

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sdp_dev/archive/adapters-sdk"
)

type State string

const (
	StatePending    State = "pending"
	StateValidated  State = "validated"
	StateAssigned   State = "assigned"
	StateExecuted   State = "executed"
	StateReviewed   State = "reviewed"
	StateCompleted  State = "completed"
	StateFailed     State = "failed"
	StateRolledBack State = "rolled_back"
)

const defaultFSMMaxRetries = 3

func (s State) String() string {
	return string(s)
}

func (s State) IsTerminal() bool {
	return s == StateCompleted || s == StateFailed || s == StateRolledBack
}

type FSMContext struct {
	WorkstreamID string
	FeatureID    string
	BeadsID      string
	SessionID    string
	GitBranch    string
	GitCommitSHA string
	Actor        *sdk.Actor
	Labels       map[string]string
}

type FSMState struct {
	State       State
	EnteredAt   time.Time
	ExitedAt    *time.Time
	Attempts    int
	LastError   *TransitionError
	Checkpoints []CheckpointRecord
}

type CheckpointRecord struct {
	Name      string
	Timestamp time.Time
	Result    string // "passed", "failed", "skipped"
	Details   map[string]interface{}
}

type TransitionError struct {
	Code      string
	Message   string
	FromState State
	ToState   State
	Timestamp time.Time
	Retryable bool
	Cause     error
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("[%s] %s → %s: %s", e.Code, e.FromState, e.ToState, e.Message)
}

func (e *TransitionError) Unwrap() error {
	return e.Cause
}

type FSMV2 struct {
	mu            sync.RWMutex
	state         *FSMState
	context       *FSMContext
	decisionMaker sdk.DecisionMaker
	eventProducer sdk.EventProducer
	hooks         []TransitionHook
	maxRetries    int
}

type TransitionHook interface {
	BeforeTransition(ctx context.Context, from, to State, fsmCtx *FSMContext) error
	AfterTransition(ctx context.Context, from, to State, fsmCtx *FSMContext, err error) error
}

type FSMOption func(*FSMV2)

func WithDecisionMaker(dm sdk.DecisionMaker) FSMOption {
	return func(f *FSMV2) {
		f.decisionMaker = dm
	}
}

func WithEventProducer(ep sdk.EventProducer) FSMOption {
	return func(f *FSMV2) {
		f.eventProducer = ep
	}
}

func WithHooks(hooks ...TransitionHook) FSMOption {
	return func(f *FSMV2) {
		f.hooks = append(f.hooks, hooks...)
	}
}

func WithMaxRetries(max int) FSMOption {
	return func(f *FSMV2) {
		f.maxRetries = max
	}
}

func NewFSMV2(ctx context.Context, fsmCtx *FSMContext, opts ...FSMOption) *FSMV2 {
	fsm := &FSMV2{
		state: &FSMState{
			State:     StatePending,
			EnteredAt: time.Now(),
		},
		context:    fsmCtx,
		maxRetries: defaultFSMMaxRetries,
	}

	for _, opt := range opts {
		opt(fsm)
	}

	return fsm
}

func (f *FSMV2) CurrentState() State {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.State
}

func (f *FSMV2) Context() *FSMContext {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.context
}

func (f *FSMV2) Transition(ctx context.Context, to State) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	from, transition, err := f.validateTransitionRequest(to)
	if err != nil {
		return err
	}
	if err := f.runBeforeTransitionHooks(ctx, from, to); err != nil {
		return err
	}
	if err := f.enforcePolicyForTransition(ctx, from, to, transition); err != nil {
		return err
	}

	if transitionErr := f.executeTransitionWork(ctx, from, to, transition); transitionErr != nil {
		return f.applyTransitionFailure(ctx, from, to, transition, transitionErr)
	}

	f.applyTransitionSuccess(ctx, from, to, transition)
	return nil
}

func (f *FSMV2) validateTransitionRequest(to State) (State, *Transition, error) {
	from := f.state.State
	if from.IsTerminal() {
		return from, nil, newTransitionError("TERMINAL_STATE", "cannot transition from terminal state", from, to, false, nil)
	}

	transition := GetTransition(from, to)
	if transition == nil {
		msg := fmt.Sprintf("transition %s → %s is not defined", from, to)
		return from, nil, newTransitionError("INVALID_TRANSITION", msg, from, to, false, nil)
	}

	return from, transition, nil
}

func (f *FSMV2) runBeforeTransitionHooks(ctx context.Context, from, to State) error {
	for _, hook := range f.hooks {
		if err := hook.BeforeTransition(ctx, from, to, f.context); err != nil {
			msg := fmt.Sprintf("before hook failed: %v", err)
			return newTransitionError("HOOK_FAILED", msg, from, to, false, err)
		}
	}
	return nil
}

func (f *FSMV2) enforcePolicyForTransition(ctx context.Context, from, to State, transition *Transition) error {
	if !transition.PolicyCheck || f.decisionMaker == nil {
		return nil
	}

	decision, err := f.checkPolicy(ctx, transition)
	if err != nil {
		msg := fmt.Sprintf("policy check failed: %v", err)
		return newTransitionError("POLICY_CHECK_FAILED", msg, from, to, false, err)
	}
	if decision.Decision != "allow" {
		msg := fmt.Sprintf("policy denied transition: %s", decision.Reason.Message)
		return newTransitionError("POLICY_DENIED", msg, from, to, false, nil)
	}

	return nil
}

func (f *FSMV2) executeTransitionWork(ctx context.Context, from, to State, transition *Transition) error {
	if transition.Validator != nil {
		if err := transition.Validator(ctx, f.context, f.state); err != nil {
			msg := fmt.Sprintf("validation failed: %v", err)
			return newTransitionError("VALIDATION_FAILED", msg, from, to, true, err)
		}
	}

	if transition.Action != nil {
		if err := transition.Action(ctx, f.context, f.state); err != nil {
			msg := fmt.Sprintf("transition action failed: %v", err)
			return newTransitionError("ACTION_FAILED", msg, from, to, true, err)
		}
	}

	return nil
}

func (f *FSMV2) applyTransitionFailure(ctx context.Context, from, to State, transition *Transition, transitionErr error) error {
	if te, ok := transitionErr.(*TransitionError); ok {
		f.state.LastError = te
	} else {
		f.state.LastError = newTransitionError("UNKNOWN_ERROR", transitionErr.Error(), from, to, false, transitionErr)
	}

	f.state.Attempts++
	if f.state.Attempts >= f.maxRetries {
		toState := StateFailed
		if transition.OnFailure != nil {
			toState = transition.OnFailure()
		}
		f.state.State = toState
	}

	f.emitEvent(ctx, "transition_failed", from, to, transitionErr)
	return transitionErr
}

func (f *FSMV2) applyTransitionSuccess(ctx context.Context, from, to State, transition *Transition) {
	now := time.Now()
	nowPtr := &now
	f.state.ExitedAt = nowPtr

	oldState := f.state
	f.state = &FSMState{
		State:     to,
		EnteredAt: now,
		Attempts:  0,
		Checkpoints: append(oldState.Checkpoints, CheckpointRecord{
			Name:      string(transition.Name),
			Timestamp: now,
			Result:    "passed",
		}),
	}

	for _, hook := range f.hooks {
		if err := hook.AfterTransition(ctx, from, to, f.context, nil); err != nil {
			f.emitEvent(ctx, "hook_warning", from, to, err)
		}
	}

	f.emitEvent(ctx, "state_transition", from, to, nil)
}

func newTransitionError(code, message string, from, to State, retryable bool, cause error) *TransitionError {
	return &TransitionError{
		Code:      code,
		Message:   message,
		FromState: from,
		ToState:   to,
		Timestamp: time.Now(),
		Retryable: retryable,
		Cause:     cause,
	}
}

func (f *FSMV2) checkPolicy(ctx context.Context, transition *Transition) (*sdk.RuntimeDecision, error) {
	decisionCtx := &sdk.DecisionContext{
		Request: sdk.DecisionRequest{
			Action:   string(transition.Name),
			Resource: "fsm_transition",
		},
		Actor:        f.context.Actor,
		WorkstreamID: f.context.WorkstreamID,
		FeatureID:    f.context.FeatureID,
		SessionID:    f.context.SessionID,
	}

	return f.decisionMaker.MakeDecision(ctx, &decisionCtx.Request, decisionCtx)
}

func (f *FSMV2) emitEvent(ctx context.Context, eventType string, from, to State, err error) {
	if f.eventProducer == nil {
		return
	}

	event := buildOrchestrationEvent(f.context, eventType, from, to, err)
	_ = f.eventProducer.EmitEventAsync(ctx, event)
}

func buildOrchestrationEvent(fsmCtx *FSMContext, eventType string, from, to State, err error) *sdk.OrchestrationEvent {
	event := &sdk.OrchestrationEvent{
		SpecVersion: "1.0.0",
		EventID:     fmt.Sprintf("%s-%d", fsmCtx.WorkstreamID, time.Now().UnixNano()),
		Timestamp:   time.Now(),
		Source: sdk.EventSource{
			System:    "sdp-lab",
			Component: "fsm-v2",
			Version:   "1.0.0",
		},
		EventType: eventType,
		Payload: map[string]interface{}{
			"from_state": string(from),
			"to_state":   string(to),
			"workstream": fsmCtx.WorkstreamID,
		},
		Context: &sdk.ExecutionContext{
			WorkstreamID: fsmCtx.WorkstreamID,
			FeatureID:    fsmCtx.FeatureID,
			BeadsID:      fsmCtx.BeadsID,
			SessionID:    fsmCtx.SessionID,
			GitContext: &sdk.GitContext{
				Branch:    fsmCtx.GitBranch,
				CommitSHA: fsmCtx.GitCommitSHA,
			},
		},
	}

	if err != nil {
		event.Payload["error"] = err.Error()
	}

	return event
}

func (f *FSMV2) Validate(ctx context.Context) error {
	return f.Transition(ctx, StateValidated)
}

func (f *FSMV2) Assign(ctx context.Context) error {
	return f.Transition(ctx, StateAssigned)
}

func (f *FSMV2) Execute(ctx context.Context) error {
	return f.Transition(ctx, StateExecuted)
}

func (f *FSMV2) Review(ctx context.Context) error {
	return f.Transition(ctx, StateReviewed)
}

func (f *FSMV2) Complete(ctx context.Context) error {
	return f.Transition(ctx, StateCompleted)
}

func (f *FSMV2) Fail(ctx context.Context, reason string) error {
	f.mu.Lock()
	f.state.LastError = &TransitionError{
		Code:      "EXPLICIT_FAILURE",
		Message:   reason,
		FromState: f.state.State,
		ToState:   StateFailed,
		Timestamp: time.Now(),
		Retryable: false,
	}
	f.mu.Unlock()

	return f.Transition(ctx, StateFailed)
}

func (f *FSMV2) Rollback(ctx context.Context) error {
	return f.Transition(ctx, StateRolledBack)
}

func (f *FSMV2) GetState() *FSMState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.state == nil {
		return nil
	}

	stateCopy := *f.state
	stateCopy.LastError = cloneTransitionError(f.state.LastError)
	stateCopy.Checkpoints = cloneCheckpointRecords(f.state.Checkpoints)
	return &stateCopy
}

func (f *FSMV2) GetCheckpoints() []CheckpointRecord {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.state == nil {
		return nil
	}
	return cloneCheckpointRecords(f.state.Checkpoints)
}

func cloneCheckpointRecords(in []CheckpointRecord) []CheckpointRecord {
	if len(in) == 0 {
		return nil
	}
	out := make([]CheckpointRecord, len(in))
	for i := range in {
		out[i] = in[i]
		if in[i].Details != nil {
			detailsCopy := make(map[string]interface{}, len(in[i].Details))
			for k, v := range in[i].Details {
				detailsCopy[k] = v
			}
			out[i].Details = detailsCopy
		}
	}
	return out
}

func cloneTransitionError(in *TransitionError) *TransitionError {
	if in == nil {
		return nil
	}
	errCopy := *in
	return &errCopy
}
