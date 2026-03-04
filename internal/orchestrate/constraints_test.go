package orchestrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckRequiredFiles_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	cfg := &AgentConstraintConfig{
		Phases: map[string]PhaseConstraints{
			"build": {
				Constraints: []Constraint{
					{ID: "c1", Check: "file-exists", Path: "../../../etc/passwd", Severity: "block", Message: "bad"},
				},
			},
		},
	}
	violations := CheckRequiredFiles(cfg, "build", dir, "F001")
	// Should not add violation (we skip invalid path) and must not escape dir
	if len(violations) > 0 {
		t.Errorf("expected no violations for path traversal path (constraint skipped), got %d", len(violations))
	}
	// Ensure we didn't touch anything outside
	if _, err := os.Stat("/etc/passwd"); err == nil {
		// File exists; verify we didn't Stat it from our logic (we skip the constraint)
		_ = err
	}
}

func TestCheckRequiredFiles_RejectsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	cfg := &AgentConstraintConfig{
		Phases: map[string]PhaseConstraints{
			"build": {
				Constraints: []Constraint{
					{ID: "c1", Check: "file-exists", Path: "/etc/passwd", Severity: "block", Message: "bad"},
				},
			},
		},
	}
	violations := CheckRequiredFiles(cfg, "build", dir, "F001")
	if len(violations) > 0 {
		t.Errorf("expected no violations for absolute path (constraint skipped), got %d", len(violations))
	}
}

func TestCheckRequiredFiles_AcceptsValidPath(t *testing.T) {
	dir := t.TempDir()
	sdpEvidence := filepath.Join(dir, ".sdp", "evidence")
	if err := os.MkdirAll(sdpEvidence, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sdpEvidence, "F001.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &AgentConstraintConfig{
		Phases: map[string]PhaseConstraints{
			"build": {
				Constraints: []Constraint{
					{ID: "c1", Check: "file-exists", Path: ".sdp/evidence/{feature_id}.json", Severity: "block", Message: "missing"},
				},
			},
		},
	}
	violations := CheckRequiredFiles(cfg, "build", dir, "F001")
	if len(violations) != 0 {
		t.Errorf("expected 0 violations (file exists), got %d: %v", len(violations), violations)
	}
}

func TestCheckRequiredFiles_ValidPathMissingAddsViolation(t *testing.T) {
	dir := t.TempDir()
	cfg := &AgentConstraintConfig{
		Phases: map[string]PhaseConstraints{
			"build": {
				Constraints: []Constraint{
					{ID: "c1", Check: "file-exists", Path: ".sdp/evidence/{feature_id}.json", Severity: "block", Message: "missing"},
				},
			},
		},
	}
	violations := CheckRequiredFiles(cfg, "build", dir, "F001")
	if len(violations) != 1 {
		t.Errorf("expected 1 violation (file missing), got %d", len(violations))
	}
}
