package telemetry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeEvidence(t *testing.T) {
	a := NewAnalyzer("/tmp", "glm-4.7", 5)

	ev := map[string]any{
		"intent": map[string]any{
			"objective": "Add feature X",
		},
		"execution": map[string]any{
			"changed_files": []any{"pkg/foo.go", "pkg/bar.go"},
		},
	}
	got := a.summarizeEvidence(ev)
	if got == "" {
		t.Error("expected non-empty summary")
	}
	if !strings.Contains(got, "Add feature X") {
		t.Errorf("summary missing objective: %q", got)
	}
	if !strings.Contains(got, "pkg/foo.go") {
		t.Errorf("summary missing changed file: %q", got)
	}
}

func TestSummarizeEvidence_Empty(t *testing.T) {
	a := NewAnalyzer("/tmp", "glm-4.7", 5)
	got := a.summarizeEvidence(map[string]any{})
	if got != "" {
		t.Errorf("expected empty summary, got %q", got)
	}
}

func TestWorkDirForProject(t *testing.T) {
	a := NewAnalyzer("/workspaces", "glm-4.7", 5)
	if got := a.workDirForProject(""); got != "/workspaces/default" {
		t.Errorf("empty project: got %q", got)
	}
	if got := a.workDirForProject("sdp_dev"); got != "/workspaces/sdp_dev" {
		t.Errorf("sdp_dev: got %q", got)
	}
}

func TestHandleClosed_NoEvidence(t *testing.T) {
	dir := t.TempDir()
	a := NewAnalyzer(dir, "glm-4.7", 5)
	created, err := a.HandleClosed(context.Background(), "nonexistent", "default")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("expected not created when evidence missing")
	}
}

func TestIsDuplicate(t *testing.T) {
	dir := t.TempDir()
	// Create a minimal beads repo so bd list works
	bdDir := filepath.Join(dir, "default")
	if err := os.MkdirAll(bdDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bdDir, ".git"), []byte("gitdir: ../.git"), 0644); err != nil {
		t.Skip("bd list requires git repo; skipping")
	}
	a := NewAnalyzer(dir, "glm-4.7", 5)
	// Without bd, isDuplicate returns false (cmd fails)
	dup := a.isDuplicate("Some proposal title", bdDir)
	if dup {
		t.Error("expected false when bd not available")
	}
}

func TestNewAnalyzer_Defaults(t *testing.T) {
	a := NewAnalyzer("/w", "", 0)
	if a.Model != "glm-4.7" {
		t.Errorf("default model: got %q", a.Model)
	}
	if a.MaxPerCycle != 5 {
		t.Errorf("default maxPerCycle: got %d", a.MaxPerCycle)
	}
}

func TestSummarizeEvidence_IntentOnly(t *testing.T) {
	a := NewAnalyzer("/tmp", "glm-4.7", 5)
	ev := map[string]any{"intent": map[string]any{"objective": "Only objective"}}
	got := a.summarizeEvidence(ev)
	if !strings.Contains(got, "Only objective") {
		t.Errorf("summary: %q", got)
	}
}

func TestSummarizeEvidence_ExecutionOnly(t *testing.T) {
	a := NewAnalyzer("/tmp", "glm-4.7", 5)
	ev := map[string]any{"execution": map[string]any{"changed_files": []any{"a.go"}}}
	got := a.summarizeEvidence(ev)
	if !strings.Contains(got, "a.go") {
		t.Errorf("summary: %q", got)
	}
}

func TestHandleClosed_InvalidJSONEvidence(t *testing.T) {
	dir := t.TempDir()
	evDir := filepath.Join(dir, "p1", ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evDir, "issue-1.json"), []byte(`{invalid}`), 0644); err != nil {
		t.Fatal(err)
	}
	a := NewAnalyzer(dir, "glm-4.7", 5)
	created, err := a.HandleClosed(context.Background(), "issue-1", "p1")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("expected not created for invalid JSON evidence")
	}
}

