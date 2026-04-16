package index

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := OpenStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenStore_CreatesSchema(t *testing.T) {
	s := openTestStore(t)

	// Verify all expected tables exist
	tables := []string{"chunks", "chunks_fts", "edges", "files", "modules", "meta"}
	for _, table := range tables {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		assert.NoError(t, err, "table %s should exist", table)
		assert.Equal(t, table, name)
	}

	// Verify indexes exist
	indexes := []string{"idx_chunks_file", "idx_chunks_kind", "idx_chunks_symbol",
		"idx_edges_source", "idx_edges_target"}
	for _, idx := range indexes {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx,
		).Scan(&name)
		assert.NoError(t, err, "index %s should exist", idx)
	}
}

func TestStore_MetaOperations(t *testing.T) {
	s := openTestStore(t)

	// Set and get meta
	err := s.SetMeta("version", "1")
	require.NoError(t, err)
	err = s.SetMeta("repo_name", "test-repo")
	require.NoError(t, err)

	val, err := s.GetMeta("version")
	require.NoError(t, err)
	assert.Equal(t, "1", val)

	val, err = s.GetMeta("repo_name")
	require.NoError(t, err)
	assert.Equal(t, "test-repo", val)

	// Non-existent key
	val, err = s.GetMeta("nonexistent")
	assert.Equal(t, "", val)
	assert.NoError(t, err)
}

func TestStore_InsertAndGetChunk(t *testing.T) {
	s := openTestStore(t)

	chunk := Chunk{
		FilePath:   "internal/foo/bar.go",
		SymbolName: "(*Bar).DoThing",
		Kind:       "method",
		Scope:      "internal/foo/bar.go > Bar",
		Language:   "go",
		LineStart:  10,
		LineEnd:    20,
		Content:    "func (b *Bar) DoThing() error { return nil }",
		Hash:       "abc123",
	}

	id, err := s.InsertChunk(chunk)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Retrieve by ID
	got, err := s.GetChunk(id)
	require.NoError(t, err)
	assert.Equal(t, chunk.FilePath, got.FilePath)
	assert.Equal(t, chunk.SymbolName, got.SymbolName)
	assert.Equal(t, chunk.Kind, got.Kind)
	assert.Equal(t, chunk.Language, got.Language)
	assert.Equal(t, chunk.LineStart, got.LineStart)
	assert.Equal(t, chunk.LineEnd, got.LineEnd)
	assert.Equal(t, chunk.Content, got.Content)
	assert.Equal(t, chunk.Hash, got.Hash)
}

func TestStore_InsertEdge(t *testing.T) {
	s := openTestStore(t)

	// Insert two chunks first
	c1 := Chunk{FilePath: "a.go", Kind: "function", Language: "go", LineStart: 1, LineEnd: 5,
		Content: "func A() {}", Hash: "h1"}
	c2 := Chunk{FilePath: "b.go", Kind: "function", Language: "go", LineStart: 1, LineEnd: 5,
		Content: "func B() {}", Hash: "h2"}

	id1, err := s.InsertChunk(c1)
	require.NoError(t, err)
	id2, err := s.InsertChunk(c2)
	require.NoError(t, err)

	edge := Edge{SourceID: id1, TargetID: id2, Relation: "calls", Weight: 1.0}
	eid, err := s.InsertEdge(edge)
	require.NoError(t, err)
	assert.Greater(t, eid, int64(0))
}

func TestStore_FileMeta(t *testing.T) {
	s := openTestStore(t)

	fm := FileMeta{
		Path:        "internal/foo/bar.go",
		Hash:        "deadbeef",
		LastIndexed: "2026-04-16T12:00:00Z",
		Language:    "go",
		Loc:         42,
		IsTest:      false,
		IsGenerated: false,
	}

	err := s.UpsertFileMeta(fm)
	require.NoError(t, err)

	got, err := s.GetFileMeta("internal/foo/bar.go")
	require.NoError(t, err)
	assert.Equal(t, fm.Path, got.Path)
	assert.Equal(t, fm.Hash, got.Hash)
	assert.Equal(t, fm.Language, got.Language)
	assert.Equal(t, fm.Loc, got.Loc)

	// Upsert should update
	fm2 := fm
	fm2.Loc = 50
	fm2.Hash = "newhash"
	err = s.UpsertFileMeta(fm2)
	require.NoError(t, err)
	got2, err := s.GetFileMeta("internal/foo/bar.go")
	require.NoError(t, err)
	assert.Equal(t, 50, got2.Loc)
	assert.Equal(t, "newhash", got2.Hash)
}

