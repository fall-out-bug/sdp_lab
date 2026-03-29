package dispatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDispatchDecisionJSONRoundTrip_Full(t *testing.T) {
	dec := DispatchDecision{
		Harness:   "claude-code",
		Provider:  "anthropic",
		Model:     "claude-opus-4",
		Score:     0.92,
		Reason:    "best fit for coding tasks",
		Timestamp: "2026-03-28T10:00:00Z",
		Alternatives: []Alternative{
			{Harness: "aider", Score: 0.75, Reason: "good but lower score"},
			{Harness: "cursor", Score: 0.60},
		},
	}

	data, err := json.Marshal(dec)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got DispatchDecision
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Harness != dec.Harness {
		t.Errorf("Harness: got %q, want %q", got.Harness, dec.Harness)
	}
	if got.Provider != dec.Provider {
		t.Errorf("Provider: got %q, want %q", got.Provider, dec.Provider)
	}
	if got.Model != dec.Model {
		t.Errorf("Model: got %q, want %q", got.Model, dec.Model)
	}
	if got.Score != dec.Score {
		t.Errorf("Score: got %v, want %v", got.Score, dec.Score)
	}
	if got.Reason != dec.Reason {
		t.Errorf("Reason: got %q, want %q", got.Reason, dec.Reason)
	}
	if got.Timestamp != dec.Timestamp {
		t.Errorf("Timestamp: got %q, want %q", got.Timestamp, dec.Timestamp)
	}
	if len(got.Alternatives) != len(dec.Alternatives) {
		t.Fatalf("Alternatives len: got %d, want %d", len(got.Alternatives), len(dec.Alternatives))
	}
	if got.Alternatives[0].Harness != dec.Alternatives[0].Harness {
		t.Errorf("Alternatives[0].Harness: got %q, want %q", got.Alternatives[0].Harness, dec.Alternatives[0].Harness)
	}
	if got.Alternatives[0].Score != dec.Alternatives[0].Score {
		t.Errorf("Alternatives[0].Score: got %v, want %v", got.Alternatives[0].Score, dec.Alternatives[0].Score)
	}
}

func TestDispatchDecisionJSONRoundTrip_Minimal(t *testing.T) {
	dec := DispatchDecision{
		Harness:  "aider",
		Provider: "openai",
		Model:    "gpt-4o",
		Score:    0.5,
	}

	data, err := json.Marshal(dec)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got DispatchDecision
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Harness != dec.Harness {
		t.Errorf("Harness: got %q, want %q", got.Harness, dec.Harness)
	}
	if got.Provider != dec.Provider {
		t.Errorf("Provider: got %q, want %q", got.Provider, dec.Provider)
	}
	if got.Model != dec.Model {
		t.Errorf("Model: got %q, want %q", got.Model, dec.Model)
	}
	if got.Score != dec.Score {
		t.Errorf("Score: got %v, want %v", got.Score, dec.Score)
	}
	// omitempty fields should be absent in JSON
	if got.Reason != "" {
		t.Errorf("Reason should be empty, got %q", got.Reason)
	}
	if got.Timestamp != "" {
		t.Errorf("Timestamp should be empty, got %q", got.Timestamp)
	}
	if len(got.Alternatives) != 0 {
		t.Errorf("Alternatives should be empty, got %v", got.Alternatives)
	}
}

func TestDispatchDecisionMinimalOmitEmpty(t *testing.T) {
	dec := DispatchDecision{
		Harness:  "cursor",
		Provider: "anthropic",
		Model:    "claude-sonnet-4",
		Score:    0.8,
	}

	data, err := json.Marshal(dec)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify omitempty fields are not present in raw JSON
	raw := string(data)
	for _, field := range []string{`"reason"`, `"timestamp"`, `"alternatives"`} {
		for i := 0; i < len(raw)-len(field)+1; i++ {
			if raw[i:i+len(field)] == field {
				t.Errorf("omitempty field %s should not appear in minimal JSON, got: %s", field, raw)
				break
			}
		}
	}
}

func TestWriteAndLoadDecision(t *testing.T) {
	dir := t.TempDir()

	dec := &DispatchDecision{
		Harness:   "claude-code",
		Provider:  "anthropic",
		Model:     "claude-opus-4",
		Score:     0.95,
		Reason:    "top performer",
		Timestamp: "2026-03-28T12:00:00Z",
		Alternatives: []Alternative{
			{Harness: "aider", Score: 0.70, Reason: "alternative"},
		},
	}

	if err := WriteDecision(dir, dec); err != nil {
		t.Fatalf("WriteDecision error: %v", err)
	}

	// Verify file exists at expected path
	expectedPath := filepath.Join(dir, ".sdp", "dispatch-decision.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected file at %s, got stat error: %v", expectedPath, err)
	}

	got, err := LoadDecision(dir)
	if err != nil {
		t.Fatalf("LoadDecision error: %v", err)
	}

	if got.Harness != dec.Harness {
		t.Errorf("Harness: got %q, want %q", got.Harness, dec.Harness)
	}
	if got.Provider != dec.Provider {
		t.Errorf("Provider: got %q, want %q", got.Provider, dec.Provider)
	}
	if got.Model != dec.Model {
		t.Errorf("Model: got %q, want %q", got.Model, dec.Model)
	}
	if got.Score != dec.Score {
		t.Errorf("Score: got %v, want %v", got.Score, dec.Score)
	}
	if got.Reason != dec.Reason {
		t.Errorf("Reason: got %q, want %q", got.Reason, dec.Reason)
	}
	if got.Timestamp != dec.Timestamp {
		t.Errorf("Timestamp: got %q, want %q", got.Timestamp, dec.Timestamp)
	}
	if len(got.Alternatives) != len(dec.Alternatives) {
		t.Fatalf("Alternatives len: got %d, want %d", len(got.Alternatives), len(dec.Alternatives))
	}
	if got.Alternatives[0].Harness != dec.Alternatives[0].Harness {
		t.Errorf("Alternatives[0].Harness: got %q, want %q", got.Alternatives[0].Harness, dec.Alternatives[0].Harness)
	}
}

func TestLoadDecision_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadDecision(dir)
	if err == nil {
		t.Error("expected error when file does not exist, got nil")
	}
}

func TestWriteDecision_Idempotent(t *testing.T) {
	dir := t.TempDir()

	first := &DispatchDecision{
		Harness:  "aider",
		Provider: "openai",
		Model:    "gpt-4o",
		Score:    0.6,
	}
	second := &DispatchDecision{
		Harness:  "claude-code",
		Provider: "anthropic",
		Model:    "claude-opus-4",
		Score:    0.9,
	}

	if err := WriteDecision(dir, first); err != nil {
		t.Fatalf("first WriteDecision error: %v", err)
	}
	if err := WriteDecision(dir, second); err != nil {
		t.Fatalf("second WriteDecision error: %v", err)
	}

	got, err := LoadDecision(dir)
	if err != nil {
		t.Fatalf("LoadDecision error: %v", err)
	}

	// Should reflect the second write
	if got.Harness != second.Harness {
		t.Errorf("Harness: got %q, want %q", got.Harness, second.Harness)
	}
	if got.Score != second.Score {
		t.Errorf("Score: got %v, want %v", got.Score, second.Score)
	}
}
