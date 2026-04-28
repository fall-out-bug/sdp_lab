package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
)

func captureStdout(fn func()) []byte {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	done := make(chan []byte)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		done <- buf.Bytes()
	}()
	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

func TestRunNextAction_Init(t *testing.T) {
	cp := &orchestrate.Checkpoint{FeatureID: "F053", Phase: orchestrate.PhaseInit}
	workstreams := []string{"00-053-01"}
	out := captureStdout(func() { runNextAction(cp, workstreams, ".", true) })
	var action orchestrate.NextAction
	if err := json.Unmarshal(out, &action); err != nil {
		t.Fatalf("output should be valid JSON: %v\n%s", err, out)
	}
	if action.Action != "init" {
		t.Errorf("action = %q, want init", action.Action)
	}
}

func TestRunNextAction_Build(t *testing.T) {
	cp := &orchestrate.Checkpoint{
		FeatureID:   "F053",
		Phase:       orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{{ID: "00-053-01", Status: "pending"}},
	}
	workstreams := []string{"00-053-01"}
	out := captureStdout(func() { runNextAction(cp, workstreams, ".", true) })
	var action orchestrate.NextAction
	if err := json.Unmarshal(out, &action); err != nil {
		t.Fatalf("output should be valid JSON: %v\n%s", err, out)
	}
	if action.Action != "build" || action.WSID != "00-053-01" {
		t.Errorf("action = %+v, want build/00-053-01", action)
	}
}

func TestRunNextAction_Done(t *testing.T) {
	cp := &orchestrate.Checkpoint{FeatureID: "F053", Phase: orchestrate.PhaseDone}
	workstreams := []string{"00-053-01"}
	out := captureStdout(func() { runNextAction(cp, workstreams, ".", true) })
	var action orchestrate.NextAction
	if err := json.Unmarshal(out, &action); err != nil {
		t.Fatalf("output should be valid JSON: %v\n%s", err, out)
	}
	if action.Action != "done" {
		t.Errorf("action = %q, want done", action.Action)
	}
}

func TestRunNextAction_HumanReadable(t *testing.T) {
	cp := &orchestrate.Checkpoint{
		FeatureID:   "F053",
		Phase:       orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{{ID: "00-053-01", Status: "pending"}},
	}
	workstreams := []string{"00-053-01"}
	out := captureStdout(func() { runNextAction(cp, workstreams, ".", false) })
	s := string(out)
	if !strings.Contains(s, "build") {
		t.Errorf("human-readable output should contain 'build', got: %s", s)
	}
	if !strings.Contains(s, "00-053-01") {
		t.Errorf("human-readable output should contain ws id, got: %s", s)
	}
	// Should NOT be valid JSON
	var action orchestrate.NextAction
	if err := json.Unmarshal(out, &action); err == nil {
		t.Error("human-readable output should not be valid JSON")
	}
}

func TestRunHydrate_BuildPhase(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wsContent := `---
ws_id: 00-053-01
feature_id: F053
---
# Test WS
## Acceptance Criteria
- [ ] AC1
`
	if err := os.WriteFile(filepath.Join(wsDir, "00-053-01.md"), []byte(wsContent), 0o644); err != nil {
		t.Fatal(err)
	}
	agentsPath := filepath.Join(dir, "AGENTS.md")
	_ = os.WriteFile(agentsPath, []byte("## Quality Gates\n\n- go build\n- go test\n"), 0o644)
	cp := &orchestrate.Checkpoint{
		FeatureID:   "F053",
		Phase:       orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{{ID: "00-053-01", Status: "pending"}},
	}
	workstreams := []string{"00-053-01"}
	out := captureStdout(func() { runHydrate(dir, "F053", "00-053-01", cp, workstreams) })
	if !bytes.Contains(out, []byte("context-packet.json")) {
		t.Errorf("expected hydrate to print context-packet message: %s", out)
	}
	packetPath := filepath.Join(dir, ".sdp", "context-packet.json")
	if _, err := os.Stat(packetPath); err != nil {
		t.Errorf("context-packet.json should exist: %v", err)
	}
}

func TestRunAdvance_InitToBuild(t *testing.T) {
	dir := t.TempDir()
	cpPath := filepath.Join(dir, ".sdp", "checkpoints")
	runsPath := filepath.Join(dir, ".sdp", "runs")
	if err := os.MkdirAll(cpPath, 0o755); err != nil {
		t.Fatal(err)
	}
	// Use repo root for git (GetChangedFiles) - fallback returns [] if not git
	projectRoot, _ := os.Getwd()
	if _, err := os.Stat(filepath.Join(projectRoot, ".git")); err != nil {
		projectRoot = dir
	}
	cp := orchestrate.CreateInitialCheckpoint("F053", "feature/F053-x", []string{"00-053-01"})
	cp.Phase = orchestrate.PhaseInit
	workstreams := []string{"00-053-01"}
	runAdvance(projectRoot, "F053", cpPath, runsPath, "", false, cp, workstreams)
	if cp.Phase != orchestrate.PhaseBuild {
		t.Errorf("phase = %q, want build", cp.Phase)
	}
	if len(cp.Workstreams) != 1 || cp.Workstreams[0].ID != "00-053-01" {
		t.Errorf("workstreams = %+v", cp.Workstreams)
	}
	// Checkpoint should be saved
	loaded, err := orchestrate.LoadCheckpoint(cpPath, "F053")
	if err != nil {
		t.Fatalf("checkpoint should be saved: %v", err)
	}
	if loaded.Phase != orchestrate.PhaseBuild {
		t.Errorf("saved phase = %q", loaded.Phase)
	}
}

func TestMainMissingFeatureExits(t *testing.T) {
	// Build and run sdp-orchestrate without --feature; expect exit 1 and stderr.
	bin := filepath.Join(t.TempDir(), "sdp-orchestrate")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit when --feature is missing")
	}
	if len(out) == 0 {
		t.Error("expected stderr output")
	}
	s := string(out)
	if !strings.Contains(s, "feature") && !strings.Contains(s, "error") {
		t.Errorf("stderr should mention feature or error, got: %s", out)
	}
}
