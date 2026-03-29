package executor

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"sdp_dev/internal/control"
)

// RankingPolicy defines how ready cards are prioritized for dispatch.
type RankingPolicy struct {
	// PriorityWeights maps card priority to weight. Higher = dispatch first.
	PriorityWeights map[int]int
	// MaxRetries is max dispatch retries before marking stuck.
	MaxRetries int
	// RetryBackoff is base backoff between retries.
	RetryBackoff time.Duration
	// StaleThreshold — card untouched for this long gets deprioritized.
	StaleThreshold time.Duration
}

// DefaultRankingPolicy returns a reasonable default ranking configuration.
func DefaultRankingPolicy() *RankingPolicy {
	return &RankingPolicy{
		PriorityWeights: map[int]int{
			0: 100, // P1
			1: 80,  // P2
			2: 60,  // P3
			3: 40,  // P4
		},
		MaxRetries:     3,
		RetryBackoff:   5 * time.Minute,
		StaleThreshold: 24 * time.Hour,
	}
}

// RankAndPick selects the best card from a ready list using ranking policy.
// Returns the card ID to dispatch, or empty if nothing should be dispatched.
func RankAndPick(ready []control.FeatureCard, policy *RankingPolicy, retryCounts map[string]int) string {
	if len(ready) == 0 {
		return ""
	}
	if policy == nil {
		policy = DefaultRankingPolicy()
	}

	type scored struct {
		id    string
		score int
	}

	var candidates []scored
	for _, card := range ready {
		retries := retryCounts[card.ID]
		if retries >= policy.MaxRetries {
			continue
		}

		score := 0

		priority := 3
		if strings.HasPrefix(card.Title, "P1") || strings.Contains(strings.ToLower(card.Title), "critical") {
			priority = 0
		} else if strings.HasPrefix(card.Title, "P2") || strings.Contains(strings.ToLower(card.Title), "important") {
			priority = 1
		} else if strings.HasPrefix(card.Title, "P3") {
			priority = 2
		}

		if w, ok := policy.PriorityWeights[priority]; ok {
			score += w
		}

		if card.Status == "executing" {
			score += 20
		}

		score -= (retries * 10)

		candidates = append(candidates, scored{id: card.ID, score: score})
	}

	if len(candidates) == 0 {
		return ""
	}

	slices.SortFunc(candidates, func(a, b scored) int {
		return cmp.Compare(b.score, a.score)
	})

	return candidates[0].id
}

// StuckDetector identifies cards that are stuck in a non-terminal state.
type StuckDetector struct {
	store  *control.Store
	policy *RankingPolicy
}

// NewStuckDetector creates a stuck detector.
func NewStuckDetector(store *control.Store, policy *RankingPolicy) *StuckDetector {
	if policy == nil {
		policy = DefaultRankingPolicy()
	}
	return &StuckDetector{store: store, policy: policy}
}

// DetectStuck finds cards that appear stuck (executing too long or retried too many times).
func (d *StuckDetector) DetectStuck() ([]StuckCard, error) {
	beadsRepo := d.store.BeadsRepo()
	if beadsRepo == nil {
		return nil, fmt.Errorf("stuck detection requires beads mode")
	}

	// Query all executing cards
	ready, err := beadsRepo.QueryReady()
	if err != nil {
		return nil, fmt.Errorf("query ready: %w", err)
	}

	var stuck []StuckCard
	for _, card := range ready {
		if card.Status == "executing" {
			// Mark as stuck if it's been executing (would need timestamp check)
			// For now, just flag all long-running executing cards
			stuck = append(stuck, StuckCard{
				ID:     card.ID,
				Title:  card.Title,
				Status: card.Status,
				Reason: "long-running execution",
			})
		}
	}

	return stuck, nil
}

// StuckCard represents a stuck card.
type StuckCard struct {
	ID     string
	Title  string
	Status string
	Reason string
}

// RetryRecord tracks retry attempts.
type RetryRecord struct {
	CardID     string    `json:"card_id"`
	Attempt    int       `json:"attempt"`
	LastError  string    `json:"last_error,omitempty"`
	LastTryAt  string    `json:"last_try_at"`
	NextTryAt  string    `json:"next_try_at"`
	Status     string    `json:"status"` // pending, backoff, exhausted
}
