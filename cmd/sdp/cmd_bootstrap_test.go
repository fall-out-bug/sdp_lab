package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildTestBinaryForBootstrap(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "sdp-bootstrap-test")
	cmd := exec.Command("go", "build", "-tags", "sqlite_fts5", "-o", binPath, ".")
	cmd.Dir = filepath.Join("..", "..", "cmd", "sdp")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback: build from current directory.
		cmd2 := exec.Command("go", "build", "-tags", "sqlite_fts5", "-o", binPath, ".")
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			t.Fatalf("failed to build test binary: %v\n%s\nfallback: %v\n%s", err, out, err2, out2)
		}
	}
	return binPath
}

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sdpPath := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdpPath, 0o755); err != nil {
		t.Fatalf("failed to create .sdp dir: %v", err)
	}
	scoutJSON := `{"primary_language":"Go","build_system":"go","languages":{"Go":100},"has_tests":true,"test_ratio":0.5,"total_files":10}`
	if err := os.WriteFile(filepath.Join(sdpPath, "scout.json"), []byte(scoutJSON), 0o644); err != nil {
		t.Fatalf("failed to write scout.json: %v", err)
	}
	return dir
}

// TestBootstrapHelp_ShowsCIADutomationFlags verifies that --yes and --auto-curate
// appear in the bootstrap usage/help output with CI automation documentation.
func TestBootstrapHelp_ShowsCIAutomationFlags(t *testing.T) {
	binPath := buildTestBinary(t)
	// Run with no args to get usage.
	cmd := exec.Command(binPath, "bootstrap")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run() // exits with 2, that's expected
	output := out.String()

	if !strings.Contains(output, "--yes") {
		t.Errorf("expected --yes flag in bootstrap usage, got: %s", output)
	}
	if !strings.Contains(output, "--auto-curate") {
		t.Errorf("expected --auto-curate flag in bootstrap usage, got: %s", output)
	}
	if !strings.Contains(output, "CI") {
		t.Errorf("expected CI mention in bootstrap usage, got: %s", output)
	}
}

// TestBootstrapDefault_UseDraftTrue verifies that the default invocation
// (no --yes or --auto-curate) produces DRAFT-prefixed files.
func TestBootstrapDefault_UseDraftTrue(t *testing.T) {
	binPath := buildTestBinary(t)
	repoDir := setupTestRepo(t)

	cmd := exec.Command(binPath, "bootstrap", "--no-verify", repoDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Fatalf("bootstrap failed: %v\n%s", err, out.String())
	}

	// DRAFT-CLAUDE.md should exist (default UseDraft=true).
	draftPath := filepath.Join(repoDir, "DRAFT-CLAUDE.md")
	if _, err := os.Stat(draftPath); os.IsNotExist(err) {
		t.Errorf("expected DRAFT-CLAUDE.md to exist with default invocation, got: %s", out.String())
	}

	// Non-DRAFT CLAUDE.md should NOT exist.
	plainPath := filepath.Join(repoDir, "CLAUDE.md")
	if _, err := os.Stat(plainPath); err == nil {
		t.Error("expected plain CLAUDE.md to NOT exist with default invocation")
	}
}

