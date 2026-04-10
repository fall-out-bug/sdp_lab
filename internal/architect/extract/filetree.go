package extract

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"sdp_dev/internal/architect"
)

// skipDirs lists directory basenames that should be ignored during tree walks.
var skipDirs = map[string]bool{
	".git":        true,
	"node_modules": true,
	"vendor":      true,
	"__pycache__": true,
	".sdp":        true,
}

// namingPatterns maps directory/file name substrings to architectural pattern
// labels.  Plural forms are included so that e.g. "entities" matches "entity".
var namingPatterns = map[string]string{
	"controller":  "controller",
	"controllers": "controller",
	"service":     "service",
	"services":    "service",
	"repository":  "repository",
	"repositories": "repository",
	"handler":     "handler",
	"handlers":    "handler",
	"middleware":  "middleware",
	"middlewares": "middleware",
	"model":       "model",
	"models":      "model",
	"entity":      "entity",
	"entities":    "entity",
}

// FileTreeExtractor walks the directory tree, counts files and directories,
// measures maximum depth, and detects common naming conventions.
type FileTreeExtractor struct{}

// Name implements architect.Extractor.
func (FileTreeExtractor) Name() string { return "filetree" }

// Extract implements architect.Extractor.
func (FileTreeExtractor) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	var (
		totalFiles int
		totalDirs  int
		maxDepth   int
		extCounts  = make(map[string]int)
		seen       = make(map[string]bool)
	)

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Honour context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rel, _ := filepath.Rel(repoRoot, path)
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			totalDirs++
			depth := strings.Count(rel, string(filepath.Separator)) + 1
			if depth > maxDepth {
				maxDepth = depth
			}
			// Check naming patterns on directories.
			lower := strings.ToLower(d.Name())
			for substr, label := range namingPatterns {
				if strings.Contains(lower, substr) && !seen[label] {
					seen[label] = true
				}
			}
			return nil
		}

		totalFiles++

		// Extension counts.
		ext := strings.TrimPrefix(filepath.Ext(d.Name()), ".")
		if ext != "" {
			extCounts[ext]++
		}

		// Check naming patterns on files (without extension).
		base := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		lower := strings.ToLower(base)
		for substr, label := range namingPatterns {
			if strings.Contains(lower, substr) && !seen[label] {
				seen[label] = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Collect unique labels.
	added := make(map[string]bool)
	var patterns []string
	for _, label := range namingPatterns {
		if seen[label] && !added[label] {
			added[label] = true
			patterns = append(patterns, label)
		}
	}
	// Sort deterministically (namingPatterns iteration order is random).
	sortStrings(patterns)

	return &architect.ProfileFragment{
		FileTree: &architect.FileTreeInfo{
			TotalFiles: totalFiles,
			TotalDirs:  totalDirs,
			MaxDepth:   maxDepth,
			Patterns:   patterns,
			ExtCounts:  extCounts,
		},
	}, nil
}

// sortStrings sorts a slice of strings in-place using insertion sort (avoids
// importing "sort" for a tiny helper).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
