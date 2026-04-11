# SDP Mini-Harness: Foundation Layer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the five foundation-layer packages for the SDP Mini-Harness (`internal/agentloop`) using strict TDD. Each task produces a tested, committed slice of the foundation. After all five tasks pass, the agentloop package has: all core types, the SessionStore interface + MemStore fake, Session/TurnRecord/MessagesFromTurnRecords with X2+Z1 fixes, a SQLite SessionStore, and EvidenceAccumulator.

**Architecture:** Loop (stateless worker) + Harness (stateful orchestrator) + PhaseRouter + GateEngine + SessionStore. This plan covers only the foundation layer — types, store, session, SQLite persistence, and evidence accumulation. Loop/Harness/PhaseRouter/GateEngine are separate tasks.

**Tech Stack:**
- Go module: `sdp_dev`, go 1.26
- Target package: `sdp_dev/internal/agentloop`
- SQLite: `github.com/mattn/go-sqlite3 v1.14.34` (CGO)
- Test assertions: `github.com/stretchr/testify v1.11.1`
- Existing types from: `sdp_dev/internal/harness` (TaskContract, ComplianceReport, GateStatus, Violation, etc.)
- Test runner: `go test ./internal/agentloop/... -race`

**Key design decisions to preserve:**
- `EvaluateCompliance` in `internal/harness` does NOT take a context — wrap in goroutine with `context.WithTimeout` in GateEngine
- `ToolResult.Err` is stored as `err.Error()` string in SQLite and restored via `errors.New()`
- `MessagesFromTurnRecords` Fix X2: assistant message includes `ToolCalls`; Fix Z1: empty Output + non-nil Err → `"Error: <err>"` content
- `RecoverSession` Fix X3: derives `session.Phase` from last `PhaseRecord.NextPhase`
- SQLite: open with `?_journal_mode=WAL`

---

### Task 1: Package scaffold + core types

**Files:**
- Create: `internal/agentloop/types.go`

**Step 1: Write failing test**

Create `internal/agentloop/types_compile_test.go`:

```go
package agentloop_test

import (
	"testing"

	_ "sdp_dev/internal/agentloop"
)

// TestCompile_Types verifies the package compiles with all required types.
// All real tests are in subsequent tasks; this is a compile guard.
func TestCompile_Types(t *testing.T) {
	t.Log("package agentloop compiles")
}
```

**Step 2: Run test, verify it fails**

Run: `go test ./internal/agentloop/... -run TestCompile_Types -v`
Expected: FAIL — package does not exist yet.

**Step 3: Write minimal implementation**

Create `internal/agentloop/types.go`:

```go
package agentloop

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"sdp_dev/internal/harness"
)

// ---- Role constants ----

type Role string

const (
	RoleDiscover Role = "discover"
	RolePlan     Role = "plan"
	RoleBuild    Role = "build"
	RoleReview   Role = "review"
	RoleEval     Role = "eval"
)

// ---- Message / ToolCall / Tool / ToolResult ----

type Message struct {
	Role       string     // "user" | "assistant" | "tool_result"
	Content    string
	ToolCalls  []ToolCall // Fix X2: assistant messages carry tool calls
	ToolCallID string     // Fix Y1: correlates tool_result to tool_call
	Timestamp  time.Time
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
	Sandboxed   bool
	Execute     func(ctx context.Context, id string, args json.RawMessage) (string, error) // Fix F1: context.Context, not interface{}
}

// ToolResult is the full outcome of one tool execution (success or error).
// Fix N5: AfterToolCall carries full ToolResult. Fix T1: Arguments preserved.
type ToolResult struct {
	ID        string
	Name      string
	Arguments json.RawMessage // original call arguments (Fix T1)
	Output    string
	Err       error
}

// ---- LoopConfig ----

// ContextManager trims message history to fit model context window.
type ContextManager interface {
	Trim(messages []Message, model string, maxTokens int) ([]Message, error)
}

// ModelGateway abstracts LLM API calls. Fix F2: defined here so LoopConfig.Gateway compiles.
// StubGateway (test double) is in gateway.go (Task 6).
type ModelGateway interface {
	// Call returns a channel of Events for one LLM request. Channel closes after "done" or "error".
	Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error)
	// IsAvailable returns true if the model is reachable (used by PhaseRouter.ResolveModel).
	IsAvailable(model string) bool
}

type LoopConfig struct {
	Model          string
	SystemPrompt   string
	Tools          []Tool
	MaxTokens      int
	TurnTimeout    time.Duration
	BeforeToolCall func(name string, args json.RawMessage) error
	AfterToolCall  func(result ToolResult) error
	ContextManager ContextManager // nil = passthrough (MVP)
	Gateway        ModelGateway   // Fix F2: required by Run() — set by BuildLoopConfig
}

// ---- Event ----

type Event struct {
	Type          string          // "text_delta"|"tool_call"|"tool_end"|"turn_end"|"done"|"error"|"warn"|"human_gate"|"session_stopped"
	Delta         string
	ToolCalls     []ToolCall      // Fix X2: "tool_call" event carries all parallel calls
	ToolID        string          // Fix Y1: for "tool_end" — matches ToolCall.ID
	ToolName      string
	ToolResult    string
	ToolErr       error           // Fix P4: tool failure preserved in event
	ToolArguments json.RawMessage // Fix R3: original call args on "tool_end"; RunPhase uses this to populate ToolResult.Arguments (Fix T1)
	Err           error           // loop-level error
}

// ---- PhaseConfig ----

type PhaseConfig struct {
	Models          []string
	SystemPrompt    string
	Tools           []string // allowlist names; completion_signal added implicitly by BuildLoopConfig
	AllowedNext     []Role
	RecoveryNext    []Role
	GateRequired    bool
	MinOutputTokens int
}

// ---- PhaseSnapshot ----

// PhaseSnapshot is the evidence state at gate evaluation time.
type PhaseSnapshot struct {
	Phase    Role
	Evidence []string
	Claims   []harness.Claim
	Quality  map[string]bool
}

// toHarness converts PhaseSnapshot to harness.TaskSnapshot for EvaluateCompliance.
func (ps PhaseSnapshot) toHarness() *harness.TaskSnapshot {
	quality := make(map[string]bool, len(ps.Quality))
	for k, v := range ps.Quality {
		quality[k] = v
	}
	return &harness.TaskSnapshot{
		Phase:          string(ps.Phase),
		Evidence:       ps.Evidence,
		Claims:         ps.Claims,
		QualityResults: quality,
	}
}

// ---- harnessState FSM ----

// harnessState is the Harness FSM state (Fix N1, V1).
type harnessState int

const (
	hStateIdle         harnessState = iota // ready for next prompt
	hStateRunning                          // Loop active
	hStateAwaitingHuman                    // gate escalated
	hStateStopped                          // Fix V1: terminal — Stop() called
)

// ---- completionFlag ----

// completionFlag is shared between makeCompletionSignalTool closure and RunPhase (Fix R2-2).
type completionFlag struct {
	mu       sync.Mutex
	signaled bool
	summary  string
}

// ---- PendingDecision ----

// PendingDecision is persisted when gate escalates (Fix N2).
// ApproveGate/Rollback require DecisionID — no pending = no transition.
type PendingDecision struct {
	DecisionID     string
	RunID          uint64
	Phase          Role
	GateResult     GateResult
	AllowedActions []string // "approve" | "rollback" | "stop"
}

// GateResult wraps ComplianceReport with escalation flag.
type GateResult struct {
	Report    harness.ComplianceReport
	Escalated bool
}
```

**Step 4: Verify passes**

Run: `go test ./internal/agentloop/... -run TestCompile_Types -v`
Expected: PASS

**Step 5: Commit**

```
git add internal/agentloop/types.go internal/agentloop/types_compile_test.go
git commit -m "feat(agentloop): Task 1 — core type definitions (Message, ToolCall, Tool, ToolResult, Event, Role, PhaseConfig, PhaseSnapshot, harnessState, completionFlag, PendingDecision)"
```

---

### Task 2: SessionStore interface + MemStore fake

