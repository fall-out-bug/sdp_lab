package orchestrator

import (
	"testing"
)

func TestNewScheduler(t *testing.T) {
	s := NewScheduler("/tmp", nil, 0)
	if s == nil || s.workDir != "/tmp" || s.limit != 10 {
		t.Fatalf("NewScheduler: got %+v", s)
	}
	if len(s.labels) == 0 {
		t.Error("expected default labels")
	}
	s2 := NewScheduler("/tmp", []string{"autonomy"}, 5)
	if s2.limit != 5 || s2.labels[0] != "autonomy" {
		t.Errorf("NewScheduler with args: limit=%d labels=%v", s2.limit, s2.labels)
	}
}

func TestScheduler_TryLock_Unlock(t *testing.T) {
	dir := t.TempDir()
	s := NewScheduler(dir, []string{"autonomy"}, 1)
	// Use unique issue ID to avoid cross-test pollution
	issueID := "test-lock-" + t.Name()

	ok, err := s.TryLock(issueID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected first TryLock to succeed")
	}

	ok2, _ := s.TryLock(issueID)
	if ok2 {
		t.Error("expected duplicate TryLock to fail")
	}

	s.Unlock(issueID)

	ok3, err := s.TryLock(issueID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok3 {
		t.Error("expected TryLock after Unlock to succeed")
	}
	s.Unlock(issueID)
}

func TestRunID(t *testing.T) {
	id := RunID("issue-1")
	if id == "" || len(id) < 20 {
		t.Errorf("RunID returned short value: %q", id)
	}
}

func TestScheduler_Adapter_WorkDir(t *testing.T) {
	dir := t.TempDir()
	s := NewScheduler(dir, []string{"lane:commit"}, 5)
	if s.Adapter() == nil {
		t.Error("Adapter() should not be nil")
	}
	if s.WorkDir() != dir {
		t.Errorf("WorkDir() = %q, want %q", s.WorkDir(), dir)
	}
}

