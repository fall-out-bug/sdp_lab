package orchestrate

import (
	"context"
	"fmt"
)

type TransitionName string

const (
	TransitionValidate TransitionName = "validate"
	TransitionAssign   TransitionName = "assign"
	TransitionExecute  TransitionName = "execute"
	TransitionReview   TransitionName = "review"
	TransitionComplete TransitionName = "complete"
	TransitionFail     TransitionName = "fail"
	TransitionRollback TransitionName = "rollback"
	TransitionRetry    TransitionName = "retry"
)

type TransitionValidator func(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error

type TransitionAction func(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error

type FailureHandler func() State

type Transition struct {
	Name        TransitionName
	From        State
	To          State
	PolicyCheck bool
	Validator   TransitionValidator
	Action      TransitionAction
	OnFailure   FailureHandler
	Description string
}

var transitions = map[State][]Transition{
	StatePending: {
		{
			Name:        TransitionValidate,
			From:        StatePending,
			To:          StateValidated,
			PolicyCheck: true,
			Validator:   validatePendingToValidated,
			Description: "Validate input and prerequisites before work can begin",
		},
		{
			Name:        TransitionFail,
			From:        StatePending,
			To:          StateFailed,
			PolicyCheck: false,
			Description: "Explicit failure from pending state",
		},
	},
	StateValidated: {
		{
			Name:        TransitionAssign,
			From:        StateValidated,
			To:          StateAssigned,
			PolicyCheck: true,
			Validator:   validateValidatedToAssigned,
			Action:      actionAssignWork,
			Description: "Assign work to available agent with matching skills",
		},
		{
			Name:        TransitionFail,
			From:        StateValidated,
			To:          StateFailed,
			PolicyCheck: false,
			Description: "Explicit failure from validated state",
		},
	},
	StateAssigned: {
		{
			Name:        TransitionExecute,
			From:        StateAssigned,
			To:          StateExecuted,
			PolicyCheck: true,
			Validator:   validateAssignedToExecuted,
			Action:      actionExecuteWork,
			Description: "Execute the assigned work within sandbox boundaries",
		},
		{
			Name:        TransitionFail,
			From:        StateAssigned,
			To:          StateFailed,
			PolicyCheck: false,
			Description: "Explicit failure from assigned state",
		},
	},
	StateExecuted: {
		{
			Name:        TransitionReview,
			From:        StateExecuted,
			To:          StateReviewed,
			PolicyCheck: true,
			Validator:   validateExecutedToReviewed,
			Action:      actionReviewWork,
			Description: "Review completed work for quality and correctness",
		},
		{
			Name:        TransitionFail,
			From:        StateExecuted,
			To:          StateFailed,
			PolicyCheck: false,
			Description: "Explicit failure from executed state",
		},
	},
	StateReviewed: {
		{
			Name:        TransitionComplete,
			From:        StateReviewed,
			To:          StateCompleted,
			PolicyCheck: true,
			Validator:   validateReviewedToCompleted,
			Action:      actionFinalizeWork,
			Description: "Finalize and mark work as complete",
		},
		{
			Name:        TransitionFail,
			From:        StateReviewed,
			To:          StateFailed,
			PolicyCheck: false,
			Description: "Explicit failure from reviewed state",
		},
	},
	StateFailed: {
		{
			Name:        TransitionRetry,
			From:        StateFailed,
			To:          StatePending,
			PolicyCheck: true,
			Validator:   validateRetry,
			Description: "Retry failed work from the beginning",
		},
		{
			Name:        TransitionRollback,
			From:        StateFailed,
			To:          StateRolledBack,
			PolicyCheck: false,
			Action:      actionRollback,
			Description: "Roll back all changes and restore previous state",
		},
	},
}

func GetTransition(from, to State) *Transition {
	trans, ok := transitions[from]
	if !ok {
		return nil
	}

	for _, t := range trans {
		if t.To == to {
			return &t
		}
	}

	return nil
}

func GetValidTransitions(from State) []Transition {
	if trans, ok := transitions[from]; ok {
		return trans
	}
	return nil
}

func validatePendingToValidated(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	if fsmCtx.WorkstreamID == "" {
		return fmt.Errorf("workstream_id is required")
	}
	if fsmCtx.FeatureID == "" {
		return fmt.Errorf("feature_id is required")
	}
	return nil
}

func validateValidatedToAssigned(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	if fsmCtx.Actor == nil {
		return fmt.Errorf("actor is required for assignment")
	}
	return nil
}

func validateAssignedToExecuted(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	if fsmCtx.Actor == nil {
		return fmt.Errorf("actor is required for execution")
	}
	if fsmCtx.SessionID == "" {
		return fmt.Errorf("session_id is required for execution")
	}
	return nil
}

func validateExecutedToReviewed(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	return nil
}

func validateReviewedToCompleted(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	if len(state.Checkpoints) == 0 {
		return fmt.Errorf("at least one checkpoint is required for completion")
	}

	for _, cp := range state.Checkpoints {
		if cp.Result == "failed" {
			return fmt.Errorf("checkpoint %s failed: cannot complete", cp.Name)
		}
	}

	return nil
}

func validateRetry(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	if state.Attempts == 0 {
		return fmt.Errorf("no previous attempts to retry")
	}
	return nil
}

func actionAssignWork(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	return nil
}

func actionExecuteWork(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	return nil
}

func actionReviewWork(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	return nil
}

func actionFinalizeWork(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	return nil
}

func actionRollback(ctx context.Context, fsmCtx *FSMContext, state *FSMState) error {
	return nil
}

type PolicyCheckpointer interface {
	Check(ctx context.Context, transition TransitionName, fsmCtx *FSMContext) error
}

type AuditLogger interface {
	LogTransition(ctx context.Context, from, to State, fsmCtx *FSMContext, err error) error
}

type DefaultAuditLogger struct{}

func (l *DefaultAuditLogger) LogTransition(ctx context.Context, from, to State, fsmCtx *FSMContext, err error) error {
	return nil
}

type LoggingHook struct {
	logger AuditLogger
}

func NewLoggingHook(logger AuditLogger) *LoggingHook {
	return &LoggingHook{logger: logger}
}

func (h *LoggingHook) BeforeTransition(ctx context.Context, from, to State, fsmCtx *FSMContext) error {
	return nil
}

func (h *LoggingHook) AfterTransition(ctx context.Context, from, to State, fsmCtx *FSMContext, err error) error {
	return h.logger.LogTransition(ctx, from, to, fsmCtx, err)
}