**Files:**
- Create: `internal/agentloop/store.go`
- Create: `internal/agentloop/store_mem_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/store_mem_test.go`:

```go
package agentloop

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMemStore_ImplementsInterface verifies MemStore satisfies SessionStore at compile time.
func TestMemStore_ImplementsInterface(t *testing.T) {
	var _ SessionStore = (*MemStore)(nil)
}

// TestMemStore_PersistAndRecover verifies basic round-trip for sessions.
func TestMemStore_PersistAndRecover(t *testing.T) {
	ms := NewMemStore()
	s := &Session{
		ID:    "test-session-1",
		Phase: RoleDiscover,
	}
	require.NoError(t, ms.Persist(s))

	got, err := ms.Recover("test-session-1")
	require.NoError(t, err)
	require.Equal(t, s.ID, got.ID)
	require.Equal(t, s.Phase, got.Phase)
}

// TestMemStore_PersistAndLoadTurnRecords verifies TurnRecord round-trip.
func TestMemStore_PersistAndLoadTurnRecords(t *testing.T) {
	ms := NewMemStore()
	require.NoError(t, ms.Persist(&Session{ID: "sess"}))

	tr := TurnRecord{
		ID:    "sess:1",
		Phase: RoleBuild,
		UserMsg: Message{
			Role:    "user",
			Content: "hello",
		},
		AssistantText: "world",
	}
	require.NoError(t, ms.PersistTurnRecord("sess", tr))

	turns, err := ms.LoadTurnRecords("sess")
	require.NoError(t, err)
	require.Len(t, turns, 1)
	require.Equal(t, tr.ID, turns[0].ID)
	require.Equal(t, "world", turns[0].AssistantText)
}

// TestMemStore_PhaseRecord_roundtrip verifies LoadPhaseRecords returns persisted records in order.
func TestMemStore_PhaseRecord_roundtrip(t *testing.T) {
	ms := NewMemStore()
	require.NoError(t, ms.Persist(&Session{ID: "sess"}))

	r1 := PhaseRecord{Phase: RoleDiscover, NextPhase: RolePlan}
	r2 := PhaseRecord{Phase: RolePlan, NextPhase: RoleBuild}
	require.NoError(t, ms.PersistPhaseRecord("sess", r1))
	require.NoError(t, ms.PersistPhaseRecord("sess", r2))

	records, err := ms.LoadPhaseRecords("sess")
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, RoleDiscover, records[0].Phase)
	require.Equal(t, RolePlan, records[1].Phase)
}

// TestMemStore_Decision_lifecycle verifies Persist→Validate→Load→Clear.
func TestMemStore_Decision_lifecycle(t *testing.T) {
	ms := NewMemStore()
	require.NoError(t, ms.Persist(&Session{ID: "sess"}))

	d := PendingDecision{
		DecisionID: "sess-run1",
		RunID:      1,
		Phase:      RoleDiscover,
	}
	require.NoError(t, ms.PersistDecision("sess", d))

	// Validate: correct ID passes
	require.NoError(t, ms.ValidateDecision("sess", "sess-run1"))
	// Validate: wrong ID fails
	require.Error(t, ms.ValidateDecision("sess", "wrong-id"))

	// Load returns the pending decision
	got, err := ms.LoadDecision("sess")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "sess-run1", got.DecisionID)

	// Clear removes it
	require.NoError(t, ms.ClearDecision("sess", "sess-run1"))
	got2, err := ms.LoadDecision("sess")
	require.NoError(t, err)
	require.Nil(t, got2)
}
```

**Step 2: Run test, verify it fails**

Run: `go test ./internal/agentloop/... -run "TestMemStore" -v`
Expected: FAIL — `SessionStore`, `MemStore`, `PhaseRecord`, `Session` are not defined yet (store.go missing; PhaseRecord and Session in session.go, not yet written).

