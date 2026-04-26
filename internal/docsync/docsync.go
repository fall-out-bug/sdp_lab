package docsync

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"sdp_dev/internal/workstream"
)

type Issue struct {
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
}

type ConsistencyReport struct {
	Issues []Issue `json:"issues"`
}

type FixAction struct {
	File   string `json:"file"`
	Fix    string `json:"fix"` // "trailing-slash", "fence-tag", "relative-link"
	Before string `json:"before"`
	After  string `json:"after"`
}

type FixReport struct {
	Fixed      []FixAction `json:"fixed"`
	Unresolved []Issue     `json:"unresolved"`
}

func (r ConsistencyReport) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Severity == "error" {
			return true
		}
	}
	return false
}

func CheckConsistency(projectRoot string, strict bool) (ConsistencyReport, error) {
	report := ConsistencyReport{Issues: []Issue{}}

	protocol, err := workstream.ValidateProtocol(projectRoot, false, strict)
	if err != nil {
		return report, err
	}
	for _, p := range protocol.Issues {
		report.Issues = append(report.Issues, Issue{Severity: p.Severity, File: p.File, Message: p.Message})
	}

	linkIssues, err := checkMarkdownLinks(projectRoot, strict)
	if err != nil {
		return report, err
	}
	report.Issues = append(report.Issues, linkIssues...)

	slices.SortFunc(report.Issues, func(a, b Issue) int {
		if c := cmp.Compare(a.Severity, b.Severity); c != 0 {
			return c
		}
		if c := cmp.Compare(a.File, b.File); c != 0 {
			return c
		}
		return cmp.Compare(a.Message, b.Message)
	})

	return report, nil
}

func UpdateChangelog(projectRoot, sinceRange string) (string, error) {
	if sinceRange == "" {
		sinceRange = "HEAD~1..HEAD"
	}

	commitLines, err := gitOutput(projectRoot, "log", sinceRange, "--pretty=format:%h\t%s\t%ad", "--date=short")
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	if strings.TrimSpace(commitLines) == "" {
		return "", nil
	}

	changedFiles, err := gitOutput(projectRoot, "diff", "--name-only", sinceRange)
	if err != nil {
		return "", fmt.Errorf("git diff --name-only: %w", err)
	}

	changelogPath := filepath.Join(projectRoot, "docs", "CHANGELOG.md")
	if err := os.MkdirAll(filepath.Dir(changelogPath), 0o755); err != nil {
		return "", err
	}

	existing := "# Changelog\n\n"
	if b, err := os.ReadFile(changelogPath); err == nil {
		existing = string(b)
	}

	date := time.Now().Format("2006-01-02")
	entry := &strings.Builder{}
	fmt.Fprintf(entry, "## %s\n\n", date)
	fmt.Fprintln(entry, "### Commits")
	for _, ln := range splitNonEmpty(commitLines) {
		parts := strings.SplitN(ln, "\t", 3)
		if len(parts) == 3 {
			fmt.Fprintf(entry, "- `%s` %s (%s)\n", parts[0], parts[1], parts[2])
		} else {
			fmt.Fprintf(entry, "- %s\n", ln)
		}
	}
	fmt.Fprintln(entry)
	fmt.Fprintln(entry, "### Changed Files")
	for _, f := range splitNonEmpty(changedFiles) {
		fmt.Fprintf(entry, "- `%s`\n", f)
	}
	fmt.Fprintln(entry)

	var newContent string
	if strings.Contains(existing, "## "+date+"\n") {
		newContent = strings.Replace(existing, "## "+date+"\n", entry.String()+"\n", 1)
	} else {
		newContent = strings.TrimRight(existing, "\n") + "\n\n" + entry.String()
	}

	if err := os.WriteFile(changelogPath, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("write changelog: %w", err)
	}
	return changelogPath, nil
}

