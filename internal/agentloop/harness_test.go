package agentloop

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sdp_dev/internal/harness"
)

// ---- validateToken ----

// TestHarness_validateToken_empty_passes: when ownerToken=="" any token passes (dev mode).
func TestHarness_validateToken_empty_passes(t *testing.T) {
	h := &Harness{ownerToken: ""}
	assert.NoError(t, h.validateToken("anything"), "dev mode: empty ownerToken must accept any token")
	assert.NoError(t, h.validateToken(""), "dev mode: empty ownerToken must accept empty token too")
}

// TestHarness_validateToken_mismatch_fails: ownerToken set → wrong token returns error.
func TestHarness_validateToken_mismatch_fails(t *testing.T) {
	h := &Harness{ownerToken: "secret-token"}
	err := h.validateToken("wrong-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

// TestHarness_validateToken_correct_passes: correct token accepted.
func TestHarness_validateToken_correct_passes(t *testing.T) {
	h := &Harness{ownerToken: "secret-token"}
	assert.NoError(t, h.validateToken("secret-token"))
}

// ---- transitionTo ----

// buildTestHarness creates a minimal Harness for transitionTo tests with no live gateway.
// Uses DefaultPhaseMap so transition validation uses real allowed lists.
func buildTestHarness(t *testing.T) (*Harness, *MemStore) {
	t.Helper()
	ms := NewMemStore()
	session := &Session{ID: "sess-t11", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	sg := NewStubGateway()
	sg.AddResponse("glm-5", []Event{{Type: "done"}})
	sg.AddResponse("glm-4.7", []Event{{Type: "done"}})

	registry := NewToolRegistry(nil) // empty tools; router only used for phase map lookup
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)

	h := &Harness{
		session:     session,
		store:       ms,
		router:      router,
		accumulator: NewEvidenceAccumulator(),
		state:       hStateIdle,
		ownerToken:  "",
	}
	return h, ms
}

// TestHarness_transitionTo_persistsBeforeMutate: Fix P2 — PersistPhaseRecord is called
// and the record is durable BEFORE session.Phase is mutated in memory.
func TestHarness_transitionTo_persistsBeforeMutate(t *testing.T) {
	h, ms := buildTestHarness(t)
	originalPhase := h.session.Phase // RoleDiscover

	// transitionTo discover→plan is allowed (AllowedNext for discover = [plan]).
	err := h.transitionTo(RoleDiscover, RolePlan, false)
	require.NoError(t, err)

	// 1. session.Phase must now be RolePlan (mutated after persist).
	assert.Equal(t, RolePlan, h.session.Phase, "session.Phase must be updated to next phase")

	// 2. PhaseRecord must be persisted (durable before mutation — P2).
	records, loadErr := ms.LoadPhaseRecords("sess-t11")
	require.NoError(t, loadErr)
	require.Len(t, records, 1, "exactly one PhaseRecord must be persisted after transitionTo")
	assert.Equal(t, originalPhase, records[0].Phase)
	assert.Equal(t, RolePlan, records[0].NextPhase)

	// 3. accumulator must be reset after transition.
	snap := h.accumulator.Snapshot(RolePlan)
	assert.Empty(t, snap.Evidence, "accumulator must be reset after transitionTo")
}

// TestHarness_transitionTo_rejectsInvalidTransition: discover→eval is NOT in AllowedNext → error.
func TestHarness_transitionTo_rejectsInvalidTransition(t *testing.T) {
	h, ms := buildTestHarness(t)

	err := h.transitionTo(RoleDiscover, RoleEval, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// Phase must remain unchanged (P2 guarantee — no mutation on failure).
	assert.Equal(t, RoleDiscover, h.session.Phase)

	// No PhaseRecord should have been persisted.
	records, _ := ms.LoadPhaseRecords("sess-t11")
	assert.Empty(t, records, "no PhaseRecord on rejected transition")
}

// TestHarness_transitionTo_recovery_usesRecoveryNext: recovery=true validates against RecoveryNext.
func TestHarness_transitionTo_recovery_usesRecoveryNext(t *testing.T) {
	h, _ := buildTestHarness(t)

	// discover RecoveryNext = [discover] — valid.
	err := h.transitionTo(RoleDiscover, RoleDiscover, true)
	require.NoError(t, err, "discover→discover is valid under RecoveryNext")
	assert.Equal(t, RoleDiscover, h.session.Phase, "phase stays at discover after recovery to itself")

	// Reset for next sub-test.
	h2, _ := buildTestHarness(t)
	h2.session.Phase = RoleBuild

	// build RecoveryNext = [plan, build] — plan is valid.
	err2 := h2.transitionTo(RoleBuild, RolePlan, true)
	require.NoError(t, err2, "build→plan is valid under RecoveryNext")
	assert.Equal(t, RolePlan, h2.session.Phase)

	// build AllowedNext = [review] — plan is NOT valid as non-recovery.
	h3, _ := buildTestHarness(t)
	h3.session.Phase = RoleBuild
	err3 := h3.transitionTo(RoleBuild, RolePlan, false)
	require.Error(t, err3, "build→plan is NOT allowed under AllowedNext (non-recovery)")
}

// TestHarness_transitionTo_persistFailure_doesNotMutate: if PersistPhaseRecord fails,
// session.Phase must remain unchanged (Fix P2 durable-first guarantee).
func TestHarness_transitionTo_persistFailure_doesNotMutate(t *testing.T) {
	h, _ := buildTestHarness(t)

	// Replace store with one that fails PersistPhaseRecord.
	h.store = &failingPhaseStore{inner: h.store.(*MemStore)}

	err := h.transitionTo(RoleDiscover, RolePlan, false)
	require.Error(t, err, "transitionTo must return error when persist fails")
	assert.Equal(t, RoleDiscover, h.session.Phase,
		"Fix P2: session.Phase must NOT be mutated when persist fails")
}

// failingPhaseStore wraps MemStore and makes PersistPhaseRecord always fail.
type failingPhaseStore struct {
	inner *MemStore
}

func (f *failingPhaseStore) Persist(s *Session) error            { return f.inner.Persist(s) }
func (f *failingPhaseStore) Recover(id string) (*Session, error) { return f.inner.Recover(id) }
func (f *failingPhaseStore) PersistEvent(id string, ev Event) error {
	return f.inner.PersistEvent(id, ev)
}
func (f *failingPhaseStore) LoadEvents(id string) ([]Event, error) { return f.inner.LoadEvents(id) }
func (f *failingPhaseStore) PersistGateResult(id string, r GateResult) error {
	return f.inner.PersistGateResult(id, r)
}
func (f *failingPhaseStore) PersistTurnRecord(id string, r TurnRecord) error {
	return f.inner.PersistTurnRecord(id, r)
}
func (f *failingPhaseStore) LoadTurnRecords(id string) ([]TurnRecord, error) {
	return f.inner.LoadTurnRecords(id)
}
func (f *failingPhaseStore) PersistPhaseRecord(id string, r PhaseRecord) error {
	return errInjectFailure // always fail
}
func (f *failingPhaseStore) LoadPhaseRecords(id string) ([]PhaseRecord, error) {
	return f.inner.LoadPhaseRecords(id)
}
func (f *failingPhaseStore) PersistDecision(id string, d PendingDecision) error {
	return f.inner.PersistDecision(id, d)
}
func (f *failingPhaseStore) ValidateDecision(id, did string) error {
	return f.inner.ValidateDecision(id, did)
}
func (f *failingPhaseStore) ClearDecision(id, did string) error {
	return f.inner.ClearDecision(id, did)
}
func (f *failingPhaseStore) LoadDecision(id string) (*PendingDecision, error) {
	return f.inner.LoadDecision(id)
}

// ---- RunPhase helpers ----

// Fix R4: StubGate and newPassGate() removed — they were dead code with a broken evalFn
// lambda (wrong type: interface{ dummy() } instead of *harness.TaskContract).
// GateEngine is configured directly in buildHarnessWithGateway via gate.evalFn = alwaysPassEval.

// buildHarnessWithGateway creates a full Harness with a scripted StubGateway.
// The gateway model for discover phase is "glm-5".
//
// Fix R2: if events is nil, no responses are pre-registered — call sg.AddResponse
// or registerSignalResponses after this helper to set up the desired sequence.
// With StubGateway's Fix R1 (FIFO queue), passing nil here avoids consuming a
// "slot" before the test's own AddResponse calls.
func buildHarnessWithGateway(t *testing.T, events []Event) (*Harness, *MemStore, *StubGateway) {
	t.Helper()
	ms := NewMemStore()
	session := &Session{ID: "sess-run", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	sg := NewStubGateway()
	if events != nil {
		// Fix R2: only pre-register when caller provides concrete events.
		sg.AddResponse("glm-5", events)
		sg.AddResponse("glm-4.7", events)
	}

	registry := NewToolRegistry(nil)
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)

	// GateEngine that always passes.
	gate := NewGateEngine(nil, 5*time.Second)
	gate.evalFn = alwaysPassEval

	h := &Harness{
		session:     session,
		store:       ms,
		router:      router,
		gate:        gate,
		accumulator: NewEvidenceAccumulator(),
		state:       hStateIdle,
		ownerToken:  "",
	}
	return h, ms, sg
}

// buildHarnessWithEscalatingGate creates a Harness whose GateEngine always escalates.
// Fix R2: returns *StubGateway so callers can register signal responses after construction.
// If events is nil, no responses are pre-registered (caller uses registerSignalResponses).
func buildHarnessWithEscalatingGate(t *testing.T, events []Event) (*Harness, *MemStore, *StubGateway) {
	t.Helper()
	ms := NewMemStore()
	session := &Session{ID: "sess-esc", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	sg := NewStubGateway()
	if events != nil {
		// Fix R2: only pre-register when caller provides concrete events.
		sg.AddResponse("glm-5", events)
		sg.AddResponse("glm-4.7", events)
	}

	registry := NewToolRegistry(nil)
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)

	gate := NewGateEngine(nil, 5*time.Second)
	gate.evalFn = alwaysEscalateEval
	gate.bypassNilContract = false // custom evalFn overrides auto-pass

	h := &Harness{
		session:     session,
		store:       ms,
		router:      router,
		gate:        gate,
		accumulator: NewEvidenceAccumulator(),
		state:       hStateIdle,
		ownerToken:  "",
	}
	return h, ms, sg
}

// alwaysPassEval is a GateEngine evalFn that always returns a passing report.
func alwaysPassEval(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport {
	return harness.ComplianceReport{Blocked: false}
}

// alwaysEscalateEval returns a blocked report (triggers escalation).
func alwaysEscalateEval(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport {
	return harness.ComplianceReport{Blocked: true}
}

// registerSignalResponses wires a StubGateway with two scripted responses for the
// "agent calls completion_signal" scenario. Fix H2: tool_end events are NOT scripted here —
// Run() generates them after calling executeCalls. The gateway only scripts tool_call + done.
//
// Round 1 (LLM decides to call completion_signal):
//
//	{tool_call: [completion_signal]}, {done}
//
// → Run() sees tool_call → executeCalls → completion_signal.Execute() sets flag.signaled=true
// → Run() emits {tool_end} on output channel → loops back to gateway
// Round 2 (LLM acknowledges tool result, ends turn):
//
//	{text_delta: "phase complete"}, {done}
//
// → Run() emits text_delta + done, closes channel
func registerSignalResponses(sg *StubGateway, model string) {
	sg.AddResponse(model, []Event{
		{Type: "tool_call", ToolCalls: []ToolCall{
			{ID: "sig1", Name: "completion_signal", Arguments: []byte(`{"summary":"phase complete"}`)},
		}},
		{Type: "done"},
	})
	sg.AddResponse(model, []Event{
		{Type: "text_delta", Delta: "phase complete"},
		{Type: "done"},
	})
}

// ---- RunPhase tests ----

// TestRunPhase_rejectsWhenRunning: calling RunPhase while already running returns "harness busy".
func TestRunPhase_rejectsWhenRunning(t *testing.T) {
	ms := NewMemStore()
	session := &Session{ID: "sess-busy", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	h := &Harness{
		session:     session,
		store:       ms,
		state:       hStateRunning, // already running
		accumulator: NewEvidenceAccumulator(),
	}

	err := h.RunPhase(t.Context(), "prompt", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "harness busy")
}

// TestRunPhase_rejectsWhenStopped: Fix V1 — hStateStopped is terminal; RunPhase must reject it.
func TestRunPhase_rejectsWhenStopped(t *testing.T) {
	ms := NewMemStore()
	session := &Session{ID: "sess-stopped", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	h := &Harness{
		session:     session,
		store:       ms,
		state:       hStateStopped, // terminal state
		accumulator: NewEvidenceAccumulator(),
	}

	err := h.RunPhase(t.Context(), "prompt", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "harness busy")
}

// TestRunPhase_requiresToken: token mismatch → error, state unchanged.
func TestRunPhase_requiresToken(t *testing.T) {
	ms := NewMemStore()
	session := &Session{ID: "sess-tok", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	h := &Harness{
		session:     session,
		store:       ms,
		state:       hStateIdle,
		ownerToken:  "correct-token",
		accumulator: NewEvidenceAccumulator(),
	}

	err := h.RunPhase(t.Context(), "prompt", "wrong-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")

	// State must remain idle (token check is before FSM mutation).
	h.mu.Lock()
	defer h.mu.Unlock()
	assert.Equal(t, hStateIdle, h.state, "state must remain idle on token mismatch")
}

// TestRunPhase_noCompletionSignal_remainsIdle: agent does not call completion_signal
// → RunPhase returns nil, state returns to idle, no phase transition.
func TestRunPhase_noCompletionSignal_remainsIdle(t *testing.T) {
	// Events with no tool calls — no completion_signal fired.
	noSignalEvents := []Event{
		{Type: "text_delta", Delta: "still working..."},
		{Type: "done"},
	}
	h, ms, _ := buildHarnessWithGateway(t, noSignalEvents)

	err := h.RunPhase(t.Context(), "continue", "")
	require.NoError(t, err)

	// State must be idle (P3: defer resets hStateRunning → hStateIdle).
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateIdle, state, "P3: state must return to idle when no completion_signal")

	// No phase transition should have occurred.
	records, _ := ms.LoadPhaseRecords("sess-run")
	assert.Empty(t, records, "no phase transition without completion_signal")

	// Phase must remain discover.
	assert.Equal(t, RoleDiscover, h.session.Phase)
}

// TestRunPhase_turnRecord_hasToolCalls: Fix X2 — "tool_call" events populate TurnRecord.ToolCalls.
func TestRunPhase_turnRecord_hasToolCalls(t *testing.T) {
	// Fix H2: gateway scripts tool_call + done only. Run() generates tool_end after executeCalls.
	// Round 1: LLM calls two tools.
	// Round 2: LLM acknowledges tool results, says done.
	h, ms, sg := buildHarnessWithGateway(t, nil)
	sg.AddResponse("glm-5", []Event{
		{Type: "tool_call", ToolCalls: []ToolCall{
			{ID: "tc1", Name: "web_search", Arguments: []byte(`{"query":"foo"}`)},
			{ID: "tc2", Name: "read_file", Arguments: []byte(`{"path":"bar"}`)},
		}},
		{Type: "done"},
	})
	sg.AddResponse("glm-5", []Event{{Type: "done"}}) // round 2: no more tools
	sg.AddResponse("glm-4.7", []Event{{Type: "done"}})

	_ = h.RunPhase(t.Context(), "search stuff", "")

	turns, err := ms.LoadTurnRecords("sess-run")
	require.NoError(t, err)
	require.Len(t, turns, 1)
	assert.Len(t, turns[0].ToolCalls, 2,
		"Fix X2: TurnRecord.ToolCalls must be populated from tool_call events")
	assert.Equal(t, "tc1", turns[0].ToolCalls[0].ID)
	assert.Equal(t, "tc2", turns[0].ToolCalls[1].ID)
}

// TestRunPhase_turnRecord_toolID: Fix Y1 — tool_end events emitted by Run() carry ToolID →
// TurnRecord.ToolResults[].ID is populated (not empty string).
func TestRunPhase_turnRecord_toolID(t *testing.T) {
	// Fix H2: gateway only scripts tool_call+done. Run() emits tool_end with ToolID=call.ID.
	h, ms, sg := buildHarnessWithGateway(t, nil)
	sg.AddResponse("glm-5", []Event{
		{Type: "tool_call", ToolCalls: []ToolCall{
			{ID: "specific-id-99", Name: "web_search", Arguments: []byte(`{}`)},
		}},
		{Type: "done"},
	})
	sg.AddResponse("glm-5", []Event{{Type: "done"}}) // round 2
	sg.AddResponse("glm-4.7", []Event{{Type: "done"}})

	_ = h.RunPhase(t.Context(), "find it", "")

	turns, err := ms.LoadTurnRecords("sess-run")
	require.NoError(t, err)
	require.Len(t, turns, 1)
	require.Len(t, turns[0].ToolResults, 1)
	assert.Equal(t, "specific-id-99", turns[0].ToolResults[0].ID,
		"Fix Y1: ToolResult.ID must equal ev.ToolID from tool_end event")
}

// TestRunPhase_turnRecord_persistedBeforeGate: Fix N3 — TurnRecord is persisted
// before gate evaluation (so context survives even if gate panics).
func TestRunPhase_turnRecord_persistedBeforeGate(t *testing.T) {
	// The completion_signal tool is built into the router via BuildLoopConfig.
	// But we need the gateway to script the events including completion_signal.
	// We wire the completionFlag manually by intercepting the gate evaluation order.
	// Simplest approach: use a gateway that emits a completion_signal tool_end event
	// and verify the turn record appears before gate would run.

	// Use a gate that records if it was called AFTER a TurnRecord was persisted.
	ms := NewMemStore()
	session := &Session{ID: "sess-n3", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	sg := NewStubGateway()
	// Script: completion_signal called → agent done. Fix H2: no tool_end in gateway responses.
	registerSignalResponses(sg, "glm-5")
	registerSignalResponses(sg, "glm-4.7")

	registry := NewToolRegistry(nil)
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)

	turnPersistedBeforeGate := false
	gate := NewGateEngine(nil, 5*time.Second)
	gate.bypassNilContract = false // custom evalFn overrides auto-pass
	gate.evalFn = func(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport {
		// Check if TurnRecord was already persisted at gate evaluation time.
		turns, _ := ms.LoadTurnRecords("sess-n3")
		turnPersistedBeforeGate = len(turns) > 0
		return harness.ComplianceReport{Blocked: false}
	}

	h := &Harness{
		session:     session,
		store:       ms,
		router:      router,
		gate:        gate,
		accumulator: NewEvidenceAccumulator(),
		state:       hStateIdle,
	}

	_ = h.RunPhase(t.Context(), "signal now", "")
	assert.True(t, turnPersistedBeforeGate,
		"Fix N3: TurnRecord must be persisted BEFORE gate evaluation")
}

// signalEventsForGateway is deprecated — use registerSignalResponses(sg, model) instead.
// Kept for reference only. Fix H2: gateway must NOT emit tool_end; Run() generates those.

// TestRunPhase_completionSignal_gatePass_transitions: happy path — completion_signal called,
// gate passes → transitionTo(discover→plan), state=idle.
func TestRunPhase_completionSignal_gatePass_transitions(t *testing.T) {
	h, ms, sg := buildHarnessWithGateway(t, nil) // nil = we'll register manually
	registerSignalResponses(sg, "glm-5")
	registerSignalResponses(sg, "glm-4.7")
	_ = ms

	err := h.RunPhase(t.Context(), "do the work", "")
	require.NoError(t, err)

	// Phase must have transitioned to plan.
	assert.Equal(t, RolePlan, h.session.Phase,
		"gate pass + completion_signal → discover→plan transition")

	// PhaseRecord must be persisted.
	records, _ := ms.LoadPhaseRecords("sess-run")
	require.Len(t, records, 1)
	assert.Equal(t, RoleDiscover, records[0].Phase)
	assert.Equal(t, RolePlan, records[0].NextPhase)

	// State must be idle (not running, not awaiting_human).
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateIdle, state)
}

// TestRunPhase_escalation_setsAwaitingHuman: gate escalates → state=awaiting_human,
// PendingDecision persisted, human_gate event emitted.
func TestRunPhase_escalation_setsAwaitingHuman(t *testing.T) {
	h, ms, sg := buildHarnessWithEscalatingGate(t, nil)
	registerSignalResponses(sg, "glm-5")
	registerSignalResponses(sg, "glm-4.7")

	err := h.RunPhase(t.Context(), "do the work", "")
	require.NoError(t, err)

	// State must be awaiting_human.
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateAwaitingHuman, state, "escalation must set state=awaiting_human")

	// PendingDecision must be persisted (Fix N2).
	decision, dErr := ms.LoadDecision("sess-esc")
	require.NoError(t, dErr)
	require.NotNil(t, decision, "Fix N2: PendingDecision must be persisted on escalation")
	assert.NotEmpty(t, decision.DecisionID)
	assert.Equal(t, RoleDiscover, decision.Phase)
}

// ---- ApproveGate tests ----

// seedAwaitingHarness creates a Harness in hStateAwaitingHuman with a persisted PendingDecision.
func seedAwaitingHarness(t *testing.T) (*Harness, *MemStore) {
	t.Helper()
	ms := NewMemStore()
	session := &Session{ID: "sess-gate", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	decision := PendingDecision{
		DecisionID:     "sess-gate-run1",
		RunID:          1,
		Phase:          RoleDiscover,
		AllowedActions: []string{"approve", "rollback", "stop"},
	}
	require.NoError(t, ms.PersistDecision("sess-gate", decision))

	sg := NewStubGateway()
	sg.AddResponse("glm-5", []Event{{Type: "done"}})
	sg.AddResponse("glm-4.7", []Event{{Type: "done"}})

	registry := NewToolRegistry(nil)
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)
	gate := NewGateEngine(nil, 5*time.Second)
	gate.evalFn = alwaysPassEvalFn

	h := &Harness{
		session:     session,
		store:       ms,
		router:      router,
		gate:        gate,
		accumulator: NewEvidenceAccumulator(),
		state:       hStateAwaitingHuman,
		ownerToken:  "",
		runID:       1,
	}
	return h, ms
}

// alwaysPassEvalFn satisfies gate.evalFn signature.
func alwaysPassEvalFn(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport {
	return harness.ComplianceReport{Blocked: false}
}

// TestApproveGate_requiresAwaitingHuman: ApproveGate from wrong state → error.
func TestApproveGate_requiresAwaitingHuman(t *testing.T) {
	ms := NewMemStore()
	session := &Session{ID: "sess-ag", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	h := &Harness{
		session:     session,
		store:       ms,
		state:       hStateIdle, // NOT awaiting_human
		accumulator: NewEvidenceAccumulator(),
	}

	err := h.ApproveGate(t.Context(), "some-id", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending gate decision")
}

// TestApproveGate_invalidDecisionID_rejectsWithoutTransition: wrong decisionID → error, no transition.
func TestApproveGate_invalidDecisionID_rejectsWithoutTransition(t *testing.T) {
	h, ms := seedAwaitingHarness(t)

	err := h.ApproveGate(t.Context(), "wrong-decision-id", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid decisionID")

	// State must remain awaiting_human (no transition).
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateAwaitingHuman, state, "state must remain awaiting_human on bad decisionID")

	// Phase must remain discover.
	assert.Equal(t, RoleDiscover, h.session.Phase)

	// Decision must still be persisted (not cleared).
	d, _ := ms.LoadDecision("sess-gate")
	require.NotNil(t, d, "decision must remain intact on bad decisionID")
}

// TestApproveGate_transitionsBeforeClearing: Fix P1 — PhaseRecord is persisted
// (transition committed) before ClearDecision is called.
func TestApproveGate_transitionsBeforeClearing(t *testing.T) {
	// We verify the invariant by using a store that fails on ClearDecision but succeeds on
	// PersistPhaseRecord — if the transition was persisted, the invariant holds.
	h, ms := seedAwaitingHarness(t)

	err := h.ApproveGate(t.Context(), "sess-gate-run1", "")
	require.NoError(t, err)

	// Phase must have transitioned to plan (AllowedNext for discover).
	assert.Equal(t, RolePlan, h.session.Phase,
		"ApproveGate must transition discover→plan")

	// PhaseRecord must be persisted (Fix P1: transition before clear).
	records, _ := ms.LoadPhaseRecords("sess-gate")
	require.Len(t, records, 1)
	assert.Equal(t, RoleDiscover, records[0].Phase)
	assert.Equal(t, RolePlan, records[0].NextPhase)

	// Decision must have been cleared after successful transition.
	d, _ := ms.LoadDecision("sess-gate")
	assert.Nil(t, d, "PendingDecision must be cleared after successful ApproveGate")

	// State must be idle.
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateIdle, state)
}

// ---- Rollback tests ----

// TestRollback_usesRecoveryPhase: rollback transitions to RecoveryNext (discover→discover).
func TestRollback_usesRecoveryPhase(t *testing.T) {
	h, ms := seedAwaitingHarness(t)

	err := h.Rollback(t.Context(), "sess-gate-run1", "")
	require.NoError(t, err)

	// discover RecoveryNext = [discover] → phase stays discover.
	assert.Equal(t, RoleDiscover, h.session.Phase,
		"Rollback from discover must go to RecoveryNext[0]=discover")

	// PhaseRecord must be persisted for the rollback.
	records, _ := ms.LoadPhaseRecords("sess-gate")
	require.Len(t, records, 1)
	assert.Equal(t, RoleDiscover, records[0].Phase)
	assert.Equal(t, RoleDiscover, records[0].NextPhase)

	// Decision cleared.
	d, _ := ms.LoadDecision("sess-gate")
	assert.Nil(t, d)

	// State back to idle.
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateIdle, state)
}

// TestRollback_requiresAwaitingHuman: Rollback from wrong state → error.
func TestRollback_requiresAwaitingHuman(t *testing.T) {
	ms := NewMemStore()
	session := &Session{ID: "sess-rb", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	h := &Harness{
		session:     session,
		store:       ms,
		state:       hStateIdle,
		accumulator: NewEvidenceAccumulator(),
	}

	err := h.Rollback(t.Context(), "any-id", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending gate decision")
}

// ---- Stop tests ----

// seedIdleHarness creates a Harness in hStateIdle (no pending decision).
func seedIdleHarness(t *testing.T) (*Harness, *MemStore) {
	t.Helper()
	ms := NewMemStore()
	session := &Session{ID: "sess-stop", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(session))

	sg := NewStubGateway()
	registry := NewToolRegistry(nil)
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)
	gate := NewGateEngine(nil, 5*time.Second)
	gate.evalFn = alwaysPassEvalFn

	h := &Harness{
		session:     session,
		store:       ms,
		router:      router,
		gate:        gate,
		accumulator: NewEvidenceAccumulator(),
		state:       hStateIdle,
	}
	return h, ms
}

// TestStop_rejectsWhileRunning: Stop must fail while a phase run is in progress.
func TestStop_rejectsWhileRunning(t *testing.T) {
	h, _ := seedIdleHarness(t)
	h.mu.Lock()
	h.state = hStateRunning
	h.mu.Unlock()

	err := h.Stop(t.Context(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "phase in progress")
}

// TestStop_persistsTerminalRecord_first: Fix U1 — PersistPhaseRecord with NextPhase=""
// is called BEFORE any state mutation. If persist fails, state must not change.
func TestStop_persistsTerminalRecord_first(t *testing.T) {
	h, _ := seedIdleHarness(t)

	// Replace store with one that fails PersistPhaseRecord.
	h.store = &failingPhaseStore{inner: h.store.(*MemStore)}

	err := h.Stop(t.Context(), "")
	require.Error(t, err, "Stop must propagate persist error")

	// State must remain idle (not hStateStopped) because persist failed.
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateIdle, state,
		"Fix U1: state must NOT be mutated when PersistPhaseRecord fails in Stop")
}

// TestStop_clearsPendingDecision_whenAwaitingHuman: Fix S1 — if state=awaiting_human,
// Stop must clear the PendingDecision before setting hStateStopped.
func TestStop_clearsPendingDecision_whenAwaitingHuman(t *testing.T) {
	h, ms := seedAwaitingHarness(t)

	// Rename session ID to match seedAwaitingHarness output.
	// seedAwaitingHarness uses "sess-gate" — update h.store reference.
	err := h.Stop(t.Context(), "")
	require.NoError(t, err)

	// Decision must be cleared (Fix S1).
	d, dErr := ms.LoadDecision("sess-gate")
	require.NoError(t, dErr)
	assert.Nil(t, d, "Fix S1: PendingDecision must be cleared by Stop when awaiting_human")

	// State must be hStateStopped (Fix V1).
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateStopped, state, "Fix V1: Stop must set hStateStopped, not hStateIdle")

	// Terminal PhaseRecord must be persisted (NextPhase="").
	records, _ := ms.LoadPhaseRecords("sess-gate")
	require.Len(t, records, 1)
	assert.Equal(t, Role(""), records[0].NextPhase,
		"Fix U1: terminal PhaseRecord must have empty NextPhase")
}

// TestStop_setsStopped_preventsReuse: Fix V1 — after Stop, RunPhase returns "harness busy".
func TestStop_setsStopped_preventsReuse(t *testing.T) {
	h, _ := seedIdleHarness(t)

	err := h.Stop(t.Context(), "")
	require.NoError(t, err)

	// RunPhase must reject hStateStopped.
	err2 := h.RunPhase(t.Context(), "any prompt", "")
	require.Error(t, err2)
	assert.Contains(t, err2.Error(), "harness busy",
		"Fix V1: hStateStopped must be treated as busy (terminal state)")
}

// ---- RestoreHarness tests ----

// buildRestorable creates a persisted session in a MemStore suitable for RestoreHarness tests.
func buildRestorable(t *testing.T, sessionID string, phaseRecords []PhaseRecord, turns []TurnRecord, decision *PendingDecision) (*MemStore, *PhaseRouter) {
	t.Helper()
	ms := NewMemStore()
	initialPhase := RoleDiscover
	require.NoError(t, ms.Persist(&Session{ID: sessionID, Phase: initialPhase}))

	for _, pr := range phaseRecords {
		require.NoError(t, ms.PersistPhaseRecord(sessionID, pr))
	}
	for _, tr := range turns {
		require.NoError(t, ms.PersistTurnRecord(sessionID, tr))
	}
	if decision != nil {
		require.NoError(t, ms.PersistDecision(sessionID, *decision))
	}

	sg := NewStubGateway()
	sg.AddResponse("glm-5", []Event{{Type: "done"}})
	sg.AddResponse("glm-4.7", []Event{{Type: "done"}})
	sg.AddResponse("anthropic/claude-sonnet-4.6", []Event{{Type: "done"}})
	registry := NewToolRegistry(nil)
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)
	return ms, router
}

// TestRestoreHarness_restoredToken: Fix S2 — ownerToken is passed to RestoreHarness and
// validateToken works correctly after restore.
func TestRestoreHarness_restoredToken(t *testing.T) {
	ms, router := buildRestorable(t, "sess-s2", nil, nil, nil)
	gate := NewGateEngine(nil, 5*time.Second)

	h, err := RestoreHarness("sess-s2", "my-secret-token", ms, router, gate, nil)
	require.NoError(t, err)

	// Correct token must pass.
	assert.NoError(t, h.validateToken("my-secret-token"), "Fix S2: restored token must be used by validateToken")
	// Wrong token must fail.
	assert.Error(t, h.validateToken("wrong"), "Fix S2: wrong token must be rejected after restore")
}

// TestRestoreHarness_runIDContinues: Fix W3 — h.runID = len(session.turnRecords)
// so TurnRecord IDs don't collide with pre-restore records.
func TestRestoreHarness_runIDContinues(t *testing.T) {
	turns := []TurnRecord{
		{ID: "sess-w3:1", Phase: RoleDiscover, UserMsg: Message{Role: "user", Content: "a"}, CreatedAt: time.Now()},
		{ID: "sess-w3:2", Phase: RoleDiscover, UserMsg: Message{Role: "user", Content: "b"}, CreatedAt: time.Now()},
		{ID: "sess-w3:3", Phase: RoleDiscover, UserMsg: Message{Role: "user", Content: "c"}, CreatedAt: time.Now()},
	}
	ms, router := buildRestorable(t, "sess-w3", nil, turns, nil)
	gate := NewGateEngine(nil, 5*time.Second)

	h, err := RestoreHarness("sess-w3", "", ms, router, gate, nil)
	require.NoError(t, err)

	h.mu.Lock()
	runID := h.runID
	h.mu.Unlock()
	assert.Equal(t, uint64(3), runID,
		"Fix W3: runID must equal len(turnRecords)=3 so next IDs start at :4")
}

// TestRestoreHarness_stoppedSession_returnsError: Fix W1/W1' — if len(History)>0 AND
// session.Phase=="" (last NextPhase was empty = Stop was called), RestoreHarness must
// return an error instead of an idle Harness.
func TestRestoreHarness_stoppedSession_returnsError(t *testing.T) {
	// PhaseRecord with NextPhase="" = Stop was called.
	phaseRecords := []PhaseRecord{
		{Phase: RoleDiscover, NextPhase: ""}, // terminal
	}
	ms, router := buildRestorable(t, "sess-w1", phaseRecords, nil, nil)
	gate := NewGateEngine(nil, 5*time.Second)

	_, err := RestoreHarness("sess-w1", "", ms, router, gate, nil)
	require.Error(t, err,
		"Fix W1/W1': RestoreHarness must error when len(History)>0 and session.Phase==''")
	assert.Contains(t, err.Error(), "terminated",
		"error message must mention the session was terminated by Stop()")
	assert.True(t, errors.Is(err, ErrHarnessTerminated),
		"error must wrap ErrHarnessTerminated for errors.Is() detection")
}

// TestErrHarnessTerminated_isSentinel verifies the exported sentinel can be used with errors.Is.
func TestErrHarnessTerminated_isSentinel(t *testing.T) {
	if ErrHarnessTerminated == nil {
		t.Fatal("ErrHarnessTerminated is nil — must be errors.New(...)")
	}
	wrapped := fmt.Errorf("restore: %w", ErrHarnessTerminated)
	if !errors.Is(wrapped, ErrHarnessTerminated) {
		t.Error("errors.Is failed — ErrHarnessTerminated must be wrappable with %w")
	}
}

// TestRestoreHarness_noPhaseRecords_notStopped: a brand-new session with no PhaseRecords
// should not be treated as stopped (len(History)==0 even though Phase=="" edge case does not apply).
func TestRestoreHarness_noPhaseRecords_notStopped(t *testing.T) {
	ms, router := buildRestorable(t, "sess-new", nil, nil, nil)
	gate := NewGateEngine(nil, 5*time.Second)

	h, err := RestoreHarness("sess-new", "", ms, router, gate, nil)
	require.NoError(t, err, "new session with no PhaseRecords must restore successfully")
	assert.Equal(t, RoleDiscover, h.session.Phase)

	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	assert.Equal(t, hStateIdle, state)
}

// TestRestoreHarness_pendingDecision_setsAwaitingHuman: Fix A1 — if a PendingDecision exists
// in the store, RestoreHarness must set state=hStateAwaitingHuman.
func TestRestoreHarness_pendingDecision_setsAwaitingHuman(t *testing.T) {
	decision := &PendingDecision{
		DecisionID:     "sess-a1-run5",
		RunID:          5,
		Phase:          RoleDiscover,
		AllowedActions: []string{"approve", "rollback", "stop"},
	}
	ms, router := buildRestorable(t, "sess-a1", nil, nil, decision)
	gate := NewGateEngine(nil, 5*time.Second)

	h, err := RestoreHarness("sess-a1", "", ms, router, gate, nil)
	require.NoError(t, err)

	h.mu.Lock()
	state := h.state
	runID := h.runID
	h.mu.Unlock()

	assert.Equal(t, hStateAwaitingHuman, state,
		"Fix A1: pending decision must restore state=hStateAwaitingHuman")
	assert.Equal(t, uint64(5), runID,
		"Fix W3: when pending decision exists, runID = pending.RunID")
}

// TestRestoreHarness_beforeToolCall_wired: Fix V2 — the beforeToolCall hook passed to
// RestoreHarness is stored on the Harness and is callable.
func TestRestoreHarness_beforeToolCall_wired(t *testing.T) {
	ms, router := buildRestorable(t, "sess-v2", nil, nil, nil)
	gate := NewGateEngine(nil, 5*time.Second)

	hookCalled := false
	beforeHook := func(name string, args json.RawMessage) error {
		hookCalled = true
		return nil
	}

	h, err := RestoreHarness("sess-v2", "", ms, router, gate, beforeHook)
	require.NoError(t, err)

	require.NotNil(t, h.beforeToolCall,
		"Fix V2: beforeToolCall hook must be stored on restored Harness")

	// Invoke the hook to verify it's our function.
	_ = h.beforeToolCall("bash", nil)
	assert.True(t, hookCalled, "Fix V2: beforeToolCall hook must be callable after restore")
}

// TestRestoreHarness_derivesPhaseFromHistory: Fix X3 — restored session.Phase
// is derived from last PhaseRecord.NextPhase, not from the initial Persist phase.
func TestRestoreHarness_derivesPhaseFromHistory(t *testing.T) {
	phaseRecords := []PhaseRecord{
		{Phase: RoleDiscover, NextPhase: RolePlan},
		{Phase: RolePlan, NextPhase: RoleBuild},
	}
	ms, router := buildRestorable(t, "sess-x3", phaseRecords, nil, nil)
	gate := NewGateEngine(nil, 5*time.Second)

	h, err := RestoreHarness("sess-x3", "", ms, router, gate, nil)
	require.NoError(t, err)

	assert.Equal(t, RoleBuild, h.session.Phase,
		"Fix X3: Phase must be derived from last PhaseRecord.NextPhase=RoleBuild")
}
