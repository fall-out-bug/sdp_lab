package session

import (
	"encoding/json"
	"testing"
)

func TestNewEvent_HashChain(t *testing.T) {
	sessionID := "test-session-001"

	// Create first event (no prev_hash)
	evt1, err := NewEvent(EventTypeSessionStart, sessionID, 1, "", SessionStartPayload{
		WorkingDir: "/tmp/test",
	})
	if err != nil {
		t.Fatalf("create event 1: %v", err)
	}

	if evt1.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", evt1.Sequence)
	}
	if evt1.PrevHash != "" {
		t.Errorf("first event should have empty prev_hash, got %q", evt1.PrevHash)
	}
	if evt1.Hash == "" {
		t.Error("hash should not be empty")
	}

	// Create second event (linked to first)
	evt2, err := NewEvent(EventTypeToolCall, sessionID, 2, evt1.Hash, ToolCallPayload{
		Tool:  "read",
		Files: []string{"test.txt"},
	})
	if err != nil {
		t.Fatalf("create event 2: %v", err)
	}

	if evt2.Sequence != 2 {
		t.Errorf("expected sequence 2, got %d", evt2.Sequence)
	}
	if evt2.PrevHash != evt1.Hash {
		t.Errorf("prev_hash should link to event 1")
	}
	if evt2.Hash == evt1.Hash {
		t.Error("hashes should be different for different events")
	}

	// Create third event (linked to second)
	evt3, err := NewEvent(EventTypeGuardCheck, sessionID, 3, evt2.Hash, GuardCheckPayload{
		Tool:    "write",
		Allowed: true,
	})
	if err != nil {
		t.Fatalf("create event 3: %v", err)
	}

	if evt3.PrevHash != evt2.Hash {
		t.Errorf("prev_hash should link to event 2")
	}
}

func TestEvent_ValidateHash(t *testing.T) {
	evt, err := NewEvent(EventTypeToolCall, "session-1", 1, "", ToolCallPayload{
		Tool: "read",
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if !evt.ValidateHash() {
		t.Error("hash should be valid")
	}

	// Tamper with the event
	evt.Sequence = 999
	if evt.ValidateHash() {
		t.Error("hash should be invalid after tampering")
	}
}

func TestEvent_ToJSONL(t *testing.T) {
	evt, err := NewEvent(EventTypeToolCall, "session-1", 1, "", ToolCallPayload{
		Tool:  "read",
		Files: []string{"a.txt", "b.txt"},
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	line, err := evt.ToJSONL()
	if err != nil {
		t.Fatalf("to JSONL: %v", err)
	}

	// Should be valid JSON
	var parsed Event
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		t.Fatalf("parse JSONL: %v", err)
	}

	// Should end with newline
	if line[len(line)-1] != '\n' {
		t.Error("JSONL should end with newline")
	}

	// Verify hash round-trips
	if parsed.Hash != evt.Hash {
		t.Errorf("hash mismatch after round-trip")
	}
}

func TestGuardCheckPayload_Serialization(t *testing.T) {
	payload := GuardCheckPayload{
		WSID:       "00-059-01",
		Tool:       "write",
		Files:      []string{"internal/session/writer.go"},
		Allowed:    false,
		Violations: []string{"out of scope: cmd/other/main.go"},
		Reason:     "file not in declared scope",
	}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed GuardCheckPayload
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.Allowed != false {
		t.Error("allowed should be false")
	}
	if len(parsed.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(parsed.Violations))
	}
}
