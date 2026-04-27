package dispatch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConventions_DefaultsOnly(t *testing.T) {
	dir := t.TempDir()

	conv, err := LoadConventions(dir)
	if err != nil {
		t.Fatalf("LoadConventions() error = %v", err)
	}

	defaults := DefaultConventions()

	if conv.CommitStyle != defaults.CommitStyle {
		t.Errorf("CommitStyle = %q, want %q", conv.CommitStyle, defaults.CommitStyle)
	}
	if conv.MergeStrategy != defaults.MergeStrategy {
		t.Errorf("MergeStrategy = %q, want %q", conv.MergeStrategy, defaults.MergeStrategy)
	}
	if conv.TestRequired != defaults.TestRequired {
		t.Errorf("TestRequired = %v, want %v", conv.TestRequired, defaults.TestRequired)
	}
	if conv.LintBeforePush != defaults.LintBeforePush {
		t.Errorf("LintBeforePush = %v, want %v", conv.LintBeforePush, defaults.LintBeforePush)
	}
	if conv.MaxFileLines != defaults.MaxFileLines {
		t.Errorf("MaxFileLines = %d, want %d", conv.MaxFileLines, defaults.MaxFileLines)
	}
	if conv.GoVersion != "" {
		t.Errorf("GoVersion = %q, want empty (no go.mod)", conv.GoVersion)
	}
	if len(conv.CustomRules) != 0 {
		t.Errorf("CustomRules = %v, want empty", conv.CustomRules)
	}
}

