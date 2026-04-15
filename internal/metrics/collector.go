package metrics

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const defaultGitTimeout = 60 * time.Second

// Collect runs the 4-call git ingestion pipeline and returns raw data
// for all seven analyzers to consume.
func Collect(repoPath string) (*GitData, error) {
	return CollectWithContext(context.Background(), repoPath)
}

// CollectWithContext runs the pipeline with context for cancellation/timeout.
func CollectWithContext(ctx context.Context, repoPath string) (*GitData, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metrics: %w", err)
	}

	// Call 1: git log --numstat (rich commit data)
	commits, err := collectCommits(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("collect commits: %w", err)
	}

	// Call 2: git tag --sort=creatordate
	tags := collectTags(ctx, repoPath)

	// Call 3: git branch -r
	branches := collectBranches(ctx, repoPath)

	// Call 4: git log --merges --first-parent main (merge count)
	mergeCount := countMerges(ctx, repoPath)

	return &GitData{
		Commits:    commits,
		Tags:       tags,
		Branches:   branches,
		MergeCount: mergeCount,
	}, nil
}

// gitDelim is the format delimiter used to separate commits.
const gitDelim = "COMMIT_BOUNDARY_9F2A"

// gitLogFormat is the rich format capturing all fields needed by analyzers.
const gitLogFormat = "COMMIT_BOUNDARY_9F2A%H%nAUTHOR:%an%nDATE:%aI%nSUBJECT:%s%nBODY:%b%nNUMSTAT"

func collectCommits(ctx context.Context, dir string) ([]RawCommit, error) {
	// Try 2 years first, fall back to full history
	raw := gitCmd(ctx, dir, "log", "--numstat", "--no-merges",
		"--since=2 years ago",
		"--format="+gitLogFormat)
	if raw == "" {
		raw = gitCmd(ctx, dir, "log", "--numstat", "--no-merges",
			"--format="+gitLogFormat)
	}
	if raw == "" {
		return nil, nil
	}
	return parseCommits(raw), nil
}

func collectTags(ctx context.Context, dir string) []TagInfo {
	raw := gitCmd(ctx, dir, "tag", "--sort=creatordate")
	if raw == "" {
		return nil
	}
	return parseTags(raw)
}

func collectBranches(ctx context.Context, dir string) []BranchInfo {
	raw := gitCmd(ctx, dir, "branch", "-r")
	if raw == "" {
		return nil
	}
	return parseBranches(ctx, dir, raw)
}

func countMerges(ctx context.Context, dir string) int {
	// Determine default branch
	branch := "main"
	if gitCmd(ctx, dir, "rev-parse", "--verify", "master") != "" {
		branch = "master"
	}
	raw := gitCmd(ctx, dir, "log", "--merges", "--first-parent", branch, "--format=%H")
	return countNonEmptyLines(raw)
}

// parseCommits parses the raw git log output into RawCommit structs.
func parseCommits(raw string) []RawCommit {
	var commits []RawCommit
	blocks := strings.Split(raw, gitDelim)

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		c, ok := parseOneCommit(block)
		if !ok {
			continue
		}
		commits = append(commits, c)
	}
	return commits
}

func parseOneCommit(block string) (RawCommit, bool) {
	var c RawCommit
	scanner := bufio.NewScanner(strings.NewReader(block))
	inNumstat := false
	var numstatLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "NUMSTAT" {
			inNumstat = true
			continue
		}

		if inNumstat {
			if strings.HasPrefix(line, "COMMIT_BOUNDARY") || strings.HasPrefix(line, "AUTHOR:") {
				continue
			}
			numstatLines = append(numstatLines, line)
			continue
		}

		if strings.HasPrefix(line, "AUTHOR:") {
			c.Author = strings.TrimSpace(strings.TrimPrefix(line, "AUTHOR:"))
		} else if strings.HasPrefix(line, "DATE:") {
			ds := strings.TrimSpace(strings.TrimPrefix(line, "DATE:"))
			t, err := time.Parse(time.RFC3339, ds)
			if err == nil {
				c.Date = t
			}
		} else if strings.HasPrefix(line, "SUBJECT:") {
			c.Subject = strings.TrimSpace(strings.TrimPrefix(line, "SUBJECT:"))
		} else if strings.HasPrefix(line, "BODY:") {
			// Body may span multiple lines — collect remaining until NUMSTAT
			c.Body = strings.TrimSpace(strings.TrimPrefix(line, "BODY:"))
		} else if strings.HasPrefix(line, "COMMIT_BOUNDARY") {
			// Skip boundary markers
		} else if len(line) == 40 && isHex(line) && c.Hash == "" {
			c.Hash = line
		} else if c.Hash == "" && len(line) > 0 && len(line) <= 40 && isHex(line) {
			c.Hash = line
		}
	}

	// Parse numstat lines: "added\tdeleted\tpath"
	for _, nl := range numstatLines {
		fc, ok := parseNumstatLine(nl)
		if ok {
			c.Files = append(c.Files, fc)
		}
	}

	if c.Hash == "" {
		return c, false
	}
	return c, true
}

func parseNumstatLine(line string) (FileChange, bool) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) != 3 {
		return FileChange{}, false
	}
	added := parseNumOrNeg(parts[0])
	deleted := parseNumOrNeg(parts[1])
	path := parts[2]
	// Skip binary files (shown as "-")
	if added < 0 || deleted < 0 {
		return FileChange{Path: path}, false
	}
	return FileChange{Added: added, Deleted: deleted, Path: path}, true
}

func parseNumOrNeg(s string) int {
	s = strings.TrimSpace(s)
	if s == "-" {
		return -1
	}
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return -1
		}
	}
	return n
}

var semverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+`)

func parseTags(raw string) []TagInfo {
	var tags []TagInfo
	for _, line := range strings.Split(raw, "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		tags = append(tags, TagInfo{
			Tag:      tag,
			IsSemver: semverRe.MatchString(tag),
		})
	}
	return tags
}

func parseBranches(ctx context.Context, dir string, raw string) []BranchInfo {
	var branches []BranchInfo
	for _, line := range strings.Split(raw, "\n") {
		name := strings.TrimSpace(line)
		// Skip HEAD references and empty lines
		if name == "" || strings.Contains(name, "->") {
			continue
		}
		bi := BranchInfo{Name: name}
		// Get last commit date for the branch
		dateStr := gitCmd(ctx, dir, "log", "-1", "--format=%aI", name)
		if dateStr != "" {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(dateStr))
			if err == nil {
				bi.LastCommit = &t
			}
		}
		branches = append(branches, bi)
	}
	return branches
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func countNonEmptyLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

// gitCmd runs a git command with timeout and returns stdout.
func gitCmd(ctx context.Context, dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, defaultGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}
