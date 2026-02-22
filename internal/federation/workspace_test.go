package federation

import (
	"path/filepath"
	"testing"
)

func TestNewWorkspaceManager(t *testing.T) {
	dir := t.TempDir()
	w := NewWorkspaceManager(dir)
	if w == nil {
		t.Fatal("NewWorkspaceManager returned nil")
	}
}

func TestWorkspaceManager_EnsureWorkspace_local(t *testing.T) {
	dir := t.TempDir()
	w := NewWorkspaceManager(dir)
	path, err := w.EnsureWorkspace("p1", ".", "")
	if err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	abs, _ := filepath.Abs(filepath.Join(dir, "p1"))
	if path != abs {
		t.Errorf("path = %s, want %s", path, abs)
	}
}

func TestWorkspaceManager_EnsureWorkspace_emptyRepo(t *testing.T) {
	dir := t.TempDir()
	w := NewWorkspaceManager(dir)
	path, err := w.EnsureWorkspace("p2", "", "main")
	if err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestWorkspaceManager_WorkspacePath(t *testing.T) {
	dir := t.TempDir()
	w := NewWorkspaceManager(dir)
	got := w.WorkspacePath("proj")
	want := filepath.Join(dir, "proj")
	if got != want {
		t.Errorf("WorkspacePath = %s, want %s", got, want)
	}
}
