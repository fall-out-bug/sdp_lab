package registry_test

import (
	"path/filepath"
	"testing"

	"sdp_dev/internal/registry"
)

func TestLoadProjectRegistryFromSpecs(t *testing.T) {
	path := filepath.Join("..", "..", "specs", "project-registry.yaml")
	s := registry.NewStore(registry.StoreConfig{RegistryPath: path})
	if err := s.Load(); err != nil {
		t.Fatalf("Load specs/project-registry.yaml: %v", err)
	}
	list := s.List()
	want := map[string]bool{"sdp_dev": true, "sdp": true, "opencode": true, "kubeopencode": true, "openclaw": true, "beads": true}
	if len(list) < 6 {
		t.Errorf("expected at least 6 projects, got %d: %v", len(list), list)
	}
	for _, p := range list {
		if want[p.ID] {
			delete(want, p.ID)
		}
	}
	if len(want) > 0 {
		t.Errorf("missing projects: %v", want)
	}
}
