package scout

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ── Helpers ─────────────────────────────────────────────────────────────

// createTempRepo creates a temp directory with files and optional git init.
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
	env := []string{
		"GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_DATE=" + date,
	}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_AUTHOR_DATE=") && !strings.HasPrefix(e, "GIT_COMMITTER_DATE=") {
			env = append(env, e)
		}
	}
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Env = env
	cmd.CombinedOutput()
	cmd = exec.Command("git", "commit", "-m", "update "+file)
	cmd.Dir = dir
	cmd.Env = env
	cmd.CombinedOutput()
}

// ── Phase 1: Identity ──────────────────────────────────────────────────

func TestIdentityDetectsGoProject(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":       "module example.com/app\ngo 1.26\n",
		"main.go":      "package main\nfunc main() {}\n",
		"README.md":    "# My App\nA cool application for testing.\nMore text here.\n",
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
		"package.json":              "{}",
		"packages/a/package.json":   "{}",
		"packages/b/package.json":   "{}",
	}, false)

	identity, _, _ := detectIdentity(dir)

	if !identity.Monorepo {
		t.Error("Monorepo should be true for packages/*/package.json")
	}
}

// ── Phase 2: Scale ─────────────────────────────────────────────────────

func TestScaleCountsFilesAndLOC(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":    "module example.com/app\ngo 1.26\n",
		"main.go":   "package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n",
		"util.go":   "package main\n\nfunc util() {}\n",
		"main_test.go": "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n",
	}, false)

	scale := detectScale(dir, nil)

	if scale.SourceFiles != 2 {
		t.Errorf("SourceFiles = %d, want 2", scale.SourceFiles)
	}
	if scale.TestFiles != 1 {
		t.Errorf("TestFiles = %d, want 1", scale.TestFiles)
	}
	if scale.TotalLoc < 4 {
		t.Errorf("TotalLoc = %d, want >= 4", scale.TotalLoc)
	}
}

func TestScaleExcludesVendorAndGenerated(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"main.go":             "package main\nfunc main() {}\n",
		"vendor/lib.go":       "package vendor\nfunc Lib() {}\n",
		"foo.pb.go":          "// generated\npackage foo\n",
		"node_modules/a.js":  "module.exports = {};\n",
	}, false)

	scale := detectScale(dir, nil)

	if scale.VendorFiles != 0 {
		t.Errorf("VendorFiles = %d, want 0 (vendor files excluded)", scale.VendorFiles)
	}
	// vendor/node_modules files should not appear in source count
	if scale.SourceFiles > 1 {
		t.Errorf("SourceFiles = %d, want <= 1 (only main.go)", scale.SourceFiles)
	}
}

func TestScaleSkipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	// Write a file with null bytes (binary)
	binFile := filepath.Join(dir, "image.png")
	binContent := make([]byte, 100)
	binContent[10] = 0x00 // null byte
	os.WriteFile(binFile, binContent, 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)

	scale := detectScale(dir, nil)

	if scale.TotalLoc != 1 {
		t.Errorf("TotalLoc = %d, want 1 (binary skipped)", scale.TotalLoc)
	}
}

// ── Phase 3: Activity ──────────────────────────────────────────────────

func TestActivityFromGitRepo(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	// Add another commit on a different date
	addAndCommit(t, dir, "util.go", "package main\nfunc util() {}\n",
		"2026-04-10T10:00:00Z")

	activity := detectActivity(dir)

	if activity.TotalCommits < 2 {
		t.Errorf("TotalCommits = %d, want >= 2", activity.TotalCommits)
	}
	if activity.Contributors < 1 {
		t.Errorf("Contributors = %d, want >= 1", activity.Contributors)
	}
	if activity.FirstCommit == nil {
		t.Error("FirstCommit should not be nil")
	}
	if activity.LastCommit == nil {
		t.Error("LastCommit should not be nil")
	}
}

