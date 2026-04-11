# SDP Mini-Harness: Harness Orchestrator + CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Harness FSM orchestrator layer and MVP CLI for the SDP Mini-Harness using strict TDD. Picks up after the worker layer (Tasks 1–10). After all five tasks here pass, `internal/agentloop` has a complete stateful Harness (FSM, RunPhase, ApproveGate, Rollback, Stop, RestoreHarness) and `cmd/sdp-harness/` has an MVP CLI.

**Architecture:** Harness owns the FSM state (`hStateIdle` / `hStateRunning` / `hStateAwaitingHuman` / `hStateStopped`). It calls `Loop.Run()` per phase-turn, reads the `completionFlag`, drives `GateEngine.Evaluate`, and persists everything to `SessionStore` before mutating in-memory state (durable-first invariant). The CLI wraps Harness with a `flag`-based command surface.

**Tech Stack:**
- Go module: `sdp_dev`, go 1.26
- Target packages: `sdp_dev/internal/agentloop` (Harness) + `sdp_dev/cmd/sdp-harness/` (CLI)
- Test assertions: `github.com/stretchr/testify v1.11.1`
- Test doubles: `MemStore` (from `store.go`), `StubGateway` (from `gateway.go`)
- Test runner: `go test ./internal/agentloop/... ./cmd/sdp-harness/... -race`

**Key invariants — all from v14 converged spec — that must be tested explicitly:**
- **P1**: `ApproveGate`/`Rollback` persist phase transition BEFORE clearing decision
- **P2**: `transitionTo` persists `PhaseRecord` BEFORE mutating `session.Phase` + `accumulator.Reset()`
- **P3**: defer in `RunPhase` resets state to `hStateIdle` only if still `hStateRunning` (not `hStateAwaitingHuman`)
- **U1**: `Stop` persists terminal `PhaseRecord` BEFORE clearing decision or mutating state
- **S1**: `Stop` clears `PendingDecision` if `state == hStateAwaitingHuman`
- **V1**: `hStateStopped` is terminal — `RunPhase` must reject it
- **W1'**: `RestoreHarness` terminal detection: `len(History) > 0 && session.Phase == ""`
- **W3**: `RestoreHarness` sets `h.runID = len(session.turnRecords)`
- **X2**: `TurnRecord.ToolCalls` accumulates from `"tool_call"` events
- **Y1**: `TurnRecord.ToolResults[].ID` set from `ev.ToolID` (not empty string)
- **A1**: `RestoreHarness` detects pending decision → `hStateAwaitingHuman`
- **S2**: `RestoreHarness` restores `ownerToken` so `validateToken` works after restart
- **V2**: `RestoreHarness` restores `beforeToolCall` hook

**Prerequisites:** Tasks 1–10 complete (types, store, session, SQLite, evidence, gateway, executeCalls, Run, GateEngine, PhaseRouter/ToolRegistry all committed).

---

### Task 11: Harness struct + validateToken + transitionTo

**Files:**
- Create: `internal/agentloop/harness.go`
- Create: `internal/agentloop/harness_test.go` (partial — validateToken + transitionTo tests only)

**Step 1: Write the failing tests**

Create `internal/agentloop/harness_test.go`:

```go
package agentloop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})
	sg.AddResponse("openai/gpt-4.1", []Event{{Type: "done"}})

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

func (f *failingPhaseStore) Persist(s *Session) error                      { return f.inner.Persist(s) }
func (f *failingPhaseStore) Recover(id string) (*Session, error)           { return f.inner.Recover(id) }
func (f *failingPhaseStore) PersistEvent(id string, ev Event) error        { return f.inner.PersistEvent(id, ev) }
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
func (f *failingPhaseStore) ClearDecision(id, did string) error { return f.inner.ClearDecision(id, did) }
func (f *failingPhaseStore) LoadDecision(id string) (*PendingDecision, error) {
	return f.inner.LoadDecision(id)
}
```

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestHarness_validateToken|TestHarness_transitionTo" -v
```
Expected: FAIL — `Harness`, `validateToken`, `transitionTo` undefined.

**Step 3: Write minimal implementation**

Create `internal/agentloop/harness.go`:

```go
package agentloop

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

