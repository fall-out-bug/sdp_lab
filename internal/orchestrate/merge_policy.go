package orchestrate

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type ConflictResolution string

const (
	ConflictResolutionFirst     ConflictResolution = "first"
	ConflictResolutionLast      ConflictResolution = "last"
	ConflictResolutionUnion     ConflictResolution = "union"
	ConflictResolutionFail      ConflictResolution = "fail"
	ConflictResolutionTimestamp ConflictResolution = "timestamp"
)

type MergeResult struct {
	Status       MergeStatus
	Data         interface{}
	Conflicts    []Conflict
	MergedFrom   []BranchID
	Timestamp    time.Time
	FailedBranch []BranchID
}

type MergeStatus string

const (
	MergeStatusSuccess MergeStatus = "success"
	MergeStatusPartial MergeStatus = "partial"
	MergeStatusFailed  MergeStatus = "failed"
)

type Conflict struct {
	Field      string
	Values     []interface{}
	Resolution ConflictResolution
	Resolved   interface{}
}

type MergePolicy interface {
	Merge(ctx context.Context, results []BranchResult) (*MergeResult, error)
	ValidateMerge(results []BranchResult) error
}

type DefaultMergePolicy struct {
	resolution ConflictResolution
	failFast   bool
	scopeCheck ScopeChecker
}

type ScopeChecker interface {
	CheckScopes(results []BranchResult) error
}

type MergePolicyOption func(*DefaultMergePolicy)

func WithConflictResolution(res ConflictResolution) MergePolicyOption {
	return func(p *DefaultMergePolicy) {
		p.resolution = res
	}
}

func WithFailFast(failFast bool) MergePolicyOption {
	return func(p *DefaultMergePolicy) {
		p.failFast = failFast
	}
}

func WithScopeChecker(checker ScopeChecker) MergePolicyOption {
	return func(p *DefaultMergePolicy) {
		p.scopeCheck = checker
	}
}

