package bootstrap

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyCommands_AllPass(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}

	cmds := BuildCommands{
		Build: "echo build-ok",
		Test:  "echo test-ok",
		Lint:  "echo lint-ok",
	}

	results := VerifyCommands(context.Background(), cmds)
	require.Len(t, results, 3)

	for _, r := range results {
		assert.Equal(t, 0, r.ExitCode, "command %s should pass", r.Command)
		assert.False(t, r.TimedOut)
		assert.Empty(t, r.Recovery)
	}

	assert.True(t, AllPassed(results))
	assert.Empty(t, UnverifiedCommands(results))
}

func TestVerifyCommands_OneFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}

	cmds := BuildCommands{
		Build: "echo build-ok",
		Test:  "false", // exits with code 1
		Lint:  "echo lint-ok",
	}

	results := VerifyCommands(context.Background(), cmds)
	require.Len(t, results, 3)

	assert.Equal(t, 0, results[0].ExitCode)
	assert.NotEqual(t, 0, results[1].ExitCode)
	assert.Equal(t, 0, results[2].ExitCode)

	assert.False(t, AllPassed(results))
	assert.Equal(t, []string{"false"}, UnverifiedCommands(results))

	// Failed result should have recovery hints.
	assert.NotEmpty(t, results[1].Recovery)
}

func TestVerifyCommands_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}

	cmds := BuildCommands{
		Build: "sleep 60",
	}

	results := VerifyCommandsWithTimeout(context.Background(), cmds, 100*time.Millisecond)
	require.Len(t, results, 1)

	assert.True(t, results[0].TimedOut)
	assert.Equal(t, -1, results[0].ExitCode)
	assert.NotEmpty(t, results[0].Recovery)

	// Recovery should mention timeout.
	found := false
	for _, hint := range results[0].Recovery {
		if hint == "Skip: re-run bootstrap with --no-verify to skip verification" {
			found = true
		}
	}
	assert.True(t, found, "recovery should mention --no-verify")
}

func TestVerifyCommands_EmptyCommandsSkipped(t *testing.T) {
	cmds := BuildCommands{
		Build: "echo ok",
		Test:  "",
		Lint:  "",
	}

	results := VerifyCommands(context.Background(), cmds)
	assert.Len(t, results, 1)
	assert.Equal(t, "echo ok", results[0].Command)
}

func TestVerifyCommands_AllEmpty(t *testing.T) {
	cmds := BuildCommands{}
	results := VerifyCommands(context.Background(), cmds)
	assert.Empty(t, results)
}

func TestVerifyCommands_CommandNotFound(t *testing.T) {
	cmds := BuildCommands{
		Build: "nonexistent_command_xyz_12345",
	}

	results := VerifyCommands(context.Background(), cmds)
	require.Len(t, results, 1)

	assert.NotEqual(t, 0, results[0].ExitCode)
	assert.NotEmpty(t, results[0].Recovery)
}

func TestVerifyCommands_GoSpecificRecoveryHints(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}

	cmds := BuildCommands{
		Test: "go test ./nonexistent/...",
	}

	results := VerifyCommands(context.Background(), cmds)
	require.Len(t, results, 1)

	assert.NotEqual(t, 0, results[0].ExitCode)
	// Should contain Go-specific hints.
	var hasGoHint bool
	for _, hint := range results[0].Recovery {
		if hint == "Fix: run 'go build ./...' and 'go test ./...' to see errors" {
			hasGoHint = true
		}
	}
	assert.True(t, hasGoHint, "should contain Go-specific recovery hint")
}

func TestVerifyCommands_MakeSpecificRecoveryHints(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}

	cmds := BuildCommands{
		Build: "make nonexistent-target",
	}

	results := VerifyCommands(context.Background(), cmds)
	require.Len(t, results, 1)

	assert.NotEqual(t, 0, results[0].ExitCode)
	// Should contain Make-specific hints.
	var hasMakeHint bool
	for _, hint := range results[0].Recovery {
		if hint == "Fix: run 'make build && make test' to see errors" {
			hasMakeHint = true
		}
	}
	assert.True(t, hasMakeHint, "should contain Make-specific recovery hint")
}

func TestFormatVerifyResults_AllPassed(t *testing.T) {
	results := []VerifyResult{
		{Command: "echo ok", ExitCode: 0, Output: "ok"},
	}
	text := FormatVerifyResults(results)
	assert.Contains(t, text, "[ok]")
	assert.Contains(t, text, "echo ok")
}

