package main

import (
	"strings"
	"testing"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/registry"
)

func TestResolveModel(t *testing.T) {
	if got := resolveModel([]string{"model:glm-5"}); got != "glm-5" {
		t.Errorf("resolveModel = %q, want glm-5", got)
	}
	if got := resolveModel([]string{}); got != "glm-4.7" {
		t.Errorf("resolveModel default = %q, want glm-4.7", got)
	}
}

func TestResolveWorkstream(t *testing.T) {
	if got := resolveWorkstream([]string{"workstream:builder"}); got != "builder" {
		t.Errorf("resolveWorkstream = %q, want builder", got)
	}
	if got := resolveWorkstream([]string{}); got != "builder" {
		t.Errorf("resolveWorkstream default = %q, want builder", got)
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
