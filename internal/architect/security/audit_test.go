package security

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLogger(t *testing.T) {
	logger := NewAuditLogger(100)
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.entries)
	assert.Equal(t, 100, logger.maxEntries)
}

func TestNewAuditLogger_NoLimit(t *testing.T) {
	logger := NewAuditLogger(0)
	assert.NotNil(t, logger)
	assert.Equal(t, 0, logger.maxEntries) // No limit
}

func TestAuditLogger_LogEntry(t *testing.T) {
	logger := NewAuditLogger(10)

	entry := AuditLogEntry{
		RequestID:         "test-123",
		Provider:          "openai",
		Model:             "gpt-4",
		InputLength:       100,
		OutputLength:      200,
		RedactionCounts:   map[string]int{"aws_key": 1, "email": 2},
		SanitizationStages: []string{"scrub", "sanitize", "wrap"},
		Latency:           100 * time.Millisecond,
		Success:           true,
	}

	logger.LogEntry(entry)

	entries := logger.GetEntries()
	assert.Len(t, entries, 1)

	assert.Equal(t, "test-123", entries[0].RequestID)
	assert.Equal(t, "openai", entries[0].Provider)
	assert.Equal(t, "gpt-4", entries[0].Model)
	assert.True(t, entries[0].Success)
}

func TestAuditLogger_LogEntry_AutoTimestamp(t *testing.T) {
	logger := NewAuditLogger(10)

	before := time.Now()

	entry := AuditLogEntry{
		RequestID: "test-456",
		Provider:  "anthropic",
		Model:     "claude-3",
		Success:   true,
	}

	logger.LogEntry(entry)

	after := time.Now()

	entries := logger.GetEntries()
	require.Len(t, entries, 1)

	assert.False(t, entries[0].Timestamp.IsZero())
	assert.True(t, entries[0].Timestamp.After(before) || entries[0].Timestamp.Equal(before))
	assert.True(t, entries[0].Timestamp.Before(after) || entries[0].Timestamp.Equal(after))
}

func TestAuditLogger_LogEntry_RedactionTotal(t *testing.T) {
	logger := NewAuditLogger(10)

	entry := AuditLogEntry{
		RequestID: "test-789",
		Provider:  "openai",
		RedactionCounts: map[string]int{
			"aws_key":  3,
			"email":    2,
			"api_key":  1,
		},
		Success: true,
	}

	logger.LogEntry(entry)

	entries := logger.GetEntries()
	require.Len(t, entries, 1)

	// Total should be sum of all counts
	assert.Equal(t, 6, entries[0].RedactionsTotal)
}

func TestAuditLogger_GetEntries(t *testing.T) {
	logger := NewAuditLogger(10)

	// Log multiple entries
	for i := 0; i < 5; i++ {
		logger.LogEntry(AuditLogEntry{
			RequestID: fmt.Sprintf("req-%d", i),
			Provider:  "test",
			Success:   true,
		})
	}

	entries := logger.GetEntries()
	assert.Len(t, entries, 5)
}

func TestAuditLogger_GetEntries_ReturnsCopy(t *testing.T) {
	logger := NewAuditLogger(10)

	logger.LogEntry(AuditLogEntry{
		RequestID: "original",
		Provider:  "test",
		Success:   true,
	})

	entries := logger.GetEntries()
	require.Len(t, entries, 1)

	// Modify the returned entries
	entries[0].RequestID = "modified"

	// Original should be unchanged
	originalEntries := logger.GetEntries()
	assert.Equal(t, "original", originalEntries[0].RequestID)
}

