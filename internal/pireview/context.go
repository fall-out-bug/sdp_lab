package pireview

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// BuildContextPacket collects the deterministic context packet for pi-review.
func BuildContextPacket(ctx context.Context, cfg Config) (*ContextPacket, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	branch, err := currentBranch(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("context: branch: %w", err)
	}

	headSHA, err := headSHA(ctx, cfg)
	if err != nil {
		headSHA = ""
	}

	status, err := gitStatus(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("context: status: %w", err)
	}

	reviewedFiles, err := resolveScope(ctx, cfg, status)
	if err != nil {
		return nil, fmt.Errorf("context: scope: %w", err)
	}

	diff, err := resolveDiff(ctx, cfg, status, reviewedFiles)
	if err != nil {
		return nil, fmt.Errorf("context: diff: %w", err)
	}

	fileHashes := make(map[string]string, len(reviewedFiles))
	for _, f := range reviewedFiles {
		h, err := fileSHA256(filepath.Join(cfg.ProjectRoot, f))
		if err != nil {
			continue
		}
		fileHashes[f] = h
	}

	contents, omitted, bytesUsed := collectFileContents(cfg.ProjectRoot, reviewedFiles, defaultSizeBudget)

	rules := loadProjectRules(cfg.ProjectRoot)

	beadCtx := ""
	if cfg.Feature != "" {
		bdID := featureToBeadsID(cfg.Feature)
		if bdID != "" {
			out, err := cfg.Runner.Output(ctx, cfg.ProjectRoot, "bd", "show", bdID)
			if err == nil {
				beadCtx = string(out)
			}
		}
	}

	return &ContextPacket{
		GitStatus:     status,
		Branch:        branch,
		BaseRef:       cfg.BaseRef,
		HeadSHA:       headSHA,
		ReviewedFiles: reviewedFiles,
		OmittedFiles:  omitted,
		UnifiedDiff:   diff,
		FileHashes:    fileHashes,
		FileContents:  contents,
		ProjectRules:  rules,
		BeadContext:   beadCtx,
		SizeBudget:    defaultSizeBudget,
		BytesUsed:     bytesUsed,
	}, nil
}

func currentBranch(ctx context.Context, cfg Config) (string, error) {
	out, err := cfg.Runner.Output(ctx, cfg.ProjectRoot, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func headSHA(ctx context.Context, cfg Config) (string, error) {
	out, err := cfg.Runner.Output(ctx, cfg.ProjectRoot, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitStatus(ctx context.Context, cfg Config) (string, error) {
	out, err := cfg.Runner.Output(ctx, cfg.ProjectRoot, "git", "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// resolveScope returns the list of files to review based on scope mode.
func resolveScope(ctx context.Context, cfg Config, status string) ([]string, error) {
	switch cfg.Scope {
	case ScopeAuto:
		if status != "" {
			return workingTreeFiles(status)
		}
		if cfg.BaseRef != "" {
			return branchDiffFiles(ctx, cfg)
		}
		return nil, fmt.Errorf("scope auto: tree is clean and no base ref provided")

	case ScopeWorkingTree:
		if status == "" {
			return nil, fmt.Errorf("scope working-tree: no uncommitted changes")
		}
		return workingTreeFiles(status)

	case ScopeBranch:
		return branchDiffFiles(ctx, cfg)

	default:
		return nil, fmt.Errorf("scope: unknown mode %q", cfg.Scope)
	}
}

// workingTreeFiles parses git status porcelain to get changed files.
func workingTreeFiles(status string) ([]string, error) {
	if status == "" {
		return nil, nil
	}

	seen := make(map[string]bool)
	var files []string

	for _, line := range strings.Split(status, "\n") {
		if len(line) < 4 {
			continue
		}
		// Extract filename from porcelain status (last 2+ chars are the path)
		// Format: XY filename or XY filename -> renamed
		path := line[3:]
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		path = strings.TrimSpace(path)
		if path == "" || seen[path] || shouldSkipFile(path) {
			continue
		}
		seen[path] = true
		files = append(files, path)
	}

	sort.Strings(files)
	return files, nil
}

// branchDiffFiles returns files changed between base and HEAD.
func branchDiffFiles(ctx context.Context, cfg Config) ([]string, error) {
	out, err := cfg.Runner.Output(ctx, cfg.ProjectRoot,
		"git", "diff", "--name-only", cfg.BaseRef+"...HEAD")
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only %s...HEAD: %w", cfg.BaseRef, err)
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !shouldSkipFile(line) {
			files = append(files, line)
		}
	}
	sort.Strings(files)
	return files, nil
}

// resolveDiff returns the unified diff for the scope.
func resolveDiff(ctx context.Context, cfg Config, status string, reviewedFiles []string) (string, error) {
	switch cfg.Scope {
	case ScopeAuto:
		if status != "" {
			return workingTreeDiff(ctx, cfg, reviewedFiles)
		}
		if cfg.BaseRef != "" {
			return branchDiff(ctx, cfg, reviewedFiles)
		}
		return "", nil

	case ScopeWorkingTree:
		return workingTreeDiff(ctx, cfg, reviewedFiles)

	case ScopeBranch:
		return branchDiff(ctx, cfg, reviewedFiles)

	default:
		return "", nil
	}
}

func workingTreeDiff(ctx context.Context, cfg Config, reviewedFiles []string) (string, error) {
	if len(reviewedFiles) == 0 {
		return "", nil
	}
	args := append([]string{"diff", "HEAD", "--"}, reviewedFiles...)
	out, err := cfg.Runner.CombinedOutput(ctx, cfg.ProjectRoot, "git", args...)
	if err != nil {
		// diff against empty tree when no commits exist
		args = append([]string{"diff", "--cached", "--"}, reviewedFiles...)
		out2, err2 := cfg.Runner.CombinedOutput(ctx, cfg.ProjectRoot, "git", args...)
		if err2 != nil {
			return "", fmt.Errorf("git diff HEAD: %w; git diff --cached: %w", err, err2)
		}
		return strings.TrimSpace(string(out2)), nil
	}
	return strings.TrimSpace(string(out)), nil
}

func branchDiff(ctx context.Context, cfg Config, reviewedFiles []string) (string, error) {
	if len(reviewedFiles) == 0 {
		return "", nil
	}
	args := append([]string{"diff", cfg.BaseRef + "...HEAD", "--"}, reviewedFiles...)
	out, err := cfg.Runner.CombinedOutput(ctx, cfg.ProjectRoot, "git", args...)
	if err != nil {
		return "", fmt.Errorf("git diff %s...HEAD: %w", cfg.BaseRef, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// featureToBeadsID maps a feature ID like "F161" to its beads epic ID.
// MVP stub: returns empty string. Full implementation reads .beads-sdp-mapping.jsonl.
func featureToBeadsID(feature string) string {
	// Read mapping from the workspace
	return ""
}
