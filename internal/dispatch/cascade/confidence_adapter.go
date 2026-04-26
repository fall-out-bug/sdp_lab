package cascade

import (
	"context"
	"fmt"

	"sdp_dev/internal/dispatch/harness"
	"sdp_dev/internal/inference/confidence"
)

// ConfidenceAdapter implements the cascade.Checker interface by wrapping
// a F144 confidence.Checker[string]. It adapts F144's confidence scoring
// (with Status and Score) into cascade's binary pass/fail decision based on
// a configurable threshold.
//
// If the wrapped F144 Checker is nil, the adapter degrades gracefully and
// treats all responses as acceptable (always returns ok=true).
type ConfidenceAdapter struct {
	checker   *confidence.Checker[string] // nil-safe: nil means no confidence gating
	threshold float64                      // score >= threshold → ok=true
}

// NewConfidenceAdapter creates a new ConfidenceAdapter that wraps a F144
// Checker. The threshold parameter (0.0-1.0) determines the confidence score
// cutoff: score >= threshold → ok=true, otherwise ok=false.
//
// If checker is nil, the adapter gracefully degrades and always returns ok=true.
func NewConfidenceAdapter(checker *confidence.Checker[string], threshold float64) *ConfidenceAdapter {
	return &ConfidenceAdapter{
		checker:   checker,
		threshold: threshold,
	}
}

// Check implements the cascade.Checker interface. It delegates to the wrapped
// F144 Checker and maps its Result to cascade's (ok, reason) semantics.
//
// Behavior:
// - If checker is nil, returns (true, "") — graceful degrade (always accept).
// - If checker returns an error, returns (false, "checker_error: <err>").
// - If status == StatusOK or status == StatusUnsure:
//   - score >= threshold → (true, "")
//   - score < threshold → (false, "confidence_below_threshold")
// - If status == StatusFail, returns (false, "confidence_below_threshold").
func (ca *ConfidenceAdapter) Check(ctx context.Context, req InvokeRequest, resp *harness.Result) (ok bool, reason string) {
	// Nil checker: graceful degrade
	if ca.checker == nil {
		return true, ""
	}

	// Build a F144 Request from cascade params
	// We only need Input and Raw; Answer is empty for now (generic over string, so "" is safe)
	f144Req := confidence.Request[string]{
		Input:  req.Prompt,
		Answer: "",
		Raw:    resp.Output,
	}

	// Call F144 Checker
	result, err := ca.checker.Check(ctx, f144Req)
	if err != nil {
		return false, fmt.Sprintf("checker_error: %v", err)
	}

	// Map F144 Result to cascade decision
	// Status=OK or Unsure with score >= threshold → pass
	// Status=Fail or score < threshold → fail
	if result.Status == confidence.StatusFail || result.Score < ca.threshold {
		return false, "confidence_below_threshold"
	}

	return true, ""
}