func TestAuditLogger_GetEntriesByRequestID(t *testing.T) {
	logger := NewAuditLogger(10)

	// Log entries for different request IDs
	logger.LogEntry(AuditLogEntry{
		RequestID: "req-1",
		Provider:  "openai",
		Success:   true,
	})

	logger.LogEntry(AuditLogEntry{
		RequestID: "req-2",
		Provider:  "anthropic",
		Success:   true,
	})

	logger.LogEntry(AuditLogEntry{
		RequestID: "req-1",
		Provider:  "openai",
		Success:   false,
	})

	// Get entries for req-1
	entries := logger.GetEntriesByRequestID("req-1")
	assert.Len(t, entries, 2)

	// Get entries for req-2
	entries = logger.GetEntriesByRequestID("req-2")
	assert.Len(t, entries, 1)

	// Get entries for non-existent request
	entries = logger.GetEntriesByRequestID("req-999")
	assert.Len(t, entries, 0)
}

func TestAuditLogger_GetRedactionSummary(t *testing.T) {
	logger := NewAuditLogger(10)

	// Log entries with different redaction types
	logger.LogEntry(AuditLogEntry{
		RequestID: "req-1",
		RedactionCounts: map[string]int{
			"aws_key": 2,
			"email":   1,
		},
		Success: true,
	})

	logger.LogEntry(AuditLogEntry{
		RequestID: "req-2",
		RedactionCounts: map[string]int{
			"aws_key": 1,
			"api_key": 3,
		},
		Success: true,
	})

	summary := logger.GetRedactionSummary()

	assert.Equal(t, 3, summary["aws_key"])
	assert.Equal(t, 1, summary["email"])
	assert.Equal(t, 3, summary["api_key"])
}

func TestAuditLogger_Clear(t *testing.T) {
	logger := NewAuditLogger(10)

	// Log some entries
	for i := 0; i < 5; i++ {
		logger.LogEntry(AuditLogEntry{
			RequestID: fmt.Sprintf("req-%d", i),
			Success:   true,
		})
	}

	assert.Len(t, logger.GetEntries(), 5)

	logger.Clear()

	assert.Len(t, logger.GetEntries(), 0)
}

func TestAuditLogger_MaxEntries(t *testing.T) {
	logger := NewAuditLogger(3)

	// Log more entries than max
	for i := 0; i < 5; i++ {
		logger.LogEntry(AuditLogEntry{
			RequestID: fmt.Sprintf("req-%d", i),
			Success:   true,
		})
	}

	entries := logger.GetEntries()

	// Should only have the last 3 entries (req-2, req-3, req-4)
	assert.Len(t, entries, 3)

	// Oldest entries should be removed
	requestIDs := make([]string, len(entries))
	for i, e := range entries {
		requestIDs[i] = e.RequestID
	}

	assert.NotContains(t, requestIDs, "req-0")
	assert.NotContains(t, requestIDs, "req-1")
	assert.Contains(t, requestIDs, "req-2")
	assert.Contains(t, requestIDs, "req-3")
	assert.Contains(t, requestIDs, "req-4")
}

func TestAuditLogger_GetStats(t *testing.T) {
	logger := NewAuditLogger(10)

	// Log various entries
	logger.LogEntry(AuditLogEntry{
		RequestID: "req-1",
		Provider:  "openai",
		RedactionCounts: map[string]int{"aws_key": 2},
		Success:   true,
	})

	logger.LogEntry(AuditLogEntry{
		RequestID: "req-2",
		Provider:  "anthropic",
		RedactionCounts: map[string]int{"email": 1},
		Success:   true,
	})

	logger.LogEntry(AuditLogEntry{
		RequestID: "req-3",
		Provider:  "openai",
		RedactionCounts: map[string]int{"api_key": 3},
		Success:   false,
	})

	stats := logger.GetStats()

	assert.Equal(t, 3, stats.TotalEntries)
	assert.Equal(t, 6, stats.TotalRedactions)
	assert.Equal(t, 2, stats.TotalRequests["openai"])
	assert.Equal(t, 1, stats.TotalRequests["anthropic"])
	assert.Equal(t, 1, stats.TotalErrors)
	assert.Equal(t, 2, stats.RedactionByType["aws_key"])
	assert.Equal(t, 1, stats.RedactionByType["email"])
	assert.Equal(t, 3, stats.RedactionByType["api_key"])
}

