package cascade

import (
	"context"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

// TestCascadeReplay_EmptyCorpus verifies that an empty corpus produces an empty report without panic.
func TestCascadeReplay_EmptyCorpus(t *testing.T) {
	invoker := NewInvoker(nil, nil)
	runner := NewReplayRunner(invoker)

	report, err := runner.Run(context.Background(), ReplayCorpus{Cases: []ReplayCase{}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if report.TotalCases != 0 {
		t.Errorf("expected TotalCases=0, got %d", report.TotalCases)
	}
	if len(report.Cases) != 0 {
		t.Errorf("expected empty Cases, got %d", len(report.Cases))
	}
	if report.StayedCheapPct != 0 {
		t.Errorf("expected StayedCheapPct=0, got %f", report.StayedCheapPct)
	}
	if report.EscalatedPct != 0 {
		t.Errorf("expected EscalatedPct=0, got %f", report.EscalatedPct)
	}
}

// TestCascadeReplay_SingleCaseStaysCheap verifies that a single case with a passing response
// records tier_used and hops correctly.
// Note: Using nil router (harness execution wiring is deferred to F148-XX)
func TestCascadeReplay_SingleCaseStaysCheap(t *testing.T) {
	// Use nil router for now (Router.Route wiring is tested separately in cascade_test.go)
	invoker := NewInvoker(nil, nil)
	runner := NewReplayRunner(invoker)

	corpus := ReplayCorpus{
		Cases: []ReplayCase{
			{
				ID:           "fizzbuzz",
				Prompt:       "Write fizzbuzz in Go",
				ExpectedTier: dispatch.TierLocal,
				ExpectedHops: 1,
			},
		},
	}

	report, err := runner.Run(context.Background(), corpus)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if report.TotalCases != 1 {
		t.Errorf("expected TotalCases=1, got %d", report.TotalCases)
	}
	if len(report.Cases) != 1 {
		t.Errorf("expected 1 case result, got %d", len(report.Cases))
	}

	result := report.Cases[0]
	if result.CaseID != "fizzbuzz" {
		t.Errorf("expected CaseID=fizzbuzz, got %s", result.CaseID)
	}
	if result.TierUsed != dispatch.TierLocal {
		t.Errorf("expected TierUsed=TierLocal, got %s", result.TierUsed)
	}
	if result.Hops != 1 {
		t.Errorf("expected Hops=1, got %d", result.Hops)
	}
	if result.Cause != "ok" {
		t.Errorf("expected Cause=ok, got %s", result.Cause)
	}
	if !result.Match {
		t.Error("expected Match=true, got false")
	}

	// 1 case stayed cheap (TierLocal + hops==1) → 100%
	if report.StayedCheapPct != 100.0 {
		t.Errorf("expected StayedCheapPct=100, got %f", report.StayedCheapPct)
	}
	if report.EscalatedPct != 0 {
		t.Errorf("expected EscalatedPct=0, got %f", report.EscalatedPct)
	}
}

// TestCascadeReplay_MultiCase verifies multi-case distribution: 1 stays cheap, 1 escalates 2 hops, 1 hits max_depth.
// Note: Using nil router (harness execution wiring is deferred to F148-XX)
func TestCascadeReplay_MultiCase(t *testing.T) {
	// Use a mock checker to control cascade behavior per case
	// Note: behavior map values are "shouldReject": true=reject, false=accept
	mockChecker := &mockCheckerPerCase{
		behaviors: map[string]map[int]bool{
			"prompt_a": {1: false}, // prompt_a: accept on first tier (hop 1)
			"prompt_b": {1: true, 2: false}, // prompt_b: reject hop 1, accept on hop 2
			"prompt_c": {1: true, 2: true, 3: true}, // prompt_c: reject all hops
		},
	}
	// Use nil router for now (Router.Route wiring is tested separately in cascade_test.go)
	invoker := NewInvoker(nil, mockChecker)
	runner := NewReplayRunner(invoker)

	corpus := ReplayCorpus{
		Cases: []ReplayCase{
			{
				ID:           "case1-cheap",
				Prompt:       "prompt_a",
				ExpectedTier: dispatch.TierLocal,
				ExpectedHops: 1,
			},
			{
				ID:           "case2-escalated",
				Prompt:       "prompt_b",
				ExpectedTier: dispatch.TierFast,
				ExpectedHops: 2,
			},
			{
				ID:           "case3-maxdepth",
				Prompt:       "prompt_c",
				ExpectedTier: dispatch.TierBalanced,
				ExpectedHops: 3,
			},
		},
	}

	report, err := runner.Run(context.Background(), corpus)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if report.TotalCases != 3 {
		t.Errorf("expected TotalCases=3, got %d", report.TotalCases)
	}
	if len(report.Cases) != 3 {
		t.Errorf("expected 3 case results, got %d", len(report.Cases))
	}

	// Check case 1: stayed cheap
	if report.Cases[0].TierUsed != dispatch.TierLocal {
		t.Errorf("case1: expected TierLocal, got %s", report.Cases[0].TierUsed)
	}
	if report.Cases[0].Hops != 1 {
		t.Errorf("case1: expected Hops=1, got %d", report.Cases[0].Hops)
	}

	// Check case 2: escalated
	if report.Cases[1].TierUsed != dispatch.TierFast {
		t.Errorf("case2: expected TierFast, got %s", report.Cases[1].TierUsed)
	}
	if report.Cases[1].Hops != 2 {
		t.Errorf("case2: expected Hops=2, got %d", report.Cases[1].Hops)
	}

	// Check case 3: escalated to balanced but still rejected, hits max_depth
	if report.Cases[2].TierUsed != dispatch.TierBalanced {
		t.Errorf("case3: expected TierBalanced, got %s", report.Cases[2].TierUsed)
	}
	if report.Cases[2].Hops != 3 {
		t.Errorf("case3: expected Hops=3, got %d", report.Cases[2].Hops)
	}
	if report.Cases[2].Cause != "max_depth" {
		t.Errorf("case3: expected cause=max_depth, got %s", report.Cases[2].Cause)
	}

	// Verify tier distribution
	if dist, ok := report.TierUsedDist[string(dispatch.TierLocal)]; !ok || dist != 1 {
		t.Errorf("expected TierLocal count=1, got %d", dist)
	}
	if dist, ok := report.TierUsedDist[string(dispatch.TierFast)]; !ok || dist != 1 {
		t.Errorf("expected TierFast count=1, got %d", dist)
	}
	if dist, ok := report.TierUsedDist[string(dispatch.TierBalanced)]; !ok || dist != 1 {
		t.Errorf("expected TierBalanced count=1, got %d", dist)
	}

	// Verify cause distribution
	if dist, ok := report.CauseDist["ok"]; !ok || dist != 2 {
		t.Errorf("expected cause=ok count=2, got %d", dist)
	}
	if dist, ok := report.CauseDist["max_depth"]; !ok || dist != 1 {
		t.Errorf("expected cause=max_depth count=1, got %d", dist)
	}

	// Verify percentages: 1 of 3 stayed cheap → 33.3%, 2 of 3 escalated (hops>1) → 66.7%
	if report.StayedCheapPct < 33.0 || report.StayedCheapPct > 34.0 {
		t.Errorf("expected StayedCheapPct≈33.3, got %f", report.StayedCheapPct)
	}
	if report.EscalatedPct < 66.0 || report.EscalatedPct > 67.0 {
		t.Errorf("expected EscalatedPct≈66.7, got %f", report.EscalatedPct)
	}
}

// TestCascadeReplay_PercentagesCorrect verifies calculation of StayedCheapPct and EscalatedPct.
// Note: Using nil router (harness execution wiring is deferred to F148-XX)
func TestCascadeReplay_PercentagesCorrect(t *testing.T) {
	mockChecker := &mockCheckerPerCase{
		behaviors: map[string]map[int]bool{
			"prompt1": {1: false}, // prompt1: accept on first tier
			"prompt2": {1: false}, // prompt2: accept on first tier
			"prompt3": {1: true, 2: false}, // prompt3: reject on first, accept on second
		},
	}
	// Use nil router for now (Router.Route wiring is tested separately in cascade_test.go)
	invoker := NewInvoker(nil, mockChecker)
	runner := NewReplayRunner(invoker)

	corpus := ReplayCorpus{
		Cases: []ReplayCase{
			{
				ID:           "case1",
				Prompt:       "prompt1",
				ExpectedTier: dispatch.TierLocal,
				ExpectedHops: 1,
			},
			{
				ID:           "case2",
				Prompt:       "prompt2",
				ExpectedTier: dispatch.TierLocal,
				ExpectedHops: 1,
			},
			{
				ID:           "case3",
				Prompt:       "prompt3",
				ExpectedTier: dispatch.TierFast,
				ExpectedHops: 2,
			},
		},
	}

	report, err := runner.Run(context.Background(), corpus)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 2 of 3 stayed cheap (TierLocal + hops==1) → 66.67%
	// 1 of 3 escalated (hops>1) → 33.33%
	if report.StayedCheapPct < 66.0 || report.StayedCheapPct > 67.0 {
		t.Errorf("expected StayedCheapPct≈66.7, got %f", report.StayedCheapPct)
	}
	if report.EscalatedPct < 33.0 || report.EscalatedPct > 34.0 {
		t.Errorf("expected EscalatedPct≈33.3, got %f", report.EscalatedPct)
	}
}

// Stub routers for deterministic testing

type stubRouterCheap struct{}

func (s *stubRouterCheap) Route(ctx context.Context, task dispatch.TaskClassification, limits map[string]*harness.Limits) (*dispatch.DispatchDecision, error) {
	return &dispatch.DispatchDecision{
		Harness: "stub",
	}, nil
}

type stubRouterMultiMock struct{}

func (s *stubRouterMultiMock) Route(ctx context.Context, task dispatch.TaskClassification, limits map[string]*harness.Limits) (*dispatch.DispatchDecision, error) {
	return &dispatch.DispatchDecision{
		Harness: "stub",
	}, nil
}

// mockCheckerPerCase controls which tier should be rejected based on case ID and hop count
type mockCheckerPerCase struct {
	behaviors map[string]map[int]bool // case_id -> (hop -> should reject)
	hopCounts map[string]int           // case_id -> current hop
}

func (m *mockCheckerPerCase) Check(ctx context.Context, req InvokeRequest, resp *harness.Result) (ok bool, reason string) {
	// Extract case ID from prompt for this test
	caseID := req.Prompt
	if m.hopCounts == nil {
		m.hopCounts = make(map[string]int)
	}
	m.hopCounts[caseID]++
	hop := m.hopCounts[caseID]

	behavior, exists := m.behaviors[caseID]
	if !exists {
		return true, "" // default accept if no behavior defined
	}

	shouldReject := behavior[hop]
	if shouldReject {
		return false, "mock rejection"
	}
	return true, ""
}
