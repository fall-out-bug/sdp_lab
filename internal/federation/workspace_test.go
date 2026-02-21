package federation

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWorkspaceManager(t *testing.T) {
	w := NewWorkspaceManager("/tmp/workspaces")
	if w == nil {
		t.Fatal("NewWorkspaceManager returned nil")
	}
}

func TestWorkspaceManager_WorkspacePath(t *testing.T) {
	w := NewWorkspaceManager("/base")
	got := w.WorkspacePath("proj1")
	want := filepath.Join("/base", "proj1")
	if got != want {
		t.Errorf("WorkspacePath = %q, want %q", got, want)
	}
}

func TestWorkspaceManager_EnsureWorkspace_local(t *testing.T) {
	dir := t.TempDir()
	w := NewWorkspaceManager(dir)
	path, err := w.EnsureWorkspace("local-proj", ".", "main")
	if err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	// With "." repoURL, it should create dir and return abs path
	if !strings.Contains(path, "local-proj") {
		t.Errorf("path %q should contain local-proj", path)
	}
}
