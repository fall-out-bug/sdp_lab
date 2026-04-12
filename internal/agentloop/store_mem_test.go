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

func TestMemStore_PersistAndLoadEvents(t *testing.T) {
	ms := NewMemStore()
	require.NoError(t, ms.Persist(&Session{ID: "sess"}))
	require.NoError(t, ms.PersistEvent("sess", Event{
		Type:   "dispatch_metric",
		Code:   "dispatch_attempt_total",
		Count:  1,
		Fields: map[string]string{"total": "1"},
	}))

	events, err := ms.LoadEvents("sess")
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "dispatch_metric", events[0].Type)
	require.Equal(t, "dispatch_attempt_total", events[0].Code)
	require.Equal(t, 1, events[0].Count)
	require.Equal(t, "1", events[0].Fields["total"])
}
