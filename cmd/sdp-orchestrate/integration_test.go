package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_NewFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bin := filepath.Join(t.TempDir(), "sdp-orchestrate-test")
	cmd := exec.Command("go", "build", "-tags", "sqlite_fts5", "-o", bin, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	out, err := exec.Command(bin, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run --help: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "no-commit") {
		t.Error("--help should show no-commit flag")
	}
	if !strings.Contains(output, "output-dir") {
		t.Error("--help should show output-dir flag")
	}
	if !strings.Contains(output, "status") {
		t.Error("--help should show status flag")
	}
}
