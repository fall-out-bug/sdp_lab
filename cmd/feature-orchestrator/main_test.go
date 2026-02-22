package main

import (
	"context"
	"strings"
	"testing"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/registry"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"sdp_dev/api/v1alpha1"
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
}

func TestParseProjectFilter(t *testing.T) {
	f := parseProjectFilter("")
	if len(f) != 0 {
		t.Errorf("empty: got %v", f)
	}
	f = parseProjectFilter("p1,p2, p3 ")
	if !f["p1"] || !f["p2"] || !f["p3"] {
		t.Errorf("filter: got %v", f)
	}
}

func TestAgentRunName(t *testing.T) {
	n := agentRunName("proj1", "sdp_dev-abc")
	if !strings.HasPrefix(n, "ar-") {
		t.Errorf("name should start with ar-: %s", n)
	}
	if strings.Contains(n, ".") || strings.Contains(n, "_") {
		t.Errorf("name should be DNS-1123: %s", n)
	}
	if len(n) > 63 {
		t.Errorf("name too long: %d", len(n))
	}
}

func TestResolveModel(t *testing.T) {
	if got := resolveModel([]string{"a", "model:glm-5", "b"}); got != "glm-5" {
		t.Errorf("model: got %q", got)
	}
	if got := resolveModel([]string{}); got != "glm-4.7" {
		t.Errorf("default model: got %q", got)
	}
}

func TestResolveWorkstream(t *testing.T) {
	if got := resolveWorkstream([]string{"workstream:builder"}); got != "builder" {
		t.Errorf("workstream: got %q", got)
	}
	if got := resolveWorkstream([]string{}); got != "builder" {
		t.Errorf("default workstream: got %q", got)
	}
}

func TestDispatch_NoTasks(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := registry.NewStore(registry.StoreConfig{})
	_ = store.Load()
	agg := federation.NewAggregator(nil, store, federation.NewWorkspaceManager("/tmp"))
	lockMgr := &mockLock{}
	dispatch(context.Background(), DispatchConfig{
		K8s:       k8s,
		Bus:       nil,
		Agg:       agg,
		LockMgr:   lockMgr,
		Store:     store,
		Filter:    map[string]bool{},
		Namespace: "ns",
		Max:       3,
	})
}

type mockLock struct{}

func (m *mockLock) TryAcquire(issueID, key string) (string, bool, error) {
	return "", false, nil
}
func (m *mockLock) Release(issueID string) error { return nil }

func TestBuildAgentRun(t *testing.T) {
	task := &federation.FederatedTask{
		ProjectID: "p1",
		Issue:     beads.Issue{ID: "sdp_dev-x1", Labels: []string{"model:glm-5", "workstream:builder"}},
	}
	proj := &registry.Project{
		ID:        "p1",
		RepoURL:   "https://github.com/org/repo",
		RepoBranch: "main",
	}
	run := buildAgentRun(task, proj, "ns")
	if run.Spec.IssueID != "sdp_dev-x1" {
		t.Errorf("IssueID: got %q", run.Spec.IssueID)
	}
	if run.Spec.Model != "glm-5" {
		t.Errorf("Model: got %q", run.Spec.Model)
	}
	if run.Namespace != "ns" {
		t.Errorf("Namespace: got %q", run.Namespace)
	}
	if run.Labels["project"] != "p1" {
		t.Errorf("Labels: %v", run.Labels)
	}
}

func TestDispatch_FilterExcludes(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := registry.NewStore(registry.StoreConfig{})
	_ = store.Load()
	agg := federation.NewAggregator(nil, store, federation.NewWorkspaceManager("/tmp"))
	lockMgr := &mockLock{}
	dispatch(context.Background(), DispatchConfig{
		K8s:       k8s,
		Bus:       nil,
		Agg:       agg,
		LockMgr:   lockMgr,
		Store:     store,
		Filter:    map[string]bool{"only-this": true},
		Namespace: "ns",
		Max:       3,
	})
}

func TestEnvDuration(t *testing.T) {
	if d := envDuration("NONEXISTENT", 0); d != 0 {
		t.Errorf("envDuration default: got %v", d)
	}
}

func TestEnvInt(t *testing.T) {
	if n := envInt("NONEXISTENT", 42); n != 42 {
		t.Errorf("envInt default: got %d", n)
	}
}

func TestEnvStr(t *testing.T) {
	if s := envStr("NONEXISTENT", "def"); s != "def" {
		t.Errorf("envStr default: got %q", s)
	}
}
