package evals

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

type traceFixture struct {
	Events []kernel.TraceEvent `json:"events"`
}

// NewRoutingTraceEvent builds a first-class routing trace event from kernel-owned routing data.
func NewRoutingTraceEvent(decision kernel.RoutingDecision, input kernel.RoutingInput) (kernel.TraceEvent, error) {
	return newTraceEvent(kernel.TraceEventRouting, map[string]any{
		"decision": decision,
		"input":    input,
	})
}

// NewToolDecisionTraceEvent builds a tool-policy trace event from a request/decision pair.
func NewToolDecisionTraceEvent(request kernel.ToolCallRequest, decision kernel.ToolCallDecision) (kernel.TraceEvent, error) {
	return newTraceEvent(kernel.TraceEventTool, map[string]any{
		"tool":          request.Tool,
		"files":         request.Files,
		"tool_request":  request,
		"tool_decision": decision,
		"decision":      decision.Decision,
	})
}

// NewMemoryCandidateTraceEvent builds a memory trace event from an emitted candidate.
func NewMemoryCandidateTraceEvent(candidate kernel.MemoryCandidate) (kernel.TraceEvent, error) {
	return newTraceEvent(kernel.TraceEventMemory, map[string]any{
		"memory": candidate,
		"scope":  candidate.Scope,
	})
}

// WriteTraceFixture stores events in the canonical eval trace document format.
func WriteTraceFixture(path string, events ...kernel.TraceEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(traceFixture{Events: events}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func newTraceEvent(kind kernel.TraceEventKind, payload any) (kernel.TraceEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return kernel.TraceEvent{}, err
	}
	return kernel.TraceEvent{
		Kind:    kind,
		At:      time.Now().UTC().Format(time.RFC3339),
		Payload: data,
	}, nil
}
