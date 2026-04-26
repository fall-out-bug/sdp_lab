// Package agentloop implements SDP's internal LLM agent loop.
// SDP uses this when it is itself the agent — running its own phases
// (discovery shaping, architectural analysis, evidence review) through
// a structured FSM with gates.
//
// This is distinct from internal/executor.ServeBridge, which dispatches
// work to an EXTERNAL harness (Claude Code, Cursor, opencode serve).
// See internal/executor/bridge_serve.go for that path.
package agentloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

// ErrHarnessTerminated is returned by RestoreHarness when the session was
// stopped via Stop() and cannot be resumed. Callers must use errors.Is() to
// detect this condition — never compare error strings directly.
var ErrHarnessTerminated = errors.New("harness: session was terminated")

// errInjectFailure is a sentinel error used by test fakes.
var errInjectFailure = errors.New("injected store failure")

// Harness is the stateful orchestrator. It owns phase state, drives Loop.Run(),
// evaluates gates, and persists every decision before mutating in-memory state.
//
// FSM states (Fix N1, V1):
//
//	hStateIdle          — ready for next prompt
//	hStateRunning       — Loop.Run is active
//	hStateAwaitingHuman — gate escalated, Decision Owner action required
//	hStateStopped       — Fix V1: terminal; Stop() was called; no further operations
type Harness struct {
	session        *Session
	store          SessionStore
	router         *PhaseRouter
	gate           *GateEngine
	accumulator    *EvidenceAccumulator
	mu             sync.Mutex
	ownerToken     string
	state          harnessState
	runID          uint64
	beforeToolCall func(name string, args json.RawMessage) error // Fix U2: nil = no-op

	// lastPhaseCompleted tracks whether the most recent RunPhase call detected
	// a completion_signal from the agent. True only when the agent explicitly
	// called the completion_signal tool; false when the agent responded with
	// text only (phase turn incomplete). Bridge consumers (runWithHarness) use
	// this to distinguish "phase done" from "agent still working".
	lastPhaseCompleted bool
}

// validateToken enforces owner-token authorization on all mutating methods.
// Fix A2: empty ownerToken = dev mode; any token passes (including empty).
func (h *Harness) validateToken(token string) error {
	if h.ownerToken == "" {
		return nil // dev mode — no token required
	}
	if token != h.ownerToken {
		return fmt.Errorf("unauthorized: invalid owner token")
	}
	return nil
}

// transitionTo persists a PhaseRecord durably BEFORE mutating in-memory state.
// Fix P2: if PersistPhaseRecord fails, session.Phase and accumulator are untouched.
// Fix A6: accumulator.Reset() is called after successful persist and phase mutation.
//
// recovery=true validates next against RecoveryNext; false validates against AllowedNext.
// Uses slices.Contains for membership check.
//
// Terminal phase completion: when current==next and AllowedNext is empty,
// the phase is a terminal (e.g., RoleEval). Self-transition is allowed so
// the completion record is persisted. The accumulator is NOT reset (same phase).
func (h *Harness) transitionTo(current, next Role, recovery bool) error {
	cfg := h.router.phaseMap[current]
	var allowed []Role
	if recovery {
		allowed = cfg.RecoveryNext
	} else {
		allowed = cfg.AllowedNext
	}

	// Terminal phase: current==next with empty AllowedNext is a valid completion.
	if !recovery && current == next && len(allowed) == 0 {
		// Persist terminal completion record but keep phase unchanged.
		if err := h.store.PersistPhaseRecord(h.session.ID, PhaseRecord{
			Phase:     current,
			NextPhase: next, // same as current — signals completion, not transition
			EndedAt:   time.Now(),
			Snapshot:  h.accumulator.Snapshot(current),
		}); err != nil {
			return fmt.Errorf("persist terminal phase record: %w", err)
		}
		// Do NOT reset accumulator — same phase continues.
		return nil
	}

	if !slices.Contains(allowed, next) {
		return fmt.Errorf("transition %s→%s not allowed (recovery=%v, allowed=%v)",
			current, next, recovery, allowed)
	}

	snapshot := h.accumulator.Snapshot(current)

	// Fix P2: persist FIRST — in-memory state is not touched until this succeeds.
	if err := h.store.PersistPhaseRecord(h.session.ID, PhaseRecord{
		Phase:     current,
		NextPhase: next,
		EndedAt:   time.Now(),
		Snapshot:  snapshot,
	}); err != nil {
		return fmt.Errorf("persist phase record: %w", err)
	}

	// Mutate in-memory only after durable commit.
	h.session.Phase = next
	h.accumulator.Reset() // Fix A6
	return nil
}