func TestStore_ModuleMeta(t *testing.T) {
	s := openTestStore(t)

	mm := ModuleMeta{
		Name:       "dispatch",
		Path:       "internal/dispatch",
		Purpose:    "Task routing and executor selection",
		Owner:      "alice",
		FilesCount: 5,
		Loc:        200,
	}

	err := s.UpsertModuleMeta(mm)
	require.NoError(t, err)

	got, err := s.GetModuleMeta("dispatch")
	require.NoError(t, err)
	assert.Equal(t, mm.Name, got.Name)
	assert.Equal(t, mm.Purpose, got.Purpose)
	assert.Equal(t, mm.FilesCount, got.FilesCount)
}

func TestStore_DeleteChunksByFile(t *testing.T) {
	s := openTestStore(t)

	// Insert chunks for two files
	for _, fp := range []string{"a.go", "b.go"} {
		_, err := s.InsertChunk(Chunk{
			FilePath: fp, Kind: "function", Language: "go",
			LineStart: 1, LineEnd: 5, Content: "func X() {}", Hash: "h",
		})
		require.NoError(t, err)
	}

	// Delete chunks for a.go
	deleted, err := s.DeleteChunksByFile("a.go")
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Verify only b.go remains
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path = 'a.go'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = s.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path = 'b.go'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStore_Stats(t *testing.T) {
	s := openTestStore(t)

	// Insert some data
	_, err := s.InsertChunk(Chunk{
		FilePath: "a.go", Kind: "function", Language: "go",
		LineStart: 1, LineEnd: 5, Content: "func A() {}", Hash: "h1",
	})
	require.NoError(t, err)
	_, err = s.InsertChunk(Chunk{
		FilePath: "b.py", Kind: "function", Language: "python",
		LineStart: 1, LineEnd: 5, Content: "def B(): pass", Hash: "h2",
	})
	require.NoError(t, err)

	stats, err := s.Stats()
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalChunks)
	assert.Contains(t, stats.Languages, "go")
	assert.Contains(t, stats.Languages, "python")
}

func TestStore_FTS5Works(t *testing.T) {
	s := openTestStore(t)

	_, err := s.InsertChunk(Chunk{
		FilePath:   "router.go",
		SymbolName: "ServeHTTP",
		Kind:       "method",
		Scope:      "router > Handler",
		Language:   "go",
		LineStart:  1,
		LineEnd:    5,
		Content:    "func (h *Handler) ServeHTTP(w ResponseWriter, r *Request)",
		Hash:       "h1",
	})
	require.NoError(t, err)

	// Search via FTS
	rows, err := s.db.Query(
		"SELECT rowid FROM chunks_fts WHERE chunks_fts MATCH ?", "ServeHTTP")
	require.NoError(t, err)
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	assert.NotEmpty(t, ids, "FTS should find ServeHTTP")
}

func TestOpenStore_FailsOnBadPath(t *testing.T) {
	_, err := OpenStore("/nonexistent/deeply/nested/path/test.db")
	assert.Error(t, err)
}

func TestStore_CloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := OpenStore(dbPath)
	require.NoError(t, err)
	// Double close should not panic
	s.Close()
	s.Close()
}

func TestStore_DeleteFileMeta(t *testing.T) {
	s := openTestStore(t)

	fm := FileMeta{Path: "x.go", Hash: "h", LastIndexed: "now", Language: "go", Loc: 10}
	err := s.UpsertFileMeta(fm)
	require.NoError(t, err)

	err = s.DeleteFileMeta("x.go")
	require.NoError(t, err)

	_, err = s.GetFileMeta("x.go")
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestStore_DeleteFileMetaAlsoRemovesChunks(t *testing.T) {
	s := openTestStore(t)

	_, err := s.InsertChunk(Chunk{
		FilePath: "x.go", Kind: "function", Language: "go",
		LineStart: 1, LineEnd: 5, Content: "func X() {}", Hash: "h",
	})
	require.NoError(t, err)

	deleted, err := s.DeleteChunksByFile("x.go")
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestStore_DBPath(t *testing.T) {
	s := openTestStore(t)
	assert.NotEmpty(t, s.DBPath())
}

func TestStore_EnsureSdpDir(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "myrepo")
	require.NoError(t, os.MkdirAll(repoPath, 0o755))

	dbPath, err := EnsureSdpDir(repoPath)
	require.NoError(t, err)
	assert.Contains(t, dbPath, ".sdp")
	assert.Contains(t, dbPath, "index.db")

	// Directory should exist
	info, err := os.Stat(filepath.Dir(dbPath))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
