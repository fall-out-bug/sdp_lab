package security

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AuditLogEntry represents a single audit log entry for LLM calls.
// It tracks redactions without logging actual secret content.
type AuditLogEntry struct {
	Timestamp         time.Time              `json:"timestamp"`
	RequestID         string                 `json:"request_id"`
	Provider          string                 `json:"provider"`
	Model             string                 `json:"model"`
	InputLength       int                    `json:"input_length"`
	OutputLength      int                    `json:"output_length"`
	RedactionCounts   map[string]int         `json:"redaction_counts"` // type -> count
	RedactionsTotal   int                    `json:"redactions_total"`
	SanitizationStages []string              `json:"sanitization_stages"`
	Latency           time.Duration          `json:"latency"`
	Success           bool                   `json:"success"`
	Error             string                 `json:"error,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger provides thread-safe audit logging for LLM interactions.
type AuditLogger struct {
	mu     sync.RWMutex
	entries []AuditLogEntry
	maxEntries int
}

// NewAuditLogger creates a new audit logger with optional size limit.
// If maxEntries <= 0, no limit is enforced.
func NewAuditLogger(maxEntries int) *AuditLogger {
	return &AuditLogger{
		entries:   make([]AuditLogEntry, 0, 100),
		maxEntries: maxEntries,
	}
}

// LogEntry records an audit log entry.
func (al *AuditLogger) LogEntry(entry AuditLogEntry) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Ensure timestamp is set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Calculate total redactions
	entry.RedactionsTotal = 0
	for _, count := range entry.RedactionCounts {
		entry.RedactionsTotal += count
	}

	al.entries = append(al.entries, entry)

	// Enforce size limit if configured
	if al.maxEntries > 0 && len(al.entries) > al.maxEntries {
		// Remove oldest entries
		overflow := len(al.entries) - al.maxEntries
		al.entries = al.entries[overflow:]
	}
}

// GetEntries returns a copy of all audit entries.
func (al *AuditLogger) GetEntries() []AuditLogEntry {
	al.mu.RLock()
	defer al.mu.RUnlock()

	entries := make([]AuditLogEntry, len(al.entries))
	copy(entries, al.entries)
	return entries
}

// GetEntriesByRequestID returns entries for a specific request ID.
func (al *AuditLogger) GetEntriesByRequestID(requestID string) []AuditLogEntry {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var result []AuditLogEntry
	for _, e := range al.entries {
		if e.RequestID == requestID {
			result = append(result, e)
		}
	}
	return result
}

// GetRedactionSummary returns a summary of redactions by type.
func (al *AuditLogger) GetRedactionSummary() map[string]int {
	al.mu.RLock()
	defer al.mu.RUnlock()

	summary := make(map[string]int)
	for _, e := range al.entries {
		for typ, count := range e.RedactionCounts {
			summary[typ] += count
		}
	}
	return summary
}

// Clear removes all audit entries.
func (al *AuditLogger) Clear() {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.entries = make([]AuditLogEntry, 0, 100)
}

// ExportJSON exports all audit entries as JSON.
func (al *AuditLogger) ExportJSON() ([]byte, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	return json.MarshalIndent(al.entries, "", "  ")
}

// GetStats returns statistics about audit log entries.
func (al *AuditLogger) GetStats() AuditStats {
	al.mu.RLock()
	defer al.mu.RUnlock()

	stats := AuditStats{
		TotalEntries:      len(al.entries),
		TotalRedactions:   0,
		TotalRequests:     make(map[string]int),
		TotalErrors:       0,
		RedactionByType:   make(map[string]int),
	}

	for _, e := range al.entries {
		stats.TotalRedactions += e.RedactionsTotal
		stats.TotalRequests[e.Provider]++

		if !e.Success {
			stats.TotalErrors++
		}

		for typ, count := range e.RedactionCounts {
			stats.RedactionByType[typ] += count
		}
	}

	return stats
}

// AuditStats provides summary statistics for audit logs.
type AuditStats struct {
	TotalEntries    int            `json:"total_entries"`
	TotalRedactions int            `json:"total_redactions"`
	TotalRequests   map[string]int `json:"total_requests"` // provider -> count
	TotalErrors     int            `json:"total_errors"`
	RedactionByType map[string]int `json:"redaction_by_type"` // type -> count
}

// String returns a formatted summary of audit statistics.
func (s AuditStats) String() string {
	return fmt.Sprintf("Entries: %d, Redactions: %d, Requests: %v, Errors: %d, Types: %v",
		s.TotalEntries, s.TotalRedactions, s.TotalRequests, s.TotalErrors, s.RedactionByType)
}
