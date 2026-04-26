package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePreCommit_Go(t *testing.T) {
	input := &HookInput{
		LintCommand:         "golangci-lint run",
		ConventionalCommits: false,
	}

	content, err := GeneratePreCommit(input)
	require.NoError(t, err)

	assert.Contains(t, content, "#!/bin/bash")
	assert.Contains(t, content, "golangci-lint run")
	assert.Contains(t, content, ".env*|*.pem|*.key|credentials.*|secrets.*")
}

func TestGeneratePreCommit_NpmLint(t *testing.T) {
	input := &HookInput{
		LintCommand:         "npm run lint",
		ConventionalCommits: false,
	}

	content, err := GeneratePreCommit(input)
	require.NoError(t, err)

	assert.Contains(t, content, "npm run lint")
}

func TestGeneratePreCommit_DefaultLint(t *testing.T) {
	input := &HookInput{
		LintCommand:         "echo 'No linter configured'",
		ConventionalCommits: false,
	}

	content, err := GeneratePreCommit(input)
	require.NoError(t, err)

	assert.Contains(t, content, "echo 'No linter configured'")
}

func TestGeneratePrepareCommitMsg_WithConventionalCommits(t *testing.T) {
	input := &HookInput{
		ConventionalCommits: true,
	}

	content, err := GeneratePrepareCommitMsg(input)
	require.NoError(t, err)

	assert.Contains(t, content, "#!/bin/bash")
	assert.Contains(t, content, "Conventional Commits")
	assert.Contains(t, content, "(feat|fix|chore|refactor|docs|test|ci|build|perf|style)")
}

func TestGeneratePrepareCommitMsg_WithoutConventionalCommits(t *testing.T) {
	input := &HookInput{
		ConventionalCommits: false,
	}

	content, err := GeneratePrepareCommitMsg(input)
	require.NoError(t, err)

	assert.Contains(t, content, "#!/bin/bash")
	// Should NOT contain the conventional commits validation block.
	assert.NotContains(t, content, "Conventional Commits")
}

func TestGeneratePrePush_Go(t *testing.T) {
	input := &HookInput{
		TestCommand: "go test ./...",
	}

	content, err := GeneratePrePush(input)
	require.NoError(t, err)

	assert.Contains(t, content, "#!/bin/bash")
	assert.Contains(t, content, "go test ./...")
	assert.Contains(t, content, "Tests failed. Push aborted.")
}

func TestGeneratePrePush_NpmTest(t *testing.T) {
	input := &HookInput{
		TestCommand: "npm test",
	}

	content, err := GeneratePrePush(input)
	require.NoError(t, err)

	assert.Contains(t, content, "npm test")
}

func TestGeneratePrePush_CargoTest(t *testing.T) {
	input := &HookInput{
		TestCommand: "cargo test",
	}

	content, err := GeneratePrePush(input)
	require.NoError(t, err)

	assert.Contains(t, content, "cargo test")
}

func TestGeneratePrePush_DefaultTest(t *testing.T) {
	input := &HookInput{
		TestCommand: "echo 'No test runner configured'",
	}

	content, err := GeneratePrePush(input)
	require.NoError(t, err)

	assert.Contains(t, content, "echo 'No test runner configured'")
}

func TestGenerateAllHooks(t *testing.T) {
	input := &HookInput{
		LintCommand:         "golangci-lint run",
		TestCommand:         "go test ./...",
		ConventionalCommits: true,
	}

	results, err := GenerateAllHooks(input)
	require.NoError(t, err)

	assert.Len(t, results, 3)

	names := make(map[string]bool)
	for _, r := range results {
		names[r.Name] = true
		assert.NotEmpty(t, r.Content)
	}
	assert.True(t, names["pre-commit"])
	assert.True(t, names["prepare-commit-msg"])
	assert.True(t, names["pre-push"])
}

func TestGenerateHooksToDir(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".claude", "hooks")

	input := &HookInput{
		LintCommand:         "golangci-lint run",
		TestCommand:         "go test ./...",
		ConventionalCommits: true,
	}

	results, err := GenerateHooksToDir(input, hooksDir)
	require.NoError(t, err)

	assert.Len(t, results, 3)

	// Verify files exist.
	assert.FileExists(t, filepath.Join(hooksDir, "pre-commit"))
	assert.FileExists(t, filepath.Join(hooksDir, "prepare-commit-msg"))
	assert.FileExists(t, filepath.Join(hooksDir, "pre-push"))

	// Verify files are executable.
	for _, r := range results {
		info, err := os.Stat(filepath.Join(hooksDir, r.Name))
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o111 != 0, "hook %s should be executable", r.Name)
	}
}

