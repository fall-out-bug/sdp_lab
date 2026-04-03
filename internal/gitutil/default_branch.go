package gitutil

import (
	"context"
	"strings"

	"sdp_dev/internal/executil"
)

const fallbackDefaultBranch = "main"

// DefaultBranch returns the repository default branch derived from origin/HEAD.
// When origin/HEAD is unavailable, it falls back to known local/remote refs and
// finally to main.
func DefaultBranch(ctx context.Context, repoRoot string) string {
	return defaultBranchWithRunner(ctx, repoRoot, executil.DefaultRunner)
}

// ComparisonBase returns the best git ref to diff/log against for a branch.
// It prefers origin/<branch> when available, then the local branch, and falls
// back to origin/<branch> so callers keep the expected remote-tracking shape.
func ComparisonBase(ctx context.Context, repoRoot, branch string) string {
	return comparisonBaseWithRunner(ctx, repoRoot, branch, executil.DefaultRunner)
}

func defaultBranchWithRunner(ctx context.Context, repoRoot string, runner executil.CommandRunner) string {
	if ctx == nil {
		ctx = context.Background()
	}
	if branch := branchFromOriginHead(ctx, repoRoot, runner); branch != "" {
		return branch
	}
	for _, candidate := range []string{"main", "master"} {
		if refExists(ctx, repoRoot, runner, "refs/remotes/origin/"+candidate) ||
			refExists(ctx, repoRoot, runner, "refs/heads/"+candidate) {
			return candidate
		}
	}
	return fallbackDefaultBranch
}

func comparisonBaseWithRunner(ctx context.Context, repoRoot, branch string, runner executil.CommandRunner) string {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := normalizeBranchName(branch)
	if normalized == "" {
		normalized = defaultBranchWithRunner(ctx, repoRoot, runner)
	}
	if refExists(ctx, repoRoot, runner, "refs/remotes/origin/"+normalized) {
		return "origin/" + normalized
	}
	if refExists(ctx, repoRoot, runner, "refs/heads/"+normalized) {
		return normalized
	}
	return "origin/" + normalized
}

func branchFromOriginHead(ctx context.Context, repoRoot string, runner executil.CommandRunner) string {
	out, err := runner.Output(ctx, repoRoot, "git", "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if err != nil {
		return ""
	}
	return normalizeBranchName(string(out))
}

func refExists(ctx context.Context, repoRoot string, runner executil.CommandRunner, ref string) bool {
	return runner.Run(ctx, repoRoot, "git", "show-ref", "--verify", "--quiet", ref) == nil
}

func normalizeBranchName(branch string) string {
	trimmed := strings.TrimSpace(branch)
	trimmed = strings.TrimPrefix(trimmed, "refs/remotes/origin/")
	trimmed = strings.TrimPrefix(trimmed, "refs/heads/")
	trimmed = strings.TrimPrefix(trimmed, "origin/")
	return strings.TrimSpace(trimmed)
}
