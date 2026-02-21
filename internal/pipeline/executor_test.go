package pipeline

import (
	"context"
	"os"
	"strings"
	"testing"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/federation"
)

func TestResolveRole(t *testing.T) {
	tests := []struct {
		name string
		task federation.FederatedTask
		want string
	}{
		{"builder from label", federation.FederatedTask{Issue: beads.Issue{Labels: []string{"workstream:builder"}}}, "builder"},
		{"generic from label", federation.FederatedTask{Issue: beads.Issue{Labels: []string{"workstream:generic"}}}, "generic"},
		{"default when no workstream", federation.FederatedTask{Issue: beads.Issue{Labels: []string{"autonomy"}}}, "builder"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveRole(tt.task)
			if got != tt.want {
				t.Errorf("resolveRole() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveModel(t *testing.T) {
	tests := []struct {
		name string
		task federation.FederatedTask
		want string
	}{
		{"glm from label", federation.FederatedTask{Issue: beads.Issue{Labels: []string{"model:glm-5"}}}, "glm-5"},
		{"default when no model", federation.FederatedTask{Issue: beads.Issue{Labels: []string{"autonomy"}}}, "glm-4.7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveModel(tt.task)
			if got != tt.want {
				t.Errorf("resolveModel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBaseBranch(t *testing.T) {
	orig := os.Getenv("SDP_REPO_BRANCH")
	defer func() { _ = os.Setenv("SDP_REPO_BRANCH", orig) }()

	_ = os.Unsetenv("SDP_REPO_BRANCH")
	if got := baseBranch(); got != "master" {
		t.Errorf("baseBranch() unset = %q, want master", got)
	}

	_ = os.Setenv("SDP_REPO_BRANCH", "main")
	if got := baseBranch(); got != "main" {
		t.Errorf("baseBranch() SDP_REPO_BRANCH=main = %q, want main", got)
	}
}

func TestExecuteTaskK8sModeFailsOutsideCluster(t *testing.T) {
	orig := os.Getenv("SDP_DISPATCH_MODE")
	defer func() { _ = os.Setenv("SDP_DISPATCH_MODE", orig) }()
	_ = os.Setenv("SDP_DISPATCH_MODE", "k8s")

	task := federation.FederatedTask{
		ProjectID: "test",
		Issue: beads.Issue{
			ID:    "test-abc",
			Title: "Test",
			Labels: []string{"autonomy", "strict-evidence", "workstream:builder"},
		},
		Workspace: t.TempDir(),
	}

	err := ExecuteTask(context.Background(), nil, task)
	if err == nil {
		t.Fatal("ExecuteTask(k8s) expected error when not in cluster")
	}
	// When not running in K8s, InClusterConfig fails with "unable to load in-cluster configuration"
	if !strings.Contains(err.Error(), "in-cluster") && !strings.Contains(err.Error(), "config") {
		t.Errorf("expected in-cluster config error, got: %v", err)
	}
}
