package bdtype

import (
	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose"
	"github.com/fall-out-bug/sdp_lab/internal/inference/microfirst/knn"
)

// BdTypeResult is the output of BdTypeMicro.
// Implements decompose.Confider.
type BdTypeResult struct {
	Type       string
	confidence float64
	status     decompose.Status
	Neighbors  []knn.Match[string]
}

func (r BdTypeResult) Confidence() float64          { return r.confidence }
func (r BdTypeResult) ConfStatus() decompose.Status { return r.status }
