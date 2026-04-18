package dispatch_test

import (
	"context"
	"testing"

	"sdp_dev/internal/dispatch"
)

func TestIsLowComplexity(t *testing.T) {
	tests := []struct {
		ws   string
		want bool
	}{
		{"add stub for ModelGateway", true},
		{"write boilerplate for FSM", true},
		{"write test for NewRunner", true},
		{"add_field Status to Session", true},
		{"rename field ID to SessionID", true},
		{"SIMPLE refactor of handler", true},
		{"implement interface io.Reader", true},
		{"design new auth architecture", false},
		{"trace goroutine leak in agentloop", false},
		{"refactor dispatch routing across 5 packages", false},
		{"", false},
	}
	for _, tc := range tests {
		got := dispatch.IsLowComplexity(tc.ws)
		if got != tc.want {
			t.Errorf("IsLowComplexity(%q) = %v, want %v", tc.ws, got, tc.want)
		}
	}
}

func TestRouter_LocalModel_WinsOnLowComplexity(t *testing.T) {
	cloudProfile := &dispatch.CapabilityProfile{
		Harness:  "claude-code",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-6",
		Capabilities: map[string]dispatch.CapabilityScore{
			"feature:go": {TestPassRate: 0.85},
		},
	}
	router := &dispatch.Router{
		Profiles: []*dispatch.CapabilityProfile{cloudProfile},
		LocalConfig: &dispatch.LocalConfig{
			BaseURL: "http://localhost:11434",
			Model:   "qwen2.5-coder:7b",
			Score:   0.9,
		},
	}

	task := dispatch.TaskClassification{
		Phase:       "build",
		TaskType:    "feature",
		Language:    "go",
		Complexity:  "low",
		RequiredCap: "coding",
	}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if dec.Provider != "ollama" {
		t.Errorf("expected ollama provider for low-complexity task, got %q", dec.Provider)
	}
	if dec.Model != "qwen2.5-coder:7b" {
		t.Errorf("expected qwen2.5-coder:7b, got %q", dec.Model)
	}
	if dec.Score < 0.9 {
		t.Errorf("expected score >= 0.9, got %.4f", dec.Score)
	}
}

func TestRouter_LocalModel_SkippedOnHighComplexity(t *testing.T) {
	cloudProfile := &dispatch.CapabilityProfile{
		Harness:  "claude-code",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-6",
		Capabilities: map[string]dispatch.CapabilityScore{
			"feature:go": {TestPassRate: 0.85},
		},
	}
	router := &dispatch.Router{
		Profiles: []*dispatch.CapabilityProfile{cloudProfile},
		LocalConfig: &dispatch.LocalConfig{
			BaseURL: "http://localhost:11434",
			Model:   "qwen2.5-coder:7b",
			Score:   0.9,
		},
	}

	task := dispatch.TaskClassification{
		Phase:       "design",
		TaskType:    "architecture",
		Language:    "go",
		Complexity:  "high",
		RequiredCap: "reasoning",
	}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if dec.Provider == "ollama" {
		t.Errorf("ollama should not be selected for high-complexity reasoning task")
	}
}

func TestRouter_LocalModel_DisabledWhenNil(t *testing.T) {
	cloudProfile := &dispatch.CapabilityProfile{
		Harness:  "claude-code",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-6",
		Capabilities: map[string]dispatch.CapabilityScore{
			"feature:go": {TestPassRate: 0.85},
		},
	}
	// LocalConfig is nil — local routing disabled
	router := &dispatch.Router{Profiles: []*dispatch.CapabilityProfile{cloudProfile}}

	task := dispatch.TaskClassification{
		Complexity:  "low",
		RequiredCap: "coding",
		TaskType:    "feature",
		Language:    "go",
	}
	dec, err := router.Route(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if dec.Provider == "ollama" {
		t.Errorf("ollama should not appear when LocalConfig is nil")
	}
}
