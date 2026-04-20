package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"sdp_dev/internal/harnesscfg"
)

var (
	rulesBinOnce sync.Once
	rulesBinPath string
	rulesBinErr  error
)

// buildRulesBinary builds the sdp binary once and caches the path.
func buildRulesBinary(t *testing.T) string {
	t.Helper()
	rulesBinOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "sdp-rules-*")
		if err != nil {
			rulesBinErr = err
			return
		}
		binPath := filepath.Join(tmpDir, "sdp-rules")
		cmd := exec.Command("go", "build", "-tags", "sqlite_fts5", "-o", binPath, ".")
		if out, err := cmd.CombinedOutput(); err != nil {
			rulesBinErr = err
			_ = out
			return
		}
		rulesBinPath = binPath
	})
	if rulesBinErr != nil {
		t.Fatal(rulesBinErr)
	}
	return rulesBinPath
}

// --- Rules update command tests ---

// TestRulesUsage_NoArgs verifies that running "sdp rules" with no args
// prints usage and exits with code 2.
func TestRulesUsage_NoArgs(t *testing.T) {
	bin := buildRulesBinary(t)
	out := runCmd(bin, "rules")
	if out.err == nil {
		t.Fatal("expected non-zero exit for 'sdp rules' with no args")
	}
	if !strings.Contains(out.combined(), "usage: sdp rules") {
		t.Errorf("expected usage message, got: %s", out.combined())
	}
}

// TestRulesUpdate_NonexistentPath verifies that "sdp rules update" on a
// nonexistent path returns an error.
func TestRulesUpdate_NonexistentPath(t *testing.T) {
	bin := buildRulesBinary(t)
	out := runCmd(bin, "rules", "update", "/nonexistent/path/that/does/not/exist")
	if out.err == nil {
		t.Fatal("expected non-zero exit for nonexistent path")
	}
	if !strings.Contains(out.combined(), "error") {
		t.Errorf("expected error message, got: %s", out.combined())
	}
}

// TestRulesUpdate_CreatesDraftFiles verifies that "sdp rules update" on a
// temp project produces DRAFT- prefixed output files.
func TestRulesUpdate_CreatesDraftFiles(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupRulesTestRepo(t)

	out := runCmd(bin, "rules", "update", repoDir)
	if out.err != nil {
		t.Fatalf("rules update failed: %v\n%s", out.err, out.combined())
	}

	// Verify DRAFT-CLAUDE-rules.md was created.
	draftPath := filepath.Join(repoDir, "DRAFT-CLAUDE-rules.md")
	if _, err := os.Stat(draftPath); os.IsNotExist(err) {
		t.Errorf("expected DRAFT-CLAUDE-rules.md to exist, got output:\n%s", out.combined())
	}

	// Verify it has DRAFT header.
	data, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatalf("failed to read DRAFT file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "DRAFT:") {
		t.Errorf("DRAFT file should contain DRAFT header, got:\n%s", content)
	}
}

// TestRulesUpdate_WithEvidence produces at least one rule when evidence
// contains failure entries.
func TestRulesUpdate_WithEvidence(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupRulesTestRepo(t)
	setupEvidence(t, repoDir)

	out := runCmd(bin, "rules", "update", repoDir)
	if out.err != nil {
		t.Fatalf("rules update failed: %v\n%s", out.err, out.combined())
	}

	// Read DRAFT file and check it contains at least one rule.
	draftPath := filepath.Join(repoDir, "DRAFT-CLAUDE-rules.md")
	data, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatalf("failed to read DRAFT file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "RULE-") {
		t.Errorf("expected at least one rule in DRAFT file, got:\n%s", content)
	}
}

// TestRulesUpdate_JSONFormat verifies JSON output contains expected fields.
func TestRulesUpdate_JSONFormat(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupRulesTestRepo(t)

	out := runCmd(bin, "rules", "update", "--format", "json", repoDir)
	if out.err != nil {
		t.Fatalf("rules update --format json failed: %v\n%s", out.err, out.combined())
	}

	var report map[string]any
	if err := json.Unmarshal(out.out, &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.combined())
	}
	if _, ok := report["repo"]; !ok {
		t.Error("JSON report missing 'repo' field")
	}
	if _, ok := report["adapters"]; !ok {
		t.Error("JSON report missing 'adapters' field")
	}
	if _, ok := report["sources"]; !ok {
		t.Error("JSON report missing 'sources' field")
	}
}

