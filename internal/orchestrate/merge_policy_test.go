package orchestrate

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewDefaultMergePolicy(t *testing.T) {
	tests := []struct {
		name string
		opts []MergePolicyOption
		want ConflictResolution
	}{
		{
			name: "default resolution",
			opts: nil,
			want: ConflictResolutionLast,
		},
		{
			name: "first resolution",
			opts: []MergePolicyOption{WithConflictResolution(ConflictResolutionFirst)},
			want: ConflictResolutionFirst,
		},
		{
			name: "union resolution",
			opts: []MergePolicyOption{WithConflictResolution(ConflictResolutionUnion)},
			want: ConflictResolutionUnion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewDefaultMergePolicy(tt.opts...)
			if p == nil {
				t.Fatal("NewDefaultMergePolicy returned nil")
				return
			}
			if p.resolution != tt.want {
				t.Errorf("resolution = %v, want %v", p.resolution, tt.want)
			}
		})
	}
}

func TestDefaultMergePolicy_WithOptions(t *testing.T) {
	p := NewDefaultMergePolicy(
		WithConflictResolution(ConflictResolutionFirst),
		WithFailFast(false),
	)

	if p.resolution != ConflictResolutionFirst {
		t.Errorf("resolution = %v, want %v", p.resolution, ConflictResolutionFirst)
	}
	if p.failFast {
		t.Error("failFast should be false")
	}
}

func TestDefaultMergePolicy_Merge_Success(t *testing.T) {
	p := NewDefaultMergePolicy()
	ctx := context.Background()

	results := []BranchResult{
		{
			BranchID: "branch-1",
			Status:   BranchStatusSucceeded,
			Result:   map[string]interface{}{"key": "value1"},
		},
		{
			BranchID: "branch-2",
			Status:   BranchStatusSucceeded,
			Result:   map[string]interface{}{"key2": "value2"},
		},
	}

	merged, err := p.Merge(ctx, results)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	if merged.Status != MergeStatusSuccess {
		t.Errorf("Status = %v, want %v", merged.Status, MergeStatusSuccess)
	}
	if len(merged.MergedFrom) != 2 {
		t.Errorf("MergedFrom length = %d, want 2", len(merged.MergedFrom))
	}
}

func TestDefaultMergePolicy_Merge_FailFast(t *testing.T) {
	p := NewDefaultMergePolicy(WithFailFast(true))
	ctx := context.Background()

	results := []BranchResult{
		{
			BranchID: "branch-1",
			Status:   BranchStatusSucceeded,
			Result:   map[string]interface{}{"key": "value1"},
		},
		{
			BranchID: "branch-2",
			Status:   BranchStatusFailed,
			Error:    errors.New("branch failed"),
		},
	}

	_, err := p.Merge(ctx, results)
	if err == nil {
		t.Error("expected error for failed branch with failFast=true")
	}
}

func TestDefaultMergePolicy_Merge_NoFailFast(t *testing.T) {
	p := NewDefaultMergePolicy(WithFailFast(false))
	ctx := context.Background()

	results := []BranchResult{
		{
			BranchID: "branch-1",
			Status:   BranchStatusSucceeded,
			Result:   map[string]interface{}{"key": "value1"},
		},
		{
			BranchID: "branch-2",
			Status:   BranchStatusFailed,
			Error:    errors.New("branch failed"),
		},
	}

	merged, err := p.Merge(ctx, results)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	if merged.Status != MergeStatusPartial {
		t.Errorf("Status = %v, want %v", merged.Status, MergeStatusPartial)
	}
}

func TestDefaultMergePolicy_Merge_NoResults(t *testing.T) {
	p := NewDefaultMergePolicy()
	ctx := context.Background()

	_, err := p.Merge(ctx, nil)
	if err == nil {
		t.Error("expected error for no results")
	}
}

func TestDefaultMergePolicy_Merge_AllFailed(t *testing.T) {
	p := NewDefaultMergePolicy(WithFailFast(false))
	ctx := context.Background()

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusFailed, Error: errors.New("failed")},
		{BranchID: "branch-2", Status: BranchStatusFailed, Error: errors.New("failed")},
	}

	_, err := p.Merge(ctx, results)
	if err == nil {
		t.Error("expected error when all branches fail")
	}
}

func TestDefaultMergePolicy_Merge_RunningBranch(t *testing.T) {
	p := NewDefaultMergePolicy()
	ctx := context.Background()

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusRunning},
	}

	_, err := p.Merge(ctx, results)
	if err == nil {
		t.Error("expected error for running branch")
	}
}

func TestDefaultMergePolicy_ValidateMerge(t *testing.T) {
	p := NewDefaultMergePolicy()

	tests := []struct {
		name    string
		results []BranchResult
		wantErr bool
	}{
		{
			name:    "no results",
			results: nil,
			wantErr: true,
		},
		{
			name: "valid results",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded},
			},
			wantErr: false,
		},
		{
			name: "running branch",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusRunning},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.ValidateMerge(tt.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMerge() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultMergePolicy_ConflictResolution(t *testing.T) {
	tests := []struct {
		name       string
		resolution ConflictResolution
		results    []BranchResult
		wantValue  interface{}
	}{
		{
			name:       "first wins",
			resolution: ConflictResolutionFirst,
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value1"}},
				{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value2"}},
			},
			wantValue: "value1",
		},
		{
			name:       "last wins",
			resolution: ConflictResolutionLast,
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value1"}},
				{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value2"}},
			},
			wantValue: "value2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewDefaultMergePolicy(WithConflictResolution(tt.resolution))
			merged, err := p.Merge(context.Background(), tt.results)
			if err != nil {
				t.Fatalf("Merge failed: %v", err)
			}

			data := merged.Data.(map[string]interface{})
			if data["key"] != tt.wantValue {
				t.Errorf("key = %v, want %v", data["key"], tt.wantValue)
			}
		})
	}
}

