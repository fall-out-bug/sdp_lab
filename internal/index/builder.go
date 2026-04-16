package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"sdp_dev/internal/common"
)

// ColdBuild performs a full index build of the repository at repoPath.
// It walks the file tree, parses source files, extracts chunks and edges,
// and stores everything in a SQLite database at .sdp/index.db.
func ColdBuild(opts BuildOptions) (*BuildResult, error) {
	start := time.Now()

	// Determine DB path
	dbPath := opts.DBPath
	if dbPath == "" {
		var err error
		dbPath, err = EnsureSdpDir(opts.RepoPath)
		if err != nil {
			return nil, fmt.Errorf("ensure .sdp dir: %w", err)
		}
	}

	// Default max file size: 100KB
	maxSize := opts.MaxFileSizeBytes
	if maxSize <= 0 {
		maxSize = 100 * 1024
	}

	// Build language filter set
	langFilter := map[string]bool{}
	for _, l := range opts.Languages {
		langFilter[l] = true
	}

	// Open store (creates schema if needed)
	store, err := OpenStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = store.Close() }()

	result := &BuildResult{DBPath: dbPath}

	// Track languages seen
	langSet := map[string]bool{}

	// Collect symbolic edges for resolution after all files are inserted.
	var allSymEdges []SymbolicEdge
	fileImports := make(map[string][]string) // filePath -> import paths

	// Begin a transaction wrapping the entire walk phase.
	tx, txErr := store.Begin()
	if txErr != nil {
		return nil, fmt.Errorf("begin transaction: %w", txErr)
	}
	defer RollbackTx(tx)

	// Walk the repo
	err = filepath.Walk(opts.RepoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}

		// Get relative path
		relPath, relErr := filepath.Rel(opts.RepoPath, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// Skip .sdp directory itself
		if relPath == ".sdp" && info.IsDir() {
			return filepath.SkipDir
		}

		// Directory exclusion
		if info.IsDir() {
			if common.DefaultMatcher.Match(relPath, true) {
				return filepath.SkipDir
			}
			return nil
		}

		// File exclusion checks
		if shouldExcludeFile(path, relPath, info, maxSize) {
			return nil
		}

		// Detect language
		language := DetectLanguage(relPath)
		if language == "" {
			// Not a recognized source file
			return nil
		}

		// Apply language filter
		if len(langFilter) > 0 && !langFilter[language] {
			return nil
		}

		// Parse file
		chunks, _, parseErr := ParseFile(path, language)
		if parseErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("parse %s: %w", relPath, parseErr))
			return nil // skip unparseable files
		}

		// Read file for hash
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("read %s: %w", relPath, readErr))
			return nil
		}
		fileHash := contentHash(string(data))
		loc := len(strings.Split(string(data), "\n"))

		// Fix chunk file paths to be relative
		for i := range chunks {
			chunks[i].FilePath = relPath
			if chunks[i].Scope != "" && strings.Contains(chunks[i].Scope, path) {
				chunks[i].Scope = strings.Replace(chunks[i].Scope, path, relPath, 1)
			}
		}

		// Extract symbolic edges from the parsed chunks
		symEdges := extractSymbolicEdges(relPath, chunks)
		allSymEdges = append(allSymEdges, symEdges...)

		// Extract imports for cross-file resolution disambiguation
		if language == "go" {
			fileImports[relPath] = extractGoImports(string(data))
		}

		// Insert chunks within the transaction
		for _, chunk := range chunks {
			_, insertErr := InsertChunkTx(tx, chunk)
			if insertErr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("insert chunk %s/%s: %w", relPath, chunk.SymbolName, insertErr))
				continue
			}
			result.TotalChunks++
		}

		// Store file metadata within transaction
		fm := FileMeta{
			Path:        relPath,
			Hash:        fileHash,
			LastIndexed: time.Now().UTC().Format(time.RFC3339),
			Language:    language,
			Loc:         loc,
			IsTest:      IsTestFile(relPath),
			IsGenerated: isGeneratedFile(relPath),
		}
		if fmErr := UpsertFileMetaTx(tx, fm); fmErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("upsert file meta %s: %w", relPath, fmErr))
		}
		result.TotalFiles++

		langSet[language] = true

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk repo: %w", err)
	}

	// Commit the walk transaction
	if commitErr := CommitTx(tx); commitErr != nil {
		return nil, fmt.Errorf("commit walk transaction: %w", commitErr)
	}

	// Build language list
	for lang := range langSet {
		result.Languages = append(result.Languages, lang)
	}

	// Resolve symbolic edges to ID-based edges and insert them.
	// This must happen after commit so all chunks have assigned IDs.
	resolvedCount := resolveAndInsertEdges(store, allSymEdges, fileImports)
	result.TotalEdges = resolvedCount

	// Store metadata (outside the main transaction, these are simple writes)
	if mErr := store.SetMeta("schema_version", fmt.Sprintf("%d", SchemaVersion)); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta schema_version: %w", mErr))
	}
	if mErr := store.SetMeta("indexed_at", time.Now().UTC().Format(time.RFC3339)); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta indexed_at: %w", mErr))
	}
	if mErr := store.SetMeta("total_chunks", fmt.Sprintf("%d", result.TotalChunks)); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta total_chunks: %w", mErr))
	}
	if mErr := store.SetMeta("total_files", fmt.Sprintf("%d", result.TotalFiles)); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta total_files: %w", mErr))
	}
	if mErr := store.SetMeta("total_edges", fmt.Sprintf("%d", result.TotalEdges)); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta total_edges: %w", mErr))
	}
	if mErr := store.SetMeta("languages", strings.Join(result.Languages, ",")); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta languages: %w", mErr))
	}
	if mErr := store.SetMeta("embedding_model", "none"); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta embedding_model: %w", mErr))
	}

	// Get repo name from directory
	repoName := filepath.Base(opts.RepoPath)
	if mErr := store.SetMeta("repo_name", repoName); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta repo_name: %w", mErr))
	}

	result.Duration = time.Since(start)
	return result, nil
}

