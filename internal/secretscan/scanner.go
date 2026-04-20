// Package secretscan provides a standalone credential and secret scanner
// for pre-deploy validation. It uses regex-based detection with no external
// dependencies.
package secretscan

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Finding represents a detected secret or credential.
type Finding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Type     string `json:"type"`     // e.g., "aws_access_key", "github_token", "private_key"
	Severity string `json:"severity"` // "critical", "high", "medium"
	Match    string `json:"match"`    // First 20 chars of match (rest masked)
}

// ScanResult holds the complete scan result.
type ScanResult struct {
	FilesScanned int       `json:"files_scanned"`
	Findings     []Finding `json:"findings"`
	Duration     string    `json:"duration"`
	Status       string    `json:"status"` // "clean" or "findings"
}

// Scanner scans files for secrets and credentials.
type Scanner struct {
	ignorePatterns   []string // File patterns to ignore (set at construction)
	localIgnores     []string // Patterns loaded from .sdp/secretscan-ignore (per-scan)
	maxFileSize     int64    // Skip files larger than this (default: 1MB)
	rules           []rule   // Compiled detection rules (set once at construction)
}

// Option configures a Scanner.
type Option func(*Scanner)

// WithIgnorePatterns sets additional file patterns to ignore.
func WithIgnorePatterns(patterns []string) Option {
	return func(s *Scanner) {
		s.ignorePatterns = append(s.ignorePatterns, patterns...)
	}
}

// WithMaxFileSize sets the maximum file size to scan (in bytes).
func WithMaxFileSize(size int64) Option {
	return func(s *Scanner) {
		s.maxFileSize = size
	}
}

// rule is an internal detection rule.
type rule struct {
	Type     string
	Pattern  *regexp.Regexp
	Severity string
}

