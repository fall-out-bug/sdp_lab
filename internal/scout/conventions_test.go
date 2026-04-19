package scout

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// ── Conventions Zero Value ──────────────────────────────────────────────

func TestConventionsZeroValueIsValid(t *testing.T) {
	c := Conventions{}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal zero Conventions: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Nil slice should serialize as null or []
	patterns := m["module_patterns"]
	if patterns != nil {
		arr, ok := patterns.([]any)
		if ok && len(arr) != 0 {
			t.Errorf("module_patterns: expected null or empty, got %v", patterns)
		}
	}

	// Nil pointers should be null
	if lint := m["lint_config"]; lint != nil {
		t.Errorf("lint_config: expected null, got %v", lint)
	}
	if ci := m["ci_workflow"]; ci != nil {
		t.Errorf("ci_workflow: expected null, got %v", ci)
	}

	// TestStructure should have empty/zero fields
	ts, ok := m["test_structure"].(map[string]any)
	if !ok {
		t.Fatalf("test_structure: expected object, got %T", m["test_structure"])
	}
	if ts["style"] != "" {
		t.Errorf("test_structure.style: expected empty, got %v", ts["style"])
	}
}

// ── Extract Conventions: Go Project ─────────────────────────────────────

func TestExtractConventions_GoProject(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":                           "module example.com/app\ngo 1.26\n",
		"cmd/app/main.go":                  "package main\nfunc main() {}\n",
		"internal/foo/handler.go":          "package foo\nfunc Handle() {}\n",
		"internal/foo/handler_test.go":     "package foo\nimport \"testing\"\nfunc TestHandle(t *testing.T) {}\n",
		"pkg/bar/util.go":                  "package bar\nfunc Util() {}\n",
		"main_test.go":                     "package main\nimport \"testing\"\nfunc TestMain(t *testing.T) {}\n",
		".golangci.yml":                    "linters:\n  enable:\n    - errcheck\n    - govet\n",
		".github/workflows/ci.yml":         "name: CI\non: [push]\njobs:\n  build:\n    steps:\n      - uses: actions/checkout@v4\n      - run: go test ./...\n",
	}, false)

	identity := Identity{
		PrimaryLanguage: "go",
		Languages:       map[string]LangStats{"go": {Files: 5, Ratio: 1.0}},
		BuildSystem:     strPtr("go-modules"),
	}
	scale := Scale{
		TestFiles:   2,
		SourceFiles: 3,
	}

	conv := extractConventions(dir, identity, scale)

	// Module patterns
	if len(conv.ModulePatterns) == 0 {
		t.Fatal("ModulePatterns: expected at least one pattern")
	}
	names := make(map[string]bool)
	for _, mp := range conv.ModulePatterns {
		names[mp.Name] = true
		if mp.Pattern == "" {
			t.Errorf("ModulePattern %q: Pattern is empty", mp.Name)
		}
	}
	if !names["cmd"] {
		t.Error("expected 'cmd' module pattern")
	}
	if !names["internal"] {
		t.Error("expected 'internal' module pattern")
	}
	if !names["pkg"] {
		t.Error("expected 'pkg' module pattern")
	}

	// Test structure: colocated (both _test.go files are next to source)
	if conv.TestStructure.Style != "colocated" {
		t.Errorf("TestStructure.Style = %q, want %q", conv.TestStructure.Style, "colocated")
	}

	// Lint config
	if conv.LintConfig == nil {
		t.Fatal("LintConfig: expected non-nil")
	}
	if conv.LintConfig.Tool != "golangci-lint" {
		t.Errorf("LintConfig.Tool = %q, want %q", conv.LintConfig.Tool, "golangci-lint")
	}
	if conv.LintConfig.ConfigFile != ".golangci.yml" {
		t.Errorf("LintConfig.ConfigFile = %q, want %q", conv.LintConfig.ConfigFile, ".golangci.yml")
	}

	// CI workflow
	if conv.CIWorkflow == nil {
		t.Fatal("CIWorkflow: expected non-nil")
	}
	if conv.CIWorkflow.System != "github-actions" {
		t.Errorf("CIWorkflow.System = %q, want %q", conv.CIWorkflow.System, "github-actions")
	}
	if conv.CIWorkflow.ConfigFile != ".github/workflows/ci.yml" {
		t.Errorf("CIWorkflow.ConfigFile = %q, want %q", conv.CIWorkflow.ConfigFile, ".github/workflows/ci.yml")
	}
}

