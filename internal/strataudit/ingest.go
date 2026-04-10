package strataudit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"sdp_dev/internal/strataudit/model"
)

// Ingest walks source directories, extracts text, classifies levels, and stores documents.
func Ingest(ctx context.Context, cfg *Config, store *SQLiteStore) (*IngestResult, error) {
	// Save configured levels
	levels := cfgToLevels(cfg)
	if err := store.SaveLevels(ctx, levels); err != nil {
		return nil, fmt.Errorf("save levels: %w", err)
	}

	result := &IngestResult{}
	levelMap := buildLevelMap(cfg.Levels)

	for _, srcDir := range cfg.Project.SourceDirs {
		absDir, err := filepath.Abs(srcDir)
		if err != nil {
			return nil, fmt.Errorf("resolve source dir %s: %w", srcDir, err)
		}

		err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.IsDir() {
				return nil
			}
			if isExcluded(path, cfg.Project.Exclude) {
				return nil
			}
			if !isSupportedExt(path) {
				return nil
			}

			// Path traversal check
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil
			}
			if !strings.HasPrefix(absPath, absDir) {
				return nil
			}

			doc, status, err := processFile(ctx, cfg, store, path, levelMap)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
				return nil
			}
			if doc == nil {
				return nil
			}

			switch status {
			case statusNew:
				result.New++
			case statusUpdated:
				result.Updated++
			case statusUnchanged:
				result.Unchanged++
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", srcDir, err)
		}
	}

	return result, nil
}

type docStatus string

const (
	statusNew       docStatus = "new"
	statusUpdated   docStatus = "updated"
	statusUnchanged docStatus = "unchanged"
)

// IngestResult holds ingestion statistics.
type IngestResult struct {
	New        int
	Updated    int
	Unchanged  int
	Errors     []error
}

func processFile(ctx context.Context, cfg *Config, store *SQLiteStore, path string, levelMap map[string]LevelConfig) (*model.Document, docStatus, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}

	// Compute content hash
	contentHash := sha256Hash(data)

	// Extract text
	content, err := extractText(path, data)
	if err != nil {
		return nil, "", fmt.Errorf("extract text: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		return nil, "", nil // skip empty documents
	}

	// Get file modification time
	info, _ := os.Stat(path)
	var modTime time.Time
	if info != nil {
		modTime = info.ModTime()
	}

	// Check for existing document by path
	existing, _ := store.DocumentByPath(ctx, path)
	if existing != nil && existing.ContentHash == contentHash {
		return existing, statusUnchanged, nil // unchanged
	}

	// Classify level
	levelID := classifyLevel(path, levelMap)
	if levelID == "" {
		return nil, "", nil // no matching level
	}

	version := 1
	if existing != nil {
		version = existing.Version + 1
		// Cascade-delete old entities for updated document
		if err := store.DeleteEntitiesForDocument(ctx, existing.ID); err != nil {
			return nil, statusUpdated, fmt.Errorf("delete entities for %s: %w", existing.ID, err)
		}
	}

	docID := generateDocID(path)
	doc := model.Document{
		ID:             docID,
		Path:           path,
		LevelID:        levelID,
		ContentHash:    contentHash,
		Content:        content,
		Version:        version,
		FileModifiedAt: modTime,
	}

	if err := store.SaveDocuments(ctx, []model.Document{doc}); err != nil {
		return nil, "", fmt.Errorf("save document: %w", err)
	}

	status := statusNew
	if existing != nil {
		status = statusUpdated
	}

	return &doc, status, nil
}

// classifyLevel matches file path against level glob patterns.
func classifyLevel(path string, levelMap map[string]LevelConfig) string {
	base := filepath.Base(path)
	for name, level := range levelMap {
		for _, pattern := range level.Patterns {
			if matched, _ := filepath.Match(pattern, base); matched {
				return name
			}
			// Also try matching against full path for directory-based patterns
			if matched, _ := filepath.Match(pattern, path); matched {
				return name
			}
			// Try case-insensitive match
			if matched, _ := filepath.Match(strings.ToLower(pattern), strings.ToLower(base)); matched {
				return name
			}
		}
	}
	return ""
}

// extractText dispatches to format-specific extractors.
func extractText(path string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".markdown":
		return string(data), nil
	case ".pdf":
		return extractPDF(data)
	case ".docx":
		return extractDOCX(data)
	default:
		return "", fmt.Errorf("unsupported format: %s", ext)
	}
}

// ChunkContent splits content into overlapping chunks of approximately chunkTokenLimit tokens.
// Uses a rough heuristic: 1 token ≈ 4 characters.
func ChunkContent(content string, chunkTokenLimit, overlapTokens int) []string {
	if chunkTokenLimit <= 0 {
		return []string{content}
	}

	charsPerToken := 4
	chunkChars := chunkTokenLimit * charsPerToken
	overlapChars := overlapTokens * charsPerToken

	if utf8.RuneCountInString(content) <= chunkChars {
		return []string{content}
	}

	runes := []rune(content)
	var chunks []string
	start := 0

	for start < len(runes) {
		end := start + chunkChars
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
		start = end - overlapChars
		if start <= 0 {
			start = end // prevent infinite loop
		}
	}

	return chunks
}

// helpers

func sha256Hash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func generateDocID(path string) string {
	h := sha256.Sum256([]byte(path))
	return fmt.Sprintf("doc_%x", h[:8])
}

func cfgToLevels(cfg *Config) []model.Level {
	levels := make([]model.Level, len(cfg.Levels))
	for i, lc := range cfg.Levels {
		levels[i] = model.Level{
			ID:          lc.Name,
			Name:        lc.Name,
			Rank:        lc.Rank,
			Description: lc.Description,
			Patterns:    lc.Patterns,
		}
	}
	return levels
}

func buildLevelMap(levels []LevelConfig) map[string]LevelConfig {
	m := make(map[string]LevelConfig, len(levels))
	for _, l := range levels {
		m[l.Name] = l
	}
	return m
}

func isExcluded(path string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, filepath.Base(path)); matched {
			return true
		}
		// Handle glob with path separators
		if strings.Contains(p, "/") || strings.Contains(p, string(os.PathSeparator)) {
			if matched, _ := filepath.Match(p, path); matched {
				return true
			}
		}
	}
	return false
}

func isSupportedExt(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".markdown", ".pdf", ".docx":
		return true
	}
	return false
}

// PDF and DOCX extraction stubs — minimal implementations for v1

func extractPDF(data []byte) (string, error) {
	// v1: basic text extraction attempt using ledongthuc/pdf
	// Falls back gracefully if library not available or extraction fails
	return extractPDFWithLedongthuc(data)
}

func extractDOCX(data []byte) (string, error) {
	// v1: basic DOCX extraction
	return extractDOCXBasic(data)
}
