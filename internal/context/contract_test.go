package sdpcontext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sort"
	"testing"
	"time"
)

// TestContractVersion verifies the contract version is defined.
func TestContractVersion(t *testing.T) {
	if ContractVersion != "v1.0.0" {
		t.Errorf("ContractVersion = %s, want v1.0.0", ContractVersion)
	}
}

// mockRepoMapper satisfies RepoMapper interface for compile-time verification.
type mockRepoMapper struct {
	repoMap *RepoMap
	files   []FileEntry
}

func (m *mockRepoMapper) Map(ctx context.Context, root string) (*RepoMap, error) {
	return m.repoMap, nil
}

func (m *mockRepoMapper) FileEntries() []FileEntry {
	return m.files
}

// mockDiffRetriever satisfies DiffRetriever interface.
type mockDiffRetriever struct {
	diff  *DiffResult
	hunks map[string][]DiffHunk
}

func (m *mockDiffRetriever) Diff(ctx context.Context, base, head string) (*DiffResult, error) {
	return m.diff, nil
}

func (m *mockDiffRetriever) Hunks(file string) []DiffHunk {
	return m.hunks[file]
}

// mockPromptBudgeter satisfies PromptBudgeter interface.
type mockPromptBudgeter struct {
	budget   int
	allocated int
}

func (m *mockPromptBudgeter) Budget(model string) int {
	return m.budget
}

func (m *mockPromptBudgeter) Allocate(task string, layers []PromptLayer) int {
	tokens := len(task) / 4 // chars per token heuristic
	for _, layer := range layers {
		tokens += layer.Tokens
	}
	m.allocated = tokens
	return tokens
}

func (m *mockPromptBudgeter) Remaining() int {
	return m.budget - m.allocated
}

// mockCacheHasher satisfies CacheHasher interface.
type mockCacheHasher struct{}

func (m *mockCacheHasher) Hash(inputs ...string) CacheKey {
	// Sort inputs for deterministic hashing
	sorted := make([]string, len(inputs))
	copy(sorted, inputs)
	sort.Strings(sorted)

	hash := sha256.New()
	for _, input := range sorted {
		hash.Write([]byte(input))
		hash.Write([]byte{0}) // null separator
	}
	return CacheKey(hex.EncodeToString(hash.Sum(nil)))
}

func (m *mockCacheHasher) Validate(key CacheKey, inputs ...string) bool {
	return m.Hash(inputs...) == key
}

// TestInterfaceCompileTimeChecks verifies all interfaces can be satisfied.
func TestInterfaceCompileTimeChecks(t *testing.T) {
	var _ RepoMapper = (*mockRepoMapper)(nil)
	var _ DiffRetriever = (*mockDiffRetriever)(nil)
	var _ PromptBudgeter = (*mockPromptBudgeter)(nil)
	var _ CacheHasher = (*mockCacheHasher)(nil)
}

// TestRepoMapFields verifies RepoMap struct field accessibility.
func TestRepoMapFields(t *testing.T) {
	rm := &RepoMap{
		Root: "/path/to/repo",
		Files: []FileEntry{
			{
				Path:         "main.go",
				Language:     "Go",
				Lines:        100,
				Hash:         "abc123",
				LastModified: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
			},
		},
		TotalFiles:        1,
		TotalLines:        100,
		LanguageBreakdown: map[string]int{"Go": 100},
		Metadata:          map[string]interface{}{"scanned_at": "2026-04-27"},
	}

	if rm.Root != "/path/to/repo" {
		t.Errorf("Root = %s, want /path/to/repo", rm.Root)
	}
	if len(rm.Files) != 1 {
		t.Errorf("Files length = %d, want 1", len(rm.Files))
	}
	if rm.TotalFiles != 1 {
		t.Errorf("TotalFiles = %d, want 1", rm.TotalFiles)
	}
	if rm.TotalLines != 100 {
		t.Errorf("TotalLines = %d, want 100", rm.TotalLines)
	}
	if rm.LanguageBreakdown["Go"] != 100 {
		t.Errorf("LanguageBreakdown[Go] = %d, want 100", rm.LanguageBreakdown["Go"])
	}
}

// TestDiffResultFields verifies DiffResult struct field accessibility.
func TestDiffResultFields(t *testing.T) {
	dr := &DiffResult{
		Base:  "main",
		Head:  "feature",
		Files: []string{"main.go", "utils.go"},
		Hunks: map[string][]DiffHunk{
			"main.go": {
				{
					File:     "main.go",
					StartLine: 10,
					EndLine:   20,
					Added:     5,
					Removed:   3,
					Content:   "@@ -10,3 +10,5 @@",
				},
			},
		},
		Stats: DiffStats{
			FilesChanged: 2,
			LinesAdded:   15,
			LinesRemoved: 8,
		},
	}

	if dr.Base != "main" {
		t.Errorf("Base = %s, want main", dr.Base)
	}
	if dr.Head != "feature" {
		t.Errorf("Head = %s, want feature", dr.Head)
	}
	if len(dr.Files) != 2 {
		t.Errorf("Files length = %d, want 2", len(dr.Files))
	}
	if dr.Stats.FilesChanged != 2 {
		t.Errorf("Stats.FilesChanged = %d, want 2", dr.Stats.FilesChanged)
	}
	if dr.Stats.LinesAdded != 15 {
		t.Errorf("Stats.LinesAdded = %d, want 15", dr.Stats.LinesAdded)
	}
	if dr.Stats.LinesRemoved != 8 {
		t.Errorf("Stats.LinesRemoved = %d, want 8", dr.Stats.LinesRemoved)
	}
}

