// Package metrics defines the MetricsReport contract and git ingestion pipeline
// for SDP process health analysis.
package metrics

import "time"

// MetricsReport is the stable output of sdp metrics.
// All fields use explicit nil/empty to signal "unknown".
// Consumers must check for nil before dereferencing pointer fields.
type MetricsReport struct {
	Version         string       `json:"version"`
	GeneratedAt     time.Time    `json:"generated_at"`
	RepoPath        string       `json:"repo_path"`
	DurationMs      int64        `json:"duration_ms"`
	CommitsAnalyzed int          `json:"commits_analyzed"`
	Period          TimePeriod   `json:"period"`
	// Category fields are populated by later workstreams (WS-02, WS-03).
	// This contract is versioned; consumers should check Version.
}

// TimePeriod describes the analysis window.
type TimePeriod struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// RawCommit is a parsed git commit with file-level change data.
// This is the shared raw data type consumed by all seven analyzers.
type RawCommit struct {
	Hash    string       `json:"hash"`
	Author  string       `json:"author"`
	Date    time.Time    `json:"date"`
	Subject string       `json:"subject"`
	Body    string       `json:"body"`
	Files   []FileChange `json:"files,omitempty"`
}

// FileChange represents a single file change within a commit (from git numstat).
type FileChange struct {
	Added   int    `json:"added"`
	Deleted int    `json:"deleted"`
	Path    string `json:"path"`
}

// TagInfo holds parsed semver tag metadata.
type TagInfo struct {
	Tag      string    `json:"tag"`
	Date     time.Time `json:"date"`
	IsSemver bool      `json:"is_semver"`
}

// BranchInfo holds parsed remote branch metadata.
type BranchInfo struct {
	Name       string     `json:"name"`
	LastCommit *time.Time `json:"last_commit,omitempty"`
}

// GitData is the complete raw dataset produced by the collector.
// All seven analyzers consume this single structure.
type GitData struct {
	Commits    []RawCommit `json:"commits"`
	Tags       []TagInfo   `json:"tags"`
	Branches   []BranchInfo `json:"branches"`
	MergeCount int         `json:"merge_count"`
}

// IsBot reports whether an author name matches known bot patterns.
func IsBot(author string) bool {
	bots := []string{"dependabot", "renovate", "github-actions", "mergify", "snyk", "semantic-release"}
	lower := toLower(author)
	for _, b := range bots {
		if contains(lower, b) {
			return true
		}
	}
	return false
}

// IsGeneratedFile reports whether a file path matches generated-file patterns.
func IsGeneratedFile(path string) bool {
	patterns := []string{".pb.go", ".generated.", ".min.js", ".min.css"}
	for _, p := range patterns {
		if contains(path, p) {
			return true
		}
	}
	// Lock files
	suffixes := []string{".lock", ".sum", "-lock.json"}
	for _, s := range suffixes {
		if hasSuffix(path, s) {
			return true
		}
	}
	return false
}

// IsCIOnly reports whether all changed files in a commit are CI/infra config.
func IsCIOnly(files []FileChange) bool {
	if len(files) == 0 {
		return false
	}
	ciPrefixes := []string{".github/", ".gitlab-ci.yml", "Jenkinsfile", ".circleci/", ".travis.yml"}
	for _, f := range files {
		isCI := false
		for _, prefix := range ciPrefixes {
			if hasPrefix(f.Path, prefix) {
				isCI = true
				break
			}
		}
		if !isCI {
			return false
		}
	}
	return true
}

// IsFormattingOnly reports whether a commit appears to be a mass reformatting.
// Heuristic: >90% of files have added ≈ deleted ± 10%.
func IsFormattingOnly(files []FileChange) bool {
	if len(files) < 3 {
		return false
	}
	formatCount := 0
	for _, f := range files {
		if f.Added == 0 && f.Deleted == 0 {
			continue
		}
		// added ≈ deleted within 10% tolerance
		min := imin(f.Added, f.Deleted)
		max := imax(f.Added, f.Deleted)
		if max > 0 && (min >= max*9/10) {
			formatCount++
		}
	}
	return formatCount*10 >= len(files)*9 // >=90%
}

func toLower(s string) string {
	// Simple ASCII lowercase without importing strings
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, sub string) bool {
	return len(sub) <= len(s) && stringContains(s, sub)
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func hasSuffix(s, suf string) bool {
	return len(suf) <= len(s) && s[len(s)-len(suf):] == suf
}

func hasPrefix(s, pre string) bool {
	return len(pre) <= len(s) && s[:len(pre)] == pre
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
