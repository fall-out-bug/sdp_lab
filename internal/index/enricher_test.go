package index

import (
	"errors"
	"strings"
	"testing"
)

func TestEnrichWithNilInput(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 500, FilesCount: 2},
	}

	err := Enrich(store, nil)
	if err != nil {
		t.Fatalf("Enrich with nil input should not error: %v", err)
	}

	// Modules should be unchanged
	mods, _ := store.LoadModules()
	if len(mods) != 1 || mods[0].Name != "core" {
		t.Error("modules should be unchanged after nil enrichment")
	}
}

func TestEnrichWithEmptyInput(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 500, FilesCount: 2},
	}

	err := Enrich(store, &EnrichmentInput{})
	if err != nil {
		t.Fatalf("Enrich with empty input should not error: %v", err)
	}
}

func TestEnrichWithArchitectReport(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "api", Path: "internal/api", Loc: 1000, FilesCount: 3},
		{Name: "store", Path: "internal/store", Loc: 800, FilesCount: 2},
	}
	store.meta = map[string]string{
		"repo_name": "test",
		"languages": "go",
	}

	enrichment := &EnrichmentInput{
		ArchitectReport: &ArchitectEnrichment{
			ArchStyle: "modular",
			ModulePurposes: map[string]string{
				"internal/api":   "REST API handlers and routing",
				"internal/store": "Data persistence and query layer",
			},
		},
	}

	err := Enrich(store, enrichment)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	mods, _ := store.LoadModules()
	if len(mods) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(mods))
	}

	// Find the api module
	var apiMod *ModuleMeta
	for i := range mods {
		if mods[i].Name == "api" {
			apiMod = &mods[i]
			break
		}
	}
	if apiMod == nil {
		t.Fatal("api module not found")
	}
	if apiMod.Purpose != "REST API handlers and routing" {
		t.Errorf("api module purpose not enriched, got %q", apiMod.Purpose)
	}

	// Verify arch_style was stored in meta
	v, _ := store.GetMeta("arch_style")
	if v != "modular" {
		t.Errorf("arch_style meta not set, got %q", v)
	}
}

func TestEnrichWithMetricsReport(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 2000, FilesCount: 5},
		{Name: "utils", Path: "internal/utils", Loc: 300, FilesCount: 2},
	}
	store.meta = map[string]string{
		"repo_name": "test",
		"languages": "go",
	}

	enrichment := &EnrichmentInput{
		MetricsReport: &MetricsEnrichment{
			BusFactor: 2,
			ModuleRisks: []ModuleRiskEntry{
				{Module: "internal/core", BusFactor: 1, PrimaryAuthor: "alice"},
				{Module: "internal/utils", BusFactor: 4, PrimaryAuthor: "bob"},
			},
			CommitStyle:    "conventional",
			ActiveBranches: 7,
		},
	}

	err := Enrich(store, enrichment)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	mods, _ := store.LoadModules()

	// Find core module and check bus factor + owner
	var coreMod *ModuleMeta
	for i := range mods {
		if mods[i].Name == "core" {
			coreMod = &mods[i]
			break
		}
	}
	if coreMod == nil {
		t.Fatal("core module not found")
	}
	if coreMod.BusFactor != 1 {
		t.Errorf("core bus_factor not enriched, got %d", coreMod.BusFactor)
	}
	if coreMod.Owner != "alice" {
		t.Errorf("core owner not enriched, got %q", coreMod.Owner)
	}

	// Check meta values
	cs, _ := store.GetMeta("commit_style")
	if cs != "conventional" {
		t.Errorf("commit_style meta not set, got %q", cs)
	}
	ab, _ := store.GetMeta("active_branches")
	if ab != "7" {
		t.Errorf("active_branches meta not set, got %q", ab)
	}
}