func TestAuditStats_String(t *testing.T) {
	stats := AuditStats{
		TotalEntries:    10,
		TotalRedactions: 25,
		TotalRequests:   map[string]int{"openai": 7, "anthropic": 3},
		TotalErrors:     2,
		RedactionByType: map[string]int{"aws_key": 10, "email": 15},
	}

	str := stats.String()
	assert.Contains(t, str, "Entries: 10")
	assert.Contains(t, str, "Redactions: 25")
	assert.Contains(t, str, "Errors: 2")
}

func TestAuditLogger_ExportJSON(t *testing.T) {
	logger := NewAuditLogger(10)

	logger.LogEntry(AuditLogEntry{
		RequestID: "req-1",
		Provider:  "openai",
		Model:     "gpt-4",
		Success:   true,
	})

	data, err := logger.ExportJSON()
	require.NoError(t, err)

	var entries []AuditLogEntry
	err = json.Unmarshal(data, &entries)
	require.NoError(t, err)

	assert.Len(t, entries, 1)
	assert.Equal(t, "req-1", entries[0].RequestID)
}

func TestAuditLogger_ConcurrentLogging(t *testing.T) {
	logger := NewAuditLogger(100)

	// Log entries concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			logger.LogEntry(AuditLogEntry{
				RequestID: fmt.Sprintf("concurrent-%d", n),
				Success:   true,
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	entries := logger.GetEntries()
	assert.Len(t, entries, 10)
}

func TestAuditLogger_EmptyLogger(t *testing.T) {
	logger := NewAuditLogger(10)

	entries := logger.GetEntries()
	assert.Empty(t, entries)

	summary := logger.GetRedactionSummary()
	assert.Empty(t, summary)

	stats := logger.GetStats()
	assert.Equal(t, 0, stats.TotalEntries)
}

func TestAuditLogEntry_Metadata(t *testing.T) {
	logger := NewAuditLogger(10)

	entry := AuditLogEntry{
		RequestID: "req-1",
		Provider:  "openai",
		Success:   true,
		Metadata: map[string]interface{}{
			"tokens_used":    1500,
			"model_version": "gpt-4-0314",
			"custom_field":  "custom_value",
		},
	}

	logger.LogEntry(entry)

	entries := logger.GetEntries()
	require.Len(t, entries, 1)

	assert.Equal(t, 1500, entries[0].Metadata["tokens_used"])
	assert.Equal(t, "gpt-4-0314", entries[0].Metadata["model_version"])
	assert.Equal(t, "custom_value", entries[0].Metadata["custom_field"])
}

func TestAuditLogger_ErrorLogging(t *testing.T) {
	logger := NewAuditLogger(10)

	entry := AuditLogEntry{
		RequestID: "req-error",
		Provider:  "openai",
		Model:     "gpt-4",
		Success:   false,
		Error:     "rate limit exceeded",
		Latency:   5 * time.Second,
	}

	logger.LogEntry(entry)

	entries := logger.GetEntries()
	require.Len(t, entries, 1)

	assert.False(t, entries[0].Success)
	assert.Equal(t, "rate limit exceeded", entries[0].Error)
	assert.Equal(t, 5*time.Second, entries[0].Latency)
}

func TestAuditLogger_SanitizationStages(t *testing.T) {
	logger := NewAuditLogger(10)

	entry := AuditLogEntry{
		RequestID:         "req-1",
		Provider:          "openai",
		Success:           true,
		SanitizationStages: []string{"scrub", "sanitize", "wrap", "api"},
	}

	logger.LogEntry(entry)

	entries := logger.GetEntries()
	require.Len(t, entries, 1)

	assert.Equal(t, []string{"scrub", "sanitize", "wrap", "api"}, entries[0].SanitizationStages)
}
