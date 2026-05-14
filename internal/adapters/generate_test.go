package adapters_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/adapters"
	"github.com/fall-out-bug/sdp_lab/internal/manifest"
)

// minimalManifest returns a small but complete manifest useful for most tests.
func minimalManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses: []manifest.Harness{
			manifest.HarnessClaudeCode,
			manifest.HarnessOpenCode,
			manifest.HarnessCodex,
			manifest.HarnessCursor,
			manifest.HarnessPi,
		},
		Skills: []manifest.Skill{
			{Name: "build", Path: "prompts/skills/build/SKILL.md", Summary: "Build skill"},
		},
		Commands: []manifest.Command{
			{Name: "build", Path: "prompts/commands/build.md", Summary: "Build command"},
		},
		Agents: []manifest.Agent{
			{Name: "implementer", Role: "implementer", SystemPromptPath: "prompts/agents/implementer.md", Summary: "Implements features"},
		},
	}
}

// TestGenerate_OutputFiles verifies that a minimal manifest produces the
// expected set of output paths.
func TestGenerate_OutputFiles(t *testing.T) {
	m := minimalManifest()
	out, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	want := []string{
		".claude/commands/build.md",
		".claude/agents/implementer.md",
		".opencode/agent/implementer.json",
		".opencode/skill/build.md",
		".opencode/commands/build.md",
		".codex/skills/build.md",
		".cursor/rules/build.mdc",
		".pi/skills/build/SKILL.md",
		".pi/skills/implementer/SKILL.md",
		".pi/prompts/build.md",
	}
	for _, path := range want {
		if _, ok := out[path]; !ok {
			t.Errorf("expected output file %q not found; got keys: %v", path, mapKeys(out))
		}
	}
}

// TestGenerate_Determinism verifies that two calls with the same manifest
// produce byte-identical maps.
func TestGenerate_Determinism(t *testing.T) {
	m := minimalManifest()
	out1, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	out2, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("second Generate: %v", err)
	}
	if len(out1) != len(out2) {
		t.Fatalf("different file counts: %d vs %d", len(out1), len(out2))
	}
	for k, v1 := range out1 {
		v2, ok := out2[k]
		if !ok {
			t.Errorf("second run missing key %q", k)
			continue
		}
		if string(v1) != string(v2) {
			t.Errorf("file %q differs between runs", k)
		}
	}
}

// TestGenerate_EmptyManifest verifies that a manifest with no entries yields
// zero output files.
func TestGenerate_EmptyManifest(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		// No harnesses, no skills/commands/agents: should produce nothing.
	}
	out, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected 0 output files for empty manifest; got %d: %v", len(out), mapKeys(out))
	}
}

// TestGenerate_HarnessFilter verifies that a command restricted to claude-code
// does NOT appear in the cursor adapter tree.
func TestGenerate_HarnessFilter(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses: []manifest.Harness{
			manifest.HarnessClaudeCode,
			manifest.HarnessCursor,
		},
		Commands: []manifest.Command{
			{
				Name:      "claude-only",
				Path:      "prompts/commands/build.md",
				Harnesses: []manifest.Harness{manifest.HarnessClaudeCode},
			},
		},
	}
	out, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, ok := out[".claude/commands/claude-only.md"]; !ok {
		t.Error("expected .claude/commands/claude-only.md to exist")
	}
	if _, ok := out[".cursor/rules/claude-only.mdc"]; ok {
		t.Error("unexpected .cursor/rules/claude-only.mdc should not exist for claude-code-only command")
	}
}