func TestGenerateHooksToDir_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "deep", "nested", "hooks")

	input := &HookInput{
		LintCommand: "echo skip",
		TestCommand: "echo skip",
	}

	results, err := GenerateHooksToDir(input, hooksDir)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.FileExists(t, filepath.Join(hooksDir, "pre-commit"))
}

func TestValidateHookSyntax_ValidHook(t *testing.T) {
	content := `#!/bin/bash
echo "hello"
go test ./...
`
	valid, errMsg := ValidateHookSyntax(content)
	assert.True(t, valid)
	assert.Empty(t, errMsg)
}

func TestValidateHookSyntax_InvalidHook(t *testing.T) {
	content := `#!/bin/bash
if then else
`
	valid, errMsg := ValidateHookSyntax(content)
	assert.False(t, valid)
	assert.NotEmpty(t, errMsg)
}

func TestValidateHookSyntax_EmptyHook(t *testing.T) {
	content := ""
	valid, _ := ValidateHookSyntax(content)
	assert.True(t, valid) // empty script is syntactically valid
}

func TestBuildHookInput_WithCommands(t *testing.T) {
	dir := t.TempDir()

	ds := &DataSourceInfo{
		Scout: &ScoutData{PrimaryLanguage: "Go"},
	}
	cmds := BuildCommands{
		Build: "go build ./...",
		Test:  "go test ./...",
		Lint:  "golangci-lint run",
	}

	input := BuildHookInput(ds, cmds, dir)

	assert.Equal(t, "golangci-lint run", input.LintCommand)
	assert.Equal(t, "go test ./...", input.TestCommand)
	assert.False(t, input.ConventionalCommits)
}

func TestBuildHookInput_EmptyCommands(t *testing.T) {
	dir := t.TempDir()

	ds := &DataSourceInfo{}
	cmds := BuildCommands{}

	input := BuildHookInput(ds, cmds, dir)

	// Should provide sensible defaults.
	assert.Equal(t, "echo 'No linter configured'", input.LintCommand)
	assert.Equal(t, "echo 'No test runner configured'", input.TestCommand)
}

func TestBuildHookInput_WithConventionalCommits(t *testing.T) {
	dir := t.TempDir()
	// Create a commitlint config.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".commitlintrc.json"), []byte(`{}`), 0o644))

	ds := &DataSourceInfo{
		Scout: &ScoutData{PrimaryLanguage: "Go"},
	}
	cmds := BuildCommands{
		Test: "go test ./...",
		Lint: "golangci-lint run",
	}

	input := BuildHookInput(ds, cmds, dir)

	assert.True(t, input.ConventionalCommits)
}

func TestHookGeneration_IntegratedWithPlanner(t *testing.T) {
	dir := setupRepoWithScout(t)
	// Add a Makefile for command detection.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"),
		[]byte("build:\n\tgo build ./...\ntest:\n\tgo test ./...\nlint:\n\tgolangci-lint run\n"), 0o644))

	planner := NewPlanner(BootstrapConfig{RepoPath: dir, NoVerify: true})
	report, err := planner.Execute()
	require.NoError(t, err)

	// Find the hook artifact result.
	var hookResult *ArtifactResult
	for i := range report.Artifacts {
		if report.Artifacts[i].Type == "hook" {
			hookResult = &report.Artifacts[i]
			break
		}
	}
	require.NotNil(t, hookResult, "hook artifact should be in report")
	assert.Equal(t, "ok", hookResult.Status)
	assert.Contains(t, hookResult.Message, "pre-commit")
	assert.Contains(t, hookResult.Message, "pre-push")
	assert.Contains(t, hookResult.Message, "prepare-commit-msg")

	// Verify hook files were created and are executable.
	hooksDir := filepath.Join(dir, ".claude", "hooks")
	assert.FileExists(t, filepath.Join(hooksDir, "pre-commit"))
	assert.FileExists(t, filepath.Join(hooksDir, "prepare-commit-msg"))
	assert.FileExists(t, filepath.Join(hooksDir, "pre-push"))

	// Check pre-commit has the lint command.
	preCommitContent, err := os.ReadFile(filepath.Join(hooksDir, "pre-commit"))
	require.NoError(t, err)
	assert.Contains(t, string(preCommitContent), "make lint")

	// Check pre-push has the test command.
	prePushContent, err := os.ReadFile(filepath.Join(hooksDir, "pre-push"))
	require.NoError(t, err)
	assert.Contains(t, string(prePushContent), "make test")

	// Verify executable bit.
	for _, name := range []string{"pre-commit", "prepare-commit-msg", "pre-push"} {
		info, err := os.Stat(filepath.Join(hooksDir, name))
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o111 != 0, "%s should be executable", name)
	}
}

