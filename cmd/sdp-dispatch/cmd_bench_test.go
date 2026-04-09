package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/dispatch"
)

func setupTestProfiles(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	store := dispatch.NewProfileStore(dir)

	profiles := []*dispatch.CapabilityProfile{
		{
			Harness:  "claude-code",
			Provider: "zai",
			Model:    "glm-5",
			Capabilities: map[string]dispatch.CapabilityScore{
				"feature:go": {TestPassRate: 0.85, AvgDuration: 5.0, SampleCount: 3},
				"bugfix:go":  {TestPassRate: 0.90, AvgDuration: 3.0, SampleCount: 2},
			},
		},
		{
			Harness:  "codex",
			Provider: "zai",
			Model:    "glm-4.7",
			Capabilities: map[string]dispatch.CapabilityScore{
				"feature:go": {TestPassRate: 0.70, AvgDuration: 8.0, SampleCount: 5},
			},
		},
	}

	for _, p := range profiles {
		if err := store.Save(p); err != nil {
			t.Fatalf("saving test profile: %v", err)
		}
	}

	return dir
}

func TestRunBench_MissingTask(t *testing.T) {
	// bench requires --task flag
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"bench"}
	err := runBench()
	if err == nil {
		t.Fatal("expected error when --task is missing")
	}
}

func TestRunBench_WithProfiles(t *testing.T) {
	dir := setupTestProfiles(t)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"bench", "-task", "feature", "-lang", "go", "-project", dir}
	err := runBench()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBench_HarnessFilter(t *testing.T) {
	dir := setupTestProfiles(t)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"bench", "-task", "feature", "-lang", "go", "-harness", "claude-code", "-project", dir}
	err := runBench()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBench_JSON(t *testing.T) {
	dir := setupTestProfiles(t)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Capture stdout.
	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	os.Args = []string{"bench", "-task", "feature", "-lang", "go", "-json", "-project", dir}
	err := runBench()
	_ = w.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var results []dispatch.BenchResult
	if err := json.NewDecoder(r).Decode(&results); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestRunBench_NoProfiles(t *testing.T) {
	dir := t.TempDir()

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"bench", "-task", "feature", "-project", dir}
	err := runBench()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBench_NoMatchingHarness(t *testing.T) {
	dir := setupTestProfiles(t)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"bench", "-task", "feature", "-harness", "nonexistent", "-project", dir}
	err := runBench()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Helper to ensure profile directory exists for other tests.
func TestSetupTestProfiles(t *testing.T) {
	dir := setupTestProfiles(t)
	profileDir := filepath.Join(dir, ".sdp", "dispatch", "profiles")
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		t.Fatalf("reading profile dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 profile files, got %d", len(entries))
	}
}
