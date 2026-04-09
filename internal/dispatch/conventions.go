package dispatch

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConventionSet holds project conventions loaded from config files.
type ConventionSet struct {
	CommitStyle    string   `json:"commit_style"     yaml:"commit_style"`
	MergeStrategy  string   `json:"merge_strategy"   yaml:"merge_strategy"`
	TestRequired   bool     `json:"test_required"    yaml:"test_required"`
	LintBeforePush bool     `json:"lint_before_push" yaml:"lint_before_push"`
	MaxFileLines   int      `json:"max_file_lines"   yaml:"max_file_lines"`
	GoVersion      string   `json:"go_version"       yaml:"go_version"`
	CustomRules    []string `json:"custom_rules"     yaml:"custom_rules"`
}

// DefaultConventions returns the built-in defaults.
func DefaultConventions() *ConventionSet {
	return &ConventionSet{
		CommitStyle:    "conventional",
		MergeStrategy:  "squash",
		TestRequired:   true,
		LintBeforePush: true,
		MaxFileLines:   500,
	}
}

// LoadConventions reads conventions from the project root in priority order:
//
//  1. .sdp/conventions.yaml (if exists)
//  2. .claude/CLAUDE.md (extract relevant sections)
//  3. go.mod (extract Go version)
//  4. Fallback to sensible defaults
//
// Missing files are silently skipped.
func LoadConventions(projectRoot string) (*ConventionSet, error) {
	conv := DefaultConventions()

	loadFromYAML(projectRoot, conv)
	loadFromCLAUDEMd(projectRoot, conv)
	loadFromGoMod(projectRoot, conv)

	return conv, nil
}

// FormatForPrompt returns conventions as a string suitable for prompt injection.
func (c *ConventionSet) FormatForPrompt() string {
	var sb strings.Builder

	sb.WriteString("## Project Conventions\n\n")
	fmt.Fprintf(&sb, "- Commit style: %s\n", c.CommitStyle)
	fmt.Fprintf(&sb, "- Merge strategy: %s\n", c.MergeStrategy)
	fmt.Fprintf(&sb, "- Tests required: %t\n", c.TestRequired)
	fmt.Fprintf(&sb, "- Lint before push: %t\n", c.LintBeforePush)

	if c.MaxFileLines > 0 {
		fmt.Fprintf(&sb, "- Max file lines: %d\n", c.MaxFileLines)
	}
	if c.GoVersion != "" {
		fmt.Fprintf(&sb, "- Go version: %s\n", c.GoVersion)
	}
	if len(c.CustomRules) > 0 {
		sb.WriteString("\n### Custom Rules\n\n")
		for _, rule := range c.CustomRules {
			fmt.Fprintf(&sb, "- %s\n", rule)
		}
	}

	return sb.String()
}

// yamlConventionSet uses pointer bools to distinguish "not present in YAML"
// from "explicitly set to false". This is necessary because Go's zero value
// for bool is false, so without pointers we cannot tell the difference.
type yamlConventionSet struct {
	CommitStyle    string   `yaml:"commit_style"`
	MergeStrategy  string   `yaml:"merge_strategy"`
	TestRequired   *bool    `yaml:"test_required"`
	LintBeforePush *bool    `yaml:"lint_before_push"`
	MaxFileLines   int      `yaml:"max_file_lines"`
	GoVersion      string   `yaml:"go_version"`
	CustomRules    []string `yaml:"custom_rules"`
}

// loadFromYAML reads .sdp/conventions.yaml and overlays its values onto conv.
// Missing or unparseable files are silently skipped.
func loadFromYAML(projectRoot string, conv *ConventionSet) {
	path := filepath.Join(projectRoot, ".sdp", "conventions.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var parsed yamlConventionSet
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		slog.Warn("conventions: failed to parse YAML", "path", path, "err", err)
		return
	}

	overlayFromYAML(conv, &parsed)
}

// loadFromCLAUDEMd reads .claude/CLAUDE.md and extracts convention hints.
// Missing files are silently skipped.
func loadFromCLAUDEMd(projectRoot string, conv *ConventionSet) {
	path := filepath.Join(projectRoot, ".claude", "CLAUDE.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := strings.ToLower(string(data))
	extractFromMarkdown(content, conv)
}

// loadFromGoMod reads go.mod and extracts the Go version.
// Missing files are silently skipped.
func loadFromGoMod(projectRoot string, conv *ConventionSet) {
	if conv.GoVersion != "" {
		return
	}

	path := filepath.Join(projectRoot, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	version := extractGoVersion(string(data))
	if version != "" {
		conv.GoVersion = version
	}
}

// overlayFromYAML copies explicitly-set fields from the parsed YAML onto conv.
// Pointer bools allow distinguishing "not in YAML" (nil) from "set to false".
func overlayFromYAML(dst *ConventionSet, src *yamlConventionSet) {
	if src.CommitStyle != "" {
		dst.CommitStyle = src.CommitStyle
	}
	if src.MergeStrategy != "" {
		dst.MergeStrategy = src.MergeStrategy
	}
	if src.TestRequired != nil {
		dst.TestRequired = *src.TestRequired
	}
	if src.LintBeforePush != nil {
		dst.LintBeforePush = *src.LintBeforePush
	}
	if src.MaxFileLines > 0 {
		dst.MaxFileLines = src.MaxFileLines
	}
	if src.GoVersion != "" {
		dst.GoVersion = src.GoVersion
	}
	if len(src.CustomRules) > 0 {
		dst.CustomRules = src.CustomRules
	}
}

// extractFromMarkdown scans lowercase markdown content for convention hints
// and updates conv accordingly. It only sets fields that are still at their
// zero/default state to respect YAML priority.
func extractFromMarkdown(content string, conv *ConventionSet) {
	if conv.CommitStyle == "" {
		if strings.Contains(content, "conventional commit") ||
			strings.Contains(content, "conventional commits") ||
			strings.Contains(content, "conventional-commit") {
			conv.CommitStyle = "conventional"
		}
	}

	if conv.MergeStrategy == "" {
		switch {
		case strings.Contains(content, "squash merge"):
			conv.MergeStrategy = "squash"
		case strings.Contains(content, "rebase and merge") || strings.Contains(content, "rebase merge"):
			conv.MergeStrategy = "rebase"
		case strings.Contains(content, "merge commit") || strings.Contains(content, "create a merge commit"):
			conv.MergeStrategy = "merge"
		}
	}

	if strings.Contains(content, "tests must pass") ||
		strings.Contains(content, "all tests must pass") ||
		strings.Contains(content, "test required") {
		conv.TestRequired = true
	}

	if strings.Contains(content, "lint before") ||
		strings.Contains(content, "run lint before") {
		conv.LintBeforePush = true
	}
}

// extractGoVersion parses a go.mod file's content and returns the Go version
// string (e.g. "1.26"). Returns empty string if not found.
func extractGoVersion(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go "))
		}
	}
	return ""
}