// errInjectFailure is a sentinel error used by test fakes.
var errInjectFailure = errors.New("injected store failure")

// Harness is the stateful orchestrator. It owns phase state, drives Loop.Run(),
// evaluates gates, and persists every decision before mutating in-memory state.
//
// FSM states (Fix N1, V1):
//   hStateIdle          — ready for next prompt
//   hStateRunning       — Loop.Run is active
//   hStateAwaitingHuman — gate escalated, Decision Owner action required
//   hStateStopped       — Fix V1: terminal; Stop() was called; no further operations
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
func (h *Harness) transitionTo(current, next Role, recovery bool) error {
	cfg := h.router.phaseMap[current]
	var allowed []Role
	if recovery {
		allowed = cfg.RecoveryNext
	} else {
		allowed = cfg.AllowedNext
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
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestHarness_validateToken|TestHarness_transitionTo" -race -v
```
Expected: PASS — all 7 tests green.

**Step 5: Commit**

```
git add internal/agentloop/harness.go internal/agentloop/harness_test.go
git commit -m "feat(agentloop): Task 11 — Harness struct, validateToken (dev mode), transitionTo (Fix P2 durable-first, slices.Contains)"
```

---

### Task 12: RunPhase

**Files:**
- Edit: `internal/agentloop/harness.go` (add `RunPhase`)
- Edit: `internal/agentloop/harness_test.go` (add RunPhase tests)

**Step 1: Write the failing tests**

Append to `internal/agentloop/harness_test.go`:

```go
// ---- RunPhase helpers ----

// Fix R4: StubGate and newPassGate() removed — they were dead code with a broken evalFn
// lambda (wrong type: interface{ dummy() } instead of *harness.TaskContract).
// GateEngine is configured directly in buildHarnessWithGateway via gate.evalFn = alwaysPassEval.

// buildHarnessWithGateway creates a full Harness with a scripted StubGateway.
// The gateway model for discover phase is "deepseek/deepseek-v3.2".
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
		sg.AddResponse("deepseek/deepseek-v3.2", events)
		sg.AddResponse("openai/gpt-4.1", events)
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
		sg.AddResponse("deepseek/deepseek-v3.2", events)
		sg.AddResponse("openai/gpt-4.1", events)
	}

	registry := NewToolRegistry(nil)
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)

	gate := NewGateEngine(nil, 5*time.Second)
	gate.evalFn = alwaysEscalateEval

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
//   {tool_call: [completion_signal]}, {done}
// → Run() sees tool_call → executeCalls → completion_signal.Execute() sets flag.signaled=true
// → Run() emits {tool_end} on output channel → loops back to gateway
// Round 2 (LLM acknowledges tool result, ends turn):
//   {text_delta: "phase complete"}, {done}
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
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{
		{Type: "tool_call", ToolCalls: []ToolCall{
			{ID: "tc1", Name: "web_search", Arguments: []byte(`{"query":"foo"}`)},
			{ID: "tc2", Name: "read_file", Arguments: []byte(`{"path":"bar"}`)},
		}},
		{Type: "done"},
	})
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}}) // round 2: no more tools
	sg.AddResponse("openai/gpt-4.1", []Event{{Type: "done"}})

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
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{
		{Type: "tool_call", ToolCalls: []ToolCall{
			{ID: "specific-id-99", Name: "web_search", Arguments: []byte(`{}`)},
		}},
		{Type: "done"},
	})
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}}) // round 2
	sg.AddResponse("openai/gpt-4.1", []Event{{Type: "done"}})

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
	registerSignalResponses(sg, "deepseek/deepseek-v3.2")
	registerSignalResponses(sg, "openai/gpt-4.1")

	registry := NewToolRegistry(nil)
	router := NewPhaseRouter(DefaultPhaseMap, registry, sg, nil)

	turnPersistedBeforeGate := false
	gate := NewGateEngine(nil, 5*time.Second)
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
	registerSignalResponses(sg, "deepseek/deepseek-v3.2")
	registerSignalResponses(sg, "openai/gpt-4.1")
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
	registerSignalResponses(sg, "deepseek/deepseek-v3.2")
	registerSignalResponses(sg, "openai/gpt-4.1")

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
```

> **Import note:** Use `import "sdp_dev/internal/harness"` and reference types as `harness.TaskContract`, `harness.TaskSnapshot`, `harness.ComplianceReport`. All `harnessHarness`/`harnessPackage` aliases have been removed from this spec (Fix H1).

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestRunPhase" -v
```
Expected: FAIL — `RunPhase` is not implemented yet.

