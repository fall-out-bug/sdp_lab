//go:build sdp_experimental

package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
)

func TestRunCompare_WithProfiles(t *testing.T) {
	dir := setupTestProfiles(t)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"compare", "-task", "feature", "-lang", "go", "-project", dir}
	err := runCompare()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCompare_DefaultTaskLang(t *testing.T) {
	dir := setupTestProfiles(t)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"compare", "-project", dir}
	err := runCompare()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCompare_JSON(t *testing.T) {
	dir := setupTestProfiles(t)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	os.Args = []string{"compare", "-task", "feature", "-lang", "go", "-json", "-project", dir}
	err := runCompare()
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

	// First result should have higher or equal score (ranked).
	score0 := dispatch.BenchScore(results[0])
	score1 := dispatch.BenchScore(results[1])
	if score0 < score1 {
		t.Fatalf("expected ranked order, got score0=%f < score1=%f", score0, score1)
	}
}

func TestRunCompare_NoProfiles(t *testing.T) {
	dir := t.TempDir()

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"compare", "-project", dir}
	err := runCompare()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
