package confidence_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sdp_dev/internal/inference/confidence"
)

// fakeStrategy is a deterministic Strategy[string] for orchestration tests.
type fakeStrategy struct {
	name     string
	subScore float64
	hardFail bool
	reason   string
	err      error
	tokens   confidence.TokenUsage
	calls    int
}

func (f *fakeStrategy) Name() string { return f.name }

func (f *fakeStrategy) Run(_ context.Context, _ confidence.StrategyInput[string]) (confidence.StrategyOutput, error) {
	f.calls++
	if f.err != nil {
		return confidence.StrategyOutput{}, f.err
	}
	return confidence.StrategyOutput{
		SubScore: f.subScore,
		HardFail: f.hardFail,
		Reason:   f.reason,
		Tokens:   f.tokens,
	}, nil
}

func TestCheckerHappyPath(t *testing.T) {
	policy := confidence.DefaultPolicy()
	strategies := []confidence.Strategy[string]{
		&fakeStrategy{name: "self_check", subScore: 1.0, reason: "critic agreed"},
		&fakeStrategy{name: "consensus", subScore: 1.0, reason: "samples agree"},
		&fakeStrategy{name: "constraint", subScore: 1.0, reason: "schema valid"},
	}

	checker, err := confidence.NewChecker[string](nil, strategies, policy)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	res, err := checker.Check(context.Background(), confidence.Request[string]{
		Input:  "hello",
		Answer: "world",
		Raw:    "world",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.Status != confidence.StatusOK {
		t.Errorf("Status = %q, want OK", res.Status)
	}
	if res.Score != 1.0 {
		t.Errorf("Score = %v, want 1.0", res.Score)
	}
	if res.Answer != "world" {
		t.Errorf("Answer = %q, want world", res.Answer)
	}
	if got := res.SubScores["self_check"]; got != 1.0 {
		t.Errorf("SubScores[self_check] = %v, want 1.0", got)
	}
	if len(res.Reasons) != 3 {
		t.Errorf("Reasons len = %d, want 3", len(res.Reasons))
	}
	if res.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", res.Attempts)
	}
}

func TestCheckerUnsureBand(t *testing.T) {
	strategies := []confidence.Strategy[string]{
		&fakeStrategy{name: "self_check", subScore: 0.6},
		&fakeStrategy{name: "consensus", subScore: 0.6},
		&fakeStrategy{name: "constraint", subScore: 0.6},
	}
	checker, err := confidence.NewChecker[string](nil, strategies, confidence.DefaultPolicy())
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	res, err := checker.Check(context.Background(), confidence.Request[string]{Answer: "a", Raw: "a"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.Status != confidence.StatusUnsure {
		t.Errorf("Status = %q, want UNSURE", res.Status)
	}
}

func TestCheckerFailBand(t *testing.T) {
	strategies := []confidence.Strategy[string]{
		&fakeStrategy{name: "self_check", subScore: 0.1},
		&fakeStrategy{name: "consensus", subScore: 0.2},
		&fakeStrategy{name: "constraint", subScore: 0.3},
	}
	checker, err := confidence.NewChecker[string](nil, strategies, confidence.DefaultPolicy())
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	res, err := checker.Check(context.Background(), confidence.Request[string]{Answer: "a", Raw: "a"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.Status != confidence.StatusFail {
		t.Errorf("Status = %q, want FAIL", res.Status)
	}
}

func TestCheckerHardFailForcesFail(t *testing.T) {
	// All semantic strategies say 1.0, but constraint hard-fails (e.g.
	// schema invalid). Status MUST be FAIL regardless of score.
	strategies := []confidence.Strategy[string]{
		&fakeStrategy{name: "self_check", subScore: 1.0},
		&fakeStrategy{name: "consensus", subScore: 1.0},
		&fakeStrategy{name: "constraint", subScore: 0.0, hardFail: true, reason: "schema broken"},
	}
	checker, err := confidence.NewChecker[string](nil, strategies, confidence.DefaultPolicy())
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	res, err := checker.Check(context.Background(), confidence.Request[string]{Answer: "a", Raw: "a"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.Status != confidence.StatusFail {
		t.Errorf("Status = %q, want FAIL (hard-fail)", res.Status)
	}
	// Reason from the hard-failing strategy must surface.
	joined := strings.Join(res.Reasons, "|")
	if !strings.Contains(joined, "schema broken") {
		t.Errorf("Reasons missing hard-fail reason: %v", res.Reasons)
	}
}

func TestCheckerStrategyErrorPropagates(t *testing.T) {
	boom := errors.New("strategy boom")
	strategies := []confidence.Strategy[string]{
		&fakeStrategy{name: "self_check", subScore: 1.0},
		&fakeStrategy{name: "consensus", err: boom},
	}
	checker, err := confidence.NewChecker[string](nil, strategies, confidence.DefaultPolicy())
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	_, err = checker.Check(context.Background(), confidence.Request[string]{Answer: "a", Raw: "a"})
	if err == nil {
		t.Fatalf("expected strategy error to propagate, got nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrap of %v", err, boom)
	}
}

func TestCheckerAggregatesTokens(t *testing.T) {
	strategies := []confidence.Strategy[string]{
		&fakeStrategy{name: "self_check", subScore: 1.0, tokens: confidence.TokenUsage{In: 100, Out: 30}},
		&fakeStrategy{name: "consensus", subScore: 1.0, tokens: confidence.TokenUsage{In: 50, Out: 10}},
	}
	checker, err := confidence.NewChecker[string](nil, strategies, confidence.DefaultPolicy())
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	res, err := checker.Check(context.Background(), confidence.Request[string]{Answer: "a", Raw: "a"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.Trace.TokensIn != 150 {
		t.Errorf("Trace.TokensIn = %d, want 150", res.Trace.TokensIn)
	}
	if res.Trace.TokensOut != 40 {
		t.Errorf("Trace.TokensOut = %d, want 40", res.Trace.TokensOut)
	}
}

func TestCheckerContextCancellation(t *testing.T) {
	strategies := []confidence.Strategy[string]{
		&fakeStrategy{name: "self_check", subScore: 1.0},
	}
	checker, err := confidence.NewChecker[string](nil, strategies, confidence.DefaultPolicy())
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	_, err = checker.Check(ctx, confidence.Request[string]{Answer: "a", Raw: "a"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestNewCheckerValidatesPolicy(t *testing.T) {
	bad := confidence.DefaultPolicy()
	bad.OKThreshold = 0.3
	bad.FailThreshold = 0.7

	_, err := confidence.NewChecker[string](nil, nil, bad)
	if err == nil {
		t.Errorf("expected error for invalid policy, got nil")
	}
}

func TestNewCheckerRequiresStrategies(t *testing.T) {
	if _, err := confidence.NewChecker[string](nil, nil, confidence.DefaultPolicy()); err == nil {
		t.Errorf("expected error for nil strategies, got nil")
	}
	if _, err := confidence.NewChecker[string](nil, []confidence.Strategy[string]{}, confidence.DefaultPolicy()); err == nil {
		t.Errorf("expected error for empty strategies, got nil")
	}
}

func TestTokenUsageAdd(t *testing.T) {
	a := confidence.TokenUsage{In: 100, Out: 50}
	b := confidence.TokenUsage{In: 30, Out: 20}
	got := a.Add(b)
	if got.In != 130 || got.Out != 70 {
		t.Errorf("Add: got %+v, want In=130 Out=70", got)
	}
	// Receiver not mutated.
	if a.In != 100 || a.Out != 50 {
		t.Errorf("Add mutated receiver: %+v", a)
	}
}
