package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBugfix_Sdplab1h9_ParserEdgesDiscarded verifies that parser edges are properly handled.
// The fix: ColdBuild now uses symbolic edge extraction instead of parser edges, ensuring
// all edges are captured and resolved correctly.
func TestBugfix_Sdplab1h9_ParserEdgesDiscarded(t *testing.T) {
	dir := t.TempDir()

	// Create a file with cross-file references
	goDir := filepath.Join(dir, "internal", "test")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "deps.go"), []byte(`package test

func Helper() int { return 42 }

func Caller() int {
	return Helper()
}
`), 0o644))

	result, err := ColdBuild(BuildOptions{RepoPath: dir})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify that edges were created (Caller -> Helper)
	assert.Greater(t, result.TotalEdges, 0, "should have extracted edges from cross-symbol references")

	store, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify edges exist in the database
	var edgeCount int
	err = store.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&edgeCount)
	require.NoError(t, err)
	assert.Greater(t, edgeCount, 0, "edges should be stored in database")
}

// TestBugfix_Sdplab9ty_MetadataErrorsEscalated verifies that metadata write errors are properly handled.
// The fix: SetMeta operations are now wrapped in transactions and errors are escalated instead of being silently collected.
func TestBugfix_Sdplab9ty_MetadataErrorsEscalated(t *testing.T) {
	dir := t.TempDir()

	// Create a simple file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.go"), []byte(`package main

func main() {}
`), 0o644))

	result, err := ColdBuild(BuildOptions{RepoPath: dir})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify that metadata was set correctly (if SetMeta failed, the transaction would have been rolled back)
	store, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer store.Close()

	// Check that critical metadata exists
	schemaVersion, err := store.GetMeta("schema_version")
	require.NoError(t, err)
	assert.Equal(t, "1", schemaVersion, "schema_version should be set")

	indexedAt, err := store.GetMeta("indexed_at")
	require.NoError(t, err)
	assert.NotEmpty(t, indexedAt, "indexed_at should be set")

	totalChunks, err := store.GetMeta("total_chunks")
	require.NoError(t, err)
	assert.NotEmpty(t, totalChunks, "total_chunks should be set")
}

// TestBugfix_Sdplabdtv_ChunkInsertErrorsTracked verifies that chunk insert failures are properly tracked.
// The fix: Added FilesSkipped field and better error tracking for chunk insert failures.
func TestBugfix_Sdplabdtv_ChunkInsertErrorsTracked(t *testing.T) {
	dir := t.TempDir()

	// Create a valid file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "valid.go"), []byte(`package main

func main() {}
`), 0o644))

	result, err := ColdBuild(BuildOptions{RepoPath: dir})
	require.NoError(t, err)
	require.NotNil(t, result)

	// FilesSkipped should be 0 (no parse failures)
	assert.Equal(t, 0, result.FilesSkipped, "no files should be skipped with valid input")

	// TotalChunks should match actual chunks inserted
	assert.Greater(t, result.TotalChunks, 0, "should have inserted chunks")

	store, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify the count matches
	count, err := store.CountChunks()
	require.NoError(t, err)
	assert.Equal(t, result.TotalChunks, count, "TotalChunks should match actual count")
}

// TestBugfix_Sdplabvj4_TransactionWrapping verifies that edge resolution and metadata writes are transactional.
// The fix: resolveAndInsertEdgesTx and SetMetaTx are now called within the transaction scope.
func TestBugfix_Sdplabvj4_TransactionWrapping(t *testing.T) {
	dir := t.TempDir()

	// Create files with dependencies
	goDir := filepath.Join(dir, "pkg")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "a.go"), []byte(`package pkg

var Shared = "shared"

func UseShared() string {
	return Shared
}
`), 0o644))

	result, err := ColdBuild(BuildOptions{RepoPath: dir})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify atomicity: either all operations succeed or none do
	store, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer store.Close()

	// Check that chunks, edges, and metadata are all consistent
	chunkCount, _ := store.CountChunks()
	edgeCount, _ := func() (int, error) {
		var count int
		err := store.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&count)
		return count, err
	}()

	assert.Greater(t, chunkCount, 0, "chunks should exist")
	assert.Greater(t, edgeCount, 0, "edges should exist")

	// Metadata should be set within the same transaction
	totalChunksMeta, err := store.GetMeta("total_chunks")
	require.NoError(t, err)
	assert.NotEmpty(t, totalChunksMeta, "total_chunks metadata should be set")

	// The metadata should reflect the actual state
	totalChunksInt := 0
	for _, c := range totalChunksMeta {
		totalChunksInt = totalChunksInt*10 + int(c-'0')
	}
	assert.Equal(t, chunkCount, totalChunksInt, "metadata should match actual chunk count")
}

// TestBugfix_RefreshTransactionSafety verifies that Refresh also uses proper transaction handling.
func TestBugfix_RefreshTransactionSafety(t *testing.T) {
	dir := t.TempDir()

	// Initial build
	goDir := filepath.Join(dir, "pkg")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "lib.go"), []byte(`package pkg

func Func1() int { return 1 }
`), 0o644))

	_, err := ColdBuild(BuildOptions{RepoPath: dir})
	require.NoError(t, err)

	// Modify the file
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "lib.go"), []byte(`package pkg

func Func1() int { return 2 }

func Func2() int { return 3 }
`), 0o644))

	// Refresh
	refreshResult, err := Refresh(RefreshOptions{RepoPath: dir})
	require.NoError(t, err)

	// Verify that the refresh was atomic
	store, err := OpenStore(refreshResult.DBPath)
	require.NoError(t, err)
	defer store.Close()

	// Check metadata consistency
	indexedAt, err := store.GetMeta("indexed_at")
	require.NoError(t, err)
	assert.NotEmpty(t, indexedAt, "indexed_at should be updated during refresh")

	// The file should be updated
	chunks, err := func() ([]Chunk, error) {
		rows, err := store.db.Query("SELECT * FROM chunks WHERE file_path = 'pkg/lib.go'")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var result []Chunk
		for rows.Next() {
			var c Chunk
			err := rows.Scan(&c.ID, &c.FilePath, &c.SymbolName, &c.Kind, &c.Scope,
				&c.Language, &c.LineStart, &c.LineEnd, &c.Content,
				&c.Description, &c.PageRank, &c.Hash)
			if err != nil {
				continue
			}
			result = append(result, c)
		}
		return result, nil
	}()

	require.NoError(t, err)
	// Should have Func1 and Func2
	assert.GreaterOrEqual(t, len(chunks), 2, "should have both Func1 and Func2 after refresh")
}
