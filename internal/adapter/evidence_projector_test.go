package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/evidence"
)

func TestNewEvidenceProjector(t *testing.T) {
	p := NewEvidenceProjector("/tmp")
	if p == nil || p.workDir != "/tmp" {
		t.Fatalf("NewEvidenceProjector: got %+v", p)
	}
}

func TestEvidenceProjector_ProjectFromIntent(t *testing.T) {
	dir := t.TempDir()
	p := NewEvidenceProjector(dir)

	_, err := p.ProjectFromIntent(nil, nil, "r1")
	if err == nil || err.Error() != "intent required" {
		t.Errorf("ProjectFromIntent(nil): got %v", err)
	}

	intent := &TaskIntent{
		IssueID:   "i1",
		RunID:     "r1",
		Prompt:    "Fix",
		Objective: "AC",
		AgentRef:  "coder",
		SpecHash:  "abc",
	}
	path, err := p.ProjectFromIntent(intent, map[string]string{"coder": "output"}, "r1")
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(dir, ".sdp", "evidence", "i1.json")
	if path != expected {
		t.Errorf("got path %q, want %q", path, expected)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	res, err := evidence.ValidateStrictFile(path, false)
	if err != nil {
		t.Fatalf("ValidateStrictFile: %v", err)
	}
	if !res.OK {
		t.Errorf("projected evidence invalid: %s", res.Reason)
	}
}
