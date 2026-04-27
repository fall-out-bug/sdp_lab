package cascade

import (
	"context"
	"fmt"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence"
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
// - If status == StatusUnsure, returns (false, "confidence_unsure: <reason>") regardless of score.
//   Per F145 design §4.7, UNSURE must always escalate.
// - If status == StatusOK:
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
	// Per F145 §4.7: StatusUnsure must always escalate
	if result.Status == confidence.StatusUnsure {
		reasonStr := ""
		if len(result.Reasons) > 0 {
			reasonStr = result.Reasons[0]
		}
		return false, fmt.Sprintf("confidence_unsure: %s", reasonStr)
	}

	// StatusOK with score >= threshold → pass
	if result.Status == confidence.StatusOK && result.Score >= ca.threshold {
		return true, ""
	}

	// StatusFail or score < threshold → fail
	return false, "confidence_below_threshold"
}