// TestPromptBudgetFields verifies PromptBudget struct field accessibility.
func TestPromptBudgetFields(t *testing.T) {
	pb := &PromptBudget{
		TotalTokens:     8000,
		ContextPct:      0.20,
		AllocatedTokens: 2000,
		Model:           "claude-opus",
	}

	if pb.TotalTokens != 8000 {
		t.Errorf("TotalTokens = %d, want 8000", pb.TotalTokens)
	}
	if pb.ContextPct != 0.20 {
		t.Errorf("ContextPct = %f, want 0.20", pb.ContextPct)
	}
	if pb.AllocatedTokens != 2000 {
		t.Errorf("AllocatedTokens = %d, want 2000", pb.AllocatedTokens)
	}
	if pb.Model != "claude-opus" {
		t.Errorf("Model = %s, want claude-opus", pb.Model)
	}
}

// TestPromptLayerFields verifies PromptLayer struct field accessibility.
func TestPromptLayerFields(t *testing.T) {
	pl := PromptLayer{
		Name:    "system",
		Content: "You are a helpful assistant.",
		Tokens:  10,
	}

	if pl.Name != "system" {
		t.Errorf("Name = %s, want system", pl.Name)
	}
	if pl.Content != "You are a helpful assistant." {
		t.Errorf("Content = %s, want 'You are a helpful assistant.'", pl.Content)
	}
	if pl.Tokens != 10 {
		t.Errorf("Tokens = %d, want 10", pl.Tokens)
	}
}

// TestCacheKeyHashingDeterministic verifies CacheKey hashing is deterministic.
func TestCacheKeyHashingDeterministic(t *testing.T) {
	hasher := &mockCacheHasher{}

	// Same inputs should produce same hash
	key1 := hasher.Hash("file1.go", "file2.go")
	key2 := hasher.Hash("file1.go", "file2.go")
	if key1 != key2 {
		t.Errorf("Hash(%q, %q) = %s, want %s (deterministic)", "file1.go", "file2.go", key2, key1)
	}

	// Different input order should produce same hash (sorted inputs)
	key3 := hasher.Hash("file2.go", "file1.go")
	if key1 != key3 {
		t.Errorf("Hash(%q, %q) = %s, want %s (order-independent)", "file2.go", "file1.go", key3, key1)
	}

	// Different inputs should produce different hash
	key4 := hasher.Hash("file3.go")
	if key1 == key4 {
		t.Errorf("Hash(%q) = %s, want different hash", "file3.go", key4)
	}
}

// TestCacheKeyValidate verifies CacheKey validation.
func TestCacheKeyValidate(t *testing.T) {
	hasher := &mockCacheHasher{}

	inputs := []string{"file1.go", "file2.go"}
	key := hasher.Hash(inputs...)

	if !hasher.Validate(key, inputs...) {
		t.Errorf("Validate(%s, %v) = false, want true", key, inputs)
	}

	if hasher.Validate(key, "file3.go") {
		t.Errorf("Validate(%s, [file3.go]) = true, want false", key)
	}
}

// TestStructFieldReflection verifies struct fields via reflection.
func TestStructFieldReflection(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		fields   []string
	}{
		{
			name: "RepoMap",
			typ:  reflect.TypeOf(RepoMap{}),
			fields: []string{"Root", "Files", "TotalFiles", "TotalLines", "LanguageBreakdown", "Metadata"},
		},
		{
			name: "FileEntry",
			typ:  reflect.TypeOf(FileEntry{}),
			fields: []string{"Path", "Language", "Lines", "Hash", "LastModified"},
		},
		{
			name: "DiffResult",
			typ:  reflect.TypeOf(DiffResult{}),
			fields: []string{"Base", "Head", "Files", "Hunks", "Stats"},
		},
		{
			name: "DiffHunk",
			typ:  reflect.TypeOf(DiffHunk{}),
			fields: []string{"File", "StartLine", "EndLine", "Added", "Removed", "Content"},
		},
		{
			name: "DiffStats",
			typ:  reflect.TypeOf(DiffStats{}),
			fields: []string{"FilesChanged", "LinesAdded", "LinesRemoved"},
		},
		{
			name: "PromptBudget",
			typ:  reflect.TypeOf(PromptBudget{}),
			fields: []string{"TotalTokens", "ContextPct", "AllocatedTokens", "Model"},
		},
		{
			name: "PromptLayer",
			typ:  reflect.TypeOf(PromptLayer{}),
			fields: []string{"Name", "Content", "Tokens"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, field := range tt.fields {
				_, found := tt.typ.FieldByName(field)
				if !found {
					t.Errorf("%s struct missing field %s", tt.name, field)
				}
			}
		})
	}
}

// TestContractDocStructs verifies contract documentation structs exist.
func TestContractDocStructs(t *testing.T) {
	_ = PromptBudgetContract{
		CharsPerToken:       4,
		ContextPct:          0.20,
		LayerInjectionOrder: []string{"system", "task", "context"},
		TruncationStrategy:  "oldest",
	}

	_ = CacheContract{
		HashAlgorithm:       "SHA-256",
		InputSorting:        true,
		InvalidationTriggers: []string{"file-change", "config-change"},
		KeyEncoding:         "hex",
	}
}
