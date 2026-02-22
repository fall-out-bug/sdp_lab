package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFeaturePRBody(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteFeaturePRBody(dir, "feat-1", []string{"sdp_dev-a1", "sdp_dev-a2"})
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	expected := filepath.Join(dir, ".sdp", "pr-body-feature-feat-1.md")
	if path != expected {
		t.Errorf("path = %s, want %s", path, expected)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, "feat-1") || !strings.Contains(content, "sdp_dev-a1") || !strings.Contains(content, "sdp_dev-a2") {
		t.Errorf("content: %s", content)
	}
}

func TestWriteFeaturePRBody_EmptySubtasks(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteFeaturePRBody(dir, "feat-x", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "feat-x") {
		t.Errorf("content: %s", b)
	}
}