// shouldExcludeFile returns true if a file should be skipped during indexing.
func shouldExcludeFile(absPath, relPath string, info os.FileInfo, maxSize int64) bool {
	// Skip excluded directories/files via common matcher
	if common.DefaultMatcher.Match(relPath, false) {
		return true
	}

	// Skip secret files
	if IsSecretFile(relPath) {
		return true
	}

	// Skip binary files
	if IsBinaryFile(absPath) {
		return true
	}

	// Skip files exceeding max size
	if info.Size() > maxSize {
		return true
	}

	// Skip generated files
	if isGeneratedFile(relPath) {
		return true
	}

	return false
}

// isGeneratedFile returns true if the file looks auto-generated.
func isGeneratedFile(path string) bool {
	generatedSuffixes := []string{
		".pb.go",
		".generated.",
		".min.js",
		".min.css",
		"_generated.go",
		".graphql.go",
	}
	base := filepath.Base(path)
	for _, suffix := range generatedSuffixes {
		if strings.Contains(base, suffix) {
			return true
		}
	}
	return false
}

// Refresh performs an incremental index update. It compares file hashes on disk
// with those stored in the index and re-indexes only changed, added, or removed files.
// The index database must already exist (run ColdBuild first).
//
// Algorithm:
//  1. Walk all source files, compute SHA256 hash.
//  2. Compare with stored hash in the files table.
//  3. For changed/added files: parse, delete old chunks (CASCADE edges), insert new chunks.
//  4. For deleted files: remove file metadata and chunks.
//  5. Update index metadata.
func Refresh(opts RefreshOptions) (*RefreshResult, error) {
	start := time.Now()

	// Determine DB path
	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = filepath.Join(opts.RepoPath, ".sdp", "index.db")
	}

	// Verify the index exists
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("index database not found at %s (run 'sdp index build' first): %w", dbPath, err)
	}

	// Default max file size: 100KB
	maxSize := opts.MaxFileSizeBytes
	if maxSize <= 0 {
		maxSize = 100 * 1024
	}

	// Build language filter set
	langFilter := map[string]bool{}
	for _, l := range opts.Languages {
		langFilter[l] = true
	}

	// Open existing store
	store, err := OpenStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = store.Close() }()

	result := &RefreshResult{DBPath: dbPath}

	// Build map of currently indexed files: path -> hash
	indexedFiles, err := buildIndexedFileMap(store)
	if err != nil {
		return nil, fmt.Errorf("load indexed files: %w", err)
	}

	// Track which indexed files we see on disk
	seenOnDisk := map[string]bool{}
	// Track changed/removed files so we can repair incoming edges from callers
	changedFiles := map[string]bool{}

	// Collect all symbolic edges across changed/added files for batch resolution.
	var allSymEdges []SymbolicEdge
	fileImports := make(map[string][]string) // filePath -> import paths

	// --- Pass 1: detect changed/removed files without mutating the DB ---
	err = filepath.Walk(opts.RepoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		relPath, relErr := filepath.Rel(opts.RepoPath, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == ".sdp" && info.IsDir() {
			return filepath.SkipDir
		}
		if info.IsDir() {
			if common.DefaultMatcher.Match(relPath, true) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldExcludeFile(path, relPath, info, maxSize) {
			return nil
		}
		language := DetectLanguage(relPath)
		if language == "" {
			return nil
		}
		if len(langFilter) > 0 && !langFilter[language] {
			return nil
		}
		result.FilesChecked++
		seenOnDisk[relPath] = true

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		fileHash := contentHash(string(data))
		storedHash, wasIndexed := indexedFiles[relPath]
		if wasIndexed && storedHash == fileHash {
			return nil // unchanged
		}
		changedFiles[relPath] = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk repo (pass 1): %w", err)
	}

	// Detect removed files
	for relPath := range indexedFiles {
		if !seenOnDisk[relPath] {
			changedFiles[relPath] = true
		}
	}

	// Query callers BEFORE any mutations — edges still intact.
	callerFiles := map[string]bool{}
	if len(changedFiles) > 0 {
		readTx, txErr := store.Begin()
		if txErr != nil {
			return nil, fmt.Errorf("begin caller query transaction: %w", txErr)
		}
		callerFiles = findCallerFiles(readTx, changedFiles)
		RollbackTx(readTx) // read-only, just discard
	}

	// Reset seenOnDisk for pass 2
	seenOnDisk = map[string]bool{}

	// --- Pass 2: mutate — delete old chunks, insert new ones ---
	tx, txErr := store.Begin()
	if txErr != nil {
		return nil, fmt.Errorf("begin transaction: %w", txErr)
	}
	defer RollbackTx(tx)

	// Walk the repo and detect changes
	err = filepath.Walk(opts.RepoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}

		// Get relative path
		relPath, relErr := filepath.Rel(opts.RepoPath, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// Skip .sdp directory itself
		if relPath == ".sdp" && info.IsDir() {
			return filepath.SkipDir
		}

		// Directory exclusion
		if info.IsDir() {
			if common.DefaultMatcher.Match(relPath, true) {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply exclusion rules (secrets, binary, size, generated)
		if shouldExcludeFile(path, relPath, info, maxSize) {
			return nil
		}

		// Detect language
		language := DetectLanguage(relPath)
		if language == "" {
			return nil
		}

		// Apply language filter
		if len(langFilter) > 0 && !langFilter[language] {
			return nil
		}

		result.FilesChecked++
		seenOnDisk[relPath] = true

		// Read file content and compute hash
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("read %s: %w", relPath, readErr))
			return nil
		}
		fileHash := contentHash(string(data))

		// Check if file is already indexed with same hash
		storedHash, wasIndexed := indexedFiles[relPath]
		if wasIndexed && storedHash == fileHash {
			return nil // unchanged, skip
		}

		// File is new or changed -- re-index it
		if wasIndexed {
			result.FilesUpdated++
		} else {
			result.FilesAdded++
		}

		// Delete old chunks for this file (CASCADE handles edges)
		if _, delErr := DeleteChunksByFileTx(tx, relPath); delErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("delete chunks %s: %w", relPath, delErr))
			return nil
		}

		// Parse file
		chunks, _, parseErr := ParseFile(path, language)
		if parseErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("parse %s: %w", relPath, parseErr))
			return nil
		}

		// Fix chunk file paths to be relative
		for i := range chunks {
			chunks[i].FilePath = relPath
			if chunks[i].Scope != "" && strings.Contains(chunks[i].Scope, path) {
				chunks[i].Scope = strings.Replace(chunks[i].Scope, path, relPath, 1)
			}
		}

		// Extract symbolic edges from the parsed chunks
		symEdges := extractSymbolicEdges(relPath, chunks)
		allSymEdges = append(allSymEdges, symEdges...)

		// Extract imports for cross-file resolution disambiguation
		if language == "go" {
			fileImports[relPath] = extractGoImports(string(data))
		}

		// Insert chunks within transaction
		for _, chunk := range chunks {
			_, insertErr := InsertChunkTx(tx, chunk)
			if insertErr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("insert chunk %s/%s: %w", relPath, chunk.SymbolName, insertErr))
				continue
			}
		}

		// Update file metadata within transaction
		loc := len(strings.Split(string(data), "\n"))
		fm := FileMeta{
			Path:        relPath,
			Hash:        fileHash,
			LastIndexed: time.Now().UTC().Format(time.RFC3339),
			Language:    language,
			Loc:         loc,
			IsTest:      IsTestFile(relPath),
			IsGenerated: isGeneratedFile(relPath),
		}
		if fmErr := UpsertFileMetaTx(tx, fm); fmErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("upsert file meta %s: %w", relPath, fmErr))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk repo: %w", err)
	}

	// Delete removed files (detected in pass 1)
	for relPath := range indexedFiles {
		if !seenOnDisk[relPath] {
			if _, delErr := DeleteChunksByFileTx(tx, relPath); delErr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("delete chunks for removed %s: %w", relPath, delErr))
				continue
			}
			if fmErr := DeleteFileMetaTx(tx, relPath); fmErr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("delete file meta %s: %w", relPath, fmErr))
			}
			result.FilesRemoved++
		}
	}

	// Commit the walk transaction
	if commitErr := CommitTx(tx); commitErr != nil {
		return nil, fmt.Errorf("commit refresh transaction: %w", commitErr)
	}

	// Repair incoming edges from unchanged callers.
	// Caller files were discovered in pass 1 (before any mutations).
	// CASCADE deleted edges from callers to changed files,
	// so re-extract edges from callers so they point to the new chunks.
	for callerFile := range callerFiles {
		if changedFiles[callerFile] {
			continue // already handled above
		}
		absPath := filepath.Join(opts.RepoPath, callerFile)
		chunks, _, parseErr := ParseFile(absPath, DetectLanguage(callerFile))
		if parseErr != nil {
			continue
		}
		for i := range chunks {
			chunks[i].FilePath = callerFile
		}
		symEdges := extractSymbolicEdges(callerFile, chunks)
		allSymEdges = append(allSymEdges, symEdges...)

		data, readErr := os.ReadFile(absPath)
		if readErr == nil && DetectLanguage(callerFile) == "go" {
			fileImports[callerFile] = extractGoImports(string(data))
		}
	}

	// Resolve symbolic edges for all changed/added files and repaired callers.
	resolveAndInsertEdges(store, allSymEdges, fileImports)

	// Get final counts
	result.TotalChunks, _ = store.CountChunks()
	result.TotalFiles, _ = store.CountFiles()

	// Update metadata
	if mErr := store.SetMeta("indexed_at", time.Now().UTC().Format(time.RFC3339)); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta indexed_at: %w", mErr))
	}
	if mErr := store.SetMeta("total_chunks", fmt.Sprintf("%d", result.TotalChunks)); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta total_chunks: %w", mErr))
	}
	if mErr := store.SetMeta("total_files", fmt.Sprintf("%d", result.TotalFiles)); mErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("set meta total_files: %w", mErr))
	}

	result.Duration = time.Since(start)
	return result, nil
}

