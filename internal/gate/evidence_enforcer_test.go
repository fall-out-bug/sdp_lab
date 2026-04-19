package gate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGate_RequireEvidence(t *testing.T) {
	tests := []struct {
		name string
		gate Gate
		want bool
	}{
		{
			name: "manual gate does not require evidence",
			gate: Gate{
				ID:   "g1",
				Type: GateTypeManual,
			},
			want: false,
		},
		{
			name: "plan gate requires evidence",
			gate: Gate{
				ID:   "g2",
				Type: GateTypePlan,
			},
			want: true,
		},
		{
			name: "review gate requires evidence",
			gate: Gate{
				ID:   "g3",
				Type: GateTypeReview,
			},
			want: true,
		},
		{
			name: "eval gate requires evidence",
			gate: Gate{
				ID:   "g4",
				Type: GateTypeEval,
			},
			want: true,
		},
		{
			name: "empty type defaults to no evidence required",
			gate: Gate{
				ID:   "g5",
				Type: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gate.RequireEvidence(); got != tt.want {
				t.Errorf("Gate.RequireEvidence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGate_ResolveWithEvidence_ManualGate_NoEvidence(t *testing.T) {
	g := &Gate{
		ID:        "manual-1",
		Question:  "Proceed?",
		Type:      GateTypeManual,
		CreatedAt: time.Now(),
	}

	// Manual gate should resolve without evidence
	err := g.ResolveWithEvidence("yes", "alice", "")
	if err != nil {
		t.Fatalf("ResolveWithEvidence failed for manual gate: %v", err)
	}

	if g.Answer != "yes" {
		t.Errorf("Answer = %q, want yes", g.Answer)
	}
	if g.Answerer != "alice" {
		t.Errorf("Answerer = %q, want alice", g.Answerer)
	}
	if g.ResolvedAt == nil {
		t.Error("ResolvedAt should be set")
	}
	if g.Status() != "resolved" {
		t.Errorf("Status = %q, want resolved", g.Status())
	}
}

func TestGate_ResolveWithEvidence_PhaseGate_NoEvidence(t *testing.T) {
	tests := []struct {
		name     string
		gateType GateType
	}{
		{"plan gate", GateTypePlan},
		{"review gate", GateTypeReview},
		{"eval gate", GateTypeEval},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Gate{
				ID:        "phase-1",
				Question:  "Approve phase?",
				Type:      tt.gateType,
				CreatedAt: time.Now(),
			}

			// Phase gate should fail without evidence
			err := g.ResolveWithEvidence("yes", "alice", "")
			if err == nil {
				t.Fatal("ResolveWithEvidence should fail for phase gate without evidence")
			}

			var reqErr *RequireEvidenceError
			if !errors.As(err, &reqErr) {
				t.Errorf("Error type = %T, want RequireEvidenceError", err)
			}

			// Gate should remain unresolved
			if g.Status() != "pending" {
				t.Errorf("Status = %q, want pending (should not be resolved)", g.Status())
			}
		})
	}
}

func TestGate_ResolveWithEvidence_PhaseGate_EvidenceNotFound(t *testing.T) {
	g := &Gate{
		ID:        "phase-2",
		Question:  "Approve phase?",
		Type:      GateTypePlan,
		CreatedAt: time.Now(),
	}

	nonExistentPath := "/tmp/nonexistent_evidence_12345.json"
	err := g.ResolveWithEvidence("yes", "alice", nonExistentPath)
	if err == nil {
		t.Fatal("ResolveWithEvidence should fail when evidence file does not exist")
	}

	var notFoundErr *EvidenceNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Error type = %T, want EvidenceNotFoundError", err)
	}

	// Gate should remain unresolved
	if g.Status() != "pending" {
		t.Errorf("Status = %q, want pending", g.Status())
	}
}

func TestGate_ResolveWithEvidence_PhaseGate_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	invalidJSONPath := filepath.Join(tmp, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(invalidJSONPath, []byte("{ not valid json }"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	g := &Gate{
		ID:        "phase-3",
		Question:  "Approve phase?",
		Type:      GateTypeReview,
		CreatedAt: time.Now(),
	}

	err := g.ResolveWithEvidence("yes", "alice", invalidJSONPath)
	if err == nil {
		t.Fatal("ResolveWithEvidence should fail for invalid JSON")
	}

	var invalidErr *InvalidEvidenceError
	if !errors.As(err, &invalidErr) {
		t.Errorf("Error type = %T, want InvalidEvidenceError", err)
	}

	// Gate should remain unresolved
	if g.Status() != "pending" {
		t.Errorf("Status = %q, want pending", g.Status())
	}
}

func TestGate_ResolveWithEvidence_PhaseGate_ValidEvidence(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name     string
		gateType GateType
		content  interface{}
	}{
		{
			name:     "plan gate with valid JSON",
			gateType: GateTypePlan,
			content:  map[string]interface{}{"phase": "plan", "approved": true},
		},
		{
			name:     "review gate with valid JSON",
			gateType: GateTypeReview,
			content:  map[string]interface{}{"review": "pass", "notes": "looks good"},
		},
		{
			name:     "eval gate with valid JSON",
			gateType: GateTypeEval,
			content:  map[string]interface{}{"score": 95, "metrics": []int{1, 2, 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create valid JSON evidence file
			evidencePath := filepath.Join(tmp, tt.name+".json")
			data, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("Failed to marshal JSON: %v", err)
			}
			if err := os.WriteFile(evidencePath, data, 0644); err != nil {
				t.Fatalf("Failed to write evidence file: %v", err)
			}

			g := &Gate{
				ID:        "phase-" + tt.name,
				Question:  "Approve phase?",
				Type:      tt.gateType,
				CreatedAt: time.Now(),
			}

			// Should succeed with valid evidence
			err = g.ResolveWithEvidence("approve", "bob", evidencePath)
			if err != nil {
				t.Fatalf("ResolveWithEvidence failed: %v", err)
			}

			// Verify resolution
			if g.Answer != "approve" {
				t.Errorf("Answer = %q, want approve", g.Answer)
			}
			if g.Answerer != "bob" {
				t.Errorf("Answerer = %q, want bob", g.Answerer)
			}
			if g.Status() != "resolved" {
				t.Errorf("Status = %q, want resolved", g.Status())
			}
			if g.EvidencePath != evidencePath {
				t.Errorf("EvidencePath = %q, want %q", g.EvidencePath, evidencePath)
			}
		})
	}
}

func TestGate_ResolveWithEvidence_ManualGate_WithEvidence(t *testing.T) {
	tmp := t.TempDir()
	evidencePath := filepath.Join(tmp, "evidence.json")

	// Create a valid evidence file (even though manual gates don't require it)
	data := []byte(`{"optional": true}`)
	if err := os.WriteFile(evidencePath, data, 0644); err != nil {
		t.Fatalf("Failed to write evidence file: %v", err)
	}

	g := &Gate{
		ID:        "manual-2",
		Question:  "Proceed?",
		Type:      GateTypeManual,
		CreatedAt: time.Now(),
	}

	// Manual gate can optionally provide evidence
	err := g.ResolveWithEvidence("yes", "charlie", evidencePath)
	if err != nil {
		t.Fatalf("ResolveWithEvidence failed: %v", err)
	}

	// Verify resolution
	if g.Answer != "yes" {
		t.Errorf("Answer = %q, want yes", g.Answer)
	}
	if g.EvidencePath != evidencePath {
		t.Errorf("EvidencePath = %q, want %q", g.EvidencePath, evidencePath)
	}
}

func TestRequireEvidenceError(t *testing.T) {
	err := &RequireEvidenceError{GateType: GateTypePlan}
	expected := "phase gate of type plan requires evidence"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestEvidenceNotFoundError(t *testing.T) {
	err := &EvidenceNotFoundError{Path: "/tmp/missing.json"}
	expected := "evidence file not found: /tmp/missing.json"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestInvalidEvidenceError(t *testing.T) {
	innerErr := errors.New("unexpected token")
	err := &InvalidEvidenceError{Path: "/tmp/bad.json", Err: innerErr}
	expected := "evidence file is not valid JSON: /tmp/bad.json: unexpected token"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}

	// Test Unwrap
	if unwrapped := err.Unwrap(); unwrapped != innerErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, innerErr)
	}
}

func TestGate_ResolveWithEvidence_BackwardCompatible(t *testing.T) {
	// Test that gates with no Type field (empty/default) work like manual gates
	g := &Gate{
		ID:        "old-gate",
		Question:  "Proceed?",
		Type:      "", // Empty type, should behave like manual
		CreatedAt: time.Now(),
	}

	// Should resolve without evidence
	err := g.ResolveWithEvidence("yes", "dave", "")
	if err != nil {
		t.Fatalf("ResolveWithEvidence failed for backward compatible gate: %v", err)
	}

	if g.Status() != "resolved" {
		t.Errorf("Status = %q, want resolved", g.Status())
	}
}
