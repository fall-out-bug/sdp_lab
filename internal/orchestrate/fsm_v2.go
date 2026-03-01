package orchestrate

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sdp_dev/internal/adapters/sdk"
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
		maxRetries: 3,
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

	from := f.state.State

	if from.IsTerminal() {
		return &TransitionError{
			Code:      "TERMINAL_STATE",
			Message:   "cannot transition from terminal state",
			FromState: from,
			ToState:   to,
			Timestamp: time.Now(),
			Retryable: false,
		}
	}

	transition := GetTransition(from, to)
	if transition == nil {
		return &TransitionError{
			Code:      "INVALID_TRANSITION",
			Message:   fmt.Sprintf("transition %s → %s is not defined", from, to),
			FromState: from,
			ToState:   to,
			Timestamp: time.Now(),
			Retryable: false,
		}
	}

	for _, hook := range f.hooks {
		if err := hook.BeforeTransition(ctx, from, to, f.context); err != nil {
			return &TransitionError{
				Code:      "HOOK_FAILED",
				Message:   fmt.Sprintf("before hook failed: %v", err),
				FromState: from,
				ToState:   to,
				Timestamp: time.Now(),
				Retryable: false,
				Cause:     err,
			}
		}
	}

	if transition.PolicyCheck && f.decisionMaker != nil {
		decision, err := f.checkPolicy(ctx, transition)
		if err != nil {
			return &TransitionError{
				Code:      "POLICY_CHECK_FAILED",
				Message:   fmt.Sprintf("policy check failed: %v", err),
				FromState: from,
				ToState:   to,
				Timestamp: time.Now(),
				Retryable: false,
				Cause:     err,
			}
		}
		if decision.Decision != "allow" {
			return &TransitionError{
				Code:      "POLICY_DENIED",
				Message:   fmt.Sprintf("policy denied transition: %s", decision.Reason.Message),
				FromState: from,
				ToState:   to,
				Timestamp: time.Now(),
				Retryable: false,
			}
		}
	}

	var transitionErr error
	if transition.Validator != nil {
		if err := transition.Validator(ctx, f.context, f.state); err != nil {
			transitionErr = &TransitionError{
				Code:      "VALIDATION_FAILED",
				Message:   fmt.Sprintf("validation failed: %v", err),
				FromState: from,
				ToState:   to,
				Timestamp: time.Now(),
				Retryable: true,
				Cause:     err,
			}
		}
	}

	if transitionErr == nil && transition.Action != nil {
		if err := transition.Action(ctx, f.context, f.state); err != nil {
			transitionErr = &TransitionError{
				Code:      "ACTION_FAILED",
				Message:   fmt.Sprintf("transition action failed: %v", err),
				FromState: from,
				ToState:   to,
				Timestamp: time.Now(),
				Retryable: true,
				Cause:     err,
			}
		}
	}

	now := time.Now()
	if transitionErr != nil {
		f.state.LastError = transitionErr.(*TransitionError)
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
	return nil
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

	event := &sdk.OrchestrationEvent{
		SpecVersion: "1.0.0",
		EventID:     fmt.Sprintf("%s-%d", f.context.WorkstreamID, time.Now().UnixNano()),
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
			"workstream": f.context.WorkstreamID,
		},
		Context: &sdk.ExecutionContext{
			WorkstreamID: f.context.WorkstreamID,
			FeatureID:    f.context.FeatureID,
			BeadsID:      f.context.BeadsID,
			SessionID:    f.context.SessionID,
			GitContext: &sdk.GitContext{
				Branch:    f.context.GitBranch,
				CommitSHA: f.context.GitCommitSHA,
			},
		},
	}

	if err != nil {
		event.Payload["error"] = err.Error()
	}

	_ = f.eventProducer.EmitEventAsync(ctx, event)
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
	return f.state
}

func (f *FSMV2) GetCheckpoints() []CheckpointRecord {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.Checkpoints
}
