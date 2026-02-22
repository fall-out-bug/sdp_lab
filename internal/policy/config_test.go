package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromPath_Empty(t *testing.T) {
	cfg, err := LoadFromPath("")
	if err != nil {
		t.Fatalf("LoadFromPath empty: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for empty path")
	}
}

func TestLoadFromPath_NotFound(t *testing.T) {
	_, err := LoadFromPath("/nonexistent/policy.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadFromPath_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := `
allowlist:
  - glm-5
  - glm-4.7
  - openrouter/claude-sonnet-4.6
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config")
	}
	if len(cfg.Allowlist) != 3 {
		t.Errorf("allowlist len = %d, want 3", len(cfg.Allowlist))
	}
}

func TestAllowedModelFromConfig_AfterLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := `
allowlist:
  - glm-5
  - new-model-xyz
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromPath(path); err != nil {
		t.Fatal(err)
	}
	if !AllowedModelFromConfig("glm-5") {
		t.Error("glm-5 should be allowed")
	}
	if !AllowedModelFromConfig("new-model-xyz") {
		t.Error("new-model-xyz should be allowed")
	}
	if AllowedModelFromConfig("glm-4.7") {
		t.Error("glm-4.7 not in config allowlist, should be denied")
	}
}

func TestAllowedModel_BuiltinFallback(t *testing.T) {
	// Reset config so we use built-in
	ApplyConfig(nil)
	if !AllowedModel("glm-5") {
		t.Error("glm-5 should be allowed (built-in)")
	}
	if !AllowedModel("glm-4.7") {
		t.Error("glm-4.7 should be allowed (built-in)")
	}
	if AllowedModel("denied-model") {
		t.Error("denied-model should be denied")
	}
}

func TestRoleModel_FromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := `
allowlist: [glm-5, glm-4.7, claude-sonnet-4.6]
roles:
  coder:
    primary: glm-4.7
    fallback: claude-sonnet-4.6
    economy: glm-5
  evaluator:
    primary: claude-sonnet-4.6
    fallback: glm-5
    economy: glm-4.7
cost_optimization:
  exempt_roles: [evaluator, telemetry-analyzer]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromPath(path); err != nil {
		t.Fatal(err)
	}
	p, f, e := RoleModel("coder")
	if p != "glm-4.7" || f != "claude-sonnet-4.6" || e != "glm-5" {
		t.Errorf("coder: got primary=%q fallback=%q economy=%q", p, f, e)
	}
	if !IsExemptFromAutoDowngrade("evaluator") {
		t.Error("evaluator should be exempt")
	}
	if IsExemptFromAutoDowngrade("coder") {
		t.Error("coder should not be exempt")
	}
}

func TestResolveFallbackSequenceFromRole(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := `
allowlist: [glm-5, glm-4.7, claude-sonnet-4.6]
roles:
  coder:
    primary: glm-4.7
    fallback: claude-sonnet-4.6
    economy: glm-5
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromPath(path); err != nil {
		t.Fatal(err)
	}
	seq := ResolveFallbackSequenceFromRole("coder", "")
	if len(seq) < 3 {
		t.Fatalf("expected at least 3, got %v", seq)
	}
	if seq[0] != "glm-4.7" {
		t.Errorf("first should be primary glm-4.7, got %q", seq[0])
	}
	if seq[len(seq)-1] != "escalated" {
		t.Errorf("last should be escalated, got %q", seq[len(seq)-1])
	}
}

func TestRoleDefaultModel_NoConfig(t *testing.T) {
	ApplyConfig(nil)
	if got := RoleDefaultModel("coder"); got != "glm-4.7" {
		t.Errorf("RoleDefaultModel(coder) = %q, want glm-4.7", got)
	}
	if got := RoleDefaultModel("unknown-role"); got != "glm-5" {
		t.Errorf("RoleDefaultModel(unknown) = %q, want glm-5", got)
	}
}
