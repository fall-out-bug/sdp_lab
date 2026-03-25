package executor

import (
	"testing"
	"time"

	"sdp_dev/internal/control"
)

func TestRankAndPick_Empty(t *testing.T) {
	policy := DefaultRankingPolicy()
	result := RankAndPick(nil, policy, nil)
	if result != "" {
		t.Error("empty list should return empty")
	}
}

func TestRankAndPick_SingleItem(t *testing.T) {
	ready := []control.FeatureCard{
		{ID: "abc", Title: "Fix bug", Status: "open"},
	}
	result := RankAndPick(ready, nil, nil)
	if result != "abc" {
		t.Errorf("expected abc, got %s", result)
	}
}

func TestRankAndPick_PriorityOrder(t *testing.T) {
	ready := []control.FeatureCard{
		{ID: "low", Title: "P3 minor fix", Status: "open"},
		{ID: "high", Title: "P1 critical auth bug", Status: "open"},
		{ID: "mid", Title: "P2 important feature", Status: "open"},
	}
	result := RankAndPick(ready, nil, nil)
	if result != "high" {
		t.Errorf("expected high priority, got %s", result)
	}
}

func TestRankAndPick_RetryPenalty(t *testing.T) {
	ready := []control.FeatureCard{
		{ID: "fresh", Title: "P2 new task", Status: "open"},
		{ID: "retried", Title: "P2 old task", Status: "open"},
	}
	retries := map[string]int{"retried": 2}
	result := RankAndPick(ready, nil, retries)
	if result != "fresh" {
		t.Errorf("expected fresh card over retried, got %s", result)
	}
}

func TestRankAndPick_MaxRetriesExhausted(t *testing.T) {
	ready := []control.FeatureCard{
		{ID: "exhausted", Title: "P1 task", Status: "open"},
	}
	retries := map[string]int{"exhausted": 5}
	policy := DefaultRankingPolicy()
	result := RankAndPick(ready, policy, retries)
	if result != "" {
		t.Error("exhausted card should not be picked")
	}
}

func TestRankAndPick_ExecutingBonus(t *testing.T) {
	ready := []control.FeatureCard{
		{ID: "fresh", Title: "P2 new task", Status: "open"},
		{ID: "resume", Title: "P2 in-progress", Status: "executing"},
	}
	result := RankAndPick(ready, nil, nil)
	if result != "resume" {
		t.Errorf("expected executing card to get bonus, got %s", result)
	}
}

func TestDefaultRankingPolicy(t *testing.T) {
	p := DefaultRankingPolicy()
	if p.MaxRetries != 3 {
		t.Error("default max retries should be 3")
	}
	if p.PriorityWeights[0] != 100 {
		t.Error("P1 weight should be 100")
	}
	if p.RetryBackoff != 5*time.Minute {
		t.Error("default backoff should be 5m")
	}
}

func TestStuckCard_Struct(t *testing.T) {
	s := StuckCard{ID: "x", Title: "stuck", Status: "executing", Reason: "timeout"}
	if s.ID != "x" {
		t.Error("struct mismatch")
	}
}

func TestRetryRecord_Struct(t *testing.T) {
	r := RetryRecord{CardID: "y", Attempt: 2, Status: "pending"}
	if r.Attempt != 2 {
		t.Error("struct mismatch")
	}
}
