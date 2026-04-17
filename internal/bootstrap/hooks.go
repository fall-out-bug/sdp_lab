package bootstrap

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/pre_commit.tmpl
var preCommitTemplateFS embed.FS

//go:embed templates/prepare_msg.tmpl
var prepareMsgTemplateFS embed.FS

//go:embed templates/pre_push.tmpl
var prePushTemplateFS embed.FS

// HookInput holds data shared across all hook templates.
type HookInput struct {
	// LintCommand is the detected lint command for the repo.
	LintCommand string
	// TestCommand is the detected test command for the repo.
	TestCommand string
	// ConventionalCommits indicates whether the repo uses conventional commits.
	ConventionalCommits bool
}

// HookResult holds the outcome of generating a single hook.
type HookResult struct {
	// Name is the hook filename (e.g., "pre-commit").
	Name string
	// Content is the rendered hook content.
	Content string
	// Valid reports whether the hook passes bash -n syntax validation.
	Valid bool
	// ValidationError is the error message from bash -n, if any.
	ValidationError string
}

// GeneratePreCommit renders the pre-commit hook from the embedded template.
func GeneratePreCommit(input *HookInput) (string, error) {
	tmplData, err := preCommitTemplateFS.ReadFile("templates/pre_commit.tmpl")
	if err != nil {
		return "", fmt.Errorf("hooks/pre-commit: reading template: %w", err)
	}
	return renderHook("pre-commit", tmplData, input)
}

// GeneratePrepareCommitMsg renders the prepare-commit-msg hook from the embedded template.
func GeneratePrepareCommitMsg(input *HookInput) (string, error) {
	tmplData, err := prepareMsgTemplateFS.ReadFile("templates/prepare_msg.tmpl")
	if err != nil {
		return "", fmt.Errorf("hooks/prepare-commit-msg: reading template: %w", err)
	}
	return renderHook("prepare-commit-msg", tmplData, input)
}

// GeneratePrePush renders the pre-push hook from the embedded template.
func GeneratePrePush(input *HookInput) (string, error) {
	tmplData, err := prePushTemplateFS.ReadFile("templates/pre_push.tmpl")
	if err != nil {
		return "", fmt.Errorf("hooks/pre-push: reading template: %w", err)
	}
	return renderHook("pre-push", tmplData, input)
}

// GenerateAllHooks generates all git hooks and validates their syntax.
// Returns a slice of HookResult for each generated hook.
func GenerateAllHooks(input *HookInput) ([]HookResult, error) {
	type hookGen struct {
		name string
		gen  func(*HookInput) (string, error)
	}
	generators := []hookGen{
		{"pre-commit", GeneratePreCommit},
		{"prepare-commit-msg", GeneratePrepareCommitMsg},
		{"pre-push", GeneratePrePush},
	}

	var results []HookResult
	for _, g := range generators {
		content, err := g.gen(input)
		if err != nil {
			return nil, fmt.Errorf("hooks: generating %s: %w", g.name, err)
		}

		valid, validationErr := ValidateHookSyntax(content)

		results = append(results, HookResult{
			Name:            g.name,
			Content:         content,
			Valid:           valid,
			ValidationError: validationErr,
		})
	}

	return results, nil
}

// GenerateHooksToDir generates all hooks and writes them to the specified directory.
// Each hook file is made executable (0o755).
func GenerateHooksToDir(input *HookInput, dirPath string) ([]HookResult, error) {
	results, err := GenerateAllHooks(input)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, fmt.Errorf("hooks: creating directory %s: %w", dirPath, err)
	}

	for i, r := range results {
		hookPath := filepath.Join(dirPath, r.Name)
		if err := os.WriteFile(hookPath, []byte(r.Content), 0o755); err != nil {
			return nil, fmt.Errorf("hooks: writing %s: %w", hookPath, err)
		}
		// Ensure executable bit.
		if err := os.Chmod(hookPath, 0o755); err != nil {
			return nil, fmt.Errorf("hooks: chmod %s: %w", hookPath, err)
		}
		results[i].Valid = true // written successfully
	}

	return results, nil
}

// ValidateHookSyntax checks that a hook's content passes bash -n syntax validation.
// Returns (true, "") if valid, or (false, errorMessage) if not.
func ValidateHookSyntax(content string) (bool, string) {
	// Check if bash is available.
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		// If bash is not available, skip validation but report the issue.
		return false, "bash not found in PATH"
	}

	cmd := exec.Command(bashPath, "-n", "-")
	cmd.Stdin = strings.NewReader(content)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, strings.TrimSpace(stderr.String())
	}

	return true, ""
}

// BuildHookInput constructs a HookInput from data sources, commands, and repo scan.
func BuildHookInput(ds *DataSourceInfo, cmds BuildCommands, repoPath string) *HookInput {
	input := &HookInput{
		LintCommand:         cmds.Lint,
		TestCommand:         cmds.Test,
		ConventionalCommits: hasConventionalCommits(repoPath),
	}

	// Provide sensible defaults if commands are empty.
	if input.LintCommand == "" {
		input.LintCommand = "echo 'No linter configured'"
	}
	if input.TestCommand == "" {
		input.TestCommand = "echo 'No test runner configured'"
	}

	return input
}

// renderHook parses and executes a hook template with the given input.
func renderHook(name string, tmplData []byte, input *HookInput) (string, error) {
	tmpl, err := template.New(name).Parse(string(tmplData))
	if err != nil {
		return "", fmt.Errorf("hooks/%s: parsing template: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, input); err != nil {
		return "", fmt.Errorf("hooks/%s: executing template: %w", name, err)
	}

	return buf.String(), nil
}
