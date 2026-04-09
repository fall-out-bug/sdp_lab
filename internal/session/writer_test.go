package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriter_HashChain(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-chain-001"

	writer, err := NewWriter(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}

	// Write several events
	evt1, err := writer.AppendToolCall("read", json.RawMessage(`{"file_path": "a.txt"}`), []string{"a.txt"}, "00-059-01")
	if err != nil {
		t.Fatalf("append tool_call: %v", err)
	}

	evt2, err := writer.AppendGuardCheck("00-059-01", "write", []string{"b.txt"}, true, nil, "all files in scope")
	if err != nil {
		t.Fatalf("append guard_check: %v", err)
	}

	evt3, err := writer.AppendToolResult("write", true, "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("append tool_result: %v", err)
	}

	// Verify chain
	if evt1.PrevHash != "" {
		t.Error("first event should have empty prev_hash")
	}
	if evt2.PrevHash != evt1.Hash {
		t.Error("second event should link to first")
	}
	if evt3.PrevHash != evt2.Hash {
		t.Error("third event should link to second")
	}

	// Finalize
	rootHash, eventCount, err := writer.Finalize("completed")
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}

	if rootHash == "" {
		t.Error("root hash should not be empty")
	}
	if eventCount != 4 { // 3 events + session_end
		t.Errorf("expected 4 events, got %d", eventCount)
	}
}

func TestWriter_LogFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-exists-001"

	writer, err := NewWriter(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}

	_, err = writer.AppendToolCall("read", nil, nil, "")
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	_ = writer.Close()

	// Verify file exists
	logPath := filepath.Join(tmpDir, defaultLogDir, "session-"+sessionID+".jsonl")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file should exist: %v", err)
	}
}

// TestWriter_HashVerification verifies that each event's hash can be validated.
// Note: hashes include timestamps, so two events created at different times
// will have different hashes even with identical payloads.
func TestWriter_HashVerification(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-hash-verify-001"

	writer, err := NewWriter(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}

	// Append several events
	evt1, _ := writer.AppendToolCall("read", json.RawMessage(`{"file": "x.txt"}`), []string{"x.txt"}, "00-001")
	evt2, _ := writer.AppendGuardCheck("00-001", "write", []string{"y.txt"}, false, []string{"out of scope"}, "denied")
	evt3, _ := writer.AppendToolResult("write", false, "not allowed", nil)

	// Each event's hash should be valid
	if !evt1.ValidateHash() {
		t.Error("evt1 hash should be valid")
	}
	if !evt2.ValidateHash() {
		t.Error("evt2 hash should be valid")
	}
	if !evt3.ValidateHash() {
		t.Error("evt3 hash should be valid")
	}

	// Hash chain should be linked
	if evt2.PrevHash != evt1.Hash {
		t.Error("evt2 should link to evt1")
	}
	if evt3.PrevHash != evt2.Hash {
		t.Error("evt3 should link to evt2")
	}

	if _, _, err := writer.Finalize("completed"); err != nil {
		t.Fatalf("finalize: %v", err)
	}
}

func TestWriter_ValidJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-jsonl-001"

	writer, err := NewWriter(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}

	// Write events
	if _, err := writer.AppendToolCall("read", json.RawMessage(`{"file_path": "test.go"}`), []string{"test.go"}, "00-059"); err != nil {
		t.Fatalf("append tool_call: %v", err)
	}
	if _, err := writer.AppendGuardCheck("00-059", "write", []string{"test.go"}, true, nil, "ok"); err != nil {
		t.Fatalf("append guard_check: %v", err)
	}
	if _, _, err := writer.Finalize("completed"); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	// Read and parse each line
	logPath := filepath.Join(tmpDir, defaultLogDir, "session-"+sessionID+".jsonl")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var evt Event
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			t.Errorf("line %d is not valid JSON: %v\n  content: %s", i+1, err, line)
		}
		if evt.Hash == "" {
			t.Errorf("line %d missing hash", i+1)
		}
		if !evt.ValidateHash() {
			t.Errorf("line %d hash validation failed", i+1)
		}
	}
}

func TestWriter_Concurrency(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-concurrent-001"

	writer, err := NewWriter(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}

	// Write events concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			_, _ = writer.AppendToolCall("read", json.RawMessage(`{"n": `+string(rune('0'+n))+`}`), nil, "")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if _, _, err := writer.Finalize("completed"); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	// Verify no interleaving corruption
	logPath := filepath.Join(tmpDir, defaultLogDir, "session-"+sessionID+".jsonl")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	// 10 tool_call events + 1 session_end = 11
	if len(lines) != 11 {
		t.Errorf("expected 11 lines, got %d", len(lines))
	}

	// Each line should be valid JSON
	for i, line := range lines {
		var evt Event
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			t.Errorf("line %d is corrupted: %v", i+1, err)
		}
	}
}

func TestPaths(t *testing.T) {
	tmpDir := t.TempDir()
	paths := NewPaths(tmpDir)

	if paths.LogDir() != filepath.Join(tmpDir, ".sdp", "log") {
		t.Error("LogDir path mismatch")
	}
	if paths.CacheDir() != filepath.Join(tmpDir, ".sdp", "cache") {
		t.Error("CacheDir path mismatch")
	}
	if paths.MemDir() != filepath.Join(tmpDir, ".sdp", "mem") {
		t.Error("MemDir path mismatch")
	}
	logPath, err := paths.SessionLog("abc123")
	if err != nil {
		t.Fatalf("SessionLog: %v", err)
	}
	if logPath != filepath.Join(tmpDir, ".sdp", "log", "session-abc123.jsonl") {
		t.Error("SessionLog path mismatch")
	}

	// Ensure directories
	if err := paths.EnsureLogDir(); err != nil {
		t.Fatalf("EnsureLogDir: %v", err)
	}
	if err := paths.EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir: %v", err)
	}

	// Verify they exist
	if _, err := os.Stat(paths.LogDir()); err != nil {
		t.Error("LogDir should exist")
	}
	if _, err := os.Stat(paths.CacheDir()); err != nil {
		t.Error("CacheDir should exist")
	}
}
