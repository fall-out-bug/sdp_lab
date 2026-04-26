package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Compact compacts all entries for a given session into a summary.
// Old entries are archived and a summary entry is created.
func Compact(store *Store, sessionID string) error {
	// Get all entries for the session
	opts := FilterOpts{
		SessionID: sessionID,
	}
	entries, err := Query(store, opts)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return fmt.Errorf("no entries found for session: %s", sessionID)
	}

	// Sort by timestamp
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	// Create summary
	summary := Summary{
		SessionID:    sessionID,
		Started:      entries[0].Timestamp,
		Ended:        entries[len(entries)-1].Timestamp,
		EntryCount:   len(entries),
		Participants: extractParticipants(entries),
	}

	// Archive old entries
	if err := archiveEntries(store, sessionID, entries); err != nil {
		return fmt.Errorf("failed to archive entries: %w", err)
	}

	// Create summary entry
	summaryEntry := MemoryEntry{
		ID:        generateUUID(),
		Timestamp: summary.Ended,
		Actor:     "system",
		SessionID: sessionID,
		FeatureID: "",
		Phase:     "compaction",
		EntryType: "decision",
		Content:   formatSummaryContent(summary),
		Metadata: map[string]string{
			"compacted_entry_count": fmt.Sprintf("%d", summary.EntryCount),
			"archived":              "true",
		},
	}

	return store.Append(summaryEntry)
}

// extractParticipants extracts unique actors from entries.
func extractParticipants(entries []MemoryEntry) []string {
	participantMap := make(map[string]bool)
	for _, entry := range entries {
		participantMap[entry.Actor] = true
	}

	var participants []string
	for actor := range participantMap {
		participants = append(participants, actor)
	}

	sort.Strings(participants)
	return participants
}

// formatSummaryContent creates a human-readable summary content.
func formatSummaryContent(summary Summary) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Session Summary: %s\n\n", summary.SessionID))
	sb.WriteString(fmt.Sprintf("Duration: %s to %s\n", summary.Started.Format("2006-01-02 15:04:05"), summary.Ended.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Total Entries: %d\n", summary.EntryCount))
	sb.WriteString(fmt.Sprintf("Participants: %s\n", strings.Join(summary.Participants, ", ")))

	return sb.String()
}

// archiveEntries moves entries for a session to an archive file.
func archiveEntries(store *Store, sessionID string, entries []MemoryEntry) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.isClosed {
		return ErrStoreClosed
	}

	// Create archive path
	dir := filepath.Dir(store.path)
	archivePath := filepath.Join(dir, fmt.Sprintf("%s.archive.jsonl", sessionID))

	// Write entries to archive
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer archiveFile.Close()

	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		data = append(data, '\n')
		if _, err := archiveFile.Write(data); err != nil {
			return err
		}
	}

	// Rewrite store file excluding archived entries
	return rewriteStoreExcluding(store.path, sessionID)
}

// rewriteStoreExcluding rewrites the store file excluding entries for a session.
func rewriteStoreExcluding(storePath, sessionID string) error {
	// Read all entries
	data, err := os.ReadFile(storePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry MemoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Keep only entries not from the archived session
		if entry.SessionID != sessionID {
			newLines = append(newLines, line)
		}
	}

	// Write back
	newData := []byte(strings.Join(newLines, "\n") + "\n")
	return os.WriteFile(storePath, newData, 0644)
}
