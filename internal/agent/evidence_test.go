package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/llm"
)

func TestEvidenceCollector_Initialize(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := map[string]any{
		"intent":     map[string]any{},
		"execution":  map[string]any{},
		"boundary":   map[string]any{},
		"provenance": map[string]any{},
		"trace":      map[string]any{},
	}
	tmplData, _ := json.Marshal(tmpl)
	if err := os.WriteFile(filepath.Join(specsDir, "strict-evidence-template.json"), tmplData, 0o644); err != nil {
		t.Fatal(err)
	}

	e := NewEvidenceCollector(dir)
	boundary := llm.BoundarySpec{
		AllowedPathPrefixes:   []string{"internal/"},
		ControlPathPrefixes:   []string{".sdp/"},
		ForbiddenPathPrefixes: []string{"vendor/"},
	}
	path, err := e.Initialize("issue-1", "main", "low", "glm-4", "coder", boundary)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	intent, _ := doc["intent"].(map[string]any)
	if intent["issue_id"] != "issue-1" || intent["risk_class"] != "low" {
		t.Errorf("intent: %v", intent)
	}
	exec, _ := doc["execution"].(map[string]any)
	if exec["branch"] != "main" {
		t.Errorf("execution: %v", exec)
	}
}

func TestEvidenceCollector_UpdateExecution(t *testing.T) {
	dir := t.TempDir()
	evDir := filepath.Join(dir, ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{
		"execution":  map[string]any{},
		"boundary": map[string]any{
			"observed":  map[string]any{},
			"compliance": map[string]any{},
		},
		"verification": map[string]any{},
	}
	data, _ := json.MarshalIndent(doc, "", "  ")
	path := filepath.Join(evDir, "issue-2.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	e := NewEvidenceCollector(dir)
	result := CollectResult{
		ChangedFiles: []string{"internal/foo.go"},
		TestsPassed:  true,
	}
	if err := e.UpdateExecution("issue-2", result); err != nil {
		t.Fatalf("UpdateExecution: %v", err)
	}

	data, _ = os.ReadFile(path)
	_ = json.Unmarshal(data, &doc)
	exec, _ := doc["execution"].(map[string]any)
	if exec["changed_files"].([]any)[0].(string) != "internal/foo.go" {
		t.Errorf("execution: %v", exec)
	}
	verif, _ := doc["verification"].(map[string]any)
	if verif["go_test_passed"] != true {
		t.Errorf("verification: %v", verif)
	}
}
