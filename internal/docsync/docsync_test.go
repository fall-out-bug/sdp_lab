package docsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckConsistencyBrokenLinkStrict(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	write(t, filepath.Join(root, "docs", "guide.md"), `See [missing](./missing.md).`)

	report, err := CheckConsistency(root, true)
	if err != nil {
		t.Fatalf("CheckConsistency error: %v", err)
	}
	if !report.HasErrors() {
		t.Fatalf("expected strict broken link error, got: %+v", report.Issues)
	}
}

func TestUpdateChangelog(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	write(t, filepath.Join(root, "docs", "note.md"), "first\n")

	git(t, root, "init")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "initial")

	write(t, filepath.Join(root, "docs", "note.md"), "second\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "docs update")

	path, err := UpdateChangelog(root, "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("UpdateChangelog error: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("docs", "CHANGELOG.md")) {
		t.Fatalf("unexpected changelog path: %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read changelog: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "### Commits") {
		t.Fatalf("changelog missing commits section: %s", content)
	}
	if !strings.Contains(content, "docs update") {
		t.Fatalf("changelog missing latest commit message: %s", content)
	}
}

func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

func write(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v; out=%s", args, err, string(out))
	}
}
