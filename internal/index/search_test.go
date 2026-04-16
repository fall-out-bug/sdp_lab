package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedStoreForSearch populates a store with predictable test data for search tests.
func seedStoreForSearch(t *testing.T, s *SQLiteStore) {
	t.Helper()

	chunks := []Chunk{
		{
			FilePath:   "internal/dispatch/router.go",
			SymbolName: "RouteTask",
			Kind:       "function",
			Scope:      "dispatch > router",
			Language:   "go",
			LineStart:  10,
			LineEnd:    25,
			Content:    "func RouteTask(task Task) (Executor, error) {\n\treturn selectExecutor(task.Type())\n}",
			Hash:       "h1",
		},
		{
			FilePath:   "internal/dispatch/executor.go",
			SymbolName: "ExecutorBridge",
			Kind:       "type",
			Scope:      "dispatch > executor",
			Language:   "go",
			LineStart:  5,
			LineEnd:    15,
			Content:    "type ExecutorBridge struct {\n\tName   string\n\thandle *execHandle\n}",
			Hash:       "h2",
		},
		{
			FilePath:   "internal/dispatch/selector.go",
			SymbolName: "selectExecutor",
			Kind:       "function",
			Scope:      "dispatch > selector",
			Language:   "go",
			LineStart:  20,
			LineEnd:    35,
			Content:    "func selectExecutor(taskType string) (Executor, error) {\n\treturn registry.Get(taskType)\n}",
			Hash:       "h3",
		},
		{
			FilePath:   "internal/queue/worker.go",
			SymbolName: "Worker",
			Kind:       "type",
			Scope:      "queue > worker",
			Language:   "go",
			LineStart:  1,
			LineEnd:    10,
			Content:    "type Worker struct {\n\tID     int\n\tqueue  chan Task\n\tdone   chan bool\n}",
			Hash:       "h4",
		},
		{
			FilePath:   "internal/queue/pool.go",
			SymbolName: "Pool.Start",
			Kind:       "method",
			Scope:      "queue > pool",
			Language:   "go",
			LineStart:  30,
			LineEnd:    45,
			Content:    "func (p *Pool) Start() error {\n\tfor i := 0; i < p.size; i++ {\n\t\tgo p.worker(i)\n\t}\n\treturn nil\n}",
			Hash:       "h5",
		},
		{
			FilePath:   "cmd/sdp/main.go",
			SymbolName: "main",
			Kind:       "function",
			Scope:      "main",
			Language:   "go",
			LineStart:  1,
			LineEnd:    20,
			Content:    "func main() {\n\tfmt.Println(\"sdp cli\")\n}",
			Hash:       "h6",
		},
	}

	ids := make([]int64, len(chunks))
	for i, c := range chunks {
		id, err := s.InsertChunk(c)
		require.NoError(t, err)
		ids[i] = id
	}

	// Create structural edges
	edges := []Edge{
		{SourceID: ids[0], TargetID: ids[2], Relation: "calls", Weight: 1.0},    // RouteTask calls selectExecutor
		{SourceID: ids[0], TargetID: ids[1], Relation: "uses", Weight: 1.0},     // RouteTask uses ExecutorBridge
		{SourceID: ids[4], TargetID: ids[3], Relation: "uses", Weight: 1.0},     // Pool.Start uses Worker
		{SourceID: ids[5], TargetID: ids[0], Relation: "calls", Weight: 1.0},    // main calls RouteTask
		{SourceID: ids[2], TargetID: ids[1], Relation: "uses", Weight: 1.0},     // selectExecutor uses ExecutorBridge
	}
	for _, e := range edges {
		_, err := s.InsertEdge(e)
		require.NoError(t, err)
	}

	// Create module metadata
	modules := []ModuleMeta{
		{Name: "dispatch", Path: "internal/dispatch", Purpose: "Task routing", Loc: 100, BusFactor: 1, IsHotspot: true},
		{Name: "queue", Path: "internal/queue", Purpose: "Task queue", Loc: 80, BusFactor: 2, IsHotspot: false},
	}
	for _, m := range modules {
		err := s.UpsertModuleMeta(m)
		require.NoError(t, err)
	}
}