func checkMarkdownLinks(projectRoot string, strict bool) ([]Issue, error) {
	issues := []Issue{}
	docsRoot := filepath.Join(projectRoot, "docs")

	var files []string
	err := filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk docs dir: %w", err)
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Also scan top-level .md files (README, AGENTS, CLAUDE, CONTRIBUTING, VISION, RTK, etc).
	// These are operator-facing and drift frequently. Limit to the repo root (non-recursive)
	// to avoid scanning the sdp/ submodule, .claude/, .opencode/, .cursor/, archive/, etc.
	rootEntries, err := os.ReadDir(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("read project root: %w", err)
	}
	for _, e := range rootEntries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		files = append(files, filepath.Join(projectRoot, e.Name()))
	}

	re := regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	for _, path := range files {
		relPath := rel(projectRoot, path)
		if skipLinkCheck(relPath) {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, Issue{Severity: "warning", File: relPath, Message: fmt.Sprintf("read file: %v", err)})
			continue
		}
		matches := re.FindAllStringSubmatch(maskMarkdownCode(string(b)), -1)
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			target := strings.TrimSpace(m[1])
			if shouldSkipMarkdownLinkTarget(target) {
				continue
			}
			if i := strings.Index(target, "#"); i >= 0 {
				target = target[:i]
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), target))
			// Links into the sdp/ submodule cannot be validated without submodule init.
			// Skip them unless the submodule content is present.
			if resolvedInUninitSubmodule(projectRoot, resolved) {
				continue
			}
			if _, err := os.Stat(resolved); err != nil {
				sev := "warning"
				if strict {
					sev = "error"
				}
				issues = append(issues, Issue{Severity: sev, File: relPath, Message: fmt.Sprintf("broken local link: %s", target)})
			}
		}
	}

	return issues, nil
}

