package dispatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCapabilityProfile_ScoreFor_NilCapabilities(t *testing.T) {
	p := &CapabilityProfile{
		Harness:      "test",
		Provider:     "test",
		Model:        "test",
		Capabilities: nil,
	}
	got := p.ScoreFor("refactor", "go")
	if got != 0.0 {
		t.Errorf("ScoreFor with nil capabilities = %v, want 0.0", got)
	}
}

func TestNewProfileStore(t *testing.T) {
	store := NewProfileStore("/tmp/my-project")
	want := filepath.Join("/tmp/my-project", ".sdp", "dispatch", "profiles")
	if store.Dir != want {
		t.Errorf("NewProfileStore dir = %q, want %q", store.Dir, want)
	}
}

func TestProfileStore_Load_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := &ProfileStore{Dir: dir}

	_, err := store.Load("ghost", "nobody", "none")
	if err == nil {
		t.Error("Load() on missing file should return error, got nil")
	}
}

func TestProfileStore_Load_ParseError(t *testing.T) {
	dir := t.TempDir()
	store := &ProfileStore{Dir: dir}

	fname := filepath.Join(dir, "bad-prov-model.json")
	if err := os.WriteFile(fname, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := store.Load("bad", "prov", "model")
	if err == nil {
		t.Error("Load() on invalid JSON should return error, got nil")
	}
}

func TestProfileStore_Save_MkdirAll(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "deep", "nested", "profiles")
	store := &ProfileStore{Dir: dir}

	p := &CapabilityProfile{
		Harness:      "gpt",
		Provider:     "openai",
		Model:        "gpt4",
		Capabilities: map[string]CapabilityScore{"feature:go": {TestPassRate: 0.9, AvgDuration: 2.0, SampleCount: 3}},
	}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save() into deep dir error = %v", err)
	}

	loaded, err := store.Load("gpt", "openai", "gpt4")
	if err != nil {
		t.Fatalf("Load() after deep-dir Save error = %v", err)
	}
	if loaded.Model != "gpt4" {
		t.Errorf("loaded.Model = %q, want %q", loaded.Model, "gpt4")
	}
}

func TestProfileStore_Save_MkdirFails(t *testing.T) {
	base := t.TempDir()
	blockingFile := filepath.Join(base, "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("block"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	store := &ProfileStore{Dir: filepath.Join(blockingFile, "subdir")}
	p := &CapabilityProfile{
		Harness:      "x",
		Provider:     "y",
		Model:        "z",
		Capabilities: map[string]CapabilityScore{},
	}
	err := store.Save(p)
	if err == nil {
		t.Error("Save() with blocked mkdir should return error, got nil")
	}
}

func TestProfileStore_LoadAll_SkipsNonJSON(t *testing.T) {
	dir := t.TempDir()
	store := &ProfileStore{Dir: dir}

	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	p := &CapabilityProfile{
		Harness:      "claude",
		Provider:     "anthropic",
		Model:        "opus",
		Capabilities: map[string]CapabilityScore{},
	}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	all, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(all) != 1 {
		t.Errorf("LoadAll() = %d profiles, want 1 (skipped non-json and bad json)", len(all))
	}
}

func TestProfileStore_LoadAll_SubdirSkipped(t *testing.T) {
	dir := t.TempDir()
	store := &ProfileStore{Dir: dir}

	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("Mkdir error = %v", err)
	}

	p := &CapabilityProfile{
		Harness:      "h",
		Provider:     "p",
		Model:        "m",
		Capabilities: map[string]CapabilityScore{},
	}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	all, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(all) != 1 {
		t.Errorf("LoadAll() = %d profiles, want 1 (subdir skipped)", len(all))
	}
}

func TestProfileStore_LoadAll_ReadDirError(t *testing.T) {
	base := t.TempDir()
	blockingFile := filepath.Join(base, "notadir")
	if err := os.WriteFile(blockingFile, []byte("block"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	store := &ProfileStore{Dir: blockingFile}
	_, err := store.LoadAll()
	if err == nil {
		t.Error("LoadAll() on file-as-dir should return error, got nil")
	}
}
