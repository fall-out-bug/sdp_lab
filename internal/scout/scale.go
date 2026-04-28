package scout

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/common"
)

// detectScale runs Phase 2: file counts, LOC, entry points, directory stats.
// buildSystem is used to identify source vs config files (may be nil).
func detectScale(root string, buildSystem *string) Scale {
	return detectScaleWithContext(context.Background(), root, buildSystem)
}

// detectScaleWithContext is the context-aware version of detectScale.
func detectScaleWithContext(ctx context.Context, root string, buildSystem *string) Scale {
	var s Scale
	var locValues []int
	dirSet := make(map[string]bool)

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			if common.DefaultMatcher.Match(d.Name(), true) {
				return filepath.SkipDir
			}
			dirSet[filepath.Dir(rel)] = true
			return nil
		}

		if common.DefaultMatcher.Match(d.Name(), false) {
			// Count vendor files separately
			if isInVendorDir(rel) {
				s.VendorFiles++
			}
			return nil
		}

		s.TotalFiles++

		// Binary check: skip files with null bytes in first 512 bytes
		if isBinary(path) {
			return nil
		}

		// Skip large files (>100KB)
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		if info.Size() > 100*1024 {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		base := filepath.Base(path)

		// Categorize file
		isTest := IsTestFile(base)
		isGenerated := isGeneratedFile(base)

		if isTest {
			s.TestFiles++
		} else if isGenerated {
			s.GeneratedFiles++
		} else if _, ok := ExtToLanguage(ext); ok {
			s.SourceFiles++
		}

		// Count LOC for source files
		if !isTest && !isGenerated {
			if lang, ok := ExtToLanguage(ext); ok {
				_ = lang
				loc, lines := countLines(path)
				if lines {
					s.TotalLoc += int64(loc)
					locValues = append(locValues, loc)
				}
			}
		}

		// Track directories and depth
		depth := strings.Count(rel, string(filepath.Separator))
		if depth > s.DepthMax {
			s.DepthMax = depth
		}

		return nil
	})

	s.Directories = len(dirSet)

	// Calculate median LOC
	if len(locValues) > 0 {
		sort.Ints(locValues)
		s.MaxFileLoc = locValues[len(locValues)-1]
		s.MedianFileLoc = locValues[len(locValues)/2]
	}

	// Test ratio
	if s.SourceFiles > 0 {
		s.TestRatio = float64(s.TestFiles) / float64(s.SourceFiles+s.TestFiles)
	}

	return s
}

func isInVendorDir(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts {
		if p == "vendor" || p == "node_modules" {
			return true
		}
	}
	return false
}

func isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return bytes.Contains(buf[:n], []byte{0x00})
}

func isGeneratedFile(name string) bool {
	patterns := []string{".pb.go", ".generated.", ".min.js", ".min.css"}
	for _, p := range patterns {
		if strings.Contains(name, p) {
			return true
		}
	}
	return false
}

func countLines(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	return strings.Count(string(data), "\n"), true
}
