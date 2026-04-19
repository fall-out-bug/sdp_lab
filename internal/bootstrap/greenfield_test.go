package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunGreenfieldFromPreset_GoWebService(t *testing.T) {
	result, err := RunGreenfieldFromPreset("go-web-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertValidBootstrapResult(t, result, "go-web-service")
	assertContentContains(t, result.PrinciplesContent, "web-service")
	assertContentContains(t, result.PrinciplesContent, "docker")
	assertContentContains(t, result.AgentsContent, "tdd")
}

func TestRunGreenfieldFromPreset_GoCLI(t *testing.T) {
	result, err := RunGreenfieldFromPreset("go-cli")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertValidBootstrapResult(t, result, "go-cli")
	assertContentContains(t, result.PrinciplesContent, "cli")
	assertContentContains(t, result.AgentsContent, "unit")
}

func TestRunGreenfieldFromPreset_GoLibrary(t *testing.T) {
	result, err := RunGreenfieldFromPreset("go-library")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertValidBootstrapResult(t, result, "go-library")
	assertContentContains(t, result.PrinciplesContent, "library")
	assertContentContains(t, result.AgentsContent, "tdd")
}

func TestRunGreenfieldFromPreset_InvalidPreset(t *testing.T) {
	_, err := RunGreenfieldFromPreset("nonexistent-preset")
	if err == nil {
		t.Fatal("expected error for invalid preset, got nil")
	}
	if !strings.Contains(err.Error(), "unknown preset") {
		t.Errorf("error should mention unknown preset, got: %v", err)
	}
}

func TestRunGreenfield_CustomConfig(t *testing.T) {
	cfg := GreenfieldConfig{
		ProjectType:     "web-service",
		PrimaryLanguage: "python",
		TestStrategy:    "integration",
		CIPreference:    "gitlab-ci",
		DeployTarget:    "kubernetes",
	}
	result, err := RunGreenfield(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	assertContentContains(t, result.PrinciplesContent, "python")
	assertContentContains(t, result.PrinciplesContent, "kubernetes")
	assertContentContains(t, result.AgentsContent, "integration")
	assertContentContains(t, result.AgentsContent, "gitlab-ci")
}

func TestRenderPrinciples_ContainsSections(t *testing.T) {
	cfg := Presets["go-web-service"]
	content := renderPrinciples(cfg)

	sections := []string{"Values", "Architecture", "Quality"}
	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("principles content missing section: %s", section)
		}
	}
}

func TestRenderAgentsRules_ContainsTestStrategy(t *testing.T) {
	cfg := Presets["go-library"]
	content := renderAgentsRules(cfg)

	if !strings.Contains(content, "tdd") {
		t.Errorf("agents rules should mention test strategy 'tdd', got:\n%s", content)
	}
	if !strings.Contains(content, "go") {
		t.Errorf("agents rules should mention language 'go', got:\n%s", content)
	}
}

func TestPresets_AllValid(t *testing.T) {
	for name, cfg := range Presets {
		t.Run(name, func(t *testing.T) {
			result, err := RunGreenfield(cfg)
			if err != nil {
				t.Fatalf("preset %s: unexpected error: %v", name, err)
			}
			if result.PrinciplesContent == "" {
				t.Errorf("preset %s: empty PrinciplesContent", name)
			}
			if result.AgentsContent == "" {
				t.Errorf("preset %s: empty AgentsContent", name)
			}
			if result.ConfigFile == "" {
				t.Errorf("preset %s: empty ConfigFile", name)
			}
			if !strings.HasPrefix(result.PrinciplesContent, "<!-- DRAFT") {
				t.Errorf("preset %s: PrinciplesContent missing DRAFT header", name)
			}
			if !strings.HasPrefix(result.AgentsContent, "<!-- DRAFT") {
				t.Errorf("preset %s: AgentsContent missing DRAFT header", name)
			}
		})
	}
}

func TestDeterministic(t *testing.T) {
	cfg := GreenfieldConfig{
		ProjectType:     "web-service",
		PrimaryLanguage: "go",
		TestStrategy:    "tdd",
		CIPreference:    "github-actions",
		DeployTarget:    "docker",
	}
	r1, err := RunGreenfield(cfg)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	r2, err := RunGreenfield(cfg)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if r1.PrinciplesContent != r2.PrinciplesContent {
		t.Error("PrinciplesContent not deterministic")
	}
	if r1.AgentsContent != r2.AgentsContent {
		t.Error("AgentsContent not deterministic")
	}
	if r1.ConfigFile != r2.ConfigFile {
		t.Error("ConfigFile not deterministic")
	}
}

func TestGreenfieldConfigJSONRoundTrip(t *testing.T) {
	cfg := GreenfieldConfig{
		ProjectType:     "cli",
		PrimaryLanguage: "go",
		TestStrategy:    "unit",
		CIPreference:    "github-actions",
		DeployTarget:    "none",
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded GreenfieldConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded != cfg {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", decoded, cfg)
	}
}

func TestQuestions_DefaultQuestionsComplete(t *testing.T) {
	if len(DefaultQuestions) == 0 {
		t.Fatal("DefaultQuestions should not be empty")
	}
	seenKeys := map[string]bool{}
	for _, q := range DefaultQuestions {
		if q.Key == "" {
			t.Error("question with empty Key")
		}
		if q.Prompt == "" {
			t.Errorf("question key=%s has empty Prompt", q.Key)
		}
		if len(q.Options) == 0 {
			t.Errorf("question key=%s has no Options", q.Key)
		}
		if q.Default == "" {
			t.Errorf("question key=%s has empty Default", q.Key)
		}
		seenKeys[q.Key] = true
	}
	expectedKeys := []string{
		"project_type", "primary_language",
		"test_strategy", "ci_preference", "deploy_target",
	}
	for _, k := range expectedKeys {
		if !seenKeys[k] {
			t.Errorf("missing question for key: %s", k)
		}
	}
}

// --- helpers ---

func assertValidBootstrapResult(t *testing.T, result *BootstrapResult, presetName string) {
	t.Helper()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.PrinciplesContent == "" {
		t.Errorf("preset %s: PrinciplesContent is empty", presetName)
	}
	if result.AgentsContent == "" {
		t.Errorf("preset %s: AgentsContent is empty", presetName)
	}
	if result.ConfigFile == "" {
		t.Errorf("preset %s: ConfigFile is empty", presetName)
	}
	if !strings.Contains(result.PrinciplesContent, "DRAFT") {
		t.Errorf("preset %s: PrinciplesContent should contain DRAFT marker", presetName)
	}
	if !strings.Contains(result.AgentsContent, "DRAFT") {
		t.Errorf("preset %s: AgentsContent should contain DRAFT marker", presetName)
	}
}

func assertContentContains(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("content should contain %q", substr)
	}
}