func gitOutput(projectRoot string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

func splitNonEmpty(s string) []string {
	result := []string{}
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func rel(projectRoot, path string) string {
	if p, err := filepath.Rel(projectRoot, path); err == nil {
		return p
	}
	return path
}

// maskMarkdownCode preserves byte offsets while hiding code fences and inline
// code spans from the naive markdown-link regexp. This keeps archived code
// examples such as fmt.Sprintf("[%s](%s)", ...) from becoming broken-link noise.
func maskMarkdownCode(content string) string {
	masked := []byte(content)
	inFence := false
	lineStart := 0
	for lineStart < len(content) {
		next := strings.IndexByte(content[lineStart:], '\n')
		lineEnd := len(content)
		if next >= 0 {
			lineEnd = lineStart + next
		}

		line := content[lineStart:lineEnd]
		trimmed := strings.TrimSpace(line)
		isFence := strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
		if isFence {
			maskRange(masked, lineStart, lineEnd)
			inFence = !inFence
		} else if inFence {
			maskRange(masked, lineStart, lineEnd)
		}

		if next < 0 {
			break
		}
		lineStart = lineEnd + 1
	}

	inlineCodeRe := regexp.MustCompile("`[^`\\n]*`")
	for _, loc := range inlineCodeRe.FindAllStringIndex(string(masked), -1) {
		maskRange(masked, loc[0], loc[1])
	}
	return string(masked)
}

func maskRange(b []byte, start, end int) {
	for i := start; i < end; i++ {
		if b[i] != '\n' && b[i] != '\r' {
			b[i] = ' '
		}
	}
}

func shouldSkipMarkdownLinkTarget(target string) bool {
	target = strings.TrimSpace(target)
	return target == "" || strings.HasPrefix(target, "#") || hasURIScheme(target)
}

func hasURIScheme(target string) bool {
	colon := strings.IndexByte(target, ':')
	if colon <= 0 {
		return false
	}
	first := target[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z')) {
		return false
	}
	for i := 1; i < colon; i++ {
		c := target[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.' {
			continue
		}
		return false
	}
	return true
}

// resolvedInUninitSubmodule reports whether resolved path lives inside the
// sdp/ submodule and the submodule is not initialized (so links inside it
// cannot be validated locally). When the submodule has been initialized
// (e.g., .git file/dir present), links are validated normally.
func resolvedInUninitSubmodule(projectRoot, resolved string) bool {
	submoduleRoot := filepath.Join(projectRoot, "sdp")
	rel, err := filepath.Rel(submoduleRoot, resolved)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if _, err := os.Stat(filepath.Join(submoduleRoot, ".git")); err == nil {
		return false
	}
	return true
}

// FixConsistency runs auto-fix routines on documentation and returns a report
// of what was fixed and what remains unresolved.
func FixConsistency(projectRoot string, strict bool) (FixReport, error) {
	report := FixReport{
		Fixed:      []FixAction{},
		Unresolved: []Issue{},
	}

	// Phase 1: trailing slash fixes
	slashFixes, slashIssues, err := FixTrailingSlashes(projectRoot)
	if err != nil {
		return report, fmt.Errorf("fix trailing slashes: %w", err)
	}
	report.Fixed = append(report.Fixed, slashFixes...)

	// Phase 2: code fence tag fixes
	fenceFixes, fenceIssues, err := FixCodeFenceTags(projectRoot)
	if err != nil {
		return report, fmt.Errorf("fix code fence tags: %w", err)
	}
	report.Fixed = append(report.Fixed, fenceFixes...)

	// Phase 3: relative link fixes (moved file detection via git)
	linkFixes, linkIssues, err := FixRelativeLinks(projectRoot)
	if err != nil {
		return report, fmt.Errorf("fix relative links: %w", err)
	}
	report.Fixed = append(report.Fixed, linkFixes...)

	// Collect remaining unresolved issues
	report.Unresolved = append(report.Unresolved, slashIssues...)
	report.Unresolved = append(report.Unresolved, fenceIssues...)
	report.Unresolved = append(report.Unresolved, linkIssues...)

	// Also run a full consistency check to capture protocol-level issues
	checkReport, err := CheckConsistency(projectRoot, strict)
	if err != nil {
		return report, fmt.Errorf("check consistency: %w", err)
	}
	for _, issue := range checkReport.Issues {
		if !issueAlreadyResolved(report.Fixed, issue) {
			report.Unresolved = append(report.Unresolved, issue)
		}
	}

	report.Unresolved = deduplicateIssues(report.Unresolved)

	return report, nil
}

// deduplicateIssues removes duplicate issues identified by (severity, file, message).
func deduplicateIssues(issues []Issue) []Issue {
	seen := make(map[string]bool)
	result := make([]Issue, 0, len(issues))
	for _, iss := range issues {
		key := iss.Severity + "\x00" + iss.File + "\x00" + iss.Message
		if !seen[key] {
			seen[key] = true
			result = append(result, iss)
		}
	}
	return result
}

func issueAlreadyResolved(fixed []FixAction, issue Issue) bool {
	for _, f := range fixed {
		if f.File == issue.File && strings.Contains(issue.Message, f.Before) {
			return true
		}
	}
	return false
}

// FixTrailingSlashes removes trailing slashes from local markdown links.
// Returns fixes applied and remaining issues.
func FixTrailingSlashes(projectRoot string) ([]FixAction, []Issue, error) {
	var fixes []FixAction
	var issues []Issue

	mdFiles, err := collectMarkdownFiles(projectRoot)
	if err != nil {
		return nil, nil, err
	}

	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	for _, path := range mdFiles {
		relPath := rel(projectRoot, path)
		if skipLinkCheck(relPath) {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, Issue{Severity: "warning", File: relPath, Message: fmt.Sprintf("read file: %v", err)})
			continue
		}
		content := string(b)
		matches := linkRe.FindAllStringSubmatchIndex(maskMarkdownCode(content), -1)

		type replace struct {
			start int
			end   int
			new   string
		}
		var replacements []replace

		for _, loc := range matches {
			// group 2 is the link target
			targetStart, targetEnd := loc[4], loc[5]
			target := content[targetStart:targetEnd]
			if shouldSkipMarkdownLinkTarget(target) {
				continue
			}

			// Strip anchor for slash check
			linkPath := target
			anchor := ""
			if i := strings.Index(target, "#"); i >= 0 {
				linkPath = target[:i]
				anchor = target[i:]
			}

			if strings.HasSuffix(linkPath, "/") && !strings.HasSuffix(linkPath, "//") {
				cleaned := strings.TrimRight(linkPath, "/") + anchor
				fixes = append(fixes, FixAction{
					File:   relPath,
					Fix:    "trailing-slash",
					Before: target,
					After:  cleaned,
				})
				replacements = append(replacements, replace{
					start: targetStart,
					end:   targetEnd,
					new:   cleaned,
				})
			}
		}

		if len(replacements) > 0 {
			// Apply replacements in reverse order to preserve indices
			result := content
			for i := len(replacements) - 1; i >= 0; i-- {
				r := replacements[i]
				result = result[:r.start] + r.new + result[r.end:]
			}
			if err := os.WriteFile(path, []byte(result), 0o644); err != nil {
				return nil, nil, fmt.Errorf("write %s: %w", relPath, err)
			}
		}
	}

	return fixes, issues, nil
}

// FixCodeFenceTags adds language tags to untagged code fences (```) based on
// content heuristics. Supports go, bash, and yaml inference.
func FixCodeFenceTags(projectRoot string) ([]FixAction, []Issue, error) {
	var fixes []FixAction
	var issues []Issue

	mdFiles, err := collectMarkdownFiles(projectRoot)
	if err != nil {
		return nil, nil, err
	}

	for _, path := range mdFiles {
		relPath := rel(projectRoot, path)
		if skipLinkCheck(relPath) {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(b)
		lines := strings.Split(content, "\n")
		changed := false

		// Track whether we're inside a code fence to avoid confusing
		// opening and closing fences (sdplab-xrz).
		inCodeBlock := false

		for i := 0; i < len(lines); i++ {
			line := lines[i]
			trimmedLine := strings.TrimRight(line, " \t")

			// Check if this is ANY fence line (tagged or untagged)
			isFence := strings.HasPrefix(trimmedLine, "```") || strings.HasPrefix(trimmedLine, "~~~")
			if !isFence {
				continue
			}

			// If we're not in a code block, this must be an opening fence
			if !inCodeBlock {
				// Only process untagged fences for fixing
				if !isUntaggedFence(line) {
					// Tagged fence - skip fixing but mark that we're in a code block
					inCodeBlock = true
					continue
				}

				// This is an untagged opening fence - find its closing fence
				inCodeBlock = true

				// Find the matching closing fence (skip all lines until we find another fence)
				closeIdx := -1
				for j := i + 1; j < len(lines); j++ {
					jline := strings.TrimRight(lines[j], " \t")
					// A closing fence is just ``` or ~~~ with optional trailing whitespace
					if jline == "```" || jline == "~~~" {
						closeIdx = j
						break
					}
				}
				if closeIdx == -1 {
					// No closing fence found - skip this block
					inCodeBlock = false
					continue
				}

				codeLines := lines[i+1 : closeIdx]
				if len(codeLines) == 0 {
					// Empty code block - skip it entirely
					inCodeBlock = false
					i = closeIdx // Jump to the closing fence
					continue
				}
				lang := inferCodeLanguage(codeLines)
				if lang == "" {
					// Can't infer language - skip this block without tagging
					inCodeBlock = false
					i = closeIdx // Jump to the closing fence
					continue
				}

				before := lines[i]
				after := "```" + lang
				fixes = append(fixes, FixAction{
					File:   relPath,
					Fix:    "fence-tag",
					Before: before,
					After:  after,
				})
				lines[i] = after
				changed = true
			} else {
				// This is a closing fence
				inCodeBlock = false
			}
		}

		if changed {
			newContent := strings.Join(lines, "\n")
			if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
				return nil, nil, fmt.Errorf("write %s: %w", relPath, err)
			}
		}
	}

	return fixes, issues, nil
}

// isUntaggedFence reports whether line is an opening code fence with no language tag.
func isUntaggedFence(line string) bool {
	trimmed := strings.TrimRight(line, " \t")
	return trimmed == "```"
}

// inferCodeLanguage uses content heuristics to detect the programming language
// of a code block. Returns "go", "bash", or "yaml" when confident, "" otherwise.
func inferCodeLanguage(codeLines []string) string {
	// YAML heuristics: key: value patterns, --- header.
	yamlScore := 0
	for _, line := range codeLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, ": ") || strings.HasSuffix(trimmed, ":") {
			yamlScore++
		}
		if strings.HasPrefix(trimmed, "- ") {
			yamlScore++
		}
	}
	if yamlScore >= 2 {
		return "yaml"
	}

	// Go heuristics: func, package, import, :=, fmt.
	goScore := 0
	for _, line := range codeLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "package ") {
			goScore += 2
		}
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import(") {
			goScore += 2
		}
		if strings.Contains(trimmed, ":=") || strings.Contains(trimmed, "fmt.") {
			goScore++
		}
	}
	if goScore >= 2 {
		return "go"
	}

	// Bash heuristics: shebang, common commands.
	for _, line := range codeLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#!/bin/bash") || strings.HasPrefix(trimmed, "#!/usr/bin/env bash") {
			return "bash"
		}
		if strings.HasPrefix(trimmed, "$ ") || strings.HasPrefix(trimmed, "sudo ") || strings.HasPrefix(trimmed, "chmod ") {
			return "bash"
		}
		if strings.HasPrefix(trimmed, "export ") && strings.Contains(trimmed, "=") {
			return "bash"
		}
		if strings.HasPrefix(trimmed, "go ") || strings.HasPrefix(trimmed, "git ") || strings.HasPrefix(trimmed, "docker ") || strings.HasPrefix(trimmed, "make ") {
			return "bash"
		}
		if strings.HasPrefix(trimmed, "cd ") || strings.HasPrefix(trimmed, "rm ") || strings.HasPrefix(trimmed, "cp ") || strings.HasPrefix(trimmed, "mv ") {
			return "bash"
		}
	}

	return ""
}

