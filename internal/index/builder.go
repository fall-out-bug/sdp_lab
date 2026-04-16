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
		chunks, edges, parseErr := ParseFile(path, language)
		if parseErr != nil {
			return nil // skip unparseable files
		}

		// Read file for hash
		data, readErr := os.ReadFile(path)
		if readErr != nil {
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

		// Insert chunks
		for _, chunk := range chunks {
			_, insertErr := store.InsertChunk(chunk)
			if insertErr != nil {
				continue // skip problematic chunks
			}
			result.TotalChunks++
		}

		// Insert edges
		for _, edge := range edges {
			if edge.SourceID > 0 && edge.TargetID > 0 {
				_, _ = store.InsertEdge(edge)
				result.TotalEdges++
			}
		}

		// Store file metadata
		fm := FileMeta{
			Path:        relPath,
			Hash:        fileHash,
			LastIndexed: time.Now().UTC().Format(time.RFC3339),
			Language:    language,
			Loc:         loc,
			IsTest:      IsTestFile(relPath),
			IsGenerated: isGeneratedFile(relPath),
		}
		_ = store.UpsertFileMeta(fm)
		result.TotalFiles++

		langSet[language] = true

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk repo: %w", err)
	}

	// Build language list
	for lang := range langSet {
		result.Languages = append(result.Languages, lang)
	}

	// Store metadata
	_ = store.SetMeta("schema_version", fmt.Sprintf("%d", SchemaVersion))
	_ = store.SetMeta("indexed_at", time.Now().UTC().Format(time.RFC3339))
	_ = store.SetMeta("total_chunks", fmt.Sprintf("%d", result.TotalChunks))
	_ = store.SetMeta("total_files", fmt.Sprintf("%d", result.TotalFiles))
	_ = store.SetMeta("total_edges", fmt.Sprintf("%d", result.TotalEdges))
	_ = store.SetMeta("languages", strings.Join(result.Languages, ","))
	_ = store.SetMeta("embedding_model", "none")

	// Get repo name from directory
	repoName := filepath.Base(opts.RepoPath)
	_ = store.SetMeta("repo_name", repoName)

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
