package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/control"
	"sdp_dev/internal/kernel"
)

type mockInvoker struct {
	agent  string
	prompt string
	output string
	code   int
	err    error
}

func (m *mockInvoker) Invoke(_ context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	m.agent = req.Agent
	m.prompt = req.Prompt
	return kernel.RuntimeResult{Output: m.output, ExitCode: m.code}, m.err
}

func setupStore(t *testing.T) *control.Store {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := []byte("projects:\n  - id: openclaw\n    repo_url: https://github.com/openclaw/openclaw\n    beads_prefix: openclaw\n")
	if err := os.WriteFile(filepath.Join(root, "docs", "specs", "project-registry.yaml"), registry, 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := control.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func createDispatchedCard(t *testing.T, store *control.Store, role string) *control.FeatureCard {
	t.Helper()
	card, err := store.CreateCard("openclaw", "Test feature", "do the thing")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "executing"
	card.TargetRepo = "openclaw"
	card.NormalizedIntent = "do the thing"
	card.ScopeIn = []string{"ship code"}
	card.ScopeOut = []string{"tests green"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	packet := &control.ExecutionPacket{
		BeadsTaskID:       "openclaw-123",
		ParentFeatureID:   card.ID,
		ProjectID:         card.ProjectID,
		TargetRepo:        card.TargetRepo,
		ExecutorRole:      role,
		Objective:         "Implement the feature",
		ScopeIn:           []string{"edit code", "add tests"},
		ScopeOut:          []string{"green build"},
		Constraints:       []string{"no unrelated changes"},
		NextHandoffTarget: "orchestrator",
	}
	packetPath := filepath.Join(store.ControlRoot, "projects", card.ProjectID, "dispatches", card.ID+".json")
	if err := os.MkdirAll(filepath.Dir(packetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packetPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	card.DispatchedPacketPath = packetPath
	card.DispatchedTo = role
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}
	return card
}

func TestDispatchAndRunHappyPath(t *testing.T) {
	store := setupStore(t)
	card := createDispatchedCard(t, store, string(control.ExecutorRoleOmOImplementation))
	invoker := &mockInvoker{output: "done\ncommit 0123456789abcdef0123456789abcdef01234567\n", code: 0}
	bridge := &ExecutorBridge{Store: store, Invoker: invoker, ProjectRoot: store.ProjectRoot}

	result, err := bridge.DispatchAndRun(context.Background(), card.ProjectID, card.ID)
	if err != nil {
		t.Fatalf("DispatchAndRun error: %v", err)
	}
	if result.Status != control.ResultStatusSuccess {
		t.Fatalf("status = %s, want success", result.Status)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Reference != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("artifacts = %+v", result.Artifacts)
	}
	if invoker.agent != "implementer" {
		t.Fatalf("agent = %s, want implementer", invoker.agent)
	}
	if !strings.Contains(invoker.prompt, "Objective:") || !strings.Contains(invoker.prompt, "Constraints:") {
		t.Fatalf("prompt missing expected sections: %s", invoker.prompt)
	}

	updated, err := store.LoadCard(card.ProjectID, card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ExecutorRuntimeState != control.ExecutorRuntimeCompleted {
		t.Fatalf("runtime state = %s, want completed", updated.ExecutorRuntimeState)
	}
	if updated.ExecutorSessionID == "" || updated.ExecutorStartedAt == "" {
		t.Fatalf("missing executor session metadata: %+v", updated)
	}
	if updated.ExecutorResult == nil || updated.ExecutorResult.Status != string(control.ResultStatusSuccess) {
		t.Fatalf("executor result not recorded: %+v", updated.ExecutorResult)
	}
	if _, err := os.Stat(filepath.Join(store.ProjectRoot, ".sdp", "prompt-provenance.json")); err != nil {
		t.Fatalf("prompt provenance not written: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(store.ControlRoot, "executor-results", card.ID+"-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 result file, got %d", len(matches))
	}
}

func TestDispatchAndRunFailurePath(t *testing.T) {
	store := setupStore(t)
	card := createDispatchedCard(t, store, string(control.ExecutorRoleOmOImplementation))
	invoker := &mockInvoker{output: "tests failed", code: 2}
	bridge := &ExecutorBridge{Store: store, Invoker: invoker, ProjectRoot: store.ProjectRoot}

	result, err := bridge.DispatchAndRun(context.Background(), card.ProjectID, card.ID)
	if err != nil {
		t.Fatalf("DispatchAndRun error: %v", err)
	}
	if result.Status != control.ResultStatusFailed {
		t.Fatalf("status = %s, want failed", result.Status)
	}

	updated, err := store.LoadCard(card.ProjectID, card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ExecutorRuntimeState != control.ExecutorRuntimeFailed {
		t.Fatalf("runtime state = %s, want failed", updated.ExecutorRuntimeState)
	}
	if updated.ExecutorResult == nil || updated.ExecutorResult.Status != string(control.ResultStatusFailed) {
		t.Fatalf("executor result not recorded: %+v", updated.ExecutorResult)
	}
}

func TestDispatchAndRunRoleMappingReview(t *testing.T) {
	store := setupStore(t)
	card := createDispatchedCard(t, store, string(control.ExecutorRoleReview))
	invoker := &mockInvoker{output: "APPROVED", code: 0}
	bridge := &ExecutorBridge{Store: store, Invoker: invoker, ProjectRoot: store.ProjectRoot}

	_, err := bridge.DispatchAndRun(context.Background(), card.ProjectID, card.ID)
	if err != nil {
		t.Fatalf("DispatchAndRun error: %v", err)
	}
	if invoker.agent != "reviewer" {
		t.Fatalf("agent = %s, want reviewer", invoker.agent)
	}
}

func TestMapExecutorRoleDefault(t *testing.T) {
	if got := mapExecutorRole("something-else"); got != "implementer" {
		t.Fatalf("mapExecutorRole fallback = %s, want implementer", got)
	}
}
