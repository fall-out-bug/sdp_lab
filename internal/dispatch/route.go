package dispatch

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync/atomic"
	"time"

	"sdp_dev/internal/dispatch/harness"
)

// ColdStartStrategy determines how the router handles tasks with no bench data.
type ColdStartStrategy string

const (
	// ColdStartCapabilityHeuristic assigns a base score from the profile's overall
	// average TestPassRate across all known capabilities. Profiles with zero
	// capabilities receive 0.5 as a neutral prior.
	ColdStartCapabilityHeuristic ColdStartStrategy = "capability-heuristic"

	// ColdStartRoundRobin cycles through profiles in sequence for each cold-start
	// routing call. Useful for exploration and benchmarking new task types.
	ColdStartRoundRobin ColdStartStrategy = "round-robin"

	// ColdStartFallbackChain uses profile order as explicit priority — the first
	// profile in the list always wins.
	ColdStartFallbackChain ColdStartStrategy = "fallback-chain"
)

// Router selects the best harness+model for a given task using capability profiles
// and live availability limits.
type Router struct {
	Profiles          []*CapabilityProfile
	ColdStartStrategy ColdStartStrategy
	StalenessConfig   *StalenessConfig
	// LocalConfig, when set, enables a local Ollama tier for low-complexity coding tasks.
	LocalConfig *LocalConfig

	// MicroRouter, when set, attempts embedding-based capability hint on cold-start
	// before falling back to the configured ColdStartStrategy.
	// nil = disabled (backward-compat).
	MicroRouter MicroRouter

	// rrCounter tracks round-robin position across calls.
	// Use atomic.Int32 for concurrent access.
	rrCounter atomic.Int32
}

// scoredProfile pairs a profile with its final computed score.
type scoredProfile struct {
	profile    *CapabilityProfile
	finalScore float64
}

// Route selects the best harness+model for the given task and limits.
// Returns an error if no profiles are configured or all scores are <= 0.
func (r *Router) Route(ctx context.Context, task TaskClassification, limits map[string]*harness.Limits) (*DispatchDecision, error) {
	if len(r.Profiles) == 0 {
		return nil, fmt.Errorf("dispatch: no capability profiles configured")
	}

	now := time.Now()

	scored := make([]scoredProfile, 0, len(r.Profiles))
	for _, p := range r.Profiles {
		capScore := p.ScoreFor(task.TaskType, task.Language)
		if r.StalenessConfig != nil {
			freshness := CheckFreshness(p, *r.StalenessConfig, now)
			capScore = DecayScore(capScore, freshness, *r.StalenessConfig)
		}
		availFactor := 1.0
		if lim, ok := limits[p.Provider]; ok {
			availFactor = AvailabilityFactor(lim)
		}
		final := capScore * availFactor
		slog.Debug("router scoring profile",
			"harness", p.Harness,
			"provider", p.Provider,
			"capScore", capScore,
			"availFactor", availFactor,
			"finalScore", final,
		)
		scored = append(scored, scoredProfile{profile: p, finalScore: final})
	}

	// Inject local model with fixed score when task is low-complexity coding.
	if r.LocalConfig != nil && task.Complexity == "low" && task.RequiredCap == "coding" {
		scored = append(scored, scoredProfile{
			profile:    localProfile(r.LocalConfig),
			finalScore: r.LocalConfig.score(),
		})
		slog.Debug("router: local model injected",
			"model", r.LocalConfig.Model, "score", r.LocalConfig.score())
	}

	// Check if all capability scores are 0.0 (cold start condition).
	allZero := true
	for _, s := range scored {
		if s.finalScore > 0 {
			allZero = false
			break
		}
	}

	coldStart := false
	if allZero {
		coldStart = true
		scored = r.applyColdStart(ctx, task, scored)
	}

	// Sort descending by finalScore; stable by harness name for determinism.
	const scoreEpsilon = 1e-9
	sort.Slice(scored, func(i, j int) bool {
		if math.Abs(scored[i].finalScore-scored[j].finalScore) > scoreEpsilon {
			return scored[i].finalScore > scored[j].finalScore
		}
		return scored[i].profile.Harness < scored[j].profile.Harness
	})

	winner := scored[0]
	if winner.finalScore <= 0 {
		return nil, fmt.Errorf("dispatch: no viable profile (all scores <= 0) for task %s:%s",
			task.TaskType, task.Language)
	}

	alts := make([]Alternative, 0, len(scored)-1)
	for _, s := range scored[1:] {
		alts = append(alts, Alternative{
			Harness: s.profile.Harness,
			Score:   s.finalScore,
		})
	}

	strategy := r.effectiveColdStartStrategy()
	reason := fmt.Sprintf("highest effective score for %s:%s", task.TaskType, task.Language)
	if coldStart {
		reason = fmt.Sprintf("cold-start: %s for %s:%s", strategy, task.TaskType, task.Language)
	}

	dec := &DispatchDecision{
		Harness:      winner.profile.Harness,
		Provider:     winner.profile.Provider,
		Model:        winner.profile.Model,
		Score:        winner.finalScore,
		Reason:       reason,
		Timestamp:    now.UTC().Format(time.RFC3339),
		Alternatives: alts,
		ColdStart:    coldStart,
	}

	if r.StalenessConfig != nil {
		dec.Staleness = string(CheckFreshness(winner.profile, *r.StalenessConfig, now))
	}

	slog.Info("router selected harness",
		"harness", dec.Harness,
		"score", dec.Score,
		"task_type", task.TaskType,
		"language", task.Language,
	)

	return dec, nil
}