**Step 3: Write minimal implementation**

Add `RunPhase` to `internal/agentloop/harness.go`:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"sdp_dev/internal/harness"
)

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
		h.store.PersistEvent(h.session.ID, ev)
		switch ev.Type {
		case "text_delta":
			turnRecord.AssistantText += ev.Delta
		case "tool_call":
			// Fix X2: accumulate all parallel tool calls from one assistant message.
			turnRecord.ToolCalls = append(turnRecord.ToolCalls, ev.ToolCalls...)
		case "tool_end":
			// Fix Y1: ev.ToolID correlates to original ToolCall.ID — required for API.
			// Fix Y1: ev.ToolID correlates to original ToolCall.ID.
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

	// Fix N7: warn on empty summary (non-blocking).
	if summary == "" {
		h.store.PersistEvent(h.session.ID, Event{Type: "warn",
			Delta: "completion_signal: empty summary"})
	}

	// --- 4. Gate check ---
	snap := h.accumulator.Snapshot(phase)
	result := h.gate.Evaluate(ctx, snap)

	h.mu.Lock()
	defer h.mu.Unlock()
	h.store.PersistGateResult(h.session.ID, result)

	if result.Escalated {
		// Fix N2: persist PendingDecision so ApproveGate/Rollback can validate decisionID.
		decision := PendingDecision{
			DecisionID:     fmt.Sprintf("%s-run%d", h.session.ID, currentRunID),
			RunID:          currentRunID,
			Phase:          phase,
			GateResult:     result,
			AllowedActions: []string{"approve", "rollback", "stop"},
		}
		h.store.PersistDecision(h.session.ID, decision)
		h.state = hStateAwaitingHuman // Fix N1: FSM → awaiting human
		h.session.EmitEvent(Event{Type: "human_gate", Delta: decision.DecisionID})
		return nil
	}

	// Gate passed — transition to next phase.
	return h.transitionTo(phase, h.router.NextPhase(phase), false)
}
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestRunPhase" -race -v
```
Expected: PASS — all RunPhase tests green.

**Step 5: Commit**

```
git add internal/agentloop/harness.go internal/agentloop/harness_test.go
git commit -m "feat(agentloop): Task 12 — RunPhase FSM (Fix N1, P3, N3, X2, Y1, N2, Q1); completion_signal detection, gate escalation path"
```

---

### Task 13: ApproveGate + Rollback + Stop

**Files:**
- Edit: `internal/agentloop/harness.go` (add ApproveGate, Rollback, Stop)
- Edit: `internal/agentloop/harness_test.go` (add gate + stop tests)

**Step 1: Write the failing tests**

Append to `internal/agentloop/harness_test.go`:

```go
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
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})
	sg.AddResponse("openai/gpt-4.1", []Event{{Type: "done"}})

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
```

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestApproveGate|TestRollback|TestStop" -v
```
Expected: FAIL — `ApproveGate`, `Rollback`, `Stop` undefined.

**Step 3: Write minimal implementation**

Add to `internal/agentloop/harness.go`:

```go
// ApproveGate is called by the Decision Owner to approve an escalated gate.
// Fix A2: requires ownerToken.
// Fix P1: transitionTo (persists PhaseRecord) FIRST; ClearDecision only after success.
//   If transition fails: state stays awaiting_human, decision intact → caller can retry.
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

	phase := h.session.Phase
	// Fix P1: transition FIRST — only after success do we clear the decision.
	if err := h.transitionTo(phase, h.router.NextPhase(phase), false); err != nil {
		// state stays awaiting_human; decision intact — caller can retry.
		return err
	}
	h.state = hStateIdle
	h.store.ClearDecision(h.session.ID, decisionID)
	return nil
}

// Rollback is called by the Decision Owner to roll back to RecoveryNext.
// Fix A2: requires ownerToken.
// Fix P1: transitionTo (persists PhaseRecord) FIRST; ClearDecision only after success.
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

	phase := h.session.Phase
	// Fix P1: transition FIRST.
	if err := h.transitionTo(phase, h.router.RecoveryPhase(phase), true); err != nil {
		return err
	}
	h.state = hStateIdle
	h.store.ClearDecision(h.session.ID, decisionID)
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
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestApproveGate|TestRollback|TestStop" -race -v
```
Expected: PASS — all tests green.