func NewDefaultMergePolicy(opts ...MergePolicyOption) *DefaultMergePolicy {
	p := &DefaultMergePolicy{
		resolution: ConflictResolutionLast,
		failFast:   true,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *DefaultMergePolicy) Merge(ctx context.Context, results []BranchResult) (*MergeResult, error) {
	if err := p.ValidateMerge(results); err != nil {
		return nil, err
	}

	if p.scopeCheck != nil {
		if err := p.scopeCheck.CheckScopes(results); err != nil {
			return nil, fmt.Errorf("scope check failed: %w", err)
		}
	}

	var succeeded, failed []BranchResult
	for _, r := range results {
		if r.Status == BranchStatusSucceeded {
			succeeded = append(succeeded, r)
		} else {
			failed = append(failed, r)
		}
	}

	if len(failed) > 0 && p.failFast {
		return &MergeResult{
			Status:       MergeStatusFailed,
			FailedBranch: extractBranchIDs(failed),
			Timestamp:    time.Now(),
		}, fmt.Errorf("merge aborted: %d branches failed", len(failed))
	}

	if len(succeeded) == 0 {
		return &MergeResult{
			Status:       MergeStatusFailed,
			FailedBranch: extractBranchIDs(failed),
			Timestamp:    time.Now(),
		}, fmt.Errorf("no successful branches to merge")
	}

	merged, conflicts, err := p.mergeData(succeeded)
	if err != nil {
		return &MergeResult{
			Status:       MergeStatusFailed,
			FailedBranch: extractBranchIDs(failed),
			Timestamp:    time.Now(),
		}, fmt.Errorf("merge failed: %w", err)
	}

	status := MergeStatusSuccess
	if len(failed) > 0 {
		status = MergeStatusPartial
	}

	return &MergeResult{
		Status:       status,
		Data:         merged,
		Conflicts:    conflicts,
		MergedFrom:   extractBranchIDs(succeeded),
		FailedBranch: extractBranchIDs(failed),
		Timestamp:    time.Now(),
	}, nil
}

func (p *DefaultMergePolicy) ValidateMerge(results []BranchResult) error {
	if len(results) == 0 {
		return fmt.Errorf("no results to merge")
	}

	for _, r := range results {
		if r.Status == BranchStatusRunning {
			return fmt.Errorf("branch %s is still running", r.BranchID)
		}
	}

	return nil
}

func (p *DefaultMergePolicy) mergeData(results []BranchResult) (interface{}, []Conflict, error) {
	var conflicts []Conflict

	sort.Slice(results, func(i, j int) bool {
		return string(results[i].BranchID) < string(results[j].BranchID)
	})

	var mergedData map[string]interface{}
	for _, r := range results {
		dataMap, ok := r.Result.(map[string]interface{})
		if !ok {
			bytes, err := json.Marshal(r.Result)
			if err != nil {
				continue
			}
			dataMap = make(map[string]interface{})
			if err := json.Unmarshal(bytes, &dataMap); err != nil {
				continue
			}
		}

		if mergedData == nil {
			mergedData = make(map[string]interface{})
		}

		for k, v := range dataMap {
			if existing, exists := mergedData[k]; exists {
				conflict := Conflict{
					Field:      k,
					Values:     []interface{}{existing, v},
					Resolution: p.resolution,
				}
				conflict.Resolved = p.resolveConflict(existing, v)
				mergedData[k] = conflict.Resolved
				conflicts = append(conflicts, conflict)
			} else {
				mergedData[k] = v
			}
		}
	}

	return mergedData, conflicts, nil
}

func (p *DefaultMergePolicy) resolveConflict(existing, new interface{}) interface{} {
	switch p.resolution {
	case ConflictResolutionFirst:
		return existing
	case ConflictResolutionLast:
		return new
	case ConflictResolutionUnion:
		return p.unionValues(existing, new)
	case ConflictResolutionTimestamp:
		return new
	default:
		return new
	}
}

func (p *DefaultMergePolicy) unionValues(a, b interface{}) interface{} {
	aSlice, aOk := a.([]interface{})
	bSlice, bOk := b.([]interface{})

	if aOk && bOk {
		seen := make(map[interface{}]bool)
		var result []interface{}
		for _, v := range aSlice {
			if !seen[v] {
				seen[v] = true
				result = append(result, v)
			}
		}
		for _, v := range bSlice {
			if !seen[v] {
				seen[v] = true
				result = append(result, v)
			}
		}
		return result
	}

	return []interface{}{a, b}
}

func extractBranchIDs(results []BranchResult) []BranchID {
	var ids []BranchID
	for _, r := range results {
		ids = append(ids, r.BranchID)
	}
	return ids
}

type PolicyConsistencyChecker struct{}

func NewPolicyConsistencyChecker() *PolicyConsistencyChecker {
	return &PolicyConsistencyChecker{}
}

func (c *PolicyConsistencyChecker) CheckConsistency(results []BranchResult) error {
	if len(results) == 0 {
		return nil
	}

	var policies []string
	for _, r := range results {
		if r.Result == nil {
			continue
		}
		if dataMap, ok := r.Result.(map[string]interface{}); ok {
			if policy, exists := dataMap["policy"]; exists {
				if policyStr, ok := policy.(string); ok {
					policies = append(policies, policyStr)
				}
			}
		}
	}

	if len(policies) == 0 {
		return nil
	}

	first := policies[0]
	for _, p := range policies[1:] {
		if p != first {
			return fmt.Errorf("inconsistent policies across branches: %s vs %s", first, p)
		}
	}

	return nil
}

func (c *PolicyConsistencyChecker) CheckScopes(results []BranchResult) error {
	return c.CheckConsistency(results)
}

type IsolatedMergePolicy struct {
	inner       MergePolicy
	contaminate bool
}

func NewIsolatedMergePolicy(inner MergePolicy) *IsolatedMergePolicy {
	return &IsolatedMergePolicy{
		inner:       inner,
		contaminate: false,
	}
}

func (p *IsolatedMergePolicy) Merge(ctx context.Context, results []BranchResult) (*MergeResult, error) {
	var cleanResults []BranchResult
	var failedBranches []BranchID

	for _, r := range results {
		if r.Status == BranchStatusSucceeded && r.Error == nil {
			cleanResults = append(cleanResults, r)
		} else {
			failedBranches = append(failedBranches, r.BranchID)
		}
	}

	if len(cleanResults) == 0 {
		return &MergeResult{
			Status:       MergeStatusFailed,
			FailedBranch: failedBranches,
			Timestamp:    time.Now(),
		}, fmt.Errorf("all branches failed - no contamination possible")
	}

	mergeResult, err := p.inner.Merge(ctx, cleanResults)
	if err != nil {
		return mergeResult, err
	}

	if len(failedBranches) > 0 {
		mergeResult.FailedBranch = append(mergeResult.FailedBranch, failedBranches...)
		if mergeResult.Status == MergeStatusSuccess {
			mergeResult.Status = MergeStatusPartial
		}
	}

	return mergeResult, nil
}

func (p *IsolatedMergePolicy) ValidateMerge(results []BranchResult) error {
	return p.inner.ValidateMerge(results)
}