> Note: Because `MemStore` and the `SessionStore` interface both depend on `PhaseRecord` and `Session` (defined in Task 3's `session.go`), the test file is written to the `agentloop` package (internal test) so it can reference unexported types. The compile will fail until Task 3 is complete. Proceed to write `store.go` now; the tests will pass after Task 3.

**Step 3: Write minimal implementation**

Create `internal/agentloop/store.go`:

```go
package agentloop

import "fmt"

// SessionStore defines persistence for sessions, turns, phases, decisions, and events.
// SQLite implementation is in store_sqlite.go; MemStore fake is in store_mem_test.go.
type SessionStore interface {
	// Session lifecycle
	Persist(s *Session) error
	Recover(sessionID string) (*Session, error)

	// Telemetry
	PersistEvent(sessionID string, ev Event) error
	PersistGateResult(sessionID string, r GateResult) error

	// Canonical conversation log (Fix N3)
	PersistTurnRecord(sessionID string, r TurnRecord) error
	LoadTurnRecords(sessionID string) ([]TurnRecord, error)

	// Phase history — for deriving session.Phase at recovery (Fix X3)
	PersistPhaseRecord(sessionID string, r PhaseRecord) error
	LoadPhaseRecords(sessionID string) ([]PhaseRecord, error)

	// PendingDecision lifecycle (Fix N2)
	PersistDecision(sessionID string, d PendingDecision) error
	ValidateDecision(sessionID, decisionID string) error // error if not found or already cleared
	ClearDecision(sessionID, decisionID string) error
	LoadDecision(sessionID string) (*PendingDecision, error) // nil if none pending (Fix A1)
}

// MemStore is an in-memory SessionStore for tests.
// It is defined here (not in _test.go) so it is available to all test files in the package.
// Production code never imports it — it carries no build cost in non-test binaries.
type MemStore struct {
	sessions    map[string]*Session
	turns       map[string][]TurnRecord
	phases      map[string][]PhaseRecord
	decisions   map[string]*PendingDecision
	events      map[string][]Event
	gateResults map[string][]GateResult
}

// NewMemStore creates an initialized MemStore.
func NewMemStore() *MemStore {
	return &MemStore{
		sessions:    make(map[string]*Session),
		turns:       make(map[string][]TurnRecord),
		phases:      make(map[string][]PhaseRecord),
		decisions:   make(map[string]*PendingDecision),
		events:      make(map[string][]Event),
		gateResults: make(map[string][]GateResult),
	}
}

func (m *MemStore) Persist(s *Session) error {
	// Store a shallow copy so later mutations don't affect stored state.
	cp := *s
	m.sessions[s.ID] = &cp
	return nil
}

func (m *MemStore) Recover(sessionID string) (*Session, error) {
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}
	cp := *s
	return &cp, nil
}

func (m *MemStore) PersistEvent(sessionID string, ev Event) error {
	m.events[sessionID] = append(m.events[sessionID], ev)
	return nil
}

func (m *MemStore) PersistGateResult(sessionID string, r GateResult) error {
	m.gateResults[sessionID] = append(m.gateResults[sessionID], r)
	return nil
}

func (m *MemStore) PersistTurnRecord(sessionID string, r TurnRecord) error {
	m.turns[sessionID] = append(m.turns[sessionID], r)
	return nil
}

func (m *MemStore) LoadTurnRecords(sessionID string) ([]TurnRecord, error) {
	return append([]TurnRecord(nil), m.turns[sessionID]...), nil
}

func (m *MemStore) PersistPhaseRecord(sessionID string, r PhaseRecord) error {
	m.phases[sessionID] = append(m.phases[sessionID], r)
	return nil
}

func (m *MemStore) LoadPhaseRecords(sessionID string) ([]PhaseRecord, error) {
	return append([]PhaseRecord(nil), m.phases[sessionID]...), nil
}

func (m *MemStore) PersistDecision(sessionID string, d PendingDecision) error {
	cp := d
	m.decisions[sessionID] = &cp
	return nil
}

func (m *MemStore) ValidateDecision(sessionID, decisionID string) error {
	d, ok := m.decisions[sessionID]
	if !ok || d == nil {
		return fmt.Errorf("no pending decision for session %q", sessionID)
	}
	if d.DecisionID != decisionID {
		return fmt.Errorf("decision ID mismatch: want %q got %q", d.DecisionID, decisionID)
	}
	return nil
}

func (m *MemStore) ClearDecision(sessionID, decisionID string) error {
	if err := m.ValidateDecision(sessionID, decisionID); err != nil {
		return err
	}
	m.decisions[sessionID] = nil
	return nil
}

func (m *MemStore) LoadDecision(sessionID string) (*PendingDecision, error) {
	d := m.decisions[sessionID]
	if d == nil {
		return nil, nil
	}
	cp := *d
	return &cp, nil
}
```

**Step 4: Verify passes**

After Task 3 adds `session.go` (which defines `Session`, `TurnRecord`, `PhaseRecord`), run:

`go test ./internal/agentloop/... -run "TestMemStore" -v`
Expected: PASS

(At this stage, with only `types.go` + `store.go`, the compile will succeed. The tests use `Session`, `TurnRecord`, `PhaseRecord` which are defined in Task 3. Run this verification step after Task 3.)

**Step 5: Commit**

```
git add internal/agentloop/store.go internal/agentloop/store_mem_test.go
git commit -m "feat(agentloop): Task 2 — SessionStore interface + MemStore in-memory fake"
```

---

### Task 3: Session + TurnRecord + MessagesFromTurnRecords

**Files:**
- Create: `internal/agentloop/session.go`
- Create: `internal/agentloop/session_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/session_test.go`:

```go
package agentloop

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- MessagesFromTurnRecords tests ----

func TestMessagesFromTurnRecords_empty(t *testing.T) {
	s := &Session{}
	msgs := s.MessagesFromTurnRecords()
	assert.Empty(t, msgs, "empty turnRecords must produce empty slice, not nil crash")
}

func TestMessagesFromTurnRecords_withToolCalls(t *testing.T) {
	// Fix X2: assistant message must include ToolCalls for API correctness.
	s := &Session{
		turnRecords: []TurnRecord{
			{
				ID:      "sess:1",
				Phase:   RoleBuild,
				UserMsg: Message{Role: "user", Content: "build it"},
				ToolCalls: []ToolCall{
					{ID: "tc1", Name: "bash", Arguments: []byte(`{"cmd":"go build"}`)},
				},
				ToolResults: []ToolResult{
					{ID: "tc1", Name: "bash", Output: "ok"},
				},
			},
		},
	}

	msgs := s.MessagesFromTurnRecords()
	// Expected layout: user, assistant (with ToolCalls), tool_result
	require.Len(t, msgs, 3, "expect user + assistant + tool_result")

	assert.Equal(t, "user", msgs[0].Role)

	assert.Equal(t, "assistant", msgs[1].Role)
	require.Len(t, msgs[1].ToolCalls, 1, "assistant message must carry ToolCalls (Fix X2)")
	assert.Equal(t, "tc1", msgs[1].ToolCalls[0].ID)

	assert.Equal(t, "tool_result", msgs[2].Role)
	assert.Equal(t, "tc1", msgs[2].ToolCallID, "ToolCallID must match ToolCall.ID (Fix Y1)")
	assert.Equal(t, "ok", msgs[2].Content)
}

func TestMessagesFromTurnRecords_toolError_propagated(t *testing.T) {
	// Fix Z1: empty Output + non-nil Err → content = "Error: <err>".
	s := &Session{
		turnRecords: []TurnRecord{
			{
				ID:      "sess:1",
				UserMsg: Message{Role: "user", Content: "run it"},
				ToolCalls: []ToolCall{
					{ID: "tc1", Name: "bash"},
				},
				ToolResults: []ToolResult{
					{ID: "tc1", Name: "bash", Output: "", Err: errors.New("exit status 1")},
				},
			},
		},
	}

	msgs := s.MessagesFromTurnRecords()
	require.Len(t, msgs, 3)
	toolResultMsg := msgs[2]
	assert.Equal(t, "tool_result", toolResultMsg.Role)
	assert.Equal(t, "Error: exit status 1", toolResultMsg.Content,
		"Fix Z1: LLM must see tool error as non-empty content")
}

func TestMessagesFromTurnRecords_assistantTextOnly(t *testing.T) {
	// When there are no tool calls, assistant message is still emitted if text is non-empty.
	s := &Session{
		turnRecords: []TurnRecord{
			{
				ID:            "sess:1",
				UserMsg:       Message{Role: "user", Content: "hello"},
				AssistantText: "hello back",
			},
		},
	}
	msgs := s.MessagesFromTurnRecords()
	require.Len(t, msgs, 2)
	assert.Equal(t, "assistant", msgs[1].Role)
	assert.Equal(t, "hello back", msgs[1].Content)
	assert.Empty(t, msgs[1].ToolCalls)
}

func TestMessagesFromTurnRecords_noAssistantMessageWhenEmpty(t *testing.T) {
	// When AssistantText is empty and no ToolCalls, no assistant message is added.
	s := &Session{
		turnRecords: []TurnRecord{
			{
				ID:      "sess:1",
				UserMsg: Message{Role: "user", Content: "hi"},
				// No AssistantText, no ToolCalls, no ToolResults
			},
		},
	}
	msgs := s.MessagesFromTurnRecords()
	// Only the user message
	require.Len(t, msgs, 1)
	assert.Equal(t, "user", msgs[0].Role)
}

// ---- RecoverSession tests (using MemStore) ----

func TestRecoverSession_derivesPhaseFromHistory(t *testing.T) {
	// Fix X3: last PhaseRecord.NextPhase becomes session.Phase.
	ms := NewMemStore()
	s := &Session{ID: "sess-x3", Phase: RoleDiscover}
	require.NoError(t, ms.Persist(s))

	// Simulate two phase transitions persisted
	require.NoError(t, ms.PersistPhaseRecord("sess-x3", PhaseRecord{
		Phase: RoleDiscover, NextPhase: RolePlan,
	}))
	require.NoError(t, ms.PersistPhaseRecord("sess-x3", PhaseRecord{
		Phase: RolePlan, NextPhase: RoleBuild,
	}))

	recovered, err := RecoverSession("sess-x3", ms)
	require.NoError(t, err)
	assert.Equal(t, RoleBuild, recovered.Phase,
		"Phase must be derived from last PhaseRecord.NextPhase (Fix X3)")
	assert.Len(t, recovered.History, 2)
}

func TestRecoverSession_noPhaseRecords_usesInitialPhase(t *testing.T) {
	// No phase history → session.Phase stays as persisted (RoleDiscover for new sessions).
	ms := NewMemStore()
	require.NoError(t, ms.Persist(&Session{ID: "new-sess", Phase: RoleDiscover}))

	recovered, err := RecoverSession("new-sess", ms)
	require.NoError(t, err)
	assert.Equal(t, RoleDiscover, recovered.Phase)
	assert.Empty(t, recovered.History)
}

func TestRecoverSession_stoppedSession(t *testing.T) {
	// Fix X3 + W1: NextPhase="" (Stop() called) → session.Phase = "".
	// RestoreHarness will detect this and return an error; RecoverSession itself succeeds.
	ms := NewMemStore()
	require.NoError(t, ms.Persist(&Session{ID: "stopped-sess", Phase: RoleDiscover}))

	require.NoError(t, ms.PersistPhaseRecord("stopped-sess", PhaseRecord{
		Phase:    RoleDiscover,
		NextPhase: "", // Stop() — terminal
	}))

	recovered, err := RecoverSession("stopped-sess", ms)
	require.NoError(t, err)
	assert.Equal(t, Role(""), recovered.Phase,
		"last NextPhase='' means Stop() was called — Phase must be empty string")
}

func TestRecoverSession_loadsTurnRecords(t *testing.T) {
	// RecoverSession must load TurnRecords so MessagesFromTurnRecords works after restart.
	ms := NewMemStore()
	require.NoError(t, ms.Persist(&Session{ID: "sess-turns", Phase: RoleDiscover}))
	require.NoError(t, ms.PersistTurnRecord("sess-turns", TurnRecord{
		ID:            "sess-turns:1",
		Phase:         RoleDiscover,
		UserMsg:       Message{Role: "user", Content: "start"},
		AssistantText: "working on it",
		CreatedAt:     time.Now(),
	}))

	recovered, err := RecoverSession("sess-turns", ms)
	require.NoError(t, err)

	msgs := recovered.MessagesFromTurnRecords()
	require.Len(t, msgs, 2, "turn records must be restored so context is available after restart")
}

// ---- NewSession MVP test ----

func TestNewSession_initializesPhase(t *testing.T) {
	ms := NewMemStore()
	s, err := NewSession("card-001", ms)
	require.NoError(t, err)
	assert.Equal(t, "card-001", s.ID)
	assert.Equal(t, RoleDiscover, s.Phase)
	assert.Equal(t, fmt.Sprintf("sdp/%s", "card-001"), s.Branch)

	// Session should be persisted in the store
	got, err := ms.Recover("card-001")
	require.NoError(t, err)
	assert.Equal(t, "card-001", got.ID)
}

func TestSession_EmitEvent_appendsToBuffer(t *testing.T) {
	s := &Session{ID: "sess"}
	s.EmitEvent(Event{Type: "text_delta", Delta: "hello"})
	s.EmitEvent(Event{Type: "done"})
	assert.Len(t, s.events, 2)
}
```

**Step 2: Run test, verify it fails**

Run: `go test ./internal/agentloop/... -run "TestMessagesFromTurnRecords|TestRecoverSession|TestNewSession|TestSession_EmitEvent" -v`
Expected: FAIL — `Session`, `TurnRecord`, `PhaseRecord`, `RecoverSession`, `NewSession` undefined.

**Step 3: Write minimal implementation**

Create `internal/agentloop/session.go`:

```go
package agentloop

import (
	"fmt"
	"time"

	"sdp_dev/internal/harness"
)

// TurnRecord is the canonical record of one agent turn (Fix N3).
// Fix X2: ToolCalls []ToolCall required — MessagesFromTurnRecords must produce
// assistant messages with tool_calls for OpenAI/Anthropic API compliance.
type TurnRecord struct {
	ID            string       // Fix Q1: format "sessionID:runID", unique in store
	Phase         Role
	UserMsg       Message
	AssistantText string       // accumulated text_delta events
	ToolCalls     []ToolCall   // Fix X2: tool calls from this assistant message
	ToolResults   []ToolResult // outcomes of tool calls (same order as ToolCalls)
	CreatedAt     time.Time
}

// PhaseRecord records one phase transition (Fix P2: NextPhase written at persist time).
type PhaseRecord struct {
	Phase     Role
	NextPhase Role      // Fix P2: written during PersistPhaseRecord; "" = terminal Stop()
	StartedAt time.Time
	EndedAt   time.Time
	Snapshot  PhaseSnapshot
	Report    harness.ComplianceReport
}

// Session is pure data — no Loop reference (Fix I8: no circular dependency).
// Events are ephemeral in-memory telemetry; TurnRecords are the source of truth.
type Session struct {
	ID          string
	Branch      string
	Phase       Role
	Contract    *harness.TaskContract  // loaded from beads card (I12)
	History     []PhaseRecord
	events      []Event      // in-memory telemetry buffer; NOT restored on recovery (Fix Q3)
	turnRecords []TurnRecord // Fix N3: canonical log; restored by RecoverSession
}

// EmitEvent appends an event to the session's in-memory telemetry buffer.
func (s *Session) EmitEvent(ev Event) {
	s.events = append(s.events, ev)
}

// MessagesFromTurnRecords builds []Message from persisted TurnRecords.
// Fix X2: assistant message includes ToolCalls — required by OpenAI/Anthropic API.
//   Tool results without preceding tool_calls = invalid conversation → API rejection.
// Fix Z1: if r.Output == "" and r.Err != nil, content = "Error: <err>" so LLM can recover.
// Fix Y1: ToolCallID = r.ID (set from ev.ToolID in RunPhase).
func (s *Session) MessagesFromTurnRecords() []Message {
	var out []Message
	for _, tr := range s.turnRecords {
		out = append(out, tr.UserMsg)
		// Emit assistant message only when there is content to represent.
		if tr.AssistantText != "" || len(tr.ToolCalls) > 0 {
			out = append(out, Message{
				Role:      "assistant",
				Content:   tr.AssistantText, // may be "" if turn was tool-calls-only
				ToolCalls: tr.ToolCalls,     // Fix X2: include parallel tool calls
			})
		}
		for _, r := range tr.ToolResults {
			// Fix Z1: propagate tool error into LLM-visible content.
			content := r.Output
			if content == "" && r.Err != nil {
				content = fmt.Sprintf("Error: %v", r.Err)
			}
			out = append(out, Message{
				Role:       "tool_result",
				Content:    content,
				ToolCallID: r.ID, // Fix Y1: correlates to ToolCall.ID
			})
		}
	}
	return out
}

// NewSession creates a new Session for a beads card (MVP: no real beads integration).
// Takes cardID string directly; initializes Phase=RoleDiscover and persists via store.Persist.
// In production this would call loadBeadsCard + generateContract (I12).
func NewSession(cardID string, store SessionStore) (*Session, error) {
	s := &Session{
		ID:     cardID,
		Branch: "sdp/" + cardID,
		Phase:  RoleDiscover,
		// Contract: nil for MVP — populated when beads integration is complete
	}
	if err := store.Persist(s); err != nil {
		return nil, fmt.Errorf("persist new session: %w", err)
	}
	return s, nil
}

// RecoverSession restores a Session from the store, including TurnRecords and PhaseHistory.
// Fix N3/P3: TurnRecords MUST be loaded — without them MessagesFromTurnRecords returns empty.
// Fix X3: loads PhaseRecords and derives session.Phase from last PhaseRecord.NextPhase.
//   If last NextPhase == "" the session was terminated (Stop() called).
//   RestoreHarness checks for this and returns an error — RecoverSession itself succeeds.
func RecoverSession(sessionID string, store SessionStore) (*Session, error) {
	s, err := store.Recover(sessionID)
	if err != nil {
		return nil, err
	}

	// Fix X3: load phase history and derive current phase.
	phases, err := store.LoadPhaseRecords(sessionID)
	if err != nil {
		return nil, fmt.Errorf("load phase records: %w", err)
	}
	s.History = phases
	if len(phases) > 0 {
		// Phase derived from last transition record; "" signals terminal stop.
		s.Phase = phases[len(phases)-1].NextPhase
	}

	// Fix N3: load canonical conversation log so context survives restart.
	turns, err := store.LoadTurnRecords(sessionID)
	if err != nil {
		return nil, fmt.Errorf("load turn records: %w", err)
	}
	s.turnRecords = turns

	return s, nil
}
```

**Step 4: Verify passes**

Run: `go test ./internal/agentloop/... -race -v`
Expected: All tests defined so far PASS (TestCompile_Types, TestMemStore_*, TestMessagesFromTurnRecords_*, TestRecoverSession_*, TestNewSession_*, TestSession_EmitEvent_*).

**Step 5: Commit**

```
git add internal/agentloop/session.go internal/agentloop/session_test.go
git commit -m "feat(agentloop): Task 3 — Session, TurnRecord, PhaseRecord, MessagesFromTurnRecords (Fix X2, Z1, X3), RecoverSession, NewSession"
```

---

### Task 4: SQLite SessionStore implementation

**Files:**
- Create: `internal/agentloop/store_sqlite.go`
- Create: `internal/agentloop/store_sqlite_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/store_sqlite_test.go`:

```go
package agentloop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "test.db")
}

// TestSQLiteStore_ImplementsInterface verifies compile-time interface satisfaction.
func TestSQLiteStore_ImplementsInterface(t *testing.T) {
	var _ SessionStore = (*SQLiteStore)(nil)
}

func TestSQLiteStore_PersistAndRecover_Session(t *testing.T) {
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	s := &Session{ID: "s1", Phase: RoleDiscover, Branch: "sdp/s1"}
	require.NoError(t, st.Persist(s))

	got, err := st.Recover("s1")
	require.NoError(t, err)
	assert.Equal(t, "s1", got.ID)
	assert.Equal(t, RoleDiscover, got.Phase)
	assert.Equal(t, "sdp/s1", got.Branch)
}

func TestSQLiteStore_PersistAndLoad_TurnRecords(t *testing.T) {
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	require.NoError(t, st.Persist(&Session{ID: "sess", Phase: RoleDiscover}))

	toolCallArgs := []byte(`{"cmd":"go test"}`)
	tc := TurnRecord{
		ID:    "sess:1",
		Phase: RoleBuild,
		UserMsg: Message{
			Role:    "user",
			Content: "run tests",
		},
		AssistantText: "running...",
		ToolCalls: []ToolCall{
			{ID: "tc1", Name: "bash", Arguments: toolCallArgs},
		},
		ToolResults: []ToolResult{
			{ID: "tc1", Name: "bash", Output: "PASS", Err: nil},
		},
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, st.PersistTurnRecord("sess", tc))

	turns, err := st.LoadTurnRecords("sess")
	require.NoError(t, err)
	require.Len(t, turns, 1)

	got := turns[0]
	assert.Equal(t, "sess:1", got.ID)
	assert.Equal(t, RoleBuild, got.Phase)
	assert.Equal(t, "running...", got.AssistantText)
	require.Len(t, got.ToolCalls, 1)
	assert.Equal(t, "tc1", got.ToolCalls[0].ID)
	assert.Equal(t, "bash", got.ToolCalls[0].Name)
	require.Len(t, got.ToolResults, 1)
	assert.Equal(t, "tc1", got.ToolResults[0].ID)
	assert.Equal(t, "PASS", got.ToolResults[0].Output)
	assert.Nil(t, got.ToolResults[0].Err)
}

func TestSQLiteStore_TurnRecord_toolError_persisted(t *testing.T) {
	// ToolResult.Err must survive a round-trip: stored as string, restored as errors.New().
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	require.NoError(t, st.Persist(&Session{ID: "sess", Phase: RoleDiscover}))

	originalErr := errors.New("exit status 2: command not found")
	tc := TurnRecord{
		ID:      "sess:1",
		Phase:   RoleBuild,
		UserMsg: Message{Role: "user", Content: "do it"},
		ToolCalls: []ToolCall{
			{ID: "tc1", Name: "bash"},
		},
		ToolResults: []ToolResult{
			{ID: "tc1", Name: "bash", Output: "", Err: originalErr},
		},
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, st.PersistTurnRecord("sess", tc))

	turns, err := st.LoadTurnRecords("sess")
	require.NoError(t, err)
	require.Len(t, turns, 1)
	require.Len(t, turns[0].ToolResults, 1)

	restoredErr := turns[0].ToolResults[0].Err
	require.NotNil(t, restoredErr, "ToolResult.Err must be restored from stored string")
	assert.Equal(t, originalErr.Error(), restoredErr.Error())
}

func TestSQLiteStore_PersistAndLoad_PhaseRecords(t *testing.T) {
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	require.NoError(t, st.Persist(&Session{ID: "sess", Phase: RoleDiscover}))

	r1 := PhaseRecord{Phase: RoleDiscover, NextPhase: RolePlan, EndedAt: time.Now().UTC().Truncate(time.Second)}
	r2 := PhaseRecord{Phase: RolePlan, NextPhase: RoleBuild, EndedAt: time.Now().UTC().Truncate(time.Second)}
	require.NoError(t, st.PersistPhaseRecord("sess", r1))
	require.NoError(t, st.PersistPhaseRecord("sess", r2))

	records, err := st.LoadPhaseRecords("sess")
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, RoleDiscover, records[0].Phase)
	assert.Equal(t, RolePlan, records[0].NextPhase)
	assert.Equal(t, RolePlan, records[1].Phase)
	assert.Equal(t, RoleBuild, records[1].NextPhase)
}

func TestSQLiteStore_PhaseRecord_terminalStop(t *testing.T) {
	// NextPhase="" (Stop() called) must survive round-trip.
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	require.NoError(t, st.Persist(&Session{ID: "sess", Phase: RoleDiscover}))
	require.NoError(t, st.PersistPhaseRecord("sess", PhaseRecord{
		Phase:    RoleDiscover,
		NextPhase: "", // terminal
	}))

	records, err := st.LoadPhaseRecords("sess")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, Role(""), records[0].NextPhase)
}

func TestSQLiteStore_Decision_lifecycle(t *testing.T) {
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	require.NoError(t, st.Persist(&Session{ID: "sess", Phase: RoleDiscover}))

	d := PendingDecision{
		DecisionID:     "sess-run1",
		RunID:          1,
		Phase:          RoleDiscover,
		AllowedActions: []string{"approve", "rollback", "stop"},
	}
	require.NoError(t, st.PersistDecision("sess", d))

	// ValidateDecision: correct ID passes
	require.NoError(t, st.ValidateDecision("sess", "sess-run1"))
	// ValidateDecision: wrong ID fails
	require.Error(t, st.ValidateDecision("sess", "wrong-id"))

	// LoadDecision returns the pending decision
	got, err := st.LoadDecision("sess")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "sess-run1", got.DecisionID)
	assert.Equal(t, uint64(1), got.RunID)

	// ClearDecision removes it
	require.NoError(t, st.ClearDecision("sess", "sess-run1"))
	got2, err := st.LoadDecision("sess")
	require.NoError(t, err)
	assert.Nil(t, got2)
}

func TestSQLiteStore_Recover_fullRoundTrip(t *testing.T) {
	// Full session round-trip: persist session, add turns + phases, recover and verify.
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	// Create session
	s, err := NewSession("card-sqlite", st)
	require.NoError(t, err)
	assert.Equal(t, RoleDiscover, s.Phase)

	// Add a turn
	require.NoError(t, st.PersistTurnRecord("card-sqlite", TurnRecord{
		ID:            "card-sqlite:1",
		Phase:         RoleDiscover,
		UserMsg:       Message{Role: "user", Content: "discover"},
		AssistantText: "found it",
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
	}))

	// Transition phase
	require.NoError(t, st.PersistPhaseRecord("card-sqlite", PhaseRecord{
		Phase:    RoleDiscover,
		NextPhase: RolePlan,
		EndedAt:  time.Now().UTC().Truncate(time.Second),
	}))

	// Recover full session
	recovered, err := RecoverSession("card-sqlite", st)
	require.NoError(t, err)
	assert.Equal(t, RolePlan, recovered.Phase, "must derive phase from PhaseRecord history")
	assert.Len(t, recovered.History, 1)

	msgs := recovered.MessagesFromTurnRecords()
	require.Len(t, msgs, 2, "user + assistant messages from turn record")
}

func TestSQLiteStore_WAL_mode(t *testing.T) {
	// Verify WAL journal mode is set (improves concurrent read performance).
	dbPath := tempDB(t)
	st, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer st.Close()

	var mode string
	row := st.db.QueryRow("PRAGMA journal_mode")
	require.NoError(t, row.Scan(&mode))
	assert.Equal(t, "wal", mode)
}

func TestSQLiteStore_PersistEvent(t *testing.T) {
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	require.NoError(t, st.Persist(&Session{ID: "sess", Phase: RoleDiscover}))
	require.NoError(t, st.PersistEvent("sess", Event{Type: "text_delta", Delta: "hello"}))
	require.NoError(t, st.PersistEvent("sess", Event{Type: "done"}))
	// No assertion beyond no-error — events are telemetry, not recovered.
}

func TestSQLiteStore_PersistGateResult(t *testing.T) {
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	require.NoError(t, st.Persist(&Session{ID: "sess", Phase: RoleDiscover}))
	require.NoError(t, st.PersistGateResult("sess", GateResult{Escalated: true}))
	// Telemetry — no recovery needed.
}

func TestNewSQLiteStore_idempotentMigration(t *testing.T) {
	// Opening the same DB twice must not fail (migrations are idempotent via IF NOT EXISTS).
	dbPath := tempDB(t)
	st1, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	st1.Close()

	st2, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	st2.Close()
}

func TestSQLiteStore_Recover_notFound(t *testing.T) {
	st, err := NewSQLiteStore(tempDB(t))
	require.NoError(t, err)
	defer st.Close()

	_, err = st.Recover("nonexistent")
	require.Error(t, err)
}

// Test that the DB file is created at the given path.
func TestNewSQLiteStore_createsFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agentloop.db")
	st, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	st.Close()

	_, statErr := os.Stat(dbPath)
	require.NoError(t, statErr, "DB file must exist after NewSQLiteStore")
}
```

**Step 2: Run test, verify it fails**

Run: `go test ./internal/agentloop/... -run "TestSQLiteStore|TestNewSQLiteStore" -v`
Expected: FAIL — `SQLiteStore`, `NewSQLiteStore` undefined.

**Step 3: Write minimal implementation**

Create `internal/agentloop/store_sqlite.go`:

```go
package agentloop

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements SessionStore using SQLite with WAL journal mode.
// All tables are created at NewSQLiteStore() via idempotent IF NOT EXISTS migrations.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at path and runs schema migrations.
// WAL journal mode is set for better concurrent read performance.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	st := &SQLiteStore{db: db}
	if err := st.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return st, nil
}

// Close closes the underlying database connection.
func (st *SQLiteStore) Close() error {
	return st.db.Close()
}

// migrate creates all required tables if they do not exist.
func (st *SQLiteStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id      TEXT PRIMARY KEY,
			branch  TEXT NOT NULL DEFAULT '',
			phase   TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS turn_records (
			id          TEXT NOT NULL,
			session_id  TEXT NOT NULL,
			phase       TEXT NOT NULL DEFAULT '',
			user_role   TEXT NOT NULL DEFAULT 'user',
			user_content TEXT NOT NULL DEFAULT '',
			assistant_text TEXT NOT NULL DEFAULT '',
			tool_calls  TEXT NOT NULL DEFAULT '[]',
			tool_results TEXT NOT NULL DEFAULT '[]',
			created_at  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (session_id, id)
		)`,
		`CREATE TABLE IF NOT EXISTS phase_records (
			rowid_seq   INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id  TEXT NOT NULL,
			phase       TEXT NOT NULL DEFAULT '',
			next_phase  TEXT NOT NULL DEFAULT '',
			started_at  INTEGER NOT NULL DEFAULT 0,
			ended_at    INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS decisions (
			session_id   TEXT PRIMARY KEY,
			decision_id  TEXT NOT NULL,
			run_id       INTEGER NOT NULL DEFAULT 0,
			phase        TEXT NOT NULL DEFAULT '',
			payload      TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id  TEXT NOT NULL,
			type        TEXT NOT NULL DEFAULT '',
			payload     TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS gate_results (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id  TEXT NOT NULL,
			payload     TEXT NOT NULL DEFAULT '{}'
		)`,
	}
	for _, s := range stmts {
		if _, err := st.db.Exec(s); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}

