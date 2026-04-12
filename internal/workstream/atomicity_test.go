package workstream

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileWorkgraphLockNormalizedLeaf(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F110** | Atomicity | 00-110-01 | Open |
`)
	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), "# Roadmap\n\n- **F110** — Atomicity\n")
	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-110-01.md"), `---
ws_id: 00-110-01
feature_id: F110
status: open
priority: P1
size: M
depends_on: []
ws_kind: leaf
parent_ws_id: null
dispatch_lifecycle: active
---

# 00-110-01: Atomicity

## Beads

- primary: sdplab-62nw

## Acceptance Criteria

- [ ] Implement strict execution contract
`)

	lock, report, err := CompileWorkgraphLock(root, DefaultCompileOptions())
	if err != nil {
		t.Fatalf("CompileWorkgraphLock error: %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("expected no compile errors, got %+v", report.Issues)
	}
	if len(lock.Features) != 1 {
		t.Fatalf("len(lock.Features) = %d, want 1", len(lock.Features))
	}
	ws := lock.Features[0].Workstreams[0]
	if ws.WSKind != "leaf" {
		t.Fatalf("ws_kind = %q, want leaf", ws.WSKind)
	}
	if ws.BoundPrimaryIssueID != "sdplab-62nw" {
		t.Fatalf("primary = %q, want sdplab-62nw", ws.BoundPrimaryIssueID)
	}
	if !strings.HasPrefix(lock.SourceInputsHash, "sha256:") {
		t.Fatalf("source_inputs_hash = %q, want sha256:*", lock.SourceInputsHash)
	}
}

func TestCompileWorkgraphLockMixedInvalidFeature(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F110** | Atomicity | 00-110-01, 00-110-02 | Open |
`)
	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), "# Roadmap\n\n- **F110** — Atomicity\n")
	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-110-01.md"), `---
ws_id: 00-110-01
feature_id: F110
status: open
priority: P1
size: M
depends_on: []
ws_kind: leaf
parent_ws_id: null
dispatch_lifecycle: active
---

# 00-110-01: Atomicity

## Beads

- primary: sdplab-62nw

## Acceptance Criteria

- [ ] Implement strict execution contract
`)
	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-110-02.md"), `---
ws_id: 00-110-02
feature_id: F110
status: backlog
priority: P2
size: S
depends_on: []
---

# 00-110-02: Legacy sibling

## Beads

- sdplab-999: legacy

## Acceptance Criteria

- [ ] Still legacy
`)

	lock, report, err := CompileWorkgraphLock(root, DefaultCompileOptions())
	if err != nil {
		t.Fatalf("CompileWorkgraphLock error: %v", err)
	}
	if !report.HasErrors() {
		t.Fatalf("expected mixed_invalid errors, got %+v", report.Issues)
	}
	if len(lock.Features) != 0 {
		t.Fatalf("expected mixed_invalid feature to be excluded from lock, got %+v", lock.Features)
	}
}

func TestReadFreshWorkgraphLockRejectsDirtyNormalizedSources(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F110** | Atomicity | 00-110-01 | Open |
`)
	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), "# Roadmap\n\n- **F110** — Atomicity\n")
	wsPath := filepath.Join(root, "docs", "workstreams", "backlog", "00-110-01.md")
	write(t, wsPath, `---
ws_id: 00-110-01
feature_id: F110
status: open
priority: P1
size: M
depends_on: []
ws_kind: leaf
parent_ws_id: null
dispatch_lifecycle: active
---

# 00-110-01: Atomicity

## Beads

- primary: sdplab-62nw

## Acceptance Criteria

- [ ] Implement strict execution contract
`)

	run(t, root, "git", "init")
	run(t, root, "git", "config", "user.email", "test@example.com")
	run(t, root, "git", "config", "user.name", "Test User")

	lock, report, err := CompileWorkgraphLock(root, DefaultCompileOptions())
	if err != nil {
		t.Fatalf("CompileWorkgraphLock error: %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("unexpected compile errors: %+v", report.Issues)
	}
	if err := WriteWorkgraphLock(root, lock); err != nil {
		t.Fatalf("WriteWorkgraphLock error: %v", err)
	}

	run(t, root, "git", "add", ".")
	run(t, root, "git", "commit", "-m", "seed workgraph")

	if _, err := ReadFreshWorkgraphLock(root, DefaultCompileOptions()); err != nil {
		t.Fatalf("ReadFreshWorkgraphLock should pass on clean tree: %v", err)
	}

	if err := os.WriteFile(wsPath, []byte(strings.ReplaceAll(readFile(t, wsPath), "status: open", "status: blocked")), 0o644); err != nil {
		t.Fatalf("rewrite workstream: %v", err)
	}
	if _, err := ReadFreshWorkgraphLock(root, DefaultCompileOptions()); err == nil {
		t.Fatal("expected dirty/stale workgraph lock error, got nil")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, string(out))
	}
}
