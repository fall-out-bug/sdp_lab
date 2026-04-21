// Package common provides shared utilities used across SDP tools
// (scout, metrics, index, etc.) to ensure consistent repository surface analysis.
package common

import (
	"path/filepath"
	"strings"
)

// DefaultExcludes lists directory and file names that should be excluded from
// repository surface analysis. Shared by scout, metrics, and index.
var DefaultExcludes = []string{
	// VCS
	".git",
	// Language-specific vendor/dependency directories
	"vendor",
	"node_modules",
	"__pycache__",
	// Build output directories
	"target",
	"build",
	"dist",
	"out",
	".next",
	".nuxt",
	// Infrastructure
	".terraform",
	".gradle",
	".mvn",
	// AI coding harness config dirs (not source code)
	".claude",
	".cursor",
	".opencode",
	// Worktrees and archives (large, non-source)
	".worktrees",
	"archive",
	// Deployment and spec directories (YAML/JSON manifests, not source)
	"deploy",
	"specs",
}

// generatedPatterns matches filenames that are typically auto-generated.
var generatedPatterns = []string{
	".pb.go",
	".generated.",
	".min.js",
	".min.css",
}

// lockExtensions matches file suffixes that are lock files (not source).
var lockSuffixes = []string{
	".lock",
	".sum",
	"-lock.json",
}

// Matcher checks whether a file or directory should be excluded from analysis.
type Matcher struct {
	dirNames     map[string]bool
	genPatterns  []string
	lockSuffixes []string
}

// DefaultMatcher is a pre-built Matcher using DefaultExcludes.
var DefaultMatcher = newMatcher(DefaultExcludes)

func newMatcher(excludes []string) *Matcher {
	m := &Matcher{
		dirNames:     make(map[string]bool, len(excludes)),
		genPatterns:  generatedPatterns,
		lockSuffixes: lockSuffixes,
	}
	for _, name := range excludes {
		m.dirNames[name] = true
	}
	return m
}

// Match reports whether a file or directory should be excluded.
// isDir indicates whether the path refers to a directory.
func (m *Matcher) Match(name string, isDir bool) bool {
	base := filepath.Base(name)

	// Check directory/file name exclusions
	if m.dirNames[base] {
		return true
	}

	// Generated file patterns (file mode only is fine, dir names don't apply)
	for _, pat := range m.genPatterns {
		if strings.Contains(base, pat) {
			return true
		}
	}

	// Lock file suffixes (e.g., ".lock", ".sum", "-lock.json")
	for _, suffix := range m.lockSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}

	return false
}
