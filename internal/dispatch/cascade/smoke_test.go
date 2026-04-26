// Package cascade smoke_test provides end-to-end integration tests for the
// CascadingInvoker with real Provider registration and wiring.
// These tests verify the full F145 stack without making real API calls.
package cascade

import (
	"context"
	"testing"
	"time"

	"sdp_dev/internal/dispatch"
	"sdp_dev/internal/dispatch/harness"
	"sdp_dev/internal/dispatch/harness/providers"
)

// TestCascade_AllProvidersRegistered verifies that all 5 providers can be
// constructed and registered without panicking. This is a smoke test for the
// provider layer integration.
func TestCascade_AllProvidersRegistered(t *testing.T) {
	registry := providers.NewRegistry()

	// Create a LimitsCache for providers that need it
	cache := harness.NewLimitsCache(30 * time.Second)

	// Register all 5 providers with minimal dependencies
	registry.Register(providers.NewOpenAIProvider(cache))
	registry.Register(providers.NewAnthropicProvider(cache))
	registry.Register(providers.NewCursorProvider(cache, nil, nil))
	registry.Register(providers.NewKimiProvider(cache))
	registry.Register(providers.NewOllamaProvider(""))

	// Verify all providers registered
	names := registry.All()
	if len(names) != 5 {
		t.Errorf("expected 5 providers, got %d", len(names))
	}

	// Verify we can retrieve each one
	expectedProviders := []string{"openai", "anthropic", "cursor", "kimi", "ollama"}
	for _, name := range expectedProviders {
		p := registry.Get(name)
		if p == nil {
			t.Errorf("provider %s not found", name)
		}
		// Sanity check: Name() matches
		if p.Name() != name {
			t.Errorf("provider Name() returned %q, expected %q", p.Name(), name)
		}
	}
}

// TestCascade_FullStack_StaysCheap verifies that when a stub Checker returns OK
// on the first tier, the cascade stays on TierLocal with hops=1.
func TestCascade_FullStack_StaysCheap(t *testing.T) {
	// Create a stub router that succeeds on TierLocal
	stubRouter := &stubRouterForSmoke{}

	// Create a stub Checker that always accepts
	alwaysOKChecker := &stubCheckerAlwaysOK{}

	invoker := NewInvoker(stubRouter, alwaysOKChecker)

	req := InvokeRequest{
		Harness:   "test",
		Prompt:    "Write FizzBuzz in Go",
		Agent:     "coder",
		Worktree:  "/tmp/test",
		TaskFile:  "/tmp/test/task.json",
		Timeout:   10 * time.Second,
		StartTier: dispatch.TierLocal,
	}

	result, err := invoker.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Tier != dispatch.TierLocal {
		t.Errorf("expected tier TierLocal, got %v", result.Tier)
	}
	if result.Hops != 1 {
		t.Errorf("expected hops=1, got %d", result.Hops)
	}
	if result.Cause != "ok" {
		t.Errorf("expected cause=ok, got %s", result.Cause)
	}
}

// TestCascade_FullStack_EscalatesOnce verifies that when a Checker rejects
// the first tier but accepts the second, the cascade escalates once to TierFast
// with hops=2.
func TestCascade_FullStack_EscalatesOnce(t *testing.T) {
	stubRouter := &stubRouterForSmoke{}

	// Checker that rejects TierLocal, accepts TierFast
	escalatingChecker := &stubCheckerEscalateOnce{}

	invoker := NewInvoker(stubRouter, escalatingChecker)

	req := InvokeRequest{
		Harness:   "test",
		Prompt:    "Design a caching layer in Go",
		Agent:     "coder",
		Worktree:  "/tmp/test",
		TaskFile:  "/tmp/test/task.json",
		Timeout:   10 * time.Second,
		StartTier: dispatch.TierLocal,
	}

	result, err := invoker.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Tier != dispatch.TierFast {
		t.Errorf("expected tier TierFast, got %v", result.Tier)
	}
	if result.Hops != 2 {
		t.Errorf("expected hops=2, got %d", result.Hops)
	}
	if result.Cause != "ok" {
		t.Errorf("expected cause=ok, got %s", result.Cause)
	}
}