func TestBootstrapDryRunBrownfieldDoesNotWriteDraftDelta(t *testing.T) {
	binPath := buildTestBinary(t)
	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module example.com/app\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cmd := exec.Command(binPath, "bootstrap", "--dry-run", "--mode", "brownfield", repoDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("bootstrap dry-run brownfield failed: %v\n%s", err, out.String())
	}

	if !strings.Contains(out.String(), "[plan]") {
		t.Fatalf("expected dry-run plan output, got:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(repoDir, "DRAFT-bootstrap-delta.json")); err == nil {
		t.Fatal("dry-run brownfield wrote DRAFT-bootstrap-delta.json")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat DRAFT-bootstrap-delta.json: %v", err)
	}
}

// TestBootstrapYesFlag_BypassesDraft verifies that --yes flag produces
// clean files without DRAFT prefix.
func TestBootstrapYesFlag_BypassesDraft(t *testing.T) {
	binPath := buildTestBinary(t)
	repoDir := setupTestRepo(t)

	cmd := exec.Command(binPath, "bootstrap", "--yes", "--no-verify", repoDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Fatalf("bootstrap --yes failed: %v\n%s", err, out.String())
	}

	// Plain CLAUDE.md should exist (--yes bypasses DRAFT).
	plainPath := filepath.Join(repoDir, "CLAUDE.md")
	if _, err := os.Stat(plainPath); os.IsNotExist(err) {
		t.Errorf("expected plain CLAUDE.md with --yes, got: %s", out.String())
	}

	// DRAFT-CLAUDE.md should NOT exist.
	draftPath := filepath.Join(repoDir, "DRAFT-CLAUDE.md")
	if _, err := os.Stat(draftPath); err == nil {
		t.Error("expected DRAFT-CLAUDE.md to NOT exist with --yes flag")
	}
}

// TestBootstrapAutoCurateFlag_BypassesDraft verifies that --auto-curate flag
// produces clean files without DRAFT prefix.
func TestBootstrapAutoCurateFlag_BypassesDraft(t *testing.T) {
	binPath := buildTestBinary(t)
	repoDir := setupTestRepo(t)

	cmd := exec.Command(binPath, "bootstrap", "--auto-curate", "--no-verify", repoDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Fatalf("bootstrap --auto-curate failed: %v\n%s", err, out.String())
	}

	// Plain CLAUDE.md should exist (--auto-curate bypasses DRAFT).
	plainPath := filepath.Join(repoDir, "CLAUDE.md")
	if _, err := os.Stat(plainPath); os.IsNotExist(err) {
		t.Errorf("expected plain CLAUDE.md with --auto-curate, got: %s", out.String())
	}

	// DRAFT-CLAUDE.md should NOT exist.
	draftPath := filepath.Join(repoDir, "DRAFT-CLAUDE.md")
	if _, err := os.Stat(draftPath); err == nil {
		t.Error("expected DRAFT-CLAUDE.md to NOT exist with --auto-curate flag")
	}
}

// TestBootstrapAutoCurate_NoDraftHeader verifies that files generated with
// --auto-curate do NOT contain the "DRAFT:" HTML comment header in their content.
func TestBootstrapAutoCurate_NoDraftHeader(t *testing.T) {
	binPath := buildTestBinary(t)
	repoDir := setupTestRepo(t)

	cmd := exec.Command(binPath, "bootstrap", "--auto-curate", "--no-verify", repoDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("bootstrap --auto-curate failed: %v\n%s", err, out.String())
	}

	claudePath := filepath.Join(repoDir, "CLAUDE.md")
	data, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "DRAFT:") {
		t.Errorf("CLAUDE.md content should NOT contain 'DRAFT:' header, got:\n%s", content)
	}
}

// TestBootstrapAutoCurate_NoTODOMarkers verifies that files generated with
// --auto-curate do NOT contain "TODO: verify" review markers in their content.
func TestBootstrapAutoCurate_NoTODOMarkers(t *testing.T) {
	binPath := buildTestBinary(t)
	repoDir := setupTestRepo(t)

	cmd := exec.Command(binPath, "bootstrap", "--auto-curate", "--no-verify", repoDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("bootstrap --auto-curate failed: %v\n%s", err, out.String())
	}

	claudePath := filepath.Join(repoDir, "CLAUDE.md")
	data, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "TODO: verify") {
		t.Errorf("CLAUDE.md content should NOT contain 'TODO: verify' markers, got:\n%s", content)
	}
}

// TestBootstrapBothFlags_BypassesDraft verifies that --yes --auto-curate together
// also produces clean files.
func TestBootstrapBothFlags_BypassesDraft(t *testing.T) {
	binPath := buildTestBinary(t)
	repoDir := setupTestRepo(t)

	cmd := exec.Command(binPath, "bootstrap", "--yes", "--auto-curate", "--no-verify", repoDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Fatalf("bootstrap --yes --auto-curate failed: %v\n%s", err, out.String())
	}

	plainPath := filepath.Join(repoDir, "CLAUDE.md")
	if _, err := os.Stat(plainPath); os.IsNotExist(err) {
		t.Errorf("expected plain CLAUDE.md with both flags, got: %s", out.String())
	}

	draftPath := filepath.Join(repoDir, "DRAFT-CLAUDE.md")
	if _, err := os.Stat(draftPath); err == nil {
		t.Error("expected DRAFT-CLAUDE.md to NOT exist with both flags")
	}
}