// ── Extract Conventions: Empty Dir ──────────────────────────────────────

func TestExtractConventions_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	conv := extractConventions(dir, Identity{}, Scale{})

	if len(conv.ModulePatterns) != 0 {
		t.Errorf("ModulePatterns: expected empty, got %d", len(conv.ModulePatterns))
	}
	if conv.TestStructure.Style != "unknown" {
		t.Errorf("TestStructure.Style = %q, want %q", conv.TestStructure.Style, "unknown")
	}
	if conv.LintConfig != nil {
		t.Error("LintConfig: expected nil for empty dir")
	}
	if conv.CIWorkflow != nil {
		t.Error("CIWorkflow: expected nil for empty dir")
	}
}

// ── Detect Module Patterns ──────────────────────────────────────────────

func TestDetectModulePatterns(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":               "module example.com/app\ngo 1.26\n",
		"cmd/server/main.go":   "package main\nfunc main() {}\n",
		"internal/pkg/foo.go":  "package pkg\n",
		"pkg/bar/bar.go":       "package bar\n",
		"api/types/types.go":   "package types\n",
		"web/static/app.js":    "// app\n",
	}, false)

	patterns := detectModulePatterns(dir)

	found := make(map[string]bool)
	for _, p := range patterns {
		found[p.Name] = true
	}
	for _, name := range []string{"cmd", "internal", "pkg", "api", "web"} {
		if !found[name] {
			t.Errorf("missing module pattern %q", name)
		}
	}
}

func TestDetectModulePatterns_NoGoDirs(t *testing.T) {
	dir := t.TempDir()

	patterns := detectModulePatterns(dir)
	if len(patterns) != 0 {
		t.Errorf("expected no patterns for empty dir, got %d", len(patterns))
	}
}

// ── Detect Test Layout ──────────────────────────────────────────────────

func TestDetectTestLayout_Colocated(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":          "module example.com/app\ngo 1.26\n",
		"foo.go":          "package main\n",
		"foo_test.go":     "package main\n",
		"bar.go":          "package main\n",
		"bar_test.go":     "package main\n",
	}, false)

	layout := detectTestLayout(dir)

	if layout.Style != "colocated" {
		t.Errorf("Style = %q, want %q", layout.Style, "colocated")
	}
}

func TestDetectTestLayout_TestDir(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":              "module example.com/app\ngo 1.26\n",
		"src/foo.go":          "package main\n",
		"test/test_foo.py":    "def test_foo(): pass\n",
		"test/test_bar.py":    "def test_bar(): pass\n",
	}, false)

	layout := detectTestLayout(dir)

	if layout.Style != "testdir" {
		t.Errorf("Style = %q, want %q", layout.Style, "testdir")
	}
}

func TestDetectTestLayout_Mixed(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":              "module example.com/app\ngo 1.26\n",
		"foo.go":              "package main\n",
		"foo_test.go":         "package main_test\n",
		"tests/integration.go": "package integration\n",
	}, false)

	layout := detectTestLayout(dir)

	if layout.Style != "mixed" {
		t.Errorf("Style = %q, want %q", layout.Style, "mixed")
	}
}

// ── Detect Lint Config ──────────────────────────────────────────────────

func TestDetectLintConfig_Golangci(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		".golangci.yml": "linters:\n  enable:\n    - errcheck\n    - govet\n    - staticcheck\n",
	}, false)

	info := detectLintConfig(dir)

	if info == nil {
		t.Fatal("expected lint config")
	}
	if info.Tool != "golangci-lint" {
		t.Errorf("Tool = %q, want %q", info.Tool, "golangci-lint")
	}
	if len(info.Rules) < 2 {
		t.Errorf("Rules: expected at least 2, got %d", len(info.Rules))
	}
}

