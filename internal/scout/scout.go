package scout

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Run executes the full three-phase scout pipeline on a repository path.
// Returns a populated ProjectCard or an error.
func Run(repoPath string) (*ProjectCard, error) {
	return RunWithContext(context.Background(), repoPath)
}

// RunWithContext executes the pipeline with context for cancellation/timeout.
func RunWithContext(ctx context.Context, repoPath string) (*ProjectCard, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("scout: %w", err)
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("scout: cannot access %q: %w", repoPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scout: %q is not a directory", repoPath)
	}

	start := time.Now()
	card := &ProjectCard{
		Version:   "1.0.0",
		ScannedAt: start.UTC(),
		Identity: Identity{
			Languages: make(map[string]LangStats),
		},
	}

	// Phase 1: Identity
	identity, maturity, build := detectIdentity(abs)
	card.Identity = identity
	card.Maturity = maturity
	card.Build = build

	// Phase 2: Scale
	card.Scale = detectScale(abs, identity.BuildSystem)
	card.Maturity.HasTests = card.Scale.TestFiles > 0

	// Entry points
	card.Build.EntryPoints = detectBuildEntries(abs, identity.BuildSystem)

	// Phase 3: Activity (with context)
	card.Activity = detectActivityWithContext(ctx, abs)
	detectMaturityFromGitWithContext(ctx, abs, &card.Maturity)

	// B3: Populate RepoURL from git remote
	card.Identity.RepoURL = detectRepoURL(abs)

	// Phase 4: Health signals (derived from other fields)
	deriveHealthSignals(card)

	card.DurationMs = time.Since(start).Milliseconds()

	return card, nil
}