// RunPhase executes one phase-turn for the current session phase.
// Fix A2: requires ownerToken validation.
// Fix N1: FSM guard — only proceeds from hStateIdle.
// Fix P3: defer resets hStateRunning → hStateIdle if NOT escalated (hStateAwaitingHuman not reset).
// Fix N3: PersistTurnRecord called BEFORE gate check.
// Fix X2: TurnRecord.ToolCalls accumulated from "tool_call" events.
// Fix Y1: TurnRecord.ToolResults[].ID set from ev.ToolID (not ev.ToolName or empty string).
func (h *Harness) RunPhase(ctx context.Context, userPrompt, token string) error {
	if err := h.validateToken(token); err != nil {
		return err
	}

	// --- 1. FSM check + state transition under lock ---
	h.mu.Lock()
	if h.state != hStateIdle {
		h.mu.Unlock()
		return fmt.Errorf("harness busy: state=%d (expected idle=0)", h.state)
	}
	h.state = hStateRunning
	h.runID++
	currentRunID := h.runID
	phase := h.session.Phase
	h.lastPhaseCompleted = false // default: phase not completed until signal

	// Fix N3: build msgs from persisted TurnRecords — not from an in-memory buffer.
	msgs := h.session.MessagesFromTurnRecords()
	msgs = append(msgs, Message{Role: "user", Content: userPrompt})
	h.mu.Unlock()

	// Fix P3: reset FSM state in defer only when still hStateRunning.
	// Escalation path sets hStateAwaitingHuman under lock before returning — do not overwrite it.
	defer func() {
		h.mu.Lock()
		if h.runID == currentRunID && h.state == hStateRunning {
			h.state = hStateIdle
		}
		h.mu.Unlock()
	}()

	// Fix R2-2: completionFlag passed to tool closure explicitly.
	flag := &completionFlag{}
	// Fix U2: BeforeToolCall passed explicitly from h.beforeToolCall field.
	cfg, err := h.router.BuildLoopConfig(phase, h.accumulator, flag, h.beforeToolCall)
	if err != nil {
		return err
	}

	// --- 2. Run Loop (without holding h.mu) ---
	events, err := Run(ctx, msgs, cfg)
	if err != nil {
		return err
	}

	// Fix Q1: TurnRecord.ID = "sessionID:runID" — unique per run.
	turnRecord := TurnRecord{
		ID:        fmt.Sprintf("%s:%d", h.session.ID, currentRunID),
		Phase:     phase,
		UserMsg:   Message{Role: "user", Content: userPrompt},
		CreatedAt: time.Now(),
	}

	// Drain events, building the TurnRecord.
	for ev := range events {
		_ = h.store.PersistEvent(h.session.ID, ev)
		switch ev.Type {
		case "text_delta":
			turnRecord.AssistantText += ev.Delta
		case "tool_call":
			// Fix X2: accumulate all parallel tool calls from one assistant message.
			turnRecord.ToolCalls = append(turnRecord.ToolCalls, ev.ToolCalls...)
		case "tool_end":
			// Fix Y1: ev.ToolID correlates to original ToolCall.ID — required for API.
			// Fix P4: ev.ToolErr preserved in TurnRecord.
			// Fix R3/T1: ev.ToolArguments carries original call args (set by Run() from result.Arguments).
			turnRecord.ToolResults = append(turnRecord.ToolResults, ToolResult{
				ID:        ev.ToolID,
				Name:      ev.ToolName,
				Arguments: ev.ToolArguments,
				Output:    ev.ToolResult,
				Err:       ev.ToolErr,
			})
		case "error":
			return ev.Err
		}
	}

	// Fix N3: persist canonical TurnRecord BEFORE gate check.
	if err := h.store.PersistTurnRecord(h.session.ID, turnRecord); err != nil {
		return fmt.Errorf("persist turn record: %w", err)
	}

	// --- 3. Check completion_signal flag ---
	flag.mu.Lock()
	signaled := flag.signaled
	summary := flag.summary
	flag.mu.Unlock()

	if !signaled {
		return nil // agent did not finish phase — wait for next prompt
	}

	// completion_signal detected — mark phase as completed for bridge consumers.
	h.mu.Lock()
	h.lastPhaseCompleted = true
	h.mu.Unlock()

	// Fix N7: warn on empty summary (non-blocking).
	if summary == "" {
		_ = h.store.PersistEvent(h.session.ID, Event{Type: "warn",
			Delta: "completion_signal: empty summary"})
	}

	// --- 4. Gate check ---
	snap := h.accumulator.Snapshot(phase)
	result := h.gate.Evaluate(ctx, snap)

	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.store.PersistGateResult(h.session.ID, result); err != nil {
		return fmt.Errorf("persist gate result: %w", err)
	}

	if result.Escalated {
		// Fix N2: persist PendingDecision so ApproveGate/Rollback can validate decisionID.
		decision := PendingDecision{
			DecisionID:     fmt.Sprintf("%s-run%d", h.session.ID, currentRunID),
			RunID:          currentRunID,
			Phase:          phase,
			GateResult:     result,
			AllowedActions: []string{"approve", "rollback", "stop"},
		}
		if err := h.store.PersistDecision(h.session.ID, decision); err != nil {
			return fmt.Errorf("persist decision: %w", err)
		}
		h.state = hStateAwaitingHuman // Fix N1: FSM → awaiting human
		h.session.EmitEvent(Event{Type: "human_gate", Delta: decision.DecisionID})
		return nil
	}

	// Gate passed — transition to next phase.
	return h.transitionTo(phase, h.router.NextPhase(phase), false)
}

