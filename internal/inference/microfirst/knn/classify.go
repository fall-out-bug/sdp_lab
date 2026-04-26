package knn

import "sdp_dev/internal/inference/decompose"

// ClassifyResult holds the outcome of MajorityVote.
type ClassifyResult[Label comparable] struct {
	Label     Label
	Score     float64        // top-1 cosine score
	Status    decompose.Status
	Neighbors []Match[Label] // top-k for explainability
}

// MajorityVote classifies by plurality among top-k matches.
// Returns StatusOK if:
//   - top-1 score >= minTop1Score
//   - top-3 (or all k if k<3) agree on the same label
//
// Otherwise returns StatusUnsure.
func MajorityVote[Label comparable](matches []Match[Label], minTop1Score float64) ClassifyResult[Label] {
	if len(matches) == 0 {
		var zero Label
		return ClassifyResult[Label]{
			Label:     zero,
			Score:     0,
			Status:    decompose.StatusUnsure,
			Neighbors: matches,
		}
	}

	top1 := matches[0]

	// Check score threshold.
	if top1.Score < minTop1Score {
		return ClassifyResult[Label]{
			Label:     top1.Label,
			Score:     top1.Score,
			Status:    decompose.StatusUnsure,
			Neighbors: matches,
		}
	}

	// Check agreement in top-3 (or fewer).
	window := 3
	if window > len(matches) {
		window = len(matches)
	}

	for i := 1; i < window; i++ {
		if matches[i].Label != top1.Label {
			return ClassifyResult[Label]{
				Label:     top1.Label,
				Score:     top1.Score,
				Status:    decompose.StatusUnsure,
				Neighbors: matches,
			}
		}
	}

	return ClassifyResult[Label]{
		Label:     top1.Label,
		Score:     top1.Score,
		Status:    decompose.StatusOK,
		Neighbors: matches,
	}
}
