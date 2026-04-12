package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/agentloop"
)

// TestCmdNew_createSession verifies that cmdNew creates a session DB file.
func TestCmdNew_createSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	projectRoot := makeBoundHarnessProject(t)

	err := cmdNew([]string{
		"--session=test-session-123",
		"--project-root=" + projectRoot,
		"--feature=F110",
		"--ws=00-110-01",
	})
	if err != nil {
		t.Fatalf("cmdNew: %v", err)
	}

	dbFile := filepath.Join(dir, "test-session-123.db")
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("DB file not created: %v", err)
	}
}

// TestCmdRun_noAPIKey verifies that run fails with a clear error when OPENROUTER_API_KEY is absent.
func TestCmdRun_noAPIKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("OPENROUTER_API_KEY", "")
	projectRoot := makeBoundHarnessProject(t)

	// Create session first
	if err := cmdNew([]string{
		"--session=nokey-test",
		"--project-root=" + projectRoot,
		"--feature=F110",
		"--ws=00-110-01",
	}); err != nil {
		t.Fatalf("cmdNew: %v", err)
	}

	err := cmdRun([]string{"--session=nokey-test", "--prompt=hello"})
	if err == nil {
		t.Fatal("expected error when OPENROUTER_API_KEY is empty, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "API key") && !strings.Contains(errMsg, "OPENROUTER_API_KEY") {
		t.Errorf("error message %q does not mention API key requirement", errMsg)
	}
}

// TestCmdRun_terminatedSession verifies errors.Is(err, ErrHarnessTerminated) works.
func TestCmdRun_terminatedSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("OPENROUTER_API_KEY", "testkey")
	projectRoot := makeBoundHarnessProject(t)

	// Manually create a terminated session via the store
	path := filepath.Join(dir, "term-test.db")
	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_, err = agentloop.NewBoundSession("term-test", agentloop.SessionBinding{
		FeatureID:   "F110",
		WSID:        "00-110-01",
		ProjectRoot: projectRoot,
	}, store)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	// Stop the session to mark it terminated
	h, err := agentloop.RestoreHarness("term-test", "", store, nil, nil, nil)
	if err != nil {
		t.Fatalf("restore for stop: %v", err)
	}
	h.Stop(context.Background(), "")
	_ = store.Close()

	// Now attempt to run — RestoreHarness should return ErrHarnessTerminated
	err = cmdRun([]string{"--session=term-test", "--prompt=hello"})
	if err == nil {
		t.Fatal("expected error for terminated session, got nil")
	}
	if !errors.Is(err, agentloop.ErrHarnessTerminated) {
		t.Errorf("expected ErrHarnessTerminated, got: %v", err)
	}
}

func makeBoundHarnessProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdirAll(t, filepath.Join(root, "docs", "workstreams"))
	mkdirAll(t, filepath.Join(root, "docs", "roadmap"))

	writeFile(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F110** | Atomicity | 00-110-01 | Open |
`)
	writeFile(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), "# Roadmap\n\n- **F110** — Atomicity\n")
	writeFile(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-110-01.md"), `---
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

	runCmd(t, root, "git", "init")
	runCmd(t, root, "git", "config", "user.email", "test@example.com")
	runCmd(t, root, "git", "config", "user.name", "Test User")
	if err := cmdCompileLock([]string{"--project-root=" + root}); err != nil {
		t.Fatalf("cmdCompileLock: %v", err)
	}
	runCmd(t, root, "git", "add", ".")
	runCmd(t, root, "git", "commit", "-m", "seed workgraph")
	return root
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, string(out))
	}
}
