package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResetMissingFeature(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a dummy WS file so FindProjectRoot works
	if err := os.WriteFile(filepath.Join(wsDir, "dummy.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// Reset with no checkpoint should not panic, just exit
	// We test by calling directly and expecting os.Exit(1)
	// Since we can't capture os.Exit easily, we test dry-run path instead
}

func TestResetDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	cpDir := filepath.Join(tmpDir, ".sdp", "checkpoints")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Dummy WS file
	if err := os.WriteFile(filepath.Join(wsDir, "dummy.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create checkpoint file
	cpContent := `{"schema":"1.0","feature_id":"F999","branch":"test","phase":"build","workstreams":[{"id":"00-999-01","status":"done"}]}`
	cpPath := filepath.Join(cpDir, "F999.json")
	if err := os.WriteFile(cpPath, []byte(cpContent), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// Dry-run should NOT delete the file
	runReset([]string{"--feature", "F999", "--dry-run"})

	if _, err := os.Stat(cpPath); os.IsNotExist(err) {
		t.Error("dry-run should not delete checkpoint file")
	}
}

func TestResetWithYes(t *testing.T) {
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
	cpContent := `{"schema":"1.0","feature_id":"F998","branch":"test","phase":"build","workstreams":[]}`
	cpPath := filepath.Join(cpDir, "F998.json")
	if err := os.WriteFile(cpPath, []byte(cpContent), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	runReset([]string{"--feature", "F998", "--yes"})

	if _, err := os.Stat(cpPath); !os.IsNotExist(err) {
		t.Error("checkpoint file should be deleted after reset --yes")
	}
}

func TestResetNoFeature(t *testing.T) {
	// --feature is required, flag.ExitOnError calls os.Exit
	// We can't test os.Exit in-process, so this is a placeholder
	// that verifies the function compiles correctly
}