// FixRelativeLinks attempts to fix relative links that point to moved files
// by consulting git history. Returns fixes applied and remaining issues.
func FixRelativeLinks(projectRoot string) ([]FixAction, []Issue, error) {
	var fixes []FixAction
	var issues []Issue

	mdFiles, err := collectMarkdownFiles(projectRoot)
	if err != nil {
		return nil, nil, err
	}

	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	for _, path := range mdFiles {
		relPath := rel(projectRoot, path)
		if skipLinkCheck(relPath) {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(b)
		matches := linkRe.FindAllStringSubmatchIndex(maskMarkdownCode(content), -1)

		type replace struct {
			start int
			end   int
			new   string
		}
		var replacements []replace

		for _, loc := range matches {
			targetStart, targetEnd := loc[4], loc[5]
			target := content[targetStart:targetEnd]
			if shouldSkipMarkdownLinkTarget(target) {
				continue
			}

			// Strip anchor for resolution
			linkPath := target
			anchor := ""
			if i := strings.Index(target, "#"); i >= 0 {
				linkPath = target[:i]
				anchor = target[i:]
			}

			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), linkPath))
			// Links into the sdp/ submodule cannot be validated without submodule init.
			// Skip them unless the submodule content is present.
			if resolvedInUninitSubmodule(projectRoot, resolved) {
				continue
			}
			if _, err := os.Stat(resolved); err == nil {
				continue // link is valid
			}

			// Try to find the file via git log
			resolvedRel := rel(projectRoot, resolved)
			newPath, err := findRenamedFile(projectRoot, resolvedRel)
			if err != nil || newPath == "" {
				issues = append(issues, Issue{
					Severity: "warning",
					File:     relPath,
					Message:  fmt.Sprintf("broken local link: %s", target),
				})
				continue
			}

			// Compute new relative path from the source file to the renamed target
			newAbs := filepath.Join(projectRoot, newPath)
			newRel, err := filepath.Rel(filepath.Dir(path), newAbs)
			if err != nil {
				continue
			}
			newTarget := newRel + anchor
			fixes = append(fixes, FixAction{
				File:   relPath,
				Fix:    "relative-link",
				Before: target,
				After:  newTarget,
			})
			replacements = append(replacements, replace{
				start: targetStart,
				end:   targetEnd,
				new:   newTarget,
			})
		}

		if len(replacements) > 0 {
			result := content
			for i := len(replacements) - 1; i >= 0; i-- {
				r := replacements[i]
				result = result[:r.start] + r.new + result[r.end:]
			}
			if err := os.WriteFile(path, []byte(result), 0o644); err != nil {
				return nil, nil, fmt.Errorf("write %s: %w", relPath, err)
			}
		}
	}

	return fixes, issues, nil
}

