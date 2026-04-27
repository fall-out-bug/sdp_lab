package cascade

import (
	"context"
	"errors"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence"
)

// fakeF144Strategy is a mock Strategy for testing with a real Checker.
type fakeF144Strategy struct {
	name     string
	subScore float64
	reason   string
	err      error
}

func (f *fakeF144Strategy) Name() string { return f.name }

func (f *fakeF144Strategy) Run(ctx context.Context, in confidence.StrategyInput[string]) (confidence.StrategyOutput, error) {
	if f.err != nil {
		return confidence.StrategyOutput{}, f.err
	}
	return confidence.StrategyOutput{
		SubScore: f.subScore,
		Reason:   f.reason,
	}, nil
}

// newTestChecker creates a real F144 Checker with a single fake strategy.
// Uses "self_check" as the strategy name so it's recognized by DefaultPolicy weights.
func newTestChecker(subScore float64, err error) *confidence.Checker[string] {
	strategies := []confidence.Strategy[string]{
		&fakeF144Strategy{name: "self_check", subScore: subScore, err: err},
	}
	policy := confidence.DefaultPolicy()
	checker, _ := confidence.NewChecker[string](nil, strategies, policy)
	return checker
}

// TestConfidenceAdapter_HighScorePasses verifies score above threshold → ok=true.
func TestConfidenceAdapter_HighScorePasses(t *testing.T) {
	// Arrange: F144 Checker with high subscore (will compose to high aggregate score)
	// DefaultPolicy composes average of subscores, so 0.9 should yield StatusOK
	checker := newTestChecker(0.9, nil)
	adapter := NewConfidenceAdapter(checker, 0.7)

	// Act
	req := InvokeRequest{Harness: "test", Prompt: "test prompt"}
	resp := &harness.Result{Output: "valid response"}
	ok, reason := adapter.Check(context.Background(), req, resp)

	// Assert
	if !ok {
		t.Errorf("expected ok=true for high subscore (0.9), got false; reason: %s", reason)
	}
	if reason != "" {
		t.Errorf("expected empty reason for passing check, got %q", reason)
	}
}

// TestConfidenceAdapter_LowScoreFails verifies score below threshold → ok=false.
func TestConfidenceAdapter_LowScoreFails(t *testing.T) {
	// Arrange: F144 Checker with low subscore (will compose to low aggregate score)
	// A subscore of 0.3 should compose to StatusFail (below OK threshold in DefaultPolicy)
	checker := newTestChecker(0.3, nil)
	adapter := NewConfidenceAdapter(checker, 0.7)

	// Act
	req := InvokeRequest{Harness: "test", Prompt: "test prompt"}
	resp := &harness.Result{Output: "weak response"}
	ok, reason := adapter.Check(context.Background(), req, resp)

	// Assert
	if ok {
		t.Errorf("expected ok=false for low subscore (0.3), got true")
	}
	if reason != "confidence_below_threshold" {
		t.Errorf("expected reason 'confidence_below_threshold', got %q", reason)
	}
}

// TestConfidenceAdapter_CheckerError verifies F144 Checker error → ok=false with error reason.
func TestConfidenceAdapter_CheckerError(t *testing.T) {
	// Arrange: F144 Checker with a strategy that errors
	testErr := errors.New("test error")
	checker := newTestChecker(0.5, testErr)
	adapter := NewConfidenceAdapter(checker, 0.7)

	// Act
	req := InvokeRequest{Harness: "test", Prompt: "test prompt"}
	resp := &harness.Result{Output: "some response"}
	ok, reason := adapter.Check(context.Background(), req, resp)

	// Assert
	if ok {
		t.Errorf("expected ok=false when checker returns error, got true")
	}
	if !contains(reason, "checker_error") {
		t.Errorf("expected reason to contain 'checker_error', got %q", reason)
	}
	if !contains(reason, "test error") {
		t.Errorf("expected reason to contain error text, got %q", reason)
	}
}

