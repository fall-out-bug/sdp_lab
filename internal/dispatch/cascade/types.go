package cascade

import (
	"context"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

// Checker is a confidence gate for cascade escalation decisions.
// It evaluates whether a response is acceptable or should be escalated to a higher tier.
// If nil, cascade treats all responses as acceptable (always-ok).
type Checker interface {
	// Check returns (ok, reason) where ok=true means the response is acceptable.
	// For F145-10, a stub implementation always returns (true, "").
	// F145-11 will wire F144's confidence.Checker here.
	Check(ctx context.Context, req InvokeRequest, resp *harness.Result) (ok bool, reason string)
}

// InvokeRequest holds the parameters for cascade invocation.
type InvokeRequest struct {
	Harness   string        // harness name (e.g. "claude", "codex")
	Prompt    string        // the prompt to invoke
	Agent     string        // agent type (e.g. "coder", "planner")
	Worktree  string        // project worktree path
	TaskFile  string        // task.json path
	Timeout   time.Duration // invocation timeout
	StartTier dispatch.TierClass
}

// InvokeResult holds the outcome of a cascade invocation.
type InvokeResult struct {
	Tier              dispatch.TierClass // final tier used
	Hops              int                // number of escalations (+1 initial = total attempts)
	Output            string             // final response text
	Cause             string             // "ok" | "max_depth" | "budget" | "checker_failed" | "no_profiles"
	TotalTokens       int                // aggregate token count (if available)
	LastError         string             // final error if Cause != "ok"
	ShortCircuitReason string             // reason for heuristic escalation, if any
}

// Budget holds time and cost constraints for cascade invocation.
type Budget struct {
	MaxDuration time.Duration // total wallclock budget
	MaxTokens   int           // max cumulative tokens (0 = unlimited)
	StartTime   time.Time     // when the budget was created
}

// Remaining returns the time remaining in the budget.
func (b *Budget) Remaining() time.Duration {
	elapsed := time.Since(b.StartTime)
	if b.MaxDuration <= elapsed {
		return 0
	}
	return b.MaxDuration - elapsed
}

// Exhausted returns true if budget is spent.
func (b *Budget) Exhausted() bool {
	return b.Remaining() <= 0
}

// ShortCircuitConfig holds heuristic thresholds for early escalation.
type ShortCircuitConfig struct {
	MinLengthChars  int      // response < N chars → escalate (default 50)
	RefusalPatterns []string // regex patterns for refusal ("I cannot", "I'm unable", "sorry, I can't")
	EmptyOK         bool     // whether empty response is treated as error (default false = error)
}

// DefaultShortCircuitConfig returns sensible defaults.
func DefaultShortCircuitConfig() ShortCircuitConfig {
	return ShortCircuitConfig{
		MinLengthChars: 50,
		RefusalPatterns: []string{
			`(?i)i\s+(cannot|can't|am\s+not\s+able)\b`,
			`(?i)sorry\s*,?\s*i\s+(cannot|can't)\b`,
			`(?i)unable\s+to\b`,
		},
		EmptyOK: false,
	}
}
