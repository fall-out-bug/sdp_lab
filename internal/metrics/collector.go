package metrics

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultGitTimeout = 60 * time.Second

// GitError represents a structured error from a git command.
type GitError struct {
	Cmd      string
	ExitCode int
	Stderr   string
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %s: exit %d: %s", e.Cmd, e.ExitCode, e.Stderr)
}

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

	if err := validateRepoPath(repoPath); err != nil {
		return nil, err
	}

	// Call 1: git log --numstat (rich commit data)
	commits, err := collectCommits(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("collect commits: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metrics: %w", err)
	}

	// Call 2: git tag --sort=creatordate
	tags := collectTags(ctx, repoPath)

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metrics: %w", err)
	}

	// Call 3: git branch -r (single batch call via for-each-ref)
	branches := collectBranches(ctx, repoPath)

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metrics: %w", err)
	}

	// Call 4: git log --merges --first-parent main (merge count)
	mergeCount := countMerges(ctx, repoPath)

	return &GitData{
		Commits:    commits,
		Tags:       tags,
		Branches:   branches,
		MergeCount: mergeCount,
	}, nil
}

func validateRepoPath(repoPath string) error {
	info, err := os.Stat(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("metrics: repo path does not exist: %s", repoPath)
		}
		return fmt.Errorf("metrics: cannot access repo path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("metrics: repo path is not a directory: %s", repoPath)
	}
	gitDir := repoPath + "/.git"
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("metrics: not a git repository (no .git directory): %s", repoPath)
	}
	return nil
}

func collectCommits(ctx context.Context, dir string) ([]RawCommit, error) {
	raw, _ := gitCmdErr(ctx, dir, "log", "--numstat", "--no-merges",
		"--since=2 years ago",
		"--format="+gitLogFormat)
	if raw == "" {
		raw, _ = gitCmdErr(ctx, dir, "log", "--numstat", "--no-merges",
			"--format="+gitLogFormat)
	}
	if raw == "" {
		return nil, nil
	}
	return parseCommits(raw), nil
}

func collectTags(ctx context.Context, dir string) []TagInfo {
	raw, _ := gitCmdErr(ctx, dir, "tag", "--sort=creatordate")
	if raw == "" {
		return nil
	}
	return parseTags(raw)
}

func collectBranches(ctx context.Context, dir string) []BranchInfo {
	raw, _ := gitCmdErr(ctx, dir, "for-each-ref",
		"--sort=creatordate",
		"--format=%(refname:short) %(creatordate:iso-strict)",
		"refs/remotes/")
	if raw == "" {
		return nil
	}
	return parseBranchesBatch(raw)
}

func countMerges(ctx context.Context, dir string) int {
	branch := "main"
	if raw, _ := gitCmdErr(ctx, dir, "rev-parse", "--verify", "master"); raw != "" {
		branch = "master"
	}
	raw, _ := gitCmdErr(ctx, dir, "log", "--merges", "--first-parent", branch, "--format=%H")
	return countNonEmptyLines(raw)
}

// gitCmdErr runs a git command with timeout and returns stdout or a structured error.
func gitCmdErr(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		stderr := ""
		if out != nil {
			stderr = strings.TrimSpace(string(out))
		}
		return "", &GitError{
			Cmd:      strings.Join(args, " "),
			ExitCode: exitCode,
			Stderr:   stderr,
		}
	}
	return string(out), nil
}