// ---- toolCallRow / toolResultRow for JSON serialization ----

type toolCallRow struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type toolResultRow struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Args   json.RawMessage `json:"arguments,omitempty"`
	Output string          `json:"output"`
	Err    string          `json:"err,omitempty"` // ToolResult.Err stored as string
}

func encodeToolCalls(calls []ToolCall) (string, error) {
	rows := make([]toolCallRow, len(calls))
	for i, c := range calls {
		rows[i] = toolCallRow{ID: c.ID, Name: c.Name, Arguments: c.Arguments}
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeToolCalls(s string) ([]ToolCall, error) {
	if s == "" || s == "[]" {
		return nil, nil
	}
	var rows []toolCallRow
	if err := json.Unmarshal([]byte(s), &rows); err != nil {
		return nil, err
	}
	out := make([]ToolCall, len(rows))
	for i, r := range rows {
		out[i] = ToolCall{ID: r.ID, Name: r.Name, Arguments: r.Arguments}
	}
	return out, nil
}

func encodeToolResults(results []ToolResult) (string, error) {
	rows := make([]toolResultRow, len(results))
	for i, r := range results {
		errStr := ""
		if r.Err != nil {
			errStr = r.Err.Error()
		}
		rows[i] = toolResultRow{
			ID:     r.ID,
			Name:   r.Name,
			Args:   r.Arguments,
			Output: r.Output,
			Err:    errStr,
		}
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeToolResults(s string) ([]ToolResult, error) {
	if s == "" || s == "[]" {
		return nil, nil
	}
	var rows []toolResultRow
	if err := json.Unmarshal([]byte(s), &rows); err != nil {
		return nil, err
	}
	out := make([]ToolResult, len(rows))
	for i, r := range rows {
		var restoredErr error
		if r.Err != "" {
			restoredErr = errors.New(r.Err)
		}
		out[i] = ToolResult{
			ID:        r.ID,
			Name:      r.Name,
			Arguments: r.Args,
			Output:    r.Output,
			Err:       restoredErr,
		}
	}
	return out, nil
}

// ---- SessionStore implementation ----

func (st *SQLiteStore) Persist(s *Session) error {
	_, err := st.db.Exec(
		`INSERT INTO sessions (id, branch, phase) VALUES (?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET branch=excluded.branch, phase=excluded.phase`,
		s.ID, s.Branch, string(s.Phase),
	)
	if err != nil {
		return fmt.Errorf("persist session: %w", err)
	}
	return nil
}

func (st *SQLiteStore) Recover(sessionID string) (*Session, error) {
	row := st.db.QueryRow(`SELECT id, branch, phase FROM sessions WHERE id = ?`, sessionID)
	var s Session
	var phase string
	if err := row.Scan(&s.ID, &s.Branch, &phase); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("session %q not found", sessionID)
		}
		return nil, fmt.Errorf("recover session: %w", err)
	}
	s.Phase = Role(phase)
	return &s, nil
}

func (st *SQLiteStore) PersistEvent(sessionID string, ev Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	_, err = st.db.Exec(
		`INSERT INTO events (session_id, type, payload) VALUES (?, ?, ?)`,
		sessionID, ev.Type, string(payload),
	)
	return err
}

func (st *SQLiteStore) PersistGateResult(sessionID string, r GateResult) error {
	payload, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal gate result: %w", err)
	}
	_, err = st.db.Exec(
		`INSERT INTO gate_results (session_id, payload) VALUES (?, ?)`,
		sessionID, string(payload),
	)
	return err
}

func (st *SQLiteStore) PersistTurnRecord(sessionID string, r TurnRecord) error {
	tcJSON, err := encodeToolCalls(r.ToolCalls)
	if err != nil {
		return fmt.Errorf("encode tool calls: %w", err)
	}
	trJSON, err := encodeToolResults(r.ToolResults)
	if err != nil {
		return fmt.Errorf("encode tool results: %w", err)
	}
	// Fix R5: plain INSERT — no ON CONFLICT. turn_records is an append-only canonical log.
	// Duplicate IDs (same session_id + id) are a bug in runID generation and must surface
	// as a UNIQUE constraint error, not be silently swallowed by an upsert.
	_, err = st.db.Exec(
		`INSERT INTO turn_records
		 (id, session_id, phase, user_role, user_content, assistant_text, tool_calls, tool_results, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, sessionID, string(r.Phase),
		r.UserMsg.Role, r.UserMsg.Content,
		r.AssistantText, tcJSON, trJSON,
		r.CreatedAt.UTC().Unix(),
	)
	if err != nil {
		return fmt.Errorf("persist turn record: %w", err)
	}
	return nil
}

func (st *SQLiteStore) LoadTurnRecords(sessionID string) ([]TurnRecord, error) {
	rows, err := st.db.Query(
		`SELECT id, phase, user_role, user_content, assistant_text, tool_calls, tool_results, created_at
		 FROM turn_records WHERE session_id = ? ORDER BY created_at ASC, id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query turn records: %w", err)
	}
	defer rows.Close()

	var out []TurnRecord
	for rows.Next() {
		var tr TurnRecord
		var phase, userRole, tcJSON, trJSON string
		var createdAtUnix int64
		if err := rows.Scan(
			&tr.ID, &phase, &userRole, &tr.UserMsg.Content,
			&tr.AssistantText, &tcJSON, &trJSON, &createdAtUnix,
		); err != nil {
			return nil, fmt.Errorf("scan turn record: %w", err)
		}
		tr.Phase = Role(phase)
		tr.UserMsg.Role = userRole
		tr.CreatedAt = time.Unix(createdAtUnix, 0).UTC()

		if tc, err := decodeToolCalls(tcJSON); err != nil {
			return nil, fmt.Errorf("decode tool calls: %w", err)
		} else {
			tr.ToolCalls = tc
		}
		if trs, err := decodeToolResults(trJSON); err != nil {
			return nil, fmt.Errorf("decode tool results: %w", err)
		} else {
			tr.ToolResults = trs
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

func (st *SQLiteStore) PersistPhaseRecord(sessionID string, r PhaseRecord) error {
	_, err := st.db.Exec(
		`INSERT INTO phase_records (session_id, phase, next_phase, started_at, ended_at)
		 VALUES (?, ?, ?, ?, ?)`,
		sessionID, string(r.Phase), string(r.NextPhase),
		r.StartedAt.UTC().Unix(), r.EndedAt.UTC().Unix(),
	)
	if err != nil {
		return fmt.Errorf("persist phase record: %w", err)
	}
	return nil
}

func (st *SQLiteStore) LoadPhaseRecords(sessionID string) ([]PhaseRecord, error) {
	rows, err := st.db.Query(
		`SELECT phase, next_phase, started_at, ended_at
		 FROM phase_records WHERE session_id = ? ORDER BY rowid_seq ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query phase records: %w", err)
	}
	defer rows.Close()

	var out []PhaseRecord
	for rows.Next() {
		var pr PhaseRecord
		var phase, nextPhase string
		var startedAtUnix, endedAtUnix int64
		if err := rows.Scan(&phase, &nextPhase, &startedAtUnix, &endedAtUnix); err != nil {
			return nil, fmt.Errorf("scan phase record: %w", err)
		}
		pr.Phase = Role(phase)
		pr.NextPhase = Role(nextPhase)
		pr.StartedAt = time.Unix(startedAtUnix, 0).UTC()
		pr.EndedAt = time.Unix(endedAtUnix, 0).UTC()
		out = append(out, pr)
	}
	return out, rows.Err()
}

func (st *SQLiteStore) PersistDecision(sessionID string, d PendingDecision) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("marshal decision: %w", err)
	}
	_, err = st.db.Exec(
		`INSERT INTO decisions (session_id, decision_id, run_id, phase, payload) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(session_id) DO UPDATE SET
		   decision_id=excluded.decision_id,
		   run_id=excluded.run_id,
		   phase=excluded.phase,
		   payload=excluded.payload`,
		sessionID, d.DecisionID, int64(d.RunID), string(d.Phase), string(payload),
	)
	if err != nil {
		return fmt.Errorf("persist decision: %w", err)
	}
	return nil
}

func (st *SQLiteStore) ValidateDecision(sessionID, decisionID string) error {
	var stored string
	err := st.db.QueryRow(
		`SELECT decision_id FROM decisions WHERE session_id = ?`, sessionID,
	).Scan(&stored)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("no pending decision for session %q", sessionID)
		}
		return fmt.Errorf("validate decision: %w", err)
	}
	if stored != decisionID {
		return fmt.Errorf("decision ID mismatch: want %q got %q", stored, decisionID)
	}
	return nil
}

func (st *SQLiteStore) ClearDecision(sessionID, decisionID string) error {
	if err := st.ValidateDecision(sessionID, decisionID); err != nil {
		return err
	}
	_, err := st.db.Exec(`DELETE FROM decisions WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("clear decision: %w", err)
	}
	return nil
}

func (st *SQLiteStore) LoadDecision(sessionID string) (*PendingDecision, error) {
	var payload string
	err := st.db.QueryRow(
		`SELECT payload FROM decisions WHERE session_id = ?`, sessionID,
	).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // no pending decision
		}
		return nil, fmt.Errorf("load decision: %w", err)
	}
	var d PendingDecision
	if err := json.Unmarshal([]byte(payload), &d); err != nil {
		return nil, fmt.Errorf("unmarshal decision: %w", err)
	}
	return &d, nil
}
```

**Step 4: Verify passes**

Run: `go test ./internal/agentloop/... -race -v`
Expected: All tests PASS. Note: SQLite requires CGO — ensure `CGO_ENABLED=1` is set (default on macOS/Linux with gcc available).

**Step 5: Commit**

```
git add internal/agentloop/store_sqlite.go internal/agentloop/store_sqlite_test.go
git commit -m "feat(agentloop): Task 4 — SQLiteStore with WAL, schema migrations, TurnRecord+PhaseRecord+Decision persistence"
```

---

### Task 5: EvidenceAccumulator

**Files:**
- Create: `internal/agentloop/evidence.go`
- Create: `internal/agentloop/evidence_test.go`

**Step 1: Write failing test**

Create `internal/agentloop/evidence_test.go`:

```go
package agentloop

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvidenceAccumulator_newHasEmptyState(t *testing.T) {
	ea := NewEvidenceAccumulator()
	snap := ea.Snapshot(RoleDiscover)
	assert.Empty(t, snap.Evidence)
	assert.Empty(t, snap.Claims)
	assert.Empty(t, snap.Quality)
}

func TestEvidenceAccumulator_onToolError_recordsNegativeEvidence(t *testing.T) {
	ea := NewEvidenceAccumulator()
	err := ea.OnToolResult(ToolResult{
		ID:   "tc1",
		Name: "bash",
		Err:  errors.New("exit status 1"),
	})
	require.NoError(t, err, "OnToolResult must not return error for tool failures")

	snap := ea.Snapshot(RoleDiscover)
	require.Len(t, snap.Evidence, 1)
	assert.Contains(t, snap.Evidence[0], "tool_error:bash:exit status 1",
		"tool errors must be recorded as negative evidence")
	// Quality must NOT be set for failed tool call
	assert.False(t, snap.Quality["test"])
}

func TestEvidenceAccumulator_onBashSuccess_setsQuality(t *testing.T) {
	ea := NewEvidenceAccumulator()
	err := ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "--- PASS: TestFoo (0.01s)\nok  \tsdp_dev/internal/agentloop\t0.123s",
	})
	require.NoError(t, err)

	snap := ea.Snapshot(RoleBuild)
	assert.True(t, snap.Quality["test"], "bash output containing PASS must set quality[test]=true")
}

