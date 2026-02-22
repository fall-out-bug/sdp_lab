package adapter

import (
	"path/filepath"
	"testing"
)

func TestNewRunLockManager(t *testing.T) {
	m := NewRunLockManager("")
	if m == nil || m.lockDir == "" {
		t.Fatal("NewRunLockManager returned nil or empty lockDir")
	}
	m2 := NewRunLockManager("/tmp/custom")
	if m2.lockDir != "/tmp/custom" {
		t.Errorf("expected lockDir /tmp/custom, got %q", m2.lockDir)
	}
}

func TestRunLockManager_TryAcquire_Release(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "locks")
	m := NewRunLockManager(dir)

	runID, ok, err := m.TryAcquire("issue-1", "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || runID != "run-1" {
		t.Errorf("expected acquired run-1, got ok=%v runID=%q", ok, runID)
	}
	if !m.IsLocked("issue-1") {
		t.Error("expected issue-1 locked")
	}

	// Duplicate acquire fails
	_, ok2, _ := m.TryAcquire("issue-1", "run-2")
	if ok2 {
		t.Error("expected duplicate acquire to fail")
	}

	if err := m.Release("issue-1"); err != nil {
		t.Fatal(err)
	}
	if m.IsLocked("issue-1") {
		t.Error("expected issue-1 unlocked after release")
	}
}

func TestRunLockManager_TryAcquire_InvalidIssueID(t *testing.T) {
	m := NewRunLockManager(t.TempDir())
	_, ok, err := m.TryAcquire("../../../etc/passwd", "run-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected invalid issueID to be rejected")
	}
}

func TestRunLockManager_Release_InvalidIssueID(t *testing.T) {
	m := NewRunLockManager(t.TempDir())
	err := m.Release("../../../etc/passwd")
	if err != nil {
		t.Fatalf("Release with invalid ID should return nil, got %v", err)
	}
}
