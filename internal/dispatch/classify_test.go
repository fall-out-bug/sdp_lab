package dispatch

import (
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name        string
		pkt         ContextPacketSummary
		wantPhase   string
		wantType    string
		wantLang    string
		wantCap     string
		wantRisk    string
		wantComplex string
	}{
		{
			name: "go refactor",
			pkt: ContextPacketSummary{
				Phase:      "build",
				Workstream: "refactor the auth module",
				ScopeFiles: []string{"auth.go", "token.go", "user.go"},
				Risk:       "medium",
			},
			wantPhase:   "build",
			wantType:    "refactor",
			wantLang:    "go",
			wantCap:     "coding",
			wantRisk:    "medium",
			wantComplex: "medium",
		},
		{
			name: "discovery phase",
			pkt: ContextPacketSummary{
				Phase:      "discovery",
				Workstream: "investigate new caching strategy",
				ScopeFiles: []string{"cache.go", "store.go"},
				Risk:       "low",
			},
			wantPhase:   "discovery",
			wantType:    "research",
			wantLang:    "go",
			wantCap:     "reasoning",
			wantRisk:    "low",
			wantComplex: "medium",
		},
		{
			name: "typescript feature",
			pkt: ContextPacketSummary{
				Phase:      "build",
				Workstream: "add dashboard component",
				ScopeFiles: []string{"Dashboard.tsx", "api.ts", "types.ts"},
				Risk:       "low",
			},
			wantPhase:   "build",
			wantType:    "feature",
			wantLang:    "typescript",
			wantCap:     "coding",
			wantRisk:    "low",
			wantComplex: "medium",
		},
		{
			name: "bugfix",
			pkt: ContextPacketSummary{
				Phase:      "build",
				Workstream: "fix login bug causing session timeout",
				ScopeFiles: []string{"session.go", "login.go"},
				Risk:       "high",
			},
			wantPhase:   "build",
			wantType:    "bugfix",
			wantLang:    "go",
			wantCap:     "coding",
			wantRisk:    "high",
			wantComplex: "medium",
		},
		{
			name: "design phase",
			pkt: ContextPacketSummary{
				Phase:      "design",
				Workstream: "design the new API layer",
				ScopeFiles: []string{"api.ts", "handler.ts"},
				Risk:       "medium",
			},
			wantPhase:   "design",
			wantType:    "architecture",
			wantLang:    "typescript",
			wantCap:     "reasoning",
			wantRisk:    "medium",
			wantComplex: "medium",
		},
		{
			name: "review phase",
			pkt: ContextPacketSummary{
				Phase:      "review",
				Workstream: "review pull request changes",
				ScopeFiles: []string{"main.py", "utils.py"},
				Risk:       "low",
			},
			wantPhase:   "review",
			wantType:    "analysis",
			wantLang:    "python",
			wantCap:     "review",
			wantRisk:    "low",
			wantComplex: "medium",
		},
		{
			name: "qa phase",
			pkt: ContextPacketSummary{
				Phase:      "qa",
				Workstream: "validate integration test suite",
				ScopeFiles: []string{"test.rs", "lib.rs"},
				Risk:       "low",
			},
			wantPhase:   "qa",
			wantType:    "analysis",
			wantLang:    "rust",
			wantCap:     "review",
			wantRisk:    "low",
			wantComplex: "low", // test keyword triggers low complexity
		},
		{
			name: "mixed extensions picks most common",
			pkt: ContextPacketSummary{
				Phase:      "build",
				Workstream: "update frontend components",
				ScopeFiles: []string{"app.tsx", "index.tsx", "style.js"},
				Risk:       "low",
			},
			wantPhase:   "build",
			wantType:    "feature",
			wantLang:    "typescript",
			wantCap:     "coding",
			wantRisk:    "low",
			wantComplex: "medium",
		},
		{
			name: "no scope files yields empty language",
			pkt: ContextPacketSummary{
				Phase:      "build",
				Workstream: "update documentation",
				ScopeFiles: []string{},
				Risk:       "low",
			},
			wantPhase:   "build",
			wantType:    "feature",
			wantLang:    "",
			wantCap:     "coding",
			wantRisk:    "low",
			wantComplex: "medium",
		},
		{
			name: "fix keyword in workstream",
			pkt: ContextPacketSummary{
				Phase:      "build",
				Workstream: "fix broken pipeline",
				ScopeFiles: []string{"pipeline.go"},
				Risk:       "high",
			},
			wantPhase:   "build",
			wantType:    "bugfix",
			wantLang:    "go",
			wantCap:     "coding",
			wantRisk:    "high",
			wantComplex: "medium",
		},
		{
			name: "jsx extension maps to javascript",
			pkt: ContextPacketSummary{
				Phase:      "build",
				Workstream: "add button component",
				ScopeFiles: []string{"Button.jsx", "helpers.js"},
				Risk:       "low",
			},
			wantPhase:   "build",
			wantType:    "feature",
			wantLang:    "javascript",
			wantCap:     "coding",
			wantRisk:    "low",
			wantComplex: "medium",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.pkt)

			if got.Phase != tc.wantPhase {
				t.Errorf("Phase: got %q, want %q", got.Phase, tc.wantPhase)
			}
			if got.TaskType != tc.wantType {
				t.Errorf("TaskType: got %q, want %q", got.TaskType, tc.wantType)
			}
			if got.Language != tc.wantLang {
				t.Errorf("Language: got %q, want %q", got.Language, tc.wantLang)
			}
			if got.RequiredCap != tc.wantCap {
				t.Errorf("RequiredCap: got %q, want %q", got.RequiredCap, tc.wantCap)
			}
			if got.Risk != tc.wantRisk {
				t.Errorf("Risk: got %q, want %q", got.Risk, tc.wantRisk)
			}
			if got.Complexity != tc.wantComplex {
				t.Errorf("Complexity: got %q, want %q", got.Complexity, tc.wantComplex)
			}
		})
	}
}

