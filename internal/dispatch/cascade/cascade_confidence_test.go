package cascade

import (
	"context"
	"testing"

	"sdp_dev/internal/dispatch/harness"
	"sdp_dev/internal/inference/confidence"
)

// TestCascadeWithConfidence_ConstructorWires verifies the NewWithConfidence helper
// properly wires a F144 Checker into the cascade via the adapter.
func TestCascadeWithConfidence_ConstructorWires(t *testing.T) {
	// Arrange: Build a real F144 Checker with mid-score strategy
	strategy := &fakeF144Strategy{name: "self_check", subScore: 0.8}
	strategies := []confidence.Strategy[string]{strategy}
	policy := confidence.DefaultPolicy()
	checker, _ := confidence.NewChecker[string](nil, strategies, policy)

	// Act: Wire via NewWithConfidence helper
	invoker := NewWithConfidence(nil, checker, 0.7)

	// Assert: Invoker should be created with the adapter in place
	if invoker == nil {
		t.Fatal("NewWithConfidence should create a non-nil invoker")
	}
	if invoker.checker == nil {
		t.Fatal("Invoker checker should not be nil after NewWithConfidence")
	}
	// The checker should be an adapter wrapping the F144 checker
	_, isAdapter := invoker.checker.(*ConfidenceAdapter)
	if !isAdapter {
		t.Errorf("Invoker checker should be a ConfidenceAdapter, got %T", invoker.checker)
	}
}

// TestCascadeWithConfidence_AdapterIntegration verifies the adapter correctly
// integrates with the cascade decision logic in a minimal scenario.
func TestCascadeWithConfidence_AdapterIntegration(t *testing.T) {
	// Arrange: Create adapters with different checkers
	lowChecker := newTestChecker(0.3, nil)  // Will fail policy
	highChecker := newTestChecker(0.95, nil) // Will pass policy

	lowAdapter := NewConfidenceAdapter(lowChecker, 0.7)
	highAdapter := NewConfidenceAdapter(highChecker, 0.7)

	// Act & Assert: Test that adapters integrate correctly
	req := InvokeRequest{Harness: "test", Prompt: "test prompt"}
	result := &harness.Result{Output: "test output"}

	// High score should pass
	ok, _ := highAdapter.Check(context.Background(), req, result)
	if !ok {
		t.Error("High-score adapter should return ok=true")
	}

	// Low score should fail
	ok, _ = lowAdapter.Check(context.Background(), req, result)
	if ok {
		t.Error("Low-score adapter should return ok=false")
	}
}
