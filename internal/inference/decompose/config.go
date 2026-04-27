package decompose

import "time"

// FailurePolicy controls how Pipeline.Run reacts when a stage returns an error.
type FailurePolicy int

const (
	// Abort stops the pipeline immediately and returns the stage error.
	Abort FailurePolicy = iota
	// RetryOnce retries the stage once with the same input; if the retry also
	// fails, behaves like Abort.
	RetryOnce
	// Fallback swaps the stage error for StageConfig.FallbackOut and continues.
	// The stage StageResult is recorded with StatusFail and SubScore 0.0.
	Fallback
)

// StageConfig holds per-stage execution parameters.
type StageConfig struct {
	// OnFailure determines error-handling behavior (default: Abort).
	OnFailure FailurePolicy
	// FallbackOut is the value used when OnFailure == Fallback and the stage
	// errors. Must be type-compatible with the stage's Out type.
	FallbackOut any
	// Timeout, if non-zero, wraps the stage context with context.WithTimeout.
	Timeout time.Duration
	// Confidence, if non-nil, wraps the stage output with a F144 confidence
	// check. StageResult.SubScore and Status are derived from the check result.
	// Call order: Cascade → Confidence → Stitcher.
	Confidence ConfidenceRunner
	// Cascade, if non-nil, wraps the stage's raw LLM call with F145
	// provider-escalation. Swapped for real F145 after sdplab-5ii8 merges.
	Cascade CascadeInvoker
	// Stitcher, if non-nil, validates the stage output format after Confidence.
	Stitcher Stitcher
}
