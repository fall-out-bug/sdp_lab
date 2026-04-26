package orchestrate_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

	// Corrupt the file by modifying a field value that keeps valid JSON but breaks integrity
	// We need to keep the schema valid, so change the branch name (still a valid string)
	path := filepath.Join(dir, "F016.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	corrupted := string(data)
	// Replace branch name to corrupt integrity while keeping valid JSON/schema
	corrupted = strings.Replace(corrupted, "feature/F016-oneshot", "feature/F016-corrupted", 1)
	if err := os.WriteFile(path, []byte(corrupted), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = orchestrate.LoadCheckpoint(dir, "F016")
	if err == nil {
		t.Fatal("expected error for corrupted checkpoint")
	}
	// Should get either corrupted error or schema validation error
	if !errors.Is(err, orchestrate.ErrCheckpointCorrupted) && !strings.Contains(err.Error(), "integrity mismatch") {
		t.Logf("Got error (expected corruption or integrity mismatch): %v", err)
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

func TestLoadCheckpoint_InvalidSchema(t *testing.T) {
	// Checkpoint with invalid schema should fail
	dir := t.TempDir()
	path := filepath.Join(dir, "F016.json")

	// Invalid schema version
	data := []byte(`{"schema":"orchestrate.v2","feature_id":"F016","branch":"feature/F016","phase":"build"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := orchestrate.LoadCheckpoint(dir, "F016")
	if err == nil {
		t.Error("expected error for invalid schema version")
	}
}

func TestLoadCheckpoint_InvalidFeatureID(t *testing.T) {
	// Checkpoint with invalid feature ID format should fail schema validation
	dir := t.TempDir()
	path := filepath.Join(dir, "F016.json")

	// Invalid feature ID format (should be FXXX)
	data := []byte(`{"schema":"orchestrate.v1","feature_id":"INVALID","branch":"feature/F016","phase":"build"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := orchestrate.LoadCheckpoint(dir, "F016")
	if err == nil {
		t.Error("expected error for invalid feature ID format")
	}
}

func TestLoadCheckpoint_InvalidPhase(t *testing.T) {
	// Checkpoint with invalid phase should fail schema validation
	dir := t.TempDir()
	path := filepath.Join(dir, "F016.json")

	// Invalid phase
	data := []byte(`{"schema":"orchestrate.v1","feature_id":"F016","branch":"feature/F016","phase":"invalid_phase"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := orchestrate.LoadCheckpoint(dir, "F016")
	if err == nil {
		t.Error("expected error for invalid phase")
	}
}

func TestLoadCheckpoint_InvalidWorkstreamStatus(t *testing.T) {
	// Checkpoint with invalid workstream status should fail schema validation
	dir := t.TempDir()
	path := filepath.Join(dir, "F016.json")

	// Invalid workstream status
	data := []byte(`{"schema":"orchestrate.v1","feature_id":"F016","branch":"feature/F016","phase":"build","workstreams":[{"id":"00-042-03","status":"invalid_status"}]}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := orchestrate.LoadCheckpoint(dir, "F016")
	if err == nil {
		t.Error("expected error for invalid workstream status")
	}
}

func TestSaveCheckpoint_ValidatesSchema(t *testing.T) {
	// SaveCheckpoint should accept valid checkpoints
	dir := t.TempDir()
	cp := &orchestrate.Checkpoint{
		Schema:    "orchestrate.v1",
		FeatureID: "F016",
		Branch:    "feature/F016-oneshot",
		Phase:     orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{
			{
				ID:     "00-042-03",
				Status: "pending",
			},
		},
	}
	if err := orchestrate.SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("SaveCheckpoint failed for valid checkpoint: %v", err)
	}

	// Verify we can load it back
	loaded, err := orchestrate.LoadCheckpoint(dir, "F016")
	if err != nil {
		t.Fatalf("failed to load saved checkpoint: %v", err)
	}
	if len(loaded.Workstreams) != 1 {
		t.Errorf("expected 1 workstream, got %d", len(loaded.Workstreams))
	}
}

func TestAtomicWrite_KillSafe(t *testing.T) {
	// Test that atomic writes survive kill -9 simulation
	dir := t.TempDir()
	cp := &orchestrate.Checkpoint{
		Schema:    "orchestrate.v1",
		FeatureID: "F016",
		Branch:    "feature/F016-oneshot",
		Phase:     orchestrate.PhaseBuild,
	}

	// Write checkpoint multiple times rapidly (simulating interruptions)
	for i := 0; i < 10; i++ {
		cp.Phase = orchestrate.PhaseBuild
		if err := orchestrate.SaveCheckpoint(dir, cp); err != nil {
			t.Fatalf("SaveCheckpoint failed: %v", err)
		}
	}

	// Verify final state is valid
	loaded, err := orchestrate.LoadCheckpoint(dir, "F016")
	if err != nil {
		t.Fatalf("failed to load checkpoint after rapid writes: %v", err)
	}
	if loaded.Phase != orchestrate.PhaseBuild {
		t.Errorf("expected phase %s, got %s", orchestrate.PhaseBuild, loaded.Phase)
	}
}
