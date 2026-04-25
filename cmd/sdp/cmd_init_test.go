package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestInit_FreshTarget verifies that sdp init --harness=all in an empty
// TempDir creates all four harness dirs, sdp.manifest.yaml, and sdp.lock.
func TestInit_FreshTarget(t *testing.T) {
	target := t.TempDir()

	code := runInit([]string{"--harness", "all", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	// All four harness directories must exist.
	for _, dir := range []string{".claude", ".opencode", ".codex", ".cursor"} {
		full := filepath.Join(target, dir)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected harness dir %s to exist: %v", dir, err)
		}
	}

	// sdp.manifest.yaml must exist.
	manifestPath := filepath.Join(target, "sdp.manifest.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("expected sdp.manifest.yaml to exist: %v", err)
	}

	// sdp.lock must exist.
	lockPath := filepath.Join(target, "sdp.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("expected sdp.lock to exist: %v", err)
	}
}

// TestInit_HarnessFilter verifies that --harness=claude-code only creates
// .claude/ and leaves other harness directories absent.
func TestInit_HarnessFilter(t *testing.T) {
	target := t.TempDir()

	code := runInit([]string{"--harness", "claude-code", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	// .claude must exist.
	if _, err := os.Stat(filepath.Join(target, ".claude")); err != nil {
		t.Errorf(".claude should exist: %v", err)
	}

	// Other harness dirs must NOT be created.
	for _, dir := range []string{".opencode", ".codex", ".cursor"} {
		full := filepath.Join(target, dir)
		if _, err := os.Stat(full); err == nil {
			t.Errorf("expected %s NOT to exist, but it does", dir)
		}
	}
}

// TestInit_AutoDetect verifies that --harness=auto with a pre-existing .claude/
// only installs claude-code and not the others.
func TestInit_AutoDetect(t *testing.T) {
	target := t.TempDir()

	// Pre-create .claude/ to simulate an existing Claude Code project.
	if err := os.MkdirAll(filepath.Join(target, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	code := runInit([]string{"--harness", "auto", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	// .claude must exist.
	if _, err := os.Stat(filepath.Join(target, ".claude")); err != nil {
		t.Errorf(".claude should exist: %v", err)
	}

	// Other harness dirs should NOT be created (auto detected only claude-code).
	for _, dir := range []string{".opencode", ".codex", ".cursor"} {
		full := filepath.Join(target, dir)
		if _, err := os.Stat(full); err == nil {
			t.Errorf("auto mode: expected %s NOT to exist (only .claude was pre-existing)", dir)
		}
	}
}

// TestInit_HarnessAuto_Empty verifies that --harness=auto in a completely empty
// target installs all 4 harnesses.
func TestInit_HarnessAuto_Empty(t *testing.T) {
	target := t.TempDir()

	code := runInit([]string{"--harness", "auto", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	// All four harness directories must be installed when none existed before.
	for _, dir := range []string{".claude", ".opencode", ".codex", ".cursor"} {
		full := filepath.Join(target, dir)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected harness dir %s to exist: %v", dir, err)
		}
	}
}

// TestInit_LockFile verifies that sdp.lock contains valid JSON with required fields.
func TestInit_LockFile(t *testing.T) {
	target := t.TempDir()

	code := runInit([]string{"--harness", "all", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	lockPath := filepath.Join(target, "sdp.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("cannot read sdp.lock: %v", err)
	}

	var lock sdpLock
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("sdp.lock is not valid JSON: %v\ncontent: %s", err, data)
	}

	if lock.SDPVersion == "" {
		t.Error("sdp.lock: sdp_version is empty")
	}
	if lock.ManifestVersion == "" {
		t.Error("sdp.lock: manifest_version is empty")
	}
	if lock.GeneratedAt == "" {
		t.Error("sdp.lock: generated_at is empty")
	}
}

// TestInit_UnknownHarness verifies that an unknown harness name returns exit code 1.
func TestInit_UnknownHarness(t *testing.T) {
	target := t.TempDir()
	code := runInit([]string{"--harness", "unknown-harness", "--target", target})
	if code == 0 {
		t.Error("expected non-zero exit code for unknown harness")
	}
}
