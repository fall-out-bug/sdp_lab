package agentloop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// BashTool tests
// ---------------------------------------------------------------------------

func TestLiveToolBash_Echo(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)

	out, err := tool.Execute(context.Background(), "tc1", json.RawMessage(`{"command":"echo hello"}`))
	require.NoError(t, err)
	assert.Equal(t, "hello\n", out)
}

func TestLiveToolBash_WorkingDir(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)

	// Verify the command runs inside workdir.
	out, err := tool.Execute(context.Background(), "tc1", json.RawMessage(`{"command":"pwd"}`))
	require.NoError(t, err)
	assert.Contains(t, out, dir)
}

func TestLiveToolBash_CombinedOutput(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)

	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"command":"echo stdout; echo stderr >&2"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "stdout")
	assert.Contains(t, out, "stderr")
}

func TestLiveToolBash_Timeout(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)

	// 1-second timeout on a sleep 5 should fail.
	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"command":"sleep 5","timeout":1}`))
	require.Error(t, err, "bash must timeout when command exceeds limit")
	assert.Contains(t, err.Error(), "timeout")
}

func TestLiveToolBash_MaxTimeout(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)

	// Timeout > 300 should be capped (we can't test the full 300s, but verify it doesn't error).
	// Run a fast command with oversized timeout.
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"command":"echo ok","timeout":999}`))
	require.NoError(t, err)
	assert.Contains(t, out, "ok")
}

func TestLiveToolBash_InvalidArgs(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)

	_, err := tool.Execute(context.Background(), "tc1", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}

func TestLiveToolBash_PassFailOutput(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)

	// Simulate go test PASS output.
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"command":"echo '--- PASS: TestFoo (0.01s)\nok  \tgithub.com/example\t0.123s\nPASS'"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "PASS")
	assert.Contains(t, out, "ok")
}

func TestBashToolSecurity(t *testing.T) {
	dangerous := []string{
		"rm -rf /",
		"rm -rf ~",
		"rm -rf $HOME",
		"chmod 777 /",
		"dd if=/dev/zero of=/dev/sda",
		"mkfs.ext4 /dev/sda1",
		":(){ :|:& };:",
		"echo foo > /dev/sda",
		"rm -rf / ", // trailing space variant
		"cd /",
		"cd /tmp",
		"cd ~",
		"cd $HOME",
	}

	for _, cmd := range dangerous {
		t.Run(cmd, func(t *testing.T) {
			err := validateBashCommand(cmd)
			assert.Error(t, err, "dangerous command must be blocked: %s", cmd)
			if err != nil {
				assert.Contains(t, err.Error(), "blocked")
			}
		})
	}
}

func TestBashToolSecurity_SafeCommands(t *testing.T) {
	safe := []string{
		"go test ./...",
		"echo hello",
		"cat file.txt",
		"ls -la",
		"rm -rf /tmp/testdir",
		"chmod 755 ./script.sh",
		"git status",
		"cd ./subdir",
		"cd subdir",
	}

	for _, cmd := range safe {
		t.Run(cmd, func(t *testing.T) {
			err := validateBashCommand(cmd)
			assert.NoError(t, err, "safe command must not be blocked: %s", cmd)
		})
	}
}

func TestLiveToolBash_EvidencePass(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)
	ea := NewEvidenceAccumulator()

	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"command":"echo 'PASS'"}`))
	require.NoError(t, err)

	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: out,
	}))

	snap := ea.Snapshot(RoleEval)
	assert.True(t, snap.Quality["test"], "bash PASS output must set quality[test]=true via EvidenceAccumulator")
}

func TestLiveToolBash_EvidenceFail(t *testing.T) {
	dir := t.TempDir()
	tool := BashTool(dir)
	ea := NewEvidenceAccumulator()

	out, _ := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"command":"echo 'FAIL'; exit 1"}`))

	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "bash",
		Output: out,
		Err:    nil, // output has FAIL but no tool error
	}))

	snap := ea.Snapshot(RoleEval)
	assert.False(t, snap.Quality["test"], "bash FAIL output must not set quality[test]")
}

// ---------------------------------------------------------------------------
// ReadFileTool tests
// ---------------------------------------------------------------------------

func TestLiveToolReadFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world"), 0o644))

	tool := ReadFileTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"path":"hello.txt"}`))
	require.NoError(t, err)
	assert.Equal(t, "hello world", out)
}

func TestLiveToolReadFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	tool := ReadFileTool(dir)

	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"path":"nonexistent.txt"}`))
	require.Error(t, err)
}

func TestLiveToolReadFile_PathEscape(t *testing.T) {
	dir := t.TempDir()
	tool := ReadFileTool(dir)

	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"path":"../../../etc/passwd"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes root")
}

