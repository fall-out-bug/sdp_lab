package orchestrate

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockBranchExecutor struct {
	executeFunc func(ctx context.Context, branch *Branch) (interface{}, error)
	cancelFunc  func(ctx context.Context, branchID BranchID) error
}

func (m *mockBranchExecutor) ExecuteBranch(ctx context.Context, branch *Branch) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, branch)
	}
	return map[string]interface{}{"result": string(branch.ID)}, nil
}

func (m *mockBranchExecutor) CancelBranch(ctx context.Context, branchID BranchID) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, branchID)
	}
	return nil
}

func TestNewParallelExecutor(t *testing.T) {
	executor := NewParallelExecutor(&mockBranchExecutor{})
	if executor == nil {
		t.Fatal("expected non-nil executor")
	}
	if executor.maxBranches != 10 {
		t.Errorf("expected maxBranches 10, got %d", executor.maxBranches)
	}
}

func TestParallelExecutorWithOptions(t *testing.T) {
	executor := NewParallelExecutor(
		&mockBranchExecutor{},
		WithMaxBranches(5),
		WithTimeout(10*time.Minute),
	)
	if executor.maxBranches != 5 {
		t.Errorf("expected maxBranches 5, got %d", executor.maxBranches)
	}
	if executor.timeout != 10*time.Minute {
		t.Errorf("expected timeout 10m, got %v", executor.timeout)
	}
}

func TestFanOut(t *testing.T) {
	executor := NewParallelExecutor(&mockBranchExecutor{})
	specs := []BranchSpec{
		{ID: "branch-1", Name: "Branch 1"},
		{ID: "branch-2", Name: "Branch 2"},
	}

	branchIDs, err := executor.FanOut(context.Background(), specs)
	if err != nil {
		t.Fatalf("fan-out failed: %v", err)
	}
	if len(branchIDs) != 2 {
		t.Errorf("expected 2 branch IDs, got %d", len(branchIDs))
	}

	status := executor.GetStatus()
	if status.TotalBranches != 2 {
		t.Errorf("expected 2 total branches, got %d", status.TotalBranches)
	}
}

func TestFanOutExceedsMax(t *testing.T) {
	executor := NewParallelExecutor(&mockBranchExecutor{}, WithMaxBranches(2))
	specs := []BranchSpec{
		{ID: "branch-1", Name: "Branch 1"},
		{ID: "branch-2", Name: "Branch 2"},
		{ID: "branch-3", Name: "Branch 3"},
	}

	_, err := executor.FanOut(context.Background(), specs)
	if err == nil {
		t.Error("expected error for exceeding max branches")
	}
}

func TestExecuteBranches(t *testing.T) {
	executor := NewParallelExecutor(&mockBranchExecutor{})
	specs := []BranchSpec{
		{ID: "branch-1", Name: "Branch 1"},
		{ID: "branch-2", Name: "Branch 2"},
	}

	branchIDs, _ := executor.FanOut(context.Background(), specs)
	resultChan := executor.Execute(context.Background(), branchIDs)

	var results []BranchResult
	for r := range resultChan {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Status != BranchStatusSucceeded {
			t.Errorf("expected succeeded status, got %s", r.Status)
		}
	}
}

func TestExecuteWithFailure(t *testing.T) {
	executor := NewParallelExecutor(&mockBranchExecutor{
		executeFunc: func(ctx context.Context, branch *Branch) (interface{}, error) {
			if branch.ID == "branch-fail" {
				return nil, errors.New("intentional failure")
			}
			return "success", nil
		},
	})

	specs := []BranchSpec{
		{ID: "branch-ok", Name: "OK Branch"},
		{ID: "branch-fail", Name: "Fail Branch"},
	}

	branchIDs, _ := executor.FanOut(context.Background(), specs)
	resultChan := executor.Execute(context.Background(), branchIDs)

	var succeeded, failed int
	for r := range resultChan {
		if r.Status == BranchStatusSucceeded {
			succeeded++
		} else if r.Status == BranchStatusFailed {
			failed++
		}
	}

	if succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", succeeded)
	}
	if failed != 1 {
		t.Errorf("expected 1 failed, got %d", failed)
	}
}