func TestEnrichWithScoutCard(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 500, FilesCount: 2},
	}
	store.meta = map[string]string{
		"repo_name": "test",
		"languages": "go",
	}

	enrichment := &EnrichmentInput{
		ScoutCard: &ScoutEnrichment{
			PrimaryLanguage: "go",
			BuildSystem:     "go_modules",
			TotalLOC:        50000,
			TestFiles:       25,
			TotalFiles:      100,
			EntryPoints:     []string{"cmd/server/main.go"},
		},
	}

	err := Enrich(store, enrichment)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	bs, _ := store.GetMeta("build_system")
	if bs != "go_modules" {
		t.Errorf("build_system not set from scout, got %q", bs)
	}
}

func TestEnrichWithGitBlame(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 500, FilesCount: 2},
	}
	store.meta = map[string]string{
		"repo_name": "test",
		"languages": "go",
	}

	enrichment := &EnrichmentInput{
		GitBlame: map[string]string{
			"internal/core/handler.go": "alice",
			"internal/core/router.go":  "alice",
			"internal/core/utils.go":   "bob",
		},
	}

	err := Enrich(store, enrichment)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	// GitBlame-only enrichment doesn't trigger meta save or module upsert
	// since no ArchitectReport, MetricsReport, or ScoutCard is set.
	// But modules should be updated via UpsertModuleMeta after blame processing.
	mods, _ := store.LoadModules()
	if len(mods) > 0 && mods[0].Owner != "alice" {
		t.Errorf("expected owner alice from git blame majority, got %q", mods[0].Owner)
	}
}

func TestEnrichDoesNotOverrideExistingPurpose(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "api", Path: "internal/api", Purpose: "existing purpose", Loc: 500, FilesCount: 2},
	}
	store.meta = map[string]string{
		"repo_name":  "test",
		"languages":  "go",
		"arch_style": "layered",
	}

	enrichment := &EnrichmentInput{
		ArchitectReport: &ArchitectEnrichment{
			ArchStyle: "modular",
			ModulePurposes: map[string]string{
				"internal/api": "new purpose from architect",
			},
		},
	}

	err := Enrich(store, enrichment)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	mods, _ := store.LoadModules()
	if mods[0].Purpose != "existing purpose" {
		t.Errorf("enrichment should not override existing purpose, got %q", mods[0].Purpose)
	}

	// But arch_style in meta should still be updated
	v, _ := store.GetMeta("arch_style")
	if v != "modular" {
		t.Errorf("arch_style should be updated even if modules are not, got %q", v)
	}
}

func TestEnrichMetricsOverrideBusFactor(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", BusFactor: 5, Loc: 500, FilesCount: 2},
	}
	store.meta = map[string]string{
		"repo_name": "test",
		"languages": "go",
	}

	enrichment := &EnrichmentInput{
		MetricsReport: &MetricsEnrichment{
			BusFactor: 0, // don't override if zero
			ModuleRisks: []ModuleRiskEntry{
				{Module: "internal/core", BusFactor: 2, PrimaryAuthor: "alice"},
			},
		},
	}

	err := Enrich(store, enrichment)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	mods, _ := store.LoadModules()
	// Module-level bus factor from ModuleRisks should override (value=2)
	if mods[0].BusFactor != 2 {
		t.Errorf("expected bus factor 2 from module risk, got %d", mods[0].BusFactor)
	}
}

func TestEnrichPartialArchitectOnly(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 500, FilesCount: 2},
	}
	store.meta = map[string]string{
		"repo_name": "test",
		"languages": "go",
	}

	// Only architect data, everything else nil
	enrichment := &EnrichmentInput{
		ArchitectReport: &ArchitectEnrichment{
			ArchStyle: "event_driven",
		},
	}

	err := Enrich(store, enrichment)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	v, _ := store.GetMeta("arch_style")
	if v != "event_driven" {
		t.Errorf("arch_style should be set even with partial enrichment, got %q", v)
	}
}

