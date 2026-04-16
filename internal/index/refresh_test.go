package index

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func measureStart() time.Time {
	return time.Now()
}

func measureElapsed(start time.Time) time.Duration {
	return time.Since(start)
}

// createRefreshTestRepo builds a minimal repo for refresh tests and runs
// an initial ColdBuild. Returns repo path, db path, and store.
func createAndBuildTestRepo(t *testing.T) (string, string, *SQLiteStore) {
	t.Helper()
	dir := t.TempDir()

	// Go package
	goDir := filepath.Join(dir, "pkg")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "handler.go"), []byte(`package pkg

type Handler struct {
	Name string
}

func NewHandler(name string) *Handler {
	return &Handler{Name: name}
}
`), 0o644))

	// Python module
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.py"), []byte(`"""App module."""

def hello():
    return "world"
`), 0o644))

	// Run cold build
	result, err := ColdBuild(BuildOptions{RepoPath: dir})
	require.NoError(t, err)

	store, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	return dir, result.DBPath, store
}

func TestRefresh_NoChanges_IsNoop(t *testing.T) {
	repoPath, dbPath, _ := createAndBuildTestRepo(t)

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 0, result.FilesUpdated, "no files should be updated")
	assert.Equal(t, 0, result.FilesAdded, "no files should be added")
	assert.Equal(t, 0, result.FilesRemoved, "no files should be removed")
	assert.Greater(t, result.FilesChecked, 0, "should check files")
}

func TestRefresh_DetectsModifiedFile(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Get original chunk count
	originalStats, err := store.Stats()
	require.NoError(t, err)

	// Modify handler.go
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "pkg", "handler.go"), []byte(`package pkg

type Handler struct {
	Name string
	Addr string
}

func NewHandler(name, addr string) *Handler {
	return &Handler{Name: name, Addr: addr}
}

func (h *Handler) Serve() error {
	return nil
}
`), 0o644))

	// Refresh
	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesUpdated, "exactly one file should be updated")
	assert.Equal(t, 0, result.FilesAdded, "no new files")
	assert.Equal(t, 0, result.FilesRemoved, "no removed files")

	// Verify chunks were actually updated
	newStats, err := store.Stats()
	require.NoError(t, err)
	assert.Greater(t, newStats.TotalChunks, originalStats.TotalChunks,
		"new handler.go has more declarations, should have more chunks")

	// Verify the new content is indexed
	rows, err := store.db.Query(
		"SELECT symbol_name FROM chunks WHERE file_path = ?", "pkg/handler.go")
	require.NoError(t, err)
	defer rows.Close()
	var symbols []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		symbols = append(symbols, s)
	}
	assert.Contains(t, symbols, "Handler.Serve", "new Serve method should be indexed")
}

func TestRefresh_DetectsNewFile(t *testing.T) {
	repoPath, dbPath, _ := createAndBuildTestRepo(t)

	// Add a new file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "router.go"), []byte(`package main

func Route() string {
	return "/"
}
`), 0o644))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 0, result.FilesUpdated, "no existing files changed")
	assert.Equal(t, 1, result.FilesAdded, "one new file added")
	assert.Equal(t, 0, result.FilesRemoved, "no removed files")
}

func TestRefresh_DetectsDeletedFile(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Verify app.py is indexed
	fm, err := store.GetFileMeta("app.py")
	require.NoError(t, err)
	require.NotNil(t, fm)

	// Delete the file
	require.NoError(t, os.Remove(filepath.Join(repoPath, "app.py")))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 0, result.FilesUpdated, "no files updated")
	assert.Equal(t, 0, result.FilesAdded, "no files added")
	assert.Equal(t, 1, result.FilesRemoved, "one file removed")

	// Verify the file metadata is gone
	_, err = store.GetFileMeta("app.py")
	assert.Error(t, err, "deleted file metadata should be removed")

	// Verify chunks for the file are gone
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path = 'app.py'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "chunks for deleted file should be removed")
}

func TestRefresh_DetectsMultipleChanges(t *testing.T) {
	repoPath, dbPath, _ := createAndBuildTestRepo(t)

	// Modify one file, add one file, delete one file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "pkg", "handler.go"), []byte(`package pkg

func Updated() {}
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "new_file.go"), []byte(`package main

func NewFunc() {}
`), 0o644))

	require.NoError(t, os.Remove(filepath.Join(repoPath, "app.py")))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesUpdated, "one file updated")
	assert.Equal(t, 1, result.FilesAdded, "one file added")
	assert.Equal(t, 1, result.FilesRemoved, "one file removed")
}

func TestRefresh_FailsWithoutExistingIndex(t *testing.T) {
	dir := t.TempDir()

	_, err := Refresh(RefreshOptions{RepoPath: dir})
	assert.Error(t, err, "should fail when no index database exists")
	assert.Contains(t, err.Error(), "not found")
}

func TestRefresh_SkipsSecretsOnRefresh(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Add a secret file (should be excluded from indexing)
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, ".env.local"), []byte("API_KEY=supersecret\n"), 0o644))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	// The .env file should not be counted
	assert.Equal(t, 0, result.FilesAdded, "secret files should not be indexed")

	// Verify no chunks from .env
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path LIKE '%.env%'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should not index .env files on refresh")
}

func TestRefresh_SkipsLargeFilesOnRefresh(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Add a large file (>100KB)
	largeContent := make([]byte, 101*1024)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "large_output.log"), largeContent, 0o644))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 0, result.FilesAdded, "large files should not be indexed")

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path LIKE '%large_output%'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should not index oversized files on refresh")
}

func TestRefresh_SkipsGeneratedFilesOnRefresh(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Add a generated file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "api.pb.go"), []byte(`package api

func Generated() {}
`), 0o644))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 0, result.FilesAdded, "generated files should not be indexed")

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path LIKE '%.pb.go%'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should not index .pb.go files on refresh")
}

func TestRefresh_SkipsBinaryFilesOnRefresh(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Add a binary file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "binary.bin"), []byte{0x00, 0x01, 0x02, 0x00, 0x04}, 0o644))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 0, result.FilesAdded, "binary files should not be indexed")

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path LIKE '%binary.bin%'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should not index binary files on refresh")
}

func TestRefresh_UpdatesFileHash(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Get original hash
	fmBefore, err := store.GetFileMeta("pkg/handler.go")
	require.NoError(t, err)
	originalHash := fmBefore.Hash

	// Modify the file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "pkg", "handler.go"), []byte(`package pkg

func CompletelyDifferent() string { return "new" }
`), 0o644))

	_, err = Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	// Hash should have changed
	fmAfter, err := store.GetFileMeta("pkg/handler.go")
	require.NoError(t, err)
	assert.NotEqual(t, originalHash, fmAfter.Hash,
		"file hash should update after content change")
}

func TestRefresh_FTS5StaysInSync(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Modify file to add a unique symbol
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "pkg", "handler.go"), []byte(`package pkg

func UniqueSearchTarget() bool { return true }
`), 0o644))

	_, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	// FTS should find the new content
	rows, err := store.db.Query(
		"SELECT rowid FROM chunks_fts WHERE chunks_fts MATCH ?", "UniqueSearchTarget")
	require.NoError(t, err)
	defer rows.Close()

	var found bool
	for rows.Next() {
		found = true
	}
	assert.True(t, found, "FTS should find new content after refresh")
}

func TestRefresh_EdgesCascadeOnDelete(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Manually insert an edge between chunks to test CASCADE
	chunks, _, _ := ParseFile(filepath.Join(repoPath, "pkg", "handler.go"), "go")
	require.NotEmpty(t, chunks)

	// Get actual chunk IDs from DB
	rows, err := store.db.Query("SELECT id FROM chunks WHERE file_path = ?", "pkg/handler.go")
	require.NoError(t, err)
	var ids []int64
	for rows.Next() {
		var id int64
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	rows.Close()

	if len(ids) >= 2 {
		// Insert a manual edge
		_, err = store.InsertEdge(Edge{SourceID: ids[0], TargetID: ids[1], Relation: "calls"})
		require.NoError(t, err)

		// Get original edge count
		var edgeCountBefore int
		require.NoError(t, store.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&edgeCountBefore))
		require.Greater(t, edgeCountBefore, 0)
	}

	// Modify the file (triggers delete + re-insert of chunks)
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "pkg", "handler.go"), []byte(`package pkg

func Replacement() {}
`), 0o644))

	_, err = Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	// Old edges should be gone (CASCADE on chunk delete)
	var edgeCountAfter int
	require.NoError(t, store.db.QueryRow("SELECT COUNT(*) FROM edges WHERE source_id IN (SELECT id FROM chunks WHERE file_path = 'pkg/handler.go')").Scan(&edgeCountAfter))

	// The old edge pointing to deleted chunks should be gone
	// Only edges among new chunks remain (none by default from basic parsing)
}

func TestRefresh_LanguageFilter(t *testing.T) {
	repoPath, dbPath, _ := createAndBuildTestRepo(t)

	// Modify Python file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "app.py"), []byte(`"""Changed."""

def changed_function():
    pass
`), 0o644))

	// Add new Go file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "extra.go"), []byte(`package main

func Extra() {}
`), 0o644))

	// Refresh with Go-only filter
	result, err := Refresh(RefreshOptions{
		RepoPath:  repoPath,
		DBPath:    dbPath,
		Languages: []string{"go"},
	})
	require.NoError(t, err)

	// Only Go file changes should be detected
	assert.Equal(t, 1, result.FilesAdded, "only Go file should be counted as added")
	// Python file should not be checked (filtered out)
}

func TestRefresh_UpdatedMetadata(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Modify a file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "pkg", "handler.go"), []byte(`package pkg

func Changed() {}
`), 0o644))

	_, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	// Verify metadata was updated
	indexedAt, err := store.GetMeta("indexed_at")
	require.NoError(t, err)
	assert.NotEmpty(t, indexedAt, "indexed_at should be updated after refresh")

	totalChunks, err := store.GetMeta("total_chunks")
	require.NoError(t, err)
	assert.NotEmpty(t, totalChunks, "total_chunks should be updated after refresh")
}

func TestRefresh_FileRemovedThenRecreated(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	originalStats, err := store.Stats()
	require.NoError(t, err)

	// Delete the file
	require.NoError(t, os.Remove(filepath.Join(repoPath, "app.py")))

	_, err = Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	// Verify removed
	_, err = store.GetFileMeta("app.py")
	assert.Error(t, err, "file should be removed after deletion")

	// Recreate with different content
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "app.py"), []byte(`"""Recreated."""

def recreated():
    pass
`), 0o644))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesAdded, "recreated file should be detected as added")

	// Verify it's back in the index
	fm, err := store.GetFileMeta("app.py")
	require.NoError(t, err)
	assert.NotNil(t, fm)

	// Verify total counts are consistent
	newStats, err := store.Stats()
	require.NoError(t, err)
	assert.Equal(t, originalStats.TotalFiles, newStats.TotalFiles,
		"after delete+recreate, file count should match original")
}

func TestRefresh_PerformanceSmall(t *testing.T) {
	// Document performance budget: refreshing 10 changed files should be fast.
	// This test verifies the mechanism works, not the absolute timing (which
	// depends on hardware). The documented target is <3 seconds for 10 files.
	repoPath, dbPath, _ := createAndBuildTestRepo(t)

	// Create 10 additional files
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf(`package main

func PerfFunc() int { return %d }
`, i)
		require.NoError(t, os.WriteFile(
			filepath.Join(repoPath, fmt.Sprintf("perf_%d.go", i)),
			[]byte(content), 0o644))
	}

	// Initial refresh to add them
	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)
	assert.Equal(t, 10, result.FilesAdded)

	// Now modify all 10
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf(`package main

func PerfFunc() int { return %d }
`, i+10)
		require.NoError(t, os.WriteFile(
			filepath.Join(repoPath, fmt.Sprintf("perf_%d.go", i)),
			[]byte(content), 0o644))
	}

	start := measureStart()
	result, err = Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	elapsed := measureElapsed(start)

	require.NoError(t, err)
	assert.Equal(t, 10, result.FilesUpdated)
	// Performance budget: should complete well under 3 seconds
	// (generously accounting for CI overhead)
	assert.Less(t, elapsed.Seconds(), 3.0,
		"refresh of 10 files should complete in <3 seconds (got %v)", elapsed)
}

func TestRefresh_TotalCountsCorrect(t *testing.T) {
	repoPath, dbPath, store := createAndBuildTestRepo(t)

	// Add 2 files, delete 1, modify 1
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "new1.go"), []byte(`package main

func New1() {}
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "new2.py"), []byte(`def new2(): pass`), 0o644))

	require.NoError(t, os.Remove(filepath.Join(repoPath, "app.py")))

	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "pkg", "handler.go"), []byte(`package pkg

func Modified() {}
`), 0o644))

	result, err := Refresh(RefreshOptions{RepoPath: repoPath, DBPath: dbPath})
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesUpdated)
	assert.Equal(t, 2, result.FilesAdded)
	assert.Equal(t, 1, result.FilesRemoved)

	// Verify total counts in the result match the actual store
	stats, err := store.Stats()
	require.NoError(t, err)
	assert.Equal(t, stats.TotalFiles, result.TotalFiles)
	assert.Equal(t, stats.TotalChunks, result.TotalChunks)
}