// buildIndexedFileMap returns a map of file_path -> hash from the files table.
func buildIndexedFileMap(store *SQLiteStore) (map[string]string, error) {
	return store.LoadFileHashMap()
}

// resolveAndInsertEdges resolves symbolic edges to chunk IDs and inserts them.
// Returns the number of edges successfully inserted.
//
// For edges where TargetFile is set (same-file edges), it resolves directly.
// For edges where TargetFile is empty (cross-file edges), it searches the
// full symbol map for any file that defines the target symbol.
func resolveAndInsertEdges(store *SQLiteStore, symEdges []SymbolicEdge, fileImports map[string][]string) int {
	if len(symEdges) == 0 {
		return 0
	}

	// Build the symbol -> ID map from the database
	symMap, err := BuildSymbolIDMap(store)
	if err != nil {
		return 0
	}

	// Build a reverse index: symbolName -> list of "file:symbol" keys
	// This allows cross-file lookups by bare symbol name.
	symNameIndex := make(map[string][]string)
	for key := range symMap {
		idx := strings.Index(key, ":")
		if idx < 0 {
			continue
		}
		symName := key[idx+1:]
		symNameIndex[symName] = append(symNameIndex[symName], key)

		// Also index the short name (after dot) for method targets
		// e.g. "Handler.Serve" -> index both "Handler.Serve" and "Serve"
		if dotIdx := strings.Index(symName, "."); dotIdx >= 0 {
			short := symName[dotIdx+1:]
			if short != symName {
				symNameIndex[short] = append(symNameIndex[short], key)
			}
		}
	}

	inserted := 0
	for _, se := range symEdges {
		// Resolve source
		sourceKey := se.SourceFile + ":" + se.SourceSymbol
		sourceID, ok := symMap[sourceKey]
		if !ok {
			continue
		}

		var targetID int64

		if se.TargetFile != "" {
			// Same-file edge: resolve directly
			targetKey := se.TargetFile + ":" + se.TargetSymbol
			targetID, ok = symMap[targetKey]
			if !ok {
				continue
			}
		} else {
			// Cross-file edge: TargetFile is empty, search across all files.
			imports := fileImports[se.SourceFile]
			targetID = resolveCrossFileTarget(se.TargetSymbol, se.SourceFile, imports, symMap, symNameIndex)
			if targetID == 0 {
				continue
			}
		}

		if sourceID == targetID {
			continue // skip self-edges
		}

		_, err := store.InsertEdge(Edge{
			SourceID: sourceID,
			TargetID: targetID,
			Relation: se.Relation,
			Weight:   se.Weight,
		})
		if err == nil {
			inserted++
		}
	}

	return inserted
}

