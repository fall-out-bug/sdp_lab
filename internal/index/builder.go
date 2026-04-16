package index

import (
	"fmt"
	"os"
	"path/filepath"
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
	defer store.Close()

	result := &BuildResult{DBPath: dbPath}

	// Track languages seen
	langSet := map[string]bool{}

	// Collect symbolic edges for resolution after all files are inserted.
	var allSymEdges []SymbolicEdge

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
	resolvedCount := resolveAndInsertEdges(store, allSymEdges)
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
	defer store.Close()

	result := &RefreshResult{DBPath: dbPath}

	// Build map of currently indexed files: path -> hash
	indexedFiles, err := buildIndexedFileMap(store)
	if err != nil {
		return nil, fmt.Errorf("load indexed files: %w", err)
	}

	// Track which indexed files we see on disk
	seenOnDisk := map[string]bool{}

	// Collect all symbolic edges across changed/added files for batch resolution.
	var allSymEdges []SymbolicEdge

	// Begin a transaction wrapping the walk phase.
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

	// Detect files removed from disk but still in the index
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

	// Resolve symbolic edges for all changed/added files.
	resolveAndInsertEdges(store, allSymEdges)

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
func resolveAndInsertEdges(store *SQLiteStore, symEdges []SymbolicEdge) int {
	if len(symEdges) == 0 {
		return 0
	}

	// Build the symbol -> ID map from the database
	symMap, err := BuildSymbolIDMap(store)
	if err != nil {
		return 0
	}

	inserted := 0
	for _, se := range symEdges {
		// Resolve source
		sourceKey := se.SourceFile + ":" + se.SourceSymbol
		sourceID, ok := symMap[sourceKey]
		if !ok {
			continue
		}

		// Resolve target (target file may be empty = same file)
		targetFile := se.TargetFile
		if targetFile == "" {
			targetFile = se.SourceFile
		}
		targetKey := targetFile + ":" + se.TargetSymbol
		targetID, ok := symMap[targetKey]
		if !ok {
			continue
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

// extractSymbolicEdges scans the chunks from a file and produces symbolic edges.
// For each function/method chunk, it looks for references to other symbols defined
// in the same file (same-file calls/uses) and produces SymbolicEdge entries.
func extractSymbolicEdges(filePath string, chunks []Chunk) []SymbolicEdge {
	if len(chunks) <= 1 {
		return nil
	}

	// Collect all symbol names defined in this file
	symbols := make(map[string]bool, len(chunks))
	for _, c := range chunks {
		if c.SymbolName != "" && c.Kind != "file" {
			symbols[c.SymbolName] = true
		}
	}

	var edges []SymbolicEdge
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
			// For bare functions (e.g. "NewHandler"), look for the bare name.
			// For method targets (e.g. "Handler.Serve"), the receiver type name
			// is also a candidate -- we check both the short name and the full name.
			shortName := targetSym
			if idx := strings.Index(targetSym, "."); idx >= 0 {
				shortName = targetSym[idx+1:]
			}

			// Use word-boundary matching: the symbol name should appear as a
			// standalone identifier, not as a substring of a larger token.
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

	return edges
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
