package strataudit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"sdp_dev/internal/glob"
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
	registry := NewExtractorRegistry(cfg)

	// Pre-compile exclusion patterns for performance
	excludeMatcher := glob.NewMatcher(cfg.Project.Exclude)
	excludeMatcherCI := glob.NewCaseInsensitiveMatcher(cfg.Project.Exclude)

	// Build optimized level matchers (includes sorting)
	levelMatchers := buildLevelMatchers(cfg.Levels)

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
			if isExcludedOptimized(path, excludeMatcher, excludeMatcherCI) {
				return nil
			}
			if !registry.CanHandle(filepath.Ext(path)) {
				return nil
			}

			// Path traversal check
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil
			}
			if evaled, err := filepath.EvalSymlinks(absPath); err == nil {
				absPath = evaled
			}
			if evaled, err := filepath.EvalSymlinks(absDir); err == nil {
				absDir = evaled
			}
			if !strings.HasPrefix(absPath, absDir) {
				return nil
			}

			doc, status, err := processFile(ctx, cfg, store, path, levelMatchers, registry)
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
	New       int
	Updated   int
	Unchanged int
	Errors    []error
}

func processFile(ctx context.Context, cfg *Config, store *SQLiteStore, path string, levelMatchers []sortedLevel, registry *ExtractorRegistry) (*model.Document, docStatus, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}

	// Compute content hash
	contentHash := sha256Hash(data)

	// Extract text via registry
	content, err := registry.Extract(ctx, path, data)
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
	levelID := classifyLevelOptimized(path, levelMatchers)
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
		if err := store.DeleteSectionsForDocument(ctx, existing.ID); err != nil {
			return nil, statusUpdated, fmt.Errorf("delete sections for %s: %w", existing.ID, err)
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
	if err := store.SaveSections(ctx, ChunkSections(doc.ID, content, cfg.Thresholds.ChunkTokenLimit, cfg.Thresholds.ChunkOverlapTokens)); err != nil {
		return nil, "", fmt.Errorf("save sections: %w", err)
	}

	status := statusNew
	if existing != nil {
		status = statusUpdated
	}

	return &doc, status, nil
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
	sections := ChunkSections("", content, chunkTokenLimit, overlapTokens)
	chunks := make([]string, 0, len(sections))
	for _, section := range sections {
		chunks = append(chunks, section.Content)
	}
	return chunks
}

// ChunkSections materializes fallback sections for a document when no logical parser exists yet.
// Offsets are stored as rune offsets in the document content.
func ChunkSections(docID, content string, chunkTokenLimit, overlapTokens int) []model.Section {
	if content == "" {
		return nil
	}

	if chunkTokenLimit <= 0 {
		return []model.Section{buildChunkSection(docID, 0, 0, utf8.RuneCountInString(content), content)}
	}

	charsPerToken := 4
	chunkChars := chunkTokenLimit * charsPerToken
	overlapChars := overlapTokens * charsPerToken

	runes := []rune(content)
	if len(runes) <= chunkChars {
		return []model.Section{buildChunkSection(docID, 0, 0, len(runes), content)}
	}

	var sections []model.Section
	start := 0
	ordinal := 0

	for start < len(runes) {
		end := start + chunkChars
		if end > len(runes) {
			end = len(runes)
		}
		sections = append(sections, buildChunkSection(docID, ordinal, start, end, string(runes[start:end])))
		if end == len(runes) {
			break
		}
		nextStart := end - overlapChars
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
		ordinal++
	}

	return sections
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

func generateSectionID(docID string, ordinal, start, end int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d|%d", docID, ordinal, start, end)))
	return fmt.Sprintf("sec_%x", h[:8])
}

func buildChunkSection(docID string, ordinal, start, end int, content string) model.Section {
	return model.Section{
		ID:           generateSectionID(docID, ordinal, start, end),
		DocumentID:   docID,
		Ordinal:      ordinal,
		CharStart:    start,
		CharEnd:      end,
		Preview:      previewText(content, 140),
		Content:      content,
		ContentHash:  sha256Hash([]byte(content)),
		QualityFlags: []string{"section_parse_fallback"},
	}
}

func previewText(content string, limit int) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit]) + "..."
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

