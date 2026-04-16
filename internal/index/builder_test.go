package index

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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
	defer func() { _ = s.Close() }()

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

// --- Issue 1: Edge Extraction Tests ---

func TestColdBuild_ProducesEdges(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	// handler.go has: ErrNotFound, Handler, NewHandler, Handler.Serve
	// NewHandler references Handler -> calls edge
	// Handler.Serve references ErrNotFound -> calls edge
	assert.Greater(t, result.TotalEdges, 0, "should produce edges from cross-symbol references")
}

func TestColdBuild_EdgesHaveValidIDs(t *testing.T) {
	repoPath := createTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	// Verify edges have valid source_id and target_id referencing real chunks
	rows, err := s.db.Query(`
		SELECT e.id, e.source_id, e.target_id, e.relation
		FROM edges e`)
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var id, sourceID, targetID int64
		var relation string
		require.NoError(t, rows.Scan(&id, &sourceID, &targetID, &relation))
		assert.Greater(t, id, int64(0))
		assert.Greater(t, sourceID, int64(0))
		assert.Greater(t, targetID, int64(0))
		assert.NotEqual(t, sourceID, targetID, "edges should not be self-referencing")

		// Verify the referenced chunks exist
		srcChunk, err := s.GetChunk(sourceID)
		assert.NoError(t, err, "source chunk %d should exist", sourceID)
		assert.NotNil(t, srcChunk)

		tgtChunk, err := s.GetChunk(targetID)
		assert.NoError(t, err, "target chunk %d should exist", targetID)
		assert.NotNil(t, tgtChunk)

		assert.Contains(t, []string{"calls", "uses", "implements"}, relation)
	}
}

func TestExtractSymbolicEdges_FunctionCalls(t *testing.T) {
	chunks := []Chunk{
		{FilePath: "a.go", SymbolName: "Helper", Kind: "function", Content: "func Helper() int { return 42 }"},
		{FilePath: "a.go", SymbolName: "Main", Kind: "function", Content: "func Main() { x := Helper() }"},
	}

	edges := extractSymbolicEdges("a.go", chunks)
	assert.NotEmpty(t, edges, "should detect Main calling Helper")

	found := false
	for _, e := range edges {
		if e.SourceSymbol == "Main" && e.TargetSymbol == "Helper" {
			found = true
			assert.Equal(t, "calls", e.Relation)
		}
	}
	assert.True(t, found, "should find Main -> Helper edge")
}

func TestExtractSymbolicEdges_MethodReferences(t *testing.T) {
	chunks := []Chunk{
		{FilePath: "handler.go", SymbolName: "ErrNotFound", Kind: "var", Content: "var ErrNotFound = errors.New(\"not found\")"},
		{FilePath: "handler.go", SymbolName: "Handler", Kind: "type", Content: "type Handler struct { Name string }"},
		{FilePath: "handler.go", SymbolName: "Handler.Serve", Kind: "method", Content: "func (h *Handler) Serve() error { return ErrNotFound }"},
	}

	edges := extractSymbolicEdges("handler.go", chunks)
	assert.NotEmpty(t, edges, "Handler.Serve should reference ErrNotFound")

	// Handler.Serve should reference ErrNotFound
	foundErr := false
	for _, e := range edges {
		if e.SourceSymbol == "Handler.Serve" && e.TargetSymbol == "ErrNotFound" {
			foundErr = true
		}
	}
	assert.True(t, foundErr, "Handler.Serve should have edge to ErrNotFound")
}

func TestExtractSymbolicEdges_NoSelfEdges(t *testing.T) {
	chunks := []Chunk{
		{FilePath: "a.go", SymbolName: "Foo", Kind: "function", Content: "func Foo() { Foo() }"},
	}

	edges := extractSymbolicEdges("a.go", chunks)
	for _, e := range edges {
		assert.NotEqual(t, e.SourceSymbol, e.TargetSymbol, "should not create self-edges")
	}
}

func TestExtractSymbolicEdges_SingleChunkNoEdges(t *testing.T) {
	chunks := []Chunk{
		{FilePath: "a.go", SymbolName: "Only", Kind: "function", Content: "func Only() {}"},
	}

	edges := extractSymbolicEdges("a.go", chunks)
	assert.Empty(t, edges, "single chunk should produce no edges")
}

func TestContainsIdentifier(t *testing.T) {
	assert.True(t, containsIdentifier("x := Helper()", "Helper"))
	assert.True(t, containsIdentifier("return ErrNotFound", "ErrNotFound"))
	assert.False(t, containsIdentifier("return ErrNotFoundWrapped", "ErrNotFound"))
	assert.False(t, containsIdentifier("myHelper()", "Helper"))
	assert.True(t, containsIdentifier("Handler.Serve()", "Handler"))
	assert.True(t, containsIdentifier("Handler{Name: name}", "Handler"))
}

