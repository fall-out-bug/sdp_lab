package bdseverity

import (
	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose"
	"github.com/fall-out-bug/sdp_lab/internal/inference/microfirst/knn"
)

// BdSeverityResult is the output of BdSeverityMicro.Run.
// Implements decompose.Confider.
type BdSeverityResult struct {
	Priority   string  // "P0"|"P1"|"P2"|"P3"
	confidence float64
	status     decompose.Status
	Neighbors  []knn.Match[string]
}

// Confidence implements decompose.Confider.
func (r BdSeverityResult) Confidence() float64 { return r.confidence }

// ConfStatus implements decompose.Confider.
func (r BdSeverityResult) ConfStatus() decompose.Status { return r.status }