func TestActivityEmptyDir(t *testing.T) {
	dir := t.TempDir()
	activity := detectActivity(dir)

	if activity.TotalCommits != 0 {
		t.Errorf("TotalCommits = %d, want 0 for non-git dir", activity.TotalCommits)
	}
	if activity.FirstCommit != nil {
		t.Error("FirstCommit should be nil for non-git dir")
	}
}

// ── Health Signals ─────────────────────────────────────────────────────

func TestHealthDerivesCorrectSignals(t *testing.T) {
	card := &ProjectCard{
		Scale: Scale{
			SourceFiles: 100,
			TestFiles:   30,
			TestRatio:   0.3,
			TotalFiles:  200,
			MaxFileLoc:  150,
			DepthMax:    4,
		},
		Activity: Activity{
			Contributors: 5,
			Commits30d:   100,
			LastCommit:   strPtr("2026-04-10"),
		},
	}

	deriveHealthSignals(card)

	if card.Health.CommitFrequency != CommitFreqHigh {
		t.Errorf("CommitFrequency = %q, want %q", card.Health.CommitFrequency, CommitFreqHigh)
	}
	if card.Health.TestCoverageHint != CovGood {
		t.Errorf("TestCoverageHint = %q, want %q", card.Health.TestCoverageHint, CovGood)
	}
	if card.Health.Staleness != StalenessActive {
		t.Errorf("Staleness = %q, want %q", card.Health.Staleness, StalenessActive)
	}
}

// ── Integration: Run on sdp_lab ────────────────────────────────────────

func TestIntegrationOnSelf(t *testing.T) {
	// Find project root (go up from .worktrees/F120/internal/scout)
	root := filepath.Join("..", "..", "..", "..")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	// Verify it's actually sdp_lab
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		t.Skip("not running from sdp_lab worktree")
	}

	card, err := Run(abs)
	if err != nil {
		t.Fatalf("Run(%s): %v", abs, err)
	}

	if card.Identity.PrimaryLanguage != "go" {
		t.Errorf("PrimaryLanguage = %q, want go", card.Identity.PrimaryLanguage)
	}
	if card.Identity.BuildSystem == nil || *card.Identity.BuildSystem != "go-modules" {
		t.Error("BuildSystem should be go-modules")
	}
	if card.Scale.TotalFiles < 10 {
		t.Errorf("TotalFiles = %d, want >= 10", card.Scale.TotalFiles)
	}
	if card.Scale.TotalLoc < 1000 {
		t.Errorf("TotalLoc = %d, want >= 1000", card.Scale.TotalLoc)
	}
	if card.Activity.TotalCommits < 10 {
		t.Errorf("TotalCommits = %d, want >= 10", card.Activity.TotalCommits)
	}
	if card.Maturity.HasReadme != true {
		t.Error("sdp_lab should have README")
	}
}

// ── Pipeline on temp repo ──────────────────────────────────────────────

func TestPipelineOnTempRepo(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":         "module example.com/app\ngo 1.26\nrequire (\n\tfmt v0.0.0\n)\n",
		"main.go":        "package main\nfunc main() { println(\"hello\") }\n",
		"main_test.go":   "package main\nimport \"testing\"\nfunc TestMain(t *testing.T) {}\n",
		"README.md":      "# App\nA test app.\n",
		".goreleaser.yml": "builds:\n  - main: .\n",
	}, true)

	card, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify all sections populated
	if card.Version != "1.0.0" {
		t.Errorf("Version = %q", card.Version)
	}
	if card.DurationMs <= 0 {
		t.Error("DurationMs should be > 0")
	}
	if card.Identity.PrimaryLanguage != "go" {
		t.Errorf("PrimaryLanguage = %q", card.Identity.PrimaryLanguage)
	}
	if card.Scale.SourceFiles < 1 {
		t.Errorf("SourceFiles = %d, want >= 1", card.Scale.SourceFiles)
	}
	if card.Activity.TotalCommits < 1 {
		t.Errorf("TotalCommits = %d, want >= 1", card.Activity.TotalCommits)
	}
	if card.Health.CommitFrequency == "" {
		t.Error("Health signals should be derived")
	}
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