// ApproveGate is called by the Decision Owner to approve an escalated gate.
// Fix A2: requires ownerToken.
// Fix P1: transitionTo (persists PhaseRecord) FIRST; ClearDecision only after success.
//
//	If transition fails: state stays awaiting_human, decision intact → caller can retry.
func (h *Harness) ApproveGate(ctx context.Context, decisionID, token string) error {
	if err := h.validateToken(token); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.state != hStateAwaitingHuman {
		return fmt.Errorf("no pending gate decision (state=%d)", h.state)
	}
	if err := h.store.ValidateDecision(h.session.ID, decisionID); err != nil {
		return fmt.Errorf("invalid decisionID: %w", err)
	}

	// Atomically persist the phase transition AND clear the pending decision.
	// Both operations execute in a single SQLite transaction, so a crash at any
	// point leaves either both done or neither done — no partial state.
	if err := h.store.TransitionAndClearDecision(h.session.ID, decisionID, PhaseRecord{
		Phase:     h.session.Phase,
		NextPhase: h.router.NextPhase(h.session.Phase),
		StartedAt: time.Now().Add(-time.Second), // approximate
		EndedAt:   time.Now(),
	}); err != nil {
		return fmt.Errorf("atomic approve transition: %w", err)
	}
	h.session.Phase = h.router.NextPhase(h.session.Phase)
	h.accumulator.Reset()
	h.state = hStateIdle
	return nil
}

// Rollback is called by the Decision Owner to roll back to RecoveryNext.
// Fix A2: requires ownerToken.
func (h *Harness) Rollback(ctx context.Context, decisionID, token string) error {
	if err := h.validateToken(token); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.state != hStateAwaitingHuman {
		return fmt.Errorf("no pending gate decision (state=%d)", h.state)
	}
	if err := h.store.ValidateDecision(h.session.ID, decisionID); err != nil {
		return fmt.Errorf("invalid decisionID: %w", err)
	}

	// Atomically persist the recovery transition AND clear the pending decision.
	if err := h.store.TransitionAndClearDecision(h.session.ID, decisionID, PhaseRecord{
		Phase:     h.session.Phase,
		NextPhase: h.router.RecoveryPhase(h.session.Phase),
		StartedAt: time.Now().Add(-time.Second),
		EndedAt:   time.Now(),
	}); err != nil {
		return fmt.Errorf("atomic rollback transition: %w", err)
	}
	h.session.Phase = h.router.RecoveryPhase(h.session.Phase)
	h.accumulator.Reset()
	h.state = hStateIdle
	return nil
}

