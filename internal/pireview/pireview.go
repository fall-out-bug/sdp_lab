package pireview

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/executil"
)

// ScopeMode controls which files the review covers.
type ScopeMode string

const (
	ScopeAuto        ScopeMode = "auto"
	ScopeWorkingTree ScopeMode = "working-tree"
	ScopeBranch      ScopeMode = "branch"
)

// Config holds all configuration for a pi-review run.
type Config struct {
	ProjectRoot string
	Scope       ScopeMode
	BaseRef     string
	Feature     string
	TestCommand string
	Runner      executil.CommandRunner
}

// Validate checks required fields and returns an error if invalid.
func (c Config) Validate() error {
	if c.ProjectRoot == "" {
		return fmt.Errorf("pireview: ProjectRoot is required")
	}
	if c.Scope != ScopeAuto && c.Scope != ScopeWorkingTree && c.Scope != ScopeBranch {
		return fmt.Errorf("pireview: invalid Scope %q", c.Scope)
	}
	if c.Scope == ScopeBranch && c.BaseRef == "" {
		return fmt.Errorf("pireview: BaseRef is required with Scope=branch")
	}
	if c.Runner == nil {
		return fmt.Errorf("pireview: Runner is required")
	}
	return nil
}

// ContextPacket holds the deterministic context sent to model reviewers.
type ContextPacket struct {
	GitStatus     string            `json:"git_status"`
	Branch        string            `json:"branch"`
	BaseRef       string            `json:"base_ref,omitempty"`
	HeadSHA       string            `json:"head_sha,omitempty"`
	ReviewedFiles []string          `json:"reviewed_files"`
	OmittedFiles  []string          `json:"omitted_files,omitempty"`
	UnifiedDiff   string            `json:"unified_diff"`
	FileHashes    map[string]string `json:"file_hashes"`
	FileContents  map[string]string `json:"file_contents,omitempty"`
	ProjectRules  map[string]string `json:"project_rules,omitempty"`
	BeadContext   string            `json:"bead_context,omitempty"`
	SizeBudget    int               `json:"size_budget"`
	BytesUsed     int               `json:"bytes_used"`
}

// TestEvidence holds the result of running the project's test suite.
type TestEvidence struct {
	Status       string `json:"status"`
	Command      string `json:"command,omitempty"`
	ExitCode     int    `json:"exit_code,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	ArtifactPath string `json:"artifact_path"`
	SkipReason   string `json:"skip_reason,omitempty"`
	Output       string `json:"-"`
}

const (
	defaultSizeBudget = 512 * 1024 // 512 KiB
	maxFileSize       = 64 * 1024  // 64 KiB per file content
	ruleFiles         = "AGENTS.md,CLAUDE.md,.codex/AGENTS.md,.sdp/config.yml"
)

// fileSHA256 returns the hex-encoded SHA-256 of the file at path.
func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:]), nil
}

// loadProjectRules reads known rule files from the project root.
func loadProjectRules(root string) map[string]string {
	rules := make(map[string]string)
	for _, name := range strings.Split(ruleFiles, ",") {
		path := filepath.Join(root, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		rules[name] = string(data)
	}
	return rules
}

// collectFileContents reads file contents up to maxFileSize per file,
// stopping when total bytes approach budget. Omitted files are recorded.
func collectFileContents(root string, files []string, budget int) (map[string]string, []string, int) {
	contents := make(map[string]string)
	var omitted []string
	used := 0

	for _, f := range files {
		fullPath := filepath.Join(root, f)
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			omitted = append(omitted, f)
			continue
		}
		if info.Size() > int64(maxFileSize) || used+int(info.Size()) > budget {
			omitted = append(omitted, f)
			continue
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			omitted = append(omitted, f)
			continue
		}
		contents[f] = string(data)
		used += len(data)
	}
	return contents, omitted, used
}

// walkUntracked returns untracked files from git status porcelain output.
func parseUntrackedFromPorcelain(status string) []string {
	var files []string
	for _, line := range strings.Split(status, "\n") {
		if len(line) < 4 {
			continue
		}
		// '??' prefix = untracked, '?? ' followed by path
		if strings.HasPrefix(line, "?? ") {
			f := strings.TrimSpace(line[3:])
			if f != "" {
				files = append(files, f)
			}
		}
	}
	return files
}

// isBinaryFile does a quick check using file extension heuristics.
func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".ico": true, ".webp": true, ".zip": true, ".tar": true,
		".gz": true, ".exe": true, ".bin": true, ".woff": true,
		".woff2": true, ".ttf": true, ".eot": true, ".pdf": true,
		".svg": true, ".lock": true, ".sum": true,
	}
	base := filepath.Base(path)
	if strings.HasSuffix(base, "-lock.json") || strings.HasSuffix(base, ".lock") {
		return true
	}
	return binaryExts[ext]
}

// shouldSkipFile returns true for files that should not be reviewed.
func shouldSkipFile(path string) bool {
	base := filepath.Base(path)
	// Skip hidden files (except .go which is valid)
	if strings.HasPrefix(base, ".") && !strings.HasPrefix(base, ".go") {
		return true
	}
	// Skip common generated/binary directories and files
	skipPrefixes := []string{
		"vendor/", "node_modules/", ".git/", ".worktrees/",
		".sdp/runs/", ".beads/",
	}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return isBinaryFile(path)
}