func TestHookGeneration_DifferentLanguages(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		lintCmd  string
		testCmd  string
	}{
		{
			name:    "Go",
			lang:    "Go",
			lintCmd: "golangci-lint run",
			testCmd: "go test ./...",
		},
		{
			name:    "JavaScript",
			lang:    "JavaScript",
			lintCmd: "npm run lint",
			testCmd: "npm test",
		},
		{
			name:    "Rust",
			lang:    "Rust",
			lintCmd: "cargo clippy",
			testCmd: "cargo test",
		},
		{
			name:    "Python",
			lang:    "Python",
			lintCmd: "ruff check",
			testCmd: "pytest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := &HookInput{
				LintCommand: tc.lintCmd,
				TestCommand: tc.testCmd,
			}

			preCommit, err := GeneratePreCommit(input)
			require.NoError(t, err)
			assert.Contains(t, preCommit, tc.lintCmd)

			prePush, err := GeneratePrePush(input)
			require.NoError(t, err)
			assert.Contains(t, prePush, tc.testCmd)
		})
	}
}

func TestBeadsInit_Default(t *testing.T) {
	dir := setupRepoWithScout(t)

	// Default config (Beads=false) should NOT create beads.
	planner := NewPlanner(BootstrapConfig{RepoPath: dir, NoVerify: true})
	report, err := planner.Execute()
	require.NoError(t, err)

	var hasBeads bool
	for _, a := range report.Artifacts {
		if a.Type == "beads" {
			hasBeads = true
		}
	}
	assert.False(t, hasBeads, "beads should not be generated by default (opt-in)")

	// .beads directory should not exist.
	assert.NoDirExists(t, filepath.Join(dir, ".beads"))
}

func TestBeadsInit_WithFlag(t *testing.T) {
	dir := setupRepoWithScout(t)

	// With Beads=true, beads should be created.
	planner := NewPlanner(BootstrapConfig{RepoPath: dir, Beads: true, NoVerify: true})
	report, err := planner.Execute()
	require.NoError(t, err)

	var foundBeads bool
	for _, a := range report.Artifacts {
		if a.Type == "beads" {
			foundBeads = true
			assert.Equal(t, "ok", a.Status)
		}
	}
	assert.True(t, foundBeads, "beads should be generated when Beads flag is set")
	assert.DirExists(t, filepath.Join(dir, ".beads"))
}

func TestBeadsInit_OnlyFilter(t *testing.T) {
	dir := setupRepoWithScout(t)

	// With Only=["beads"], only beads should be created.
	planner := NewPlanner(BootstrapConfig{RepoPath: dir, Only: []string{"beads"}, NoVerify: true})
	report, err := planner.Execute()
	require.NoError(t, err)

	for _, a := range report.Artifacts {
		assert.Equal(t, "beads", a.Type, "only beads should be generated")
	}
	assert.DirExists(t, filepath.Join(dir, ".beads"))
}

func TestReportRecordsGenerated(t *testing.T) {
	dir := setupRepoWithScout(t)

	planner := NewPlanner(BootstrapConfig{RepoPath: dir, NoVerify: true})
	report, err := planner.Execute()
	require.NoError(t, err)

	// Report should have entries for each artifact type.
	artifactTypes := make(map[string]string) // type -> status
	for _, a := range report.Artifacts {
		artifactTypes[a.Type] = a.Status
	}

	assert.Equal(t, "ok", artifactTypes["claude_md"])
	assert.Equal(t, "ok", artifactTypes["agents_md"])
	assert.Equal(t, "ok", artifactTypes["policy"])
	assert.Equal(t, "ok", artifactTypes["hook"])
	// Beads is opt-in; should NOT be present by default.
	_, hasBeads := artifactTypes["beads"]
	assert.False(t, hasBeads, "beads should not be generated by default (opt-in)")

	// Notes should mention what was generated.
	var hasCreateNote bool
	for _, note := range report.Notes {
		if strings.Contains(note, "Will create") {
			hasCreateNote = true
		}
	}
	assert.True(t, hasCreateNote)
}

// --- DRAFT guard tests ---

func TestGeneratePreCommit_ContainsDraftGuard(t *testing.T) {
	input := &HookInput{
		LintCommand: "golangci-lint run",
	}

	content, err := GeneratePreCommit(input)
	require.NoError(t, err)

	assert.Contains(t, content, "DRAFT-")
	assert.Contains(t, content, "DRAFT files detected")
	assert.Contains(t, content, "Rename (remove DRAFT- prefix) before committing")
}

func TestGeneratePreCommit_DraftGuardBeforeOtherChecks(t *testing.T) {
	input := &HookInput{
		LintCommand: "golangci-lint run",
	}

	content, err := GeneratePreCommit(input)
	require.NoError(t, err)

	draftPos := strings.Index(content, "DRAFT files detected")
	lintPos := strings.Index(content, "golangci-lint run")
	sensitivePos := strings.Index(content, ".env*")

	assert.True(t, draftPos >= 0, "DRAFT guard should be present")
	assert.True(t, lintPos >= 0, "lint command should be present")
	assert.True(t, sensitivePos >= 0, "sensitive file check should be present")

	// DRAFT check must appear before lint and sensitive file checks (fail fast).
	assert.True(t, draftPos < lintPos, "DRAFT guard should run before lint command")
	assert.True(t, draftPos < sensitivePos, "DRAFT guard should run before sensitive file check")
}

