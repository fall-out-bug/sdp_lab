package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRepo builds a temporary directory with a mix of source files
// for cold-build testing.
func createTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Go package
	goDir := filepath.Join(dir, "internal", "foo")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "handler.go"), []byte(`package foo

import "errors"

var ErrNotFound = errors.New("not found")

type Handler struct {
	Name string
}

func NewHandler(name string) *Handler {
	return &Handler{Name: name}
}

func (h *Handler) Serve() error {
	if h.Name == "" {
		return ErrNotFound
	}
	return nil
}
`), 0o644))

	// Python module
	pyDir := filepath.Join(dir, "service")
	require.NoError(t, os.MkdirAll(pyDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pyDir, "app.py"), []byte(`"""Application module."""

class App:
    def __init__(self):
        self.name = "default"

    def run(self):
        print("running")

def create_app():
    return App()
`), 0o644))

	// TypeScript file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "router.ts"), []byte(`export interface Route {
  path: string;
}

export function createRouter(): Route[] {
  return [];
}
`), 0o644))

	// Files that should be excluded
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=abc123\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cert.pem"), []byte("-----BEGIN CERTIFICATE-----\nfake\n"), 0o644))

	// Binary file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "binary.dat"), []byte{0x00, 0x01, 0x02, 0x00, 0x04}, 0o644))

	// Vendor directory (should be excluded)
	vendorDir := filepath.Join(dir, "vendor", "lib")
	require.NoError(t, os.MkdirAll(vendorDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "vendor.go"), []byte(`package lib

func Vendored() {}
`), 0o644))

	// node_modules (should be excluded)
	nodeDir := filepath.Join(dir, "node_modules", "pkg")
	require.NoError(t, os.MkdirAll(nodeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nodeDir, "index.js"), []byte("module.exports = {};"), 0o644))

	// .git directory (should be excluded)
	gitDir := filepath.Join(dir, ".git", "objects")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "data"), []byte("git data"), 0o644))

	// Large file (>100KB)
	largeContent := make([]byte, 101*1024)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "large.log"), largeContent, 0o644))

	// Generated file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "api.pb.go"), []byte(`package api

func Generated() {}
`), 0o644))

	return dir
}

func TestColdBuild_CreatesIndexDB(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)
	defer os.Remove(result.DBPath)

	assert.NotEmpty(t, result.DBPath)
	assert.Contains(t, result.DBPath, ".sdp")
	assert.Contains(t, result.DBPath, "index.db")

	// DB file should exist
	_, err = os.Stat(result.DBPath)
	assert.NoError(t, err, "index.db should exist on disk")
}

func TestColdBuild_ProducesChunks(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	assert.Greater(t, result.TotalChunks, 0, "should produce at least some chunks")
	assert.Greater(t, result.TotalFiles, 0, "should index at least some files")
}

func TestColdBuild_ExcludesVendorAndGit(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	// Open the store and verify no vendor/node_modules/.git chunks
	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	// Check no chunks from excluded directories
	var excludedChunks int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM chunks
		WHERE file_path LIKE '%/vendor/%'
		   OR file_path LIKE '%/node_modules/%'
		   OR file_path LIKE '%/.git/%'
	`).Scan(&excludedChunks)
	require.NoError(t, err)
	assert.Equal(t, 0, excludedChunks, "should not index vendor/node_modules/.git files")
}

func TestColdBuild_ExcludesSecrets(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	var secretChunks int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM chunks
		WHERE file_path LIKE '%.env'
		   OR file_path LIKE '%.pem'
	`).Scan(&secretChunks)
	require.NoError(t, err)
	assert.Equal(t, 0, secretChunks, "should not index secret files")
}

func TestColdBuild_ExcludesBinaries(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	var binaryChunks int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE file_path LIKE '%binary.dat%'`).Scan(&binaryChunks)
	require.NoError(t, err)
	assert.Equal(t, 0, binaryChunks, "should not index binary files")
}

func TestColdBuild_ExcludesLargeFiles(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	var largeChunks int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE file_path LIKE '%large.log%'`).Scan(&largeChunks)
	require.NoError(t, err)
	assert.Equal(t, 0, largeChunks, "should not index large files")
}

func TestColdBuild_ExcludesGeneratedFiles(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	var genChunks int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE file_path LIKE '%.pb.go%'`).Scan(&genChunks)
	require.NoError(t, err)
	assert.Equal(t, 0, genChunks, "should not index generated .pb.go files")
}

func TestColdBuild_DetectsLanguages(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	assert.Contains(t, result.Languages, "go")
	assert.Contains(t, result.Languages, "python")
	assert.Contains(t, result.Languages, "typescript")
}

