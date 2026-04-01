package augmentation

import (
	"context"

	"sdp_dev/internal/kernel"
)

type ToolPolicyHook interface {
	ID() string
	DecideToolCall(ctx context.Context, session kernel.SessionState, req kernel.ToolCallRequest, current kernel.ToolCallDecision) (kernel.ToolCallDecision, error)
}

type ApprovalRuntimeHook interface {
	ID() string
	DecideApproval(ctx context.Context, session kernel.SessionState, req kernel.ToolCallRequest, current kernel.ApprovalDecision) (kernel.ApprovalDecision, error)
}

type MemoryCandidateHook interface {
	ID() string
	EmitMemoryCandidates(ctx context.Context, session kernel.SessionState, event kernel.TraceEvent) ([]kernel.MemoryCandidate, error)
}

type TraceEnrichmentHook interface {
	ID() string
	EnrichTrace(ctx context.Context, session kernel.SessionState, event kernel.TraceEvent) (kernel.TraceEvent, error)
}

type HookSet struct {
	toolPolicy []ToolPolicyHook
	approval   []ApprovalRuntimeHook
	memory     []MemoryCandidateHook
	trace      []TraceEnrichmentHook
}

func NewHookSet() *HookSet {
	return &HookSet{}
}

func (h *HookSet) RegisterToolPolicy(hook ToolPolicyHook) {
	if h == nil || hook == nil {
		return
	}
	h.toolPolicy = append(h.toolPolicy, hook)
}

func (h *HookSet) RegisterApproval(hook ApprovalRuntimeHook) {
	if h == nil || hook == nil {
		return
	}
	h.approval = append(h.approval, hook)
}

func (h *HookSet) RegisterMemoryCandidate(hook MemoryCandidateHook) {
	if h == nil || hook == nil {
		return
	}
	h.memory = append(h.memory, hook)
}

func (h *HookSet) RegisterTraceEnrichment(hook TraceEnrichmentHook) {
	if h == nil || hook == nil {
		return
	}
	h.trace = append(h.trace, hook)
}

func (h *HookSet) ApplyToolPolicy(ctx context.Context, session kernel.SessionState, req kernel.ToolCallRequest, current kernel.ToolCallDecision) (kernel.ToolCallDecision, error) {
	var err error
	decision := current
	for _, hook := range h.toolPolicy {
		decision, err = hook.DecideToolCall(ctx, session, req, decision)
		if err != nil {
			return decision, err
		}
	}
	return decision, nil
}

func (h *HookSet) ApplyApproval(ctx context.Context, session kernel.SessionState, req kernel.ToolCallRequest, current kernel.ApprovalDecision) (kernel.ApprovalDecision, error) {
	var err error
	decision := current
	for _, hook := range h.approval {
		decision, err = hook.DecideApproval(ctx, session, req, decision)
		if err != nil {
			return decision, err
		}
	}
	return decision, nil
}

func (h *HookSet) EmitMemoryCandidates(ctx context.Context, session kernel.SessionState, event kernel.TraceEvent) ([]kernel.MemoryCandidate, error) {
	var out []kernel.MemoryCandidate
	for _, hook := range h.memory {
		candidates, err := hook.EmitMemoryCandidates(ctx, session, event)
		if err != nil {
			return nil, err
		}
		out = append(out, candidates...)
	}
	return out, nil
}

func (h *HookSet) EnrichTrace(ctx context.Context, session kernel.SessionState, event kernel.TraceEvent) (kernel.TraceEvent, error) {
	current := event
	for _, hook := range h.trace {
		enriched, err := hook.EnrichTrace(ctx, session, current)
		if err != nil {
			return current, err
		}
		current = enriched
	}
	return current, nil
}
