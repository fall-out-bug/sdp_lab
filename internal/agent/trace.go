package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sdp_dev/internal/bus"
)

// TraceEmitter writes run events to .sdp/runs/ and publishes to NATS sdp.lifecycle.
type TraceEmitter struct {
	bus       bus.Bus
	projectID string
	runID     string
	agentID   string
	role      string
	workDir   string
	issueID   string // set by BeginTrace
	mu        sync.Mutex
	events    []TraceEvent
	runPath   string
}

// TraceEvent is a single phase/state transition.
type TraceEvent struct {
	At      string `json:"at"`
	Phase   string `json:"phase"`
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
	PRURL   string `json:"pr_url,omitempty"`
}

// NewTraceEmitter creates a TraceEmitter. b must be non-nil: EmitPhase and
// Publish calls will panic if b is nil. Callers (e.g. orchestrator) always pass
// a connected bus; adapter-controller may pass nil when NATS is unavailable
// and does not call EmitPhase on that emitter.
func NewTraceEmitter(b bus.Bus, projectID, runID, agentID, role, workDir string) *TraceEmitter {
	runPath := filepath.Join(workDir, ".sdp", "runs", runID+".json")
	return &TraceEmitter{
		bus:       b,
		projectID: projectID,
		runID:     runID,
		agentID:   agentID,
		role:      role,
		workDir:   workDir,
		runPath:   runPath,
		events:    []TraceEvent{},
	}
}

// BeginTrace records the start of execution and publishes to NATS.
func (t *TraceEmitter) BeginTrace(issueID string) error {
	t.mu.Lock()
	t.issueID = issueID
	t.mu.Unlock()
	return t.EmitPhase("claimed", "ok", "agent "+t.agentID+" claimed "+issueID)
}

// EmitPhase records a phase transition and publishes to NATS.
// No-op if bus is nil (avoids panic when TraceEmitter is used without NATS).
func (t *TraceEmitter) EmitPhase(phase, state, message string) error {
	if t.bus == nil {
		return nil
	}
	evt := TraceEvent{
		At:      time.Now().UTC().Format(time.RFC3339Nano),
		Phase:   phase,
		State:   state,
		Message: message,
	}

	t.mu.Lock()
	t.events = append(t.events, evt)
	events := append([]TraceEvent(nil), t.events...)
	t.mu.Unlock()

	t.mu.Lock()
	issueID := t.issueID
	t.mu.Unlock()

	doc := map[string]any{
		"run_id":     t.runID,
		"issue_id":   issueID,
		"events":     events,
		"last_phase": phase,
		"last_state": state,
		"agent_id":   t.agentID,
		"role":       t.role,
	}

	if err := t.writeRunFile(doc); err != nil {
		return err
	}

	// Publish to NATS for real-time subscribers
	subject := fmt.Sprintf("sdp.lifecycle.%s.%s", t.projectID, t.runID)
	env := bus.Envelope{
		IssueID:       "",
		ArtifactID:    "trace-" + phase,
		ArtifactClass: "lifecycle",
		Phase:         phase,
		Role:          t.role,
		CapturedAt:    evt.At,
		Payload:       mustMarshal(map[string]any{"phase": phase, "state": state, "message": message}),
		RunID:         t.runID,
		ProjectID:     t.projectID,
	}
	return t.bus.Publish(subject, env)
}

func (t *TraceEmitter) writeRunFile(doc map[string]any) error {
	dir := filepath.Dir(t.runPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.runPath, data, 0o644)
}

// RunPath returns the path to the run file.
func (t *TraceEmitter) RunPath() string {
	return t.runPath
}

// Events returns a copy of events so far.
func (t *TraceEmitter) Events() []TraceEvent {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]TraceEvent(nil), t.events...)
}

// EmitHeartbeatIfDue emits a heartbeat phase only if the last heartbeat was >= interval ago.
// Use during Running phase to satisfy "heartbeat every 60s" requirement.
func (t *TraceEmitter) EmitHeartbeatIfDue(interval time.Duration) error {
	t.mu.Lock()
	var lastAt time.Time
	for i := len(t.events) - 1; i >= 0; i-- {
		if t.events[i].Phase == "heartbeat" {
			lastAt, _ = time.Parse(time.RFC3339Nano, t.events[i].At)
			break
		}
	}
	t.mu.Unlock()

	if lastAt.IsZero() || time.Since(lastAt) >= interval {
		return t.EmitPhase("heartbeat", "ok", "alive")
	}
	return nil
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