func TestClassify_LowComplexityKeywords(t *testing.T) {
	tests := []struct {
		name           string
		workstream     string
		wantComplexity string
	}{
		{
			name:           "stub keyword",
			workstream:     "add stub for ModelGateway",
			wantComplexity: "low",
		},
		{
			name:           "boilerplate keyword",
			workstream:     "write boilerplate for FSM",
			wantComplexity: "low",
		},
		{
			name:           "test keyword",
			workstream:     "write test for NewRunner",
			wantComplexity: "low",
		},
		{
			name:           "add_field keyword",
			workstream:     "add_field Status to Session",
			wantComplexity: "low",
		},
		{
			name:           "add field with space",
			workstream:     "add field UserID to Request",
			wantComplexity: "low",
		},
		{
			name:           "rename keyword",
			workstream:     "rename field ID to SessionID",
			wantComplexity: "low",
		},
		{
			name:           "simple keyword",
			workstream:     "SIMPLE refactor of handler",
			wantComplexity: "low",
		},
		{
			name:           "implement interface keyword",
			workstream:     "implement interface io.Reader",
			wantComplexity: "low",
		},
		{
			name:           "docstring keyword",
			workstream:     "add docstring to Parse",
			wantComplexity: "low",
		},
		{
			name:           "complex architecture task",
			workstream:     "design new auth architecture",
			wantComplexity: "medium",
		},
		{
			name:           "complex debugging task",
			workstream:     "trace goroutine leak in agentloop",
			wantComplexity: "medium",
		},
		{
			name:           "complex refactor across packages",
			workstream:     "refactor dispatch routing across 5 packages",
			wantComplexity: "medium",
		},
		{
			name:           "empty workstream",
			workstream:     "",
			wantComplexity: "medium",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pkt := ContextPacketSummary{
				Phase:      "build",
				Workstream: tc.workstream,
				ScopeFiles: []string{"internal/dispatch/route.go"},
				Risk:       "low",
			}
			got := Classify(pkt)
			if got.Complexity != tc.wantComplexity {
				t.Errorf("Classify(%q).Complexity = %q, want %q", tc.workstream, got.Complexity, tc.wantComplexity)
			}
			if got.RequiredCap != "coding" {
				t.Errorf("Classify(%q).RequiredCap = %q, want coding", tc.workstream, got.RequiredCap)
			}
		})
	}
}

func TestClassify_HighRiskOverridesLowComplexity(t *testing.T) {
	pkt := ContextPacketSummary{
		Phase:      "build",
		Workstream: "add stub for ModelGateway",
		ScopeFiles: []string{"internal/dispatch/route.go"},
		Risk:       "high",
	}
	got := Classify(pkt)
	if got.Complexity != "medium" {
		t.Errorf("Classify with high risk should override to medium, got %q", got.Complexity)
	}
	if got.Risk != "high" {
		t.Errorf("Classify should preserve original Risk field, got %q", got.Risk)
	}
}

func TestClassify_MediumRiskDoesNotOverride(t *testing.T) {
	pkt := ContextPacketSummary{
		Phase:      "build",
		Workstream: "add stub for ModelGateway",
		ScopeFiles: []string{"internal/dispatch/route.go"},
		Risk:       "medium",
	}
	got := Classify(pkt)
	if got.Complexity != "low" {
		t.Errorf("Classify with medium risk should not override low complexity, got %q", got.Complexity)
	}
}
