package kernel

import (
	"encoding/json"
	"testing"
	"time"
)

func TestKernelContractsJSONRoundTrip(t *testing.T) {
	timestamp := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	payload := struct {
		Agent    AgentDefinition   `json:"agent"`
		Session  SessionState      `json:"session"`
		Pack     WorkflowPack      `json:"pack"`
		Policy   ToolPolicy        `json:"policy"`
		Trace    TraceEvent        `json:"trace"`
		Runtime  RuntimeResult     `json:"runtime"`
		Request  RuntimeInvocation `json:"request"`
		Provider ProviderMeta      `json:"provider"`
		Routing  RoutingDecision   `json:"routing"`
		Approval ApprovalHook      `json:"approval"`
		Eval     EvalCase          `json:"eval"`
	}{
		Agent: AgentDefinition{
			ID:                   "planner",
			Role:                 "planner",
			Description:          "Plans one execution slice.",
			AllowedWorkflowPacks: []string{"core.pack"},
			ToolPolicyRef:        "default.policy",
			RequiredCapabilities: CapabilitySet{ToolCalling: true, MaxContextTokens: 128000},
		},
		Session: SessionState{
			RunID:            RunID("run-123"),
			SessionID:        SessionID("sess-123"),
			AgentID:          "planner",
			WorkflowPackRefs: []string{"core.pack"},
			ContextSegments: []ContextSegment{
				{ID: "ctx-1", Kind: "system", Source: "pack", Content: "You are the planner.", TokenCount: 5},
			},
			ToolPolicyRef: "default.policy",
			MemoryCandidates: []MemoryCandidate{
				{ID: "mem-1", Scope: MemoryScopeTask, Content: "Use kernel types first.", Confidence: 0.9, Reason: "design note", TraceRefs: []string{"trace-1"}},
			},
			ArtifactRefs: []ArtifactRef{
				{Type: ArtifactEvidence, Path: ".sdp/evidence/run-123.json", Hash: "abc", CreatedAt: timestamp, Size: 42},
			},
			TraceRefs: []string{"trace-1"},
		},
		Pack: WorkflowPack{
			ID:           "core.pack",
			Version:      "1.0.0",
			Description:  "Core pack",
			Dependencies: []string{"base.pack"},
			PromptFragments: []PromptFragment{
				{ID: "core.instructions", Kind: "system", Content: "Keep changes narrow."},
			},
			Roles: []RoleDefinition{
				{ID: "planner", Phase: "plan", Agent: "metis", PromptFragmentIDs: []string{"core.instructions"}},
			},
			Hooks: []HookRegistration{
				{ID: "approval.default", Kind: HookKindApproval},
			},
			EvalRefs: []string{"eval.route"},
		},
		Policy: ToolPolicy{
			ID:              "default.policy",
			Description:     "Default tool policy",
			Version:         "v1",
			EvaluationShape: []string{"tool", "files"},
			DefaultDecision: ToolPolicyAsk,
		},
		Trace: TraceEvent{
			ID:            "trace-1",
			RunID:         RunID("run-123"),
			SessionID:     SessionID("sess-123"),
			AgentID:       "planner",
			Kind:          TraceEventTool,
			Phase:         "plan",
			At:            "2026-03-31T12:00:00Z",
			CorrelationID: "corr-1",
			Payload:       json.RawMessage(`{"tool":"bd ready"}`),
		},
		Runtime: RuntimeResult{
			Output:   "done",
			ExitCode: 0,
		},
		Request: RuntimeInvocation{
			WorkDir: "/tmp/project",
			Agent:   "implementer",
			Prompt:  "Execute @build 00-092-02",
		},
		Provider: ProviderMeta{
			ProviderID: "openai",
			ModelName:  "gpt-4o",
			Capabilities: ModelCapabilities{
				Vision:          true,
				FunctionCall:    true,
				Streaming:       true,
				MaxContext:      128000,
				SupportedModels: []ModelID{"gpt-4o"},
			},
			Latency: 120 * time.Millisecond,
		},
		Routing: RoutingDecision{
			SelectedProvider: "openai",
			SelectedModel:    "gpt-4o",
			FallbackChain:    []ProviderID{"openai", "selfhosted"},
			DecisionReason:   "default provider",
			PolicyID:         "default.policy",
			EvaluatedAt:      timestamp,
			InputHash:        "route-123",
			Constraints: RoutingConstraints{
				AllowedProviders: []ProviderID{"openai", "selfhosted"},
				AllowedModels:    []ModelID{"gpt-4o"},
				MaxCostPerToken:  0.01,
			},
		},
		Approval: ApprovalHook{
			ID:          "approval.default",
			RequestType: "tool_call",
			Description: "Escalate restricted tool use.",
		},
		Eval: EvalCase{
			ID:                    "eval.route",
			Scenario:              "planner routes shell use through policy",
			Inputs:                map[string]any{"phase": "plan"},
			ExpectedTraceKinds:    []TraceEventKind{TraceEventTool},
			ExpectedToolDecisions: []ToolPolicyDecision{ToolPolicyAsk},
			ExpectedArtifacts:     []ArtifactType{ArtifactEvidence},
			ExpectedMemoryScopes:  []MemoryScope{MemoryScopeTask},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal kernel payload: %v", err)
	}

	var decoded struct {
		Agent    AgentDefinition   `json:"agent"`
		Session  SessionState      `json:"session"`
		Pack     WorkflowPack      `json:"pack"`
		Policy   ToolPolicy        `json:"policy"`
		Trace    TraceEvent        `json:"trace"`
		Runtime  RuntimeResult     `json:"runtime"`
		Request  RuntimeInvocation `json:"request"`
		Provider ProviderMeta      `json:"provider"`
		Routing  RoutingDecision   `json:"routing"`
		Approval ApprovalHook      `json:"approval"`
		Eval     EvalCase          `json:"eval"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal kernel payload: %v", err)
	}

	if decoded.Agent.ID != payload.Agent.ID {
		t.Fatalf("agent id mismatch: got %q want %q", decoded.Agent.ID, payload.Agent.ID)
	}
	if decoded.Session.RunID != payload.Session.RunID {
		t.Fatalf("run id mismatch: got %q want %q", decoded.Session.RunID, payload.Session.RunID)
	}
	if len(decoded.Session.ArtifactRefs) != 1 || decoded.Session.ArtifactRefs[0].Type != ArtifactEvidence {
		t.Fatalf("artifact refs mismatch: %#v", decoded.Session.ArtifactRefs)
	}
	if decoded.Trace.Kind != TraceEventTool || string(decoded.Trace.Payload) != `{"tool":"bd ready"}` {
		t.Fatalf("trace mismatch: %#v", decoded.Trace)
	}
	if len(decoded.Pack.Roles) != 1 || decoded.Pack.Roles[0].Agent != "metis" {
		t.Fatalf("workflow pack roles mismatch: %#v", decoded.Pack.Roles)
	}
	if len(decoded.Pack.Hooks) != 1 || decoded.Pack.Hooks[0].Kind != HookKindApproval {
		t.Fatalf("workflow pack hooks mismatch: %#v", decoded.Pack.Hooks)
	}
	if decoded.Runtime.ExitCode != 0 || decoded.Request.Agent != "implementer" {
		t.Fatalf("runtime mismatch: %#v %#v", decoded.Runtime, decoded.Request)
	}
	if decoded.Provider.ProviderID != "openai" || decoded.Provider.Capabilities.MaxContext != 128000 {
		t.Fatalf("provider mismatch: %#v", decoded.Provider)
	}
	if decoded.Routing.SelectedModel != "gpt-4o" || len(decoded.Routing.FallbackChain) != 2 {
		t.Fatalf("routing mismatch: %#v", decoded.Routing)
	}
	if len(decoded.Eval.ExpectedToolDecisions) != 1 || decoded.Eval.ExpectedToolDecisions[0] != ToolPolicyAsk {
		t.Fatalf("eval mismatch: %#v", decoded.Eval)
	}
}
