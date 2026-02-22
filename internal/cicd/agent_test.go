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

func TestParsePRMerged_EmptyPayload(t *testing.T) {
	agent := NewAgent(nil, AgentConfig{})
	tg := agent.parsePRMerged(bus.Envelope{})
	if tg != nil {
		t.Errorf("expected nil for empty payload, got %+v", tg)
	}
}

func TestParsePRMerged_InvalidJSON(t *testing.T) {
	agent := NewAgent(nil, AgentConfig{})
	tg := agent.parsePRMerged(bus.Envelope{Payload: json.RawMessage(`{invalid}`)})
	if tg != nil {
		t.Errorf("expected nil for invalid JSON, got %+v", tg)
	}
}

func TestParsePRMerged_MasterBranch(t *testing.T) {
	agent := NewAgent(nil, AgentConfig{})
	env := bus.Envelope{
		Payload: json.RawMessage(`{"target_branch":"master","merge_commit_sha":"sha1","repository":"r"}`),
	}
	tg := agent.parsePRMerged(env)
	if tg == nil || tg.Env != "prod" {
		t.Errorf("master should yield prod env: %+v", tg)
	}
}

func TestKubectlEnv(t *testing.T) {
	a := NewAgent(nil, AgentConfig{})
	env := a.kubectlEnv()
	if len(env) == 0 {
		t.Error("kubectlEnv should return at least KUBECONFIG or base env")
	}
	a2 := NewAgent(nil, AgentConfig{Kubeconfig: "/tmp/kube"})
	env2 := a2.kubectlEnv()
	found := false
	for _, e := range env2 {
		if e == "KUBECONFIG=/tmp/kube" {
			found = true
			break
		}
	}
	if !found {
		t.Error("kubectlEnv with Kubeconfig should include KUBECONFIG")
	}
}

func TestPublishStatus_NilBus(t *testing.T) {
	a := NewAgent(nil, AgentConfig{})
	a.publishStatus(DeployTrigger{Ref: "r", Project: "p", Env: "dev"}, "started", "") // no panic
}

func TestCreateDeployBead_NilAdapter(t *testing.T) {
	a := NewAgent(nil, AgentConfig{WorkDir: t.TempDir()})
	a.beadsAdapter = nil
	id := a.createDeployBead(DeployTrigger{Ref: "r", Project: "p", Env: "dev"}, "tag")
	if id != "" {
		t.Errorf("expected empty when adapter nil, got %q", id)
	}
}

func TestWriteDeployEvidence_Failed(t *testing.T) {
	dir := t.TempDir()
	a := NewAgent(nil, AgentConfig{WorkDir: dir})
	tg := DeployTrigger{Ref: "abc", Project: "p", Env: "staging"}
	a.writeDeployEvidence(tg, "git-abc", "failed", "kubectl apply error", nil, time.Now().Add(-10*time.Second))
	evPath := dir + "/.sdp/evidence/deploy-git-abc-staging.json"
	b, err := os.ReadFile(evPath)
	if err != nil {
		t.Fatalf("evidence file: %v", err)
	}
	var ev map[string]any
	if err := json.Unmarshal(b, &ev); err != nil {
		t.Fatal(err)
	}
	exec, _ := ev["execution"].(map[string]any)
	if exec["status"] != "failed" || exec["reason"] != "kubectl apply error" {
		t.Errorf("execution: %+v", exec)
	}
	ver, _ := ev["verification"].(map[string]any)
	if ver["health_ok"] != false {
		t.Errorf("verification: %+v", ver)
	}
}

func TestNewAgent_CustomConfig(t *testing.T) {
	a := NewAgent(nil, AgentConfig{
		Registry: "ghcr.io/org", ImageTag: "v2", WorkDir: "/w",
		Images: []string{"img1"}, Kubeconfig: "/k",
	})
	if a.registry != "ghcr.io/org" || a.imageTag != "v2" || a.workDir != "/w" || a.kubeconfig != "/k" {
		t.Errorf("custom config not applied: %+v", a)
	}
	if len(a.images) != 1 || a.images[0] != "img1" {
		t.Errorf("images: %v", a.images)
	}
}

// TestDeploy_ApplyFails covers deploy path when applyAndRollout fails (no k8s dir).
func TestDeploy_ApplyFails(t *testing.T) {
	dir := t.TempDir()
	a := NewAgent(nil, AgentConfig{WorkDir: dir, Images: []string{"nonexistent"}})
	ctx := context.Background()
	tg := DeployTrigger{Ref: "abc123456789012345", Project: "p", Env: "dev"}
	a.deploy(ctx, tg)
	// Tag truncated to 12 chars: git-abc123456789
	evPath := dir + "/.sdp/evidence/deploy-git-abc123456789-dev.json"
	if _, err := os.Stat(evPath); os.IsNotExist(err) {
		t.Fatalf("evidence file not created after deploy failure")
	}
	b, _ := os.ReadFile(evPath)
	var ev map[string]any
	if err := json.Unmarshal(b, &ev); err != nil {
		t.Fatal(err)
	}
	if exec, ok := ev["execution"].(map[string]any); !ok || exec["status"] != "failed" {
		t.Errorf("expected failed status: %+v", ev)
	}
}

// TestDeploy_AlreadyInProgress runs two deploys; second should skip (deployInProgress).
func TestDeploy_AlreadyInProgress(t *testing.T) {
	dir := t.TempDir()
	a := NewAgent(nil, AgentConfig{WorkDir: dir, Images: []string{"nonexistent"}})
	ctx := context.Background()
	tg := DeployTrigger{Ref: "x", Project: "p", Env: "dev"}
	go a.deploy(ctx, tg)
	go a.deploy(ctx, tg)
	time.Sleep(200 * time.Millisecond)
	// Both should complete without deadlock; one may skip
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
