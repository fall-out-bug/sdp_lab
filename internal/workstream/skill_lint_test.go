package workstream

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSkillsNoDirectory(t *testing.T) {
	root := t.TempDir()
	result, err := ValidateSkills(root)
	if err != nil {
		t.Fatalf("ValidateSkills error: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues when .agents/skills missing, got: %+v", result.Issues)
	}
}

func TestValidateSkillsValid(t *testing.T) {
	root := makeSkillProject(t, "my-skill.md", `---
name: my-skill
description: A portable skill that does something useful across harnesses; demonstrates the valid frontmatter format.
version: 1.0.0
compatibility:
  - claude-code
  - opencode
---

# My Skill

Body content.
`)

	result, err := ValidateSkills(root)
	if err != nil {
		t.Fatalf("ValidateSkills error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("expected no errors, got: %+v", result.Issues)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no warnings, got: %+v", result.Issues)
	}
}

func TestValidateSkillsMissingRequired(t *testing.T) {
	root := makeSkillProject(t, "broken.md", `---
name: broken
---

# Broken
`)

	result, err := ValidateSkills(root)
	if err != nil {
		t.Fatalf("ValidateSkills error: %v", err)
	}
	if !result.HasErrors() {
		t.Fatalf("expected errors for missing description/version, got: %+v", result.Issues)
	}

	wantErrors := map[string]bool{
		`missing required frontmatter key "description"`: false,
		`missing required frontmatter key "version"`:     false,
	}
	for _, issue := range result.Issues {
		if _, ok := wantErrors[issue.Message]; ok {
			wantErrors[issue.Message] = true
		}
	}
	for msg, found := range wantErrors {
		if !found {
			t.Errorf("expected error %q in result, got: %+v", msg, result.Issues)
		}
	}
}

func TestValidateSkillsMissingCompatibilityWarning(t *testing.T) {
	root := makeSkillProject(t, "no-compat.md", `---
name: no-compat
description: A skill without compatibility; should trigger a warning about harness portability because it is undeclared.
version: 1.0.0
---

# Body
`)

	result, err := ValidateSkills(root)
	if err != nil {
		t.Fatalf("ValidateSkills error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("expected no errors, got: %+v", result.Issues)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Severity == "warning" && strings.Contains(issue.Message, "compatibility") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected compatibility warning, got: %+v", result.Issues)
	}
}

func TestValidateSkillsNameFilenameMismatch(t *testing.T) {
	root := makeSkillProject(t, "actual-name.md", `---
name: declared-name
description: Description long enough to satisfy the 60-120 character requirement window policy for skill linting.
version: 1.0.0
compatibility:
  - claude-code
---

# Body
`)

	result, err := ValidateSkills(root)
	if err != nil {
		t.Fatalf("ValidateSkills error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Severity == "warning" && strings.Contains(issue.Message, "does not match filename") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected name/filename mismatch warning, got: %+v", result.Issues)
	}
}

func TestValidateSkillsHarnessSpecificPhrase(t *testing.T) {
	root := makeSkillProject(t, "bad-prose.md", `---
name: bad-prose
description: Description long enough to satisfy the 60-120 character requirement window policy for skill linting.
version: 1.0.0
compatibility:
  - claude-code
---

# Body

In Claude Code, use the Task tool to dispatch sub-agents.
`)

	result, err := ValidateSkills(root)
	if err != nil {
		t.Fatalf("ValidateSkills error: %v", err)
	}
	count := 0
	for _, issue := range result.Issues {
		if issue.Severity == "warning" && strings.Contains(issue.Message, "harness-specific phrase") {
			count++
		}
	}
	if count == 0 {
		t.Fatalf("expected harness-specific phrase warning, got: %+v", result.Issues)
	}
}

func TestValidateSkillsDescriptionLength(t *testing.T) {
	root := makeSkillProject(t, "short.md", `---
name: short
description: too short
version: 1.0.0
compatibility:
  - claude-code
---

# Body
`)

	result, err := ValidateSkills(root)
	if err != nil {
		t.Fatalf("ValidateSkills error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Severity == "warning" && strings.Contains(issue.Message, "description length") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected description length warning, got: %+v", result.Issues)
	}
}

func TestValidateSkillsSkipsReadme(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".agents", "skills")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Index\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := ValidateSkills(root)
	if err != nil {
		t.Fatalf("ValidateSkills error: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("README.md should be ignored, got issues: %+v", result.Issues)
	}
}

func makeSkillProject(t *testing.T, filename, content string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".agents", "skills")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}
