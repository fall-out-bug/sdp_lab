package omoclient

import "time"

// EventClass represents the type of OmO event
type EventClass string

const (
	// EventToolStarted is emitted when an agent tool invocation begins
	EventToolStarted EventClass = "tool.started"
	// EventToolCompleted is emitted when an agent tool invocation completes
	EventToolCompleted EventClass = "tool.completed"
	// EventCompletionSucceeded indicates the overall invocation succeeded
	EventCompletionSucceeded EventClass = "completion.succeeded"
	// EventCompletionFailed indicates the overall invocation failed
	EventCompletionFailed EventClass = "completion.failed"
	// EventWarning indicates a non-fatal warning condition
	EventWarning EventClass = "warning"
	// EventUnknown is for unrecognized event types
	EventUnknown EventClass = "unknown"
)

// SessionInfo contains session metadata from OmO serve
type SessionInfo struct {
	ID        string    `json:"id"`
	Project   string    `json:"project,omitempty"`
	Session   string    `json:"session,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// CreateSessionRequest creates a new OmO serve session
type CreateSessionRequest struct {
	Project string `json:"project"`
	Session string `json:"session,omitempty"`
}

// SendMessageRequest sends a message to an OmO serve session
type SendMessageRequest struct {
	Content string `json:"content"`
}

// OmOEvent represents an event from OmO serve SSE stream
type OmOEvent struct {
	Class     EventClass `json:"class"`
	Prefix    string     `json:"prefix,omitempty"`
	Data      string     `json:"data,omitempty"`
	Timestamp time.Time  `json:"timestamp,omitempty"`
}
