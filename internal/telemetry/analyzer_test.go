package telemetry

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeEvidence(t *testing.T) {
	a := NewAnalyzer("/tmp", "glm-4.7", 5)

	ev := map[string]any{
		"intent": map[string]any{
			"objective": "Add feature X",
		},
		"execution": map[string]any{
			"changed_files": []any{"pkg/foo.go", "pkg/bar.go"},
		},
	}
	got := a.summarizeEvidence(ev)
	if got == "" {
		t.Error("expected non-empty summary")
	}
	if !strings.Contains(got, "Add feature X") {
		t.Errorf("summary missing objective: %q", got)
	}
	if !strings.Contains(got, "pkg/foo.go") {
		t.Errorf("summary missing changed file: %q", got)
	}
}

func TestSummarizeEvidence_Empty(t *testing.T) {
	a := NewAnalyzer("/tmp", "glm-4.7", 5)
	got := a.summarizeEvidence(map[string]any{})
	if got != "" {
		t.Errorf("expected empty summary, got %q", got)
	}
}

func TestWorkDirForProject(t *testing.T) {
	a := NewAnalyzer("/workspaces", "glm-4.7", 5)
	if got := a.workDirForProject(""); got != "/workspaces/default" {
		t.Errorf("empty project: got %q", got)
	}
	if got := a.workDirForProject("sdp_dev"); got != "/workspaces/sdp_dev" {
		t.Errorf("sdp_dev: got %q", got)
	}
}

func TestHandleClosed_NoEvidence(t *testing.T) {
	dir := t.TempDir()
	a := NewAnalyzer(dir, "glm-4.7", 5)
	created, err := a.HandleClosed(context.Background(), "nonexistent", "default")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("expected not created when evidence missing")
	}
}

func TestIsDuplicate(t *testing.T) {
	dir := t.TempDir()
	// Create a minimal beads repo so bd list works
	bdDir := filepath.Join(dir, "default")
	if err := os.MkdirAll(bdDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bdDir, ".git"), []byte("gitdir: ../.git"), 0644); err != nil {
		t.Skip("bd list requires git repo; skipping")
	}
	a := NewAnalyzer(dir, "glm-4.7", 5)
	// Without bd, isDuplicate returns false (cmd fails)
	dup := a.isDuplicate("Some proposal title", bdDir)
	if dup {
		t.Error("expected false when bd not available")
	}
}