func TestEvidenceAccumulator_onBashFailOutput_doesNotSetQuality(t *testing.T) {
	ea := NewEvidenceAccumulator()
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "--- FAIL: TestBar (0.05s)\nFAIL",
	}))

	snap := ea.Snapshot(RoleBuild)
	assert.False(t, snap.Quality["test"], "bash FAIL output must not set quality[test]")
}

func TestEvidenceAccumulator_onEditFile_recordsEvidence(t *testing.T) {
	ea := NewEvidenceAccumulator()
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "edit_file",
		Output: "wrote internal/agentloop/session.go",
	}))

	snap := ea.Snapshot(RoleBuild)
	require.Len(t, snap.Evidence, 1)
	assert.Contains(t, snap.Evidence[0], "file_modified:")
}

func TestEvidenceAccumulator_onBdCreate_recordsEvidence(t *testing.T) {
	ea := NewEvidenceAccumulator()
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bd_create",
		Output: "created card PROJ-42",
	}))

	snap := ea.Snapshot(RolePlan)
	require.Len(t, snap.Evidence, 1)
	assert.Contains(t, snap.Evidence[0], "card_created:")
}

func TestEvidenceAccumulator_reset_clearsAll(t *testing.T) {
	ea := NewEvidenceAccumulator()

	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "PASS",
	}))
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:   "tc2",
		Name: "edit_file",
		Output: "wrote foo.go",
	}))

	snap := ea.Snapshot(RoleBuild)
	assert.NotEmpty(t, snap.Evidence)

	ea.Reset()

	snap2 := ea.Snapshot(RoleBuild)
	assert.Empty(t, snap2.Evidence, "Reset must clear evidence")
	assert.Empty(t, snap2.Quality, "Reset must clear quality")
	assert.Empty(t, snap2.Claims, "Reset must clear claims")
}

