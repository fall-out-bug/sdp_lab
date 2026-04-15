package metrics

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// ── Test Helpers ──────────────────────────────────────────────────

func createTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test Author")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func commitFile(t *testing.T, dir, path, content, date, msg string) {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", path)
	env := append(os.Environ(),
		"GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_DATE="+date,
	)
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit in %s: %v\n%s", dir, err, out)
	}
}

// ── AC1: One collector pass captures raw commit/change data ────────

func TestCollectCapturesCommitsWithFiles(t *testing.T) {
	dir := createTempGitRepo(t)
	commitFile(t, dir, "main.go", "package main\nfunc main() {}\n", "2026-04-01T10:00:00Z", "feat: initial")
	commitFile(t, dir, "util.go", "package main\nfunc util() {}\n", "2026-04-02T10:00:00Z", "feat: add util")

	data, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Commits) < 2 {
		t.Fatalf("Commits = %d, want >= 2", len(data.Commits))
	}
	// Verify file changes are captured
	var totalFiles int
	for _, c := range data.Commits {
		totalFiles += len(c.Files)
	}
	if totalFiles < 2 {
		t.Errorf("total file changes = %d, want >= 2", totalFiles)
	}
	// Verify hash, author, date, subject are populated
	for _, c := range data.Commits {
		if c.Hash == "" {
			t.Error("commit missing hash")
		}
		if c.Author == "" {
			t.Error("commit missing author")
		}
		if c.Date.IsZero() {
			t.Error("commit missing date")
		}
		if c.Subject == "" {
			t.Error("commit missing subject")
		}
	}
}

func TestCollectTags(t *testing.T) {
	dir := createTempGitRepo(t)
	commitFile(t, dir, "main.go", "package main\n", "2026-04-01T10:00:00Z", "feat: initial")
	runGit(t, dir, "tag", "v1.0.0")
	commitFile(t, dir, "fix.go", "package main\n", "2026-04-02T10:00:00Z", "fix: bug")
	runGit(t, dir, "tag", "v1.1.0")

	data, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Tags) < 2 {
		t.Errorf("Tags = %d, want >= 2", len(data.Tags))
	}
	semverCount := 0
	for _, tag := range data.Tags {
		if tag.IsSemver {
			semverCount++
		}
	}
	if semverCount < 2 {
		t.Errorf("semver tags = %d, want >= 2", semverCount)
	}
}

// ── AC2: Generated, bot, and formatting-only noise filtered ────────

func TestFilterRemovesBotCommits(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{Hash: "a1", Author: "dependabot[bot]", Subject: "chore: bump"},
			{Hash: "a2", Author: "Alice", Subject: "feat: real work"},
			{Hash: "a3", Author: "renovate[bot]", Subject: "chore: update"},
		},
	}
	filtered := Filter(data)
	if len(filtered.Commits) != 1 {
		t.Errorf("filtered commits = %d, want 1", len(filtered.Commits))
	}
	if filtered.Commits[0].Author != "Alice" {
		t.Errorf("remaining author = %q, want Alice", filtered.Commits[0].Author)
	}
}

func TestFilterRemovesGeneratedFiles(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{
				Hash: "a1", Author: "Alice", Subject: "feat: add proto",
				Files: []FileChange{
					{Added: 100, Deleted: 0, Path: "api.pb.go"},
					{Added: 50, Deleted: 0, Path: "handler.go"},
				},
			},
		},
	}
	filtered := Filter(data)
	if len(filtered.Commits) != 1 {
		t.Fatal("commit should be kept")
	}
	if len(filtered.Commits[0].Files) != 1 {
		t.Errorf("files after filter = %d, want 1", len(filtered.Commits[0].Files))
	}
	if filtered.Commits[0].Files[0].Path != "handler.go" {
		t.Errorf("remaining file = %q, want handler.go", filtered.Commits[0].Files[0].Path)
	}
}

func TestFilterRemovesCIOnlyCommits(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{
				Hash: "a1", Author: "CI", Subject: "ci: update workflows",
				Files: []FileChange{
					{Added: 10, Deleted: 5, Path: ".github/workflows/ci.yml"},
				},
			},
			{
				Hash: "a2", Author: "Alice", Subject: "feat: real",
				Files: []FileChange{
					{Added: 30, Deleted: 0, Path: "main.go"},
				},
			},
		},
	}
	filtered := Filter(data)
	if len(filtered.Commits) != 1 {
		t.Errorf("filtered commits = %d, want 1 (CI-only removed)", len(filtered.Commits))
	}
}

func TestFilterRemovesFormattingOnlyCommits(t *testing.T) {
	files := make([]FileChange, 10)
	for i := range files {
		files[i] = FileChange{Added: 50, Deleted: 48, Path: "file.go"}
	}
	data := &GitData{
		Commits: []RawCommit{
			{Hash: "a1", Author: "Alice", Subject: "style: reformat", Files: files},
			{Hash: "a2", Author: "Alice", Subject: "feat: real", Files: []FileChange{{Added: 30, Deleted: 0, Path: "main.go"}}},
		},
	}
	filtered := Filter(data)
	if len(filtered.Commits) != 1 {
		t.Errorf("filtered commits = %d, want 1 (formatting removed)", len(filtered.Commits))
	}
}

// ── AC3: MetricsReport contract stable for JSON rendering ──────────

func TestMetricsReportJSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	report := MetricsReport{
		Version:         "1.0.0",
		GeneratedAt:     now,
		RepoPath:        "/tmp/test",
		DurationMs:      150,
		CommitsAnalyzed: 42,
		Period: TimePeriod{
			From: time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
			To:   now,
		},
	}
	b, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	// Verify key fields in JSON
	s := string(b)
	if !contains(s, `"version"`) || !contains(s, `"1.0.0"`) {
		t.Error("missing version in JSON")
	}
	if !contains(s, `"commits_analyzed"`) {
		t.Error("missing commits_analyzed in JSON")
	}
	if !contains(s, `"period"`) {
		t.Error("missing period in JSON")
	}
}

// ── AC4: Collector tests for edge cases ────────────────────────────

func TestCollectEmptyRepo(t *testing.T) {
	dir := createTempGitRepo(t)
	// No commits

	data, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Commits) != 0 {
		t.Errorf("Commits = %d, want 0 for empty repo", len(data.Commits))
	}
	if len(data.Tags) != 0 {
		t.Errorf("Tags = %d, want 0 for empty repo", len(data.Tags))
	}
}

func TestCollectSingleAuthorRepo(t *testing.T) {
	dir := createTempGitRepo(t)
	for i := range 5 {
		name := "file" + string(rune('A'+i)) + ".go"
		commitFile(t, dir, name, "package main\n", "2026-04-01T10:00:00Z", "commit "+string(rune('A'+i)))
	}

	data, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Commits) < 5 {
		t.Errorf("Commits = %d, want >= 5", len(data.Commits))
	}
	// All by same author
	for _, c := range data.Commits {
		if c.Author != "Test Author" {
			t.Errorf("Author = %q, want Test Author", c.Author)
		}
	}
}

func TestCollectRepoWithoutTags(t *testing.T) {
	dir := createTempGitRepo(t)
	commitFile(t, dir, "main.go", "package main\n", "2026-04-01T10:00:00Z", "initial")

	data, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Commits) < 1 {
		t.Fatal("expected at least 1 commit")
	}
	if len(data.Tags) != 0 {
		t.Errorf("Tags = %d, want 0 (no tags created)", len(data.Tags))
	}
}

// ── AC5: Metrics respect shared exclusion rules ────────────────────

func TestFilterRespectsVendorExclusion(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{
				Hash: "a1", Author: "Alice", Subject: "feat: vendor update",
				Files: []FileChange{
					{Added: 100, Deleted: 0, Path: "vendor/pkg/util.go"},
					{Added: 50, Deleted: 0, Path: "internal/app.go"},
				},
			},
		},
	}
	filtered := Filter(data)
	if len(filtered.Commits) != 1 {
		t.Fatal("commit should be kept (has non-vendor files)")
	}
	if len(filtered.Commits[0].Files) != 1 {
		t.Errorf("files = %d, want 1 (vendor excluded)", len(filtered.Commits[0].Files))
	}
}

func TestFilterRespectsNodeModulesExclusion(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{
				Hash: "a1", Author: "Alice", Subject: "feat: add dep",
				Files: []FileChange{
					{Added: 500, Deleted: 0, Path: "node_modules/lodash/index.js"},
					{Added: 10, Deleted: 0, Path: "src/app.ts"},
				},
			},
		},
	}
	filtered := Filter(data)
	if len(filtered.Commits[0].Files) != 1 {
		t.Errorf("files = %d, want 1 (node_modules excluded)", len(filtered.Commits[0].Files))
	}
}

// ── Context cancellation ───────────────────────────────────────────

func TestCollectWithContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := CollectWithContext(ctx, "/tmp")
	if err == nil {
		t.Error("expected error with cancelled context")
	}
}

// ── Helper functions for filter types ──────────────────────────────

func TestIsBot(t *testing.T) {
	tests := []struct {
		author string
		want   bool
	}{
		{"dependabot[bot]", true},
		{"renovate[bot]", true},
		{"github-actions", true},
		{"Alice", false},
		{"Bob", false},
		{"mergify[bot]", true},
	}
	for _, tt := range tests {
		got := IsBot(tt.author)
		if got != tt.want {
			t.Errorf("IsBot(%q) = %v, want %v", tt.author, got, tt.want)
		}
	}
}

func TestIsGeneratedFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"api.pb.go", true},
		{"foo.generated.go", true},
		{"app.min.js", true},
		{"go.sum", true},
		{"package-lock.json", true},
		{"handler.go", false},
		{"main.py", false},
	}
	for _, tt := range tests {
		got := IsGeneratedFile(tt.path)
		if got != tt.want {
			t.Errorf("IsGeneratedFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsFormattingOnly(t *testing.T) {
	// 3 files with balanced adds/deletes → formatting
	files := []FileChange{
		{Added: 50, Deleted: 48, Path: "a.go"},
		{Added: 30, Deleted: 28, Path: "b.go"},
		{Added: 20, Deleted: 19, Path: "c.go"},
	}
	if !IsFormattingOnly(files) {
		t.Error("expected formatting-only detection")
	}

	// Unbalanced → not formatting
	unbalanced := []FileChange{
		{Added: 100, Deleted: 5, Path: "a.go"},
		{Added: 50, Deleted: 3, Path: "b.go"},
		{Added: 30, Deleted: 2, Path: "c.go"},
	}
	if IsFormattingOnly(unbalanced) {
		t.Error("should not detect as formatting-only")
	}
}
