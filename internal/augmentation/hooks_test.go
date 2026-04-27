package augmentation

import (
	"context"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

type toolPolicyHookStub struct{}

func (toolPolicyHookStub) ID() string { return "tool-policy.default" }
func (toolPolicyHookStub) DecideToolCall(_ context.Context, _ kernel.SessionState, _ kernel.ToolCallRequest, current kernel.ToolCallDecision) (kernel.ToolCallDecision, error) {
	if current.Decision == kernel.ToolPolicyAsk {
		current.Decision = kernel.ToolPolicyAllow
		current.Reason = "approved by hook"
	}
	return current, nil
}

type approvalHookStub struct{}

func (approvalHookStub) ID() string { return "approval.default" }
func (approvalHookStub) DecideApproval(_ context.Context, _ kernel.SessionState, _ kernel.ToolCallRequest, current kernel.ApprovalDecision) (kernel.ApprovalDecision, error) {
	return kernel.ApprovalApprove, nil
}

type memoryHookStub struct{}

func (memoryHookStub) ID() string { return "memory.default" }
func (memoryHookStub) EmitMemoryCandidates(_ context.Context, _ kernel.SessionState, event kernel.TraceEvent) ([]kernel.MemoryCandidate, error) {
	return []kernel.MemoryCandidate{{ID: "mem-1", Scope: kernel.MemoryScopeTask, Reason: event.Phase}}, nil
}

type traceHookStub struct{}

func (traceHookStub) ID() string { return "trace.default" }
func (traceHookStub) EnrichTrace(_ context.Context, _ kernel.SessionState, event kernel.TraceEvent) (kernel.TraceEvent, error) {
	event.CorrelationID = "augmented"
	return event, nil
}

func TestHookSetAppliesSurfacesWithoutRuntime(t *testing.T) {
	hooks := NewHookSet()
	hooks.RegisterToolPolicy(toolPolicyHookStub{})
	hooks.RegisterApproval(approvalHookStub{})
	hooks.RegisterMemoryCandidate(memoryHookStub{})
	hooks.RegisterTraceEnrichment(traceHookStub{})

	session := kernel.SessionState{RunID: kernel.RunID("run-1"), SessionID: kernel.SessionID("sess-1")}
	req := kernel.ToolCallRequest{Tool: "exec"}

	decision, err := hooks.ApplyToolPolicy(context.Background(), session, req, kernel.ToolCallDecision{
		Decision: kernel.ToolPolicyAsk,
	})
	if err != nil {
		t.Fatalf("ApplyToolPolicy: %v", err)
	}
	if decision.Decision != kernel.ToolPolicyAllow {
		t.Fatalf("decision = %s, want allow", decision.Decision)
	}

	approval, err := hooks.ApplyApproval(context.Background(), session, req, kernel.ApprovalEscalate)
	if err != nil {
		t.Fatalf("ApplyApproval: %v", err)
	}
	if approval != kernel.ApprovalApprove {
		t.Fatalf("approval = %s, want approve", approval)
	}

	memories, err := hooks.EmitMemoryCandidates(context.Background(), session, kernel.TraceEvent{Phase: "build"})
	if err != nil {
		t.Fatalf("EmitMemoryCandidates: %v", err)
	}
	if len(memories) != 1 || memories[0].Scope != kernel.MemoryScopeTask {
		t.Fatalf("unexpected memories: %+v", memories)
	}

	trace, err := hooks.EnrichTrace(context.Background(), session, kernel.TraceEvent{Phase: "build"})
	if err != nil {
		t.Fatalf("EnrichTrace: %v", err)
	}
	if trace.CorrelationID != "augmented" {
		t.Fatalf("correlation id = %q, want augmented", trace.CorrelationID)
	}
}