// NewScanner creates a new Scanner with the given options.
func NewScanner(opts ...Option) *Scanner {
	s := &Scanner{
		maxFileSize: 1 * 1024 * 1024, // 1MB
		rules:       compileRules(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// compileRules builds the detection rules once. Rules are ordered so that
// more specific patterns (anthropic_key) are checked before broader ones
// (openai_key) to avoid duplicate findings from overlapping regexes.
func compileRules() []rule {
	return []rule{
		{
			Type:     "aws_access_key",
			Pattern:  regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			Severity: "critical",
		},
		{
			Type:     "aws_secret_key",
			Pattern:  regexp.MustCompile(`(?i)aws_secret_access_key\s*[=:]\s*[A-Za-z0-9/+=]{40}`),
			Severity: "critical",
		},
		{
			// Unified: matches both gh[p]_ (PAT) and gh[s]_ (secret) tokens.
			Type:     "github_token",
			Pattern:  regexp.MustCompile(`gh[ps]_[A-Za-z0-9_]{36,}`),
			Severity: "critical",
		},
		{
			Type:     "api_key",
			Pattern:  regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[=:]\s*['"][A-Za-z0-9]{20,}['"]`),
			Severity: "high",
		},
		{
			Type:     "private_key",
			Pattern:  regexp.MustCompile(`-----BEGIN (RSA |EC |DSA )?PRIVATE KEY-----`),
			Severity: "critical",
		},
		{
			Type:     "database_uri",
			Pattern:  regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://[^:]+:[^@]+@`),
			Severity: "high",
		},
		{
			Type:     "stripe_key",
			Pattern:  regexp.MustCompile(`sk_live_[A-Za-z0-9]{24,}`),
			Severity: "critical",
		},
		{
			// Must come before openai_key: sk-ant- is a specific prefix.
			Type:     "anthropic_key",
			Pattern:  regexp.MustCompile(`sk-ant-[A-Za-z0-9]{20,}`),
			Severity: "high",
		},
		{
			// Matches generic sk- prefixed keys (non-ant, non-live).
			Type:     "openai_key",
			Pattern:  regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),
			Severity: "high",
		},
		{
			Type:     "password",
			Pattern:  regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*['"][^'"]{8,}['"]`),
			Severity: "medium",
		},
		{
			Type:     "bearer_token",
			Pattern:  regexp.MustCompile(`Bearer [A-Za-z0-9\-._~+/]+=*`),
			Severity: "high",
		},
		{
			Type:     "slack_token",
			Pattern:  regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
			Severity: "high",
		},
	}
}

// maskMatch returns the match string masked for safe display.
func maskMatch(s string) string {
	if len(s) <= 20 {
		return s
	}
	return s[:20] + "...<MASKED>"
}

// defaultIgnoreDirs lists directory names that are always skipped.
var defaultIgnoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".sdp":         true,
}

// isBinary checks whether data looks like a binary file by looking for
// null bytes in the first 512 bytes.
func isBinary(data []byte) bool {
	chunk := data
	if len(chunk) > 512 {
		chunk = chunk[:512]
	}
	return bytes.ContainsRune(chunk, 0)
}

// shouldIgnorePath checks whether a file path should be skipped.
func (s *Scanner) shouldIgnorePath(path string) bool {
	for part := range defaultIgnoreDirs {
		if strings.Contains(filepath.ToSlash(path), part+"/") {
			return true
		}
	}
	for _, pattern := range s.ignorePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}
	for _, pattern := range s.localIgnores {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}
	return false
}

// loadIgnoreFile reads an .sdp/secretscan-ignore file and appends its
// line-based glob patterns to the scanner's local ignore list (not shared state).
func (s *Scanner) loadIgnoreFile(root string) {
	ignorePath := filepath.Join(root, ".sdp", "secretscan-ignore")
	data, err := os.ReadFile(ignorePath)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		s.localIgnores = append(s.localIgnores, line)
	}
}

// ScanDir scans all files in a directory recursively.
func (s *Scanner) ScanDir(ctx context.Context, dir string) (*ScanResult, error) {
	start := time.Now()
	result := &ScanResult{}

	// Load project-local ignore file if present.
	s.loadIgnoreFile(dir)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Skip ignored directories.
		if d.IsDir() {
			if defaultIgnoreDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		if s.shouldIgnorePath(path) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if info.Size() > s.maxFileSize {
			return nil
		}

		findings, err := s.ScanFile(ctx, path)
		if err != nil {
			// Skip files we cannot read but do not abort the entire scan.
			return nil
		}

		result.FilesScanned++
		result.Findings = append(result.Findings, findings...)
		return nil
	})
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	result.Duration = time.Since(start).Round(time.Millisecond).String()
	if len(result.Findings) > 0 {
		result.Status = "findings"
	} else {
		result.Status = "clean"
	}
	return result, nil
}

// ScanFile scans a single file for secrets.
func (s *Scanner) ScanFile(ctx context.Context, path string) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Read a small chunk to detect binary files.
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read file header: %w", err)
	}
	if isBinary(buf[:n]) {
		return nil, nil
	}

	// Seek back to start.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}

	return s.ScanReader(ctx, f, path)
}

// ScanReader scans an io.Reader for secrets.
func (s *Scanner) ScanReader(ctx context.Context, r io.Reader, filename string) ([]Finding, error) {
	var findings []Finding
	detected := s.rules

	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if ctx.Err() != nil {
			return findings, ctx.Err()
		}

		line := scanner.Text()
		for _, rule := range detected {
			locs := rule.Pattern.FindAllStringIndex(line, -1)
			for _, loc := range locs {
				match := line[loc[0]:loc[1]]
				findings = append(findings, Finding{
					File:     filename,
					Line:     lineNum,
					Type:     rule.Type,
					Severity: rule.Severity,
					Match:    maskMatch(match),
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan reader: %w", err)
	}
	return findings, nil
}

// ToJSON serializes a ScanResult to JSON.
func (r *ScanResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// FromJSON deserializes a ScanResult from JSON.
func FromJSON(data []byte) (*ScanResult, error) {
	var r ScanResult
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal scan result: %w", err)
	}
	return &r, nil
}