func TestEnrichUnknownModuleRiskIgnored(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 500, FilesCount: 2},
	}

	enrichment := &EnrichmentInput{
		MetricsReport: &MetricsEnrichment{
			ModuleRisks: []ModuleRiskEntry{
				{Module: "internal/nonexistent", BusFactor: 1, PrimaryAuthor: "x"},
			},
		},
	}

	err := Enrich(store, enrichment)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}

	// Should not fail, unknown module risk is simply skipped
	mods, _ := store.LoadModules()
	if len(mods) != 1 || mods[0].Name != "core" {
		t.Error("core module should be unchanged")
	}
}

func TestEnrichStoreUpdateError(t *testing.T) {
	store := newMemStore()
	store.modules = []ModuleMeta{
		{Name: "core", Path: "internal/core", Loc: 500, FilesCount: 2},
	}
	store.meta = map[string]string{
		"repo_name": "test",
		"languages": "go",
	}

	// Use a store that returns error on UpsertModuleMeta
	failStore := &failUpdateStore{memStore: store}

	enrichment := &EnrichmentInput{
		ArchitectReport: &ArchitectEnrichment{
			ArchStyle: "modular",
		},
	}

	err := Enrich(failStore, enrichment)
	if err == nil {
		t.Error("expected error from failing store update")
	}
}

// failUpdateStore wraps memStore and fails on UpsertModuleMeta.
type failUpdateStore struct {
	*memStore
}

func (f *failUpdateStore) UpsertModuleMeta(_ ModuleMeta) error {
	return errTestUpdateFailed
}

func (f *failUpdateStore) UpdateModules(_ []ModuleMeta) error {
	return errTestUpdateFailed
}

func (f *failUpdateStore) LoadModules() ([]ModuleMeta, error) {
	return f.memStore.modules, nil
}

func (f *failUpdateStore) GetMeta(key string) (string, error) {
	return f.memStore.meta[key], nil
}

func (f *failUpdateStore) SetMeta(key, value string) error {
	return nil
}

func (f *failUpdateStore) SaveMeta(key, value string) error {
	return nil
}

func (f *failUpdateStore) LoadMeta(keys ...string) (map[string]string, error) {
	return f.memStore.LoadMeta(keys...)
}

func (f *failUpdateStore) ListModules() ([]ModuleMeta, error) {
	return f.memStore.modules, nil
}

func (f *failUpdateStore) LoadEntryPoints() ([]string, error) {
	return nil, nil
}

func (f *failUpdateStore) ListEntryPoints() ([]string, error) {
	return nil, nil
}

func (f *failUpdateStore) LoadStats() (*IndexStats, error) {
	return f.memStore.stats, nil
}

// Verify failUpdateStore satisfies ManifestStore at compile time.
var _ ManifestStore = (*failUpdateStore)(nil)

var errTestUpdateFailed = errors.New("test: UpsertModuleMeta failed")

