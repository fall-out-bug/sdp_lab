// Package confidence provides a generic confidence wrapper around LLM inference
// calls. It composes self-check, redundancy, and constraint-based strategies
// into an aggregate Score and Status (OK / UNSURE / FAIL) so callers can route
// low-confidence results to retry, human review, or conservative fallbacks.
//
// See docs/plans/2026-04-26-f144-inference-confidence-design.md for the full
// design and intended call-sites (ws-verdict, architect classify, dispatch
// classify).
package confidence

// Status is the gating verdict on an inference result.
type Status string

const (
	// StatusOK — score at or above the OK threshold; result is acceptable.
	StatusOK Status = "ok"
	// StatusUnsure — score in the band between FAIL and OK thresholds; caller
	// should route to retry, human review, or conservative fallback per policy.
	StatusUnsure Status = "unsure"
	// StatusFail — score below the FAIL threshold or a hard-fail strategy
	// (e.g. broken format) tripped; result must not be used.
	StatusFail Status = "fail"
)

// Valid reports whether s is one of the defined Status constants.
func (s Status) Valid() bool {
	switch s {
	case StatusOK, StatusUnsure, StatusFail:
		return true
	default:
		return false
	}
}
