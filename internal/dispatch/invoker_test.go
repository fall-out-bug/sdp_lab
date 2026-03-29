package dispatch

import (
	"context"
	"errors"
	"testing"

	"sdp_dev/internal/dispatch/harness"
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

func (f *fakeInvoker) Invoke(_ context.Context, _, agent, prompt string) (string, int, error) {
	f.lastAgent = agent
	f.lastPrompt = prompt
	return f.output, f.exitCode, f.err
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
