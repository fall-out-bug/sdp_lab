package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type VerifierID string

type Verdict string

const (
	VerdictApprove Verdict = "approve"
	VerdictReject  Verdict = "reject"
	VerdictAbstain Verdict = "abstain"
)

type VerifierRole string

const (
	VerifierRoleQA       VerifierRole = "qa"
	VerifierRoleSecurity VerifierRole = "security"
	VerifierRolePolicy   VerifierRole = "policy"
	VerifierRoleCustom   VerifierRole = "custom"
)

type VerifierResult struct {
	VerifierID VerifierID
	Role       VerifierRole
	Verdict    Verdict
	Reason     string
	Details    map[string]interface{}
	Timestamp  time.Time
	Duration   time.Duration
	Confidence float64
}

type QuorumPolicy struct {
	RequiredRoles   []VerifierRole
	MinApprovals    int
	RejectThreshold int
	RequireAll      bool
	Timeout         time.Duration
}

type QuorumVerdict struct {
	Passed         bool
	TotalVerifiers int
	Approvals      int
	Rejections     int
	Abstentions    int
	Results        []VerifierResult
	Dissenting     []VerifierResult
	Timestamp      time.Time
	PolicyApplied  QuorumPolicy
}

func (v *QuorumVerdict) ToEvidence() map[string]interface{} {
	return map[string]interface{}{
		"passed":          v.Passed,
		"total_verifiers": v.TotalVerifiers,
		"approvals":       v.Approvals,
		"rejections":      v.Rejections,
		"abstentions":     v.Abstentions,
		"timestamp":       v.Timestamp.Format(time.RFC3339),
		"policy": map[string]interface{}{
			"required_roles":   v.PolicyApplied.RequiredRoles,
			"min_approvals":    v.PolicyApplied.MinApprovals,
			"reject_threshold": v.PolicyApplied.RejectThreshold,
			"require_all":      v.PolicyApplied.RequireAll,
		},
		"dissenting": v.dissentingToEvidence(),
	}
}

func (v *QuorumVerdict) dissentingToEvidence() []map[string]interface{} {
	var dissenting []map[string]interface{}
	for _, d := range v.Dissenting {
		dissenting = append(dissenting, map[string]interface{}{
			"verifier_id": string(d.VerifierID),
			"role":        string(d.Role),
			"verdict":     string(d.Verdict),
			"reason":      d.Reason,
		})
	}
	return dissenting
}

type Verifier interface {
	Verify(ctx context.Context, input interface{}) (*VerifierResult, error)
	ID() VerifierID
	Role() VerifierRole
}

type Quorum struct {
	mu        sync.RWMutex
	verifiers map[VerifierID]Verifier
	policy    QuorumPolicy
	promoter  PromotionGate
}

type PromotionGate interface {
	CanPromote(ctx context.Context, verdict *QuorumVerdict) (bool, string)
}

type QuorumOption func(*Quorum)

func WithPolicy(policy QuorumPolicy) QuorumOption {
	return func(q *Quorum) {
		q.policy = policy
	}
}

func WithPromoter(promoter PromotionGate) QuorumOption {
	return func(q *Quorum) {
		q.promoter = promoter
	}
}

func NewQuorum(opts ...QuorumOption) *Quorum {
	q := &Quorum{
		verifiers: make(map[VerifierID]Verifier),
		policy: QuorumPolicy{
			RequiredRoles:   []VerifierRole{VerifierRoleQA, VerifierRoleSecurity},
			MinApprovals:    2,
			RejectThreshold: 1,
			RequireAll:      false,
			Timeout:         5 * time.Minute,
		},
	}
	for _, opt := range opts {
		opt(q)
	}
	return q
}

func (q *Quorum) RegisterVerifier(v Verifier) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if v.ID() == "" {
		return fmt.Errorf("verifier ID cannot be empty")
	}

	if _, exists := q.verifiers[v.ID()]; exists {
		return fmt.Errorf("verifier %s already registered", v.ID())
	}

	q.verifiers[v.ID()] = v
	return nil
}

func (q *Quorum) UnregisterVerifier(id VerifierID) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.verifiers, id)
}