// resolveCrossFileTarget finds a chunk ID for a symbol across all files.
// It uses the source file's import list to disambiguate when multiple files
// define the same symbol name. Candidates whose directory matches an import
// path suffix are preferred over arbitrary matches.
func resolveCrossFileTarget(targetSymbol, sourceFile string, imports []string, symMap map[string]int64, symNameIndex map[string][]string) int64 {
	candidates := symNameIndex[targetSymbol]

	// Phase 1: prefer candidates from imported packages
	if len(imports) > 0 {
		for _, key := range candidates {
			id := symMap[key]
			if id == 0 {
				continue
			}
			idx := strings.Index(key, ":")
			if idx < 0 {
				continue
			}
			candidateFile := key[:idx]
			if candidateFile == sourceFile {
				continue
			}
			if fileMatchesImport(candidateFile, imports) {
				return id
			}
		}
	}

	// Phase 2: prefer any candidate from a different file
	for _, key := range candidates {
		id := symMap[key]
		if id == 0 {
			continue
		}
		idx := strings.Index(key, ":")
		if idx < 0 {
			continue
		}
		candidateFile := key[:idx]
		if candidateFile != sourceFile {
			return id
		}
	}

	// Fall back to any candidate, even same file
	for _, key := range candidates {
		if id := symMap[key]; id > 0 {
			return id
		}
	}

	return 0
}

