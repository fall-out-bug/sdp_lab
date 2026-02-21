package retrospective

import (
	"path/filepath"
	"testing"
)

func TestNewAggregator(t *testing.T) {
	a := NewAggregator("/work")
	if a == nil {
		t.Fatal("NewAggregator returned nil")
	}
}

func TestAggregator_CollectPaths(t *testing.T) {
	dir := t.TempDir()
	a := NewAggregator(dir)
	runs, evidence, intakePath := a.CollectPaths("epic1")
	if len(runs) != 0 || len(evidence) != 0 {
		t.Errorf("expected empty: runs=%d evidence=%d", len(runs), len(evidence))
	}
	wantIntake := filepath.Join(dir, ".sdp", "intake.jsonl")
	if intakePath != wantIntake {
		t.Errorf("intakePath = %q, want %q", intakePath, wantIntake)
	}
}
