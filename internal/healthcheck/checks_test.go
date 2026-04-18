package healthcheck_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"sdp_dev/internal/healthcheck"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Skipf("git setup failed (%v): %s", err, out)
		}
	}
}

func TestGitCleanChecker_CleanRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: requires git in PATH")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)

	runner, _ := healthcheck.NewRunner(healthcheck.Config{ProjectRoot: dir, Only: "git-clean"})
	results := runner.Run(context.Background())
	if results[0].Status != healthcheck.StatusPass {
		t.Errorf("expected pass on clean repo, got %s: %s", results[0].Status, results[0].Detail)
	}
}

func TestGitCleanChecker_DirtyRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: requires git in PATH")
	}
	dir := t.TempDir()
	initGitRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	runner, _ := healthcheck.NewRunner(healthcheck.Config{ProjectRoot: dir, Only: "git-clean"})
	results := runner.Run(context.Background())
	if results[0].Status != healthcheck.StatusWarn {
		t.Errorf("expected warn on dirty repo, got %s: %s", results[0].Status, results[0].Detail)
	}
}

func TestGitCleanChecker_NotARepo(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: requires git in PATH")
	}
	runner, _ := healthcheck.NewRunner(healthcheck.Config{ProjectRoot: t.TempDir(), Only: "git-clean"})
	results := runner.Run(context.Background())
	if results[0].Status != healthcheck.StatusFail {
		t.Errorf("expected fail outside git repo, got %s", results[0].Status)
	}
}

func TestBeadsReadyChecker_MissingBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: modifies PATH")
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	runner, _ := healthcheck.NewRunner(healthcheck.Config{ProjectRoot: t.TempDir(), Only: "beads-ready"})
	results := runner.Run(context.Background())
	if results[0].Status != healthcheck.StatusFail {
		t.Errorf("expected fail when bd not in PATH, got %s: %s", results[0].Status, results[0].Detail)
	}
}