// fileMatchesImport checks if a candidate file's directory path matches
// any of the source file's import paths. Import paths like
// "example.com/internal/dispatch" should match files in "internal/dispatch/".
func fileMatchesImport(candidateFile string, imports []string) bool {
	candidateDir := candidateFile
	if idx := strings.LastIndex(candidateFile, "/"); idx >= 0 {
		candidateDir = candidateFile[:idx]
	}

	for _, imp := range imports {
		// Match import path suffix: "example.com/internal/dispatch" matches
		// candidate directory "internal/dispatch"
		if strings.HasSuffix(imp, "/"+candidateDir) || imp == candidateDir {
			return true
		}
		// Also check if the import path ends with the candidate dir
		if strings.HasSuffix(imp, candidateDir) {
			return true
		}
	}
	return false
}

// extractSymbolicEdges scans the chunks from a file and produces symbolic edges.
// For each function/method chunk, it looks for:
//   - references to other symbols defined in the same file (same-file calls/uses)
//   - references to symbols NOT defined in this file (cross-file calls)
//
// Cross-file edges have TargetFile="" and are resolved later by
// resolveAndInsertEdges using the full symbol map across all files.
func extractSymbolicEdges(filePath string, chunks []Chunk) []SymbolicEdge {
	if len(chunks) == 0 {
		return nil
	}

	// Collect all symbol names defined in this file
	symbols := make(map[string]bool, len(chunks))
	// Also track short names (after the dot) to avoid duplicating cross-file refs
	// that are already covered by same-file edges.
	shortNameToFull := make(map[string]string, len(chunks))
	for _, c := range chunks {
		if c.SymbolName != "" && c.Kind != "file" {
			symbols[c.SymbolName] = true
			short := c.SymbolName
			if idx := strings.Index(c.SymbolName, "."); idx >= 0 {
				short = c.SymbolName[idx+1:]
			}
			shortNameToFull[short] = c.SymbolName
		}
	}

	var edges []SymbolicEdge

	// Phase 1: same-file edges (unchanged logic)
	for _, c := range chunks {
		if c.Kind != "function" && c.Kind != "method" {
			continue
		}
		if c.SymbolName == "" || c.Content == "" {
			continue
		}

		for targetSym := range symbols {
			if targetSym == c.SymbolName {
				continue // no self-edge
			}

			// Check if the function/method body references this symbol.
			shortName := targetSym
			if idx := strings.Index(targetSym, "."); idx >= 0 {
				shortName = targetSym[idx+1:]
			}

			if containsIdentifier(c.Content, targetSym) || containsIdentifier(c.Content, shortName) {
				relation := "calls"
				if strings.HasPrefix(targetSym, c.SymbolName) {
					continue
				}
				edges = append(edges, SymbolicEdge{
					SourceFile:   filePath,
					SourceSymbol: c.SymbolName,
					TargetSymbol: targetSym,
					Relation:     relation,
					Weight:       1.0,
				})
			}
		}
	}

	// Phase 2: cross-file edges
	// Extract capitalized identifiers from function/method bodies that are NOT
	// defined in this file. These are candidates for cross-file references.
	for _, c := range chunks {
		if c.Kind != "function" && c.Kind != "method" {
			continue
		}
		if c.SymbolName == "" || c.Content == "" {
			continue
		}

		refs := extractExternalRefs(c.Content, symbols)
		for _, ref := range refs {
			// Skip if this reference is covered by a same-file symbol's short name.
			if _, isLocal := shortNameToFull[ref]; isLocal {
				continue
			}
			// Avoid self-edges
			if ref == c.SymbolName {
				continue
			}
			short := c.SymbolName
			if idx := strings.Index(c.SymbolName, "."); idx >= 0 {
				short = c.SymbolName[idx+1:]
			}
			if ref == short {
				continue
			}

			edges = append(edges, SymbolicEdge{
				SourceFile:   filePath,
				SourceSymbol: c.SymbolName,
				TargetFile:   "", // unknown, resolved later
				TargetSymbol: ref,
				Relation:     "calls",
				Weight:       0.5, // lower weight than same-file edges
			})
		}
	}

	return edges
}