// --- Issue 2: Error Collection Tests ---

func TestBuildResult_HasErrorField(t *testing.T) {
	result := &BuildResult{}
	assert.Empty(t, result.Errors)
	result.Errors = append(result.Errors, fmt.Errorf("test error"))
	assert.Len(t, result.Errors, 1)
}

func TestRefreshResult_HasErrorField(t *testing.T) {
	result := &RefreshResult{}
	assert.Empty(t, result.Errors)
	result.Errors = append(result.Errors, fmt.Errorf("test error"))
	assert.Len(t, result.Errors, 1)
}

// --- Issue 3: Transaction Safety Tests ---

func TestStore_BeginCommit(t *testing.T) {
	s := openTestStore(t)

	tx, err := s.Begin()
	require.NoError(t, err)

	_, err = InsertChunkTx(tx, Chunk{
		FilePath: "tx.go", Kind: "function", Language: "go",
		LineStart: 1, LineEnd: 1, Content: "func Tx() {}", Hash: "h1",
	})
	require.NoError(t, err)

	require.NoError(t, CommitTx(tx))

	// Chunk should be visible
	var count int
	require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path = 'tx.go'").Scan(&count))
	assert.Equal(t, 1, count)
}

func TestStore_TransactionRollback(t *testing.T) {
	s := openTestStore(t)

	tx, err := s.Begin()
	require.NoError(t, err)

	_, err = InsertChunkTx(tx, Chunk{
		FilePath: "rb.go", Kind: "function", Language: "go",
		LineStart: 1, LineEnd: 1, Content: "func Rb() {}", Hash: "h1",
	})
	require.NoError(t, err)

	// Rollback instead of commit
	RollbackTx(tx)

	// Chunk should NOT be visible
	var count int
	require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE file_path = 'rb.go'").Scan(&count))
	assert.Equal(t, 0, count)
}

// --- Cross-File Edge Tests ---

// createCrossFileTestRepo builds a temp repo with two Go packages where one
// calls functions defined in the other.
func createCrossFileTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Package: internal/dispatch
	dispatchDir := filepath.Join(dir, "internal", "dispatch")
	require.NoError(t, os.MkdirAll(dispatchDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dispatchDir, "dispatcher.go"), []byte(`package dispatch

import "errors"

var ErrStopped = errors.New("stopped")

type Dispatcher struct {
	queue []string
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{queue: nil}
}

func (d *Dispatcher) Start() error {
	return nil
}

func (d *Dispatcher) Stop() error {
	return ErrStopped
}
`), 0o644))

	// Package: cmd/app  (calls dispatch.NewDispatcher, dispatch.Dispatcher, etc.)
	appDir := filepath.Join(dir, "cmd", "app")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "main.go"), []byte(`package main

import "fmt"

func main() {
	d := NewDispatcher()
	d.Start()
	fmt.Println("running")
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

type Dispatcher struct {
	x int
}
`), 0o644))

	return dir
}

func TestColdBuild_CrossFileEdges(t *testing.T) {
	repoPath := createCrossFileTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)
	defer os.Remove(result.DBPath)

	// We expect edges within each file (same-file) AND potentially cross-file
	// if identifier names overlap. The key test: total edges > 0.
	assert.Greater(t, result.TotalEdges, 0, "should produce edges from cross-file references")
}

func TestColdBuild_CrossFileEdges_VerifyEdgeTargets(t *testing.T) {
	repoPath := createCrossFileTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	// Query edges and verify that cross-file edges exist
	rows, err := s.db.Query(`
		SELECT
			sc.file_path AS source_file,
			sc.symbol_name AS source_symbol,
			tc.file_path AS target_file,
			tc.symbol_name AS target_symbol,
			e.relation
		FROM edges e
		JOIN chunks sc ON sc.id = e.source_id
		JOIN chunks tc ON tc.id = e.target_id
		ORDER BY source_file, source_symbol, target_file`)
	require.NoError(t, err)
	defer rows.Close()

	type edgeInfo struct {
		srcFile, srcSym string
		tgtFile, tgtSym string
		relation        string
	}

	var edges []edgeInfo
	for rows.Next() {
		var e edgeInfo
		require.NoError(t, rows.Scan(&e.srcFile, &e.srcSym, &e.tgtFile, &e.tgtSym, &e.relation))
		edges = append(edges, e)
	}

	// Verify that at least some edges exist
	assert.NotEmpty(t, edges, "should have at least some edges")

	// Check that cross-file edges exist (source and target in different files)
	var crossFileCount int
	for _, e := range edges {
		if e.srcFile != e.tgtFile {
			crossFileCount++
		}
	}
	assert.Greater(t, crossFileCount, 0, "should have at least one cross-file edge")
}

