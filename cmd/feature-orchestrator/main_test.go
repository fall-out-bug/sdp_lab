package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/adapter"
	"sdp_dev/internal/beads"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/registry"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestResolveModel(t *testing.T) {
	if got := resolveModel([]string{"model:glm-5"}); got != "glm-5" {
		t.Errorf("resolveModel = %q, want glm-5", got)
	}
	if got := resolveModel([]string{}); got != "glm-4.7" {
		t.Errorf("resolveModel default = %q, want glm-4.7", got)
	}
	// Multiple model: labels — first wins (sdp_dev-f4c)
	if got := resolveModel([]string{"model:glm-5", "model:glm-4"}); got != "glm-5" {
		t.Errorf("resolveModel multiple labels = %q, want glm-5 (first wins)", got)
	}
	if got := resolveModel([]string{"other", "model:gpt-4"}); got != "gpt-4" {
		t.Errorf("resolveModel with other label = %q, want gpt-4", got)
	}
}

func TestResolveWorkstream(t *testing.T) {
	if got := resolveWorkstream([]string{"workstream:builder"}); got != "builder" {
		t.Errorf("resolveWorkstream = %q, want builder", got)
	}
	if got := resolveWorkstream([]string{}); got != "builder" {
		t.Errorf("resolveWorkstream default = %q, want builder", got)
	}
	// Multiple workstream: labels — first wins (sdp_dev-f4c)
	if got := resolveWorkstream([]string{"workstream:analyst", "workstream:coder"}); got != "analyst" {
		t.Errorf("resolveWorkstream multiple labels = %q, want analyst (first wins)", got)
	}
}

func TestParseProjectFilter(t *testing.T) {
	f := parseProjectFilter("a, b , c")
	if !f["a"] || !f["b"] || !f["c"] {
		t.Errorf("parseProjectFilter: %v", f)
	}
	f = parseProjectFilter("")
	if len(f) != 0 {
		t.Errorf("parseProjectFilter empty: %v", f)
	}
}

func TestBuildAgentRun(t *testing.T) {
	task := &federation.FederatedTask{
		ProjectID: "p1",
		Issue:     beads.Issue{ID: "p1-abc", Title: "Test", Labels: []string{"workstream:builder", "model:glm-4.7"}},
		Workspace: "/ws/p1",
	}
	proj := &registry.Project{
		ID:         "p1",
		RepoURL:    "https://github.com/org/repo",
		RepoBranch: "main",
	}
	run := buildAgentRun(task, proj, "sdp-workers")
	if run.Name == "" {
		t.Error("buildAgentRun: empty name")
	}
	if !strings.HasPrefix(run.Name, "ar-") {
		t.Errorf("buildAgentRun name should start with ar-: %s", run.Name)
	}
	if run.Spec.IssueID != "p1-abc" {
		t.Errorf("buildAgentRun IssueID = %q", run.Spec.IssueID)
	}
	if run.Spec.Model != "glm-4.7" {
		t.Errorf("buildAgentRun Model = %q", run.Spec.Model)
	}
	if run.Spec.Workstream != "builder" {
		t.Errorf("buildAgentRun Workstream = %q", run.Spec.Workstream)
	}
	if run.Namespace != "sdp-workers" {
		t.Errorf("buildAgentRun Namespace = %q", run.Namespace)
	}
}

func TestBuildAgentRun_ForkUpstreamURL(t *testing.T) {
	task := &federation.FederatedTask{
		ProjectID: "p1",
		Issue:     beads.Issue{ID: "i1", Title: "T", Labels: []string{}},
		Workspace: "/ws/p1",
	}
	proj := &registry.Project{
		ID:            "p1",
		RepoURL:       "https://github.com/fork/repo",
		UpstreamURL:   "https://github.com/upstream/repo",
		RepoBranch:    "main",
		Fork:          true,
		UpstreamRemote: "upstream",
	}
	run := buildAgentRun(task, proj, "ns")
	if run.Spec.Repo != proj.UpstreamURL {
		t.Errorf("Fork+UpstreamURL: Repo = %q, want %q", run.Spec.Repo, proj.UpstreamURL)
	}
}

