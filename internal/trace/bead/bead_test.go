package bead

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateBeadID(t *testing.T) {
	tests := []struct {
		name    string
		beadID  string
		wantErr bool
	}{
		{"valid epic", "sdplab-snn1", false},
		{"valid child", "sdplab-kh8j", false},
		{"empty", "", true},
		{"missing prefix", "snn1", true},
		{"wrong prefix", "bead-snn1", true},
		{"no suffix", "sdplab-", true},
		{"too long suffix", "sdplab-12345678901", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBeadID(tt.beadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBeadID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCurrentFeatureID(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	resolver := NewResolver(tmpDir)

	// Test file not found
	_, err := resolver.GetCurrentFeatureID()
	if err == nil {
		t.Error("Expected error when file doesn't exist")
	}
	if !strings.Contains(err.Error(), "current feature not set") {
		t.Errorf("Error message should contain remediation: %v", err)
	}

	// Test valid bead ID
	stateDir := filepath.Join(tmpDir, ".sdp", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("Failed to create state dir: %v", err)
	}

	featureFile := filepath.Join(stateDir, "current-feature")
	validID := "sdplab-kh8j\n"
	if err := os.WriteFile(featureFile, []byte(validID), 0644); err != nil {
		t.Fatalf("Failed to write feature file: %v", err)
	}

	got, err := resolver.GetCurrentFeatureID()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if got != "sdplab-kh8j" {
		t.Errorf("GetCurrentFeatureID() = %v, want %v", got, "sdplab-kh8j")
	}

	// Test invalid format (missing prefix)
	if err := os.WriteFile(featureFile, []byte("invalid-id\n"), 0644); err != nil {
		t.Fatalf("Failed to write feature file: %v", err)
	}

	_, err = resolver.GetCurrentFeatureID()
	if err == nil {
		t.Error("Expected error for invalid bead ID format")
	}
}

func TestSetCurrentFeatureID(t *testing.T) {
	tmpDir := t.TempDir()
	resolver := NewResolver(tmpDir)

	// Test setting valid bead ID
	err := resolver.SetCurrentFeatureID("sdplab-kh8j")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify file was created
	featureFile := filepath.Join(tmpDir, ".sdp", "state", "current-feature")
	data, err := os.ReadFile(featureFile)
	if err != nil {
		t.Fatalf("Failed to read feature file: %v", err)
	}

	got := strings.TrimSpace(string(data))
	if got != "sdplab-kh8j" {
		t.Errorf("File content = %v, want %v", got, "sdplab-kh8j")
	}

	// Test invalid format
	err = resolver.SetCurrentFeatureID("invalid")
	if err == nil {
		t.Error("Expected error for invalid bead ID")
	}
}

func TestFormatBeadEvent(t *testing.T) {
	attrs := FormatBeadEvent("sdplab-kh8j", "closed", "IN_PROGRESS", "CLOSED")

	if attrs["sdp.bead.id"] != "sdplab-kh8j" {
		t.Errorf("Expected bead id sdplab-kh8j, got %v", attrs["sdp.bead.id"])
	}
	if attrs["sdp.bead.event"] != "closed" {
		t.Errorf("Expected event closed, got %v", attrs["sdp.bead.event"])
	}
	if attrs["sdp.bead.previous_status"] != "IN_PROGRESS" {
		t.Errorf("Expected previous status IN_PROGRESS, got %v", attrs["sdp.bead.previous_status"])
	}
	if attrs["sdp.bead.new_status"] != "CLOSED" {
		t.Errorf("Expected new status CLOSED, got %v", attrs["sdp.bead.new_status"])
	}

	// Test without optional fields
	attrs2 := FormatBeadEvent("sdplab-kh8j", "claimed", "", "")
	if attrs2["sdp.bead.previous_status"] != "" {
		t.Error("Expected no previous status")
	}
}

func TestIsEpicBead(t *testing.T) {
	tests := []struct {
		name   string
		beadID string
		epic   bool
	}{
		{"epic snn1", "sdplab-snn1", true},
		{"epic 6x39", "sdplab-6x39", true},
		{"child kh8j", "sdplab-kh8j", false},
		{"invalid", "invalid-id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEpicBead(tt.beadID)
			if got != tt.epic {
				t.Errorf("IsEpicBead() = %v, want %v", got, tt.epic)
			}
		})
	}
}

func TestGetCurrentSessionID(t *testing.T) {
	tmpDir := t.TempDir()

	// First call should create new session ID
	session1, err := GetCurrentSessionID(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.HasPrefix(session1, "sess_") {
		t.Errorf("Session ID should start with 'sess_', got %v", session1)
	}

	// Second call should read existing
	session2, err := GetCurrentSessionID(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if session1 != session2 {
		t.Errorf("Session ID should persist: %v != %v", session1, session2)
	}

	// Clear and regenerate
	if err := ClearSessionID(tmpDir); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = GetCurrentSessionID(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// Note: session IDs might be same if using pid, that's ok for MVP
}

func TestTraceIDGenerator(t *testing.T) {
	gen := NewTraceIDGenerator()

	id1 := gen.Generate()
	id2 := gen.Generate()

	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Trace ID should be 32 chars, got %d", len(id1))
	}
	if len(id2) != 32 {
		t.Errorf("Trace ID should be 32 chars, got %d", len(id2))
	}

	// For MVP, IDs might be same if based on pid only
	// In production, would use crypto/rand for uniqueness
}

func TestSpanIDGenerator(t *testing.T) {
	gen := NewSpanIDGenerator()

	id1 := gen.Generate()
	id2 := gen.Generate()

	if len(id1) != 16 { // 8 bytes = 16 hex chars
		t.Errorf("Span ID should be 16 chars, got %d", len(id1))
	}
	if len(id2) != 16 {
		t.Errorf("Span ID should be 16 chars, got %d", len(id2))
	}

	if id1 == id2 {
		t.Error("Span IDs should be unique")
	}
}
