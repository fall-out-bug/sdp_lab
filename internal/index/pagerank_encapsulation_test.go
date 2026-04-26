package index

import (
	"path/filepath"
	"testing"
)

// TestPageRankEncapsulation tests that PageRank computation uses Store methods
// instead of direct db access (fixes sdplab-9zb, sdplab-xlm, sdplab-ytr).
func TestPageRankEncapsulation(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Insert test chunks
	chunks := []Chunk{
		{FilePath: "a.go", SymbolName: "A", Kind: "function", Language: "go", LineStart: 1, LineEnd: 10, Content: "func A() {}", Hash: "a1"},
		{FilePath: "b.go", SymbolName: "B", Kind: "function", Language: "go", LineStart: 1, LineEnd: 10, Content: "func B() {}", Hash: "b1"},
		{FilePath: "c.go", SymbolName: "C", Kind: "function", Language: "go", LineStart: 1, LineEnd: 10, Content: "func C() {}", Hash: "c1"},
	}

	chunkIDs := make([]int64, len(chunks))
	for i, c := range chunks {
		id, err := store.InsertChunk(c)
		if err != nil {
			t.Fatalf("Failed to insert chunk: %v", err)
		}
		chunkIDs[i] = id
	}

	// Insert test edges (A -> B, A -> C, B -> C)
	edges := []Edge{
		{SourceID: chunkIDs[0], TargetID: chunkIDs[1], Relation: "calls", Weight: 1.0},
		{SourceID: chunkIDs[0], TargetID: chunkIDs[2], Relation: "calls", Weight: 1.0},
		{SourceID: chunkIDs[1], TargetID: chunkIDs[2], Relation: "calls", Weight: 1.0},
	}

	for _, e := range edges {
		if _, err := store.InsertEdge(e); err != nil {
			t.Fatalf("Failed to insert edge: %v", err)
		}
	}

	// Test GetAllEdges (fixes sdplab-xlm)
	allEdges, scanErrors, err := store.GetAllEdges()
	if err != nil {
		t.Fatalf("GetAllEdges failed: %v", err)
	}
	if len(allEdges) != 3 {
		t.Errorf("GetAllEdges returned %d edges, want 3", len(allEdges))
	}
	if scanErrors != 0 {
		t.Errorf("GetAllEdges reported %d scan errors, want 0", scanErrors)
	}

	// Test GetEdgeCount
	count, err := store.GetEdgeCount()
	if err != nil {
		t.Fatalf("GetEdgeCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("GetEdgeCount returned %d, want 3", count)
	}

	// Test UpdatePageRank (fixes sdplab-9zb)
	testScore := 0.5
	if err := store.UpdatePageRank(chunkIDs[0], testScore); err != nil {
		t.Fatalf("UpdatePageRank failed: %v", err)
	}

	// Verify the update
	chunk, err := store.GetChunk(chunkIDs[0])
	if err != nil {
		t.Fatalf("GetChunk failed: %v", err)
	}
	if chunk.PageRank != testScore {
		t.Errorf("PageRank = %f, want %f", chunk.PageRank, testScore)
	}

	// Test ComputePageRank using Store methods (integration test)
	updated, err := ComputePageRank(store)
	if err != nil {
		t.Fatalf("ComputePageRank failed: %v", err)
	}
	if updated != 3 {
		t.Errorf("ComputePageRank updated %d chunks, want 3", updated)
	}

	// Verify PageRank scores are non-zero and normalized
	for _, id := range chunkIDs {
		c, err := store.GetChunk(id)
		if err != nil {
			t.Fatalf("GetChunk failed: %v", err)
		}
		if c.PageRank <= 0 {
			t.Errorf("Chunk %d has non-positive PageRank: %f", id, c.PageRank)
		}
	}
}

// TestGetAllEdgesWithScanErrors tests that GetAllEdges properly reports scan errors
// (fixes sdplab-ytr - ensures errors are reported instead of silently skipped).
func TestGetAllEdgesWithScanErrors(t *testing.T) {
	// This test verifies the API returns error counts even if we can't easily
	// create corrupted data in a test database.
	// The key fix is that GetAllEdges now returns (edges, scanErrors, error)
	// instead of silently skipping bad rows.

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Insert test data
	chunk := Chunk{FilePath: "test.go", SymbolName: "Test", Kind: "function", Language: "go", LineStart: 1, LineEnd: 10, Content: "func Test() {}", Hash: "test1"}
	chunkID, err := store.InsertChunk(chunk)
	if err != nil {
		t.Fatalf("Failed to insert chunk: %v", err)
	}

	edge := Edge{SourceID: chunkID, TargetID: chunkID, Relation: "calls", Weight: 1.0}
	if _, err := store.InsertEdge(edge); err != nil {
		t.Fatalf("Failed to insert edge: %v", err)
	}

	// Test that GetAllEdges returns error count
	edges, scanErrors, err := store.GetAllEdges()
	if err != nil {
		t.Fatalf("GetAllEdges failed: %v", err)
	}
	if len(edges) != 1 {
		t.Errorf("GetAllEdges returned %d edges, want 1", len(edges))
	}
	// With valid data, scanErrors should be 0
	if scanErrors != 0 {
		t.Errorf("GetAllEdges reported %d scan errors with valid data, want 0", scanErrors)
	}
}

// TestStoreEncapsulationForSearch tests that search functions use Store methods
// instead of direct db access (fixes sdplab-9zb).
func TestStoreEncapsulationForSearch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Insert test chunks
	chunks := []Chunk{
		{FilePath: "internal/module/a.go", SymbolName: "FuncA", Kind: "function", Language: "go", LineStart: 1, LineEnd: 10, Content: "package module\n\nfunc FuncA() {}", Hash: "a1"},
		{FilePath: "internal/module/b.go", SymbolName: "FuncB", Kind: "function", Language: "go", LineStart: 1, LineEnd: 10, Content: "package module\n\nfunc FuncB() {}", Hash: "b1"},
	}

	chunkIDs := make([]int64, len(chunks))
	for i, c := range chunks {
		id, err := store.InsertChunk(c)
		if err != nil {
			t.Fatalf("Failed to insert chunk: %v", err)
		}
		chunkIDs[i] = id
	}

	// Test QueryChunksByFilePrefix (used by findModuleChunks)
	ids, err := store.QueryChunksByFilePrefix("internal/module/", "internal/module")
	if err != nil {
		t.Fatalf("QueryChunksByFilePrefix failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("QueryChunksByFilePrefix returned %d chunks, want 2", len(ids))
	}

	// Test QueryEdgesBySourceOrTarget (used by traverseEdges)
	// First insert an edge
	edge := Edge{SourceID: chunkIDs[0], TargetID: chunkIDs[1], Relation: "calls", Weight: 1.0}
	if _, err := store.InsertEdge(edge); err != nil {
		t.Fatalf("Failed to insert edge: %v", err)
	}

	// Test forward direction
	results, err := store.QueryEdgesBySourceOrTarget(chunkIDs[0], false)
	if err != nil {
		t.Fatalf("QueryEdgesBySourceOrTarget (forward) failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("QueryEdgesBySourceOrTarget returned %d results (forward), want 1", len(results))
	}
	if results[0].ID != chunkIDs[1] {
		t.Errorf("QueryEdgesBySourceOrTarget returned ID %d, want %d", results[0].ID, chunkIDs[1])
	}

	// Test reverse direction
	results, err = store.QueryEdgesBySourceOrTarget(chunkIDs[1], true)
	if err != nil {
		t.Fatalf("QueryEdgesBySourceOrTarget (reverse) failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("QueryEdgesBySourceOrTarget returned %d results (reverse), want 1", len(results))
	}
	if results[0].ID != chunkIDs[0] {
		t.Errorf("QueryEdgesBySourceOrTarget returned ID %d, want %d", results[0].ID, chunkIDs[0])
	}

	// Test QueryModuleByPath (used by lookupModule)
	mod := ModuleMeta{
		Name: "test-module", Path: "internal/module", Purpose: "test",
		Owner: "test", BusFactor: 1, FilesCount: 2, Loc: 20, IsHotspot: false,
	}
	if err := store.UpsertModuleMeta(mod); err != nil {
		t.Fatalf("UpsertModuleMeta failed: %v", err)
	}

	gotMod, err := store.QueryModuleByPath("internal/module")
	if err != nil {
		t.Fatalf("QueryModuleByPath failed: %v", err)
	}
	if gotMod.Name != "test-module" {
		t.Errorf("QueryModuleByPath returned name %s, want test-module", gotMod.Name)
	}
}

// TestLoadFileHashMapNotNPlusOne verifies that LoadFileHashMap is used
// instead of per-file queries (fixes sdplab-erz).
func TestLoadFileHashMapNotNPlusOne(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Insert file metadata for multiple files
	files := []FileMeta{
		{Path: "a.go", Hash: "hash1", LastIndexed: "2024-01-01", Language: "go", Loc: 100},
		{Path: "b.go", Hash: "hash2", LastIndexed: "2024-01-01", Language: "go", Loc: 200},
		{Path: "c.go", Hash: "hash3", LastIndexed: "2024-01-01", Language: "go", Loc: 300},
	}

	for _, f := range files {
		if err := store.UpsertFileMeta(f); err != nil {
			t.Fatalf("UpsertFileMeta failed: %v", err)
		}
	}

	// Test LoadFileHashMap - this should be a single query
	// instead of N separate GetFileMeta calls
	fileMap, err := store.LoadFileHashMap()
	if err != nil {
		t.Fatalf("LoadFileHashMap failed: %v", err)
	}

	if len(fileMap) != 3 {
		t.Errorf("LoadFileHashMap returned %d files, want 3", len(fileMap))
	}

	// Verify all files are present
	expectedFiles := map[string]string{
		"a.go": "hash1",
		"b.go": "hash2",
		"c.go": "hash3",
	}

	for path, expectedHash := range expectedFiles {
		gotHash, ok := fileMap[path]
		if !ok {
			t.Errorf("LoadFileHashMap missing file %s", path)
			continue
		}
		if gotHash != expectedHash {
			t.Errorf("LoadFileHashMap[%s] = %s, want %s", path, gotHash, expectedHash)
		}
	}
}

// TestFTSAndRegexSearchEncapsulation tests that FTS and regex search
// use Store methods instead of direct db access (fixes sdplab-9zb).
func TestFTSAndRegexSearchEncapsulation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Insert test chunks with searchable content
	chunks := []Chunk{
		{FilePath: "search.go", SymbolName: "SearchFunc", Kind: "function", Language: "go", LineStart: 1, LineEnd: 10, Content: "func SearchFunc() {}", Description: "A search function", Hash: "s1"},
		{FilePath: "query.go", SymbolName: "QueryFunc", Kind: "function", Language: "go", LineStart: 1, LineEnd: 10, Content: "func QueryFunc() {}", Description: "A query function", Hash: "q1"},
	}

	for _, c := range chunks {
		if _, err := store.InsertChunk(c); err != nil {
			t.Fatalf("Failed to insert chunk: %v", err)
		}
	}

	// Test FTS5Search
	results, err := store.FTS5Search("search", 10)
	if err != nil {
		t.Fatalf("FTS5Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("FTS5Search returned no results for 'search'")
	}

	// Test FTS5ExactSearchWithChunks
	searchResults, err := store.FTS5ExactSearchWithChunks("search", 10)
	if err != nil {
		t.Fatalf("FTS5ExactSearchWithChunks failed: %v", err)
	}
	if len(searchResults) == 0 {
		t.Error("FTS5ExactSearchWithChunks returned no results for 'search'")
	}
	// Verify we get full SearchResult structs
	for _, r := range searchResults {
		if r.Chunk.ID == 0 {
			t.Error("FTS5ExactSearchWithChunks returned empty Chunk")
		}
		if r.Score == 0 {
			t.Error("FTS5ExactSearchWithChunks returned zero score")
		}
		if r.MatchSrc != "fts" {
			t.Errorf("FTS5ExactSearchWithChunks returned MatchSrc=%s, want 'fts'", r.MatchSrc)
		}
	}

	// Test RegexSearchWithChunks
	searchResults, err = store.RegexSearchWithChunks("%Query%", 10)
	if err != nil {
		t.Fatalf("RegexSearchWithChunks failed: %v", err)
	}
	if len(searchResults) == 0 {
		t.Error("RegexSearchWithChunks returned no results for '%Query%'")
	}
}
