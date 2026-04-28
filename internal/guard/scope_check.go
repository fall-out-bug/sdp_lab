package guard

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/executil"
	"github.com/fall-out-bug/sdp_lab/internal/sdputil"
)

// ScopeVerdict is the result of a scope check.
type ScopeVerdict struct {
	Pass       bool     // true if all changes are in scope or allowlisted
	Violations []string // files outside scope and not allowlisted
	Warnings   []string // files outside scope but allowlisted
}

// scopeFilesRe matches markdown list items with backtick paths: - `path/to/file`
var scopeFilesRe = regexp.MustCompile(`^\s*-\s*` + "`" + `([^` + "`" + `]+)` + "`")

// ParseScopeFiles reads the workstream markdown and extracts paths from ## Scope Files.
func ParseScopeFiles(wsPath string) ([]string, error) {
	data, err := os.ReadFile(wsPath)
	if err != nil {
		return nil, err
	}
	return ParseScopeFilesFromContent(string(data))
}

// ParseScopeFilesFromContent extracts scope paths from markdown content (for testing).
func ParseScopeFilesFromContent(content string) ([]string, error) {
	var paths []string
	inScope := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			if strings.Contains(line, "Scope Files") {
				inScope = true
				continue
			}
			if inScope {
				break // next section, stop
			}
		}
		if inScope {
			if m := scopeFilesRe.FindStringSubmatch(line); len(m) > 1 {
				p := strings.TrimSpace(m[1])
				if p != "" {
					paths = append(paths, p)
				}
			}
		}
	}
	return paths, scanner.Err()
}

// ChangedFiles returns files changed in the last commit (git diff --name-only HEAD~1 HEAD).
// If useCached is true, uses --cached for staged changes.
// Uses HEAD~1..HEAD to compare only the last commit, ignoring uncommitted changes.
func ChangedFiles(projectRoot string, useCached bool) ([]string, error) {
	args := []string{"diff", "--name-only"}
	if useCached {
		args = append(args, "--cached")
	} else {
		args = append(args, "HEAD~1", "HEAD")
	}
	out, err := executil.GetDefaultRunner().Output(context.Background(), projectRoot, "git", args...)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// inScope returns true if file matches any scope path (exact or prefix).
// Paths like "sdp/sdp-plugin/internal/quality/" match files under that directory.
func inScope(file string, scopePaths []string) bool {
	for _, p := range scopePaths {
		if file == p {
			return true
		}
		if !strings.HasPrefix(file, p) {
			continue
		}
		// Prefix match: "p" or "p/" matches "p/foo" but not "p_other"
		if len(file) == len(p) {
			return true
		}
		if len(p) > 0 && p[len(p)-1] == '/' {
			return true // p is dir path like "verify/"
		}
		if file[len(p)] == '/' {
			return true // p is "verify", file is "verify/foo"
		}
	}
	return false
}

// CheckScope compares changed files against workstream scope and allowlist.
func CheckScope(projectRoot, wsID string, useCached bool) (*ScopeVerdict, error) {
	if err := sdputil.ValidateWSID(wsID); err != nil {
		return nil, err
	}
	wsPath := filepath.Join(projectRoot, "docs", "workstreams", "backlog", wsID+".md")
	scopePaths, err := ParseScopeFiles(wsPath)
	if err != nil {
		return nil, fmt.Errorf("parse scope: %w", err)
	}

	changed, err := ChangedFiles(projectRoot, useCached)
	if err != nil {
		return nil, err
	}

	allowlist, err := LoadAllowlist(projectRoot)
	if err != nil {
		return nil, err
	}

	var violations, warnings []string
	for _, f := range changed {
		if inScope(f, scopePaths) {
			continue
		}
		if IsAllowlisted(f, allowlist) {
			warnings = append(warnings, f)
			continue
		}
		violations = append(violations, f)
	}

	return &ScopeVerdict{
		Pass:       len(violations) == 0,
		Violations: violations,
		Warnings:   warnings,
	}, nil
}
