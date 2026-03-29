package orchestrate_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"sdp_dev/internal/orchestrate"
)

func TestGenerateOrchestratorAttestation_WithDispatch(t *testing.T) {
	// Set up a minimal git repo so gitHeadSHA and friends work
	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	runGit("init")
	runGit("checkout", "-b", "master")

	// Create an initial commit on master
	dummyFile := filepath.Join(dir, "dummy.txt")
	if err := os.WriteFile(dummyFile, []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "dummy.txt")
	runGit("commit", "-m", "init")

	// Create feature branch
	runGit("checkout", "-b", "feature/dispatch-test")
	if err := os.WriteFile(filepath.Join(dir, "code.go"), []byte("package x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "code.go")
	runGit("commit", "-m", "add code")

	cp := &orchestrate.Checkpoint{
		FeatureID: "F099",
		Branch:    "feature/dispatch-test",
		Phase:     "build",
		Workstreams: []orchestrate.WSStatus{
			{
				ID:     "00-099-01",
				Status: "done",
				Dispatch: &orchestrate.WSDispatchInfo{
					Harness:   "claude-code",
					Provider:  "anthropic",
					Model:     "claude-sonnet-4-20250514",
					Score:     0.92,
					Reason:    "best match",
					ColdStart: true,
				},
			},
			{
				ID:     "00-099-02",
				Status: "pending",
			},
		},
	}

	stmt, err := orchestrate.GenerateOrchestratorAttestation(dir, cp)
	if err != nil {
		t.Fatalf("GenerateOrchestratorAttestation: %v", err)
	}

	// Dispatch evidence should be populated from first workstream
	if stmt.Predicate.Dispatch == nil {
		t.Fatal("expected Dispatch evidence, got nil")
	}
	de := stmt.Predicate.Dispatch
	if de.Harness != "claude-code" {
		t.Errorf("Dispatch.Harness = %q, want %q", de.Harness, "claude-code")
	}
	if de.Provider != "anthropic" {
		t.Errorf("Dispatch.Provider = %q, want %q", de.Provider, "anthropic")
	}
	if de.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Dispatch.Model = %q, want %q", de.Model, "claude-sonnet-4-20250514")
	}
	if de.Score != 0.92 {
		t.Errorf("Dispatch.Score = %v, want 0.92", de.Score)
	}
	if de.Reason != "best match" {
		t.Errorf("Dispatch.Reason = %q, want %q", de.Reason, "best match")
	}
	if !de.ColdStart {
		t.Error("Dispatch.ColdStart should be true")
	}

	// Provenance.Model should be populated from dispatch
	if stmt.Predicate.Provenance.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Provenance.Model = %q, want %q", stmt.Predicate.Provenance.Model, "claude-sonnet-4-20250514")
	}
}

func TestGenerateOrchestratorAttestation_NoDispatch(t *testing.T) {
	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	runGit("init")
	runGit("checkout", "-b", "master")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "f.txt")
	runGit("commit", "-m", "init")
	runGit("checkout", "-b", "feature/no-dispatch")

	cp := &orchestrate.Checkpoint{
		FeatureID: "F100",
		Branch:    "feature/no-dispatch",
		Phase:     "build",
		Workstreams: []orchestrate.WSStatus{
			{ID: "00-100-01", Status: "pending"},
		},
	}

	stmt, err := orchestrate.GenerateOrchestratorAttestation(dir, cp)
	if err != nil {
		t.Fatalf("GenerateOrchestratorAttestation: %v", err)
	}

	if stmt.Predicate.Dispatch != nil {
		t.Errorf("expected nil Dispatch, got %+v", stmt.Predicate.Dispatch)
	}
	if stmt.Predicate.Provenance.Model != "" {
		t.Errorf("expected empty Provenance.Model, got %q", stmt.Predicate.Provenance.Model)
	}
}
