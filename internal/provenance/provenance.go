// Package provenance formats and parses git trailer provenance markers
// for AI-vs-human attribution in the SDP pipeline.
package provenance

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Provenance represents AI-vs-human attribution metadata.
type Provenance struct {
	GeneratedBy string    // agent|human|hybrid
	Harness     string    // claude-code|codex|opencode|cursor
	Model       string    // claude-opus-4-7, gpt-4, etc.
	SessionID   string
	Timestamp   time.Time
}

// FormatTrailer formats the provenance as a git trailer line.
// Output format: "AI-Attribution: agent/claude-opus-4-7/session-abc123"
func (p Provenance) FormatTrailer() string {
	var parts []string

	if p.GeneratedBy != "" {
		parts = append(parts, p.GeneratedBy)
	}

	if p.Model != "" {
		parts = append(parts, p.Model)
	}

	if p.SessionID != "" {
		parts = append(parts, p.SessionID)
	}

	value := strings.Join(parts, "/")

	if p.Harness != "" {
		return fmt.Sprintf("AI-Attribution: %s (%s)", value, p.Harness)
	}
	return fmt.Sprintf("AI-Attribution: %s", value)
}

// ParseTrailer parses a git trailer line back into a Provenance struct.
// Expected format: "AI-Attribution: agent/claude-opus-4-7/session-abc123 (claude-code)"
func ParseTrailer(line string) (Provenance, error) {
	p := Provenance{
		Timestamp: time.Now(),
	}

	// Remove prefix if present
	const prefix = "AI-Attribution: "
	if !strings.HasPrefix(line, prefix) {
		return p, fmt.Errorf("invalid provenance trailer: missing AI-Attribution prefix")
	}

	line = strings.TrimPrefix(line, prefix)
	line = strings.TrimSpace(line)

	// Extract harness from suffix if present (e.g. " (claude-code)")
	if strings.Contains(line, " (") && strings.HasSuffix(line, ")") {
		openParen := strings.LastIndex(line, " (")
		closeParen := strings.LastIndex(line, ")")
		if openParen != -1 && closeParen == len(line)-1 {
			p.Harness = line[openParen+2 : closeParen]
			line = line[:openParen]
		}
	}

	// Parse slash-delimited parts: generated_by/model/session_id
	parts := strings.Split(line, "/")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		switch i {
		case 0:
			p.GeneratedBy = part
		case 1:
			p.Model = part
		case 2:
			p.SessionID = part
		}
	}

	return p, nil
}

// FromEnv creates a Provenance struct from environment variables.
// Reads from SDP_HARNESS, SDP_MODEL, SDP_SESSION_ID env vars.
func FromEnv() Provenance {
	p := Provenance{
		Timestamp:   time.Now(),
		GeneratedBy: "agent",
		Harness:     os.Getenv("SDP_HARNESS"),
		Model:       os.Getenv("SDP_MODEL"),
		SessionID:   os.Getenv("SDP_SESSION_ID"),
	}

	// If no harness/model/session info, assume human
	if p.Harness == "" && p.Model == "" && p.SessionID == "" {
		p.GeneratedBy = "human"
	}

	return p
}
