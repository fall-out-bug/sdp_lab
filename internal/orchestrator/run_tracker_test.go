package orchestrator

import (
	"os"
	"strings"
	"testing"
)

func TestNewRunTracker(t *testing.T) {
	rt := NewRunTracker("/tmp")
	if rt == nil || rt.runsDir != "/tmp/.sdp/runs" {
		t.Fatalf("NewRunTracker: got %+v", rt)
	}
}

func TestRunTracker_Create_AppendPhase(t *testing.T) {
	dir := t.TempDir()
	rt := NewRunTracker(dir)

	path, err := rt.Create("run-1", "issue-1", "orchestrator", "host1", 30, 300)
	if err != nil {
		t.Fatal(err)
	}
	if path == "" || !strings.HasPrefix(path, dir) {
		t.Errorf("Create returned path %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	err = rt.AppendPhase("run-1", "dispatch", "ok", "dispatched", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunTracker_RunFilePath_RunDir(t *testing.T) {
	rt := NewRunTracker("/tmp")
	if rt.RunFilePath("r1") != "/tmp/.sdp/runs/r1.json" {
		t.Errorf("RunFilePath: %s", rt.RunFilePath("r1"))
	}
	if rt.RunDir() != "/tmp/.sdp/runs" {
		t.Errorf("RunDir: %s", rt.RunDir())
	}
}

func TestRunTracker_EnsureRunsDir(t *testing.T) {
	dir := t.TempDir()
	rt := NewRunTracker(dir)
	if err := rt.EnsureRunsDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rt.runsDir); err != nil {
		t.Fatal(err)
	}
}

func TestRunTrackerFromWorkDir(t *testing.T) {
	rt, err := RunTrackerFromWorkDir("/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if rt == nil {
		t.Fatal("expected non-nil RunTracker")
	}
}
