package architect

import "context"

// Extractor produces a ProfileFragment from a target repository.
// Each extractor handles one concern (file tree, imports, infra, etc.)
// and contributes its fragment to the assembled CodebaseProfile.
type Extractor interface {
	// Name returns a human-readable identifier for this extractor.
	Name() string

	// Extract analyzes the repository at repoRoot and returns a fragment.
	Extract(ctx context.Context, repoRoot string) (*ProfileFragment, error)
}
