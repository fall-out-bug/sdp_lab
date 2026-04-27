package harnessadapter

import (
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/harnesscfg"
	"github.com/fall-out-bug/sdp_lab/internal/rules"
	"github.com/fall-out-bug/sdp_lab/internal/scout"
)

func enabledPtr(v bool) *bool { return &v }

func sampleManifest() *harnesscfg.Manifest {
	return &harnesscfg.Manifest{
		Version:        "0.1.0",
		LifecycleStage: "greenfield",
		Harnesses: []harnesscfg.Harness{
			{Name: "claude-code", ConfigFile: "CLAUDE.md"},
			{Name: "cursor", ConfigFile: ".cursorrules"},
			{Name: "codex-cli", ConfigFile: "AGENTS.md"},
			{Name: "opencode", ConfigFile: "AGENTS.md"},
		},
	}
}

func sampleCard() *scout.ProjectCard {
	return &scout.ProjectCard{
		Version: "1",
		Identity: scout.Identity{
			Name:            "testproj",
			PrimaryLanguage: "go",
		},
		Conventions: scout.Conventions{
			ModulePatterns: []scout.ModulePattern{
				{
					Name:     "Repository pattern",
					Pattern:  "internal/*/repo.go",
					Examples: []string{"internal/user/repo.go", "internal/order/repo.go"},
				},
				{
					Name:     "Handler pattern",
					Pattern:  "internal/*/handler.go",
					Examples: []string{"internal/user/handler.go"},
				},
			},
			TestStructure: scout.TestLayout{
				Style:      "colocated",
				DirPattern: "*_test.go",
			},
		},
	}
}

func sampleRules() []rules.Rule {
	return []rules.Rule{
		{
			ID:          "RULE-002",
			Title:       "No global state",
			Source:      rules.SourceHumanAnnotated,
			EvidenceRef: "docs/rules.md",
			Severity:    rules.SeverityWarning,
			Description: "Avoid package-level mutable state.",
		},
		{
			ID:          "RULE-001",
			Title:       "Observed build failure",
			Source:      rules.SourceObservedFailure,
			EvidenceRef: "evidence/run1.json",
			Severity:    rules.SeverityError,
			Description: "Build failed due to missing import.",
		},
	}
}

// --- Registry tests ---

func TestNewRegistry(t *testing.T) {
	r := NewRegistry(sampleManifest())
	all := r.All()
	// 4 harnesses in manifest: claude-code, cursor, codex-cli, opencode
	if len(all) != 4 {
		t.Fatalf("expected 4 adapters from sample manifest, got %d", len(all))
	}

	names := make(map[string]bool)
	for _, na := range all {
		names[na.harnessName] = true
	}

	// All four harness names must be present (not collapsed into adapter names).
	if !names["claude-code"] {
		t.Error("expected claude-code harness")
	}
	if !names["cursor"] {
		t.Error("expected cursor harness")
	}
	if !names["codex-cli"] {
		t.Error("expected codex-cli harness")
	}
	if !names["opencode"] {
		t.Error("expected opencode harness")
	}
}