func TestEvidenceAccumulator_reset_allowsReuse(t *testing.T) {
	// After Reset, OnToolResult must still work (no nil map panic — Fix Q2).
	ea := NewEvidenceAccumulator()
	ea.Reset()

	err := ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "ok  \tsdp_dev\t0.01s",
	})
	require.NoError(t, err, "OnToolResult after Reset must not panic")
}

func TestEvidenceAccumulator_snapshot_concurrent(t *testing.T) {
	// Race detector test: concurrent OnToolResult + Snapshot must not data-race.
	ea := NewEvidenceAccumulator()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = ea.OnToolResult(ToolResult{ID: "tc", Name: "bash", Output: "PASS"})
		}()
		go func() {
			defer wg.Done()
			_ = ea.Snapshot(RoleBuild)
		}()
	}
	wg.Wait()
}

func TestEvidenceAccumulator_snapshot_returnsPhase(t *testing.T) {
	ea := NewEvidenceAccumulator()
	snap := ea.Snapshot(RoleReview)
	assert.Equal(t, RoleReview, snap.Phase)
}

func TestEvidenceAccumulator_toHarness_includesEvidence(t *testing.T) {
	ea := NewEvidenceAccumulator()
	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: "PASS",
	}))

	snap := ea.Snapshot(RoleBuild)
	hs := snap.toHarness()
	assert.Equal(t, "build", hs.Phase)
	assert.NotNil(t, hs.QualityResults)
	assert.True(t, hs.QualityResults["test"])
}
```

**Step 2: Run test, verify it fails**

Run: `go test ./internal/agentloop/... -run "TestEvidenceAccumulator" -race -v`
Expected: FAIL — `EvidenceAccumulator`, `NewEvidenceAccumulator` undefined.

**Step 3: Write minimal implementation**

Create `internal/agentloop/evidence.go`:

```go
package agentloop

