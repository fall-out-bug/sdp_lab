package kernel

import (
	"encoding/json"
	"testing"
	"time"
)

func TestKernelContractsJSONRoundTrip(t *testing.T) {
	timestamp := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	payload := struct {
		Agent    AgentDefinition `json:"agent"`
		Session  SessionState    `json:"session"`
		Pack     WorkflowPack    `json:"pack"`
		Policy   ToolPolicy      `json:"policy"`
		Trace    TraceEvent      `json:"trace"`
		Approval ApprovalHook    `json:"approval"`
		Eval     EvalCase        `json:"eval"`
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
			Version:      "v1",
			Description:  "Core pack",
			Dependencies: []string{"base.pack"},
			Roles:        []string{"planner", "verifier"},
			Hooks:        []string{"approval.default"},
			EvalRefs:     []string{"eval.route"},
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
		Agent    AgentDefinition `json:"agent"`
		Session  SessionState    `json:"session"`
		Pack     WorkflowPack    `json:"pack"`
		Policy   ToolPolicy      `json:"policy"`
		Trace    TraceEvent      `json:"trace"`
		Approval ApprovalHook    `json:"approval"`
		Eval     EvalCase        `json:"eval"`
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
	if len(decoded.Eval.ExpectedToolDecisions) != 1 || decoded.Eval.ExpectedToolDecisions[0] != ToolPolicyAsk {
		t.Fatalf("eval mismatch: %#v", decoded.Eval)
	}
}
