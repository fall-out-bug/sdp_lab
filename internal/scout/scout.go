package scout

import (
	"path/filepath"
	"time"
)

// Run executes the full three-phase scout pipeline on a repository path.
// Returns a populated ProjectCard or an error.
func Run(repoPath string) (*ProjectCard, error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
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

	// Phase 3: Activity
	card.Activity = detectActivity(abs)
	detectMaturityFromGit(abs, &card.Maturity)

	// Phase 4: Health signals (derived from other fields)
	deriveHealthSignals(card)

	card.DurationMs = time.Since(start).Milliseconds()

	return card, nil
}
