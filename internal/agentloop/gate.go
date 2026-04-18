package agentloop

import (
	"context"
	"time"

	"sdp_dev/internal/harness"
)

// GateEngine wraps harness.EvaluateCompliance with a circuit breaker timeout.
//
// Critical note: harness.EvaluateCompliance does NOT accept a context parameter.
// Signature: EvaluateCompliance(contract *TaskContract, snapshot *TaskSnapshot) ComplianceReport
// GateEngine wraps it in a goroutine and uses select on result channel + evalCtx.Done().
// Fix N4: goroutine selects on both result channel and evalCtx.Done() — no hang after timeout.
// Fix R2-3: timeout → Escalated=true with GateWarn, NOT silent pass.
type GateEngine struct {
	contract *harness.TaskContract
	timeout  time.Duration

	// evalFn is the evaluation function. Defaults to harness.EvaluateCompliance.
	// Overridable in tests to inject slow/blocking behavior for timeout tests.
	evalFn func(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport
}

// NewGateEngine creates a GateEngine with the given contract and timeout.
// If timeout is zero, defaults to 5 seconds.
func NewGateEngine(contract *harness.TaskContract, timeout time.Duration) *GateEngine {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &GateEngine{
		contract: contract,
		timeout:  timeout,
		evalFn:   harness.EvaluateCompliance,
	}
}

// NewPassingGateEngine creates a GateEngine that always passes (never escalates).
// Intended for tests that need the completion_signal path to succeed without
// providing real compliance evidence.
func NewPassingGateEngine() *GateEngine {
	return &GateEngine{
		evalFn: func(_ *harness.TaskContract, _ *harness.TaskSnapshot) harness.ComplianceReport {
			return harness.ComplianceReport{Blocked: false}
		},
		timeout: 5 * time.Second,
	}
}

// Evaluate runs compliance evaluation with a circuit breaker timeout.
// If evaluation completes in time: returns GateResult with Escalated=true iff report.Blocked.
// If evaluation times out: returns GateResult with Escalated=true + GateWarn violation (Fix R2-3).
// If context is already cancelled: treated as timeout (also escalates).
func (g *GateEngine) Evaluate(ctx context.Context, snap PhaseSnapshot) GateResult {
	evalCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	ch := make(chan harness.ComplianceReport, 1)

	// Fix N4: goroutine selects on both ch and evalCtx.Done() — exits promptly on timeout.
	go func() {
		report := g.evalFn(g.contract, snap.toHarness())
		select {
		case ch <- report:
		case <-evalCtx.Done():
			// Timeout already fired while we were evaluating — just discard result.
		}
	}()

	select {
	case report := <-ch:
		if report.Blocked {
			// Gate blocked → escalate for human decision.
			return GateResult{Report: report, Escalated: true}
		}
		return GateResult{Report: report, Escalated: false}

	case <-evalCtx.Done():
		// Fix R2-3: timeout is NOT automatic pass. Escalated=true requires human review.
		// Blocked=false so the automated path does not block; Escalated triggers human gate.
		return GateResult{
			Report: harness.ComplianceReport{
				Blocked: false,
				GateResults: []harness.GateResult{{
					GateID: "gate_timeout",
					Status: harness.GateWarn,
					Violations: []harness.Violation{{
						Type:    harness.DriftProcessIncomplete,
						Message: "gate evaluation timed out — human review required before transition",
					}},
				}},
			},
			Escalated: true,
		}
	}
}
