package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/agentloop"
	"sdp_dev/internal/workstream"
)

// TestCmdNew_createSession verifies that cmdNew creates a session DB file.
func TestCmdNew_createSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("SDP_HARNESS_BD_PATH", installFakeBD(t, fakeBDIssue{
		ID:        "sdplab-62nw",
		Status:    "open",
		Priority:  2,
		CreatedAt: "2026-04-12T15:25:25Z",
	}))
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

	store, err := agentloop.NewSQLiteStore(dbFile)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	events, err := store.LoadEvents("test-session-123")
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("expected dispatch events, got %+v", events)
	}
	if events[0].Type != "dispatch_metric" || events[0].Code != workstream.DispatchAttemptTotal {
		t.Fatalf("first event = %+v, want dispatch attempt metric", events[0])
	}
	if events[1].Type != "dispatch_metric" || events[1].Code != workstream.DispatchSuccessTotal {
		t.Fatalf("second event = %+v, want dispatch success metric", events[1])
	}
	if events[2].Type != "dispatch_diagnostic" || events[2].Code != "dispatch_success" {
		t.Fatalf("third event = %+v, want dispatch success diagnostic", events[2])
	}
}

// TestCmdRun_noAPIKey verifies that run fails with a clear error when OPENROUTER_API_KEY is absent.
func TestCmdRun_noAPIKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("SDP_HARNESS_BD_PATH", installFakeBD(t, fakeBDIssue{
		ID:        "sdplab-62nw",
		Status:    "open",
		Priority:  2,
		CreatedAt: "2026-04-12T15:25:25Z",
	}))
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

func TestCmdRelease_clearsClaim(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("SDP_HARNESS_BD_PATH", installFakeBD(t, fakeBDIssue{
		ID:        "sdplab-62nw",
		Status:    "open",
		Priority:  2,
		CreatedAt: "2026-04-12T15:25:25Z",
	}))
	projectRoot := makeBoundHarnessProject(t)

	if err := cmdNew([]string{
		"--session=release-test",
		"--project-root=" + projectRoot,
		"--feature=F110",
		"--ws=00-110-01",
	}); err != nil {
		t.Fatalf("cmdNew: %v", err)
	}
	if err := cmdRelease([]string{"--session=release-test"}); err != nil {
		t.Fatalf("cmdRelease: %v", err)
	}

	store, err := agentloop.NewSQLiteStore(filepath.Join(dir, "release-test.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()
	session, err := agentloop.RecoverSession("release-test", store)
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}
	if session.ClaimedIssueID != "" {
		t.Fatalf("ClaimedIssueID = %q, want empty", session.ClaimedIssueID)
	}
}

func TestCmdEvents_printsStructuredDispatchEvents(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("SDP_HARNESS_BD_PATH", installFakeBD(t, fakeBDIssue{
		ID:        "sdplab-62nw",
		Status:    "open",
		Priority:  2,
		CreatedAt: "2026-04-12T15:25:25Z",
	}))
	projectRoot := makeBoundHarnessProject(t)

	if err := cmdNew([]string{
		"--session=events-test",
		"--project-root=" + projectRoot,
		"--feature=F110",
		"--ws=00-110-01",
	}); err != nil {
		t.Fatalf("cmdNew: %v", err)
	}

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	if err := cmdEvents([]string{"--session=events-test"}); err != nil {
		t.Fatalf("cmdEvents: %v", err)
	}
	_ = w.Close()
	payload, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}

	var events []agentloop.Event
	if err := json.Unmarshal(payload, &events); err != nil {
		t.Fatalf("unmarshal events: %v\n%s", err, payload)
	}
	if len(events) == 0 || events[0].Code != workstream.DispatchAttemptTotal {
		t.Fatalf("events = %+v, want dispatch attempt metric first", events)
	}
}

func TestCmdNew_queryFailurePersistsStructuredDiagnostic(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("SDP_HARNESS_BD_PATH", installFakeBD(t))
	projectRoot := makeBoundHarnessProject(t)

	origStderr := os.Stderr
	readErr, writeErr, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	os.Stderr = writeErr
	defer func() { os.Stderr = origStderr }()

	err := cmdNew([]string{
		"--session=query-fail-test",
		"--project-root=" + projectRoot,
		"--feature=F110",
		"--ws=00-110-01",
	})
	_ = writeErr.Close()
	if err == nil {
		t.Fatal("expected cmdNew failure, got nil")
	}
	stderrPayload, readPipeErr := io.ReadAll(readErr)
	if readPipeErr != nil {
		t.Fatalf("read stderr pipe: %v", readPipeErr)
	}
	if !strings.Contains(string(stderrPayload), `"code":"beads_query_failed"`) {
		t.Fatalf("stderr = %s, want structured beads_query_failed diagnostic", stderrPayload)
	}

	store, openErr := agentloop.NewSQLiteStore(filepath.Join(dir, "query-fail-test.db"))
	if openErr != nil {
		t.Fatalf("open store: %v", openErr)
	}
	defer store.Close()
	events, loadErr := store.LoadEvents("query-fail-test")
	if loadErr != nil {
		t.Fatalf("load events: %v", loadErr)
	}
	if len(events) < 2 {
		t.Fatalf("events = %+v, want attempt metric and failure diagnostic", events)
	}
	last := events[len(events)-1]
	if last.Type != "dispatch_diagnostic" || last.Code != "beads_query_failed" {
		t.Fatalf("last event = %+v, want beads_query_failed diagnostic", last)
	}
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
