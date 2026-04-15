package scout

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ── Shared test helpers ─────────────────────────────────────────────────

func createTempRepo(t *testing.T, files map[string]string, initGit bool) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if initGit {
		runGit(t, dir, "init")
		runGit(t, dir, "config", "user.email", "test@test.com")
		runGit(t, dir, "config", "user.name", "Tester")
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "initial")
	}
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2026-04-01T12:00:00Z", "GIT_COMMITTER_DATE=2026-04-01T12:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %s: %v", strings.Join(args, " "), out, err)
	}
}

func addAndCommit(t *testing.T, dir, file, content, date string) {
	t.Helper()
	full := filepath.Join(dir, file)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_AUTHOR_DATE=") && !strings.HasPrefix(e, "GIT_COMMITTER_DATE=") {
			env = append(env, e)
		}
	}
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Env = env
	_, _ = cmd.CombinedOutput()
	cmd = exec.Command("git", "commit", "-m", "update "+file)
	cmd.Dir = dir
	cmd.Env = env
	_, _ = cmd.CombinedOutput()
}

func sortedKeys(m map[string]LangStats) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func strPtr(s string) *string { return &s }

// ── Identity Tests ──────────────────────────────────────────────────────

func TestIdentityDetectsGoProject(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":    "module example.com/app\ngo 1.26\n",
		"main.go":   "package main\nfunc main() {}\n",
		"README.md": "# My App\nA cool application for testing.\nMore text here.\n",
	}, false)

	identity, maturity, build := detectIdentity(dir)

	if identity.PrimaryLanguage != "go" {
		t.Errorf("PrimaryLanguage = %q, want go", identity.PrimaryLanguage)
	}
	if identity.BuildSystem == nil || *identity.BuildSystem != "go-modules" {
		t.Error("BuildSystem should be go-modules")
	}
	if identity.Description == nil || !strings.Contains(*identity.Description, "cool application") {
		t.Errorf("Description = %v, want README first paragraph", identity.Description)
	}
	if !maturity.HasReadme {
		t.Error("HasReadme should be true")
	}
	if build.PackageManager == nil || *build.PackageManager != "go-modules" {
		t.Error("PackageManager should be go-modules")
	}
}

func TestIdentityDetectsMixedLanguage(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":  "module example.com/app\ngo 1.26\n",
		"main.go": "package main\nfunc main() {}\n",
		"util.go": "package main\nfunc util() {}\n",
		"api.ts":  "export function handler() {}\n",
		"api.py":  "def handler(): pass\n",
	}, false)

	identity, _, _ := detectIdentity(dir)

	if identity.PrimaryLanguage != "go" {
		t.Errorf("PrimaryLanguage = %q, want go (highest count)", identity.PrimaryLanguage)
	}
	langs := sortedKeys(identity.Languages)
	if len(langs) < 2 {
		t.Errorf("expected multiple languages, got %v", langs)
	}
}

func TestIdentityDetectsMonorepo(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"package.json":            "{}",
		"packages/a/package.json": "{}",
		"packages/b/package.json": "{}",
	}, false)

	identity, _, _ := detectIdentity(dir)
	if !identity.Monorepo {
		t.Error("Monorepo should be true for packages/*/package.json")
	}
}