// Stop terminates the session. After Stop, RunPhase/ApproveGate/Rollback always fail.
// Fix U1: PersistPhaseRecord (NextPhase="") is called FIRST; errors propagate; state not mutated.
// Fix S1: if state=awaiting_human, LoadDecision → ClearDecision before setting hStateStopped.
// Fix V1: sets hStateStopped (terminal), not hStateIdle.
func (h *Harness) Stop(ctx context.Context, token string) error {
	if err := h.validateToken(token); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.state == hStateRunning {
		return fmt.Errorf("phase in progress; cancel ctx first to stop")
	}

	// Fix U1: durable-first — persist terminal record BEFORE clearing decision or mutating state.
	// NextPhase="" signals Stop() was called — RestoreHarness detects this and returns error.
	if err := h.store.PersistPhaseRecord(h.session.ID, PhaseRecord{
		Phase:     h.session.Phase,
		NextPhase: "", // terminal marker
		EndedAt:   time.Now(),
		Snapshot:  h.accumulator.Snapshot(h.session.Phase),
	}); err != nil {
		return fmt.Errorf("persist terminal record: %w", err)
	}

	// Fix S1: clear pending decision if awaiting_human.
	// Runs AFTER terminal record is persisted — if ClearDecision fails, the terminal record
	// already exists; next RestoreHarness detects NextPhase="" and treats session as stopped.
	if h.state == hStateAwaitingHuman {
		pending, err := h.store.LoadDecision(h.session.ID)
		if err != nil {
			return fmt.Errorf("load decision for stop: %w", err)
		}
		if pending != nil {
			if err := h.store.ClearDecision(h.session.ID, pending.DecisionID); err != nil {
				return fmt.Errorf("clear decision for stop: %w", err)
			}
		}
	}

	// Fix V1: terminal state — prevents reuse.
	h.state = hStateStopped
	h.session.EmitEvent(Event{Type: "session_stopped"})
	return nil
}

// IsAwaitingHuman reports whether the harness is waiting for a human gate decision.
// Returns true if state is hStateAwaitingHuman (gate escalated, Decision Owner action required).
// The harnessState type is intentionally unexported — callers use this accessor.
func (h *Harness) IsAwaitingHuman() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state == hStateAwaitingHuman
}

// Phase returns the current session phase. Thread-safe.
// Bridge consumers use this to pass the actual agentloop phase into bookkeeping
// instead of hardcoding "build".
func (h *Harness) Phase() Role {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.session.Phase
}

// LastPhaseCompleted reports whether the most recent RunPhase call finished
// with a completion_signal from the agent. Returns false when the agent
// responded with text but did not call completion_signal (phase turn incomplete).
// Bridge consumers use this to distinguish ResultStatusSuccess (phase done)
// from ResultStatusNeedsInput (agent still working).
func (h *Harness) LastPhaseCompleted() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastPhaseCompleted
}

// RestoreHarness creates a Harness from a previously persisted session.
//
// Key fixes applied:
//
//	Fix A1 (v7): detects pending decision → sets hStateAwaitingHuman.
//	Fix S2 (v8): ownerToken restored so validateToken works after restart.
//	Fix V2 (v10): beforeToolCall restored so pre-hook works after restart.
//	Fix W1/W1' (v11): if len(History)>0 && session.Phase=="" → session was terminated by Stop(); return error.
//	Fix W3 (v11): h.runID = len(turnRecords); if pending decision exists, h.runID = pending.RunID.
//	Fix X3 (v12): session.Phase derived from last PhaseRecord.NextPhase by RecoverSession.
func RestoreHarness(
	sessionID, ownerToken string,
	store SessionStore,
	router *PhaseRouter,
	gate *GateEngine,
	beforeToolCall func(name string, args json.RawMessage) error, // nil = no-op; Fix V2
) (*Harness, error) {
	session, err := RecoverSession(sessionID, store)
	if err != nil {
		return nil, err
	}

	h := &Harness{
		session:        session,
		store:          store,
		router:         router,
		gate:           gate,
		accumulator:    NewEvidenceAccumulator(),
		state:          hStateIdle,
		ownerToken:     ownerToken,                       // Fix S2: restore token
		beforeToolCall: beforeToolCall,                   // Fix V2: restore hook
		runID:          uint64(len(session.turnRecords)), // Fix W3: continue after existing records
	}

	// Fix W1/W1' (v11): detect terminal session.
	// session.Phase is derived from last PhaseRecord.NextPhase by RecoverSession (Fix X3).
	// len(History)>0 means at least one transition happened; Phase=="" means the last was Stop().
	// len(History)==0 means no transitions — new session, Phase=RoleDiscover from Persist.
	if len(session.History) > 0 && session.Phase == "" {
		return nil, fmt.Errorf("session %s: %w", sessionID, ErrHarnessTerminated)
	}

	// Fix A1 (v7): check for pending decision — may need to restore awaiting_human state.
	pending, err := store.LoadDecision(sessionID)
	if err != nil {
		return nil, fmt.Errorf("load decision for restore: %w", err)
	}
	if pending != nil {
		h.state = hStateAwaitingHuman
		// Fix W3: pending decision carries exact runID for this gate decision.
		// Overrides the len(turnRecords) default — the pending decision's runID is authoritative.
		h.runID = pending.RunID
	}

	return h, nil
}
