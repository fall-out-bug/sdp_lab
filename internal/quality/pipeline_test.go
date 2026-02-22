package quality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/agent"
	"sdp_dev/internal/llm"
)

func TestRunTests_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	ok, err := RunTests(dir)
	if err != nil {
		t.Fatalf("RunTests err: %v", err)
	}
	if ok {
		t.Error("RunTests in empty dir should not pass (no go.mod)")
	}
}

func TestRunTests_ValidModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err := RunTests(dir)
	if err != nil {
		t.Fatalf("RunTests err: %v", err)
	}
	if !ok {
		t.Error("RunTests in valid minimal module should pass")
	}
}

func TestCollectEvidence_InitAndUpdate(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := map[string]any{
		"intent": map[string]any{}, "execution": map[string]any{},
		"boundary": map[string]any{}, "provenance": map[string]any{}, "trace": map[string]any{},
	}
	tmplData, _ := json.Marshal(tmpl)
	if err := os.WriteFile(filepath.Join(specsDir, "strict-evidence-template.json"), tmplData, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := EvidenceConfig{
		WorkDir:      dir,
		IssueID:      "test-1",
		Branch:       "worker/test-1",
		RiskClass:    "low",
		Model:        "glm-4.7",
		Role:         "builder",
		Boundary:     llm.BoundarySpec{},
		ChangedFiles: []string{"a.go"},
		ModelUsed:    "glm-4.7",
		TestsPassed:  true,
	}
	path, err := CollectEvidence(cfg)
	if err != nil {
		t.Fatalf("CollectEvidence: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("evidence file not created: %v", err)
	}
}

func TestUpdateEvidence(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := map[string]any{
		"intent": map[string]any{}, "execution": map[string]any{},
		"boundary": map[string]any{}, "provenance": map[string]any{}, "trace": map[string]any{},
	}
	tmplData, _ := json.Marshal(tmpl)
	if err := os.WriteFile(filepath.Join(specsDir, "strict-evidence-template.json"), tmplData, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = CollectEvidence(EvidenceConfig{
		WorkDir: dir, IssueID: "u1", Branch: "worker/u1", Model: "glm", Role: "builder",
		Boundary: llm.BoundarySpec{}, TestsPassed: false,
	})
	err := UpdateEvidence(dir, "u1", agent.CollectResult{ChangedFiles: []string{"x.go"}, ModelUsed: "glm", TestsPassed: true})
	if err != nil {
		t.Fatalf("UpdateEvidence: %v", err)
	}
}

func TestSignProvenance(t *testing.T) {
	cfg := ProvenanceConfig{
		AgentID:      "test-agent",
		Role:         "builder",
		IssueID:      "issue-1",
		ArtifactID:   "run-1:strict",
		Phase:        "completed",
		Payload:      map[string]any{"x": 1},
		ModelUsed:    "glm-4.7",
		TraceLink:    "",
		EvidenceLink: "",
	}
	env, err := SignProvenance(cfg)
	if err != nil {
		t.Fatalf("SignProvenance: %v", err)
	}
	if env.IssueID != "issue-1" || env.Phase != "completed" {
		t.Errorf("envelope: issue_id=%q phase=%q", env.IssueID, env.Phase)
	}
}

func TestRunPRGate_NoBinary(t *testing.T) {
	dir := t.TempDir()
	err := RunPRGate("x", dir)
	if err == nil {
		t.Error("RunPRGate without pr-gate binary should error")
	}
}

func TestTransitionFSM_NoBinary(t *testing.T) {
	dir := t.TempDir()
	err := TransitionFSM("x", "review", dir)
	if err == nil {
		t.Error("TransitionFSM without beads-fsm binary should error")
	}
}

func TestBaseBranch(t *testing.T) {
	orig := os.Getenv("SDP_REPO_BRANCH")
	defer func() { _ = os.Setenv("SDP_REPO_BRANCH", orig) }()

	_ = os.Unsetenv("SDP_REPO_BRANCH")
	if got := BaseBranch(); got != "master" {
		t.Errorf("BaseBranch() = %q, want master", got)
	}
	_ = os.Setenv("SDP_REPO_BRANCH", "main")
	if got := BaseBranch(); got != "main" {
		t.Errorf("BaseBranch() = %q, want main", got)
	}
}

func TestCommitAndPublish_NoGit(t *testing.T) {
	dir := t.TempDir()
	_, err := CommitAndPublish(PublishConfig{
		WorkDir:   dir,
		IssueID:   "x",
		Title:     "t",
		Changed:   []string{},
		BaseBranch: "master",
	})
	if err == nil {
		t.Error("CommitAndPublish in non-git dir should error")
	}
}
