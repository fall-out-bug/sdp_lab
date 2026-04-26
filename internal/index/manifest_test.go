package index

import (
	"os"
	"strings"
	"testing"
)

// ── Test Doubles ────────────────────────────────────────────────────

// memStore is an in-memory test double for the ManifestStore interface.
type memStore struct {
	modules     []ModuleMeta
	meta        map[string]string
	entryPoints []string
	stats       *IndexStats
}

func newMemStore() *memStore {
	return &memStore{
		meta: make(map[string]string),
	}
}

func (m *memStore) GetMeta(key string) (string, error) {
	return m.meta[key], nil
}

func (m *memStore) SetMeta(key, value string) error {
	m.meta[key] = value
	return nil
}

func (m *memStore) UpsertModuleMeta(mm ModuleMeta) error {
	for i, existing := range m.modules {
		if existing.Name == mm.Name {
			m.modules[i] = mm
			return nil
		}
	}
	m.modules = append(m.modules, mm)
	return nil
}

func (m *memStore) LoadModules() ([]ModuleMeta, error) {
	return m.modules, nil
}

func (m *memStore) LoadMeta(keys ...string) (map[string]string, error) {
	out := make(map[string]string)
	for _, k := range keys {
		if v, ok := m.meta[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func (m *memStore) UpdateModules(modules []ModuleMeta) error {
	m.modules = modules
	return nil
}

func (m *memStore) LoadEntryPoints() ([]string, error) {
	return m.entryPoints, nil
}

func (m *memStore) LoadStats() (*IndexStats, error) {
	return m.stats, nil
}

func (m *memStore) LoadMetaPrefix(prefix string) (map[string]string, error) {
	out := make(map[string]string)
	for k, v := range m.meta {
		if strings.HasPrefix(k, prefix) {
			out[k] = v
		}
	}
	return out, nil
}

// Verify memStore satisfies ManifestStore at compile time.
var _ ManifestStore = (*memStore)(nil)

// ── Manifest Generation Tests ──────────────────────────────────────

func TestManifestBasicGeneration(t *testing.T) {
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name":   "myapp",
		"languages":   "go,typescript",
		"arch_style":  "modular",
	}
	store.modules = []ModuleMeta{
		{Name: "api", Path: "internal/api", Purpose: "HTTP handlers", Loc: 1200, FilesCount: 5},
		{Name: "store", Path: "internal/store", Purpose: "Database layer", Loc: 800, Owner: "alice", BusFactor: 2, FilesCount: 3},
	}
	store.entryPoints = []string{"cmd/server/main.go", "cmd/cli/main.go"}

	md, err := GenerateManifest(store)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}

	// Verify header line
	if !strings.Contains(md, "# myapp") {
		t.Error("manifest should contain repo name in header")
	}
	if !strings.Contains(md, "go") {
		t.Error("manifest should mention primary language")
	}
	if !strings.Contains(md, "modular") {
		t.Error("manifest should mention arch style")
	}

	// Verify modules section
	if !strings.Contains(md, "## Modules") {
		t.Error("manifest should have Modules section")
	}
	if !strings.Contains(md, "api") || !strings.Contains(md, "store") {
		t.Error("manifest should list modules")
	}
	if !strings.Contains(md, "1200") {
		t.Error("manifest should contain LOC for modules")
	}

	// Verify entry points
	if !strings.Contains(md, "## Entry Points") {
		t.Error("manifest should have Entry Points section")
	}
	if !strings.Contains(md, "cmd/server/main.go") {
		t.Error("manifest should list entry points")
	}
}

func TestManifestWithEmptyModules(t *testing.T) {
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name": "empty-project",
		"languages": "go",
	}
	store.modules = nil

	md, err := GenerateManifest(store)
	if err != nil {
		t.Fatalf("GenerateManifest with empty modules: %v", err)
	}
	if !strings.Contains(md, "# empty-project") {
		t.Error("manifest should still have header even with no modules")
	}
}

func TestManifestModulesSortedByLOC(t *testing.T) {
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name": "sort-test",
		"languages": "go",
	}
	store.modules = []ModuleMeta{
		{Name: "small", Path: "internal/small", Loc: 100, FilesCount: 1},
		{Name: "large", Path: "internal/large", Loc: 5000, FilesCount: 10},
		{Name: "medium", Path: "internal/medium", Loc: 1000, FilesCount: 3},
	}

	md, err := GenerateManifest(store)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}

	// "large" should appear before "small" in the output
	largeIdx := strings.Index(md, "large")
	smallIdx := strings.Index(md, "small")
	if largeIdx == -1 || smallIdx == -1 {
		t.Fatal("manifest should contain both modules")
	}
	if largeIdx > smallIdx {
		t.Error("modules should be sorted by LOC descending; large should come before small")
	}
}