func TestNewRegistry_NilManifest(t *testing.T) {
	r := NewRegistry(nil)
	if len(r.All()) != 0 {
		t.Error("nil manifest should produce empty registry")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry(sampleManifest())
	a, err := r.Get("claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Name() != "claude-code" {
		t.Errorf("expected name claude-code, got %s", a.Name())
	}
}

func TestRegistry_GetAgentsHarness(t *testing.T) {
	r := NewRegistry(sampleManifest())
	a, err := r.Get("codex-cli")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Name() != "agents" {
		t.Errorf("expected adapter name agents, got %s", a.Name())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry(sampleManifest())
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown adapter name")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestRegistry_DisabledHarness(t *testing.T) {
	m := &harnesscfg.Manifest{
		Version:        "0.1.0",
		LifecycleStage: "greenfield",
		Harnesses: []harnesscfg.Harness{
			{Name: "claude-code", ConfigFile: "CLAUDE.md", Enabled: enabledPtr(false)},
		},
	}
	r := NewRegistry(m)
	if len(r.All()) != 0 {
		t.Error("disabled harness should not appear in registry")
	}
}

// --- Claude adapter tests ---

func TestClaudeAdapter_Render(t *testing.T) {
	a := newClaudeAdapter()
	out, err := a.Render(sampleCard(), sampleRules())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(out)

	// Must contain conventions
	if !strings.Contains(s, "## Language Patterns") {
		t.Error("expected Language Patterns heading")
	}
	if !strings.Contains(s, "Handler pattern") {
		t.Error("expected Handler pattern in output")
	}
	if !strings.Contains(s, "Repository pattern") {
		t.Error("expected Repository pattern in output")
	}
	if !strings.Contains(s, "colocated") {
		t.Error("expected test layout style")
	}

	// Must contain rules, sorted by ID
	if !strings.Contains(s, "## Rules") {
		t.Error("expected Rules heading")
	}
	if !strings.Contains(s, "RULE-001") {
		t.Error("expected RULE-001")
	}
	if !strings.Contains(s, "RULE-002") {
		t.Error("expected RULE-002")
	}

	// Verify ordering: RULE-001 before RULE-002
	idx1 := strings.Index(s, "RULE-001")
	idx2 := strings.Index(s, "RULE-002")
	if idx1 >= idx2 {
		t.Error("RULE-001 should appear before RULE-002")
	}
}

func TestClaudeAdapter_Name(t *testing.T) {
	a := newClaudeAdapter()
	if a.Name() != "claude-code" {
		t.Errorf("expected claude-code, got %s", a.Name())
	}
}

// --- Cursor adapter tests ---

func TestCursorAdapter_Render(t *testing.T) {
	a := newCursorAdapter()
	out, err := a.Render(sampleCard(), sampleRules())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(out)

	if !strings.Contains(s, "# Conventions") {
		t.Error("expected Conventions heading")
	}
	if !strings.Contains(s, "Handler pattern") {
		t.Error("expected Handler pattern")
	}
	if !strings.Contains(s, "RULE-001") {
		t.Error("expected RULE-001")
	}
	if !strings.Contains(s, "RULE-002") {
		t.Error("expected RULE-002")
	}
	if !strings.Contains(s, "Severity:") {
		t.Error("expected Severity field")
	}
}

func TestCursorAdapter_Name(t *testing.T) {
	a := newCursorAdapter()
	if a.Name() != "cursor" {
		t.Errorf("expected cursor, got %s", a.Name())
	}
}

// --- Agents adapter tests ---

func TestAgentsAdapter_Render(t *testing.T) {
	a := newAgentsAdapter()
	out, err := a.Render(sampleCard(), sampleRules())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(out)

	if !strings.Contains(s, "## Code Conventions") {
		t.Error("expected Code Conventions heading")
	}
	if !strings.Contains(s, "## Observed Rules") {
		t.Error("expected Observed Rules heading")
	}
	if !strings.Contains(s, "RULE-001") {
		t.Error("expected RULE-001")
	}
	if !strings.Contains(s, "RULE-002") {
		t.Error("expected RULE-002")
	}
}

func TestAgentsAdapter_Name(t *testing.T) {
	a := newAgentsAdapter()
	if a.Name() != "agents" {
		t.Errorf("expected agents, got %s", a.Name())
	}
}

// --- RenderAll test ---

func TestRenderAll(t *testing.T) {
	r := NewRegistry(sampleManifest())
	out, err := r.RenderAll(sampleCard(), sampleRules())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output for all adapters")
	}

	// Must have 4 entries keyed by harness name, not collapsed to adapter names.
	if _, ok := out["claude-code"]; !ok {
		t.Error("missing claude-code output")
	}
	if _, ok := out["cursor"]; !ok {
		t.Error("missing cursor output")
	}
	if _, ok := out["codex-cli"]; !ok {
		t.Error("missing codex-cli output")
	}
	if _, ok := out["opencode"]; !ok {
		t.Error("missing opencode output")
	}
}

func TestRenderAll_DistinctOutput(t *testing.T) {
	r := NewRegistry(sampleManifest())
	out, err := r.RenderAll(sampleCard(), sampleRules())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// codex-cli and opencode share agentsAdapter so content is identical,
	// but they must be distinct entries in the map.
	if len(out) != 4 {
		t.Errorf("expected 4 entries, got %d", len(out))
	}
}

// --- Determinism test ---

func TestRender_Deterministic(t *testing.T) {
	card := sampleCard()
	rl := sampleRules()

	cl := newClaudeAdapter()
	cur := newCursorAdapter()
	ag := newAgentsAdapter()

	adapters := []Adapter{cl, cur, ag}

	for _, a := range adapters {
		out1, err1 := a.Render(card, rl)
		out2, err2 := a.Render(card, rl)
		if err1 != nil || err2 != nil {
			t.Fatalf("adapter %s: unexpected error: %v / %v", a.Name(), err1, err2)
		}
		if string(out1) != string(out2) {
			t.Errorf("adapter %s: output not deterministic", a.Name())
		}
	}
}

// --- Edge cases ---

func TestRender_EmptyRules(t *testing.T) {
	cl := newClaudeAdapter()
	cur := newCursorAdapter()
	ag := newAgentsAdapter()

	card := sampleCard()

	for _, a := range []Adapter{cl, cur, ag} {
		out, err := a.Render(card, nil)
		if err != nil {
			t.Fatalf("adapter %s: unexpected error with nil rules: %v", a.Name(), err)
		}
		if len(out) == 0 {
			// Convention-only output is valid; only fail if we expected content.
			// With a card that has conventions, output should be non-empty for claude.
			if a.Name() == "claude-code" && len(out) == 0 {
				t.Errorf("claude adapter should produce conventions section even without rules")
			}
		}
	}

	// No rules should NOT produce "Rules" heading
	clOut, _ := cl.Render(card, nil)
	if strings.Contains(string(clOut), "## Rules") {
		t.Error("claude adapter should not emit Rules section when no rules provided")
	}
}

func TestRender_NilCard(t *testing.T) {
	cl := newClaudeAdapter()
	cur := newCursorAdapter()
	ag := newAgentsAdapter()

	rl := sampleRules()
	for _, a := range []Adapter{cl, cur, ag} {
		out, err := a.Render(nil, rl)
		if err != nil {
			t.Fatalf("adapter %s: unexpected error with nil card: %v", a.Name(), err)
		}
		if len(out) == 0 {
			t.Errorf("adapter %s: expected non-empty output with rules only", a.Name())
		}
	}
}

func TestRender_NilConventions(t *testing.T) {
	cl := newClaudeAdapter()
	card := &scout.ProjectCard{
		Version: "1",
		Identity: scout.Identity{
			Name:            "testproj",
			PrimaryLanguage: "go",
		},
		// Conventions is zero-value (nil ModulePatterns, empty TestStructure)
	}

	out, err := cl.Render(card, sampleRules())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(out)
	// Should have rules but not Language Patterns heading
	if strings.Contains(s, "## Language Patterns") {
		t.Error("should not produce Language Patterns section with empty conventions")
	}
	if !strings.Contains(s, "## Rules") {
		t.Error("should still produce Rules section")
	}
}