// effectiveColdStartStrategy returns the configured strategy, defaulting to capability-heuristic.
func (r *Router) effectiveColdStartStrategy() ColdStartStrategy {
	if r.ColdStartStrategy == "" {
		return ColdStartCapabilityHeuristic
	}
	return r.ColdStartStrategy
}

// applyColdStart assigns scores to profiles when no bench data exists for the task.
func (r *Router) applyColdStart(ctx context.Context, task TaskClassification, scored []scoredProfile) []scoredProfile {
	// When MicroRouter is set, attempt an embedding-based capability hint first.
	// Only short-circuit if at least one profile matched the hint; otherwise fall
	// through to the configured cold-start strategy to avoid all-zero scores.
	if r.MicroRouter != nil {
		if hint, ok := r.MicroRouter.SuggestCapability(ctx, task.Title, task.Description); ok {
			matched := false
			for i := range scored {
				if scored[i].profile.HasCapability(hint) {
					scored[i].finalScore = 0.9
					matched = true
				}
			}
			if matched {
				return scored
			}
		}
	}

	strategy := r.effectiveColdStartStrategy()

	slog.Info("applying cold-start strategy", "strategy", strategy, "profiles", len(scored))

	switch strategy {
	case ColdStartRoundRobin:
		// Pick the next profile in sequence; all others get 0.
		// Use atomic operations for thread safety.
		idx := int(r.rrCounter.Load()) % len(scored)
		r.rrCounter.Add(1)
		for i := range scored {
			if i == idx {
				scored[i].finalScore = 1.0
			} else {
				scored[i].finalScore = 0.0
			}
		}

	case ColdStartFallbackChain:
		// Profile order is priority: first gets highest score, descending.
		for i := range scored {
			scored[i].finalScore = float64(len(scored)-i) / float64(len(scored))
		}

	default: // ColdStartCapabilityHeuristic
		for i := range scored {
			scored[i].finalScore = averageTestPassRate(scored[i].profile)
		}
	}

	return scored
}

// averageTestPassRate computes the mean TestPassRate across all capabilities.
// Returns 0.5 as a neutral prior if the profile has no capabilities.
func averageTestPassRate(p *CapabilityProfile) float64 {
	if len(p.Capabilities) == 0 {
		return 0.5
	}
	var sum float64
	for _, cs := range p.Capabilities {
		sum += cs.TestPassRate
	}
	return sum / float64(len(p.Capabilities))
}
