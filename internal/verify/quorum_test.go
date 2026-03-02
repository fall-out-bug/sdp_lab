package verify

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockVerifier struct {
	id      VerifierID
	role    VerifierRole
	verdict Verdict
	reason  string
	err     error
	delay   time.Duration
}

func (m *mockVerifier) Verify(ctx context.Context, input interface{}) (*VerifierResult, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.err != nil {
		return nil, m.err
	}
	return &VerifierResult{
		VerifierID: m.id,
		Role:       m.role,
		Verdict:    m.verdict,
		Reason:     m.reason,
		Timestamp:  time.Now(),
	}, nil
}

func (m *mockVerifier) ID() VerifierID     { return m.id }
func (m *mockVerifier) Role() VerifierRole { return m.role }

func TestNewQuorum(t *testing.T) {
	q := NewQuorum()
	if q == nil {
		t.Fatal("expected non-nil quorum")
	}
	if len(q.GetVerifiers()) != 0 {
		t.Error("expected empty verifiers")
	}
}

func TestQuorumWithPolicy(t *testing.T) {
	policy := QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA},
		MinApprovals:    1,
		RejectThreshold: 1,
	}
	q := NewQuorum(WithPolicy(policy))

	got := q.GetPolicy()
	if len(got.RequiredRoles) != 1 {
		t.Errorf("expected 1 required role, got %d", len(got.RequiredRoles))
	}
}

func TestRegisterVerifier(t *testing.T) {
	q := NewQuorum()
	v := &mockVerifier{id: "v1", role: VerifierRoleQA}

	err := q.RegisterVerifier(v)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	verifiers := q.GetVerifiers()
	if len(verifiers) != 1 {
		t.Errorf("expected 1 verifier, got %d", len(verifiers))
	}
}

func TestRegisterVerifierEmptyID(t *testing.T) {
	q := NewQuorum()
	v := &mockVerifier{id: "", role: VerifierRoleQA}

	err := q.RegisterVerifier(v)
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestRegisterVerifierDuplicate(t *testing.T) {
	q := NewQuorum()
	v1 := &mockVerifier{id: "v1", role: VerifierRoleQA}
	v2 := &mockVerifier{id: "v1", role: VerifierRoleSecurity}

	_ = q.RegisterVerifier(v1)
	err := q.RegisterVerifier(v2)
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}

func TestUnregisterVerifier(t *testing.T) {
	q := NewQuorum()
	v := &mockVerifier{id: "v1", role: VerifierRoleQA}

	_ = q.RegisterVerifier(v)
	q.UnregisterVerifier("v1")

	verifiers := q.GetVerifiers()
	if len(verifiers) != 0 {
		t.Errorf("expected 0 verifiers, got %d", len(verifiers))
	}
}

func TestExecuteNoVerifiers(t *testing.T) {
	q := NewQuorum()

	_, err := q.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for no verifiers")
	}
}

func TestExecuteSingleApprove(t *testing.T) {
	q := NewQuorum(WithPolicy(DefaultQAPolicy()))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictApprove})

	verdict, err := q.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !verdict.Passed {
		t.Error("expected verdict to pass")
	}
	if verdict.Approvals != 1 {
		t.Errorf("expected 1 approval, got %d", verdict.Approvals)
	}
}

func TestExecuteSingleReject(t *testing.T) {
	q := NewQuorum(WithPolicy(DefaultQAPolicy()))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictReject})

	verdict, err := q.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if verdict.Passed {
		t.Error("expected verdict to fail")
	}
	if verdict.Rejections != 1 {
		t.Errorf("expected 1 rejection, got %d", verdict.Rejections)
	}
}

func TestExecuteMultipleVerifiers(t *testing.T) {
	q := NewQuorum(WithPolicy(QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA, VerifierRoleSecurity},
		MinApprovals:    2,
		RejectThreshold: 1,
	}))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictApprove})
	_ = q.RegisterVerifier(&mockVerifier{id: "sec1", role: VerifierRoleSecurity, verdict: VerdictApprove})

	verdict, err := q.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !verdict.Passed {
		t.Error("expected verdict to pass")
	}
	if verdict.Approvals != 2 {
		t.Errorf("expected 2 approvals, got %d", verdict.Approvals)
	}
}

func TestExecuteRejectThreshold(t *testing.T) {
	q := NewQuorum(WithPolicy(QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA, VerifierRoleSecurity},
		MinApprovals:    2,
		RejectThreshold: 1,
	}))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictApprove})
	_ = q.RegisterVerifier(&mockVerifier{id: "sec1", role: VerifierRoleSecurity, verdict: VerdictReject})

	verdict, err := q.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if verdict.Passed {
		t.Error("expected verdict to fail due to reject threshold")
	}
}

func TestExecuteMissingRequiredRole(t *testing.T) {
	q := NewQuorum(WithPolicy(QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA, VerifierRoleSecurity},
		MinApprovals:    2,
		RejectThreshold: 2,
		RequireAll:      true,
	}))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictApprove})

	verdict, err := q.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if verdict.Passed {
		t.Error("expected verdict to fail due to missing required role")
	}
}

func TestExecuteVerifierError(t *testing.T) {
	q := NewQuorum(WithPolicy(DefaultQAPolicy()))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, err: errors.New("verify failed")})

	_, err := q.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error when verifier fails")
	}
}

