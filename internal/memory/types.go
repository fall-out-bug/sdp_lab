package memory

import (
	"time"
)

// MemoryEntry represents a single memory record in the episodic memory store.
type MemoryEntry struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Actor     string            `json:"actor"`
	SessionID string            `json:"session_id"`
	FeatureID string            `json:"feature_id"`
	Phase     string            `json:"phase"`
	EntryType string            `json:"entry_type"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Summary represents a condensed view of a session's memory.
type Summary struct {
	SessionID    string      `json:"session_id"`
	Started      time.Time   `json:"started"`
	Ended        time.Time   `json:"ended"`
	Participants []string    `json:"participants"`
	EntryCount   int         `json:"entry_count"`
}
