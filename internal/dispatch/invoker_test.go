package dispatch

import (
	"context"
	"errors"
	"testing"

	"sdp_dev/internal/dispatch/harness"
	"sdp_dev/internal/kernel"
)

// fakeInvoker is a test double for LLMInvoker.
type fakeInvoker struct {
	name       string
	output     string
	exitCode   int
	err        error
	lastAgent  string
	lastPrompt string
}

func (f *fakeInvoker) Invoke(_ context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	f.lastAgent = req.Agent
	f.lastPrompt = req.Prompt
	return kernel.RuntimeResult{Output: f.output, ExitCode: f.exitCode}, f.err
}

// fakePacketLoader returns a fixed ContextPacketSummary for testing.
func fakePacketLoader(_ string) (ContextPacketSummary, error) {
	return ContextPacketSummary{
		Phase:      "build",
		Workstream: "add new feature",
		ScopeFiles: []string{"main.go", "util.go"},
		Risk:       "low",
	}, nil
}

// TestDispatchingInvoker_Route verifies that with profiles configured, the
// DispatchingInvoker routes to the dispatched invoker (not fallback).
func TestDispatchingInvoker_Route(t *testing.T) {
	dispatched := &fakeInvoker{name: "dispatched", output: "dispatched-output", exitCode: 0}
	fallback := &fakeInvoker{name: "fallback", output: "fallback-output", exitCode: 0}

	profile := makeProfile("test-harness", "anthropic", "claude-3", featureCaps(0.92))

	di := &DispatchingInvoker{
		Router: &Router{
			Profiles: []*CapabilityProfile{profile},
		},
		Fallback: fallback,
		Limits: map[string]*harness.Limits{
			"anthropic": {Total: 100, Used: 10},
		},
		InvokerFor: func(_ string) LLMInvoker {
			return dispatched
		},
		PacketLoader: fakePacketLoader,
	}

	output, code, err := di.Invoke(context.Background(), "/project", "agent1", "do the thing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "dispatched-output" {
		t.Errorf("expected dispatched-output, got %q", output)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if fallback.lastAgent != "" {
		t.Error("fallback should not have been called")
	}
	if dispatched.lastAgent != "agent1" {
		t.Errorf("expected dispatched agent=agent1, got %q", dispatched.lastAgent)
	}
}

// TestDispatchingInvoker_FallbackOnNoProfiles verifies that with no profiles,
// the invoker falls back to the Fallback invoker.
func TestDispatchingInvoker_FallbackOnNoProfiles(t *testing.T) {
	dispatched := &fakeInvoker{name: "dispatched", output: "dispatched-output", exitCode: 0}
	fallback := &fakeInvoker{name: "fallback", output: "fallback-output", exitCode: 0}

	di := &DispatchingInvoker{
		Router:   &Router{Profiles: []*CapabilityProfile{}},
		Fallback: fallback,
		Limits:   map[string]*harness.Limits{},
		InvokerFor: func(_ string) LLMInvoker {
			return dispatched
		},
		PacketLoader: fakePacketLoader,
	}

	output, code, err := di.Invoke(context.Background(), "/project", "agent1", "do the thing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "fallback-output" {
		t.Errorf("expected fallback-output, got %q", output)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if dispatched.lastAgent != "" {
		t.Error("dispatched should not have been called")
	}
}

// TestDispatchingInvoker_FallbackOnPacketError verifies that a packet loader
// error causes the invoker to fall back to Fallback.
func TestDispatchingInvoker_FallbackOnPacketError(t *testing.T) {
	fallback := &fakeInvoker{name: "fallback", output: "fallback-output", exitCode: 0}

	profile := makeProfile("test-harness", "anthropic", "claude-3", featureCaps(0.92))

	di := &DispatchingInvoker{
		Router:   &Router{Profiles: []*CapabilityProfile{profile}},
		Fallback: fallback,
		Limits:   map[string]*harness.Limits{},
		InvokerFor: func(_ string) LLMInvoker {
			return &fakeInvoker{output: "dispatched"}
		},
		PacketLoader: func(_ string) (ContextPacketSummary, error) {
			return ContextPacketSummary{}, errors.New("context packet not found")
		},
	}

	output, _, err := di.Invoke(context.Background(), "/project", "agent1", "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "fallback-output" {
		t.Errorf("expected fallback-output on packet error, got %q", output)
	}
}

// TestDispatchingInvoker_FallbackOnNilInvoker verifies that when InvokerFor
// returns nil, the invoker falls back to Fallback.
func TestDispatchingInvoker_FallbackOnNilInvoker(t *testing.T) {
	fallback := &fakeInvoker{name: "fallback", output: "fallback-output", exitCode: 0}

	profile := makeProfile("unknown-harness", "anthropic", "claude-3", featureCaps(0.92))

	di := &DispatchingInvoker{
		Router:   &Router{Profiles: []*CapabilityProfile{profile}},
		Fallback: fallback,
		Limits:   map[string]*harness.Limits{},
		InvokerFor: func(_ string) LLMInvoker {
			return nil // simulates harness not registered
		},
		PacketLoader: fakePacketLoader,
	}

	output, _, err := di.Invoke(context.Background(), "/tmp", "agent1", "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "fallback-output" {
		t.Errorf("expected fallback-output on nil invoker, got %q", output)
	}
}

// TestDispatchingInvoker_ContextEnricher verifies that ContextEnricher enriches
// the prompt before it reaches the dispatched invoker.
func TestDispatchingInvoker_ContextEnricher(t *testing.T) {
	dispatched := &fakeInvoker{name: "dispatched", output: "ok", exitCode: 0}

	profile := makeProfile("test-harness", "anthropic", "claude-3", featureCaps(0.92))

	di := &DispatchingInvoker{
		Router:   &Router{Profiles: []*CapabilityProfile{profile}},
		Fallback: &fakeInvoker{name: "fallback"},
		Limits:   map[string]*harness.Limits{},
		InvokerFor: func(_ string) LLMInvoker {
			return dispatched
		},
		PacketLoader: fakePacketLoader,
		ContextEnricher: func(root, basePrompt string) string {
			return basePrompt + " [ENRICHED:" + root + "]"
		},
	}

	output, _, err := di.Invoke(context.Background(), "/project", "implementer", "do the thing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "ok" {
		t.Errorf("expected ok, got %q", output)
	}
	if dispatched.lastPrompt != "do the thing [ENRICHED:/project]" {
		t.Errorf("expected enriched prompt, got %q", dispatched.lastPrompt)
	}
}

// TestDispatchingInvoker_ContextEnricherNil verifies that nil ContextEnricher
// passes the prompt unchanged (backward compatible).
func TestDispatchingInvoker_ContextEnricherNil(t *testing.T) {
	dispatched := &fakeInvoker{name: "dispatched", output: "ok", exitCode: 0}

	profile := makeProfile("test-harness", "anthropic", "claude-3", featureCaps(0.92))

	di := &DispatchingInvoker{
		Router:   &Router{Profiles: []*CapabilityProfile{profile}},
		Fallback: &fakeInvoker{name: "fallback"},
		Limits:   map[string]*harness.Limits{},
		InvokerFor: func(_ string) LLMInvoker {
			return dispatched
		},
		PacketLoader:    fakePacketLoader,
		ContextEnricher: nil, // explicitly nil
	}

	_, _, err := di.Invoke(context.Background(), "/project", "implementer", "raw prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched.lastPrompt != "raw prompt" {
		t.Errorf("expected raw prompt, got %q", dispatched.lastPrompt)
	}
}

// TestDispatchingInvoker_CrossHarnessVerification verifies that review/qa agents
// get routed through VerifyHarness when set.
func TestDispatchingInvoker_CrossHarnessVerification(t *testing.T) {
	buildInvoker := &fakeInvoker{name: "build-harness", output: "built", exitCode: 0}
	verifyInvoker := &fakeInvoker{name: "verify-harness", output: "verified", exitCode: 0}

	buildProfile := makeProfile("claude", "anthropic", "claude-3", featureCaps(0.92))
	verifyProfile := makeProfile("opencode", "omo", "default", map[string]CapabilityScore{
		"review:go": {TestPassRate: 0.88},
	})

	di := &DispatchingInvoker{
		Router:   &Router{Profiles: []*CapabilityProfile{buildProfile, verifyProfile}},
		Fallback: &fakeInvoker{name: "fallback"},
		Limits:   map[string]*harness.Limits{},
		InvokerFor: func(name string) LLMInvoker {
			switch name {
			case "claude":
				return buildInvoker
			case "opencode":
				return verifyInvoker
			}
			return nil
		},
		PacketLoader:    fakePacketLoader,
		ContextEnricher: func(_, p string) string { return p + " [CTX]" },
		VerifyHarness: func(_ *DispatchDecision, _ TaskClassification) (*DispatchDecision, error) {
			return &DispatchDecision{
				Harness:  "opencode",
				Provider: "omo",
				Model:    "default",
				Score:    0.88,
			}, nil
		},
	}

	// Build agent should go through normal routing → claude
	output, _, err := di.Invoke(context.Background(), "/project", "implementer", "build it")
	if err != nil {
		t.Fatalf("build: unexpected error: %v", err)
	}
	if output != "built" {
		t.Errorf("build: expected built, got %q", output)
	}
	if buildInvoker.lastPrompt != "build it [CTX]" {
		t.Errorf("build: expected enriched prompt, got %q", buildInvoker.lastPrompt)
	}

	// Review agent should go through VerifyHarness → opencode
	output, _, err = di.Invoke(context.Background(), "/project", "reviewer", "review it")
	if err != nil {
		t.Fatalf("review: unexpected error: %v", err)
	}
	if output != "verified" {
		t.Errorf("review: expected verified, got %q", output)
	}
	if verifyInvoker.lastPrompt != "review it [CTX]" {
		t.Errorf("review: expected enriched prompt, got %q", verifyInvoker.lastPrompt)
	}
}
