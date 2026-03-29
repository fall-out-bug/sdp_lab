package dispatch

import "time"

// StalenessConfig controls when profiles are considered stale.
type StalenessConfig struct {
	MaxAge      time.Duration // profiles older than this are stale (default: 7 days)
	DecayFactor float64       // multiplier applied to stale profile scores (default: 0.5)
	ExpireAge   time.Duration // profiles older than this are expired/ignored (default: 30 days)
}

// DefaultStalenessConfig provides sensible defaults for profile staleness.
var DefaultStalenessConfig = StalenessConfig{
	MaxAge:      7 * 24 * time.Hour,
	DecayFactor: 0.5,
	ExpireAge:   30 * 24 * time.Hour,
}

// Freshness returns the freshness status of a profile.
type Freshness string

const (
	FreshnessFresh   Freshness = "fresh"
	FreshnessStale   Freshness = "stale"
	FreshnessExpired Freshness = "expired"
)

// CheckFreshness evaluates a profile's UpdatedAt against the config.
// Profiles with empty or unparseable UpdatedAt are treated as expired.
func CheckFreshness(profile *CapabilityProfile, cfg StalenessConfig, now time.Time) Freshness {
	if profile.UpdatedAt == "" {
		return FreshnessExpired
	}
	t, err := time.Parse(time.RFC3339, profile.UpdatedAt)
	if err != nil {
		return FreshnessExpired
	}
	age := now.Sub(t)
	if age >= cfg.ExpireAge {
		return FreshnessExpired
	}
	if age >= cfg.MaxAge {
		return FreshnessStale
	}
	return FreshnessFresh
}

// DecayScore applies a decay factor to a score based on freshness.
//   - fresh: score unchanged
//   - stale: score * DecayFactor
//   - expired: 0.0
func DecayScore(score float64, freshness Freshness, cfg StalenessConfig) float64 {
	switch freshness {
	case FreshnessStale:
		return score * cfg.DecayFactor
	case FreshnessExpired:
		return 0.0
	default:
		return score
	}
}