// goImportRe matches quoted strings in import declarations.
var goImportRe = regexp.MustCompile(`"([^"]+)"`)

// findCallerFiles queries the edges table to find files that had edges
// pointing to any chunk in the changed files. Returns a set of caller
// file paths. Must be called BEFORE the transaction commits (while edges
// still exist).
func findCallerFiles(tx *sql.Tx, changedFiles map[string]bool) map[string]bool {
	callers := map[string]bool{}

	var args []any
	placeholders := ""
	i := 0
	for f := range changedFiles {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, f)
		i++
	}

	query := `SELECT DISTINCT src.file_path
		FROM edges e
		JOIN chunks src ON e.source_id = src.id
		JOIN chunks tgt ON e.target_id = tgt.id
		WHERE tgt.file_path IN (` + placeholders + `)`

	rows, err := tx.Query(query, args...)
	if err != nil {
		return callers
	}
	defer rows.Close()

	for rows.Next() {
		var fp string
		if err := rows.Scan(&fp); err == nil {
			callers[fp] = true
		}
	}
	return callers
}

// extractGoImports parses import paths from Go source content.
// Returns the import paths (e.g. "example.com/internal/dispatch").
func extractGoImports(content string) []string {
	var imports []string
	inImportBlock := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !inImportBlock {
			if strings.HasPrefix(trimmed, "import") {
				matches := goImportRe.FindAllStringSubmatch(trimmed, -1)
				for _, m := range matches {
					imports = append(imports, m[1])
				}
				if strings.Contains(trimmed, "(") && !strings.Contains(trimmed, ")") {
					inImportBlock = true
				}
				continue
			}
			continue
		}
		if strings.Contains(trimmed, ")") {
			inImportBlock = false
		}
		matches := goImportRe.FindAllStringSubmatch(trimmed, -1)
		for _, m := range matches {
			imports = append(imports, m[1])
		}
	}

	return imports
}