func TestSemanticSearch_FTSONly(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := SemanticSearch(s, "RouteTask", 10, nil)
	require.NoError(t, err)
	assert.Equal(t, "semantic", resp.Mode)
	assert.Equal(t, "RouteTask", resp.Query)
	assert.NotEmpty(t, resp.Results, "should find at least one result")

	// The RouteTask chunk should be in results
	found := false
	for _, r := range resp.Results {
		if r.Chunk.SymbolName == "RouteTask" {
			found = true
			assert.Equal(t, "fts", r.MatchSrc)
			assert.Equal(t, "internal/dispatch/router.go", r.Chunk.FilePath)
			break
		}
	}
	assert.True(t, found, "RouteTask should be found in results")
}

func TestSemanticSearch_MultipleKeywords(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := SemanticSearch(s, "task routing", 10, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results)
	assert.Equal(t, "semantic", resp.Mode)
}

func TestSemanticSearch_EmptyQuery(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := SemanticSearch(s, "", 10, nil)
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}

func TestSemanticSearch_NoResults(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := SemanticSearch(s, "nonexistentXYZ123", 10, nil)
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}

func TestSemanticSearch_WithEmbedFallback(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	// Provide an embed function that returns a vector.
	// Since the store has no vector data, it should fall back to FTS-only.
	embedFn := func(q string) ([]float32, error) {
		return make([]float32, 128), nil
	}

	resp, err := SemanticSearch(s, "executor", 10, embedFn)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results)
	// Should be FTS since there is no vector data
	for _, r := range resp.Results {
		assert.Equal(t, "fts", r.MatchSrc)
	}
}

func TestFindSearch_ExactIdentifier(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := FindSearch(s, "ExecutorBridge", false, 20)
	require.NoError(t, err)
	assert.Equal(t, "find", resp.Mode)
	assert.NotEmpty(t, resp.Results)

	found := false
	for _, r := range resp.Results {
		if r.Chunk.SymbolName == "ExecutorBridge" {
			found = true
			assert.Equal(t, "type", r.Chunk.Kind)
			assert.Equal(t, "internal/dispatch/executor.go", r.Chunk.FilePath)
			break
		}
	}
	assert.True(t, found, "ExecutorBridge should be found")
}

func TestFindSearch_Keyword(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := FindSearch(s, "Worker", false, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results)

	found := false
	for _, r := range resp.Results {
		if r.Chunk.SymbolName == "Worker" {
			found = true
			break
		}
	}
	assert.True(t, found, "Worker should be found")
}

func TestFindSearch_RegexPattern(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := FindSearch(s, "Route.*", true, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results)

	// Should find RouteTask at minimum
	found := false
	for _, r := range resp.Results {
		if r.Chunk.SymbolName == "RouteTask" {
			found = true
			break
		}
	}
	assert.True(t, found, "RouteTask should be found via regex pattern")
}

func TestFindSearch_EmptyQuery(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := FindSearch(s, "", false, 20)
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}

func TestFindSearch_ContentMatch(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := FindSearch(s, "selectExecutor", false, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results, "FTS should match in content field")
}

func TestDepsSearch_Forward(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	// cmd/sdp/main.go -> RouteTask (in dispatch) is a forward dep
	resp, err := DepsSearch(s, "cmd/sdp", false, 3)
	require.NoError(t, err)
	assert.Equal(t, "cmd/sdp", resp.Module)
	assert.NotEmpty(t, resp.Results)

	// Should find that cmd/sdp depends on internal/dispatch
	found := false
	for _, r := range resp.Results {
		assert.Equal(t, "forward", r.Relation)
		if r.ModuleName == "internal/dispatch" {
			found = true
		}
	}
	assert.True(t, found, "cmd/sdp should depend on internal/dispatch")
}

func TestDepsSearch_Reverse(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := DepsSearch(s, "internal/dispatch", true, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results)

	// cmd/sdp/main.go calls RouteTask (in dispatch), so dispatch should have
	// a reverse dependency from cmd/sdp
	found := false
	for _, r := range resp.Results {
		if r.Relation == "reverse" {
			found = true
			break
		}
	}
	assert.True(t, found, "should have reverse dependencies")
}