// TestCascade_FullStack_HitsMaxDepth verifies that when a Checker rejects all
// tiers, the cascade exhausts and returns with cause=max_depth on the last tier tried.
func TestCascade_FullStack_HitsMaxDepth(t *testing.T) {
	stubRouter := &stubRouterForSmoke{}

	// Checker that always rejects
	alwaysRejectChecker := &stubCheckerAlwaysReject{}

	invoker := NewInvoker(stubRouter, alwaysRejectChecker)

	req := InvokeRequest{
		Harness:   "test",
		Prompt:    "Implement a distributed consensus algorithm",
		Agent:     "coder",
		Worktree:  "/tmp/test",
		TaskFile:  "/tmp/test/task.json",
		Timeout:   10 * time.Second,
		StartTier: dispatch.TierLocal,
	}

	result, err := invoker.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should hit max_depth (default maxDepth=3)
	if result.Cause != "max_depth" {
		t.Errorf("expected cause=max_depth, got %s", result.Cause)
	}
	// Should have tried up to maxDepth tiers
	if result.Hops > 3 {
		t.Errorf("expected hops <= 3, got %d", result.Hops)
	}
}

// TestCascade_FullStack_BudgetExhausted verifies that when the budget runs out,
// the cascade returns early with cause=budget.
func TestCascade_FullStack_BudgetExhausted(t *testing.T) {
	stubRouter := &stubRouterForSmoke{}
	alwaysOKChecker := &stubCheckerAlwaysOK{}

	invoker := NewInvoker(stubRouter, alwaysOKChecker)

	// Set a very short budget (already expired)
	invoker.budget = &Budget{
		MaxDuration: 1 * time.Millisecond,
		StartTime:   time.Now().Add(-2 * time.Millisecond), // already expired
	}

	req := InvokeRequest{
		Harness:   "test",
		Prompt:    "Write a simple function",
		Agent:     "coder",
		Worktree:  "/tmp/test",
		TaskFile:  "/tmp/test/task.json",
		Timeout:   10 * time.Second,
		StartTier: dispatch.TierLocal,
	}

	result, err := invoker.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Cause != "budget" {
		t.Errorf("expected cause=budget, got %s", result.Cause)
	}
}

// ============ Stub implementations for smoke tests ============

// stubRouterForSmoke is a minimal router that returns a dummy decision.
// It doesn't actually invoke a harness; we populate result.Output to exercise
// the heuristic and checker gates.
type stubRouterForSmoke struct{}

func (s *stubRouterForSmoke) Route(ctx context.Context, task dispatch.TaskClassification, limits map[string]*harness.Limits) (*dispatch.DispatchDecision, error) {
	return &dispatch.DispatchDecision{
		Harness: "stub",
	}, nil
}

// stubCheckerAlwaysOK always returns (true, "").
type stubCheckerAlwaysOK struct{}

func (c *stubCheckerAlwaysOK) Check(ctx context.Context, req InvokeRequest, resp *harness.Result) (ok bool, reason string) {
	return true, ""
}

// stubCheckerEscalateOnce rejects hop 1 (TierLocal), accepts hop 2+ (TierFast+).
type stubCheckerEscalateOnce struct {
	hopCount int // track number of calls
}

func (c *stubCheckerEscalateOnce) Check(ctx context.Context, req InvokeRequest, resp *harness.Result) (ok bool, reason string) {
	c.hopCount++
	if c.hopCount == 1 {
		return false, "mock rejection on first hop"
	}
	return true, ""
}

// stubCheckerAlwaysReject always returns (false, "mock rejection").
type stubCheckerAlwaysReject struct{}

func (c *stubCheckerAlwaysReject) Check(ctx context.Context, req InvokeRequest, resp *harness.Result) (ok bool, reason string) {
	return false, "mock rejection"
}