// TestRulesUpdate_CustomEvidenceDir verifies --source-evidence flag works.
func TestRulesUpdate_CustomEvidenceDir(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupRulesTestRepo(t)

	// Create evidence in a custom location.
	evDir := filepath.Join(t.TempDir(), "custom-evidence")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeEvidenceEntry(t, evDir, "entry-001.json", evidenceFixture{
		RunID: "run-1", Phase: "build", Verdict: "fail", Summary: "compile error",
	})

	out := runCmd(bin, "rules", "update", "--source-evidence", evDir, repoDir)
	if out.err != nil {
		t.Fatalf("rules update with custom evidence failed: %v\n%s", out.err, out.combined())
	}

	// Check the DRAFT file for rule content.
	draftPath := filepath.Join(repoDir, "DRAFT-CLAUDE-rules.md")
	data, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatalf("failed to read DRAFT file: %v", err)
	}
	if !strings.Contains(string(data), "RULE-") {
		t.Errorf("expected rule in DRAFT file with custom evidence, got:\n%s", string(data))
	}
}

// TestRulesUpdate_NoDraftAutoCommit verifies DRAFT files are never auto-committed.
func TestRulesUpdate_NoDraftAutoCommit(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupRulesTestRepo(t)

	// Initialize a git repo so we can check for uncommitted files.
	runCmd("git", "init", repoDir)
	runCmd("git", "-C", repoDir, "add", "-A")
	runCmd("git", "-C", repoDir, "commit", "-m", "initial")

	out := runCmd(bin, "rules", "update", repoDir)
	if out.err != nil {
		t.Fatalf("rules update failed: %v\n%s", out.err, out.combined())
	}

	// Verify DRAFT files are untracked (not committed).
	status := runCmd("git", "-C", repoDir, "status", "--porcelain")
	if !strings.Contains(status.combined(), "DRAFT-") {
		t.Errorf("DRAFT files should be untracked in git, got status:\n%s", status.combined())
	}
}

// --- Bootstrap --conventions integration tests ---

// TestBootstrapConventions_CreatesAdapterFiles verifies that bootstrap with
// --conventions creates adapter output files in addition to normal artifacts.
func TestBootstrapConventions_CreatesAdapterFiles(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupTestRepo(t)

	out := runCmd(bin, "bootstrap", "--no-verify", "--conventions", repoDir)
	if out.err != nil {
		t.Fatalf("bootstrap --conventions failed: %v\n%s", out.err, out.combined())
	}

	// Should still produce standard DRAFT-CLAUDE-rules.md.
	draftClaude := filepath.Join(repoDir, "DRAFT-CLAUDE-rules.md")
	if _, err := os.Stat(draftClaude); os.IsNotExist(err) {
		t.Errorf("expected DRAFT-CLAUDE-rules.md, got output:\n%s", out.combined())
	}

	// Should also produce adapter output (DRAFT-CLAUDE-rules.md).
	draftRules := filepath.Join(repoDir, "DRAFT-CLAUDE-rules.md")
	if _, err := os.Stat(draftRules); os.IsNotExist(err) {
		t.Errorf("expected DRAFT-CLAUDE-rules.md with --conventions, got output:\n%s", out.combined())
	}
}

// TestBootstrapWithoutConventions_Unchanged verifies that bootstrap without
// --conventions behaves the same as before (backward compatibility).
func TestBootstrapWithoutConventions_Unchanged(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupTestRepo(t)

	out := runCmd(bin, "bootstrap", "--no-verify", repoDir)
	if out.err != nil {
		t.Fatalf("bootstrap without --conventions failed: %v\n%s", out.err, out.combined())
	}

	// DRAFT-CLAUDE-rules.md should NOT exist (no --conventions).
	draftRules := filepath.Join(repoDir, "DRAFT-CLAUDE-rules.md")
	if _, err := os.Stat(draftRules); err == nil {
		t.Error("DRAFT-CLAUDE-rules.md should NOT exist without --conventions")
	}
}

// TestBootstrapConventions_WithEvidence verifies that bootstrap --conventions
// includes rules when evidence exists.
func TestBootstrapConventions_WithEvidence(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupTestRepo(t)
	setupEvidence(t, repoDir)

	out := runCmd(bin, "bootstrap", "--no-verify", "--conventions", repoDir)
	if out.err != nil {
		t.Fatalf("bootstrap --conventions with evidence failed: %v\n%s", out.err, out.combined())
	}

	output := out.combined()
	if !strings.Contains(output, "rule") && !strings.Contains(output, "conventions") {
		t.Errorf("expected conventions-related output, got:\n%s", output)
	}

	// Adapter output file should contain rule content.
	draftRules := filepath.Join(repoDir, "DRAFT-CLAUDE-rules.md")
	data, err := os.ReadFile(draftRules)
	if err != nil {
		t.Fatalf("failed to read DRAFT-CLAUDE-rules.md: %v", err)
	}
	if !strings.Contains(string(data), "RULE-") {
		t.Errorf("expected at least one rule in adapter output, got:\n%s", string(data))
	}
}

