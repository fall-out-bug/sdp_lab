package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultLogDir is the default directory for session logs relative to project root.
const DefaultLogDir = ".sdp/log"

// Writer writes hash-chained events to a session log file.
type Writer struct {
	mu        sync.Mutex
	file      *os.File
	sessionID string
	prevHash  string
	sequence  int
	closed    bool
}

// NewWriter creates a new session log writer.
// The log file is created at .sdp/log/session-{sessionID}.jsonl.
func NewWriter(projectRoot, sessionID string) (*Writer, error) {
	logDir := filepath.Join(projectRoot, DefaultLogDir)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("session-%s.jsonl", sessionID))
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return &Writer{
		file:      f,
		sessionID: sessionID,
		prevHash:  "",
		sequence:  0,
		closed:    false,
	}, nil
}

// Append writes a new event to the session log.
// The event is hash-chained to the previous event via prev_hash.
func (w *Writer) Append(eventType string, payload any) (Event, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return Event{}, fmt.Errorf("writer is closed")
	}

	w.sequence++
	evt, err := NewEvent(eventType, w.sessionID, w.sequence, w.prevHash, payload)
	if err != nil {
		return Event{}, fmt.Errorf("create event: %w", err)
	}

	line, err := evt.ToJSONL()
	if err != nil {
		return Event{}, fmt.Errorf("serialize event: %w", err)
	}

	if _, err := w.file.WriteString(line); err != nil {
		return Event{}, fmt.Errorf("write event: %w", err)
	}

	// Update prev_hash for next event
	w.prevHash = evt.Hash

	return evt, nil
}

// AppendToolCall appends a tool_call event.
func (w *Writer) AppendToolCall(tool string, args json.RawMessage, files []string, wsID string) (Event, error) {
	return w.Append(EventTypeToolCall, ToolCallPayload{
		Tool:  tool,
		Args:  args,
		Files: files,
		WSID:  wsID,
	})
}

// AppendToolResult appends a tool_result event.
func (w *Writer) AppendToolResult(tool string, success bool, errMsg string, output json.RawMessage) (Event, error) {
	return w.Append(EventTypeToolResult, ToolResultPayload{
		Tool:    tool,
		Success: success,
		Error:   errMsg,
		Output:  output,
	})
}

// AppendGuardCheck appends a guard_check event.
func (w *Writer) AppendGuardCheck(wsID, tool string, files []string, allowed bool, violations []string, reason string) (Event, error) {
	return w.Append(EventTypeGuardCheck, GuardCheckPayload{
		WSID:       wsID,
		Tool:       tool,
		Files:      files,
		Allowed:    allowed,
		Violations: violations,
		Reason:     reason,
	})
}

// Finalize closes the log and returns the root hash (hash of last event).
// After Finalize, no more events can be appended.
func (w *Writer) Finalize(status string) (rootHash string, eventCount int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return "", 0, fmt.Errorf("writer already closed")
	}

	// Write session_end event
	evt, err := NewEvent(EventTypeSessionEnd, w.sessionID, w.sequence+1, w.prevHash, SessionEndPayload{
		RootHash:   w.prevHash, // Root hash is the last event's hash
		EventCount: w.sequence,
		Status:     status,
	})
	if err != nil {
		return "", 0, fmt.Errorf("create session_end event: %w", err)
	}

	line, err := evt.ToJSONL()
	if err != nil {
		return "", 0, fmt.Errorf("serialize session_end: %w", err)
	}

	if _, err := w.file.WriteString(line); err != nil {
		return "", 0, fmt.Errorf("write session_end: %w", err)
	}

	w.closed = true
	if err := w.file.Close(); err != nil {
		return "", 0, fmt.Errorf("close file: %w", err)
	}

	return evt.Hash, w.sequence + 1, nil
}

// Close closes the log file without writing a session_end event.
// Prefer Finalize for proper session completion.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true
	return w.file.Close()
}

// SessionID returns the session ID.
func (w *Writer) SessionID() string {
	return w.sessionID
}

// Sequence returns the current event sequence number.
func (w *Writer) Sequence() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.sequence
}