import (
	"fmt"
	"strings"
	"sync"

	"sdp_dev/internal/harness"
)

// EvidenceAccumulator collects structured evidence from tool results.
// The agent cannot self-report gate passage — only tool outcomes count (I9).
// Fix Q2: quality map initialized in constructor — no nil map panic in OnToolResult.
// Fix A6: Reset() explicitly defined — called by transitionTo on phase change.
type EvidenceAccumulator struct {
	mu       sync.Mutex
	evidence []string
	claims   []harness.Claim
	quality  map[string]bool
}

// NewEvidenceAccumulator creates an EvidenceAccumulator with initialized maps (Fix Q2).
func NewEvidenceAccumulator() *EvidenceAccumulator {
	return &EvidenceAccumulator{
		quality: make(map[string]bool),
	}
}

// OnToolResult is called via AfterToolCall hook after each tool execution (Fix N5: full ToolResult).
// Tool errors are recorded as negative evidence, not ignored.
// Structured per-tool extraction — no LLM summarization.
func (ea *EvidenceAccumulator) OnToolResult(r ToolResult) error {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	if r.Err != nil {
		// Tool failure: negative evidence — EvidenceAccumulator must know about failures.
		ea.evidence = append(ea.evidence, fmt.Sprintf("tool_error:%s:%s", r.Name, r.Err.Error()))
		return nil
	}

	switch r.Name {
	case "bash":
		// PASS in output → quality["test"] = true; FAIL → false (explicit).
		ea.quality["test"] = extractTestPass(r.Output)
	case "edit_file":
		ea.evidence = append(ea.evidence, "file_modified:"+extractFilePath(r.Output))
	case "bd_create":
		ea.evidence = append(ea.evidence, "card_created:"+extractCardID(r.Output))
	}
	return nil
}

