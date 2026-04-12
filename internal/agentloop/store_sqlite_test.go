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

	s := &Session{ID: "s1", Phase: RoleDiscover, Branch: "sdp/s1", ClaimedIssueID: "sdplab-62nw"}
	require.NoError(t, st.Persist(s))

	got, err := st.Recover("s1")
	require.NoError(t, err)
	assert.Equal(t, "s1", got.ID)
	assert.Equal(t, RoleDiscover, got.Phase)
	assert.Equal(t, "sdp/s1", got.Branch)
	assert.Equal(t, "sdplab-62nw", got.ClaimedIssueID)
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
		Phase:     RoleDiscover,
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
		Phase:     RoleDiscover,
		NextPhase: RolePlan,
		EndedAt:   time.Now().UTC().Truncate(time.Second),
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
	require.NoError(t, st.PersistEvent("sess", Event{
		Type:   "dispatch_metric",
		Code:   "dispatch_attempt_total",
		Count:  1,
		Fields: map[string]string{"total": "1"},
		Delta:  "hello",
	}))
	require.NoError(t, st.PersistEvent("sess", Event{Type: "done"}))

	events, err := st.LoadEvents("sess")
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "dispatch_metric", events[0].Type)
	assert.Equal(t, "dispatch_attempt_total", events[0].Code)
	assert.Equal(t, 1, events[0].Count)
	assert.Equal(t, "1", events[0].Fields["total"])
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
