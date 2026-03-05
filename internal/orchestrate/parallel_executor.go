package orchestrate

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type BranchID string

type BranchStatus string

const (
	BranchStatusPending   BranchStatus = "pending"
	BranchStatusRunning   BranchStatus = "running"
	BranchStatusSucceeded BranchStatus = "succeeded"
	BranchStatusFailed    BranchStatus = "failed"
	BranchStatusCancelled BranchStatus = "cancelled"
)

type Branch struct {
	ID          BranchID
	Name        string
	Status      BranchStatus
	StartedAt   *time.Time
	CompletedAt *time.Time
	Result      interface{}
	Error       error
	Metadata    map[string]interface{}
}

type BranchResult struct {
	BranchID BranchID
	Status   BranchStatus
	Result   interface{}
	Error    error
}

// BranchExecutor implementations must stop work when ctx is canceled.
// ExecuteWithTimeout relies on this contract; otherwise branch goroutines may
// outlive timeout boundaries.
type BranchExecutor interface {
	ExecuteBranch(ctx context.Context, branch *Branch) (interface{}, error)
	CancelBranch(ctx context.Context, branchID BranchID) error
}

type ParallelExecutor struct {
	mu          sync.RWMutex
	branches    map[BranchID]*Branch
	executor    BranchExecutor
	mergePolicy MergePolicy
	maxBranches int
	timeout     time.Duration
}

type ParallelExecutorOption func(*ParallelExecutor)

func WithMaxBranches(max int) ParallelExecutorOption {
	return func(p *ParallelExecutor) {
		p.maxBranches = max
	}
}

func WithTimeout(timeout time.Duration) ParallelExecutorOption {
	return func(p *ParallelExecutor) {
		p.timeout = timeout
	}
}

func WithMergePolicy(policy MergePolicy) ParallelExecutorOption {
	return func(p *ParallelExecutor) {
		p.mergePolicy = policy
	}
}

func NewParallelExecutor(executor BranchExecutor, opts ...ParallelExecutorOption) *ParallelExecutor {
	p := &ParallelExecutor{
		branches:    make(map[BranchID]*Branch),
		executor:    executor,
		maxBranches: 10,
		timeout:     30 * time.Minute,
		mergePolicy: &DefaultMergePolicy{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *ParallelExecutor) FanOut(ctx context.Context, specs []BranchSpec) ([]BranchID, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(specs) > p.maxBranches {
		return nil, fmt.Errorf("exceeded max branches: %d > %d", len(specs), p.maxBranches)
	}

	if len(p.branches)+len(specs) > p.maxBranches {
		return nil, fmt.Errorf("would exceed max branches after fan-out")
	}

	var branchIDs []BranchID
	now := time.Now()

	for _, spec := range specs {
		branch := &Branch{
			ID:        spec.ID,
			Name:      spec.Name,
			Status:    BranchStatusPending,
			StartedAt: &now,
			Metadata:  spec.Metadata,
		}
		p.branches[branch.ID] = branch
		branchIDs = append(branchIDs, branch.ID)
	}

	return branchIDs, nil
}

type BranchSpec struct {
	ID       BranchID
	Name     string
	Metadata map[string]interface{}
}

func (p *ParallelExecutor) Execute(ctx context.Context, branchIDs []BranchID) <-chan BranchResult {
	resultChan := make(chan BranchResult, len(branchIDs))

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		for _, id := range branchIDs {
			wg.Add(1)
			go func(branchID BranchID) {
				defer wg.Done()

				p.mu.RLock()
				branch, exists := p.branches[branchID]
				p.mu.RUnlock()

				if !exists {
					resultChan <- BranchResult{
						BranchID: branchID,
						Status:   BranchStatusFailed,
						Error:    fmt.Errorf("branch %s not found", branchID),
					}
					return
				}

				p.mu.Lock()
				branch.Status = BranchStatusRunning
				p.mu.Unlock()

				result, err := p.executeWithTimeout(ctx, branch)

				p.mu.Lock()
				if err != nil {
					branch.Status = BranchStatusFailed
					branch.Error = err
				} else {
					branch.Status = BranchStatusSucceeded
					branch.Result = result
				}
				now := time.Now()
				branch.CompletedAt = &now
				p.mu.Unlock()

				resultChan <- BranchResult{
					BranchID: branchID,
					Status:   branch.Status,
					Result:   result,
					Error:    err,
				}
			}(id)
		}
		wg.Wait()
	}()

	return resultChan
}

func (p *ParallelExecutor) executeWithTimeout(ctx context.Context, branch *Branch) (interface{}, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := p.executor.ExecuteBranch(timeoutCtx, branch)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("branch %s timed out", branch.ID)
	}
}

func (p *ParallelExecutor) FanIn(ctx context.Context, branchIDs []BranchID) (*MergeResult, error) {
	results := p.collectResults(branchIDs)

	if p.mergePolicy == nil {
		return nil, fmt.Errorf("no merge policy configured")
	}

	return p.mergePolicy.Merge(ctx, results)
}

func (p *ParallelExecutor) collectResults(branchIDs []BranchID) []BranchResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var results []BranchResult
	for _, id := range branchIDs {
		branch, exists := p.branches[id]
		if !exists {
			results = append(results, BranchResult{
				BranchID: id,
				Status:   BranchStatusFailed,
				Error:    fmt.Errorf("branch %s not found", id),
			})
			continue
		}
		results = append(results, BranchResult{
			BranchID: id,
			Status:   branch.Status,
			Result:   branch.Result,
			Error:    branch.Error,
		})
	}
	return results
}

func (p *ParallelExecutor) GetBranch(branchID BranchID) (*Branch, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	branch, exists := p.branches[branchID]
	if !exists {
		return nil, fmt.Errorf("branch %s not found", branchID)
	}
	return branch, nil
}

func (p *ParallelExecutor) CancelAll(ctx context.Context) []error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for id, branch := range p.branches {
		if branch.Status == BranchStatusRunning {
			if err := p.executor.CancelBranch(ctx, id); err != nil {
				errs = append(errs, err)
			} else {
				branch.Status = BranchStatusCancelled
				now := time.Now()
				branch.CompletedAt = &now
			}
		}
	}
	return errs
}

func (p *ParallelExecutor) GetStatus() *ParallelExecutorStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var pending, running, succeeded, failed, cancelled int
	for _, branch := range p.branches {
		switch branch.Status {
		case BranchStatusPending:
			pending++
		case BranchStatusRunning:
			running++
		case BranchStatusSucceeded:
			succeeded++
		case BranchStatusFailed:
			failed++
		case BranchStatusCancelled:
			cancelled++
		}
	}

	return &ParallelExecutorStatus{
		TotalBranches: len(p.branches),
		Pending:       pending,
		Running:       running,
		Succeeded:     succeeded,
		Failed:        failed,
		Cancelled:     cancelled,
	}
}

type ParallelExecutorStatus struct {
	TotalBranches int
	Pending       int
	Running       int
	Succeeded     int
	Failed        int
	Cancelled     int
}
