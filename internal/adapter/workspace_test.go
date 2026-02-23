package adapter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectIDFromLabels(t *testing.T) {
	tests := []struct {
		labels map[string]string
		want   string
	}{
		{nil, ""},
		{map[string]string{}, ""},
		{map[string]string{LabelProject: "p1"}, "p1"},
		{map[string]string{"project": "p2"}, "p2"},
		{map[string]string{LabelProject: "a", "project": "b"}, "a"},
		{map[string]string{LabelProject: "  p3  "}, "p3"},
	}
	for _, tt := range tests {
		got := ProjectIDFromLabels(tt.labels)
		if got != tt.want {
			t.Errorf("ProjectIDFromLabels(%v) = %q, want %q", tt.labels, got, tt.want)
		}
	}
}

func TestNewWorkspaceResolver(t *testing.T) {
	resolver := NewWorkspaceResolver("/workspaces")
	if got := resolver(""); got != "/workspaces/default" {
		t.Errorf("resolver(\"\") = %q, want /workspaces/default", got)
	}
	if got := resolver("p1"); got != "/workspaces/p1" {
		t.Errorf("resolver(\"p1\") = %q, want /workspaces/p1", got)
	}
}

func TestCheckWorkspaceHealth_NoBeadsDir(t *testing.T) {
	dir := t.TempDir()
	h := CheckWorkspaceHealth(dir)
	if h.BeadsAvailable {
		t.Error("expected BeadsAvailable=false when .beads/ absent")
	}
	if h.Reason == "" {
		t.Error("expected Reason set")
	}
}

func TestCheckWorkspaceHealth_WithBeadsDir(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	h := CheckWorkspaceHealth(dir)
	// bd may or may not be in PATH; beads-fsm may or may not be in PATH
	// We only assert that if .beads exists, we get past that check
	// If bd is in PATH (e.g. in dev), BeadsAvailable will be true
	if h.Reason == ".beads/ absent or not a directory" {
		t.Error("expected .beads/ to be found")
	}
}

func TestBeadsFSMAvailable(t *testing.T) {
	// Just ensure it doesn't panic; result depends on test environment
	_ = BeadsFSMAvailable()
}
