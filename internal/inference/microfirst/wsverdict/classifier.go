package wsverdict

import (
	"context"

	"sdp_dev/internal/inference/decompose"
)

// WsVerdictMicro implements decompose.Stage[WsVerdictInput, WsVerdictMicroResult].
// Rules (applied in order, first match wins):
//
//	R1: Report.Failed > 0                                   → FAIL,   confidence=0.95
//	R2: Failed==0 && Errored==0 && len(OutOfScope)==0       → PASS,   confidence=0.90
//	R3: len(Guard.OutOfScope) > 0                           → UNSURE, confidence=0.40
//	R4: Report.Skipped > cfg.SkipThreshold                  → UNSURE, confidence=0.50
//	R5: MinCoverage>0 && Report.Coverage<MinCoverage        → UNSURE, confidence=0.45
//	R6: catch-all                                           → UNSURE, confidence=0.30
type WsVerdictMicro struct{ cfg RulesConfig }

// New constructs a WsVerdictMicro with the given RulesConfig.
func New(cfg RulesConfig) *WsVerdictMicro { return &WsVerdictMicro{cfg: cfg} }

// Name implements decompose.Stage.
func (w *WsVerdictMicro) Name() string { return "wsverdict-micro" }

// Run implements decompose.Stage.
func (w *WsVerdictMicro) Run(_ context.Context, in WsVerdictInput) (WsVerdictMicroResult, decompose.StageTrace, error) {
	res := w.classify(in)
	return res, decompose.StageTrace{Attempts: 1}, nil
}

func (w *WsVerdictMicro) classify(in WsVerdictInput) WsVerdictMicroResult {
	// R1: any test failures → definitive FAIL
	if in.Report.Failed > 0 {
		return WsVerdictMicroResult{
			Verdict:    VerdictFail,
			confidence: 0.95,
			status:     decompose.StatusOK,
			Reasons:    []string{"tests failed"},
		}
	}

	// Pre-compute thresholds used by R2 and R4/R5 to avoid duplication.
	skipThreshold := w.cfg.SkipThreshold
	if skipThreshold == 0 {
		skipThreshold = 5
	}
	// Effective coverage floor: per-call value takes precedence; fall back to
	// the configured RulesConfig value so New(RulesConfig{MinCoverage: X}) works
	// even when the caller omits MinCoverage from WsVerdictInput.
	minCov := in.MinCoverage
	if minCov == 0 {
		minCov = w.cfg.MinCoverage
	}
	coverageOK := minCov == 0 || in.Report.Coverage >= minCov

	// R2: no failures, no errors, no out-of-scope files, skipped within threshold,
	// coverage meets minimum → confident PASS.
	// Must check all secondary gates here to avoid bypassing R4/R5 for edge inputs.
	if in.Report.Errored == 0 && len(in.Guard.OutOfScope) == 0 &&
		in.Report.Skipped <= skipThreshold && coverageOK {
		return WsVerdictMicroResult{
			Verdict:    VerdictPass,
			confidence: 0.90,
			status:     decompose.StatusOK,
			Reasons:    []string{"all tests pass, guard clean"},
		}
	}

	// R3: out-of-scope files present
	if len(in.Guard.OutOfScope) > 0 {
		return WsVerdictMicroResult{
			Verdict:    VerdictUnsure,
			confidence: 0.40,
			status:     decompose.StatusUnsure,
			Reasons:    []string{"guard out-of-scope files detected"},
		}
	}

	// R4: too many skipped tests
	if in.Report.Skipped > skipThreshold {
		return WsVerdictMicroResult{
			Verdict:    VerdictUnsure,
			confidence: 0.50,
			status:     decompose.StatusUnsure,
			Reasons:    []string{"too many skipped tests"},
		}
	}

	// R5: coverage below minimum
	if !coverageOK {
		return WsVerdictMicroResult{
			Verdict:    VerdictUnsure,
			confidence: 0.45,
			status:     decompose.StatusUnsure,
			Reasons:    []string{"coverage below minimum"},
		}
	}

	// R6: catch-all
	return WsVerdictMicroResult{
		Verdict:    VerdictUnsure,
		confidence: 0.30,
		status:     decompose.StatusUnsure,
		Reasons:    []string{"no confident rule matched"},
	}
}
