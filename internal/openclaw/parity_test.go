package openclaw

import (
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/runtime"
)

func TestAdapter_BranchNamingParity(t *testing.T) {
	dir := findRepoRoot(t)
	adapter := NewAdapter(dir)
	branch, err := adapter.CreateBranch("sdp_dev-xyz", "my-task")
	if err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	// OpenCode uses feat/<issue-id>-<slug>
	if branch != "feat/sdp_dev-xyz-my-task" {
		t.Errorf("expected feat/sdp_dev-xyz-my-task, got %s", branch)
	}
}

func TestAdapter_ImplementsContract(t *testing.T) {
	var _ runtime.AutonomousRuntimeModule = (*Adapter)(nil)
}

func TestAdapter_LoadTaskRequiresValidIssue(t *testing.T) {
	dir := findRepoRoot(t)
	adapter := NewAdapter(dir)
	_, err := adapter.LoadTask("nonexistent-issue-12345")
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func findRepoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".beads")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("no .beads found")
			return ""
		}
		dir = parent
	}
}
