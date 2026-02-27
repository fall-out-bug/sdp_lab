package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAutoAttestCLI_Success runs auto-attest in repo. Run with: go test -run TestAutoAttestCLI_Success -count=1
func TestAutoAttestCLI_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}
	bin := filepath.Join(t.TempDir(), "auto-attest")
	if err := exec.Command("go", "build", "-o", bin, ".").Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}
	wd, _ := os.Getwd()
	repoRoot := wd
	for repoRoot != "" && repoRoot != "/" {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		repoRoot = filepath.Dir(repoRoot)
	}
	if repoRoot == "" || repoRoot == "/" {
		t.Skip("repo root not found")
	}
	cmd := exec.Command(bin, "-base-branch", "master", "-pr-number", "1", "-output", filepath.Join(t.TempDir(), "attest.json"))
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("auto-attest failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "attestation") {
		t.Errorf("expected attestation message: %s", out)
	}
}

func TestAutoAttestCLI_Builds(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "auto-attest")
	if err := exec.Command("go", "build", "-o", bin, ".").Run(); err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Errorf("binary should exist: %v", err)
	}
}
