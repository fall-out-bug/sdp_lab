package wsverdict

import "sdp_dev/internal/inference/decompose"

// Verdict is the outcome of the micro classifier.
type Verdict string

const (
	VerdictPass   Verdict = "pass"
	VerdictFail   Verdict = "fail"
	VerdictUnsure Verdict = "unsure"
)

// WsVerdictMicroResult is the output of WsVerdictMicro.
// Implements decompose.Confider.
type WsVerdictMicroResult struct {
	Verdict    Verdict
	confidence float64
	status     decompose.Status
	Reasons    []string
}

func (r WsVerdictMicroResult) Confidence() float64          { return r.confidence }
func (r WsVerdictMicroResult) ConfStatus() decompose.Status { return r.status }
