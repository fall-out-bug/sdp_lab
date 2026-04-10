package extract

import (
	"bufio"
	"context"
	"io/fs"
	"os"
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
	"controller":   "controller",
	"controllers":  "controller",
	"service":      "service",
	"services":     "service",
	"repository":   "repository",
	"repositories": "repository",
	"handler":      "handler",
	"handlers":     "handler",
	"middleware":   "middleware",
	"middlewares":  "middleware",
	"model":        "model",
	"models":       "model",
	"entity":       "entity",
	"entities":     "entity",
}

// dirConventions lists well-known directory names that signal project layout
// conventions. These are reported alongside naming patterns.
var dirConventions = map[string]string{
	"src":      "src_layout",
	"lib":      "lib_layout",
	"cmd":      "cmd_layout",
	"pkg":      "pkg_layout",
	"internal": "internal_layout",
	"api":      "api_layout",
	"web":      "web_layout",
	"app":      "app_layout",
	"bin":      "bin_layout",
	"docs":     "docs_layout",
	"config":   "config_layout",
	"scripts":  "scripts_layout",
	"test":     "test_layout",
	"tests":    "test_layout",
}

// textExtensions lists file extensions that should be treated as text files
// for line counting purposes.
var textExtensions = map[string]bool{
	".scala": true, ".java": true, ".py": true, ".go": true,
	".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".sql": true, ".R": true, ".r": true,
	".xml": true, ".json": true, ".yaml": true, ".yml": true,
	".toml": true, ".md": true, ".txt": true, ".properties": true,
	".conf": true, ".sh": true, ".bat": true, ".css": true, ".html": true,
	".kt": true, ".kts": true, ".c": true, ".h": true, ".cpp": true,
	".rs": true, ".rb": true, ".php": true, ".swift": true,
}

// FileTreeExtractor walks the directory tree, counts files and directories,
// measures maximum depth, counts lines of code, and detects common naming conventions.
type FileTreeExtractor struct{}

// Name implements architect.Extractor.
func (FileTreeExtractor) Name() string { return "filetree" }

// Extract implements architect.Extractor.
func (FileTreeExtractor) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	var (
		totalFiles int
		totalDirs  int
		totalLOC   int
		maxDepth   int
		extCounts  = make(map[string]int)
		seen       = make(map[string]bool)
		dirConvos  = make(map[string]bool)
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
			// Check directory conventions (src, lib, cmd, pkg, internal, etc.).
			lower := strings.ToLower(d.Name())
			for name, label := range dirConventions {
				if lower == name && !dirConvos[label] {
					dirConvos[label] = true
				}
			}
			// Check naming patterns on directories.
			for substr, label := range namingPatterns {
				if strings.Contains(lower, substr) && !seen[label] {
					seen[label] = true
				}
			}
			return nil
		}

		totalFiles++

		// Extension counts - keep the dot (e.g., ".scala", ".java")
		ext := filepath.Ext(d.Name())
		if ext != "" {
			extCounts[ext]++
		}

		// Count lines of code for text-like files
		if textExtensions[ext] {
			totalLOC += countLines(path)
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
	for _, label := range dirConventions {
		if dirConvos[label] && !added[label] {
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
		Metrics: &architect.CodeMetrics{
			TotalFiles: totalFiles,
			TotalLOC:   totalLOC,
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

// countLines counts the number of lines in a file.
func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count
}