func TestDefaultMergePolicy_UnionResolution(t *testing.T) {
	p := NewDefaultMergePolicy(WithConflictResolution(ConflictResolutionUnion))
	ctx := context.Background()

	results := []BranchResult{
		{
			BranchID: "branch-1",
			Status:   BranchStatusSucceeded,
			Result:   map[string]interface{}{"items": []interface{}{"a", "b"}},
		},
		{
			BranchID: "branch-2",
			Status:   BranchStatusSucceeded,
			Result:   map[string]interface{}{"items": []interface{}{"b", "c"}},
		},
	}

	merged, err := p.Merge(ctx, results)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	data := merged.Data.(map[string]interface{})
	items, ok := data["items"].([]interface{})
	if !ok {
		t.Fatal("items is not a slice")
	}
	if len(items) != 3 {
		t.Errorf("items length = %d, want 3", len(items))
	}
}

func TestPolicyConsistencyChecker(t *testing.T) {
	checker := NewPolicyConsistencyChecker()

	tests := []struct {
		name    string
		results []BranchResult
		wantErr bool
	}{
		{
			name:    "no results",
			results: nil,
			wantErr: false,
		},
		{
			name: "consistent policies",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"policy": "policy-a"}},
				{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"policy": "policy-a"}},
			},
			wantErr: false,
		},
		{
			name: "inconsistent policies",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"policy": "policy-a"}},
				{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"policy": "policy-b"}},
			},
			wantErr: true,
		},
		{
			name: "nil result",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: nil},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.CheckConsistency(tt.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckConsistency() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsolatedMergePolicy(t *testing.T) {
	innerPolicy := NewDefaultMergePolicy()
	p := NewIsolatedMergePolicy(innerPolicy)
	ctx := context.Background()

	tests := []struct {
		name    string
		results []BranchResult
		want    MergeStatus
		wantErr bool
	}{
		{
			name: "all succeeded",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value"}},
				{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key2": "value2"}},
			},
			want:    MergeStatusSuccess,
			wantErr: false,
		},
		{
			name: "partial success",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value"}},
				{BranchID: "branch-2", Status: BranchStatusFailed, Error: errors.New("failed")},
			},
			want:    MergeStatusPartial,
			wantErr: false,
		},
		{
			name: "all failed",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusFailed, Error: errors.New("failed")},
				{BranchID: "branch-2", Status: BranchStatusFailed, Error: errors.New("failed")},
			},
			want:    MergeStatusFailed,
			wantErr: true,
		},
		{
			name: "branch with error",
			results: []BranchResult{
				{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: nil, Error: errors.New("error")},
				{BranchID: "branch-2", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value"}},
			},
			want:    MergeStatusPartial,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := p.Merge(ctx, tt.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("Merge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && merged.Status != tt.want {
				t.Errorf("Status = %v, want %v", merged.Status, tt.want)
			}
		})
	}
}

func TestIsolatedMergePolicy_ValidateMerge(t *testing.T) {
	innerPolicy := NewDefaultMergePolicy()
	p := NewIsolatedMergePolicy(innerPolicy)

	// IsolatedMergePolicy should delegate to inner policy
	err := p.ValidateMerge(nil)
	if err == nil {
		t.Error("expected error for nil results")
	}
}

func TestExtractBranchIDs(t *testing.T) {
	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded},
		{BranchID: "branch-2", Status: BranchStatusFailed},
		{BranchID: "branch-3", Status: BranchStatusSucceeded},
	}

	ids := extractBranchIDs(results)
	if len(ids) != 3 {
		t.Errorf("extractBranchIDs length = %d, want 3", len(ids))
	}
	expectedIDs := []BranchID{"branch-1", "branch-2", "branch-3"}
	for i, id := range ids {
		if id != expectedIDs[i] {
			t.Errorf("ids[%d] = %v, want %v", i, id, expectedIDs[i])
		}
	}
}

func TestDefaultMergePolicy_WithScopeChecker(t *testing.T) {
	scopeErr := errors.New("scope check failed")
	checker := &mockScopeChecker{err: scopeErr}

	p := NewDefaultMergePolicy(WithScopeChecker(checker))
	ctx := context.Background()

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded},
	}

	_, err := p.Merge(ctx, results)
	if err == nil {
		t.Error("expected scope check error")
	}
}

// mockScopeChecker for testing
type mockScopeChecker struct {
	err error
}

func (m *mockScopeChecker) CheckScopes(results []BranchResult) error {
	return m.err
}

func TestMergeResult_Timestamp(t *testing.T) {
	p := NewDefaultMergePolicy()
	ctx := context.Background()

	results := []BranchResult{
		{BranchID: "branch-1", Status: BranchStatusSucceeded, Result: map[string]interface{}{"key": "value"}},
	}

	before := time.Now()
	merged, err := p.Merge(ctx, results)
	after := time.Now()

	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if merged.Timestamp.Before(before) || merged.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, expected between %v and %v", merged.Timestamp, before, after)
	}
}