func TestBuildAgentRun_BaseBranchDefault(t *testing.T) {
	task := &federation.FederatedTask{
		ProjectID: "p1",
		Issue:     beads.Issue{ID: "i1", Title: "T", Labels: []string{}},
		Workspace: "/ws/p1",
	}
	proj := &registry.Project{ID: "p1", RepoURL: "https://github.com/org/repo", RepoBranch: ""}
	run := buildAgentRun(task, proj, "ns")
	if run.Spec.BaseBranch != "main" {
		t.Errorf("empty RepoBranch: BaseBranch = %q, want main", run.Spec.BaseBranch)
	}
}

func TestBuildAgentRun_ModelAllowlist(t *testing.T) {
	task := &federation.FederatedTask{
		ProjectID: "p1",
		Issue:     beads.Issue{ID: "i1", Title: "T", Labels: []string{"model:evil-unknown-model"}},
		Workspace: "/ws/p1",
	}
	proj := &registry.Project{ID: "p1", RepoURL: "https://github.com/org/repo", RepoBranch: "main"}
	run := buildAgentRun(task, proj, "ns")
	if run.Spec.Model != "glm-5" {
		t.Errorf("disallowed model should fallback to DefaultModel, got %q", run.Spec.Model)
	}
}

func TestDispatchConfig(t *testing.T) {
	cfg := DispatchConfig{
		Namespace: "sdp-workers",
		Max:       3,
		Filter:    map[string]bool{"p1": true},
	}
	if cfg.Namespace != "sdp-workers" || cfg.Max != 3 || !cfg.Filter["p1"] {
		t.Errorf("DispatchConfig: %+v", cfg)
	}
}

func TestAgentRunName(t *testing.T) {
	tests := []struct {
		projectID, issueID string
		wantPrefix          string
		maxLen              int
	}{
		{"p1", "sdp_dev-4pg", "ar-p1-sdp_dev-4pg", 63},
		{"P1", "ABC.1", "ar-p1-abc-1", 63},
		{"proj", "x", "ar-proj-x", 63},
	}
	for _, tt := range tests {
		got := agentRunName(tt.projectID, tt.issueID)
		if !strings.HasPrefix(got, "ar-") {
			t.Errorf("agentRunName(%q, %q) = %q, want prefix ar-", tt.projectID, tt.issueID, got)
		}
		if len(got) > tt.maxLen {
			t.Errorf("agentRunName(%q, %q) = %q, length %d > %d", tt.projectID, tt.issueID, got, len(got), tt.maxLen)
		}
		for _, r := range got {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				t.Errorf("agentRunName(%q, %q) = %q has invalid rune %q", tt.projectID, tt.issueID, got, r)
			}
		}
	}
	// Long name is truncated to 63
	long := agentRunName("project", strings.Repeat("a", 70))
	if len(long) != 63 {
		t.Errorf("long name: len = %d, want 63", len(long))
	}
}