func isSupportedExt(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".markdown", ".pdf", ".docx", ".pptx":
		return true
	}
	return false
}

// PDF and DOCX extraction stubs — minimal implementations for v1

// sortedLevel wraps a LevelConfig with pre-compiled matchers for performance.
type sortedLevel struct {
	Level            LevelConfig
	PatternMatcher   *glob.Matcher
	PatternMatcherCI *glob.CaseInsensitiveMatcher
}

// buildLevelMatchers creates sortedLevel structs with pre-compiled matchers.
func buildLevelMatchers(levels []LevelConfig) []sortedLevel {
	sorted := make([]sortedLevel, len(levels))
	for i, level := range levels {
		patterns := level.Patterns
		sorted[i] = sortedLevel{
			Level:            level,
			PatternMatcher:   glob.NewMatcher(patterns),
			PatternMatcherCI: glob.NewCaseInsensitiveMatcher(patterns),
		}
	}
	// Sort by rank (ascending) for deterministic classification
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Level.Rank < sorted[j].Level.Rank
	})
	return sorted
}

// isExcludedOptimized checks if a path matches any exclusion pattern using pre-compiled matchers.
// It handles basename matching, full path matching, and individual segment matching (for directory exclusions).
func isExcludedOptimized(path string, matcher *glob.Matcher, matcherCI *glob.CaseInsensitiveMatcher) bool {
	cleanPath := filepath.Clean(path)

	// Check basename
	base := filepath.Base(cleanPath)
	if matcher != nil && matcher.MatchAny(base) {
		return true
	}
	if matcherCI != nil && matcherCI.Match(base) {
		return true
	}

	// Check full path
	if matcher != nil && matcher.MatchAny(cleanPath) {
		return true
	}
	if matcherCI != nil && matcherCI.Match(cleanPath) {
		return true
	}

	// Check individual segments (for directory exclusions like "Downloads")
	segments := strings.Split(cleanPath, string(os.PathSeparator))
	for _, segment := range segments {
		if matcher != nil && matcher.MatchAny(segment) {
			return true
		}
		if matcherCI != nil && matcherCI.Match(segment) {
			return true
		}
	}

	return false
}

// classifyLevelOptimized matches file path against level glob patterns using pre-compiled matchers.
// When multiple levels match, the one with the lowest rank wins (more strategic = higher priority).
// Iteration is deterministic because levels are sorted by rank.
func classifyLevelOptimized(path string, levels []sortedLevel) string {
	base := filepath.Base(path)
	matchedRank := -1
	matchedName := ""

	for _, level := range levels {
		// Try case-sensitive match against basename
		if level.PatternMatcher != nil && level.PatternMatcher.MatchAny(base) {
			if matchedRank == -1 || level.Level.Rank < matchedRank {
				matchedRank = level.Level.Rank
				matchedName = level.Level.Name
			}
			continue // Next level
		}

		// Try case-sensitive match against full path
		if level.PatternMatcher != nil && level.PatternMatcher.MatchAny(path) {
			if matchedRank == -1 || level.Level.Rank < matchedRank {
				matchedRank = level.Level.Rank
				matchedName = level.Level.Name
			}
			continue // Next level
		}

		// Try case-insensitive match against basename
		if level.PatternMatcherCI != nil && level.PatternMatcherCI.Match(base) {
			if matchedRank == -1 || level.Level.Rank < matchedRank {
				matchedRank = level.Level.Rank
				matchedName = level.Level.Name
			}
			continue // Next level
		}
	}

	return matchedName
}

func extractPDF(data []byte) (string, error) {
	// v1: basic text extraction attempt using ledongthuc/pdf
	// Falls back gracefully if library not available or extraction fails
	return extractPDFWithLedongthuc(data)
}

func extractDOCX(data []byte) (string, error) {
	// v1: basic DOCX extraction
	return extractDOCXBasic(data)
}