func TestColdBuild_StoresFileMetadata(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	// Check that file metadata was stored
	fm, err := s.GetFileMeta("internal/foo/handler.go")
	require.NoError(t, err)
	assert.Equal(t, "go", fm.Language)
	assert.Greater(t, fm.Loc, 0)
	assert.NotEmpty(t, fm.Hash)
}

func TestColdBuild_StoresMeta(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	version, err := s.GetMeta("schema_version")
	require.NoError(t, err)
	assert.Equal(t, "1", version)

	indexed, err := s.GetMeta("indexed_at")
	require.NoError(t, err)
	assert.NotEmpty(t, indexed)

	totalChunks, err := s.GetMeta("total_chunks")
	require.NoError(t, err)
	assert.NotEmpty(t, totalChunks)
}

func TestColdBuild_HashBasedDedup(t *testing.T) {
	repoPath := createTestRepo(t)

	// Create two identical files with same content
	content := []byte(`package dup

func Same() int { return 42 }
`)
	dir := filepath.Join(repoPath, "dup")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), content, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.go"), content, 0o644))

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	// Both files should have their own chunks (different paths), but same content hash
	fmA, err := s.GetFileMeta("dup/a.go")
	require.NoError(t, err)
	fmB, err := s.GetFileMeta("dup/b.go")
	require.NoError(t, err)
	assert.Equal(t, fmA.Hash, fmB.Hash, "identical files should have same hash")
}

func TestColdBuild_CustomDBPath(t *testing.T) {
	repoPath := createTestRepo(t)
	customPath := filepath.Join(t.TempDir(), "custom.db")

	result, err := ColdBuild(BuildOptions{
		RepoPath: repoPath,
		DBPath:   customPath,
	})
	require.NoError(t, err)
	assert.Equal(t, customPath, result.DBPath)

	_, err = os.Stat(customPath)
	assert.NoError(t, err)
}

func TestColdBuild_EmptyRepo(t *testing.T) {
	emptyDir := t.TempDir()
	// Only create excluded dirs
	require.NoError(t, os.MkdirAll(filepath.Join(emptyDir, ".git"), 0o755))

	result, err := ColdBuild(BuildOptions{RepoPath: emptyDir})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalChunks)
	assert.Equal(t, 0, result.TotalFiles)
}

func TestColdBuild_LanguageFilter(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{
		RepoPath: repoPath,
		Languages: []string{"go"},
	})
	require.NoError(t, err)

	// Should only index Go files
	for _, lang := range result.Languages {
		assert.Equal(t, "go", lang, "should only index Go when language filter is set")
	}
}

func TestColdBuild_TestFileDetection(t *testing.T) {
	repoPath := t.TempDir()

	// Create a test file
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "handler_test.go"), []byte(`package handler

func TestSomething(t *testing.T) {}
`), 0o644))

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	fm, err := s.GetFileMeta("handler_test.go")
	require.NoError(t, err)
	assert.True(t, fm.IsTest, "should detect test files")
}

func TestColdBuild_ChunksHaveCorrectFields(t *testing.T) {
	repoPath := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "simple.go"), []byte(`package simple

func Hello() string {
	return "world"
}
`), 0o644))

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	// Get chunks and verify fields
	stats, err := s.Stats()
	require.NoError(t, err)
	assert.Greater(t, stats.TotalChunks, 0)

	// Verify chunk content has expected fields
	rows, err := s.db.Query("SELECT file_path, kind, language, line_start, line_end, content, hash FROM chunks LIMIT 10")
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var fp, kind, lang, content, hash string
		var lineStart, lineEnd int
		require.NoError(t, rows.Scan(&fp, &kind, &lang, &lineStart, &lineEnd, &content, &hash))
		assert.NotEmpty(t, fp)
		assert.NotEmpty(t, kind)
		assert.NotEmpty(t, lang)
		assert.NotEmpty(t, content)
		assert.NotEmpty(t, hash)
		assert.Greater(t, lineEnd, 0)
		assert.GreaterOrEqual(t, lineEnd, lineStart)
	}
}

func TestColdBuild_FTS5SearchWorks(t *testing.T) {
	repoPath := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "router.go"), []byte(`package router

func ServeHTTP() error {
	return nil
}
`), 0o644))

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer s.Close()

	// Search via FTS
	rows, err := s.db.Query("SELECT rowid FROM chunks_fts WHERE chunks_fts MATCH ?", "ServeHTTP")
	require.NoError(t, err)
	defer rows.Close()

	var found bool
	for rows.Next() {
		found = true
	}
	assert.True(t, found, "FTS should find ServeHTTP")
}
