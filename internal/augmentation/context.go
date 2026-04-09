package augmentation

import (
	"context"
	"fmt"

	"sdp_dev/internal/kernel"
)

func ResolvePromptContext(ctx context.Context, loader Loader, packRefs []string) ([]kernel.ContextSegment, error) {
	resolved, err := NewResolver(loader).Resolve(ctx, packRefs)
	if err != nil {
		return nil, fmt.Errorf("resolve prompt context: %w", err)
	}
	var segments []kernel.ContextSegment
	for _, packID := range resolved.Order {
		pack := resolved.Packs[packID]
		for _, fragment := range pack.PromptFragments {
			segments = append(segments, kernel.ContextSegment{
				ID:      fragment.ID,
				Kind:    fragment.Kind,
				Source:  pack.ID,
				Content: fragment.Content,
			})
		}
	}
	return segments, nil
}

func MustResolveDefaultPromptContext(packRefs ...string) []kernel.ContextSegment {
	segments, err := ResolvePromptContext(context.Background(), DefaultLoader(), packRefs)
	if err != nil {
		return nil
	}
	return segments
}
