package bead

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// CurrentFeatureFile is the canonical source for current epic bead ID
	CurrentFeatureFile = ".sdp/state/current-feature"
)

// Resolver resolves bead IDs from the filesystem
type Resolver struct {
	projectRoot string
}

// NewResolver creates a new bead resolver
func NewResolver(projectRoot string) *Resolver {
	return &Resolver{
		projectRoot: projectRoot,
	}
}

// GetCurrentFeatureID reads the current epic bead ID from .sdp/state/current-feature
// Returns error with remediation message if file doesn't exist
func (r *Resolver) GetCurrentFeatureID() (string, error) {
	path := filepath.Join(r.projectRoot, CurrentFeatureFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", r.newFeatureNotSetError()
		}
		return "", fmt.Errorf("failed to read current feature file: %w", err)
	}

	// File format: exactly one line with <bead_id>\n
	beadID := strings.TrimSpace(string(data))
	if beadID == "" {
		return "", r.newFeatureNotSetError()
	}

	// Validate bead ID format (basic check: sdplab-xxxx)
	if !strings.HasPrefix(beadID, "sdplab-") {
		return "", fmt.Errorf("invalid bead ID format in %s: %s (must start with 'sdplab-')", path, beadID)
	}

	return beadID, nil
}

// SetCurrentFeatureID writes the epic bead ID to .sdp/state/current-feature
func (r *Resolver) SetCurrentFeatureID(beadID string) error {
	if !strings.HasPrefix(beadID, "sdplab-") {
		return fmt.Errorf("invalid bead ID format: %s (must start with 'sdplab-')", beadID)
	}

	path := filepath.Join(r.projectRoot, CurrentFeatureFile)

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write bead ID with newline
	data := []byte(beadID + "\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write current feature file: %w", err)
	}

	return nil
}

// ValidateBeadID checks if a bead ID is valid
func ValidateBeadID(beadID string) error {
	if beadID == "" {
		return fmt.Errorf("bead ID cannot be empty")
	}

	if !strings.HasPrefix(beadID, "sdplab-") {
		return fmt.Errorf("bead ID must start with 'sdplab-', got: %s", beadID)
	}

	// Check length: sdplab-xxxx (typically 4 chars after dash)
	parts := strings.Split(beadID, "-")
	if len(parts) != 2 {
		return fmt.Errorf("bead ID must be in format 'sdplab-xxxx', got: %s", beadID)
	}

	if len(parts[1]) < 1 || len(parts[1]) > 10 {
		return fmt.Errorf("bead ID suffix invalid length: %s", beadID)
	}

	return nil
}

// newFeatureNotSetError returns an error with remediation message
func (r *Resolver) newFeatureNotSetError() error {
	return fmt.Errorf(`current feature not set: %s does not exist

Remediation:
  1. Start a delivery loop: @deliver or @oneshot
  2. Or manually set: echo "sdplab-xxxx" > %s

The trace system requires an active feature to associate telemetry with.`,
		CurrentFeatureFile,
		CurrentFeatureFile,
	)
}

// FindProjectRoot searches up the directory tree for .sdp directory
// Returns current directory if not found (for testing)
func FindProjectRoot(startDir string) string {
	dir := startDir
	for {
		sdpPath := filepath.Join(dir, ".sdp")
		if info, err := os.Stat(sdpPath); err == nil && info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, return start dir
			return startDir
		}
		dir = parent
	}
}

// ReadBeadMetadata reads additional metadata from beads state
// This is optional for MVP but useful for enrichment
type BeadMetadata struct {
	ID       string
	Status   string
	Feature  string
	Priority string
}

// ReadBeadNote reads a note from a bead (for trace integrity in v2)
func ReadBeadNote(projectRoot, beadID, noteKey string) (string, error) {
	// For MVP, this is a stub
	// In v2, this will read from beads dolt database
	return "", fmt.Errorf("not implemented in MVP")
}