func (q *Quorum) Execute(ctx context.Context, input interface{}) (*QuorumVerdict, error) {
	q.mu.RLock()
	if len(q.verifiers) == 0 {
		q.mu.RUnlock()
		return nil, fmt.Errorf("no verifiers registered")
	}
	policy := q.policy
	verifiers := make([]Verifier, 0, len(q.verifiers))
	for _, v := range q.verifiers {
		verifiers = append(verifiers, v)
	}
	q.mu.RUnlock()

	timeout := policy.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan *VerifierResult, len(verifiers))
	errorChan := make(chan error, len(verifiers))

	var wg sync.WaitGroup
	for _, v := range verifiers {
		wg.Add(1)
		go func(verifier Verifier) {
			defer wg.Done()
			result, err := verifier.Verify(timeoutCtx, input)
			if err != nil {
				errorChan <- fmt.Errorf("verifier %s failed: %w", verifier.ID(), err)
				return
			}
			resultChan <- result
		}(v)
	}

	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	var results []VerifierResult
	var errors []error

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				resultChan = nil
			} else {
				results = append(results, *result)
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else {
				errors = append(errors, err)
			}
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("quorum verification timed out")
		}

		if resultChan == nil && errorChan == nil {
			break
		}
	}

	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("all verifiers failed: %v", errors)
	}

	return q.evaluateVerdict(results, len(verifiers), policy), nil
}

func (q *Quorum) evaluateVerdict(results []VerifierResult, totalVerifiers int, policy QuorumPolicy) *QuorumVerdict {
	verdict := &QuorumVerdict{
		TotalVerifiers: totalVerifiers,
		Results:        results,
		Timestamp:      time.Now(),
		PolicyApplied:  policy,
	}

	rolesPresent := make(map[VerifierRole]bool)
	for _, r := range results {
		rolesPresent[r.Role] = true

		switch r.Verdict {
		case VerdictApprove:
			verdict.Approvals++
		case VerdictReject:
			verdict.Rejections++
			verdict.Dissenting = append(verdict.Dissenting, r)
		case VerdictAbstain:
			verdict.Abstentions++
		}
	}

	for _, role := range policy.RequiredRoles {
		if !rolesPresent[role] {
			verdict.Dissenting = append(verdict.Dissenting, VerifierResult{
				Role:    role,
				Verdict: VerdictReject,
				Reason:  fmt.Sprintf("required role %s not present", role),
			})
		}
	}

	verdict.Passed = q.evaluatePolicy(verdict, policy)

	return verdict
}

func (q *Quorum) evaluatePolicy(verdict *QuorumVerdict, policy QuorumPolicy) bool {
	if verdict.Rejections >= policy.RejectThreshold {
		return false
	}

	if verdict.Approvals < policy.MinApprovals {
		return false
	}

	rolesPresent := make(map[VerifierRole]bool)
	for _, r := range verdict.Results {
		if r.Verdict == VerdictApprove {
			rolesPresent[r.Role] = true
		}
	}

	for _, role := range policy.RequiredRoles {
		if !rolesPresent[role] {
			if policy.RequireAll {
				return false
			}
		}
	}

	return true
}

func (q *Quorum) CanPromote(ctx context.Context, verdict *QuorumVerdict) (bool, string) {
	if !verdict.Passed {
		return false, "quorum verdict did not pass"
	}

	if q.promoter != nil {
		return q.promoter.CanPromote(ctx, verdict)
	}

	return true, "quorum passed, promotion allowed"
}

func (q *Quorum) GetPolicy() QuorumPolicy {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.policy
}

func (q *Quorum) SetPolicy(policy QuorumPolicy) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.policy = policy
}

func (q *Quorum) GetVerifiers() []VerifierID {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var ids []VerifierID
	for id := range q.verifiers {
		ids = append(ids, id)
	}
	return ids
}

func (v *VerifierResult) ToJSON() string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func DefaultQAPolicy() QuorumPolicy {
	return QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA},
		MinApprovals:    1,
		RejectThreshold: 1,
		RequireAll:      false,
		Timeout:         5 * time.Minute,
	}
}

func DefaultSecurityPolicy() QuorumPolicy {
	return QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA, VerifierRoleSecurity},
		MinApprovals:    2,
		RejectThreshold: 1,
		RequireAll:      true,
		Timeout:         10 * time.Minute,
	}
}

func DefaultReleasePolicy() QuorumPolicy {
	return QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA, VerifierRoleSecurity, VerifierRolePolicy},
		MinApprovals:    3,
		RejectThreshold: 1,
		RequireAll:      true,
		Timeout:         15 * time.Minute,
	}
}
