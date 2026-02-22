package federation

import (
	"os/exec"
	"path/filepath"
	"testing"

	"sdp_dev/internal/registry"
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

func TestWorkspaceManager_EnsureWorkspaceFromProject_noFork(t *testing.T) {
	dir := t.TempDir()
	w := NewWorkspaceManager(dir)
	proj := &registry.Project{
		ID:         "p3",
		RepoURL:    ".",
		RepoBranch: "main",
	}
	path, err := w.EnsureWorkspaceFromProject(proj)
	if err != nil {
		t.Fatalf("EnsureWorkspaceFromProject: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	abs, _ := filepath.Abs(filepath.Join(dir, "p3"))
	if path != abs {
		t.Errorf("path = %s, want %s", path, abs)
	}
}

func TestWorkspaceManager_EnsureWorkspaceFromProject_forkWithLocalRepo(t *testing.T) {
	// Create a local git repo to clone from (and use as upstream for fetch)
	src := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"checkout", "-b", "main"},
		{"commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = src
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v failed: %v: %s", args, err, string(out))
		}
	}

	dir := t.TempDir()
	w := NewWorkspaceManager(dir)
	proj := &registry.Project{
		ID:             "fork1",
		RepoURL:        src,
		RepoBranch:     "main",
		Fork:           true,
		UpstreamRemote: "upstream",
		UpstreamURL:    src, // same repo as origin for test (fetch works locally)
	}
	path, err := w.EnsureWorkspaceFromProject(proj)
	if err != nil {
		t.Fatalf("EnsureWorkspaceFromProject: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	// Verify workspace is isolated per project
	got := w.WorkspacePath("fork1")
	want := filepath.Join(dir, "fork1")
	if got != want {
		t.Errorf("WorkspacePath = %s, want %s", got, want)
	}
}
