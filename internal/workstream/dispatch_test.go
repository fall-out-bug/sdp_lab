package workstream

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAcquireExecutionClaimReleasesOnRevalidationFailure(t *testing.T) {
	root := seedDispatchProject(t)
	runner := &fakeRunner{
		outputs: map[string][]byte{
			"bd show sdplab-62nw --json": []byte(`[
				{"id":"sdplab-62nw","status":"open","priority":2,"created_at":"2026-04-12T15:25:25Z","assignee":""}
			]`),
			"bd show sdplab-62nw --json#2": []byte(`[
				{"id":"sdplab-62nw","status":"closed","priority":2,"created_at":"2026-04-12T15:25:25Z","assignee":"Andrei"}
			]`),
		},
		combinedOutputs: map[string][]byte{
			"bd update sdplab-62nw --claim --json":           []byte(`[]`),
			"bd update sdplab-62nw --status open -a  --json": []byte(`[]`),
		},
	}
	adapter := &ShellBeadsRuntimeAdapter{ProjectRoot: root, Runner: newSequencedRunner(runner), BDPath: "bd"}

	_, err := AcquireExecutionClaim(context.Background(), root, "F110", "00-110-01", DefaultCompileOptions(), adapter)
	if err == nil {
		t.Fatal("expected revalidation failure, got nil")
	}
	var dispatchErr *DispatchError
	if !errors.As(err, &dispatchErr) {
		t.Fatalf("expected DispatchError, got %T", err)
	}
	if dispatchErr.Code != "dispatch_aborted_revalidation" {
		t.Fatalf("code = %q, want dispatch_aborted_revalidation", dispatchErr.Code)
	}
	assertCallSeen(t, runner.calls, "COMB:bd update sdplab-62nw --status open -a  --json")
}

func TestRevalidateExecutionClaimRejectsChangedActiveIssue(t *testing.T) {
	root := seedDispatchProjectWithFinding(t)
	runner := &fakeRunner{
		outputs: map[string][]byte{
			"bd show sdplab-62nw sdplab-finding --json": []byte(`[
				{"id":"sdplab-62nw","status":"open","priority":2,"created_at":"2026-04-12T15:25:25Z","assignee":"Andrei"},
				{"id":"sdplab-finding","status":"open","priority":0,"created_at":"2026-04-12T15:20:25Z","assignee":""}
			]`),
		},
	}
	adapter := &ShellBeadsRuntimeAdapter{ProjectRoot: root, Runner: runner, BDPath: "bd"}

	_, err := RevalidateExecutionClaim(context.Background(), root, "F110", "00-110-01", "sdplab-62nw", DefaultCompileOptions(), adapter)
	if err == nil {
		t.Fatal("expected active issue change error, got nil")
	}
	var dispatchErr *DispatchError
	if !errors.As(err, &dispatchErr) {
		t.Fatalf("expected DispatchError, got %T", err)
	}
	if dispatchErr.Code != "active_issue_changed" {
		t.Fatalf("code = %q, want active_issue_changed", dispatchErr.Code)
	}
	if dispatchErr.IssueID != "sdplab-finding" {
		t.Fatalf("issue = %q, want sdplab-finding", dispatchErr.IssueID)
	}
}

func TestRevalidateExecutionClaimSuccess(t *testing.T) {
	root := seedDispatchProject(t)
	runner := &fakeRunner{
		outputs: map[string][]byte{
			"bd show sdplab-62nw --json": []byte(`[
				{"id":"sdplab-62nw","status":"open","priority":2,"created_at":"2026-04-12T15:25:25Z","assignee":"Andrei"}
			]`),
		},
	}
	adapter := &ShellBeadsRuntimeAdapter{ProjectRoot: root, Runner: runner, BDPath: "bd"}

	lease, err := RevalidateExecutionClaim(context.Background(), root, "F110", "00-110-01", "sdplab-62nw", DefaultCompileOptions(), adapter)
	if err != nil {
		t.Fatalf("RevalidateExecutionClaim: %v", err)
	}
	if lease.ClaimedIssueID != "sdplab-62nw" {
		t.Fatalf("ClaimedIssueID = %q, want sdplab-62nw", lease.ClaimedIssueID)
	}
	if lease.Target.Workstream.WSID != "00-110-01" {
		t.Fatalf("ws = %q, want 00-110-01", lease.Target.Workstream.WSID)
	}
}

type sequencedRunner struct {
	*fakeRunner
	outputCount map[string]int
}

func newSequencedRunner(inner *fakeRunner) *sequencedRunner {
	return &sequencedRunner{
		fakeRunner:  inner,
		outputCount: map[string]int{},
	}
}

func (s *sequencedRunner) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	key := name + " " + joinArgs(args)
	s.outputCount[key]++
	seqKey := key
	if s.outputCount[key] > 1 {
		seqKey = key + "#" + itoa(s.outputCount[key])
	}
	if payload, ok := s.outputs[seqKey]; ok {
		s.calls = append(s.calls, "OUT:"+key)
		return payload, s.outputErrs[seqKey]
	}
	return s.fakeRunner.Output(ctx, dir, name, args...)
}

func seedDispatchProject(t *testing.T) string {
	t.Helper()
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

	run(t, root, "git", "init")
	run(t, root, "git", "config", "user.email", "test@example.com")
	run(t, root, "git", "config", "user.name", "Test User")

	lock, report, err := CompileWorkgraphLock(root, DefaultCompileOptions())
	if err != nil {
		t.Fatalf("CompileWorkgraphLock: %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("compile errors: %+v", report.Issues)
	}
	if err := WriteWorkgraphLock(root, lock); err != nil {
		t.Fatalf("WriteWorkgraphLock: %v", err)
	}
	run(t, root, "git", "add", ".")
	run(t, root, "git", "commit", "-m", "seed workgraph")
	return root
}

func seedDispatchProjectWithFinding(t *testing.T) string {
	t.Helper()
	root := seedDispatchProject(t)
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
- finding: sdplab-finding

## Acceptance Criteria

- [ ] Implement strict execution contract
`)
	lock, report, err := CompileWorkgraphLock(root, DefaultCompileOptions())
	if err != nil {
		t.Fatalf("CompileWorkgraphLock: %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("compile errors: %+v", report.Issues)
	}
	if err := WriteWorkgraphLock(root, lock); err != nil {
		t.Fatalf("WriteWorkgraphLock: %v", err)
	}
	run(t, root, "git", "add", ".")
	run(t, root, "git", "commit", "-m", "add finding")
	return root
}

func assertCallSeen(t *testing.T, calls []string, want string) {
	t.Helper()
	for _, call := range calls {
		if call == want {
			return
		}
	}
	t.Fatalf("expected call %q, got %v", want, calls)
}

func joinArgs(args []string) string {
	out := ""
	for i, arg := range args {
		if i > 0 {
			out += " "
		}
		out += arg
	}
	return out
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
