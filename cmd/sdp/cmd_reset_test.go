package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResetDryRun(t *testing.T) {
	tmpDir, cleanup := setupResetEnv(t)
	defer cleanup()
	cpPath := createCheckpoint(t, tmpDir, "F999")

	var stdout, stderr bytes.Buffer
	err := resetCheckpoint("F999", true, false, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
	if !strings.Contains(stdout.String(), "[dry-run]") {
		t.Error("dry-run should output dry-run marker")
	}
	if _, err := os.Stat(cpPath); os.IsNotExist(err) {
		t.Error("dry-run should not delete checkpoint file")
	}
}

func TestResetWithYes(t *testing.T) {
	tmpDir, cleanup := setupResetEnv(t)
	defer cleanup()
	cpPath := createCheckpoint(t, tmpDir, "F998")

	var stdout, stderr bytes.Buffer
	err := resetCheckpoint("F998", false, true, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("reset --yes should not error: %v", err)
	}
	if _, err := os.Stat(cpPath); !os.IsNotExist(err) {
		t.Error("checkpoint file should be deleted after reset --yes")
	}
}

func TestResetPathTraversalRejected(t *testing.T) {
	tmpDir, cleanup := setupResetEnv(t)
	defer cleanup()
	// Create a file outside checkpoints that the traversal would target
	outsideDir := filepath.Join(tmpDir, ".sdp")
	outsideFile := filepath.Join(outsideDir, "sensitive.json")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := resetCheckpoint("001/../../sensitive", false, true, nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("path traversal should be rejected")
	}
	if !strings.Contains(err.Error(), "invalid feature_id") {
		t.Errorf("expected validation error, got: %v", err)
	}
	// Verify the outside file is untouched
	data, err := os.ReadFile(outsideFile)
	if err != nil || string(data) != "secret" {
		t.Error("path traversal should not delete files outside checkpoints")
	}
}

func TestResetInvalidFeatureIDRejected(t *testing.T) {
	_, cleanup := setupResetEnv(t)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	err := resetCheckpoint("../../../etc/passwd", false, true, nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("invalid feature ID should be rejected")
	}
	if !strings.Contains(err.Error(), "invalid feature_id") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestResetMissingCheckpoint(t *testing.T) {
	_, cleanup := setupResetEnv(t)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	err := resetCheckpoint("F777", false, true, nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("missing checkpoint should error")
	}
	if !strings.Contains(err.Error(), "no checkpoint found") {
		t.Errorf("expected 'no checkpoint found', got: %v", err)
	}
}

func TestResetConfirmationAbort(t *testing.T) {
	tmpDir, cleanup := setupResetEnv(t)
	defer cleanup()
	cpPath := createCheckpoint(t, tmpDir, "F100")

	// User types "n" to abort
	stdin := strings.NewReader("n\n")
	var stdout, stderr bytes.Buffer
	err := resetCheckpoint("F100", false, false, stdin, &stdout, &stderr)
	if err == nil {
		t.Fatal("abort should return error")
	}
	if !strings.Contains(err.Error(), "aborted") {
		t.Errorf("expected 'aborted', got: %v", err)
	}
	// File should still exist
	if _, err := os.Stat(cpPath); os.IsNotExist(err) {
		t.Error("checkpoint should not be deleted after abort")
	}
}

func TestResetConfirmationAccept(t *testing.T) {
	tmpDir, cleanup := setupResetEnv(t)
	defer cleanup()
	cpPath := createCheckpoint(t, tmpDir, "F101")

	stdin := strings.NewReader("y\n")
	var stdout, stderr bytes.Buffer
	err := resetCheckpoint("F101", false, false, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("confirm should succeed: %v", err)
	}
	if _, err := os.Stat(cpPath); !os.IsNotExist(err) {
		t.Error("checkpoint should be deleted after confirm")
	}
}

func TestResetEmptyFeatureRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := resetCheckpoint("", false, true, nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("empty feature should be rejected")
	}
	if !strings.Contains(err.Error(), "invalid feature_id") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

// setupResetEnv creates a temp project dir with docs/workstreams/backlog.
// Returns tmpDir and a cleanup function.
func setupResetEnv(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	cpDir := filepath.Join(tmpDir, ".sdp", "checkpoints")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "dummy.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	return tmpDir, func() { os.Chdir(oldWd) }
}

// createCheckpoint creates a checkpoint file for the given feature ID.
func createCheckpoint(t *testing.T, tmpDir, featureID string) string {
	t.Helper()
	cpDir := filepath.Join(tmpDir, ".sdp", "checkpoints")
	cpPath := filepath.Join(cpDir, featureID+".json")
	content := `{"schema":"1.0","feature_id":"` + featureID + `","branch":"test","phase":"build","workstreams":[]}`
	if err := os.WriteFile(cpPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return cpPath
}
