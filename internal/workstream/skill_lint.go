package workstream

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SkillLintResult is the result of scanning .agents/skills/*.md files.
type SkillLintResult struct {
	Issues []ValidationIssue `json:"issues"`
}

// HasErrors reports whether any lint issue is at error severity.
func (r SkillLintResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// ValidateSkills scans all *.md files under .agents/skills/ and returns lint
// issues per SKILL.md authoring policy (see docs/reference/skill-authoring.md,
// F127-03, F127-08).
//
// Error-level checks:
//   - missing frontmatter keys: name, description, version.
//
// Warning-level checks:
//   - missing compatibility list.
//   - name in frontmatter does not match filename (kebab-case).
//   - description outside 60-120 character window.
//   - body contains hardcoded harness-specific phrases.
//
// If the skills directory is absent, the function returns an empty result
// with no error — repos that do not declare skills are valid.
func ValidateSkills(projectRoot string) (SkillLintResult, error) {
	result := SkillLintResult{Issues: []ValidationIssue{}}
	skillsDir := filepath.Join(projectRoot, ".agents", "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, fmt.Errorf("read skills directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		// README.md is documentation, not a skill.
		if strings.EqualFold(name, "README.md") {
			continue
		}

		path := filepath.Join(skillsDir, name)
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			result.Issues = append(result.Issues, ValidationIssue{
				Severity: "error",
				File:     rel(projectRoot, path),
				Message:  fmt.Sprintf("read file: %v", err),
			})
			continue
		}
		result.Issues = append(result.Issues, lintSkillFile(projectRoot, path, name, string(contentBytes))...)
	}

	return result, nil
}

func lintSkillFile(projectRoot, path, filename, content string) []ValidationIssue {
	issues := []ValidationIssue{}
	file := rel(projectRoot, path)

	fm := parseFrontmatter(content)

	// Required fields.
	required := []string{"name", "description", "version"}
	for _, key := range required {
		if _, ok := fm[key]; !ok {
			issues = append(issues, ValidationIssue{
				Severity: "error",
				File:     file,
				Message:  fmt.Sprintf("missing required frontmatter key %q", key),
			})
		}
	}

	// Compatibility recommended.
	if !hasCompatibility(content) {
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			File:     file,
			Message:  "missing 'compatibility' frontmatter — skill portability across harnesses is not declared",
		})
	}

	// name must match filename (kebab-case, without .md).
	if nameField, ok := fm["name"]; ok {
		base := strings.TrimSuffix(filename, ".md")
		if nameField != base {
			issues = append(issues, ValidationIssue{
				Severity: "warning",
				File:     file,
				Message:  fmt.Sprintf("frontmatter name %q does not match filename base %q", nameField, base),
			})
		}
	}

	// description length window 60..120.
	if desc, ok := fm["description"]; ok {
		n := len(desc)
		if n < 60 || n > 120 {
			issues = append(issues, ValidationIssue{
				Severity: "warning",
				File:     file,
				Message:  fmt.Sprintf("description length %d outside recommended 60-120 character window", n),
			})
		}
	}

	// Harness-specific phrases in body.
	body := stripFrontmatter(content)
	for _, match := range findHarnessSpecificPhrases(body) {
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			File:     file,
			Message:  fmt.Sprintf("harness-specific phrase %q — prefer harness-neutral prose", match),
		})
	}

	return issues
}

// hasCompatibility detects the 'compatibility' key in frontmatter. Because
// compatibility is typically a YAML sequence spanning multiple lines,
// parseFrontmatter (which only returns scalars) misses it. We scan the
// frontmatter block directly for a 'compatibility:' line.
func hasCompatibility(content string) bool {
	if !strings.HasPrefix(content, "---\n") {
		return false
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return false
	}
	block := content[4 : 4+end]
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "compatibility:") {
			return true
		}
	}
	return false
}

func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---\n") {
		return content
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return content
	}
	rest := content[4+end+4:]
	return strings.TrimPrefix(rest, "\n")
}

// harnessSpecificPatterns detect phrases that lock a skill to a single harness.
// Warning-level; authors may intentionally pin a skill (then suppress via
// dedicated `compatibility: [claude-code]` declaration), but the default is
// harness-neutral prose.
var harnessSpecificPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bclaude code only\b`),
	regexp.MustCompile(`(?i)\bopencode only\b`),
	regexp.MustCompile(`(?i)\bcursor only\b`),
	regexp.MustCompile(`(?i)\bcodex only\b`),
	regexp.MustCompile(`(?i)\bin claude code[, ]`),
	regexp.MustCompile(`(?i)\bin opencode[, ]`),
	regexp.MustCompile(`(?i)\buse the task tool\b`),
}

func findHarnessSpecificPhrases(body string) []string {
	matches := []string{}
	seen := map[string]bool{}
	for _, re := range harnessSpecificPatterns {
		for _, m := range re.FindAllString(body, -1) {
			key := strings.ToLower(m)
			if seen[key] {
				continue
			}
			seen[key] = true
			matches = append(matches, m)
		}
	}
	return matches
}