// TestConfidenceAdapter_NilChecker verifies nil F144 Checker → ok=true (graceful degrade).
func TestConfidenceAdapter_NilChecker(t *testing.T) {
	// Arrange: nil F144 Checker
	adapter := NewConfidenceAdapter(nil, 0.7)

	// Act
	req := InvokeRequest{Harness: "test", Prompt: "test prompt"}
	resp := &harness.Result{Output: "any response"}
	ok, reason := adapter.Check(context.Background(), req, resp)

	// Assert
	if !ok {
		t.Errorf("expected ok=true for nil checker (graceful degrade), got false; reason: %s", reason)
	}
	if reason != "" {
		t.Errorf("expected empty reason for nil checker, got %q", reason)
	}
}

// TestConfidenceAdapter_ExactlyAtThreshold verifies boundary condition.
func TestConfidenceAdapter_ExactlyAtThreshold(t *testing.T) {
	// Arrange: F144 Checker with subscore that composes to exactly at adapter threshold
	// Using subscore 0.8 which composes to 0.8 (single strategy, no weight averaging)
	// Set adapter threshold to 0.8 so they're exactly equal
	checker := newTestChecker(0.8, nil)
	adapter := NewConfidenceAdapter(checker, 0.8)

	// Act
	req := InvokeRequest{Harness: "test", Prompt: "test prompt"}
	resp := &harness.Result{Output: "borderline response"}
	ok, reason := adapter.Check(context.Background(), req, resp)

	// Assert: should pass because composed score >= threshold (>=, not just >)
	if !ok {
		t.Errorf("expected ok=true for score exactly at threshold (0.8), got false; reason: %s", reason)
	}
	if reason != "" {
		t.Errorf("expected empty reason for passing threshold check, got %q", reason)
	}
}

// TestConfidenceAdapter_HighThresholdStaysOK verifies StatusOK with score at or above threshold passes.
func TestConfidenceAdapter_HighThresholdStaysOK(t *testing.T) {
	// Arrange: F144 Checker with high subscore that composes to StatusOK
	// DefaultPolicy: OKThreshold=0.8, so subscore 0.85 should give StatusOK
	checker := newTestChecker(0.85, nil)
	adapter := NewConfidenceAdapter(checker, 0.7) // reasonable threshold

	// Act
	req := InvokeRequest{Harness: "test", Prompt: "test prompt"}
	resp := &harness.Result{Output: "response"}
	ok, reason := adapter.Check(context.Background(), req, resp)

	// Assert: Should pass because status is OK and score is above threshold
	if !ok {
		t.Errorf("expected ok=true for StatusOK with score above threshold, got false; reason: %s", reason)
	}
}

// TestConfidenceAdapter_UnsureEscalates verifies StatusUnsure always escalates (returns ok=false)
// regardless of score, per F145 §4.7 escalation policy.
func TestConfidenceAdapter_UnsureEscalates(t *testing.T) {
	// Arrange: F144 Checker configured to produce StatusUnsure
	// To get StatusUnsure, use a subscore that composes between FailThreshold and OKThreshold
	// DefaultPolicy: FailThreshold=0.5, OKThreshold=0.8
	// So a subscore of 0.65 should compose to StatusUnsure
	checker := newTestChecker(0.65, nil)
	adapter := NewConfidenceAdapter(checker, 0.5) // threshold below the composed score

	// Act
	req := InvokeRequest{Harness: "test", Prompt: "test prompt"}
	resp := &harness.Result{Output: "borderline response"}
	ok, reason := adapter.Check(context.Background(), req, resp)

	// Assert: Must escalate (ok=false) even though score is above adapter threshold
	if ok {
		t.Errorf("expected ok=false for StatusUnsure, got true (must escalate per §4.7)")
	}
	if !contains(reason, "confidence_unsure") {
		t.Errorf("expected reason to start with 'confidence_unsure', got %q", reason)
	}
	t.Logf("SUCCESS: StatusUnsure correctly escalates, reason=%q", reason)
}