func TestGenerate_CommandDispatchOverride(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses: []manifest.Harness{
			manifest.HarnessClaudeCode,
			manifest.HarnessCursor,
			manifest.HarnessOpenCode,
			manifest.HarnessPi,
		},
		Commands: []manifest.Command{
			{
				Name: "custom",
				Path: "prompts/commands/custom.md",
				Dispatch: map[manifest.Harness]string{
					manifest.HarnessClaudeCode: ".claude/commands/custom-alias.md",
					manifest.HarnessCursor:     ".cursor/rules/custom-alias.mdc",
					manifest.HarnessPi:         ".pi/prompts/custom-alias.md",
					manifest.HarnessOpenCode:   ".opencode/commands/custom-alias.md",
				},
			},
		},
	}

	out, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, path := range []string{
		".claude/commands/custom-alias.md",
		".cursor/rules/custom-alias.mdc",
		".pi/prompts/custom-alias.md",
		".opencode/commands/custom-alias.md",
	} {
		if _, ok := out[path]; !ok {
			t.Fatalf("expected dispatch override output %q, got keys: %v", path, mapKeys(out))
		}
	}
	for _, path := range []string{
		".claude/commands/custom.md",
		".cursor/rules/custom.mdc",
		".pi/prompts/custom.md",
		".opencode/commands/custom.md",
	} {
		if _, ok := out[path]; ok {
			t.Fatalf("unexpected default output %q when dispatch override is set", path)
		}
	}
}

func TestGenerate_CommandDispatchOverrideRejectsTraversal(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses: []manifest.Harness{
			manifest.HarnessClaudeCode,
		},
		Commands: []manifest.Command{
			{
				Name: "escape",
				Path: "prompts/commands/escape.md",
				Dispatch: map[manifest.Harness]string{
					manifest.HarnessClaudeCode: "../../outside.md",
				},
			},
		},
	}

	if _, err := adapters.Generate(m, ""); err == nil {
		t.Fatal("Generate returned nil error for traversal dispatch override")
	}
}

// TestGenerate_PerHarnessHasFiles verifies that a full minimal manifest has at
// least one file per harness prefix.
func TestGenerate_PerHarnessHasFiles(t *testing.T) {
	m := minimalManifest()
	out, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	prefixes := map[string]bool{
		".claude/":   false,
		".opencode/": false,
		".codex/":    false,
		".cursor/":   false,
		".pi/":       false,
	}
	for k := range out {
		for prefix := range prefixes {
			if strings.HasPrefix(k, prefix) {
				prefixes[prefix] = true
			}
		}
	}
	for prefix, found := range prefixes {
		if !found {
			t.Errorf("expected at least one file with prefix %q", prefix)
		}
	}
}

// TestGenerate_ContainsGeneratedMarker verifies that every generated file
// contains the canonical "GENERATED by sdp generate-adapters" marker.
func TestGenerate_ContainsGeneratedMarker(t *testing.T) {
	m := minimalManifest()
	out, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no output files produced")
	}
	for path, contents := range out {
		if !strings.Contains(string(contents), "GENERATED by sdp generate-adapters") {
			t.Errorf("file %q missing generated marker; got:\n%s", path, string(contents))
		}
	}
}

// TestGenerate_OpenCodeAgentJSON verifies that .opencode/agent/<name>.json
// is valid JSON.
func TestGenerate_OpenCodeAgentJSON(t *testing.T) {
	m := minimalManifest()
	out, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for path, contents := range out {
		if !strings.HasPrefix(path, ".opencode/agent/") {
			continue
		}
		var v map[string]any
		if err := json.Unmarshal(contents, &v); err != nil {
			t.Errorf("file %q is not valid JSON: %v\ncontent:\n%s", path, err, string(contents))
		}
	}
}

// TestGenerate_EmptyPath verifies that when a manifest item has an empty path
// (or the file does not exist at repoRoot), generation succeeds without error
// and still produces output (just without embedded body).
func TestGenerate_EmptyPath(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses:  []manifest.Harness{manifest.HarnessClaudeCode},
		Commands: []manifest.Command{
			{Name: "nopath", Path: "", Summary: "Command with no path"},
		},
	}
	// Use a real temp dir as repoRoot — the path "" won't resolve to any file
	dir := t.TempDir()
	out, err := adapters.Generate(m, dir)
	if err != nil {
		t.Fatalf("Generate with empty path returned error: %v", err)
	}
	content, ok := out[".claude/commands/nopath.md"]
	if !ok {
		t.Fatal("expected .claude/commands/nopath.md in output")
	}
	// Should still have the generated marker
	if !strings.Contains(string(content), "GENERATED by sdp generate-adapters") {
		t.Errorf("missing generated marker; got:\n%s", string(content))
	}
}