// --- Draft filename derivation tests ---

// TestDraftFilename_StandardPath verifies standard ConfigFile paths.
func TestDraftFilename_StandardPath(t *testing.T) {
	tests := []struct {
		config   string
		expected string
	}{
		{"CLAUDE.md", "DRAFT-CLAUDE-rules.md"},
		{".cursorrules", "DRAFT-cursorrules-rules.md"},
		{".cursor/rules.md", "DRAFT-.cursor/rules-rules.md"},
		{"", "DRAFT-harness-rules.md"},
	}
	for _, tc := range tests {
		h := &harnesscfg.Harness{ConfigFile: tc.config}
		got := draftFilenameFromHarness(h)
		if got != tc.expected {
			t.Errorf("draftFilenameFromHarness(%q) = %q, want %q", tc.config, got, tc.expected)
		}
	}
	// Nil harness should return default.
	if got := draftFilenameFromHarness(nil); got != "DRAFT-harness-rules.md" {
		t.Errorf("draftFilenameFromHarness(nil) = %q, want DRAFT-harness-rules.md", got)
	}
}

// TestBootstrapArtifactName_Dotfile verifies bootstrapRulesArtifactName handles dotfiles.
func TestBootstrapArtifactName_Dotfile(t *testing.T) {
	tests := []struct {
		config   string
		draft    bool
		expected string
	}{
		{"CLAUDE.md", false, "CLAUDE-rules.md"},
		{"CLAUDE.md", true, "DRAFT-CLAUDE-rules.md"},
		{".cursorrules", false, "cursorrules-rules.md"},
		{".cursorrules", true, "DRAFT-cursorrules-rules.md"},
	}
	for _, tc := range tests {
		h := &harnesscfg.Harness{ConfigFile: tc.config}
		got := bootstrapRulesArtifactName(h, tc.draft)
		if got != tc.expected {
			t.Errorf("bootstrapRulesArtifactName(%q, draft=%v) = %q, want %q",
				tc.config, tc.draft, got, tc.expected)
		}
	}
}

// TestRulesUpdate_ExplicitManifestError verifies that providing an explicit
// --manifest path that doesn't exist causes a fatal error (not silent fallback).
func TestRulesUpdate_ExplicitManifestError(t *testing.T) {
	bin := buildRulesBinary(t)
	repoDir := setupRulesTestRepo(t)

	out := runCmd(bin, "rules", "update", "--manifest", "/nonexistent/manifest.json", repoDir)
	if out.err == nil {
		t.Fatal("expected non-zero exit for explicit --manifest with missing file")
	}
	if !strings.Contains(out.combined(), "error") {
		t.Errorf("expected error message, got: %s", out.combined())
	}
}

// --- Test helpers ---

// cmdResult captures the output and error from a command execution.
type cmdResult struct {
	out []byte
	err error
}

func (r *cmdResult) combined() string { return string(r.out) }

// runCmd executes a command and returns its result.
func runCmd(name string, args ...string) cmdResult {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return cmdResult{out: buf.Bytes(), err: err}
}

// evidenceFixture is a simplified evidence entry for test fixtures.
type evidenceFixture struct {
	RunID   string `json:"run_id"`
	Phase   string `json:"phase"`
	Verdict string `json:"verdict"`
	Summary string `json:"summary"`
}

// setupRulesTestRepo creates a minimal project directory that can be
// successfully processed by scout and rules update.
func setupRulesTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	mainGo := `package main
func main() {}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatal(err)
	}

	goMod := `module testproj
go 1.26
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

// setupEvidence creates a .sdp/evidence/ directory with sample failure entries.
func setupEvidence(t *testing.T, repoDir string) {
	t.Helper()
	evDir := filepath.Join(repoDir, ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeEvidenceEntry(t, evDir, "run-001.json", evidenceFixture{
		RunID:   "run-001",
		Phase:   "build",
		Verdict: "fail",
		Summary: "undefined variable in main.go",
	})
	writeEvidenceEntry(t, evDir, "run-002.json", evidenceFixture{
		RunID:   "run-002",
		Phase:   "test",
		Verdict: "error",
		Summary: "test panicked",
	})
}

// writeEvidenceEntry writes a single evidence entry as JSON.
func writeEvidenceEntry(t *testing.T, dir, name string, entry evidenceFixture) {
	t.Helper()
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