func TestGeneratePreCommit_DraftGuardUsesGitDiff(t *testing.T) {
	input := &HookInput{
		LintCommand: "echo skip",
	}

	content, err := GeneratePreCommit(input)
	require.NoError(t, err)

	assert.Contains(t, content, "git diff --cached --name-only")
	assert.Contains(t, content, "^DRAFT-")
}

func TestDryRunReportRecordsSkipped(t *testing.T) {
	dir := setupRepoWithScout(t)

	planner := NewPlanner(BootstrapConfig{RepoPath: dir, DryRun: true})
	report, err := planner.DryRun()
	require.NoError(t, err)

	// All artifacts should be dry_run or skipped.
	for _, a := range report.Artifacts {
		if a.Action == "skip" {
			assert.Equal(t, "skipped", a.Status)
		} else {
			assert.Equal(t, "dry_run", a.Status)
		}
	}
}

// --- Integration test: pre-commit hook DRAFT blocking ---

// TestPreCommitHook_BlocksDraftFiles is an integration test that creates a real
// git repo, installs the generated pre-commit hook, and verifies that committing
// a DRAFT-prefixed file fails while committing a normal file succeeds.
func TestPreCommitHook_BlocksDraftFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Generate the pre-commit hook content.
	input := &HookInput{
		LintCommand: "echo skip",
	}
	hookContent, err := GeneratePreCommit(input)
	require.NoError(t, err)

	// Create a temp directory and initialize a git repo.
	repoDir := t.TempDir()

	gitInit := exec.Command("git", "init")
	gitInit.Dir = repoDir
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	// Configure git user (required for commits).
	gitConfig := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfig.Dir = repoDir
	if out, err := gitConfig.CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %v\n%s", err, out)
	}
	gitConfigName := exec.Command("git", "config", "user.name", "Test")
	gitConfigName.Dir = repoDir
	if out, err := gitConfigName.CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %v\n%s", err, out)
	}

	// Write the generated pre-commit hook into .git/hooks/pre-commit.
	hooksDir := filepath.Join(repoDir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0o755))
	hookPath := filepath.Join(hooksDir, "pre-commit")
	require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0o755))
	require.NoError(t, os.Chmod(hookPath, 0o755))

	// Sub-test: committing a DRAFT file should fail.
	t.Run("blocks_draft_file", func(t *testing.T) {
		draftFile := filepath.Join(repoDir, "DRAFT-test.md")
		require.NoError(t, os.WriteFile(draftFile, []byte("# draft content\n"), 0o644))

		gitAdd := exec.Command("git", "add", "DRAFT-test.md")
		gitAdd.Dir = repoDir
		if out, err := gitAdd.CombinedOutput(); err != nil {
			t.Fatalf("git add failed: %v\n%s", err, out)
		}

		gitCommit := exec.Command("git", "commit", "-m", "test draft")
		gitCommit.Dir = repoDir
		out, err := gitCommit.CombinedOutput()
		if err == nil {
			t.Error("expected git commit to fail for DRAFT file, but it succeeded")
		}
		if !strings.Contains(string(out), "DRAFT files detected") {
			t.Errorf("expected 'DRAFT files detected' in output, got: %s", string(out))
		}

		// Unstage the DRAFT file so the next sub-test starts with a clean index.
		gitRm := exec.Command("git", "rm", "--cached", "-f", "DRAFT-test.md")
		gitRm.Dir = repoDir
		_, _ = gitRm.CombinedOutput() // best-effort; ignore errors
		// Remove the physical file too.
		os.Remove(filepath.Join(repoDir, "DRAFT-test.md"))
	})

	// Sub-test: committing a non-DRAFT file should succeed.
	t.Run("allows_normal_file", func(t *testing.T) {
		normalFile := filepath.Join(repoDir, "normal.md")
		require.NoError(t, os.WriteFile(normalFile, []byte("# normal content\n"), 0o644))

		gitAdd := exec.Command("git", "add", "normal.md")
		gitAdd.Dir = repoDir
		if out, err := gitAdd.CombinedOutput(); err != nil {
			t.Fatalf("git add failed: %v\n%s", err, out)
		}

		gitCommit := exec.Command("git", "commit", "-m", "test normal")
		gitCommit.Dir = repoDir
		out, err := gitCommit.CombinedOutput()
		if err != nil {
			t.Errorf("expected git commit to succeed for normal file, got error: %v\n%s", err, out)
		}
	})
}