func TestDetectLintConfig_None(t *testing.T) {
	dir := t.TempDir()

	info := detectLintConfig(dir)
	if info != nil {
		t.Error("expected nil lint config for empty dir")
	}
}

// ── Detect CI Workflow ──────────────────────────────────────────────────

func TestDetectCIWorkflow_GitHubActions(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		".github/workflows/build.yml": "name: Build\non: [push]\njobs:\n  build:\n    steps:\n      - run: go build\n      - run: go test\n",
	}, false)

	info := detectCIWorkflow(dir)

	if info == nil {
		t.Fatal("expected CI workflow")
	}
	if info.System != "github-actions" {
		t.Errorf("System = %q, want %q", info.System, "github-actions")
	}
	if len(info.Steps) == 0 {
		t.Error("Steps: expected at least one step")
	}
}

func TestDetectCIWorkflow_GitLabCI(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		".gitlab-ci.yml": "stages:\n  - test\ntest:\n  script:\n    - go test ./...\n",
	}, false)

	info := detectCIWorkflow(dir)

	if info == nil {
		t.Fatal("expected CI workflow")
	}
	if info.System != "gitlab-ci" {
		t.Errorf("System = %q, want %q", info.System, "gitlab-ci")
	}
}

func TestDetectCIWorkflow_None(t *testing.T) {
	dir := t.TempDir()

	info := detectCIWorkflow(dir)
	if info != nil {
		t.Error("expected nil CI workflow for empty dir")
	}
}

// ── Integration: Pipeline includes conventions ──────────────────────────

func TestPipelineIncludesConventions(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":                       "module example.com/app\ngo 1.26\n",
		"main.go":                      "package main\nfunc main() { println(\"hello\") }\n",
		"main_test.go":                 "package main\nimport \"testing\"\nfunc TestMain(t *testing.T) {}\n",
		".golangci.yml":                "linters:\n  enable:\n    - errcheck\n",
		".github/workflows/ci.yml":     "name: CI\non: [push]\njobs:\n  test:\n    steps:\n      - run: go test\n",
	}, true)

	card, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if card.Conventions.TestStructure.Style == "" {
		t.Error("Conventions.TestStructure.Style should not be empty")
	}
	if card.Conventions.LintConfig == nil {
		t.Error("Conventions.LintConfig should not be nil")
	}
	if card.Conventions.CIWorkflow == nil {
		t.Error("Conventions.CIWorkflow should not be nil")
	}
}

// ── Golden fixture compatibility ─────────────────────────────────────────

func TestConventionsFieldInJSON(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":       "module example.com/app\ngo 1.26\n",
		"internal/pkg/foo.go": "package pkg\n",
	}, false)

	card, _ := Run(dir)
	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	conv, ok := m["conventions"].(map[string]any)
	if !ok {
		t.Fatal("conventions field missing or not an object")
	}
	if _, ok := conv["module_patterns"]; !ok {
		t.Error("conventions.module_patterns missing")
	}
	if _, ok := conv["test_structure"]; !ok {
		t.Error("conventions.test_structure missing")
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────

func TestDetectGoModulePatternExamples(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":                "module example.com/app\ngo 1.26\n",
		"cmd/server/main.go":    "package main\nfunc main() {}\n",
		"cmd/cli/main.go":       "package main\nfunc main() {}\n",
		"internal/handler/h.go": "package handler\n",
	}, false)

	patterns := detectModulePatterns(dir)

	for _, p := range patterns {
		if p.Name == "cmd" {
			found := false
			for _, ex := range p.Examples {
				if filepath.ToSlash(ex) == "cmd/server" || filepath.ToSlash(ex) == "cmd/cli" {
					found = true
				}
			}
			if !found {
				t.Errorf("cmd pattern: expected example with cmd/server or cmd/cli, got %v", p.Examples)
			}
		}
	}
}
