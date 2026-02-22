package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSkillRegistry(t *testing.T) {
	r := NewSkillRegistry("coder", t.TempDir())
	if r == nil {
		t.Fatal("NewSkillRegistry returned nil")
	}
}

func TestSkillRegistry_Skills(t *testing.T) {
	r := NewSkillRegistry("coder", t.TempDir())
	skills := r.Skills()
	if len(skills) == 0 {
		t.Error("expected default skills for coder")
	}
	// Should not mutate internal slice
	skills2 := r.Skills()
	if len(skills) != len(skills2) {
		t.Error("Skills() should return copy")
	}
}

func TestSkillRegistry_HasSkill(t *testing.T) {
	r := NewSkillRegistry("coder", t.TempDir())
	if !r.HasSkill("code-generation") {
		t.Error("coder should have code-generation")
	}
	if r.HasSkill("nonexistent-skill-xyz") {
		t.Error("should not have nonexistent skill")
	}
	if !r.HasSkill("CODE-GENERATION") {
		t.Error("HasSkill should be case-insensitive")
	}
}

func TestSkillRegistry_LoadFromFile(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `roles:
  coder:
    skills: [custom-skill]
`
	if err := os.WriteFile(filepath.Join(specsDir, "agent-skills.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewSkillRegistry("coder", dir)
	if !r.HasSkill("custom-skill") {
		t.Error("expected custom-skill from file")
	}
}
