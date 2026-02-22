package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBoundary(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "specs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `workstreams:
  - label: workstream:test-ws
    path_prefixes:
      - internal/
      - cmd/
`
	if err := os.WriteFile(filepath.Join(cfgDir, "workstream-config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := LoadBoundary(root, "test-ws")
	if err != nil {
		t.Fatalf("LoadBoundary: %v", err)
	}
	if len(spec.AllowedPathPrefixes) != 2 {
		t.Errorf("AllowedPathPrefixes: want 2, got %d", len(spec.AllowedPathPrefixes))
	}
	if len(spec.ForbiddenPathPrefixes) == 0 {
		t.Error("ForbiddenPathPrefixes should include .git/")
	}
}

func TestLoadBoundaryBuilderFallback(t *testing.T) {
	root, _ := os.Getwd()
	for filepath.Base(root) != "sdp_dev" && len(root) > 1 {
		root = filepath.Dir(root)
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "workstream-config.yaml")); err != nil {
		t.Skip("workstream-config.yaml not found, skipping")
	}
	spec, err := LoadBoundary(root, "builder")
	if err != nil {
		t.Fatalf("LoadBoundary builder: %v", err)
	}
	if len(spec.AllowedPathPrefixes) == 0 {
		t.Error("builder should fallback to generic with path_prefixes")
	}
}

func TestValidateChangedPaths(t *testing.T) {
	spec := BoundarySpec{
		AllowedPathPrefixes:   []string{"internal/", "cmd/"},
		ControlPathPrefixes:   []string{".beads/", ".sdp/"},
		ForbiddenPathPrefixes: []string{".git/"},
	}
	if err := ValidateChangedPaths([]string{"internal/foo.go", "cmd/bar.go"}, spec); err != nil {
		t.Errorf("valid paths: %v", err)
	}
	if err := ValidateChangedPaths([]string{"internal/foo.go", ".git/config"}, spec); err == nil {
		t.Error("expected error for .git/ path")
	}
	if err := ValidateChangedPaths([]string{"docs/readme.md"}, spec); err == nil {
		t.Error("expected error for docs/ (not in allowed)")
	}
}
