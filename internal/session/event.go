// Package session provides hash-chained session evidence logging for tool calls.
// Every tool call emits a typed event to .sdp/log/session-{id}.jsonl with prev_hash linking.
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Event types for session logging.
const (
	EventTypeToolCall     = "tool_call"
	EventTypeToolResult   = "tool_result"
	EventTypeGuardCheck   = "guard_check"
	EventTypeSessionStart = "session_start"
	EventTypeSessionEnd   = "session_end"
)

// Event is a single entry in the session log.
type Event struct {
	// Type identifies the event type (tool_call, tool_result, guard_check, etc.)
	Type string `json:"type"`

	// Timestamp is ISO 8601 UTC timestamp.
	Timestamp string `json:"timestamp"`

	// SessionID is the unique session identifier.
	SessionID string `json:"session_id"`

	// Sequence is the monotonically increasing event number (1-indexed).
	Sequence int `json:"sequence"`

	// Hash is SHA-256 of this event's canonical JSON (excluding hash field).
	Hash string `json:"hash"`

	// PrevHash is SHA-256 of the previous event's hash (empty for first event).
	PrevHash string `json:"prev_hash,omitempty"`

	// Payload contains type-specific data.
	Payload json.RawMessage `json:"payload"`
}

// ToolCallPayload is the payload for tool_call events.
type ToolCallPayload struct {
	Tool  string          `json:"tool"`
	Args  json.RawMessage `json:"args,omitempty"`
	Files []string        `json:"files,omitempty"`
	WSID  string          `json:"ws_id,omitempty"`
}

// ToolResultPayload is the payload for tool_result events.
type ToolResultPayload struct {
	Tool    string          `json:"tool"`
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Output  json.RawMessage `json:"output,omitempty"`
}

// GuardCheckPayload is the payload for guard_check events.
type GuardCheckPayload struct {
	WSID       string   `json:"ws_id"`
	Tool       string   `json:"tool"`
	Files      []string `json:"files,omitempty"`
	Allowed    bool     `json:"allowed"`
	Violations []string `json:"violations,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

// SessionStartPayload is the payload for session_start events.
type SessionStartPayload struct {
	WSID       string `json:"ws_id,omitempty"`
	Intent     string `json:"intent,omitempty"`
	WorkingDir string `json:"working_dir"`
}

// SessionEndPayload is the payload for session_end events.
type SessionEndPayload struct {
	RootHash   string `json:"root_hash"`
	EventCount int    `json:"event_count"`
	Status     string `json:"status"`
}

// NewEvent creates a new event with computed hash.
func NewEvent(eventType, sessionID string, sequence int, prevHash string, payload any) (Event, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal payload: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	evt := Event{
		Type:      eventType,
		Timestamp: now,
		SessionID: sessionID,
		Sequence:  sequence,
		PrevHash:  prevHash,
		Payload:   json.RawMessage(payloadBytes),
	}

	// Compute hash of everything except the hash field itself
	evt.Hash = computeEventHash(evt)

	return evt, nil
}

// computeEventHash computes SHA-256 hash of the event (excluding hash field).
func computeEventHash(evt Event) string {
	// Create a copy without the hash field for hashing
	hashInput := struct {
		Type      string          `json:"type"`
		Timestamp string          `json:"timestamp"`
		SessionID string          `json:"session_id"`
		Sequence  int             `json:"sequence"`
		PrevHash  string          `json:"prev_hash,omitempty"`
		Payload   json.RawMessage `json:"payload"`
	}{
		Type:      evt.Type,
		Timestamp: evt.Timestamp,
		SessionID: evt.SessionID,
		Sequence:  evt.Sequence,
		PrevHash:  evt.PrevHash,
		Payload:   evt.Payload,
	}

	// Marshal with deterministic ordering (Go's json.Marshal sorts map keys)
	b, err := json.Marshal(hashInput)
	if err != nil {
		// This should never happen with valid structs
		return ""
	}

	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// ToJSONL returns the event as a JSONL line (single line JSON with newline).
func (e Event) ToJSONL() (string, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("marshal event: %w", err)
	}
	return string(b) + "\n", nil
}

// ValidateHash verifies that the event's hash matches its computed hash.
func (e Event) ValidateHash() bool {
	computed := computeEventHash(e)
	return computed == e.Hash
}