func TestTopAuthorSelectsHighest(t *testing.T) {
	tests := []struct {
		name     string
		counts   map[string]int
		expected string
	}{
		{"single author", map[string]int{"alice": 5}, "alice"},
		{"clear winner", map[string]int{"alice": 5, "bob": 3}, "alice"},
		{"tie picks one", map[string]int{"alice": 3, "bob": 3}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := topAuthor(tt.counts)
			if tt.expected != "" && got != tt.expected {
				t.Errorf("topAuthor() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEnrichFromBlamePrefixMatch(t *testing.T) {
	modules := []ModuleMeta{
		{Name: "core", Path: "internal/core"},
		{Name: "core_test", Path: "internal/core_test"},
	}
	blame := map[string]string{
		"internal/core/handler.go":      "alice",
		"internal/core/router.go":       "alice",
		"internal/core_test/mock.go":    "bob",
		"internal/core_test/stub.go":    "bob",
		"internal/core_test/fake.go":    "bob",
	}

	enrichFromBlame(modules, blame)

	if modules[0].Owner != "alice" {
		t.Errorf("internal/core owner should be alice, got %q", modules[0].Owner)
	}
	if modules[1].Owner != "bob" {
		t.Errorf("internal/core_test owner should be bob, got %q", modules[1].Owner)
	}
}

func TestEnrichFromBlameNoOverride(t *testing.T) {
	modules := []ModuleMeta{
		{Name: "core", Path: "internal/core", Owner: "existing_owner"},
	}
	blame := map[string]string{
		"internal/core/handler.go": "alice",
	}

	enrichFromBlame(modules, blame)

	if modules[0].Owner != "existing_owner" {
		t.Errorf("should not override existing owner, got %q", modules[0].Owner)
	}
}

func TestPrimaryLangExtractsFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go", "go"},
		{"go,typescript", "go"},
		{"python,go,javascript", "python"},
		{"", ""},
		{"  rust  ,  go  ", "rust"},
	}
	for _, tt := range tests {
		got := primaryLang(tt.input)
		if got != tt.expected {
			t.Errorf("primaryLang(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestEnrichBlamePathPrefixIsExact(t *testing.T) {
	// Verify that "internal/core" does not match "internal/coreutils/..."
	modules := []ModuleMeta{
		{Name: "core", Path: "internal/core"},
	}
	blame := map[string]string{
		"internal/coreutils/helpers.go": "wrong_author",
	}

	enrichFromBlame(modules, blame)

	// Owner should remain empty since no files match internal/core
	if modules[0].Owner != "" {
		t.Errorf("expected no owner match, got %q", modules[0].Owner)
	}
}

func TestEnrichBlamePathSeparator(t *testing.T) {
	// Verify that "internal/core" does NOT match "internal/core_extra/..."
	// because we require a path separator after the prefix.
	modules := []ModuleMeta{
		{Name: "core", Path: "internal/core"},
	}
	blame := map[string]string{
		"internal/core_extra/helper.go": "extra_author",
	}

	enrichFromBlame(modules, blame)

	// Owner should remain empty since core_extra is not under internal/core
	if modules[0].Owner != "" {
		t.Errorf("expected no match for sibling directory, got %q", modules[0].Owner)
	}
}

func TestRenderManifestSmokeTest(t *testing.T) {
	data := &ManifestData{
		RepoName:        "test-repo",
		PrimaryLanguage: "go",
		ArchStyle:       "modular",
		Summary:         "A test repository for manifest generation.",
		Modules: []ModuleMeta{
			{Name: "api", Path: "internal/api", Purpose: "API layer", Loc: 1000, Owner: "alice", BusFactor: 3, FilesCount: 5},
			{Name: "db", Path: "internal/db", Purpose: "Database", Loc: 500, Owner: "bob", BusFactor: 1, FilesCount: 3},
		},
		EntryPoints: []string{"cmd/server/main.go"},
		Conventions: ConventionSet{
			CommitStyle:   "conventional",
			TestFramework: "go test",
			BuildSystem:   "go modules",
		},
		ActiveWork: ActiveWork{
			LastCommitDate: "2026-04-16",
			LastAuthor:     "alice",
			ActiveBranches: 3,
			OpenIssues:     12,
		},
	}

	output := renderManifest(data)

	// Basic smoke checks
	if !strings.Contains(output, "# test-repo") {
		t.Error("should contain header")
	}
	if !strings.Contains(output, "A test repository") {
		t.Error("should contain summary")
	}
	if !strings.Contains(output, "| api |") {
		t.Error("should contain api module row")
	}
	if !strings.Contains(output, "cmd/server/main.go") {
		t.Error("should contain entry point")
	}
	if !strings.Contains(output, "Commit style: conventional") {
		t.Error("should contain commit style")
	}
	if !strings.Contains(output, "Active branches: 3") {
		t.Error("should contain active branches")
	}
	if !strings.Contains(output, "Open issues: 12") {
		t.Error("should contain open issues")
	}
}