func TestEnvHelpers(t *testing.T) {
	// envDuration
	os.Unsetenv("SDP_POLL_INTERVAL")
	if got := envDuration("SDP_POLL_INTERVAL", 30*time.Second); got != 30*time.Second {
		t.Errorf("envDuration unset: got %v", got)
	}
	os.Setenv("SDP_POLL_INTERVAL", "1m")
	if got := envDuration("SDP_POLL_INTERVAL", 30*time.Second); got != time.Minute {
		t.Errorf("envDuration 1m: got %v", got)
	}
	os.Unsetenv("SDP_POLL_INTERVAL")

	// envInt
	os.Unsetenv("SDP_MAX_CONCURRENT")
	if got := envInt("SDP_MAX_CONCURRENT", 3); got != 3 {
		t.Errorf("envInt unset: got %d", got)
	}
	os.Setenv("SDP_MAX_CONCURRENT", "5")
	if got := envInt("SDP_MAX_CONCURRENT", 3); got != 5 {
		t.Errorf("envInt 5: got %d", got)
	}
	os.Unsetenv("SDP_MAX_CONCURRENT")

	// envStr
	os.Unsetenv("SDP_AGENTRUN_NAMESPACE")
	if got := envStr("SDP_AGENTRUN_NAMESPACE", "sdp-workers"); got != "sdp-workers" {
		t.Errorf("envStr unset: got %q", got)
	}
	os.Setenv("SDP_AGENTRUN_NAMESPACE", "custom-ns")
	if got := envStr("SDP_AGENTRUN_NAMESPACE", "sdp-workers"); got != "custom-ns" {
		t.Errorf("envStr custom: got %q", got)
	}
	os.Unsetenv("SDP_AGENTRUN_NAMESPACE")
}

func TestDispatch_noTasks(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	regPath := filepath.Join(dir, "reg.yaml")
	_ = os.WriteFile(regPath, []byte("projects:\n  - id: p1\n    repo_url: .\n    repo_branch: main\n"), 0o644)
	store := registry.NewStore(registry.StoreConfig{RegistryPath: regPath})
	_ = store.Load()
	ws := federation.NewWorkspaceManager(dir)
	agg := federation.NewAggregator(nil, store, ws)
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	lockMgr := adapter.NewRunLockManager(filepath.Join(dir, "locks"))

	dispatch(ctx, DispatchConfig{
		K8s:       k8s,
		Agg:       agg,
		LockMgr:   lockMgr,
		Store:     store,
		Filter:    nil,
		Namespace: "sdp-workers",
		Max:       2,
	})

	var list v1alpha1.AgentRunList
	if err := k8s.List(ctx, &list, client.InNamespace("sdp-workers")); err != nil {
		t.Fatalf("list AgentRuns: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 AgentRuns when agg has no tasks, got %d", len(list.Items))
	}
}

// TestDispatch_createsAgentRun is an integration test for the dispatch loop with mock k8s and agg (zdr).
func TestDispatch_createsAgentRun(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	regPath := filepath.Join(dir, "reg.yaml")
	_ = os.WriteFile(regPath, []byte("projects:\n  - id: p1\n    repo_url: .\n    repo_branch: main\n"), 0o644)
	store := registry.NewStore(registry.StoreConfig{RegistryPath: regPath})
	_ = store.Load()
	ws := federation.NewWorkspaceManager(dir)
	agg := federation.NewAggregator(nil, store, ws)
	agg.InjectReadySnapshot("p1", []beads.Issue{
		{ID: "sdp_dev-1", Title: "Test task", Priority: 1},
	})
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	lockMgr := adapter.NewRunLockManager(filepath.Join(dir, "locks"))

	dispatch(ctx, DispatchConfig{
		K8s:       k8s,
		Bus:       nil,
		Agg:       agg,
		LockMgr:   lockMgr,
		Store:     store,
		Filter:    nil,
		Namespace: "sdp-workers",
		Max:       2,
	})

	var list v1alpha1.AgentRunList
	if err := k8s.List(ctx, &list, client.InNamespace("sdp-workers")); err != nil {
		t.Fatalf("list AgentRuns: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 AgentRun, got %d", len(list.Items))
	}
	run := &list.Items[0]
	if run.Spec.IssueID != "sdp_dev-1" || run.Spec.Repo != "." {
		t.Errorf("AgentRun spec: IssueID=%q Repo=%q", run.Spec.IssueID, run.Spec.Repo)
	}
	if run.Namespace != "sdp-workers" {
		t.Errorf("AgentRun namespace: %q", run.Namespace)
	}
}