func TestDepsSearch_NoResults(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := DepsSearch(s, "nonexistent/module", false, 3)
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}

func TestDepsSearch_ModuleMetadata(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	// Search for deps of cmd/sdp which calls into dispatch
	resp, err := DepsSearch(s, "cmd/sdp", false, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results)

	// Should find dispatch module with metadata
	for _, r := range resp.Results {
		if r.ModuleName == "internal/dispatch" {
			assert.Equal(t, 100, r.LOC)
			assert.True(t, r.IsHotspot)
			assert.Equal(t, 1, r.BusFactor)
			return
		}
	}
}

func TestDepsSearch_TrailingSlash(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	// Should work with trailing slash -- same as TestDepsSearch_Reverse but with trailing slash
	resp, err := DepsSearch(s, "internal/dispatch/", true, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results)
}

func TestRRFFuse(t *testing.T) {
	s := openTestStore(t)

	// Create two chunks
	_, err := s.InsertChunk(Chunk{
		FilePath: "a.go", SymbolName: "FuncA", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func A() {}", Hash: "h1",
	})
	require.NoError(t, err)
	id2, err := s.InsertChunk(Chunk{
		FilePath: "b.go", SymbolName: "FuncB", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func B() {}", Hash: "h2",
	})
	require.NoError(t, err)

	fts := []rankedItem{
		{chunkID: id2, score: -1.5},
		{chunkID: 1, score: -2.0},
	}
	vec := []rankedItem{
		{chunkID: 1, score: 0.95},
	}

	results := rrfFuse(s, fts, vec, 10)
	assert.NotEmpty(t, results)

	// Item that appears in both lists should rank higher
	assert.Equal(t, "fused", results[0].MatchSrc)

	// The item appearing in both FTS and vec should have a higher score
	// than the item appearing only in FTS
	if len(results) >= 2 {
		assert.GreaterOrEqual(t, results[0].Score, results[1].Score)
	}
}

func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", `"hello" OR "world"`},
		{"  spaces  ", `"spaces"`},
		{"", ""},
		{"a", ""},     // too short
		{"ab", `"ab"`},
		{`"quoted"`, `"quoted"`},
		{"*star*", `"star"`},
	}
	for _, tc := range tests {
		got := sanitizeFTSQuery(tc.input)
		assert.Equal(t, tc.expected, got, "sanitizeFTSQuery(%q)", tc.input)
	}
}

func TestRegexToLike(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"RouteTask", "%RouteTask%"},
		{"Route.*", "Route%"},
		{"^prefix", "%prefix%"},
		{"suffix$", "%suffix%"},
		{".+", "%"},
	}
	for _, tc := range tests {
		got := regexToLike(tc.input)
		assert.Equal(t, tc.expected, got, "regexToLike(%q)", tc.input)
	}
}

func TestExtractModuleFromPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"internal/dispatch/router.go", "internal/dispatch"},
		{"cmd/sdp/main.go", "cmd/sdp"},
		{"main.go", "main.go"},
	}
	for _, tc := range tests {
		got := extractModuleFromPath(tc.input)
		assert.Equal(t, tc.expected, got, "extractModuleFromPath(%q)", tc.input)
	}
}

func TestSearchResponse_JSON(t *testing.T) {
	s := openTestStore(t)
	seedStoreForSearch(t, s)

	resp, err := FindSearch(s, "ExecutorBridge", false, 5)
	require.NoError(t, err)
	assert.Equal(t, "find", resp.Mode)
	assert.Equal(t, "ExecutorBridge", resp.Query)
	assert.Greater(t, resp.Total, 0)

	// Verify the response has all expected fields
	for _, r := range resp.Results {
		assert.NotEmpty(t, r.Chunk.FilePath)
		assert.NotEmpty(t, r.Chunk.Kind)
		assert.NotEmpty(t, r.Chunk.Language)
		assert.Greater(t, r.Chunk.LineStart, 0)
		assert.Greater(t, r.Chunk.LineEnd, 0)
	}
}