// externalIdentRe matches capitalized identifiers (exported symbols) followed
// by '(' or '{' or '.' or whitespace. This catches function calls, type
// references, and qualified accesses like pkg.Func.
var externalIdentRe = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\b`)

// extractExternalRefs finds capitalized identifiers in content that are not
// present in the localSymbols set. Returns deduplicated symbol names.
func extractExternalRefs(content string, localSymbols map[string]bool) []string {
	matches := externalIdentRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool, len(matches))
	var result []string

	// Language keywords and builtins to skip
	skip := map[string]bool{
		"true": true, "false": true, "nil": true,
		"True": true, "False": true, "None": true,
		"String": true, "Int": true, "Float": true, "Bool": true,
		"Error": true, "Printf": true, "Sprintf": true, "Fprintf": true,
		"Println": true, "Print": true,
		"Make": true, "New": true, "Len": true, "Cap": true,
		"Append": true, "Copy": true, "Delete": true, "Close": true,
		"Panic": true, "Recover": true,
		"Return": true, "Defer": true, "Go": true, "Select": true,
		"Range": true, "Type": true, "Map": true, "Chan": true,
		"Func": true, "Interface": true, "Struct": true,
		"Package": true, "Import": true,
		"IF": true, "ELSE": true, "FOR": true, "WHILE": true,
		"SWITCH": true, "CASE": true, "DEFAULT": true,
		"BREAK": true, "CONTINUE": true, "GOTO": true,
		"VAR": true, "CONST": true,
		"ASYNC": true, "AWAIT": true, "CLASS": true, "EXPORT": true,
		"IMPORT": true, "FROM": true, "THROW": true, "TRY": true,
		"CATCH": true, "FINALLY": true, "NEW": true, "DELETE": true,
		"TYPEOF": true, "INSTANCEOF": true, "VOID": true,
		"SELF": true, "SUPER": true, "WITH": true, "YIELD": true,
	}

	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		// Skip if it's defined locally
		if localSymbols[name] {
			continue
		}
		// Also check if the short name (after dot) matches a local symbol
		if idx := strings.Index(name, "."); idx >= 0 {
			short := name[idx+1:]
			if localSymbols[short] {
				continue
			}
		}
		// Skip language keywords and builtins
		if skip[name] {
			continue
		}
		// Skip very short names (likely not meaningful symbols)
		if len(name) < 2 {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}

// containsIdentifier checks if the identifier appears as a standalone token in
// the content. It avoids matching substrings by checking that the identifier is
// preceded and followed by a non-identifier character (or is at the boundary).
func containsIdentifier(content, ident string) bool {
	if len(ident) == 0 {
		return false
	}
	idx := 0
	for {
		pos := strings.Index(content[idx:], ident)
		if pos < 0 {
			return false
		}
		absPos := idx + pos

		// Check character before the match
		if absPos > 0 {
			ch := content[absPos-1]
			if isIdentChar(ch) {
				idx = absPos + len(ident)
				continue
			}
		}

		// Check character after the match
		afterPos := absPos + len(ident)
		if afterPos < len(content) {
			ch := content[afterPos]
			if isIdentChar(ch) {
				idx = absPos + len(ident)
				continue
			}
		}

		return true
	}
}

// isIdentChar returns true if the byte is a valid Go identifier character.
func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}
