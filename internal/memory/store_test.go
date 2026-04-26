package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) (store *Store, cleanup func()) {
	t.Helper()

	tmpDir := filepath.Join(os.TempDir(), "memory-test-"+t.Name())
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	cleanup = func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func generateTestEntry(entryType string) MemoryEntry {
	return MemoryEntry{
		ID:        generateUUID(),
		Timestamp: time.Now(),
		Actor:     "test-agent",
		SessionID: "test-session",
		FeatureID: "test-feature",
		Phase:     "testing",
		EntryType: entryType,
		Content:   "Test content for validation",
		Metadata:  map[string]string{"test": "value"},
	}
}

// TestValidateEntry tests the ValidateEntry function.
func TestValidateEntry(t *testing.T) {
	tests := []struct {
		name      string
		entry     MemoryEntry
		wantErr   error
	}{
		{
			name:    "valid entry",
			entry:   generateTestEntry("decision"),
			wantErr: nil,
		},
		{
			name:    "nil entry",
			entry:   MemoryEntry{},
			wantErr: ErrInvalidEntry,
		},
		{
			name: "missing ID",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.ID = ""
				return e
			}(),
			wantErr: ErrMissingID,
		},
		{
			name: "empty ID",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.ID = "   "
				return e
			}(),
			wantErr: ErrEmptyID,
		},
		{
			name: "missing actor",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.Actor = ""
				return e
			}(),
			wantErr: ErrMissingActor,
		},
		{
			name: "empty actor",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.Actor = "  "
				return e
			}(),
			wantErr: ErrEmptyActor,
		},
		{
			name: "missing session ID",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.SessionID = ""
				return e
			}(),
			wantErr: ErrMissingSessionID,
		},
		{
			name: "missing entry type",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.EntryType = ""
				return e
			}(),
			wantErr: ErrMissingEntryType,
		},
		{
			name: "invalid entry type",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.EntryType = "invalid_type"
				return e
			}(),
			wantErr: ErrInvalidEntryType,
		},
		{
			name: "missing content",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.Content = ""
				return e
			}(),
			wantErr: ErrMissingContent,
		},
		{
			name: "empty content",
			entry: func() MemoryEntry {
				e := generateTestEntry("decision")
				e.Content = "   "
				return e
			}(),
			wantErr: ErrEmptyContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntry(&tt.entry)
			if err != tt.wantErr {
				t.Errorf("ValidateEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestScanForSecrets tests the ScanForSecrets function.
func TestScanForSecrets(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantSecret string
	}{
		{
			name:       "clean content",
			content:    "This is just normal text with no secrets",
			wantFound:  false,
			wantSecret: "",
		},
		{
			name:       "AWS key",
			content:    "My AWS key is AKIA1234567890123456",
			wantFound:  true,
			wantSecret: "AWS Access Key",
		},
		{
			name:       "GitHub token",
			content:    "Token: ghp_1234567890abcdefghijklmnopqrstuvwxyz123456",
			wantFound:  true,
			wantSecret: "GitHub Personal Access Token (classic)",
		},
		{
			name:       "private key",
			content:    "-----BEGIN RSA PRIVATE KEY-----",
			wantFound:  true,
			wantSecret: "Private Key",
		},
		{
			name:       "JWT token",
			content:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			wantFound:  true,
			wantSecret: "JWT Token",
		},
		{
			name:       "benign phrase - token budget",
			content:    "We need to monitor the token budget carefully",
			wantFound:  false,
			wantSecret: "",
		},
		{
			name:       "benign phrase - password field",
			content:    "The password field should be encrypted",
			wantFound:  false,
			wantSecret: "",
		},
		{
			name:       "benign phrase - key value",
			content:    "Store this as a key value pair",
			wantFound:  false,
			wantSecret: "",
		},
		{
			name:       "benign phrase - access key id",
			content:    "The access key id field is required",
			wantFound:  false,
			wantSecret: "",
		},
		{
			name:       "actual secret in context",
			content:    "Here is the real key: AKIA1234567890123456 for production",
			wantFound:  true,
			wantSecret: "AWS Access Key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, secret := ScanForSecrets(tt.content)
			if found != tt.wantFound {
				t.Errorf("ScanForSecrets() found = %v, wantFound %v", found, tt.wantFound)
			}
			if secret != tt.wantSecret {
				t.Errorf("ScanForSecrets() secret = %v, wantSecret %v", secret, tt.wantSecret)
			}
		})
	}
}

// TestStoreAppendAndRead tests appending and reading entries.
func TestStoreAppendAndRead(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Test append
	entry := generateTestEntry("decision")
	if err := store.Append(entry); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	// Test read
	entries, err := store.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("ReadAll() returned %d entries, want 1", len(entries))
	}

	// Verify entry content
	readEntry := entries[0]
	if readEntry.ID != entry.ID {
		t.Errorf("ReadAll() ID = %v, want %v", readEntry.ID, entry.ID)
	}
	if readEntry.Actor != entry.Actor {
		t.Errorf("ReadAll() Actor = %v, want %v", readEntry.Actor, entry.Actor)
	}
	if readEntry.Content != entry.Content {
		t.Errorf("ReadAll() Content = %v, want %v", readEntry.Content, entry.Content)
	}
}