func TestLiveToolReadFile_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub", "dir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "dir", "file.txt"), []byte("nested"), 0o644))

	tool := ReadFileTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"path":"sub/dir/file.txt"}`))
	require.NoError(t, err)
	assert.Equal(t, "nested", out)
}

// ---------------------------------------------------------------------------
// EditFileTool tests
// ---------------------------------------------------------------------------

func TestLiveToolEditFile(t *testing.T) {
	dir := t.TempDir()
	tool := EditFileTool(dir)

	out, err := tool.Execute(context.Background(), "tc1", json.RawMessage(`{
		"path": "newfile.go",
		"content": "package main\n"
	}`))
	require.NoError(t, err)

	// Output format compatible with extractFilePath.
	assert.Equal(t, "edited: newfile.go", out)

	// File must exist.
	data, err := os.ReadFile(filepath.Join(dir, "newfile.go"))
	require.NoError(t, err)
	assert.Equal(t, "package main\n", string(data))
}

func TestLiveToolEditFile_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	tool := EditFileTool(dir)

	out, err := tool.Execute(context.Background(), "tc1", json.RawMessage(`{
		"path": "a/b/c/deep.txt",
		"content": "deep content"
	}`))
	require.NoError(t, err)
	assert.Equal(t, "edited: a/b/c/deep.txt", out)

	data, err := os.ReadFile(filepath.Join(dir, "a", "b", "c", "deep.txt"))
	require.NoError(t, err)
	assert.Equal(t, "deep content", string(data))
}

func TestLiveToolEditFile_PathEscape(t *testing.T) {
	dir := t.TempDir()
	tool := EditFileTool(dir)

	_, err := tool.Execute(context.Background(), "tc1", json.RawMessage(`{
		"path": "../../../tmp/evil.txt",
		"content": "pwned"
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes root")
}

func TestLiveToolEditFile_EvidenceFormat(t *testing.T) {
	dir := t.TempDir()
	tool := EditFileTool(dir)
	ea := NewEvidenceAccumulator()

	out, err := tool.Execute(context.Background(), "tc1", json.RawMessage(`{
		"path": "evidence_test.go",
		"content": "package test"
	}`))
	require.NoError(t, err)

	require.NoError(t, ea.OnToolResult(ToolResult{
		ID:     "tc1",
		Name:   "edit_file",
		Output: out,
	}))

	snap := ea.Snapshot(RoleBuild)
	require.Len(t, snap.Evidence, 1)
	assert.Contains(t, snap.Evidence[0], "file_modified:")
	assert.Contains(t, snap.Evidence[0], "evidence_test.go")
}

// ---------------------------------------------------------------------------
// GlobTool tests
// ---------------------------------------------------------------------------

func TestLiveToolGlob(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.go"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.txt"), nil, 0o644))

	tool := GlobTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"*.go"}`))
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, out, "a.go")
	assert.Contains(t, out, "b.go")
	assert.NotContains(t, out, "c.txt")
}

func TestLiveToolGlob_NoMatches(t *testing.T) {
	dir := t.TempDir()
	tool := GlobTool(dir)

	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"*.xyz"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "no matches")
}

func TestLiveToolGlob_DoubleStar(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "deep", "found.go"), nil, 0o644))

	tool := GlobTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"**/*.go"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "found.go")
}

// ---------------------------------------------------------------------------
// GrepTool tests
// ---------------------------------------------------------------------------

func TestLiveToolGrep(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.go"),
		[]byte("package main\n\nfunc hello() {\n\tfmt.Println(\"hello\")\n}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "world.go"),
		[]byte("package main\n\nfunc world() {}\n"), 0o644))

	tool := GrepTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"hello"}`))
	require.NoError(t, err)

	assert.Contains(t, out, "hello.go:")
	assert.Contains(t, out, "func hello()")
}

func TestLiveToolGrep_WithLineNumber(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "code.go"),
		[]byte("line1\nline2\nfunc target() {}\nline4\n"), 0o644))

	tool := GrepTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"target"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "code.go:3:")
}

func TestLiveToolGrep_NoMatches(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "code.go"), []byte("nothing here"), 0o644))

	tool := GrepTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"nonexistent_pattern_xyz"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "no matches")
}

func TestLiveToolGrep_SkipsBinary(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.png"), []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "code.go"), []byte("hello"), 0o644))

	tool := GrepTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"hello"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "code.go:")
	assert.NotContains(t, out, "data.png")
}

func TestLiveToolGrep_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pkg", "a.go"), []byte("findme"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.go"), []byte("findme"), 0o644))

	tool := GrepTool(dir)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"findme","path":"pkg"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "pkg/a.go")
	assert.NotContains(t, out, "b.go")
}

// ---------------------------------------------------------------------------
// Bd tools (nil store)
// ---------------------------------------------------------------------------

func TestLiveToolBdSearch_NilStore(t *testing.T) {
	tool := BdSearchTool(nil)
	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"query":"test"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store is nil")
}

func TestLiveToolBdCreate_NilStore(t *testing.T) {
	tool := BdCreateTool(nil)
	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"project":"p","title":"t"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store is nil")
}

func TestLiveToolBdComment_NilStore(t *testing.T) {
	tool := BdCommentTool(nil)
	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"card_id":"X-1","comment":"note"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store is nil")
}