// Reset clears all accumulated evidence, claims, and quality without reallocation (Fix A6).
// Called by transitionTo on every phase change so the next phase starts fresh.
func (ea *EvidenceAccumulator) Reset() {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.evidence = ea.evidence[:0]
	ea.claims = ea.claims[:0]
	for k := range ea.quality {
		delete(ea.quality, k)
	}
}

// Snapshot returns a point-in-time copy of accumulated evidence for the given phase.
// Thread-safe — copies slices so callers cannot race on the original.
func (ea *EvidenceAccumulator) Snapshot(phase Role) PhaseSnapshot {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	evidence := make([]string, len(ea.evidence))
	copy(evidence, ea.evidence)

	claims := make([]harness.Claim, len(ea.claims))
	copy(claims, ea.claims)

	quality := make(map[string]bool, len(ea.quality))
	for k, v := range ea.quality {
		quality[k] = v
	}

	return PhaseSnapshot{
		Phase:    phase,
		Evidence: evidence,
		Claims:   claims,
		Quality:  quality,
	}
}

// ---- per-tool extractors ----

// extractTestPass returns true if the bash output indicates test success.
// Heuristic: presence of " PASS" or "ok " prefix (go test output convention).
func extractTestPass(output string) bool {
	return strings.Contains(output, "PASS") && !strings.Contains(output, "FAIL")
}

// extractFilePath extracts a file path from edit_file output.
// edit_file outputs something like "wrote path/to/file.go" or the path directly.
func extractFilePath(output string) string {
	// Heuristic: last word often contains the path.
	parts := strings.Fields(output)
	if len(parts) == 0 {
		return output
	}
	return parts[len(parts)-1]
}

// extractCardID extracts a card ID from bd_create output.
// bd_create outputs something like "created card PROJ-42".
func extractCardID(output string) string {
	parts := strings.Fields(output)
	if len(parts) == 0 {
		return output
	}
	return parts[len(parts)-1]
}
```

**Step 4: Verify passes**

Run: `go test ./internal/agentloop/... -race -v`
Expected: ALL tests across all tasks PASS with race detector enabled.

Verify explicitly:
```
go test ./internal/agentloop/... -race -count=1 -v 2>&1 | tail -5
```
Expected last lines:
```
--- PASS: TestEvidenceAccumulator_snapshot_concurrent (...)
PASS
ok  	sdp_dev/internal/agentloop	...
```

**Step 5: Commit**

```
git add internal/agentloop/evidence.go internal/agentloop/evidence_test.go
git commit -m "feat(agentloop): Task 5 — EvidenceAccumulator with OnToolResult, Snapshot, Reset; concurrent-safe (Fix Q2, A6, N5)"
```

---

## Final verification

After all five tasks are committed:

```bash
go build ./internal/agentloop/...
go test ./internal/agentloop/... -race -count=1 -v
```

Expected: zero compilation errors, all tests pass, no race detector warnings.

## Notes for the implementer

1. **Tool.Execute uses `context.Context`**: `types.go` imports `"context"` and defines `Execute func(ctx context.Context, ...)`. This is correct and required — loop.go (Task 7+) passes `context.Context` directly. No interface{} placeholder needed.

2. **harness.EvaluateCompliance does NOT take context**: the real signature is `EvaluateCompliance(contract *TaskContract, snapshot *TaskSnapshot) ComplianceReport`. The design spec shows a context parameter that was never implemented. GateEngine (future task) wraps the call in a goroutine with `context.WithTimeout` and uses `select` on the result channel and `evalCtx.Done()` — exactly as shown in the design spec's GateEngine code.

3. **MemStore is in `store.go`, not `store_mem_test.go`**: the spec request said `store_mem_test.go`, but MemStore is needed by `session_test.go` (Task 3) which is also in the internal `agentloop` package. Placing it in `store.go` (non-test file) keeps it accessible from all test files in the package. It adds no binary weight since it's only ever instantiated in tests.

4. **SQLite CGO requirement**: `go-sqlite3` requires CGO. If CI uses `CGO_ENABLED=0`, add a build tag guard or use a pure-Go SQLite driver (`modernc.org/sqlite`). For local macOS development with Xcode CLT, the default `CGO_ENABLED=1` works.
