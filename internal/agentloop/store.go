package agentloop

import "fmt"

// SessionStore defines persistence for sessions, turns, phases, decisions, and events.
// SQLite implementation is in store_sqlite.go. MemStore is defined below (not in _test.go,
// so all internal test files in package agentloop can access it without a _test suffix).
type SessionStore interface {
	// Session lifecycle
	Persist(s *Session) error
	Recover(sessionID string) (*Session, error)

	// Telemetry
	PersistEvent(sessionID string, ev Event) error
	LoadEvents(sessionID string) ([]Event, error)
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

	// TransitionAndClearDecision atomically persists a phase record and clears the
	// pending decision in a single transaction. Used by ApproveGate/Rollback to avoid
	// partial state on crash: either both operations commit or neither does.
	TransitionAndClearDecision(sessionID, decisionID string, record PhaseRecord) error
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

func (m *MemStore) LoadEvents(sessionID string) ([]Event, error) {
	return append([]Event(nil), m.events[sessionID]...), nil
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

func (m *MemStore) TransitionAndClearDecision(sessionID, decisionID string, record PhaseRecord) error {
	// Validate first — fail if decision is missing or mismatched.
	if err := m.ValidateDecision(sessionID, decisionID); err != nil {
		return err
	}
	if err := m.PersistPhaseRecord(sessionID, record); err != nil {
		return err
	}
	m.decisions[sessionID] = nil
	return nil
}
