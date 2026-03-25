package control

import (
	"testing"
)

func TestDeployPhaseConstants(t *testing.T) {
	if DeployPhaseStaging != "staging" {
		t.Error("wrong staging constant")
	}
	if DeployPhaseProd != "prod" {
		t.Error("wrong prod constant")
	}
}

func TestTrimOutput(t *testing.T) {
	if trimOutput([]byte("abc")) != "abc" {
		t.Error("short output")
	}
	// Long JSON output should be returned as-is
	long := string(make([]byte, 60))
	if trimOutput([]byte(long)) != long {
		t.Error("long output should be preserved")
	}
}

func TestDeployPhaseTransition_NilBeads(t *testing.T) {
	store := &Store{RepoMode: RepoModeFile}
	_, err := store.DeployPhaseTransition("test", DeployPhaseStaging)
	if err == nil {
		t.Error("should fail without beads mode")
	}
}

func TestDeployGate_NilBeads(t *testing.T) {
	store := &Store{RepoMode: RepoModeFile}
	_, _, err := store.DeployGate("test", DeployPhaseStaging)
	if err == nil {
		t.Error("should fail without beads mode")
	}
}

func TestRecordDeployEvidence_NilBeads(t *testing.T) {
	store := &Store{RepoMode: RepoModeFile}
	err := store.RecordDeployEvidence("test", DeployPhaseStaging, map[string]any{"ok": true})
	if err == nil {
		t.Error("should fail without beads mode")
	}
}
