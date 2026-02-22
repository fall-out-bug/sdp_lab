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

func TestAdapter_ExecuteTask_NilPlan(t *testing.T) {
	a := NewAdapter(t.TempDir())
	_, err := a.ExecuteTask(nil)
	if err == nil || err.Error() != "plan required" {
		t.Errorf("ExecuteTask(nil): got %v", err)
	}
}

func TestAdapter_ExecuteTask_ValidPlan(t *testing.T) {
	dir := t.TempDir()
	a := NewAdapter(dir)
	ctx, err := a.ExecuteTask(&runtime.Plan{IssueID: "i1", Model: "glm-5"})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.IssueID != "i1" || ctx.Branch != "feat/i1-openclaw" || ctx.RunID != "i1-openclaw" {
		t.Errorf("got %+v", ctx)
	}
	if ctx.EvidencePath != filepath.Join(dir, ".sdp", "evidence", "i1.json") {
		t.Errorf("EvidencePath: %s", ctx.EvidencePath)
	}
}

func TestAdapter_CreateBranch_EmptySlug(t *testing.T) {
	a := NewAdapter(t.TempDir())
	branch, err := a.CreateBranch("i1", "")
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feat/i1-task" {
		t.Errorf("expected feat/i1-task, got %s", branch)
	}
}

func TestAdapter_RunVerification_NilContext(t *testing.T) {
	a := NewAdapter(t.TempDir())
	ok, err := a.RunVerification(nil)
	if err == nil || ok {
		t.Errorf("RunVerification(nil): got ok=%v err=%v", ok, err)
	}
}

func TestAdapter_BuildEvidence_NilContext(t *testing.T) {
	a := NewAdapter(t.TempDir())
	_, err := a.BuildEvidence(nil)
	if err == nil || err.Error() != "context required" {
		t.Errorf("BuildEvidence(nil): got %v", err)
	}
}

func TestAdapter_PublishPR_NilContext(t *testing.T) {
	a := NewAdapter(t.TempDir())
	_, err := a.PublishPR(nil)
	if err == nil || err.Error() != "context required" {
		t.Errorf("PublishPR(nil): got %v", err)
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
