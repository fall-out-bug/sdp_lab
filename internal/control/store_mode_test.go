package control

import (
	"os"
	"path/filepath"
	"testing"
)

func setupProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Create minimal project structure required by OpenWithMode
	dirs := []string{
		"docs/specs",
		".sdp/control/projects",
	}
	for _, d := range dirs {
		_ = os.MkdirAll(filepath.Join(dir, d), 0o755)
	}
	// Create project registry
	os.WriteFile(filepath.Join(dir, "docs/specs/project-registry.yaml"), []byte("projects: []"), 0o644)
	return dir
}

func TestOpenWithMode_FileMode(t *testing.T) {
	dir := setupProjectDir(t)
	store, err := OpenWithMode(dir, RepoModeFile, "")
	if err != nil {
		t.Fatalf("OpenWithMode file: %v", err)
	}
	if store.RepoMode != RepoModeFile {
		t.Errorf("mode: got %s, want %s", store.RepoMode, RepoModeFile)
	}
	if store.BeadsRepo() != nil {
		t.Error("expected nil beads repo in file mode")
	}
	if store.DualRepo() != nil {
		t.Error("expected nil dual repo in file mode")
	}
}

func TestOpenWithMode_BeadsMode(t *testing.T) {
	dir := setupProjectDir(t)
	bdDir := filepath.Join(dir, ".beads-worktrees", "main")
	_ = os.MkdirAll(bdDir, 0o755)

	store, err := OpenWithMode(dir, RepoModeBeads, "")
	if err != nil {
		t.Fatalf("OpenWithMode beads: %v", err)
	}
	if store.RepoMode != RepoModeBeads {
		t.Errorf("mode: got %s, want %s", store.RepoMode, RepoModeBeads)
	}
	if store.BeadsRepo() == nil {
		t.Error("expected non-nil beads repo in beads mode")
	}
}

func TestOpenWithMode_DualMode(t *testing.T) {
	dir := setupProjectDir(t)
	bdDir := filepath.Join(dir, ".beads-worktrees", "main")
	_ = os.MkdirAll(bdDir, 0o755)

	store, err := OpenWithMode(dir, RepoModeDual, "")
	if err != nil {
		t.Fatalf("OpenWithMode dual: %v", err)
	}
	if store.RepoMode != RepoModeDual {
		t.Errorf("mode: got %s, want %s", store.RepoMode, RepoModeDual)
	}
	if store.BeadsRepo() == nil {
		t.Error("expected non-nil beads repo in dual mode")
	}
	if store.DualRepo() == nil {
		t.Error("expected non-nil dual repo in dual mode")
	}
}

func TestOpen_BackwardsCompatible(t *testing.T) {
	dir := setupProjectDir(t)
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if store.RepoMode != RepoModeFile {
		t.Errorf("Open() should default to file mode, got %s", store.RepoMode)
	}
}