// TestGenerate_PiPromptTemplatePreservesArguments verifies that Pi slash
// templates keep user command arguments. Pi drops the original slash invocation
// during expansion unless the template includes $ARGUMENTS explicitly.
func TestGenerate_PiPromptTemplatePreservesArguments(t *testing.T) {
	m := minimalManifest()
	out, err := adapters.Generate(m, "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content, ok := out[".pi/prompts/build.md"]
	if !ok {
		t.Fatal("expected .pi/prompts/build.md in output")
	}
	if !strings.Contains(string(content), "User arguments: $ARGUMENTS") {
		t.Errorf("Pi prompt template must preserve command args; got:\n%s", string(content))
	}
}

func TestGenerate_PiPromptRewritesLegacyClaudeSkillRefs(t *testing.T) {
	dir := t.TempDir()
	commandDir := filepath.Join(dir, "prompts", "commands")
	if err := os.MkdirAll(commandDir, 0o755); err != nil {
		t.Fatalf("mkdir command dir: %v", err)
	}
	body := "---\ndescription: Demo\n---\n# Demo\n\n1. Load skill: `@.claude/skills/build/SKILL.md`\n2. Load skill: `.claude/skills/ship/SKILL.md`\n"
	if err := os.WriteFile(filepath.Join(commandDir, "demo.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses:  []manifest.Harness{manifest.HarnessPi},
		Commands: []manifest.Command{
			{Name: "demo", Path: "prompts/commands/demo.md", Summary: "Demo command"},
		},
	}
	out, err := adapters.Generate(m, dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content := string(out[".pi/prompts/demo.md"])
	if strings.Contains(content, ".claude/skills/") {
		t.Fatalf("Pi prompt leaked legacy Claude skill path; got:\n%s", content)
	}
	for _, want := range []string{
		"1. Load skill: `build`",
		"2. Load skill: `ship`",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("Pi prompt missing rewritten skill ref %q; got:\n%s", want, content)
		}
	}
}

func TestGenerate_OpenCodeCommandRewritesLegacyClaudeSkillRefs(t *testing.T) {
	dir := t.TempDir()
	commandDir := filepath.Join(dir, "prompts", "commands")
	if err := os.MkdirAll(commandDir, 0o755); err != nil {
		t.Fatalf("mkdir command dir: %v", err)
	}
	body := "---\ndescription: Demo\nagent: builder\n---\n# Demo\n\n1. Load skill: `@.claude/skills/build/SKILL.md`\n2. Load skill: `.claude/skills/ship.md`\n3. Load skill: `.claude/skills/deploy`\n"
	if err := os.WriteFile(filepath.Join(commandDir, "demo.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses:  []manifest.Harness{manifest.HarnessOpenCode},
		Commands: []manifest.Command{
			{Name: "demo", Path: "prompts/commands/demo.md", Summary: "Demo command"},
		},
	}

	out, err := adapters.Generate(m, dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content, ok := out[".opencode/commands/demo.md"]
	if !ok {
		t.Fatal("expected .opencode/commands/demo.md in output")
	}
	s := string(content)
	if strings.Contains(s, ".claude/skills/") {
		t.Fatalf("OpenCode command leaked legacy Claude skill path; got:\n%s", s)
	}
	for _, want := range []string{
		"1. Load skill: `build`",
		"2. Load skill: `ship`",
		"3. Load skill: `deploy`",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("OpenCode command missing rewritten skill ref %q; got:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "GENERATED by sdp generate-adapters") {
		t.Fatalf("OpenCode command missing generated marker; got:\n%s", s)
	}
	if !strings.Contains(s, "Source: prompts/commands/demo.md") {
		t.Fatalf("OpenCode command missing source provenance; got:\n%s", s)
	}
	if strings.Contains(s, "---\ndescription: Demo") {
		t.Fatalf("OpenCode command leaked source frontmatter; got:\n%s", s)
	}
	if strings.Contains(s, "agent: builder") {
		t.Fatalf("OpenCode command leaked source command agent hint; got:\n%s", s)
	}
}

func TestGenerate_OpenCodeCommandIsHarnessAppropriate(t *testing.T) {
	dir := t.TempDir()
	commandDir := filepath.Join(dir, "prompts", "commands")
	if err := os.MkdirAll(commandDir, 0o755); err != nil {
		t.Fatalf("mkdir command dir: %v", err)
	}
	body := "# Demo\n\nJust execute the build command with arguments."
	if err := os.WriteFile(filepath.Join(commandDir, "nofront.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses:  []manifest.Harness{manifest.HarnessOpenCode},
		Commands: []manifest.Command{
			{Name: "nofront", Path: "prompts/commands/nofront.md", Summary: "Command with no frontmatter"},
		},
	}

	out, err := adapters.Generate(m, dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content, ok := out[".opencode/commands/nofront.md"]
	if !ok {
		t.Fatal("expected .opencode/commands/nofront.md in output")
	}
	s := string(content)
	if !strings.Contains(s, "GENERATED by sdp generate-adapters") {
		t.Fatalf("OpenCode command missing generated marker; got:\n%s", s)
	}
	if !strings.Contains(s, "Source: prompts/commands/nofront.md") {
		t.Fatalf("OpenCode command missing source provenance; got:\n%s", s)
	}
	if strings.Contains(s, ".claude/skills/") {
		t.Fatalf("OpenCode command should not carry Claude-specific skill refs; got:\n%s", s)
	}
}

func TestGenerate_CursorRuleRewritesLegacyClaudeSkillRefsAndHookPaths(t *testing.T) {
	dir := t.TempDir()
	commandDir := filepath.Join(dir, "prompts", "commands")
	if err := os.MkdirAll(commandDir, 0o755); err != nil {
		t.Fatalf("mkdir command dir: %v", err)
	}
	body := "---\ndescription: Demo\nagent: builder\n---\n# Demo\n\n1. Load skill: `@.claude/skills/build/SKILL.md`\n2. Run pre-build hook: `hooks/pre-build.sh {WS-ID}`\n3. Run post-build hook: `hooks/post-build.sh {WS-ID}`\n"
	if err := os.WriteFile(filepath.Join(commandDir, "demo.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses:  []manifest.Harness{manifest.HarnessCursor},
		Commands: []manifest.Command{
			{Name: "demo", Path: "prompts/commands/demo.md", Summary: "Demo command"},
		},
	}

	out, err := adapters.Generate(m, dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content, ok := out[".cursor/rules/demo.mdc"]
	if !ok {
		t.Fatal("expected .cursor/rules/demo.mdc in output")
	}
	s := string(content)
	for _, bad := range []string{".claude/skills/", "Run pre-build hook: `hooks/pre-build.sh", "Run post-build hook: `hooks/post-build.sh", "agent: builder"} {
		if strings.Contains(s, bad) {
			t.Fatalf("Cursor rule leaked %q; got:\n%s", bad, s)
		}
	}
	for _, want := range []string{
		"1. Load skill: `build`",
		"2. Run pre-build hook: `scripts/hooks/pre-build.sh {WS-ID}`",
		"3. Run post-build hook: `scripts/hooks/post-build.sh {WS-ID}`",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("Cursor rule missing rewritten content %q; got:\n%s", want, s)
		}
	}
}

// TestGenerate_BodyEmbed verifies that when a manifest item points to a real
// file, the body of that file is embedded verbatim in the generated output.
func TestGenerate_BodyEmbed(t *testing.T) {
	const magicToken = "MAGIC_BODY_TOKEN_42"

	dir := t.TempDir()
	// Create fixture file
	skillDir := filepath.Join(dir, "prompts", "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("# Demo\n\n"+magicToken+"\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Harnesses:  []manifest.Harness{manifest.HarnessCodex},
		Skills: []manifest.Skill{
			{Name: "demo", Path: "prompts/skills/demo/SKILL.md", Summary: "Demo skill"},
		},
	}

	out, err := adapters.Generate(m, dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content, ok := out[".codex/skills/demo.md"]
	if !ok {
		t.Fatal("expected .codex/skills/demo.md in output")
	}
	if !strings.Contains(string(content), magicToken) {
		t.Errorf("expected body token %q in generated output; got:\n%s", magicToken, string(content))
	}
}

// mapKeys returns a sorted list of map keys for diagnostic output.
func mapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