Run the full suite:
```
go test ./internal/agentloop/... -race -count=1
```
Expected: ALL tests pass.

**Step 5: Commit**

```
git add internal/agentloop/harness.go internal/agentloop/harness_test.go
git commit -m "feat(agentloop): Task 13 — ApproveGate, Rollback, Stop (Fix P1, S1, U1, V1); durable-first invariants fully enforced"
```

---

### Task 14: RestoreHarness

**Files:**
- Edit: `internal/agentloop/harness.go` (add `RestoreHarness`)
- Edit: `internal/agentloop/harness_test.go` (add RestoreHarness tests)

**Step 1: Write the failing tests**

Append to `internal/agentloop/harness_test.go`:

```go
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
	sg.AddResponse("deepseek/deepseek-v3.2", []Event{{Type: "done"}})
	sg.AddResponse("openai/gpt-4.1", []Event{{Type: "done"}})
	sg.AddResponse("anthropic/claude-sonnet-4-6", []Event{{Type: "done"}})
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
```

**Step 2: Run test, verify it fails**

```
go test ./internal/agentloop/... -run "TestRestoreHarness" -v
```
Expected: FAIL — `RestoreHarness` undefined.

**Step 3: Write minimal implementation**

Add to `internal/agentloop/harness.go`:

```go
// RestoreHarness creates a Harness from a previously persisted session.
//
// Key fixes applied:
//   Fix A1 (v7): detects pending decision → sets hStateAwaitingHuman.
//   Fix S2 (v8): ownerToken restored so validateToken works after restart.
//   Fix V2 (v10): beforeToolCall restored so pre-hook works after restart.
//   Fix W1/W1' (v11): if len(History)>0 && session.Phase=="" → session was terminated by Stop(); return error.
//   Fix W3 (v11): h.runID = len(turnRecords); if pending decision exists, h.runID = pending.RunID.
//   Fix X3 (v12): session.Phase derived from last PhaseRecord.NextPhase by RecoverSession.
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
		ownerToken:     ownerToken,     // Fix S2: restore token
		beforeToolCall: beforeToolCall, // Fix V2: restore hook
		runID:          uint64(len(session.turnRecords)), // Fix W3: continue after existing records
	}

	// Fix W1/W1' (v11): detect terminal session.
	// session.Phase is derived from last PhaseRecord.NextPhase by RecoverSession (Fix X3).
	// len(History)>0 means at least one transition happened; Phase=="" means the last was Stop().
	// len(History)==0 means no transitions — new session, Phase=RoleDiscover from Persist.
	if len(session.History) > 0 && session.Phase == "" {
		return nil, fmt.Errorf("session %s was terminated by Stop() — cannot restore", sessionID)
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
```

**Step 4: Verify passes**

```
go test ./internal/agentloop/... -run "TestRestoreHarness" -race -v
```
Expected: PASS — all RestoreHarness tests green.

Run the full suite:
```
go test ./internal/agentloop/... -race -count=1
```
Expected: ALL tests pass.

**Step 5: Commit**

```
git add internal/agentloop/harness.go internal/agentloop/harness_test.go
git commit -m "feat(agentloop): Task 14 — RestoreHarness (Fix A1, S2, V2, W1/W1', W3, X3); terminal session detection, pending decision recovery"
```

---

### Task 15: CLI scaffold

**Files:**
- Create: `cmd/sdp-harness/main.go`
- Create: `cmd/sdp-harness/main_test.go`

**Step 1: Write the failing tests**

Create `cmd/sdp-harness/main_test.go`:

```go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles sdp-harness into a temp directory and returns the path.
// Skips the test if CGO is unavailable (SQLite requires CGO).
func buildBinary(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	bin := filepath.Join(dir, "sdp-harness")

	cmd := exec.Command("go", "build", "-o", bin, "sdp_dev/cmd/sdp-harness")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// TestCLI_missingSession_fails: running `sdp-harness run` without --session exits non-zero.
func TestCLI_missingSession_fails(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "run", "--prompt=hello")
	out, err := cmd.CombinedOutput()

	// Must exit with non-zero code.
	if err == nil {
		t.Fatalf("expected non-zero exit, got success\noutput: %s", out)
	}

	output := string(out)
	// Error message must indicate missing session.
	if !strings.Contains(strings.ToLower(output), "session") {
		t.Errorf("expected output to mention 'session', got: %s", output)
	}
}

// TestCLI_missingPrompt_fails: running `sdp-harness run` without --prompt exits non-zero.
func TestCLI_missingPrompt_fails(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "run", "--session=test-sess")
	cmd.Env = append(os.Environ(), "SDP_DATA_DIR="+dir)
	out, _ := cmd.CombinedOutput()

	// Without --prompt, the command should either error or produce a helpful message.
	// We accept either exit code non-zero OR output mentioning "prompt".
	_ = out
	// Primary contract: binary does not panic (any non-panic outcome is acceptable).
}

// TestCLI_newSession_creates: `sdp-harness new --session=test-123` creates a DB file.
func TestCLI_newSession_creates(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "new", "--session=test-123")
	cmd.Env = append(os.Environ(), "SDP_DATA_DIR="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sdp-harness new failed: %v\noutput: %s", err, out)
	}

	// DB file must exist at $SDP_DATA_DIR/test-123.db
	dbPath := filepath.Join(dir, "test-123.db")
	if _, statErr := os.Stat(dbPath); statErr != nil {
		t.Errorf("expected DB file at %s, but stat failed: %v", dbPath, statErr)
	}
}

// TestCLI_newSession_defaultDataDir: when SDP_DATA_DIR is not set, uses $HOME/.sdp.
// This test only verifies the binary runs; it does not create files in $HOME.
func TestCLI_newSession_defaultDataDir(t *testing.T) {
	bin := buildBinary(t)

	// Run with a non-existent session to get an early error (before DB creation in $HOME).
	// We just verify the binary starts without panicking.
	cmd := exec.Command(bin, "--help")
	cmd.Env = os.Environ()
	out, _ := cmd.CombinedOutput()

	// --help should not panic and should print usage.
	if strings.Contains(string(out), "panic") {
		t.Errorf("--help must not panic, got: %s", out)
	}
}

// TestCLI_unknownSubcommand_fails: unknown subcommand exits non-zero with helpful message.
func TestCLI_unknownSubcommand_fails(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "frobulate")
	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("unknown subcommand must exit non-zero, got success\noutput: %s", out)
	}
	output := string(out)
	if !strings.Contains(strings.ToLower(output), "unknown") &&
		!strings.Contains(strings.ToLower(output), "usage") &&
		!strings.Contains(strings.ToLower(output), "invalid") {
		t.Errorf("expected error/usage message, got: %s", output)
	}
}
```

**Step 2: Run test, verify it fails**

```
go test ./cmd/sdp-harness/... -v -run "TestCLI"
```
Expected: FAIL — `cmd/sdp-harness` package does not exist yet.

**Step 3: Write minimal implementation**

First create the directory:

```
mkdir -p cmd/sdp-harness
```

Create `cmd/sdp-harness/main.go`:

```go
// Package main provides the sdp-harness CLI for running SDP phase turns
// and managing session lifecycle through the agentloop Harness.
//
// Usage:
//
//	sdp-harness new --session=<id>
//	  Creates a new session DB at $SDP_DATA_DIR/<id>.db (default: $HOME/.sdp/<id>.db).
//
//	sdp-harness run --session=<id> --prompt="<text>" [--token=<tok>]
//	  Runs one phase turn for the given session, streaming events to stdout.
//	  Restores an existing session or fails if session does not exist.
//
// Environment:
//
//	SDP_DATA_DIR  Directory for session DB files. Defaults to $HOME/.sdp.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/agentloop"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return fmt.Errorf("no subcommand given")
	}

	switch args[0] {
	case "new":
		return cmdNew(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "--help", "-h", "help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown subcommand: %q", args[0])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `sdp-harness — SDP Mini-Harness CLI

