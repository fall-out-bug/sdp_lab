package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore(StoreConfig{})
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
}

func TestStore_CRUD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")
	s := NewStore(StoreConfig{RegistryPath: path})

	p := &Project{ID: "p1", RepoURL: "https://github.com/org/repo", RepoBranch: "main"}
	if err := s.Create(p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, ok := s.Get("p1")
	if !ok || got == nil || got.ID != "p1" {
		t.Errorf("Get: got %v, ok=%v", got, ok)
	}

	list := s.List()
	if len(list) != 1 || list[0].ID != "p1" {
		t.Errorf("List: %v", list)
	}

	if err := s.Delete("p1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("p1"); ok {
		t.Error("Get after Delete should fail")
	}
}

func TestStore_Create_duplicate(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(StoreConfig{RegistryPath: filepath.Join(dir, "r.yaml")})
	p := &Project{ID: "dup", RepoURL: "x"}
	if err := s.Create(p); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(p); err == nil {
		t.Error("Create duplicate should fail")
	}
}

func TestStore_FindByIssueID(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(StoreConfig{RegistryPath: filepath.Join(dir, "r.yaml")})
	_ = s.Create(&Project{ID: "sdp_dev", BeadsPrefix: "sdp_dev", RepoURL: "https://github.com/org/sdp_dev", RepoBranch: "main"})
	_ = s.Create(&Project{ID: "opencode", BeadsPrefix: "opencode", RepoURL: "https://github.com/org/opencode", RepoBranch: "main"})

	proj, ok := s.FindByIssueID("sdp_dev-5l9.2")
	if !ok || proj == nil || proj.ID != "sdp_dev" {
		t.Errorf("FindByIssueID(sdp_dev-5l9.2) = %v, ok=%v", proj, ok)
	}
	proj, ok = s.FindByIssueID("opencode-abc")
	if !ok || proj == nil || proj.ID != "opencode" {
		t.Errorf("FindByIssueID(opencode-abc) = %v, ok=%v", proj, ok)
	}
	_, ok = s.FindByIssueID("unknown-1")
	if ok {
		t.Error("FindByIssueID(unknown-1) should not find")
	}
	_, ok = s.FindByIssueID("no-prefix")
	if ok {
		t.Error("FindByIssueID(no-prefix) should not find")
	}
}

func TestStore_Update_notFound(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(StoreConfig{RegistryPath: filepath.Join(dir, "r.yaml")})
	if err := s.Update(&Project{ID: "nonexistent"}); err == nil {
		t.Error("Update nonexistent should fail")
	}
}

func TestStore_Load_missing(t *testing.T) {
	s := NewStore(StoreConfig{RegistryPath: "/nonexistent/path/registry.yaml"})
	if err := s.Load(); err != nil {
		t.Errorf("Load (missing file): %v", err)
	}
	if len(s.List()) != 0 {
		t.Error("expected empty list after load of missing file")
	}
}

func TestStore_Load_exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")
	cfg := `projects:
  - id: proj1
    repo_url: https://github.com/a/b
    repo_branch: main
`
	if err := os.WriteFile(path, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(StoreConfig{RegistryPath: path})
	if err := s.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	list := s.List()
	if len(list) != 1 || list[0].ID != "proj1" {
		t.Errorf("List: %v", list)
	}
}

func TestStore_Load_forkFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")
	cfg := `projects:
  - id: fork-proj
    repo_url: https://github.com/org/fork
    repo_branch: main
    fork: true
    upstream_remote: origin
`
	if err := os.WriteFile(path, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(StoreConfig{RegistryPath: path})
	if err := s.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, ok := s.Get("fork-proj")
	if !ok || !p.Fork || p.UpstreamRemote != "origin" {
		t.Errorf("Get fork-proj: %+v, ok=%v", p, ok)
	}
}
