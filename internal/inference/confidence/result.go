package confidence

// Result is the output of a confidence-checked inference call. The Answer is
// generic over T so call-sites keep their own response type without an
// untyped any-cast at the boundary.
type Result[T any] struct {
	// Answer is the value the caller actually wanted from the LLM.
	Answer T
	// Status is the gating verdict derived from Score and any hard-fail
	// strategies; see Policy for thresholds.
	Status Status
	// Score is the aggregate confidence in [0, 1], composed from SubScores
	// using the per-call-site Policy weights.
	Score float64
	// SubScores breaks Score down by strategy name (e.g. "self_check",
	// "consensus", "constraint") for telemetry and debugging.
	SubScores map[string]float64
	// Reasons are human-readable explanations of why Status / Score is
	// what it is; populated by strategies and the composer.
	Reasons []string
	// Trace captures per-call evidence (samples, timings, tokens, cost).
	Trace Trace
	// Attempts records how many checker invocations produced this Result —
	// 1 for first-try, >1 if UnsureBehavior=RetryOnce kicked in.
	Attempts int
}

// Trace captures evidence about a confidence-check execution: which samples
// were drawn, what each strategy logged, and the resource cost. It is always
// populated, even on failure, so the replay harness can reproduce decisions.
type Trace struct {
	// LatencyMs is wall-clock duration of the entire Check call.
	LatencyMs int64
	// TokensIn is total prompt tokens summed across all LLM calls (main +
	// samples + critic).
	TokensIn int
	// TokensOut is total completion tokens summed across all LLM calls.
	TokensOut int
	// CostUSD is the monetary cost of the call, computed by the telemetry
	// sink from token counts and provider pricing.
	CostUSD float64
}

// TotalTokens returns the sum of input and output tokens for the trace.
func (t Trace) TotalTokens() int {
	return t.TokensIn + t.TokensOut
}
