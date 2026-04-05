package orchestrate_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/orchestrate"
)

func TestSaveCheckpoint_WritesIntegrity(t *testing.T) {
	dir := t.TempDir()
	cp := &orchestrate.Checkpoint{
		Schema:    "orchestrate.v1",
		FeatureID: "F016",
		Branch:    "feature/F016-oneshot",
		Phase:     orchestrate.PhaseBuild,
	}
	if err := orchestrate.SaveCheckpoint(dir, cp); err != nil {
		t.Fatal(err)
	}
	if cp.Integrity == "" {
		t.Error("SaveCheckpoint should set Integrity field")
	}
	if len(cp.Integrity) < 10 {
		t.Errorf("Integrity too short: %q", cp.Integrity)
	}
}

func TestLoadCheckpoint_ValidIntegrity(t *testing.T) {
	dir := t.TempDir()
	cp := &orchestrate.Checkpoint{
		Schema:    "orchestrate.v1",
		FeatureID: "F016",
		Branch:    "feature/F016-oneshot",
		Phase:     orchestrate.PhaseBuild,
	}
	if err := orchestrate.SaveCheckpoint(dir, cp); err != nil {
		t.Fatal(err)
	}
	loaded, err := orchestrate.LoadCheckpoint(dir, "F016")
	if err != nil {
		t.Fatalf("should load valid checkpoint: %v", err)
	}
	if loaded.FeatureID != "F016" {
		t.Error("loaded checkpoint mismatch")
	}
}

func TestLoadCheckpoint_CorruptedIntegrity(t *testing.T) {
	dir := t.TempDir()
	cp := &orchestrate.Checkpoint{
		Schema:    "orchestrate.v1",
		FeatureID: "F016",
		Branch:    "feature/F016-oneshot",
		Phase:     orchestrate.PhaseBuild,
	}
	if err := orchestrate.SaveCheckpoint(dir, cp); err != nil {
		t.Fatal(err)
	}

	// Corrupt the file by modifying the phase
	path := filepath.Join(dir, "F016.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	corrupted := []byte(string(data))
	// Replace "build" with "bild!" to corrupt while keeping valid JSON
	for i := 0; i < len(corrupted)-4; i++ {
		if string(corrupted[i:i+5]) == "build" {
			copy(corrupted[i:i+5], "bild!")
			break
		}
	}
	if err := os.WriteFile(path, corrupted, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = orchestrate.LoadCheckpoint(dir, "F016")
	if err == nil {
		t.Fatal("expected error for corrupted checkpoint")
	}
	if !errors.Is(err, orchestrate.ErrCheckpointCorrupted) {
		t.Errorf("expected ErrCheckpointCorrupted, got: %v", err)
	}
}

func TestLoadCheckpoint_NoIntegrityField_StillWorks(t *testing.T) {
	// Checkpoints written by old code without integrity field should still load
	dir := t.TempDir()
	path := filepath.Join(dir, "F016.json")
	data := []byte(`{"schema":"orchestrate.v1","feature_id":"F016","branch":"feature/F016","phase":"build"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := orchestrate.LoadCheckpoint(dir, "F016")
	if err != nil {
		t.Fatalf("old checkpoint without integrity should load: %v", err)
	}
	if loaded.FeatureID != "F016" {
		t.Error("loaded checkpoint mismatch")
	}
}

func TestFormatProgress(t *testing.T) {
	p := orchestrate.ProgressInfo{Done: 2, Total: 7, WSID: "00-042-03", Phase: "building"}
	got := orchestrate.FormatProgress(p)
	want := "[3/7] building 00-042-03"
	if got != want {
		t.Errorf("FormatProgress = %q, want %q", got, want)
	}
}

func TestExitCodes(t *testing.T) {
	if orchestrate.ExitSuccess != 0 {
		t.Error("ExitSuccess should be 0")
	}
	if orchestrate.ExitFailure != 1 {
		t.Error("ExitFailure should be 1")
	}
	if orchestrate.ExitNeedsHuman != 2 {
		t.Error("ExitNeedsHuman should be 2")
	}
	if orchestrate.ExitCorrupted != 3 {
		t.Error("ExitCorrupted should be 3")
	}
}