func TestLiveToolBdComment_EmptyArgs(t *testing.T) {
	tool := BdCommentTool(nil)
	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"card_id":"","comment":"note"}`))
	require.Error(t, err)

	_, err = tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"card_id":"X-1","comment":""}`))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// BuildLiveTools
// ---------------------------------------------------------------------------

func TestLiveToolBuildLiveTools(t *testing.T) {
	tools := BuildLiveTools(t.TempDir(), nil)
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}

	// All DefaultPhaseMap tool names except web_search and completion_signal.
	expected := []string{"bash", "read_file", "edit_file", "glob", "grep", "bd_search", "bd_create", "bd_comment"}
	for _, name := range expected {
		assert.True(t, names[name], "BuildLiveTools must include %s", name)
	}

	// web_search must NOT be included (out of scope).
	assert.False(t, names["web_search"], "web_search must not be in BuildLiveTools")
}

func TestLiveToolBuildLiveTools_Schemas(t *testing.T) {
	tools := BuildLiveTools(t.TempDir(), nil)
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.True(t, json.Valid(tool.Schema), "tool %s must have valid JSON schema", tool.Name)
		assert.NotNil(t, tool.Execute, "tool %s must have Execute function", tool.Name)
	}
}

// ---------------------------------------------------------------------------
// safePath unit tests
// ---------------------------------------------------------------------------

func TestSafePath_Valid(t *testing.T) {
	dir := t.TempDir()
	// Resolve to canonical form (macOS /var -> /private/var).
	canonical, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)

	p, err := safePath(dir, "foo/bar.txt")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(canonical, "foo", "bar.txt"), p)
}

func TestSafePath_Escape(t *testing.T) {
	dir := t.TempDir()
	_, err := safePath(dir, "../../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes root")

	// Also check single-level escape.
	_, err = safePath(dir, "../etc/passwd")
	require.Error(t, err)
}

func TestSafePath_Empty(t *testing.T) {
	dir := t.TempDir()
	_, err := safePath(dir, "")
	require.Error(t, err)
}

func TestSafePath_ExactRoot(t *testing.T) {
	dir := t.TempDir()
	canonical, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)

	p, err := safePath(dir, ".")
	require.NoError(t, err)
	assert.Equal(t, canonical, p)
}

// ---------------------------------------------------------------------------
// safePath symlink security tests
// ---------------------------------------------------------------------------

func TestSafePath_SymlinkEscape(t *testing.T) {
	// Create a temp dir structure:
	//   root/
	//     link -> /tmp (outside root)
	//     link/secret.txt should be blocked
	root := t.TempDir()

	// Create a symlink inside root pointing outside.
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("pwned"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "link")))

	_, err := safePath(root, "link/secret.txt")
	require.Error(t, err, "reading through symlink that escapes root must fail")
	assert.Contains(t, err.Error(), "escapes root")
}

func TestSafePath_SymlinkEscapeEdit(t *testing.T) {
	// EditFileTool must refuse to write through an escaping symlink.
	root := t.TempDir()

	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "link")))

	tool := EditFileTool(root)
	_, err := tool.Execute(context.Background(), "tc1", json.RawMessage(`{
		"path": "link/evil.txt",
		"content": "pwned"
	}`))
	require.Error(t, err, "writing through escaping symlink must fail")
	assert.Contains(t, err.Error(), "escapes root")
}

func TestSafePath_SymlinkInsideRoot(t *testing.T) {
	// A symlink that stays inside root is fine.
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "real"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "real", "file.txt"), []byte("ok"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")))

	p, err := safePath(root, "link/file.txt")
	require.NoError(t, err, "symlink staying inside root must be allowed")
	assert.Contains(t, p, "real")
}

func TestSafePath_NewFileInExistingDir(t *testing.T) {
	// Creating a new file (path doesn't exist yet) in an existing dir must work.
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sub"), 0o755))

	p, err := safePath(root, "sub/newfile.txt")
	require.NoError(t, err)
	assert.Contains(t, p, "sub")
}

// ---------------------------------------------------------------------------
// GlobTool security tests
// ---------------------------------------------------------------------------

func TestLiveToolGlob_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	tool := GlobTool(dir)

	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"/etc/*"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "absolute paths are not allowed")
}

func TestLiveToolGlob_DotDotPattern(t *testing.T) {
	dir := t.TempDir()
	tool := GlobTool(dir)

	_, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"../*"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'..' components")
}

func TestLiveToolGlob_SymlinkEscape(t *testing.T) {
	// Glob must not return files reached through escaping symlinks.
	root := t.TempDir()

	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "outside.go"), []byte("package p"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "link")))

	tool := GlobTool(root)
	out, err := tool.Execute(context.Background(), "tc1",
		json.RawMessage(`{"pattern":"**/*.go"}`))
	require.NoError(t, err)
	assert.NotContains(t, out, "outside.go", "glob must not return files from symlinked dirs outside root")
}
