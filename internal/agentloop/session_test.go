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