func TestManifestTokenBudget(t *testing.T) {
	// The manifest must stay under ~2K tokens. Rough heuristic:
	// ~4 chars per token, so ~8000 chars max.
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name":  "big-repo",
		"languages":  "go,typescript,python",
		"arch_style": "modular",
	}

	// Simulate a large repo with many modules
	modules := make([]ModuleMeta, 30)
	for i := 0; i < 30; i++ {
		modules[i] = ModuleMeta{
			Name:       "module_" + strings.Repeat("x", 5),
			Path:       "internal/mod_" + strings.Repeat("x", 5),
			Purpose:    "Does something important with a moderate description",
			Loc:        (30 - i) * 500,
			Owner:      "dev@example.com",
			BusFactor:  3,
			FilesCount: 5,
		}
	}
	store.modules = modules

	entryPoints := make([]string, 10)
	for i := range entryPoints {
		entryPoints[i] = "cmd/service_" + strings.Repeat("y", 5) + "/main.go"
	}
	store.entryPoints = entryPoints

	md, err := GenerateManifest(store)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}

	// Very rough check: under 10K chars (approximately 2.5K tokens)
	if len(md) > 10000 {
		t.Errorf("manifest too long: %d chars (target under ~8K for 2K token budget)", len(md))
	}
}

func TestManifestConventionsSection(t *testing.T) {
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name":    "conv-test",
		"languages":    "go",
		"commit_style": "conventional",
		"build_system": "go_modules",
	}
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 200, FilesCount: 2},
	}

	md, err := GenerateManifest(store)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}
	if !strings.Contains(md, "## Conventions") {
		t.Error("manifest should have Conventions section")
	}
}

func TestManifestActiveWorkSection(t *testing.T) {
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name":        "active-test",
		"languages":        "go",
		"last_commit_date": "2026-04-15",
		"last_author":      "alice",
		"active_branches":  "5",
	}
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 200, FilesCount: 2},
	}

	md, err := GenerateManifest(store)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}
	if !strings.Contains(md, "## Active Work") {
		t.Error("manifest should have Active Work section")
	}
	if !strings.Contains(md, "2026-04-15") {
		t.Error("manifest should show last commit date")
	}
	if !strings.Contains(md, "alice") {
		t.Error("manifest should show last author")
	}
}

func TestWriteManifestCreatesFile(t *testing.T) {
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name": "write-test",
		"languages": "go",
	}
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 100, FilesCount: 1},
	}

	dir := t.TempDir()
	path, err := WriteManifest(dir, store)
	if err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	if !strings.HasSuffix(path, "manifest.md") {
		t.Errorf("expected path ending in manifest.md, got %s", path)
	}

	// Verify file was written and has content
	data, err := readTestFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(data) == 0 {
		t.Error("manifest file should not be empty")
	}
	if !strings.Contains(data, "# write-test") {
		t.Error("written manifest should contain repo name")
	}
}

func TestWriteManifestCreatesDirectory(t *testing.T) {
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name": "dir-test",
		"languages": "go",
	}
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 100, FilesCount: 1},
	}

	dir := t.TempDir()
	nestedDir := dir + "/.sdp/deep"
	path, err := WriteManifest(nestedDir, store)
	if err != nil {
		t.Fatalf("WriteManifest with nested dir: %v", err)
	}
	if _, err := readTestFile(path); err != nil {
		t.Fatalf("manifest not readable: %v", err)
	}
}

func TestManifestWithEnrichmentData(t *testing.T) {
	store := newMemStore()
	store.meta = map[string]string{
		"repo_name":  "enriched",
		"languages":  "go",
		"arch_style": "microservices",
	}
	store.modules = []ModuleMeta{
		{Name: "api", Path: "internal/api", Loc: 2000, Owner: "bob", BusFactor: 3, FilesCount: 5},
		{Name: "db", Path: "internal/db", Purpose: "Database access layer", Loc: 800, FilesCount: 3},
	}
	store.entryPoints = []string{"cmd/server/main.go"}

	enrichment := &EnrichmentInput{
		ArchitectReport: &ArchitectEnrichment{
			ArchStyle: "modular",
			ModulePurposes: map[string]string{
				"internal/api": "REST API handlers and middleware",
			},
		},
		MetricsReport: &MetricsEnrichment{
			BusFactor: 2,
			ModuleRisks: []ModuleRiskEntry{
				{Module: "internal/db", BusFactor: 1, PrimaryAuthor: "alice"},
			},
		},
	}

	// Enrich first, then generate
	if err := Enrich(store, enrichment); err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	md, err := GenerateManifest(store)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}

	if !strings.Contains(md, "# enriched") {
		t.Error("manifest should have repo header")
	}
}

// readTestFile is a helper for tests to read file contents.
func readTestFile(path string) (string, error) {
	data, err := readFileForTest(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readFileForTest reads a file's contents for test assertions.
func readFileForTest(path string) ([]byte, error) {
	return os.ReadFile(path)
}
