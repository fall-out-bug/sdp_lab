package dispatch

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

// VerificationRouter selects a harness for verification (review/qa) that is
// different from the one used for build. This ensures independent
// cross-harness verification — the agent checking the work is not the same
// agent (or at least not the same harness) that produced it.
type VerificationRouter struct {
	Profiles []*CapabilityProfile
}

// RouteVerification selects the best profile that is NOT the given buildHarness.
// Returns nil (no error) if no alternative exists — the caller should fall back
// to the build harness.
func (vr *VerificationRouter) RouteVerification(
	ctx context.Context,
	task TaskClassification,
	buildHarness string,
	limits map[string]*harness.Limits,
) (*DispatchDecision, error) {
	if len(vr.Profiles) == 0 {
		return nil, nil
	}

	now := time.Now()
	type scored struct {
		profile *CapabilityProfile
		score   float64
	}

	var candidates []scored
	for _, p := range vr.Profiles {
		if p.Harness == buildHarness {
			continue
		}
		capScore := p.ScoreFor(task.TaskType, task.Language)
		availFactor := 1.0
		if lim, ok := limits[p.Provider]; ok {
			availFactor = AvailabilityFactor(lim)
		}
		final := capScore * availFactor
		candidates = append(candidates, scored{profile: p, score: final})
	}

	if len(candidates) == 0 {
		slog.Info("dispatch: no alternative harness for verification, will use build harness",
			"build_harness", buildHarness)
		return nil, nil
	}

	const epsilon = 1e-9
	sort.Slice(candidates, func(i, j int) bool {
		if math.Abs(candidates[i].score-candidates[j].score) > epsilon {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].profile.Harness < candidates[j].profile.Harness
	})

	winner := candidates[0]
	if winner.score <= 0 {
		// All alternatives have zero scores — apply cold-start heuristic.
		for i := range candidates {
			candidates[i].score = averageTestPassRate(candidates[i].profile)
		}
		sort.Slice(candidates, func(i, j int) bool {
			if math.Abs(candidates[i].score-candidates[j].score) > epsilon {
				return candidates[i].score > candidates[j].score
			}
			return candidates[i].profile.Harness < candidates[j].profile.Harness
		})
		winner = candidates[0]
		if winner.score <= 0 {
			return nil, nil
		}
	}

	alts := make([]Alternative, 0, len(candidates)-1)
	for _, s := range candidates[1:] {
		alts = append(alts, Alternative{
			Harness: s.profile.Harness,
			Score:   s.score,
		})
	}

	dec := &DispatchDecision{
		Harness:      winner.profile.Harness,
		Provider:     winner.profile.Provider,
		Model:        winner.profile.Model,
		Score:        winner.score,
		Reason:       fmt.Sprintf("cross-harness verification (not %s) for %s:%s", buildHarness, task.TaskType, task.Language),
		Timestamp:    now.UTC().Format(time.RFC3339),
		Alternatives: alts,
	}

	slog.Info("dispatch: verification harness selected",
		"harness", dec.Harness,
		"score", dec.Score,
		"build_harness", buildHarness,
	)

	return dec, nil
}