// findRenamedFile uses git log to find where a file was moved/renamed to.
// relPath is the old (broken) path; it returns the new (current) path, or "" if not found.
func findRenamedFile(projectRoot, relPath string) (string, error) {
	// Get all .md files currently tracked by git.
	out, err := gitOutput(projectRoot, "ls-files", "--", "*.md")
	if err != nil {
		return "", nil
	}
	currentFiles := splitNonEmpty(out)

	// For each current markdown file, check if it was renamed from relPath using --follow.
	for _, current := range currentFiles {
		logOut, err := gitOutput(projectRoot, "log", "--follow", "--diff-filter=R", "--name-status", "--max-count=100", "--format=%H", "--", current)
		if err != nil || strings.TrimSpace(logOut) == "" {
			continue
		}
		lines := splitNonEmpty(logOut)
		for _, line := range lines {
			// Skip commit hash lines
			if len(line) == 40 && isHexString(line) {
				continue
			}
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) >= 3 {
				oldPath := parts[1]
				newPath := parts[2]
				if oldPath == relPath && newPath == current {
					return newPath, nil
				}
			}
		}
	}
	return "", nil
}

// isHexString reports whether s consists entirely of hex digits.
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// collectMarkdownFiles returns all .md files under docs/ and at the project root level.
func collectMarkdownFiles(projectRoot string) ([]string, error) {
	var files []string
	docsRoot := filepath.Join(projectRoot, "docs")

	if _, err := os.Stat(docsRoot); err == nil {
		err := filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("walk docs dir: %w", err)
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	rootEntries, err := os.ReadDir(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("read project root: %w", err)
	}
	for _, e := range rootEntries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		files = append(files, filepath.Join(projectRoot, e.Name()))
	}

	return files, nil
}

func skipLinkCheck(relPath string) bool {
	legacyPrefixes := []string{
		"docs/reference/",
		"docs/decisions/",
		"docs/design/",
		"docs/attestation/",
		"docs/beads-integration/",
		"docs/integrations/",
		"docs/specs/",
		"docs/vision/",
	}
	for _, p := range legacyPrefixes {
		if strings.HasPrefix(relPath, p) {
			return true
		}
	}
	if relPath == "docs/INCIDENT_RESPONSE.md" {
		return true
	}
	if strings.HasPrefix(relPath, "docs/workstreams/backlog/") {
		base := filepath.Base(relPath)
		var prefix, feature, seq int
		if _, err := fmt.Sscanf(strings.TrimSuffix(base, filepath.Ext(base)), "%d-%d-%d", &prefix, &feature, &seq); err == nil {
			if feature < 59 {
				return true
			}
		}
	}
	if relPath == "docs/plans/2026-02-25-beads-remediation-plan.md" {
		return true
	}
	return false
}
