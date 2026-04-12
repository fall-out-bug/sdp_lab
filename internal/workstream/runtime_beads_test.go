package workstream

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	outputs         map[string][]byte
	outputErrs      map[string]error
	combinedOutputs map[string][]byte
	combinedErrs    map[string]error
	calls           []string
}

func (f *fakeRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, "OUT:"+key)
	if err := f.outputErrs[key]; err != nil {
		return nil, err
	}
	return f.outputs[key], nil
}

func (f *fakeRunner) CombinedOutput(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, "COMB:"+key)
	if err := f.combinedErrs[key]; err != nil {
		return nil, err
	}
	return f.combinedOutputs[key], nil
}

func (f *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, "RUN:"+key)
	return nil
}

func TestQueryBoundIssuesSuccess(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{
			"bd show sdplab-finding sdplab-primary --json": []byte(`[
				{"id":"sdplab-finding","status":"open","priority":0,"created_at":"2026-04-12T15:22:25Z","assignee":""},
				{"id":"sdplab-primary","status":"open","priority":2,"created_at":"2026-04-12T15:25:25Z","assignee":"Andrei"}
			]`),
		},
	}
	adapter := &ShellBeadsRuntimeAdapter{
		ProjectRoot: "/tmp/project",
		Runner:      runner,
		BDPath:      "bd",
	}
	leaf := WorkstreamLock{
		WSID:                "00-110-02",
		WSKind:              "leaf",
		BoundPrimaryIssueID: "sdplab-primary",
		FindingIssueIDs:     []string{"sdplab-finding"},
	}

	states, err := adapter.QueryBoundIssues(context.Background(), leaf)
	if err != nil {
		t.Fatalf("QueryBoundIssues: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("len(states) = %d, want 2", len(states))
	}
	if !states[1].IsClaimed {
		t.Fatal("expected assignee to mark issue as claimed")
	}
}

func TestQueryBoundIssuesMissingIssueFailsClosed(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{
			"bd show sdplab-primary sdplab-secondary --json": []byte(`[
				{"id":"sdplab-primary","status":"open","priority":1,"created_at":"2026-04-12T15:22:25Z","assignee":""}
			]`),
		},
	}
	adapter := &ShellBeadsRuntimeAdapter{ProjectRoot: "/tmp/project", Runner: runner, BDPath: "bd"}
	leaf := WorkstreamLock{
		WSID:                "00-110-02",
		WSKind:              "leaf",
		BoundPrimaryIssueID: "sdplab-primary",
		FindingIssueIDs:     []string{"sdplab-secondary"},
	}

	_, err := adapter.QueryBoundIssues(context.Background(), leaf)
	if err == nil {
		t.Fatal("expected missing issue error, got nil")
	}
	var queryErr *RuntimeQueryError
	if !errors.As(err, &queryErr) {
		t.Fatalf("expected RuntimeQueryError, got %T", err)
	}
	if queryErr.Reason != "not_found" {
		t.Fatalf("reason = %q, want not_found", queryErr.Reason)
	}
}

func TestResolveActiveIssuePrefersBlockingFinding(t *testing.T) {
	leaf := WorkstreamLock{
		WSID:                "00-110-02",
		WSKind:              "leaf",
		BoundPrimaryIssueID: "sdplab-primary",
		FindingIssueIDs:     []string{"sdplab-f2", "sdplab-f1"},
	}
	states := []RuntimeIssueState{
		{ID: "sdplab-primary", IsOpen: true, Priority: 1, CreatedAt: mustTime("2026-04-12T15:30:00Z")},
		{ID: "sdplab-f1", IsOpen: true, Priority: 1, CreatedAt: mustTime("2026-04-12T15:20:00Z")},
		{ID: "sdplab-f2", IsOpen: true, Priority: 0, CreatedAt: mustTime("2026-04-12T15:25:00Z")},
	}

	active := ResolveActiveIssue(leaf, states)
	if active == nil {
		t.Fatal("expected active issue, got nil")
	}
	if active.ID != "sdplab-f2" {
		t.Fatalf("active = %s, want sdplab-f2", active.ID)
	}
}

func TestCompetingClaimedIssuesExcludesAllowedIssue(t *testing.T) {
	leaf := WorkstreamLock{
		WSID:                "00-110-02",
		WSKind:              "leaf",
		BoundPrimaryIssueID: "sdplab-primary",
		FindingIssueIDs:     []string{"sdplab-f1", "sdplab-f2"},
	}
	states := []RuntimeIssueState{
		{ID: "sdplab-primary", IsClaimed: true},
		{ID: "sdplab-f1", IsClaimed: true},
		{ID: "sdplab-f2", IsClaimed: false},
	}

	conflicts := CompetingClaimedIssues(leaf, states, "sdplab-primary")
	if len(conflicts) != 1 || conflicts[0] != "sdplab-f1" {
		t.Fatalf("conflicts = %v, want [sdplab-f1]", conflicts)
	}
}

func TestClaimAndReleaseIssueCommands(t *testing.T) {
	runner := &fakeRunner{
		combinedOutputs: map[string][]byte{
			"bd update sdplab-primary --claim --json":           []byte(`[]`),
			"bd update sdplab-primary --status open -a  --json": []byte(`[]`),
		},
	}
	adapter := &ShellBeadsRuntimeAdapter{ProjectRoot: "/tmp/project", Runner: runner, BDPath: "bd"}

	if err := adapter.ClaimIssue(context.Background(), "sdplab-primary"); err != nil {
		t.Fatalf("ClaimIssue: %v", err)
	}
	if err := adapter.ReleaseClaim(context.Background(), "sdplab-primary"); err != nil {
		t.Fatalf("ReleaseClaim: %v", err)
	}
}

func mustTime(raw string) time.Time {
	tm, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		panic(err)
	}
	return tm.UTC()
}