// TestStoreAppendSecret tests that secrets are rejected.
func TestStoreAppendSecret(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	entry := generateTestEntry("decision")
	entry.Content = "My key is AKIA1234567890123456"

	err := store.Append(entry)
	if err == nil {
		t.Error("Append() should reject secret content")
	}
	if err != nil && !strings.Contains(err.Error(), "secret detected") {
		t.Errorf("Append() error = %v, should mention secret detection", err)
	}
}

// TestQuery tests the Query function.
func TestQuery(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Add test entries
	entry1 := generateTestEntry("decision")
	entry1.Actor = "agent-1"
	entry1.SessionID = "session-1"

	entry2 := generateTestEntry("context")
	entry2.Actor = "agent-2"
	entry2.SessionID = "session-2"

	entry3 := generateTestEntry("observation")
	entry3.Actor = "agent-1"
	entry3.SessionID = "session-1"

	store.Append(entry1)
	store.Append(entry2)
	store.Append(entry3)

	// Test filter by actor
	opts := FilterOpts{Actor: "agent-1"}
	entries, err := Query(store, opts)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("Query() by actor returned %d entries, want 2", len(entries))
	}

	// Test filter by session ID
	opts = FilterOpts{SessionID: "session-2"}
	entries, err = Query(store, opts)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Query() by session returned %d entries, want 1", len(entries))
	}

	// Test filter by entry type
	opts = FilterOpts{EntryType: "decision"}
	entries, err = Query(store, opts)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Query() by type returned %d entries, want 1", len(entries))
	}

	// Test limit
	opts = FilterOpts{Limit: 2}
	entries, err = Query(store, opts)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("Query() with limit returned %d entries, want 2", len(entries))
	}

	// Test invalid limit
	opts = FilterOpts{Limit: -1}
	_, err = Query(store, opts)
	if err != ErrInvalidLimit {
		t.Errorf("Query() with invalid limit error = %v, want %v", err, ErrInvalidLimit)
	}
}

// TestRecent tests the Recent function.
func TestRecent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Add test entries
	for i := 0; i < 5; i++ {
		entry := generateTestEntry("decision")
		store.Append(entry)
	}

	// Test recent
	entries, err := Recent(store, 3)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("Recent() returned %d entries, want 3", len(entries))
	}

	// Test invalid limit
	_, err = Recent(store, 0)
	if err != ErrInvalidLimit {
		t.Errorf("Recent() with invalid limit error = %v, want %v", err, ErrInvalidLimit)
	}
}

// TestCompact tests the Compact function.
func TestCompact(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sessionID := "test-session"

	// Add entries for a session
	for i := 0; i < 3; i++ {
		entry := generateTestEntry("decision")
		entry.SessionID = sessionID
		if err := store.Append(entry); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	// Add entry for different session
	entry := generateTestEntry("context")
	entry.SessionID = "other-session"
	if err := store.Append(entry); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	// Compact the session
	if err := Compact(store, sessionID); err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Verify compaction
	entries, err := store.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Should have 2 entries: summary + other session entry
	if len(entries) != 2 {
		t.Errorf("After compaction, ReadAll() returned %d entries, want 2", len(entries))
	}

	// Check that archive file exists
	dir := filepath.Dir(store.path)
	archivePath := filepath.Join(dir, sessionID+".archive.jsonl")
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Compact() did not create archive file")
	}
}
