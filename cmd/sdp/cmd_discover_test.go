package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/control"
	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func setupControlStore(t *testing.T) *control.Store {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := "projects:\n  - id: sdp\n    repo_url: https://github.com/test/sdp\n    beads_prefix: sdp\n"
	if err := os.WriteFile(filepath.Join(root, "docs", "specs", "project-registry.yaml"), []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}
	control.SetCreateBeadsIssueFn(control.MockCreateBeadsIssue("bd-test-123"))
	t.Cleanup(func() {
		control.SetCreateBeadsIssueFn(nil)
	})
	store, err := control.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestCreateFeatureCard_onGO(t *testing.T) {
	store := setupControlStore(t)

	frame := &discovery.FrameResult{
		ProblemStatement: "Users cannot track their goals",
		Scope:            "Goal tracking only; no social features",
	}
	hyp := &discovery.HypothesisResult{
		Requirements: []string{"Track goals", "Set reminders"},
	}
	discoveryDir := "/tmp/docs/discovery/test-feature"

	card, err := createFeatureCard(store, "sdp", "test-feature", frame, hyp, discoveryDir)
	if err != nil {
		t.Fatalf("createFeatureCard: %v", err)
	}
	if card.NormalizedIntent != frame.ProblemStatement {
		t.Errorf("NormalizedIntent = %q, want %q", card.NormalizedIntent, frame.ProblemStatement)
	}
	if card.DiscoveryDir != discoveryDir {
		t.Errorf("DiscoveryDir = %q, want %q", card.DiscoveryDir, discoveryDir)
	}
	if card.Status != "shaping" {
		t.Errorf("Status = %q, want %q", card.Status, "shaping")
	}
	if len(card.AcceptanceShape) != 2 {
		t.Errorf("AcceptanceShape = %v, want 2 items", card.AcceptanceShape)
	}
	if card.ID == "" {
		t.Error("card ID must not be empty")
	}
}

func TestCreateFeatureCard_nilHypothesis(t *testing.T) {
	store := setupControlStore(t)

	frame := &discovery.FrameResult{
		ProblemStatement: "Some problem",
		Scope:            "In scope",
	}
	discoveryDir := "/tmp/docs/discovery/x"

	card, err := createFeatureCard(store, "sdp", "x", frame, nil, discoveryDir)
	if err != nil {
		t.Fatalf("createFeatureCard: %v", err)
	}
	if len(card.AcceptanceShape) != 0 {
		t.Errorf("AcceptanceShape should be empty with nil hypothesis, got %v", card.AcceptanceShape)
	}
}

func TestCreateWorkstreamStub_onGO(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "docs/workstreams/backlog")

	frame := &discovery.FrameResult{
		ProblemStatement: "Users cannot track their goals",
		Scope:            "Goal tracking only",
	}
	hyp := &discovery.HypothesisResult{
		Requirements: []string{"Track goals", "Set reminders"},
	}
	discoveryDir := "docs/discovery/test-feature"
	wsID := "00-999-01"
	featureID := "F999"

	path, err := createWorkstreamStub(wsDir, wsID, featureID, frame, hyp, discoveryDir)
	if err != nil {
		t.Fatalf("createWorkstreamStub: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workstream file: %v", err)
	}
	s := string(content)

	if !strings.Contains(s, "ws_id: "+wsID) {
		t.Error("missing ws_id in frontmatter")
	}
	if !strings.Contains(s, "feature_id: "+featureID) {
		t.Error("missing feature_id in frontmatter")
	}
	if !strings.Contains(s, frame.ProblemStatement) {
		t.Error("missing ProblemStatement in content")
	}
	if !strings.Contains(s, discoveryDir) {
		t.Error("missing discoveryDir reference")
	}
	if !strings.Contains(s, "Track goals") {
		t.Error("missing acceptance criteria from requirements")
	}
	if !strings.Contains(s, frame.Scope) {
		t.Error("missing scope in Out of Scope section")
	}
}

func TestCreateWorkstreamStub_noRequirements(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "ws")

	frame := &discovery.FrameResult{
		ProblemStatement: "Some problem",
		Scope:            "In scope",
	}

	path, err := createWorkstreamStub(wsDir, "00-000-01", "F000", frame, nil, "/tmp/out")
	if err != nil {
		t.Fatalf("createWorkstreamStub: %v", err)
	}

	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "to be defined in planning phase") {
		t.Error("should have placeholder AC when no requirements")
	}
}