func TestColdBuild_CrossFileEdges_DepsSearch(t *testing.T) {
	repoPath := createCrossFileTestRepo(t)

	result, err := ColdBuild(BuildOptions{RepoPath: repoPath})
	require.NoError(t, err)

	s, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	// Search forward deps from the cmd/app module
	resp, err := DepsSearch(s, "cmd/app", false, 3)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should find dependencies reachable from cmd/app via edges
	assert.NotEmpty(t, resp.Results, "DepsSearch from cmd/app should return results")
}

func TestExtractExternalRefs(t *testing.T) {
	content := `func main() {
	d := dispatch.NewDispatcher()
	d.Start()
	fmt.Println("hello")
	return ErrNotFound
}`
	localSyms := map[string]bool{
		"main": true,
	}

	refs := extractExternalRefs(content, localSyms)

	// Should find capitalized identifiers that aren't local
	refSet := make(map[string]bool)
	for _, r := range refs {
		refSet[r] = true
	}

	// These should be found as external refs
	assert.True(t, refSet["NewDispatcher"], "should find NewDispatcher")
	assert.True(t, refSet["Start"], "should find Start")
	assert.True(t, refSet["ErrNotFound"], "should find ErrNotFound")

	// "main" is local, should NOT appear
	assert.False(t, refSet["main"], "should not include local symbols")
}

func TestExtractExternalRefs_SkipsBuiltins(t *testing.T) {
	content := `func Handler() error {
	if x {
		return nil
	}
	return New("test")
}`
	localSyms := map[string]bool{}

	refs := extractExternalRefs(content, localSyms)

	refSet := make(map[string]bool)
	for _, r := range refs {
		refSet[r] = true
	}

	// "nil" should be skipped (lowercase anyway, but also in skip list)
	assert.False(t, refSet["nil"])
	// "New" is in the builtin skip list
	assert.False(t, refSet["New"])
}

func TestResolveCrossFileTarget(t *testing.T) {
	// Build a mock symMap and symNameIndex
	symMap := map[string]int64{
		"cmd/app/main.go:main":            1,
		"internal/dispatch/dispatcher.go:NewDispatcher": 2,
		"internal/dispatch/dispatcher.go:Dispatcher":    3,
		"internal/dispatch/dispatcher.go:Dispatcher.Start": 4,
	}

	symNameIndex := make(map[string][]string)
	for key := range symMap {
		idx := strings.Index(key, ":")
		symName := key[idx+1:]
		symNameIndex[symName] = append(symNameIndex[symName], key)
		if dotIdx := strings.Index(symName, "."); dotIdx >= 0 {
			short := symName[dotIdx+1:]
			symNameIndex[short] = append(symNameIndex[short], key)
		}
	}

	// Looking up "NewDispatcher" from cmd/app/main.go should find it in dispatch
	imports := []string{"example.com/repo/internal/dispatch"}
	id := resolveCrossFileTarget("NewDispatcher", "cmd/app/main.go", imports, symMap, symNameIndex)
	assert.Equal(t, int64(2), id, "should resolve NewDispatcher to dispatch package")

	// Looking up "Dispatcher" should find the type in dispatch
	id = resolveCrossFileTarget("Dispatcher", "cmd/app/main.go", imports, symMap, symNameIndex)
	assert.Equal(t, int64(3), id, "should resolve Dispatcher type to dispatch package")

	// Looking up a non-existent symbol should return 0
	id = resolveCrossFileTarget("NonExistent", "cmd/app/main.go", imports, symMap, symNameIndex)
	assert.Equal(t, int64(0), id, "should return 0 for non-existent symbol")
}

func TestExtractSymbolicEdgesSingleChunk(t *testing.T) {
	// Bug fix: a file with a single function (e.g. cmd/app/main.go with just main())
	// must still produce cross-file edges when it references external symbols.
	chunks := []Chunk{
		{
			SymbolName: "main",
			Kind:       "function",
			Content:    "func main() {\n\td := NewDispatcher()\n\td.Start()\n}",
		},
	}

	edges := extractSymbolicEdges("cmd/app/main.go", chunks)

	// Should produce cross-file edges for NewDispatcher and Start
	var crossFileEdges []SymbolicEdge
	for _, e := range edges {
		if e.TargetFile == "" {
			crossFileEdges = append(crossFileEdges, e)
		}
	}
	assert.True(t, len(crossFileEdges) >= 1,
		"single-chunk file should produce cross-file edges, got %d edges total", len(edges))

	// Verify NewDispatcher is among the cross-file references
	foundNewDisp := false
	for _, e := range crossFileEdges {
		if e.TargetSymbol == "NewDispatcher" {
			foundNewDisp = true
			break
		}
	}
	assert.True(t, foundNewDisp, "should find cross-file edge for NewDispatcher")
}