// WriteBeadNote writes a note to a bead (for trace integrity in v2)
func WriteBeadNote(projectRoot, beadID, noteKey, noteValue string) error {
	// For MVP, this is a stub
	// In v2, this will write to beads dolt database
	return fmt.Errorf("not implemented in MVP")
}

// FormatBeadEvent creates a bead event attribute map
func FormatBeadEvent(beadID, event string, previousStatus, newStatus string) map[string]string {
	attrs := map[string]string{
		"sdp.bead.id":     beadID,
		"sdp.bead.event":  event,
	}

	if previousStatus != "" {
		attrs["sdp.bead.previous_status"] = previousStatus
	}
	if newStatus != "" {
		attrs["sdp.bead.new_status"] = newStatus
	}

	return attrs
}

// IsEpicBead checks if a bead ID is an epic (parent bead)
// For MVP, assumes epic beads have pattern sdplab-snn1, sdplab-6x39, etc.
// This is heuristic; real check would query beads database
func IsEpicBead(beadID string) bool {
	// For MVP, assume beads with 4-char suffix are epic
	parts := strings.Split(beadID, "-")
	if len(parts) != 2 {
		return false
	}

	// Common epic patterns: snn1, 6x39, etc.
	// This is a heuristic for MVP
	suffix := parts[1]
	if len(suffix) == 4 && (suffix[0] == 's' || suffix[0] == '6') {
		return true
	}

	return false
}

// GetCurrentSessionID reads or generates a session ID
// Session ID persists for the duration of a delivery loop session
func GetCurrentSessionID(projectRoot string) (string, error) {
	sessionFile := filepath.Join(projectRoot, ".sdp", "traces", "session-id")

	// Try to read existing session ID
	if data, err := os.ReadFile(sessionFile); err == nil {
		sessionID := strings.TrimSpace(string(data))
		if sessionID != "" {
			return sessionID, nil
		}
	}

	// Generate new session ID
	// For MVP, use simple format: sess_<timestamp>
	// In production, would use crypto/rand or uuid
	sessionID := fmt.Sprintf("sess_%d_%d", os.Getpid(), time.Now().UnixNano())

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(sessionFile), 0755); err != nil {
		return "", fmt.Errorf("failed to create traces directory: %w", err)
	}

	// Write session ID
	if err := os.WriteFile(sessionFile, []byte(sessionID+"\n"), 0644); err != nil {
		return "", fmt.Errorf("failed to write session ID: %w", err)
	}

	return sessionID, nil
}

// ClearSessionID removes the current session ID (called on session end)
func ClearSessionID(projectRoot string) error {
	sessionFile := filepath.Join(projectRoot, ".sdp", "traces", "session-id")
	if err := os.Remove(sessionFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear session ID: %w", err)
	}
	return nil
}

// TraceIDGenerator generates trace IDs
type TraceIDGenerator struct {
	pid int
}

// NewTraceIDGenerator creates a new trace ID generator
func NewTraceIDGenerator() *TraceIDGenerator {
	return &TraceIDGenerator{
		pid: os.Getpid(),
	}
}

// Generate generates a new trace ID
// Format: 16-byte hexadecimal (32 hex characters)
func (g *TraceIDGenerator) Generate() string {
	// Use crypto/rand for unique IDs to avoid collisions
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based ID if crypto/rand fails
		return fmt.Sprintf("%016x%016x", os.Getpid(), time.Now().UnixNano())
	}
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7],
		b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15])
}

// SpanIDGenerator generates span IDs
type SpanIDGenerator struct {
	counter uint64
}

// NewSpanIDGenerator creates a new span ID generator
func NewSpanIDGenerator() *SpanIDGenerator {
	return &SpanIDGenerator{
		counter: 1,
	}
}

// Generate generates a new span ID
// Format: 8-byte hexadecimal (16 hex characters)
func (g *SpanIDGenerator) Generate() string {
	id := atomic.AddUint64(&g.counter, 1)
	return fmt.Sprintf("%016x", id)
}
