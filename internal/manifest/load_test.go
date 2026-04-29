package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/manifest"
)

func TestParse_MinimalValid(t *testing.T) {
	src := []byte(`
version: "1.0.0"
sdp_version: "0.1.0"
`)
	res, err := manifest.Parse(src, "")
	if err != nil {
		t.Fatalf("expected valid minimal manifest, got: %v", err)
	}
	if res.Manifest.Version != "1.0.0" {
		t.Errorf("version mismatch: %q", res.Manifest.Version)
	}
}

func TestParse_RejectsMissingRequiredFields(t *testing.T) {
	src := []byte(`
version: "1.0.0"
`)
	_, err := manifest.Parse(src, "")
	if err == nil {
		t.Fatal("expected error for missing sdp_version")
	}
	if !strings.Contains(err.Error(), "sdp_version") {
		t.Errorf("error should mention sdp_version: %v", err)
	}
}

func TestParse_RejectsBadVersionPattern(t *testing.T) {
	src := []byte(`
version: "v1"
sdp_version: "0.1.0"
`)
	_, err := manifest.Parse(src, "")
	if err == nil {
		t.Fatal("expected error for bad version pattern")
	}
}

func TestParse_RejectsUnknownHarness(t *testing.T) {
	src := []byte(`
version: "1.0.0"
sdp_version: "0.1.0"
harnesses: ["claude-code", "windsurf"]
`)
	_, err := manifest.Parse(src, "")
	if err == nil {
		t.Fatal("expected error for unknown harness")
	}
}

func TestParse_RejectsBadSkillName(t *testing.T) {
	src := []byte(`
version: "1.0.0"
sdp_version: "0.1.0"
skills:
  - name: "Bad_Name"
    path: "skills/x.md"
`)
	_, err := manifest.Parse(src, "")
	if err == nil {
		t.Fatal("expected error for invalid skill name")
	}
}

func TestParse_AcceptsFullManifest(t *testing.T) {
	src := []byte(`
version: "1.0.0"
sdp_version: "0.1.0"
harnesses: ["claude-code", "opencode", "codex", "cursor"]
skills:
  - name: "build"
    path: "prompts/skills/build/SKILL.md"
    version: "1.0.0"
    harnesses: ["claude-code", "opencode"]
commands:
  - name: "deliver"
    path: "prompts/commands/deliver.md"
    type: "skill"
    harnesses: ["claude-code"]
agents:
  - name: "implementer"
    role: "executor"
    system_prompt_path: ".claude/agents/implementer.md"
    harnesses: ["claude-code"]
hooks:
  - event: "session-start"
    script: "scripts/hooks/session-start.sh"
mcp_servers:
  - name: "beads"
    url: "https://beads.example/mcp"
    optional: true
`)
	res, err := manifest.Parse(src, "")
	if err != nil {
		t.Fatalf("expected valid full manifest, got: %v", err)
	}
	if len(res.Manifest.Skills) != 1 || res.Manifest.Skills[0].Name != "build" {
		t.Errorf("skill not parsed: %+v", res.Manifest.Skills)
	}
	if len(res.Manifest.Commands) != 1 || res.Manifest.Commands[0].Name != "deliver" {
		t.Errorf("command not parsed: %+v", res.Manifest.Commands)
	}
	if len(res.Manifest.Agents) != 1 {
		t.Errorf("agent not parsed: %+v", res.Manifest.Agents)
	}
	if len(res.Manifest.Hooks) != 1 {
		t.Errorf("hook not parsed: %+v", res.Manifest.Hooks)
	}
	if len(res.Manifest.MCPServers) != 1 || !res.Manifest.MCPServers[0].Optional {
		t.Errorf("mcp server not parsed: %+v", res.Manifest.MCPServers)
	}
}

func TestParse_DetectsDuplicateNames(t *testing.T) {
	src := []byte(`
version: "1.0.0"
sdp_version: "0.1.0"
skills:
  - name: "build"
    path: "a.md"
  - name: "build"
    path: "b.md"
`)
	tmp := t.TempDir()
	mustWrite(t, filepath.Join(tmp, "a.md"), "")
	mustWrite(t, filepath.Join(tmp, "b.md"), "")
	_, err := manifest.Parse(src, tmp)
	if err == nil {
		t.Fatal("expected duplicate-name error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention duplicate: %v", err)
	}
}

func TestParse_DetectsMissingPaths(t *testing.T) {
	tmp := t.TempDir()
	src := []byte(`
version: "1.0.0"
sdp_version: "0.1.0"
skills:
  - name: "build"
    path: "does/not/exist.md"
`)
	_, err := manifest.Parse(src, tmp)
	if err == nil {
		t.Fatal("expected missing-path error")
	}
	if !strings.Contains(err.Error(), "does/not/exist.md") {
		t.Errorf("error should reference missing path: %v", err)
	}
}

func TestParse_RejectsPathsOutsideRepo(t *testing.T) {
	tmp := t.TempDir()
	outside := t.TempDir()
	mustWrite(t, filepath.Join(outside, "secret.md"), "secret")
	src := []byte(`
version: "1.0.0"
sdp_version: "0.1.0"
skills:
  - name: "leak"
    path: "../outside/secret.md"
commands:
  - name: "abs"
    path: "` + filepath.ToSlash(filepath.Join(outside, "secret.md")) + `"
`)
	_, err := manifest.Parse(src, tmp)
	if err == nil {
		t.Fatal("expected outside-repo path error")
	}
	if !strings.Contains(err.Error(), "invalid paths") {
		t.Errorf("error should mention invalid paths: %v", err)
	}
}

func TestParse_AcceptsExistingPaths(t *testing.T) {
	tmp := t.TempDir()
	mustWrite(t, filepath.Join(tmp, "skills", "build.md"), "# build")
	src := []byte(`
version: "1.0.0"
sdp_version: "0.1.0"
skills:
  - name: "build"
    path: "skills/build.md"
`)
	if _, err := manifest.Parse(src, tmp); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestLoad_FromDisk(t *testing.T) {
	tmp := t.TempDir()
	mustWrite(t, filepath.Join(tmp, "skills", "x.md"), "")
	manifestPath := filepath.Join(tmp, "sdp.manifest.yaml")
	mustWrite(t, manifestPath, `version: "1.0.0"
sdp_version: "0.1.0"
skills:
  - name: "x"
    path: "skills/x.md"
`)
	res, err := manifest.Load(manifestPath, tmp)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(res.Manifest.Skills) != 1 {
		t.Errorf("skill not loaded")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
