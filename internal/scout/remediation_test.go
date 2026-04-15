package scout

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// ── B1: Context/Timeout ────────────────────────────────────────────────

func TestRunWithContext(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":  "module example.com/app\ngo 1.26\n",
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	ctx := context.Background()
	card, err := RunWithContext(ctx, dir)
	if err != nil {
		t.Fatalf("RunWithContext: %v", err)
	}
	if card == nil {
		t.Fatal("card should not be nil")
	}
}

func TestRunWithContextCancelled(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":  "module example.com/app\ngo 1.26\n",
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := RunWithContext(ctx, dir)
	if err == nil {
		t.Error("expected error with cancelled context")
	}
}

func TestGitCmdRespectsContext(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := gitCmdWithContext(ctx, dir, "log", "--format=%aI|%aN", "--no-merges")
	if result == "" {
		t.Error("expected non-empty git output")
	}
}

// ── B2: Branch Detection Fallback ─────────────────────────────────────

func TestBranchDetectionMasterRepo(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	// Rename branch to master
	cmd := exec.Command("git", "branch", "-M", "master")
	cmd.Dir = dir
	_, _ = cmd.CombinedOutput()

	// Add a remote branch to test
	activity := detectActivity(dir)
	// Should not error — just verify it doesn't crash
	_ = activity.ActiveBranches
}

func TestDetectDefaultBranch(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	branch := detectDefaultBranch(dir)
	if branch == "" {
		t.Error("detectDefaultBranch should return non-empty for git repo")
	}
}

// ── B3: Populated Contract Fields ─────────────────────────────────────

func TestRepoURLPopulated(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":  "module example.com/app\ngo 1.26\n",
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	// Add a remote (local)
	originPath := filepath.Join(dir, "../fake-remote")
	_ = os.MkdirAll(originPath, 0o755)
	cmd := exec.Command("git", "remote", "add", "origin", originPath)
	cmd.Dir = dir
	_, _ = cmd.CombinedOutput()

	ctx := context.Background()
	card, err := RunWithContext(ctx, dir)
	if err != nil {
		t.Fatalf("RunWithContext: %v", err)
	}
	if card.Identity.RepoURL == nil {
		t.Error("RepoURL should be populated when origin remote exists")
	}
}

func TestConfigFilesPopulated(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":         "module example.com/app\ngo 1.26\n",
		"main.go":        "package main\nfunc main() {}\n",
		".goreleaser.yml": "builds:\n  - main: .\n",
		".github/workflows/ci.yml": "name: CI\non: push\n",
	}, false)

	ctx := context.Background()
	card, err := RunWithContext(ctx, dir)
	if err != nil {
		t.Fatalf("RunWithContext: %v", err)
	}
	if len(card.Build.ConfigFiles) == 0 {
		t.Errorf("ConfigFiles should be populated, got %v", card.Build.ConfigFiles)
	}
}

func TestDependencyCountPopulated(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod": "module example.com/app\ngo 1.26\n\nrequire (\n\tgithub.com/foo/bar v1.0.0\n\tgithub.com/baz/qux v2.0.0\n)\n",
		"main.go": "package main\nfunc main() {}\n",
	}, false)

	ctx := context.Background()
	card, err := RunWithContext(ctx, dir)
	if err != nil {
		t.Fatalf("RunWithContext: %v", err)
	}
	if card.Build.DependencyCount < 2 {
		t.Errorf("DependencyCount = %d, want >= 2", card.Build.DependencyCount)
	}
}

func TestRepoURLNilWithoutRemote(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}\n",
	}, true)
	// No remote added

	ctx := context.Background()
	card, err := RunWithContext(ctx, dir)
	if err != nil {
		t.Fatalf("RunWithContext: %v", err)
	}
	if card.Identity.RepoURL != nil {
		t.Error("RepoURL should be nil when no remote exists")
	}
}
