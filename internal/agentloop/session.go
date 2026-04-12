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
	ID            string // Fix Q1: format "sessionID:runID", unique in store
	Phase         Role
	UserMsg       Message
	AssistantText string       // accumulated text_delta events
	ToolCalls     []ToolCall   // Fix X2: tool calls from this assistant message
	ToolResults   []ToolResult // outcomes of tool calls (same order as ToolCalls)
	CreatedAt     time.Time
}

type SessionBinding struct {
	FeatureID      string
	WSID           string
	ProjectRoot    string
	ClaimedIssueID string
}

// PhaseRecord records one phase transition (Fix P2: NextPhase written at persist time).
type PhaseRecord struct {
	Phase     Role
	NextPhase Role // Fix P2: written during PersistPhaseRecord; "" = terminal Stop()
	StartedAt time.Time
	EndedAt   time.Time
	Snapshot  PhaseSnapshot
	Report    harness.ComplianceReport
}

// Session is pure data — no Loop reference (Fix I8: no circular dependency).
// Events are ephemeral in-memory telemetry; TurnRecords are the source of truth.
type Session struct {
	ID             string
	Branch         string
	FeatureID      string
	WSID           string
	ProjectRoot    string
	ClaimedIssueID string
	Phase          Role
	Contract       *harness.TaskContract // loaded from beads card (I12)
	History        []PhaseRecord
	events         []Event      // in-memory telemetry buffer; NOT restored on recovery (Fix Q3)
	turnRecords    []TurnRecord // Fix N3: canonical log; restored by RecoverSession
}

// EmitEvent appends an event to the session's in-memory telemetry buffer.
func (s *Session) EmitEvent(ev Event) {
	s.events = append(s.events, ev)
}

// MessagesFromTurnRecords builds []Message from persisted TurnRecords.
// Fix X2: assistant message includes ToolCalls — required by OpenAI/Anthropic API.
//
//	Tool results without preceding tool_calls = invalid conversation → API rejection.
//
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
	return NewBoundSession(cardID, SessionBinding{}, store)
}

// NewBoundSession creates a new Session bound to one feature/workstream execution target.
func NewBoundSession(cardID string, binding SessionBinding, store SessionStore) (*Session, error) {
	branch := "sdp/" + cardID
	if binding.FeatureID != "" && binding.WSID != "" {
		branch = "feature/" + binding.FeatureID + "/" + binding.WSID
	}
	s := &Session{
		ID:             cardID,
		Branch:         branch,
		FeatureID:      binding.FeatureID,
		WSID:           binding.WSID,
		ProjectRoot:    binding.ProjectRoot,
		ClaimedIssueID: binding.ClaimedIssueID,
		Phase:          RoleDiscover,
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
//
//	If last NextPhase == "" the session was terminated (Stop() called).
//	RestoreHarness checks for this and returns an error — RecoverSession itself succeeds.
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
