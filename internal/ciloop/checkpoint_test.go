package ciloop_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/ciloop"
	"sdp_dev/internal/orchestrate"
)

func TestLoadCheckpoint(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"schema": "1.0",
		"feature_id": "F014",
		"branch": "feature/F014-ci-loop-cli",
		"pr_number": 42,
		"pr_url": "https://github.com/org/repo/pull/42",
		"phase": "build"
	}`
	if err := os.WriteFile(filepath.Join(dir, "F014.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cp, err := ciloop.LoadCheckpoint(dir, "F014")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cp.FeatureID != "F014" {
		t.Errorf("expected feature_id F014, got %q", cp.FeatureID)
	}
	if cp.PRNumber == nil || *cp.PRNumber != 42 {
		t.Errorf("expected pr_number 42, got %v", cp.PRNumber)
	}
	if cp.Branch != "feature/F014-ci-loop-cli" {
		t.Errorf("expected branch feature/F014-ci-loop-cli, got %q", cp.Branch)
	}
}

func TestLoadCheckpointNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ciloop.LoadCheckpoint(dir, "F999")
	if err == nil {
		t.Fatal("expected error for missing checkpoint, got nil")
	}
}

func TestLoadCheckpointPathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	_, err := ciloop.LoadCheckpoint(dir, "../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal featureID, got nil")
	}
}

func TestSaveCheckpointPathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	cp := &ciloop.Checkpoint{FeatureID: "../../../etc/passwd"}
	err := ciloop.SaveCheckpoint(dir, cp)
	if err == nil {
		t.Fatal("expected error for path traversal featureID in save, got nil")
	}
}

func TestLoadCheckpointInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "F014.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ciloop.LoadCheckpoint(dir, "F014")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestSaveCheckpoint(t *testing.T) {
	dir := t.TempDir()
	prNum := 42
	cp := &ciloop.Checkpoint{
		Schema:    "orchestrate.v1",
		FeatureID: "F014",
		Branch:    "feature/F014-ci-loop-cli",
		PRNumber:  &prNum,
		PRURL:     "https://github.com/org/repo/pull/42",
		Phase:     "build",
	}
	if err := ciloop.SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Read back and verify.
	loaded, err := ciloop.LoadCheckpoint(dir, "F014")
	if err != nil {
		t.Fatalf("load after save: %v", err)
	}
	if loaded.Phase != "build" {
		t.Errorf("expected phase=build (saved as given), got %q", loaded.Phase)
	}
	if loaded.UpdatedAt == "" {
		t.Error("expected updated_at to be set")
	}
}

// TestSaveCheckpointPreservesOrchestrateFields verifies ciloop save does not drop Workstreams, Review, CreatedAt.
func TestSaveCheckpointPreservesOrchestrateFields(t *testing.T) {
	dir := t.TempDir()
	// Write orchestrate-style checkpoint with workstreams, review, created_at
	ocp := &orchestrate.Checkpoint{
		Schema:     "orchestrate.v1",
		FeatureID:  "F053",
		Branch:     "feature/F053-x",
		Phase:      "build",
		CreatedAt:  "2026-02-25T10:00:00Z",
		Workstreams: []orchestrate.WSStatus{{ID: "00-053-01", Status: "done"}, {ID: "00-053-02", Status: "pending"}},
		Review:     &orchestrate.ReviewStatus{Iteration: 1, Status: "pending"},
	}
	if err := orchestrate.SaveCheckpoint(dir, ocp); err != nil {
		t.Fatalf("orchestrate save: %v", err)
	}
	// ciloop loads, updates phase, saves
	ccp, err := ciloop.LoadCheckpoint(dir, "F053")
	if err != nil {
		t.Fatalf("ciloop load: %v", err)
	}
	ccp.Phase = "ci"
	if err := ciloop.SaveCheckpoint(dir, ccp); err != nil {
		t.Fatalf("ciloop save: %v", err)
	}
	// Verify orchestrate fields preserved
	loaded, err := orchestrate.LoadCheckpoint(dir, "F053")
	if err != nil {
		t.Fatalf("orchestrate load after ciloop save: %v", err)
	}
	if loaded.Phase != "ci" {
		t.Errorf("phase should be ci (ciloop update), got %q", loaded.Phase)
	}
	if loaded.CreatedAt != "2026-02-25T10:00:00Z" {
		t.Errorf("created_at lost: got %q", loaded.CreatedAt)
	}
	if len(loaded.Workstreams) != 2 {
		t.Errorf("workstreams lost: got %d", len(loaded.Workstreams))
	}
	if loaded.Review == nil || loaded.Review.Status != "pending" {
		t.Errorf("review lost: %+v", loaded.Review)
	}
}

// TestSaveCheckpointNewFile creates checkpoint when none exists (no merge).
func TestSaveCheckpointNewFile(t *testing.T) {
	dir := t.TempDir()
	prNum := 7
	cp := &ciloop.Checkpoint{
		Schema:    "orchestrate.v1",
		FeatureID: "F099",
		Branch:    "feature/F099",
		PRNumber:  &prNum,
		Phase:     "pr",
	}
	if err := ciloop.SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("save new: %v", err)
	}
	// Should have only ciloop fields
	var raw map[string]any
	data, _ := os.ReadFile(filepath.Join(dir, "F099.json"))
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["workstreams"]; ok {
		t.Error("new file should not have workstreams")
	}
}