func TestLoadConventions_FromCLAUDEMd(t *testing.T) {
	dir := t.TempDir()
	claudDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `# Project Conventions

Use conventional commits for all PRs.
Squash merge PRs when merging to main.
All tests must pass before merging.
Run lint before push.
`
	if err := os.WriteFile(filepath.Join(claudDir, "CLAUDE.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	conv, err := LoadConventions(dir)
	if err != nil {
		t.Fatalf("LoadConventions() error = %v", err)
	}

	if conv.CommitStyle != "conventional" {
		t.Errorf("CommitStyle = %q, want %q", conv.CommitStyle, "conventional")
	}
	if conv.MergeStrategy != "squash" {
		t.Errorf("MergeStrategy = %q, want %q", conv.MergeStrategy, "squash")
	}
	if !conv.TestRequired {
		t.Errorf("TestRequired = false, want true")
	}
	if !conv.LintBeforePush {
		t.Errorf("LintBeforePush = false, want true")
	}
}

func TestLoadConventions_FromGoMod(t *testing.T) {
	dir := t.TempDir()

	goModContent := `module github.com/fall-out-bug/sdp_lab

go 1.26

require (
	github.com/google/uuid v1.6.0
)
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0o644); err != nil {
		t.Fatal(err)
	}

	conv, err := LoadConventions(dir)
	if err != nil {
		t.Fatalf("LoadConventions() error = %v", err)
	}

	if conv.GoVersion != "1.26" {
		t.Errorf("GoVersion = %q, want %q", conv.GoVersion, "1.26")
	}
}

func TestLoadConventions_YamlOverridesAll(t *testing.T) {
	dir := t.TempDir()

	// Create .sdp/conventions.yaml
	sdpDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yamlContent := `commit_style: plain
merge_strategy: rebase
test_required: false
lint_before_push: false
max_file_lines: 300
go_version: "1.21"
custom_rules:
  - "Always use t.Parallel() in tests"
  - "Prefer cmp.Or over ternary patterns"
`
	if err := os.WriteFile(filepath.Join(sdpDir, "conventions.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also create CLAUDE.md — its values should NOT override YAML
	claudDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudDir, 0o755); err != nil {
		t.Fatal(err)
	}
	claudeMd := `Use conventional commits.
Squash merge PRs.
`
	if err := os.WriteFile(filepath.Join(claudDir, "CLAUDE.md"), []byte(claudeMd), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also create go.mod — its version should NOT override YAML
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.19\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	conv, err := LoadConventions(dir)
	if err != nil {
		t.Fatalf("LoadConventions() error = %v", err)
	}

	if conv.CommitStyle != "plain" {
		t.Errorf("CommitStyle = %q, want %q", conv.CommitStyle, "plain")
	}
	if conv.MergeStrategy != "rebase" {
		t.Errorf("MergeStrategy = %q, want %q", conv.MergeStrategy, "rebase")
	}
	if conv.TestRequired {
		t.Errorf("TestRequired = true, want false (YAML says false)")
	}
	if conv.LintBeforePush {
		t.Errorf("LintBeforePush = true, want false (YAML says false)")
	}
	if conv.MaxFileLines != 300 {
		t.Errorf("MaxFileLines = %d, want 300", conv.MaxFileLines)
	}
	if conv.GoVersion != "1.21" {
		t.Errorf("GoVersion = %q, want %q", conv.GoVersion, "1.21")
	}
	if len(conv.CustomRules) != 2 {
		t.Fatalf("CustomRules length = %d, want 2", len(conv.CustomRules))
	}
	if conv.CustomRules[0] != "Always use t.Parallel() in tests" {
		t.Errorf("CustomRules[0] = %q, want %q", conv.CustomRules[0], "Always use t.Parallel() in tests")
	}
	if conv.CustomRules[1] != "Prefer cmp.Or over ternary patterns" {
		t.Errorf("CustomRules[1] = %q, want %q", conv.CustomRules[1], "Prefer cmp.Or over ternary patterns")
	}
}

func TestDefaultConventions(t *testing.T) {
	conv := DefaultConventions()

	if conv.CommitStyle != "conventional" {
		t.Errorf("CommitStyle = %q, want %q", conv.CommitStyle, "conventional")
	}
	if conv.MergeStrategy != "squash" {
		t.Errorf("MergeStrategy = %q, want %q", conv.MergeStrategy, "squash")
	}
	if !conv.TestRequired {
		t.Errorf("TestRequired = false, want true")
	}
	if !conv.LintBeforePush {
		t.Errorf("LintBeforePush = false, want true")
	}
	if conv.MaxFileLines != 500 {
		t.Errorf("MaxFileLines = %d, want 500", conv.MaxFileLines)
	}
	if conv.GoVersion != "" {
		t.Errorf("GoVersion = %q, want empty", conv.GoVersion)
	}
	if conv.CustomRules != nil {
		t.Errorf("CustomRules = %v, want nil", conv.CustomRules)
	}
}

func TestFormatForPrompt(t *testing.T) {
	conv := &ConventionSet{
		CommitStyle:    "conventional",
		MergeStrategy:  "squash",
		TestRequired:   true,
		LintBeforePush: true,
		MaxFileLines:   500,
		GoVersion:      "1.26",
		CustomRules:    []string{"Always use t.Parallel()", "No global state"},
	}

	output := conv.FormatForPrompt()

	if !strings.Contains(output, "## Project Conventions") {
		t.Error("FormatForPrompt missing '## Project Conventions' header")
	}
	if !strings.Contains(output, "Commit style: conventional") {
		t.Error("FormatForPrompt missing 'Commit style: conventional'")
	}
	if !strings.Contains(output, "Merge strategy: squash") {
		t.Error("FormatForPrompt missing 'Merge strategy: squash'")
	}
	if !strings.Contains(output, "Tests required: true") {
		t.Error("FormatForPrompt missing 'Tests required: true'")
	}
	if !strings.Contains(output, "Max file lines: 500") {
		t.Error("FormatForPrompt missing 'Max file lines: 500'")
	}
	if !strings.Contains(output, "Go version: 1.26") {
		t.Error("FormatForPrompt missing 'Go version: 1.26'")
	}
	if !strings.Contains(output, "### Custom Rules") {
		t.Error("FormatForPrompt missing '### Custom Rules' header")
	}
	if !strings.Contains(output, "Always use t.Parallel()") {
		t.Error("FormatForPrompt missing custom rule 'Always use t.Parallel()'")
	}
	if !strings.Contains(output, "No global state") {
		t.Error("FormatForPrompt missing custom rule 'No global state'")
	}
}

func TestExtractGoVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "standard go.mod",
			content: "module github.com/fall-out-bug/sdp_lab\n\ngo 1.26\n",
			want:    "1.26",
		},
		{
			name:    "no go line",
			content: "module github.com/fall-out-bug/sdp_lab\n",
			want:    "",
		},
		{
			name:    "go line with extra spaces",
			content: "module github.com/fall-out-bug/sdp_lab\n\ngo    1.21\n",
			want:    "1.21",
		},
		{
			name:    "go line in require block comment",
			content: "module foo\n\ngo 1.22\n\nrequire (\n  // something\n)\n",
			want:    "1.22",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractGoVersion(tc.content)
			if got != tc.want {
				t.Errorf("extractGoVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}
