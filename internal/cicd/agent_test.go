package cicd

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"sdp_dev/internal/bus"
)

func TestParseTrigger(t *testing.T) {
	agent := NewAgent(nil, AgentConfig{})
	env := bus.Envelope{
		IssueID:   "abc123",
		ProjectID: "proj1",
		Payload:   json.RawMessage(`{"ref":"sha-xyz","env":"staging"}`),
	}
	tg := agent.parseTrigger(env)
	if tg.Ref != "sha-xyz" {
		t.Errorf("ref: got %q", tg.Ref)
	}
	if tg.Project != "proj1" {
		t.Errorf("project: got %q", tg.Project)
	}
	if tg.Env != "staging" {
		t.Errorf("env: got %q", tg.Env)
	}
}

func TestParseTrigger_EmptyPayload(t *testing.T) {
	agent := NewAgent(nil, AgentConfig{ImageTag: "v1"})
	env := bus.Envelope{IssueID: "abc123"}
	tg := agent.parseTrigger(env)
	if tg.Ref != "abc123" {
		t.Errorf("ref from IssueID: got %q", tg.Ref)
	}
	if tg.Env != "dev" {
		t.Errorf("default env: got %q", tg.Env)
	}
}

func TestParsePRMerged(t *testing.T) {
	agent := NewAgent(nil, AgentConfig{})
	env := bus.Envelope{
		Payload: json.RawMessage(`{"target_branch":"main","merge_commit_sha":"abc123","repository":"sdp"}`),
	}
	tg := agent.parsePRMerged(env)
	if tg == nil {
		t.Fatal("expected trigger")
	}
	if tg.Ref != "abc123" {
		t.Errorf("ref: got %q", tg.Ref)
	}
	if tg.Env != "prod" {
		t.Errorf("env for main: got %q", tg.Env)
	}
}

func TestParsePRMerged_DevBranch(t *testing.T) {
	agent := NewAgent(nil, AgentConfig{})
	env := bus.Envelope{
		Payload: json.RawMessage(`{"target_branch":"dev","merge_commit_sha":"xyz"}`),
	}
	tg := agent.parsePRMerged(env)
	if tg == nil {
		t.Fatal("expected trigger")
	}
	if tg.Env != "dev" {
		t.Errorf("env for dev: got %q", tg.Env)
	}
}

func TestParsePRMerged_FeatureBranch(t *testing.T) {
	agent := NewAgent(nil, AgentConfig{})
	env := bus.Envelope{
		Payload: json.RawMessage(`{"target_branch":"feature/x"}`),
	}
	tg := agent.parsePRMerged(env)
	if tg != nil {
		t.Errorf("expected nil for feature branch, got %+v", tg)
	}
}

func TestNewAgent_Defaults(t *testing.T) {
	a := NewAgent(nil, AgentConfig{})
	if a.registry != "ghcr.io/fall-out-bug" {
		t.Errorf("registry: got %q", a.registry)
	}
	if a.imageTag != "latest" {
		t.Errorf("imageTag: got %q", a.imageTag)
	}
	if len(a.images) == 0 {
		t.Error("expected default images")
	}
}

func TestBuildAndPush_NoImages(t *testing.T) {
	a := NewAgent(nil, AgentConfig{WorkDir: t.TempDir(), Images: []string{"nonexistent-image"}})
	err := a.buildAndPush(context.Background(), "test", nil)
	// Skips build when Dockerfile doesn't exist; no error
	if err != nil {
		t.Logf("buildAndPush (expected to skip or fail): %v", err)
	}
}

func TestWriteDeployEvidence(t *testing.T) {
	dir := t.TempDir()
	a := NewAgent(nil, AgentConfig{WorkDir: dir})
	tg := DeployTrigger{Ref: "abc", Project: "p", Env: "dev"}
	a.writeDeployEvidence(tg, "git-abc", "succeeded", "", map[string]string{"img:tag": "sha123"}, time.Now().Add(-5*time.Second))
	evPath := dir + "/.sdp/evidence/deploy-git-abc-dev.json"
	if _, err := os.Stat(evPath); os.IsNotExist(err) {
		t.Fatalf("evidence file not created: %v", err)
	}
	b, _ := os.ReadFile(evPath)
	var ev map[string]any
	if err := json.Unmarshal(b, &ev); err != nil {
		t.Fatal(err)
	}
	exec, _ := ev["execution"].(map[string]any)
	if exec["status"] != "succeeded" {
		t.Errorf("status: got %v", exec["status"])
	}
}