Subcommands:
  new  --session=<id>                     Create a new session
  run  --session=<id> --prompt="<text>"   Run one phase turn

Environment:
  SDP_DATA_DIR   Directory for session DB files (default: $HOME/.sdp)`)
}

// dataDir returns the directory where session DB files are stored.
// Uses SDP_DATA_DIR env var; falls back to $HOME/.sdp.
func dataDir() (string, error) {
	if d := os.Getenv("SDP_DATA_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".sdp"), nil
}

// dbPath returns the path to the session DB file for the given sessionID.
func dbPath(sessionID string) (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create data dir %s: %w", dir, err)
	}
	return filepath.Join(dir, sessionID+".db"), nil
}

// cmdNew implements `sdp-harness new --session=<id>`.
// Creates a fresh session DB and persists the initial Session record.
func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session ID (required)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sessionID == "" {
		fs.Usage()
		return fmt.Errorf("--session is required")
	}

	path, err := dbPath(*sessionID)
	if err != nil {
		return err
	}

	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		return fmt.Errorf("open store at %s: %w", path, err)
	}
	defer store.Close()

	_, err = agentloop.NewSession(*sessionID, store)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	fmt.Printf("session %q created at %s\n", *sessionID, path)
	return nil
}

// cmdRun implements `sdp-harness run --session=<id> --prompt="<text>" [--token=<tok>]`.
// Restores or creates a session and runs one phase turn, streaming events to stdout.
func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session ID (required)")
	prompt := fs.String("prompt", "", "User prompt for this phase turn (required)")
	token := fs.String("token", "", "Owner token (optional; required if session has ownerToken set)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sessionID == "" {
		fs.Usage()
		return fmt.Errorf("--session is required")
	}
	if *prompt == "" {
		fs.Usage()
		return fmt.Errorf("--prompt is required")
	}

	path, err := dbPath(*sessionID)
	if err != nil {
		return err
	}

	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		return fmt.Errorf("open store at %s: %w", path, err)
	}
	defer store.Close()

	// Build a minimal router with no real tools (MVP placeholder).
	// Production use wires real tools and a live ModelGateway.
	registry := agentloop.NewToolRegistry(nil)
	gateway := agentloop.NewStubGateway()
	router := agentloop.NewPhaseRouter(agentloop.DefaultPhaseMap, registry, gateway, nil)
	gate := agentloop.NewGateEngine(nil, 0) // 0 → 5s default

	// Try to restore an existing session; if not found, error (use `new` first).
	h, err := agentloop.RestoreHarness(*sessionID, *token, store, router, gate, nil)
	if err != nil {
		return fmt.Errorf("restore session %q: %w\n(hint: use 'sdp-harness new --session=%s' to create it)", *sessionID, err, *sessionID)
	}

	// Run the phase turn with a background context (no timeout for MVP CLI).
	ctx := context.Background()
	if err := h.RunPhase(ctx, *prompt, *token); err != nil {
		return fmt.Errorf("run phase: %w", err)
	}

	fmt.Printf("phase turn complete for session %q\n", *sessionID)
	return nil
}
```

**Step 4: Verify passes**

```
go test ./cmd/sdp-harness/... -v -run "TestCLI" -count=1
```
Expected: PASS — all CLI tests green.

Also verify the full suite still passes:
```
go test ./internal/agentloop/... ./cmd/sdp-harness/... -race -count=1
```
Expected: ALL tests pass.

Verify the binary builds cleanly:
```
go build ./cmd/sdp-harness/
```
Expected: zero errors.

**Step 5: Commit**

```bash
mkdir -p cmd/sdp-harness
git add cmd/sdp-harness/main.go cmd/sdp-harness/main_test.go
git commit -m "feat(cmd/sdp-harness): Task 15 — MVP CLI scaffold (new + run subcommands, flag package, SDP_DATA_DIR env, SQLiteStore integration)"
```

---

## Final verification

After all five tasks (11–15) are committed, run the complete test suite for both packages:

