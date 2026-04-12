package workstream

import (
	"context"
	"errors"
	"fmt"
)

type DispatchLease struct {
	Target         ExecutionTarget
	ClaimedIssueID string
	IssueStates    []RuntimeIssueState
}

type DispatchError struct {
	Code      string   `json:"code"`
	FeatureID string   `json:"feature_id,omitempty"`
	LeafWSID  string   `json:"leaf_ws_id"`
	IssueID   string   `json:"issue_id,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
	Reason    string   `json:"reason,omitempty"`
	Cause     error    `json:"-"`
}

func (e *DispatchError) Error() string {
	msg := fmt.Sprintf("%s: feature=%s leaf=%s", e.Code, e.FeatureID, e.LeafWSID)
	if e.IssueID != "" {
		msg += fmt.Sprintf(" issue=%s", e.IssueID)
	}
	if e.Reason != "" {
		msg += fmt.Sprintf(" reason=%s", e.Reason)
	}
	if len(e.Conflicts) > 0 {
		msg += fmt.Sprintf(" conflicts=%v", e.Conflicts)
	}
	if e.Cause != nil {
		msg += fmt.Sprintf(": %v", e.Cause)
	}
	return msg
}

func (e *DispatchError) Unwrap() error {
	return e.Cause
}

func AcquireExecutionClaim(ctx context.Context, projectRoot, featureID, wsID string, opts CompileOptions, adapter RuntimeAdapter) (DispatchLease, error) {
	if adapter == nil {
		return DispatchLease{}, fmt.Errorf("runtime adapter is required")
	}

	target, err := ResolveExecutableLeaf(projectRoot, featureID, wsID, opts)
	if err != nil {
		return DispatchLease{}, err
	}
	_, active, err := resolveCurrentDispatchState(ctx, adapter, target, "")
	if err != nil {
		return DispatchLease{}, err
	}

	if err := adapter.ClaimIssue(ctx, active.ID); err != nil {
		return DispatchLease{}, &DispatchError{
			Code:      "issue_claim_failed",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   active.ID,
			Cause:     err,
		}
	}

	lease, err := RevalidateExecutionClaim(ctx, projectRoot, featureID, wsID, active.ID, opts, adapter)
	if err == nil {
		return lease, nil
	}

	if releaseErr := adapter.ReleaseClaim(ctx, active.ID); releaseErr != nil {
		return DispatchLease{}, &DispatchError{
			Code:      "dispatch_claim_release_failed",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   active.ID,
			Reason:    dispatchReason(err),
			Cause:     errors.Join(err, releaseErr),
		}
	}

	return DispatchLease{}, &DispatchError{
		Code:      "dispatch_aborted_revalidation",
		FeatureID: featureID,
		LeafWSID:  wsID,
		IssueID:   active.ID,
		Reason:    dispatchReason(err),
		Conflicts: dispatchConflicts(err),
		Cause:     err,
	}

}

func RevalidateExecutionClaim(ctx context.Context, projectRoot, featureID, wsID, claimedIssueID string, opts CompileOptions, adapter RuntimeAdapter) (DispatchLease, error) {
	if adapter == nil {
		return DispatchLease{}, fmt.Errorf("runtime adapter is required")
	}
	if claimedIssueID == "" {
		return DispatchLease{}, &DispatchError{
			Code:      "claimed_issue_missing",
			FeatureID: featureID,
			LeafWSID:  wsID,
			Reason:    "empty_session_claim",
		}
	}

	target, err := ResolveExecutableLeaf(projectRoot, featureID, wsID, opts)
	if err != nil {
		return DispatchLease{}, err
	}
	states, active, err := resolveCurrentDispatchState(ctx, adapter, target, claimedIssueID)
	if err != nil {
		return DispatchLease{}, err
	}

	claimedState, ok := FindIssueState(states, claimedIssueID)
	if !ok {
		return DispatchLease{}, &DispatchError{
			Code:      "claimed_issue_missing",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   claimedIssueID,
			Reason:    "not_bound",
		}
	}
	if !claimedState.IsOpen {
		return DispatchLease{}, &DispatchError{
			Code:      "claimed_issue_inactive",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   claimedIssueID,
			Reason:    claimedState.Status,
		}
	}
	if !claimedState.IsClaimed {
		return DispatchLease{}, &DispatchError{
			Code:      "claimed_issue_not_claimed",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   claimedIssueID,
		}
	}
	if active == nil {
		return DispatchLease{}, &DispatchError{
			Code:      "no_active_issue",
			FeatureID: featureID,
			LeafWSID:  wsID,
		}
	}
	if active.ID != claimedIssueID {
		return DispatchLease{}, &DispatchError{
			Code:      "active_issue_changed",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   active.ID,
			Reason:    claimedIssueID,
		}
	}

	return DispatchLease{
		Target:         target,
		ClaimedIssueID: claimedIssueID,
		IssueStates:    states,
	}, nil
}

func ReleaseExecutionClaim(ctx context.Context, adapter RuntimeAdapter, lease DispatchLease) error {
	if adapter == nil {
		return fmt.Errorf("runtime adapter is required")
	}
	if lease.ClaimedIssueID == "" {
		return nil
	}
	return adapter.ReleaseClaim(ctx, lease.ClaimedIssueID)
}

func resolveCurrentDispatchState(ctx context.Context, adapter RuntimeAdapter, target ExecutionTarget, allowedIssueID string) ([]RuntimeIssueState, *RuntimeIssueState, error) {
	states, err := adapter.QueryBoundIssues(ctx, target.Workstream)
	if err != nil {
		return nil, nil, err
	}

	conflicts := CompetingClaimedIssues(target.Workstream, states, allowedIssueID)
	if len(conflicts) > 0 {
		return nil, nil, &DispatchError{
			Code:      "leaf_conflict",
			FeatureID: target.Feature.FeatureID,
			LeafWSID:  target.Workstream.WSID,
			Conflicts: conflicts,
		}
	}

	active := ResolveActiveIssue(target.Workstream, states)
	if active == nil {
		return states, nil, &DispatchError{
			Code:      "no_active_issue",
			FeatureID: target.Feature.FeatureID,
			LeafWSID:  target.Workstream.WSID,
		}
	}
	return states, active, nil
}

func dispatchReason(err error) string {
	var dispatchErr *DispatchError
	if errors.As(err, &dispatchErr) && dispatchErr.Code != "" {
		return dispatchErr.Code
	}
	var queryErr *RuntimeQueryError
	if errors.As(err, &queryErr) && queryErr.Code != "" {
		return queryErr.Code
	}
	return "unknown"
}

func dispatchConflicts(err error) []string {
	var dispatchErr *DispatchError
	if errors.As(err, &dispatchErr) && len(dispatchErr.Conflicts) > 0 {
		return append([]string(nil), dispatchErr.Conflicts...)
	}
	return nil
}
