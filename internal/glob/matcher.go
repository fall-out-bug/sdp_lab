// Package glob provides optimized glob pattern matching for large codebases.
// It pre-compiles patterns and uses efficient matching strategies.
package glob

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// CompiledPattern represents a pre-compiled glob pattern for efficient matching.
type CompiledPattern struct {
	original    string
	isLiteral   bool
	literal     string
	hasWildcard bool
	prefix      string
	suffix      string
	regex       *regexp.Regexp
	regexOnce   sync.Once
	regexErr    error
}

// Compile creates a new CompiledPattern from a glob pattern.
func Compile(pattern string) *CompiledPattern {
	cp := &CompiledPattern{
		original: pattern,
	}

	// Check if it's a literal (no wildcards)
	if !strings.ContainsAny(pattern, "*?[") {
		cp.isLiteral = true
		cp.literal = pattern
		return cp
	}

	cp.hasWildcard = true

	// Extract prefix (up to first wildcard)
	if idx := strings.IndexAny(pattern, "*?["); idx >= 0 {
		cp.prefix = pattern[:idx]
	}

	// Extract suffix (after last wildcard)
	// For simple patterns like "*.go", we can use this optimization
	if strings.Count(pattern, "*") == 1 && !strings.Contains(pattern, "?") && !strings.Contains(pattern, "[") {
		// Pattern like "*.go" or "test_*"
		if strings.HasPrefix(pattern, "*") {
			cp.suffix = pattern[1:]
		}
	}

	return cp
}

// Match checks if the path matches this pattern.
// It's safe to call from multiple goroutines.
func (cp *CompiledPattern) Match(path string) (bool, error) {
	// Fast path: literal match
	if cp.isLiteral {
		return path == cp.literal, nil
	}

	// Fast path: prefix check for quick rejection
	if cp.prefix != "" && !strings.HasPrefix(path, cp.prefix) {
		return false, nil
	}

	// Fast path: suffix check for patterns like "*.go"
	if cp.suffix != "" && strings.HasPrefix(cp.original, "*") {
		return strings.HasSuffix(path, cp.suffix), nil
	}

	// Use filepath.Match for complex patterns
	// This is slower but handles all glob syntax
	return filepath.Match(cp.original, path)
}

// MatchString is a convenience method that ignores errors (returns false on error).
func (cp *CompiledPattern) MatchString(path string) bool {
	matched, _ := cp.Match(path)
	return matched
}

// Original returns the original pattern string.
func (cp *CompiledPattern) Original() string {
	return cp.original
}

// Matcher holds multiple compiled patterns for efficient batch matching.
type Matcher struct {
	patterns []*CompiledPattern
}

// NewMatcher creates a new Matcher from a list of glob patterns.
func NewMatcher(patterns []string) *Matcher {
	compiled := make([]*CompiledPattern, len(patterns))
	for i, p := range patterns {
		compiled[i] = Compile(p)
	}
	return &Matcher{patterns: compiled}
}

// MatchAny checks if the path matches any of the patterns.
// Returns true on first match, or false if no pattern matches.
func (m *Matcher) MatchAny(path string) bool {
	for _, p := range m.patterns {
		if p.MatchString(path) {
			return true
		}
	}
	return false
}

// MatchAnyWithError checks if the path matches any pattern, returning the first error.
func (m *Matcher) MatchAnyWithError(path string) (bool, error) {
	for _, p := range m.patterns {
		matched, err := p.Match(path)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// MatchAll checks if the path matches all patterns.
func (m *Matcher) MatchAll(path string) bool {
	for _, p := range m.patterns {
		if !p.MatchString(path) {
			return false
		}
	}
	return true
}

// MatchFirst returns the first pattern that matches, or nil if none match.
func (m *Matcher) MatchFirst(path string) *CompiledPattern {
	for _, p := range m.patterns {
		if p.MatchString(path) {
			return p
		}
	}
	return nil
}

// Patterns returns the compiled patterns.
func (m *Matcher) Patterns() []*CompiledPattern {
	return m.patterns
}

// CaseInsensitiveMatcher creates a matcher that matches case-insensitively.
// It lowercases both patterns and paths for matching.
type CaseInsensitiveMatcher struct {
	patterns []string
	matcher  *Matcher
}

// NewCaseInsensitiveMatcher creates a new case-insensitive matcher.
func NewCaseInsensitiveMatcher(patterns []string) *CaseInsensitiveMatcher {
	lowered := make([]string, len(patterns))
	for i, p := range patterns {
		lowered[i] = strings.ToLower(p)
	}
	return &CaseInsensitiveMatcher{
		patterns: lowered,
		matcher:  NewMatcher(lowered),
	}
}

// Match checks if the lowercased path matches any pattern.
func (m *CaseInsensitiveMatcher) Match(path string) bool {
	return m.matcher.MatchAny(strings.ToLower(path))
}