func TestFanIn(t *testing.T) {
	executor := NewParallelExecutor(
		&mockBranchExecutor{
			executeFunc: func(ctx context.Context, branch *Branch) (interface{}, error) {
				return map[string]interface{}{"data": string(branch.ID)}, nil
			},
		},
		WithMergePolicy(NewDefaultMergePolicy()),
	)

	specs := []BranchSpec{
		{ID: "branch-1", Name: "Branch 1"},
		{ID: "branch-2", Name: "Branch 2"},
	}

	branchIDs, _ := executor.FanOut(context.Background(), specs)
	_ = executor.Execute(context.Background(), branchIDs)

	result, err := executor.FanIn(context.Background(), branchIDs)
	if err != nil {
		t.Fatalf("fan-in failed: %v", err)
	}
	if result.Status != MergeStatusSuccess {
		t.Errorf("expected success status, got %s", result.Status)
	}
	if len(result.MergedFrom) != 2 {
		t.Errorf("expected 2 merged branches, got %d", len(result.MergedFrom))
	}
}

func TestFanInWithFailures(t *testing.T) {
	executor := NewParallelExecutor(
		&mockBranchExecutor{
			executeFunc: func(ctx context.Context, branch *Branch) (interface{}, error) {
				if branch.ID == "branch-fail" {
					return nil, errors.New("failure")
				}
				return "success", nil
			},
		},
		WithMergePolicy(NewDefaultMergePolicy()),
	)

	specs := []BranchSpec{
		{ID: "branch-ok", Name: "OK Branch"},
		{ID: "branch-fail", Name: "Fail Branch"},
	}

	branchIDs, _ := executor.FanOut(context.Background(), specs)
	_ = executor.Execute(context.Background(), branchIDs)

	result, err := executor.FanIn(context.Background(), branchIDs)
	if err != nil {
		t.Fatalf("unexpected error with fail-fast disabled")
	}
	if len(result.FailedBranch) != 1 {
		t.Errorf("expected 1 failed branch, got %d", len(result.FailedBranch))
	}
}

func TestGetBranch(t *testing.T) {
	executor := NewParallelExecutor(&mockBranchExecutor{})
	specs := []BranchSpec{{ID: "branch-1", Name: "Branch 1"}}

	_, _ = executor.FanOut(context.Background(), specs)

	branch, err := executor.GetBranch("branch-1")
	if err != nil {
		t.Fatalf("get branch failed: %v", err)
	}
	if branch.Name != "Branch 1" {
		t.Errorf("expected name 'Branch 1', got %s", branch.Name)
	}

	_, err = executor.GetBranch("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
}

func TestCancelAll(t *testing.T) {
	executor := NewParallelExecutor(&mockBranchExecutor{})
	specs := []BranchSpec{{ID: "branch-1", Name: "Branch 1"}}

	_, _ = executor.FanOut(context.Background(), specs)

	errs := executor.CancelAll(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected no cancel errors for pending branches, got %d", len(errs))
	}
}

func TestDefaultMergePolicy(t *testing.T) {
	policy := NewDefaultMergePolicy()

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value1"}},
		{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value2"}},
	}

	merged, err := policy.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if merged.Status != MergeStatusSuccess {
		t.Errorf("expected success, got %s", merged.Status)
	}
}

func TestDefaultMergePolicyWithConflict(t *testing.T) {
	policy := NewDefaultMergePolicy(WithConflictResolution(ConflictResolutionLast))

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value1"}},
		{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value2"}},
	}

	merged, err := policy.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	data, ok := merged.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected map data")
	}
	if data["key"] != "value2" {
		t.Errorf("expected value2 with last resolution, got %v", data["key"])
	}
	if len(merged.Conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(merged.Conflicts))
	}
}

func TestDefaultMergePolicyFailFast(t *testing.T) {
	policy := NewDefaultMergePolicy(WithFailFast(true))

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: "ok"},
		{BranchID: "branch-2", Status: BranchStatusFailed, Error: errors.New("failed")},
	}

	_, err := policy.Merge(context.Background(), results)
	if err == nil {
		t.Error("expected error with fail-fast enabled")
	}
}