func TestEstimateCostUSD(t *testing.T) {
	if got := estimateCostUSD(0, 0); got != 0 {
		t.Errorf("estimateCostUSD(0,0) = %v, want 0", got)
	}
	if got := estimateCostUSD(1000, 500); got != 0.003 {
		t.Errorf("estimateCostUSD(1000,500) = %v, want 0.003", got)
	}
}

func TestHandleClosed_RateLimit(t *testing.T) {
	dir := t.TempDir()
	evDir := filepath.Join(dir, "p1", ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"intent": map[string]any{"objective": "Done"}}
	body, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "i1.json"), body, 0644); err != nil {
		t.Fatal(err)
	}
	a := NewAnalyzer(dir, "glm-4.7", 2)
	a.cycleCount = 2
	created, err := a.HandleClosed(context.Background(), "i1", "p1")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("expected not created when rate limit exceeded")
	}
}

func TestHandleClosed_SuccessWithAnalyzeFunc(t *testing.T) {
	dir := t.TempDir()
	evDir := filepath.Join(dir, "p1", ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"intent": map[string]any{"objective": "Done"}}
	body, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "i1.json"), body, 0644); err != nil {
		t.Fatal(err)
	}
	pathDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(pathDir, 0755); err != nil {
		t.Fatal(err)
	}
	fakeBd := filepath.Join(pathDir, "bd")
	if err := os.WriteFile(fakeBd, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", pathDir+string(filepath.ListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)
	a := NewAnalyzer(dir, "glm-4.7", 5)
	a.AnalyzeFunc = func(_ context.Context, _, _ string) (string, error) {
		return "Follow-up: Add tests", nil
	}
	created, err := a.HandleClosed(context.Background(), "i1", "p1")
	if err != nil {
		t.Fatalf("HandleClosed: %v", err)
	}
	if !created {
		t.Error("expected created when AnalyzeFunc returns proposal and bd succeeds")
	}
}

func TestHandleClosed_EmptyProposalFromAnalyzeFunc(t *testing.T) {
	dir := t.TempDir()
	evDir := filepath.Join(dir, "p1", ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"intent": map[string]any{"objective": "Done"}}
	body, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "i1.json"), body, 0644); err != nil {
		t.Fatal(err)
	}
	a := NewAnalyzer(dir, "glm-4.7", 5)
	a.AnalyzeFunc = func(_ context.Context, _, _ string) (string, error) {
		return "", nil
	}
	created, err := a.HandleClosed(context.Background(), "i1", "p1")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("expected not created when proposal empty")
	}
}

func TestIsDuplicate_WithDuplicate(t *testing.T) {
	dir := t.TempDir()
	pathDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(pathDir, 0755); err != nil {
		t.Fatal(err)
	}
	fakeBd := filepath.Join(pathDir, "bd")
	if err := os.WriteFile(fakeBd, []byte("#!/bin/sh\necho '1 - Add feature X'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", pathDir+string(filepath.ListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)
	workDir := filepath.Join(dir, "default")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}
	a := NewAnalyzer(dir, "glm-4.7", 5)
	dup := a.isDuplicate("Add feature X", workDir)
	if !dup {
		t.Error("expected true when bd returns matching title")
	}
}

// TestHandleClosed_WithEvidence_NoAPIKey: evidence exists, analyzeWithLLM returns fallback (no OPENROUTER_API_KEY), createBeadsIssue fails (no bd).
func TestHandleClosed_WithEvidence_NoAPIKey(t *testing.T) {
	dir := t.TempDir()
	evDir := filepath.Join(dir, "p1", ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{
		"intent":    map[string]any{"objective": "Done"},
		"execution": map[string]any{},
	}
	body, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "issue-2.json"), body, 0644); err != nil {
		t.Fatal(err)
	}
	a := NewAnalyzer(dir, "glm-4.7", 5)
	created, err := a.HandleClosed(context.Background(), "issue-2", "p1")
	// No OPENROUTER_API_KEY -> analyzeWithLLM returns fallback title; createBeadsIssue fails (no bd) -> err != nil
	if created {
		t.Error("expected not created when createBeadsIssue fails")
	}
	_ = err
}
