package scout

import (
	"time"
)

// deriveHealthSignals populates HealthSignals from other card fields.
func deriveHealthSignals(card *ProjectCard) {
	h := &card.Health

	// Commit frequency: based on commits_30d / 4.3 (per-week rate)
	weeklyRate := float64(card.Activity.Commits30d) / 4.3
	switch {
	case card.Activity.Commits30d == 0 && card.Activity.TotalCommits == 0:
		h.CommitFrequency = Unknown
	case weeklyRate >= 20:
		h.CommitFrequency = CommitFreqHigh
	case weeklyRate >= 5:
		h.CommitFrequency = CommitFreqMedium
	default:
		h.CommitFrequency = CommitFreqLow
	}

	// Staleness: based on last commit date
	if card.Activity.LastCommit == nil {
		h.Staleness = Unknown
	} else {
		last, err := time.Parse("2006-01-02", *card.Activity.LastCommit)
		if err != nil {
			h.Staleness = Unknown
		} else {
			days := int(time.Since(last).Hours() / 24)
			switch {
			case days <= 7:
				h.Staleness = StalenessActive
			case days <= 30:
				h.Staleness = StalenessRecent
			case days <= 90:
				h.Staleness = StalenessStale
			default:
				h.Staleness = StalenessDormant
			}
		}
	}

	// Test coverage hint: based on test ratio
	switch {
	case card.Scale.SourceFiles == 0:
		h.TestCoverageHint = Unknown
	case card.Scale.TestRatio >= 0.3:
		h.TestCoverageHint = CovGood
	case card.Scale.TestRatio >= 0.1:
		h.TestCoverageHint = CovPartial
	case card.Scale.TestFiles > 0:
		h.TestCoverageHint = CovLow
	default:
		h.TestCoverageHint = CovNone
	}

	// Complexity hint: heuristic from max LOC, depth, total files
	complexityScore := 0
	if card.Scale.MaxFileLoc > 500 {
		complexityScore++
	}
	if card.Scale.DepthMax > 5 {
		complexityScore++
	}
	if card.Scale.TotalFiles > 200 {
		complexityScore++
	}
	switch {
	case complexityScore >= 3:
		h.ComplexityHint = ComplexityHigh
	case complexityScore >= 1:
		h.ComplexityHint = ComplexityMedium
	default:
		h.ComplexityHint = ComplexityLow
	}

	// Bus factor: sort authors by commit count, find minimum covering >50%
	h.BusFactorEstimate = calcBusFactor(card.Activity.Contributors, card.Activity.TotalCommits)
}

// calcBusFactor estimates minimum number of contributors whose departure
// would remove >50% of commits. Simplified: even distribution approximation.
func calcBusFactor(contributors, totalCommits int) int {
	if contributors == 0 || totalCommits == 0 {
		return 0
	}
	threshold := totalCommits / 2
	// In the simplest approximation, assume roughly even distribution:
	// bus factor ≈ ceil(contributors / 2)
	// For small teams, cap at contributor count.
	bf := contributors/2 + contributors%2
	if bf > contributors {
		bf = contributors
	}
	if bf < 1 {
		bf = 1
	}
	_ = threshold
	return bf
}