func TestExecuteTimeout(t *testing.T) {
	q := NewQuorum(WithPolicy(QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA},
		MinApprovals:    1,
		RejectThreshold: 1,
		Timeout:         100 * time.Millisecond,
	}))
	_ = q.RegisterVerifier(&mockVerifier{
		id:      "qa1",
		role:    VerifierRoleQA,
		verdict: VerdictApprove,
		delay:   1 * time.Second,
	})

	_, err := q.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestDissentingVerdictsPreserved(t *testing.T) {
	q := NewQuorum(WithPolicy(QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA, VerifierRoleSecurity},
		MinApprovals:    2,
		RejectThreshold: 1,
	}))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictApprove})
	_ = q.RegisterVerifier(&mockVerifier{id: "sec1", role: VerifierRoleSecurity, verdict: VerdictReject, reason: "security vulnerability"})

	verdict, err := q.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if len(verdict.Dissenting) != 1 {
		t.Errorf("expected 1 dissenting, got %d", len(verdict.Dissenting))
	}
	if verdict.Dissenting[0].Reason != "security vulnerability" {
		t.Errorf("expected reason preserved, got %s", verdict.Dissenting[0].Reason)
	}
}

func TestVerdictToEvidence(t *testing.T) {
	q := NewQuorum(WithPolicy(DefaultQAPolicy()))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictApprove})

	verdict, _ := q.Execute(context.Background(), nil)
	evidence := verdict.ToEvidence()

	if evidence["passed"] != true {
		t.Error("expected passed in evidence")
	}
	if evidence["approvals"] != 1 {
		t.Errorf("expected 1 approval in evidence, got %v", evidence["approvals"])
	}
}

func TestCanPromote(t *testing.T) {
	q := NewQuorum(WithPolicy(DefaultQAPolicy()))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictApprove})

	verdict, _ := q.Execute(context.Background(), nil)

	canPromote, reason := q.CanPromote(context.Background(), verdict)
	if !canPromote {
		t.Errorf("expected can promote, got: %s", reason)
	}
}

func TestCanPromoteFailed(t *testing.T) {
	q := NewQuorum(WithPolicy(DefaultQAPolicy()))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictReject})

	verdict, _ := q.Execute(context.Background(), nil)

	canPromote, _ := q.CanPromote(context.Background(), verdict)
	if canPromote {
		t.Error("expected cannot promote for failed verdict")
	}
}

func TestSetPolicy(t *testing.T) {
	q := NewQuorum()
	newPolicy := QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRolePolicy},
		MinApprovals:    1,
		RejectThreshold: 1,
	}

	q.SetPolicy(newPolicy)
	got := q.GetPolicy()

	if len(got.RequiredRoles) != 1 || got.RequiredRoles[0] != VerifierRolePolicy {
		t.Error("policy not updated")
	}
}

func TestAbstainVerdict(t *testing.T) {
	q := NewQuorum(WithPolicy(QuorumPolicy{
		RequiredRoles:   []VerifierRole{VerifierRoleQA},
		MinApprovals:    1,
		RejectThreshold: 2,
	}))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictAbstain})

	verdict, err := q.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if verdict.Abstentions != 1 {
		t.Errorf("expected 1 abstention, got %d", verdict.Abstentions)
	}
	if verdict.Passed {
		t.Error("expected verdict to fail with only abstentions")
	}
}

func TestDefaultPolicies(t *testing.T) {
	qa := DefaultQAPolicy()
	if len(qa.RequiredRoles) != 1 {
		t.Error("QA policy should have 1 required role")
	}

	sec := DefaultSecurityPolicy()
	if len(sec.RequiredRoles) != 2 {
		t.Error("Security policy should have 2 required roles")
	}

	rel := DefaultReleasePolicy()
	if len(rel.RequiredRoles) != 3 {
		t.Error("Release policy should have 3 required roles")
	}
}

func TestVerifierResultToJSON(t *testing.T) {
	r := &VerifierResult{
		VerifierID: "v1",
		Role:       VerifierRoleQA,
		Verdict:    VerdictApprove,
		Reason:     "looks good",
		Timestamp:  time.Now(),
	}

	json := r.ToJSON()
	if json == "" {
		t.Error("expected non-empty JSON")
	}
}

func TestQuorumWithPromoter(t *testing.T) {
	promoter := &mockPromoter{allow: true}
	q := NewQuorum(WithPolicy(DefaultQAPolicy()), WithPromoter(promoter))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictApprove})

	verdict, _ := q.Execute(context.Background(), nil)

	canPromote, _ := q.CanPromote(context.Background(), verdict)
	if !canPromote {
		t.Error("expected promoter to allow promotion")
	}
}

type mockPromoter struct {
	allow bool
}

func (m *mockPromoter) CanPromote(ctx context.Context, verdict *QuorumVerdict) (bool, string) {
	if m.allow {
		return true, "mock promoter allows"
	}
	return false, "mock promoter denies"
}

func TestPromotionBlockedWhenQuorumFails(t *testing.T) {
	q := NewQuorum(WithPolicy(DefaultQAPolicy()))
	_ = q.RegisterVerifier(&mockVerifier{id: "qa1", role: VerifierRoleQA, verdict: VerdictReject})

	verdict, _ := q.Execute(context.Background(), nil)

	canPromote, reason := q.CanPromote(context.Background(), verdict)
	if canPromote {
		t.Error("expected promotion to be blocked")
	}
	if reason != "quorum verdict did not pass" {
		t.Errorf("unexpected reason: %s", reason)
	}
}
