package workstream

import (
	"context"
	"errors"
	"fmt"
)

const (
	DispatchAttemptTotal             = "dispatch_attempt_total"
	DispatchSuccessTotal             = "dispatch_success_total"
	DispatchAbortedRevalidationTotal = "dispatch_aborted_revalidation_total"
	DispatchLeafConflictTotal        = "dispatch_leaf_conflict_total"
	DispatchClaimReleaseFailedTotal  = "dispatch_claim_release_failed_total"
	DispatchBeadsQueryFailedTotal    = "dispatch_beads_query_failed_total"
)

type DispatchLease struct {
	Target         ExecutionTarget
	ClaimedIssueID string
	IssueStates    []RuntimeIssueState
}

type DispatchDiagnostic struct {
	Code      string            `json:"code"`
	FeatureID string            `json:"feature_id,omitempty"`
	LeafWSID  string            `json:"leaf_ws_id"`
	IssueID   string            `json:"issue_id,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	Conflicts []string          `json:"conflicts,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type DispatchObserver interface {
	IncrementCounter(name string)
	RecordDiagnostic(diag DispatchDiagnostic)
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

func AcquireExecutionClaim(ctx context.Context, projectRoot, featureID, wsID string, opts CompileOptions, adapter RuntimeAdapter, observer DispatchObserver) (DispatchLease, error) {
	if adapter == nil {
		return DispatchLease{}, fmt.Errorf("runtime adapter is required")
	}
	emitDispatchCounter(observer, DispatchAttemptTotal)

	target, err := ResolveExecutableLeaf(projectRoot, featureID, wsID, opts)
	if err != nil {
		return DispatchLease{}, err
	}
	_, active, err := resolveCurrentDispatchState(ctx, adapter, target, "")
	if err != nil {
		emitDispatchFailure(observer, err)
		return DispatchLease{}, err
	}

	if err := adapter.ClaimIssue(ctx, active.ID); err != nil {
		claimErr := &DispatchError{
			Code:      "issue_claim_failed",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   active.ID,
			Cause:     err,
		}
		emitDispatchFailure(observer, claimErr)
		return DispatchLease{}, claimErr
	}

	lease, err := RevalidateExecutionClaim(ctx, projectRoot, featureID, wsID, active.ID, opts, adapter, observer)
	if err == nil {
		emitDispatchCounter(observer, DispatchSuccessTotal)
		emitDispatchDiagnostic(observer, DispatchDiagnostic{
			Code:      "dispatch_success",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   active.ID,
		})
		return lease, nil
	}

	if releaseErr := adapter.ReleaseClaim(ctx, active.ID, active.Status); releaseErr != nil {
		releaseFailure := &DispatchError{
			Code:      "dispatch_claim_release_failed",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   active.ID,
			Reason:    dispatchReason(err),
			Cause:     errors.Join(err, releaseErr),
		}
		emitDispatchCounter(observer, DispatchClaimReleaseFailedTotal)
		emitDispatchDiagnostic(observer, diagnosticFromDispatchError(releaseFailure))
		return DispatchLease{}, releaseFailure
	}

	abortErr := &DispatchError{
		Code:      "dispatch_aborted_revalidation",
		FeatureID: featureID,
		LeafWSID:  wsID,
		IssueID:   active.ID,
		Reason:    dispatchReason(err),
		Conflicts: dispatchConflicts(err),
		Cause:     err,
	}
	emitDispatchCounter(observer, DispatchAbortedRevalidationTotal)
	emitDispatchDiagnostic(observer, diagnosticFromDispatchError(abortErr))
	return DispatchLease{}, abortErr

}

func RevalidateExecutionClaim(ctx context.Context, projectRoot, featureID, wsID, claimedIssueID string, opts CompileOptions, adapter RuntimeAdapter, observer DispatchObserver) (DispatchLease, error) {
	if adapter == nil {
		return DispatchLease{}, fmt.Errorf("runtime adapter is required")
	}
	if claimedIssueID == "" {
		err := &DispatchError{
			Code:      "claimed_issue_missing",
			FeatureID: featureID,
			LeafWSID:  wsID,
			Reason:    "empty_session_claim",
		}
		emitDispatchFailure(observer, err)
		return DispatchLease{}, err
	}

	target, err := ResolveExecutableLeaf(projectRoot, featureID, wsID, opts)
	if err != nil {
		return DispatchLease{}, err
	}
	states, active, err := resolveCurrentDispatchState(ctx, adapter, target, claimedIssueID)
	if err != nil {
		emitDispatchFailure(observer, err)
		return DispatchLease{}, err
	}

	claimedState, ok := FindIssueState(states, claimedIssueID)
	if !ok {
		err := &DispatchError{
			Code:      "claimed_issue_missing",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   claimedIssueID,
			Reason:    "not_bound",
		}
		emitDispatchFailure(observer, err)
		return DispatchLease{}, err
	}
	if !claimedState.IsOpen {
		err := &DispatchError{
			Code:      "claimed_issue_inactive",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   claimedIssueID,
			Reason:    claimedState.Status,
		}
		emitDispatchFailure(observer, err)
		return DispatchLease{}, err
	}
	if !claimedState.IsClaimed {
		err := &DispatchError{
			Code:      "claimed_issue_not_claimed",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   claimedIssueID,
		}
		emitDispatchFailure(observer, err)
		return DispatchLease{}, err
	}
	if active == nil {
		err := &DispatchError{
			Code:      "no_active_issue",
			FeatureID: featureID,
			LeafWSID:  wsID,
		}
		emitDispatchFailure(observer, err)
		return DispatchLease{}, err
	}
	if active.ID != claimedIssueID {
		err := &DispatchError{
			Code:      "active_issue_changed",
			FeatureID: featureID,
			LeafWSID:  wsID,
			IssueID:   active.ID,
			Reason:    claimedIssueID,
		}
		emitDispatchFailure(observer, err)
		return DispatchLease{}, err
	}

	return DispatchLease{
		Target:         target,
		ClaimedIssueID: claimedIssueID,
		IssueStates:    states,
	}, nil
}

func ReleaseExecutionClaim(ctx context.Context, adapter RuntimeAdapter, lease DispatchLease, observer DispatchObserver) error {
	if adapter == nil {
		return fmt.Errorf("runtime adapter is required")
	}
	if lease.ClaimedIssueID == "" {
		return nil
	}
	if err := adapter.ReleaseClaim(ctx, lease.ClaimedIssueID, ""); err != nil {
		releaseErr := &DispatchError{
			Code:      "dispatch_claim_release_failed",
			FeatureID: lease.Target.Feature.FeatureID,
			LeafWSID:  lease.Target.Workstream.WSID,
			IssueID:   lease.ClaimedIssueID,
			Cause:     err,
		}
		emitDispatchCounter(observer, DispatchClaimReleaseFailedTotal)
		emitDispatchDiagnostic(observer, diagnosticFromDispatchError(releaseErr))
		return releaseErr
	}
	emitDispatchDiagnostic(observer, DispatchDiagnostic{
		Code:      "dispatch_claim_released",
		FeatureID: lease.Target.Feature.FeatureID,
		LeafWSID:  lease.Target.Workstream.WSID,
		IssueID:   lease.ClaimedIssueID,
	})
	return nil
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

func DispatchDiagnosticFromError(err error) (DispatchDiagnostic, bool) {
	var dispatchErr *DispatchError
	if errors.As(err, &dispatchErr) {
		return diagnosticFromDispatchError(dispatchErr), true
	}
	var queryErr *RuntimeQueryError
	if errors.As(err, &queryErr) {
		return DispatchDiagnostic{
			Code:     queryErr.Code,
			LeafWSID: queryErr.LeafWSID,
			Reason:   queryErr.Reason,
			Fields: map[string]string{
				"issue_ids": fmt.Sprintf("%v", queryErr.IssueIDs),
			},
		}, true
	}
	return DispatchDiagnostic{}, false
}

func emitDispatchFailure(observer DispatchObserver, err error) {
	if observer == nil {
		return
	}
	diag, ok := DispatchDiagnosticFromError(err)
	if !ok {
		return
	}
	switch diag.Code {
	case "beads_query_failed":
		observer.IncrementCounter(DispatchBeadsQueryFailedTotal)
	case "leaf_conflict":
		observer.IncrementCounter(DispatchLeafConflictTotal)
	case "dispatch_claim_release_failed":
		observer.IncrementCounter(DispatchClaimReleaseFailedTotal)
	}
	observer.RecordDiagnostic(diag)
}

func emitDispatchCounter(observer DispatchObserver, name string) {
	if observer != nil {
		observer.IncrementCounter(name)
	}
}

func emitDispatchDiagnostic(observer DispatchObserver, diag DispatchDiagnostic) {
	if observer != nil {
		observer.RecordDiagnostic(diag)
	}
}

func diagnosticFromDispatchError(err *DispatchError) DispatchDiagnostic {
	diag := DispatchDiagnostic{
		Code:      err.Code,
		FeatureID: err.FeatureID,
		LeafWSID:  err.LeafWSID,
		IssueID:   err.IssueID,
		Reason:    err.Reason,
		Conflicts: append([]string(nil), err.Conflicts...),
	}
	if err.Cause != nil {
		diag.Fields = map[string]string{
			"cause": err.Cause.Error(),
		}
	}
	return diag
}
