package confidence_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"sdp_dev/internal/inference/confidence"
)

// slowStrategy waits for `delay` before returning, simulating a strategy
// that exhausts the latency budget.
type slowStrategy struct {
	name     string
	delay    time.Duration
	subScore float64
	calls    int
}

func (s *slowStrategy) Name() string { return s.name }

func (s *slowStrategy) Run(ctx context.Context, _ confidence.StrategyInput[string]) (confidence.StrategyOutput, error) {
	s.calls++
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return confidence.StrategyOutput{}, ctx.Err()
	}
	return confidence.StrategyOutput{SubScore: s.subScore, Reason: "ran"}, nil
}

func TestLatencyBudgetSkipsLaterStrategies(t *testing.T) {
	policy := confidence.DefaultPolicy()
	policy.MaxLatencyMs = 30 // tight

	first := &slowStrategy{name: "self_check", delay: 50 * time.Millisecond, subScore: 1.0}
	second := &slowStrategy{name: "consensus", delay: 100 * time.Millisecond, subScore: 1.0}
	third := &slowStrategy{name: "constraint", delay: 100 * time.Millisecond, subScore: 1.0}

	checker, err := confidence.NewChecker[string](nil,
		[]confidence.Strategy[string]{first, second, third}, policy)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	res, err := checker.Check(context.Background(), confidence.Request[string]{Answer: "a"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	// First strategy ran (≥1 call). Subsequent strategies should have been
	// skipped because budget was already blown after first finished.
	if first.calls != 1 {
		t.Errorf("first.calls = %d, want 1", first.calls)
	}
	if second.calls != 0 {
		t.Errorf("second.calls = %d, want 0 (budget skipped)", second.calls)
	}
	if third.calls != 0 {
		t.Errorf("third.calls = %d, want 0 (budget skipped)", third.calls)
	}

	// Skipped strategies must surface in Reasons so debugging is possible.
	joined := strings.Join(res.Reasons, "|")
	if !strings.Contains(joined, "skipped: budget") {
		t.Errorf("Reasons missing budget skip note: %v", res.Reasons)
	}
	// Skipped strategies contribute neutral (0.5) — composed score is
	// (0.4*1.0 + 0.4*0.5 + 0.2*0.5) = 0.7 → UNSURE.
	if res.Status != confidence.StatusUnsure {
		t.Errorf("Status = %q, want UNSURE (mixed neutral)", res.Status)
	}
}

func TestLatencyBudgetZeroDisablesEnforcement(t *testing.T) {
	policy := confidence.DefaultPolicy()
	policy.MaxLatencyMs = 0 // disabled

	first := &slowStrategy{name: "self_check", delay: 20 * time.Millisecond, subScore: 1.0}
	second := &slowStrategy{name: "consensus", delay: 20 * time.Millisecond, subScore: 1.0}

	checker, err := confidence.NewChecker[string](nil,
		[]confidence.Strategy[string]{first, second}, policy)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	_, err = checker.Check(context.Background(), confidence.Request[string]{Answer: "a"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if second.calls != 1 {
		t.Errorf("second.calls = %d with budget=0, want 1 (no enforcement)", second.calls)
	}
}
