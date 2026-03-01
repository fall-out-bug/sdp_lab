package workstream

import (
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/beads"
)

func TestNewWorkstreamTemplate(t *testing.T) {
	wt := NewWorkstreamTemplate(TemplateConfig{
		ProjectID: "00",
		FeatureID: "F061",
		OutputDir: "/tmp/test",
	})
	
	if wt.projectID != "00" {
		t.Errorf("projectID = %q, want %q", wt.projectID, "00")
	}
	if wt.featureID != "F061" {
		t.Errorf("featureID = %q, want %q", wt.featureID, "F061")
	}
}

func TestNewWorkstreamTemplateDefaults(t *testing.T) {
	wt := NewWorkstreamTemplate(TemplateConfig{})
	
	if wt.projectID != "00" {
		t.Errorf("projectID = %q, want %q", wt.projectID, "00")
	}
	if wt.outputDir != "docs/workstreams/backlog" {
		t.Errorf("outputDir = %q, want %q", wt.outputDir, "docs/workstreams/backlog")
	}
}

func TestGenerateWorkstream(t *testing.T) {
	tmpDir := t.TempDir()
	
	formula := &beads.Formula{
		Name:        "test-formula",
		Version:     "1.0",
		Description: "Test formula",
		Variables: map[string]beads.Variable{
			"name": {Default: "default-name"},
		},
		Steps: []beads.FormulaStep{
			{
				Name:        "step1",
				Title:       "Step 1: {{name}}",
				Description: "Description for {{name}}",
				Type:        "task",
				Size:        "M",
				ScopeFiles:  []string{"cmd/{{name}}.go"},
				AcceptanceCriteria: []string{
					"Implementation complete for {{name}}",
					"Tests pass",
				},
			},
			{
				Name:         "step2",
				Title:        "Step 2",
				Type:         "task",
				Size:         "L",
				Dependencies: []string{"step1"},
			},
		},
	}
	
	wt := NewWorkstreamTemplate(TemplateConfig{
		ProjectID: "00",
		FeatureID: "F999",
		OutputDir: tmpDir,
	})
	
	vars := map[string]string{"name": "myfeature"}
	
	files, err := wt.Generate(formula, vars)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	if len(files) != 2 {
		t.Fatalf("Generated %d files, want 2", len(files))
	}
	
	// Check first file
	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	
	contentStr := string(content)
	
	// Check variable substitution
	if !contains(contentStr, "Step 1: myfeature") {
		t.Error("Expected title to have myfeature substituted")
	}
	if !contains(contentStr, "Description for myfeature") {
		t.Error("Expected description to have myfeature substituted")
	}
	if !contains(contentStr, "cmd/myfeature.go") {
		t.Error("Expected scope file to have myfeature substituted")
	}
	if !contains(contentStr, "Implementation complete for myfeature") {
		t.Error("Expected AC to have myfeature substituted")
	}
	
	// Check second file has dependency
	content2, err := os.ReadFile(files[1])
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	
	if !contains(string(content2), "depends_on: [00-F999-01]") {
		t.Error("Expected second step to depend on first")
	}
}

func TestGenerateWorkstreamFeatureIDFromFormula(t *testing.T) {
	tmpDir := t.TempDir()
	
	formula := &beads.Formula{
		Name:    "my-feature",
		Version: "1.0",
		Steps: []beads.FormulaStep{
			{Name: "step1"},
		},
	}
	
	wt := NewWorkstreamTemplate(TemplateConfig{
		OutputDir: tmpDir,
		// No FeatureID set - should derive from formula name
	})
	
	files, err := wt.Generate(formula, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	// FeatureID should be derived as "FMYFEATURE"
	expectedPath := filepath.Join(tmpDir, "00-FMYFEATURE-01.md")
	if files[0] != expectedPath {
		t.Errorf("file = %q, want %q", files[0], expectedPath)
	}
}

func TestSubstituteVars(t *testing.T) {
	wt := NewWorkstreamTemplate(TemplateConfig{})
	
	vars := map[string]interface{}{
		"name": "auth",
		"module": "internal",
	}
	
	tests := []struct {
		input string
		want  string
	}{
		{"{{name}}", "auth"},
		{"{{.name}}", "auth"},
		{"prefix-{{name}}-suffix", "prefix-auth-suffix"},
		{"{{name}}/{{module}}", "auth/internal"},
		{"no vars", "no vars"},
	}
	
	for _, tt := range tests {
		got := wt.substituteVars(tt.input, vars)
		if got != tt.want {
			t.Errorf("substituteVars(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormulaHash(t *testing.T) {
	wt := NewWorkstreamTemplate(TemplateConfig{})
	
	formula := &beads.Formula{
		Name:    "test",
		Version: "1.0",
		Steps: []beads.FormulaStep{
			{Name: "step1"},
		},
	}
	
	hash := wt.formulaHash(formula)
	if len(hash) != 8 {
		t.Errorf("hash length = %d, want 8", len(hash))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