func TestFormatVerifyResults_Failure(t *testing.T) {
	results := []VerifyResult{
		{Command: "false", ExitCode: 1, Output: "error", Recovery: []string{"Fix: run it again"}},
	}
	text := FormatVerifyResults(results)
	assert.Contains(t, text, "[fail]")
	assert.Contains(t, text, "Recovery:")
	assert.Contains(t, text, "Fix: run it again")
}

func TestFormatVerifyResults_Timeout(t *testing.T) {
	results := []VerifyResult{
		{Command: "sleep 60", ExitCode: -1, TimedOut: true},
	}
	text := FormatVerifyResults(results)
	assert.Contains(t, text, "[timeout]")
}

func TestFormatVerifyResults_Empty(t *testing.T) {
	text := FormatVerifyResults(nil)
	assert.Contains(t, text, "No commands to verify")
}

func TestAllPassed_Empty(t *testing.T) {
	assert.True(t, AllPassed(nil))
}

func TestUnverifiedCommands_None(t *testing.T) {
	results := []VerifyResult{
		{Command: "echo ok", ExitCode: 0},
	}
	assert.Empty(t, UnverifiedCommands(results))
}

func TestContentChanged_Identical(t *testing.T) {
	assert.False(t, ContentChanged("hello", "hello"))
}

func TestContentChanged_Different(t *testing.T) {
	assert.True(t, ContentChanged("hello", "world"))
}

func TestContentChanged_Empty(t *testing.T) {
	assert.False(t, ContentChanged("", ""))
	assert.True(t, ContentChanged("", "not empty"))
}

func TestContentChanged_LargeContent(t *testing.T) {
	// Create two large strings that differ only in the middle.
	base := make([]byte, 10000)
	for i := range base {
		base[i] = byte(i % 256)
	}

	changed := make([]byte, 10000)
	copy(changed, base)
	changed[5000] = ^changed[5000]

	assert.True(t, ContentChanged(string(base), string(changed)))
}

func TestContentHash_Consistent(t *testing.T) {
	s := "test content for hashing"
	h1 := contentHash(s)
	h2 := contentHash(s)
	assert.Equal(t, h1, h2)
}

func TestContentHash_DifferentStrings(t *testing.T) {
	h1 := contentHash("string1")
	h2 := contentHash("string2")
	assert.NotEqual(t, h1, h2)
}

func TestTruncateString_Short(t *testing.T) {
	assert.Equal(t, "hello", truncateString("hello", 100))
}

func TestTruncateString_Long(t *testing.T) {
	long := make([]byte, 3000)
	for i := range long {
		long[i] = 'a'
	}
	result := truncateString(string(long), 2048)
	assert.Len(t, result, 2048+3) // truncated + "..."
	assert.True(t, strings.HasSuffix(result, "..."), "truncated string should end with '...'")
}

func TestBuildRecoveryHints_Timeout(t *testing.T) {
	hints := buildRecoveryHints("sleep 60", "timeout")
	assert.NotEmpty(t, hints)
	assert.Contains(t, hints[0], "timed out")
}

func TestBuildRecoveryHints_Failure(t *testing.T) {
	hints := buildRecoveryHints("go test ./...", "failure")
	assert.NotEmpty(t, hints)
	// Should contain rollback and skip hints.
	var hasRollback, hasSkip bool
	for _, h := range hints {
		if h == "Rollback: 'git checkout -- CLAUDE.md AGENTS.md' to restore previous files" {
			hasRollback = true
		}
		if h == "Skip: re-run bootstrap with --no-verify to skip verification" {
			hasSkip = true
		}
	}
	assert.True(t, hasRollback, "should contain rollback hint")
	assert.True(t, hasSkip, "should contain --no-verify skip hint")
}

func TestBuildRecoveryHints_Npm(t *testing.T) {
	hints := buildRecoveryHints("npm run build", "failure")
	var hasNpmHint bool
	for _, h := range hints {
		if h == "Fix: run 'npm install' then 'npm run build' to see errors" {
			hasNpmHint = true
		}
	}
	assert.True(t, hasNpmHint)
}

func TestBuildRecoveryHints_Cargo(t *testing.T) {
	hints := buildRecoveryHints("cargo test", "failure")
	var hasCargoHint bool
	for _, h := range hints {
		if h == "Fix: run 'cargo check' and 'cargo test' to see errors" {
			hasCargoHint = true
		}
	}
	assert.True(t, hasCargoHint)
}
