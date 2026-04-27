package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
)

func TestRunStatus_NoDecision(t *testing.T) {
	dir := setupTestProfiles(t)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"status", "-project", dir}
	err := runStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStatus_WithDecision(t *testing.T) {
	dir := setupTestProfiles(t)

	dec := &dispatch.DispatchDecision{
		Harness:   "claude-code",
		Provider:  "zai",
		Model:     "glm-5",
		Score:     0.85,
		Reason:    "highest effective score for feature:go",
		Timestamp: "2026-03-28T12:00:00Z",
	}
	if err := dispatch.WriteDecision(dir, dec); err != nil {
		t.Fatalf("writing test decision: %v", err)
	}

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"status", "-project", dir}
	err := runStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStatus_JSON(t *testing.T) {
	dir := setupTestProfiles(t)

	dec := &dispatch.DispatchDecision{
		Harness:   "claude-code",
		Provider:  "zai",
		Model:     "glm-5",
		Score:     0.85,
		Reason:    "test",
		Timestamp: "2026-03-28T12:00:00Z",
	}
	if err := dispatch.WriteDecision(dir, dec); err != nil {
		t.Fatalf("writing test decision: %v", err)
	}

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	os.Args = []string{"status", "-json", "-project", dir}
	err := runStatus()
	_ = w.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out statusOutput
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}

	if out.Decision == nil {
		t.Fatal("expected decision in JSON output")
	}
	if out.Decision.Harness != "claude-code" {
		t.Fatalf("expected harness claude-code, got %s", out.Decision.Harness)
	}
	if out.ProfileCount != 2 {
		t.Fatalf("expected 2 profiles, got %d", out.ProfileCount)
	}
	if out.ColdStart != "capability-heuristic" {
		t.Fatalf("expected capability-heuristic, got %s", out.ColdStart)
	}
}

func TestRunStatus_EmptyProject(t *testing.T) {
	dir := t.TempDir()

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"status", "-project", dir}
	err := runStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