func TestResolveCrossFileTargetImportDisambiguation(t *testing.T) {
	// Bug fix: when two packages define the same symbol name, the resolver must
	// prefer the one matching the source file's imports.
	symMap := map[string]int64{
		"cmd/app/main.go:main":                       1,
		"internal/dispatch/dispatcher.go:RouteTask":   2,
		"internal/other/handler.go:RouteTask":         3,
	}

	symNameIndex := make(map[string][]string)
	for key := range symMap {
		idx := strings.Index(key, ":")
		symName := key[idx+1:]
		symNameIndex[symName] = append(symNameIndex[symName], key)
	}

	// cmd/app/main.go imports internal/other, NOT internal/dispatch.
	// RouteTask should resolve to internal/other/handler.go (id=3), not dispatch (id=2).
	imports := []string{"example.com/repo/internal/other"}
	id := resolveCrossFileTarget("RouteTask", "cmd/app/main.go", imports, symMap, symNameIndex)
	assert.Equal(t, int64(3), id, "should prefer symbol from imported package internal/other")

	// With no imports, falls back to any different-file candidate
	id = resolveCrossFileTarget("RouteTask", "cmd/app/main.go", nil, symMap, symNameIndex)
	assert.True(t, id == int64(2) || id == int64(3),
		"without imports, should pick some different-file candidate, got %d", id)
}

func TestExtractGoImports(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "single import",
			content: `package main

import "fmt"

func main() { fmt.Println("hi") }`,
			expected: []string{"fmt"},
		},
		{
			name: "grouped imports",
			content: `package main

import (
	"fmt"
	"strings"

	"example.com/repo/internal/dispatch"
)

func main() {}`,
			expected: []string{"fmt", "strings", "example.com/repo/internal/dispatch"},
		},
		{
			name: "aliased import",
			content: `package main

import (
	dispatch "example.com/repo/internal/dispatch"
)

func main() {}`,
			expected: []string{"example.com/repo/internal/dispatch"},
		},
		{
			name:     "no imports",
			content:  "package main\n\nfunc main() {}",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			imports := extractGoImports(tc.content)
			assert.Equal(t, tc.expected, imports)
		})
	}
}

func TestRefresh_PreservesCallerEdges(t *testing.T) {
	// Reproduce the bug: editing a callee file must not destroy
	// cross-file edges from an unchanged caller.
	dir := t.TempDir()

	// Caller: cmd/app/main.go
	appDir := filepath.Join(dir, "cmd", "app")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "main.go"), []byte(`package main

import "example.com/repo/internal/dispatch"

func main() {
	dispatch.NewRouter()
}
`), 0o644))

	// Callee: internal/dispatch/router.go
	dispDir := filepath.Join(dir, "internal", "dispatch")
	require.NoError(t, os.MkdirAll(dispDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dispDir, "router.go"), []byte(`package dispatch

func NewRouter() *Router {
	return &Router{}
}

type Router struct{}
`), 0o644))

	dbPath := filepath.Join(dir, ".sdp", "index.db")

	// Cold build
	result, err := ColdBuild(BuildOptions{RepoPath: dir, DBPath: dbPath})
	require.NoError(t, err)
	require.True(t, result.TotalEdges > 0, "cold build should produce edges")

	store, err := OpenStore(result.DBPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify caller edge exists: main → NewRouter
	var beforeCount int
	require.NoError(t, store.db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN chunks src ON e.source_id = src.id
		JOIN chunks tgt ON e.target_id = tgt.id
		WHERE src.file_path = 'cmd/app/main.go'
		  AND tgt.file_path = 'internal/dispatch/router.go'
	`).Scan(&beforeCount))
	require.Greater(t, beforeCount, 0, "should have caller edges after cold build")

	// Edit only the callee file
	require.NoError(t, os.WriteFile(filepath.Join(dispDir, "router.go"), []byte(`package dispatch

func NewRouter() *Router {
	return &Router{Port: 8080}
}

type Router struct {
	Port int
}
`), 0o644))

	// Refresh
	_, err = Refresh(RefreshOptions{RepoPath: dir, DBPath: result.DBPath})
	require.NoError(t, err)

	// Verify caller edge still exists after refresh
	var afterCount int
	require.NoError(t, store.db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN chunks src ON e.source_id = src.id
		JOIN chunks tgt ON e.target_id = tgt.id
		WHERE src.file_path = 'cmd/app/main.go'
		  AND tgt.file_path = 'internal/dispatch/router.go'
	`).Scan(&afterCount))
	assert.Equal(t, beforeCount, afterCount,
		"caller edges should survive refresh of callee file")
}
