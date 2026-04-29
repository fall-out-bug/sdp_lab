package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInit_FreshTarget verifies that sdp init --harness=all in an empty
// TempDir creates all harness dirs, sdp.manifest.yaml, and sdp.lock.
func TestInit_FreshTarget(t *testing.T) {
	target := t.TempDir()

	code := runInit([]string{"--harness", "all", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	// All harness directories must exist.
	for _, dir := range []string{".claude", ".opencode", ".codex", ".cursor", ".pi"} {
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
	for _, dir := range []string{".opencode", ".codex", ".cursor", ".pi"} {
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
	for _, dir := range []string{".opencode", ".codex", ".cursor", ".pi"} {
		full := filepath.Join(target, dir)
		if _, err := os.Stat(full); err == nil {
			t.Errorf("auto mode: expected %s NOT to exist (only .claude was pre-existing)", dir)
		}
	}
}

// TestInit_HarnessAuto_Empty verifies that --harness=auto in a completely empty
// target installs all harnesses.
func TestInit_HarnessAuto_Empty(t *testing.T) {
	target := t.TempDir()

	code := runInit([]string{"--harness", "auto", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	// All harness directories must be installed when none existed before.
	for _, dir := range []string{".claude", ".opencode", ".codex", ".cursor", ".pi"} {
		full := filepath.Join(target, dir)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected harness dir %s to exist: %v", dir, err)
		}
	}
}

func TestInit_PiHarnessWritesSkillsAndPrompts(t *testing.T) {
	target := t.TempDir()
	const token = "PI_INIT_BODY_TOKEN_42"

	skillDir := filepath.Join(target, "prompts", "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: demo\ndescription: Demo Pi skill\n---\n# Demo\n\n"+token+"\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	commandDir := filepath.Join(target, "prompts", "commands")
	if err := os.MkdirAll(commandDir, 0o755); err != nil {
		t.Fatalf("mkdir command dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandDir, "demo.md"), []byte("---\ndescription: Demo Pi command\n---\n# Demo command\n\n"+token+"\n"), 0o644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	manifest := []byte(`version: "1.0.0"
sdp_version: "1.0.0"
harnesses:
  - pi
skills:
  - { name: demo, path: prompts/skills/demo/SKILL.md }
commands:
  - { name: demo, path: prompts/commands/demo.md }
agents: []
hooks: []
mcp_servers: []
`)
	if err := os.WriteFile(filepath.Join(target, "sdp.manifest.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	code := runInit([]string{"--harness", "pi", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	for _, rel := range []string{
		filepath.Join(".pi", "skills", "demo", "SKILL.md"),
		filepath.Join(".sdp", "generated", ".pi", "skills", "demo", "SKILL.md"),
	} {
		data, err := os.ReadFile(filepath.Join(target, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if got := string(data); !strings.Contains(got, token) {
			t.Fatalf("%s does not contain embedded body token %q; got:\n%s", rel, token, got)
		}
	}

	prompt, err := os.ReadFile(filepath.Join(target, ".pi", "prompts", "demo.md"))
	if err != nil {
		t.Fatalf("read pi prompt: %v", err)
	}
	if got := string(prompt); !strings.Contains(got, "User arguments: $ARGUMENTS") || !strings.Contains(got, token) {
		t.Fatalf("pi prompt does not preserve args and body token; got:\n%s", got)
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

// TestInit_ExistingManifestEmbedsBodies verifies downstream installs with a
// real manifest write both live harness files and .sdp/generated files with
// embedded prompt bodies.
func TestInit_ExistingManifestEmbedsBodies(t *testing.T) {
	target := t.TempDir()
	const token = "INIT_BODY_TOKEN_42"

	skillDir := filepath.Join(target, "prompts", "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("# Demo Skill\n\n"+token+"\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	manifest := []byte(`version: "1.0.0"
sdp_version: "1.0.0"
harnesses:
  - claude-code
  - opencode
  - codex
  - cursor
  - pi
skills:
  - { name: demo, path: prompts/skills/demo/SKILL.md }
commands: []
agents: []
hooks: []
mcp_servers: []
`)
	if err := os.WriteFile(filepath.Join(target, "sdp.manifest.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	code := runInit([]string{"--harness", "codex", "--target", target})
	if code != 0 {
		t.Fatalf("runInit returned %d, want 0", code)
	}

	if code := runDoctorAdapters([]string{
		"--manifest", filepath.Join(target, "sdp.manifest.yaml"),
		"--out", filepath.Join(target, ".sdp", "generated"),
	}); code != 0 {
		t.Fatalf("runDoctorAdapters returned %d, want 0 after codex-only init", code)
	}

	for _, rel := range []string{
		filepath.Join(".codex", "skills", "demo.md"),
		filepath.Join(".sdp", "generated", ".codex", "skills", "demo.md"),
	} {
		data, err := os.ReadFile(filepath.Join(target, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if got := string(data); !strings.Contains(got, token) {
			t.Fatalf("%s does not contain embedded body token %q; got:\n%s", rel, token, got)
		}
	}
}