func TestDefaultMergePolicyNoFailFast(t *testing.T) {
	policy := NewDefaultMergePolicy(WithFailFast(false))

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value"}},
		{BranchID: "branch-2", Status: BranchStatusFailed, Error: errors.New("failed")},
	}

	merged, err := policy.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("unexpected error with fail-fast disabled: %v", err)
	}
	if merged.Status != MergeStatusPartial {
		t.Errorf("expected partial status, got %s", merged.Status)
	}
}

func TestDefaultMergePolicyValidate(t *testing.T) {
	policy := NewDefaultMergePolicy()

	err := policy.ValidateMerge([]BranchResult{})
	if err == nil {
		t.Error("expected error for empty results")
	}

	err = policy.ValidateMerge([]BranchResult{
		{BranchID: "branch-1", Status: BranchStatusRunning},
	})
	if err == nil {
		t.Error("expected error for running branch")
	}
}

func TestConflictResolutionFirst(t *testing.T) {
	policy := NewDefaultMergePolicy(WithConflictResolution(ConflictResolutionFirst))

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "first"}},
		{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "second"}},
	}

	merged, err := policy.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	data := merged.Data.(map[string]interface{})
	if data["key"] != "first" {
		t.Errorf("expected 'first' with first resolution, got %v", data["key"])
	}
}

func TestConflictResolutionUnion(t *testing.T) {
	policy := NewDefaultMergePolicy(WithConflictResolution(ConflictResolutionUnion))

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"items": []interface{}{"a", "b"}}},
		{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"items": []interface{}{"b", "c"}}},
	}

	merged, err := policy.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	data := merged.Data.(map[string]interface{})
	items, ok := data["items"].([]interface{})
	if !ok {
		t.Fatal("expected items slice")
	}
	if len(items) != 3 {
		t.Errorf("expected 3 unique items with union, got %d", len(items))
	}
}

func TestIsolatedMergePolicy(t *testing.T) {
	inner := NewDefaultMergePolicy(WithFailFast(false))
	policy := NewIsolatedMergePolicy(inner)

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value"}},
		{BranchID: "branch-2", Status: BranchStatusFailed, Error: errors.New("failed")},
	}

	merged, err := policy.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	if len(merged.FailedBranch) != 1 {
		t.Errorf("expected 1 failed branch tracked, got %d", len(merged.FailedBranch))
	}
	if merged.Status != MergeStatusPartial {
		t.Errorf("expected partial status, got %s", merged.Status)
	}
}

func TestIsolatedMergePolicyAllFailed(t *testing.T) {
	inner := NewDefaultMergePolicy()
	policy := NewIsolatedMergePolicy(inner)

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusFailed, Error: errors.New("failed")},
		{BranchID: "branch-2", Status: BranchStatusFailed, Error: errors.New("failed")},
	}

	_, err := policy.Merge(context.Background(), results)
	if err == nil {
		t.Error("expected error when all branches fail")
	}
}

func TestPolicyConsistencyChecker(t *testing.T) {
	checker := NewPolicyConsistencyChecker()

	consistent := []BranchResult{
		{BranchID: "b1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"policy": "p1"}},
		{BranchID: "b2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"policy": "p1"}},
	}
	if err := checker.CheckConsistency(consistent); err != nil {
		t.Errorf("expected consistent policies to pass: %v", err)
	}

	inconsistent := []BranchResult{
		{BranchID: "b1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"policy": "p1"}},
		{BranchID: "b2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"policy": "p2"}},
	}
	if err := checker.CheckConsistency(inconsistent); err == nil {
		t.Error("expected error for inconsistent policies")
	}
}

func TestMergeResultLogging(t *testing.T) {
	policy := NewDefaultMergePolicy()

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value1"}},
		{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value2"}},
	}

	merged, err := policy.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	if merged.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
	if len(merged.Conflicts) == 0 {
		t.Error("expected conflicts to be logged for duplicate keys")
	}

	for _, c := range merged.Conflicts {
		if c.Field == "" {
			t.Error("expected conflict field to be set")
		}
		if c.Resolution == "" {
			t.Error("expected conflict resolution to be set")
		}
	}
}

func TestBranchStatusString(t *testing.T) {
	tests := []struct {
		status   BranchStatus
		expected string
	}{
		{BranchStatusPending, "pending"},
		{BranchStatusRunning, "running"},
		{BranchStatusSucceeded, "succeeded"},
		{BranchStatusFailed, "failed"},
		{BranchStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}