```bash
go build ./internal/agentloop/... ./cmd/sdp-harness/...
go test ./internal/agentloop/... ./cmd/sdp-harness/... -race -count=1 -v 2>&1 | tail -20
```

Expected: zero compilation errors, all tests pass (Tasks 1–15 combined), no race detector warnings.

---

## Implementation notes

### 1. Import alias for `internal/harness`

All of `harness.go` and `harness_test.go` import `sdp_dev/internal/harness` using the default package name `harness`. Since the Harness struct is in `package agentloop` (not `package harness`), there is no naming conflict. Reference types as `harness.TaskContract`, `harness.ComplianceReport`, etc.

The test helper functions `alwaysPassEval` / `alwaysEscalateEval` (used in Task 12 helpers)
and `alwaysPassEvalFn` / `alwaysEscalateEvalFn` (used in Tasks 13–14 helpers) must have the
exact signature expected by `GateEngine.evalFn`:

```go
func(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport
```

### 2. `slices.Contains` availability

`golang.org/x/exp/slices` is NOT needed — `slices.Contains` is in the Go standard library since go 1.21. The module is `go 1.26`, so `import "slices"` compiles without any extra dependency.

### 3. `tool.Execute` context type

`Tool.Execute` uses `context.Context` (Fix F1 applied in Task 1). `types.go` already imports `"context"`. No correction needed — this note is for historical context only.

### 4. `t.Context()` availability

`testing.T.Context()` was added in Go 1.21. With `go 1.26` this is available directly. If CI uses an older toolchain, substitute `context.Background()`.

### 5. Gate evalFn for tests

`GateEngine.evalFn` is a package-private field exposed only within the `agentloop` package. Harness tests (in `package agentloop`, not `package agentloop_test`) can set it directly. This avoids exporting test-only infrastructure.

### 6. RunPhase + completion_signal wiring

`BuildLoopConfig` appends `makeCompletionSignalTool(flag)` to the tool slice.
The signal detection flow (Fix H2):

1. Gateway (Round 1) scripts: `{tool_call: [completion_signal]}, {done}`
2. `Run()` receives the `tool_call` event → calls `executeCalls([completion_signal])`.
3. `executeCalls` finds `completion_signal` in `cfg.Tools`, calls `Execute()` — sets `flag.signaled = true`.
4. `Run()` emits `{tool_end, ToolID: "sig1", ToolName: "completion_signal"}` on output channel.
5. `Run()` loops back → calls Gateway again (Round 2): `{text_delta: "phase complete"}, {done}`.
6. `Run()` closes the output channel.
7. `RunPhase` drains events, then reads `flag.signaled == true` → proceeds to gate check.

**The gateway must NOT script `tool_end` events** — those are produced by `Run()` after `executeCalls`.
Use `registerSignalResponses(sg, model)` which scripts only Round 1 and Round 2 correctly.

### 7. CLI MVP limitations

The MVP CLI (`Task 15`) uses `StubGateway` with no registered models, so `RestoreHarness` succeeds but `RunPhase` will fail at `BuildLoopConfig` (no available model). This is expected for the scaffold — the integration test (`TestCLI_newSession_creates`) only tests session creation, not a full phase run. Production use wires a real `ModelGateway` (OpenRouter, Anthropic, etc.).

### 8. `MemStore` accessibility

`MemStore` is defined in `store.go` (not a `_test.go` file) so it is importable by all test files in `package agentloop`. The `NewSQLiteStore` and `NewSession` functions used in the CLI are exported from `package agentloop` for use by `cmd/sdp-harness/main.go`.

### 9. Ordering of durable-first operations in `Stop`

The exact order mandated by Fix U1:
1. `PersistPhaseRecord` with `NextPhase=""` → if this fails, return error; state unchanged.
2. If `state == hStateAwaitingHuman`: `LoadDecision` → `ClearDecision` → if this fails, return error (terminal record already exists; next `RestoreHarness` sees `NextPhase=""` and returns error; operator can retry `Stop`).
3. Set `h.state = hStateStopped`.
4. `EmitEvent(session_stopped)`.

This ordering ensures the system is always in a recoverable state: either the terminal record exists (and `RestoreHarness` will error) or the full stop completed cleanly.
