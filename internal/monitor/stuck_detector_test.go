package monitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStuckDetector(t *testing.T) {
	tmpDir := t.TempDir()
	
	cfg := StuckDetectorConfig{
		SessionPath:   tmpDir,
		Timeout:       time.Minute,
		CheckInterval: time.Second * 10,
	}
	
	sd, err := NewStuckDetector(cfg)
	if err != nil {
		t.Fatalf("NewStuckDetector failed: %v", err)
	}
	
	if sd.sessionPath != tmpDir {
		t.Errorf("sessionPath = %q, want %q", sd.sessionPath, tmpDir)
	}
	
	if sd.timeout != time.Minute {
		t.Errorf("timeout = %v, want %v", sd.timeout, time.Minute)
	}
}

func TestStuckDetectorDefaults(t *testing.T) {
	sd, err := NewStuckDetector(StuckDetectorConfig{})
	if err != nil {
		t.Fatalf("NewStuckDetector failed: %v", err)
	}
	
	if sd.timeout != DefaultStuckTimeout {
		t.Errorf("timeout = %v, want %v", sd.timeout, DefaultStuckTimeout)
	}
	
	if sd.checkTicker == nil {
		t.Error("checkTicker not initialized")
	}
}

func TestStuckDetection(t *testing.T) {
	tmpDir := t.TempDir()
	
	stuckCalled := false
	var stuckSessionID string
	var stuckLastEvent time.Time
	
	cfg := StuckDetectorConfig{
		SessionPath:   tmpDir,
		Timeout:       time.Second * 2,
		CheckInterval: time.Millisecond * 100,
		OnStuck: func(sessionID string, lastEvent time.Time) {
			stuckCalled = true
			stuckSessionID = sessionID
			stuckLastEvent = lastEvent
		},
		OnRecovered: func(sessionID string) {
			// Session recovered
		},
	}
	
	sd, err := NewStuckDetector(cfg)
	if err != nil {
		t.Fatalf("NewStuckDetector failed: %v", err)
	}
	
	// Create a session file with old timestamp
	sessionFile := filepath.Join(tmpDir, "session-test123.jsonl")
	oldContent := `{"type":"session_start","timestamp":"2020-01-01T00:00:00Z"}
{"type":"tool_call","timestamp":"2020-01-01T00:01:00Z"}
`
	if err := os.WriteFile(sessionFile, []byte(oldContent), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	
	// Set file modification time to be old
	oldTime := time.Now().Add(-time.Minute * 10)
	if err := os.Chtimes(sessionFile, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes failed: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	
	sd.Start(ctx)
	
	// Wait for detection
	time.Sleep(time.Millisecond * 500)
	
	if !stuckCalled {
		t.Error("OnStuck was not called")
	}
	
	if stuckSessionID != "test123" {
		t.Errorf("stuckSessionID = %q, want %q", stuckSessionID, "test123")
	}
	
	if stuckLastEvent.IsZero() {
		t.Error("stuckLastEvent should not be zero")
	}
	
	// Verify session is marked as stuck
	if !sd.IsStuck("test123") {
		t.Error("session should be marked as stuck")
	}
	
	stats := sd.Stats()
	if stats.StuckCount != 1 {
		t.Errorf("StuckCount = %d, want 1", stats.StuckCount)
	}
}

func TestStuckDetectorStats(t *testing.T) {
	tmpDir := t.TempDir()
	
	cfg := StuckDetectorConfig{
		SessionPath: tmpDir,
		Timeout:     time.Minute,
	}
	
	sd, err := NewStuckDetector(cfg)
	if err != nil {
		t.Fatalf("NewStuckDetector failed: %v", err)
	}
	
	stats := sd.Stats()
	if stats.StuckCount != 0 {
		t.Errorf("initial StuckCount = %d, want 0", stats.StuckCount)
	}
	
	if stats.Timeout != time.Minute {
		t.Errorf("Timeout = %v, want %v", stats.Timeout, time.Minute)
	}
}

func TestGetLastEventTime(t *testing.T) {
	tmpDir := t.TempDir()
	
	sd, err := NewStuckDetector(StuckDetectorConfig{SessionPath: tmpDir})
	if err != nil {
		t.Fatalf("NewStuckDetector failed: %v", err)
	}
	
	// Create a session file
	sessionFile := filepath.Join(tmpDir, "session-abc.jsonl")
	content := `{"type":"session_start","timestamp":"2024-01-01T10:00:00Z"}
{"type":"tool_call","timestamp":"2024-01-01T10:05:00Z"}
{"type":"tool_result","timestamp":"2024-01-01T10:05:30Z"}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	
	sessionID, lastEvent, err := sd.getLastEventTime(sessionFile)
	if err != nil {
		t.Fatalf("getLastEventTime failed: %v", err)
	}
	
	if sessionID != "abc" {
		t.Errorf("sessionID = %q, want %q", sessionID, "abc")
	}
	
	// lastEvent should be approximately now (file modification time)
	// since the backwards reading is best-effort
	if lastEvent.IsZero() {
		t.Error("lastEvent should not be zero")
	}
}
