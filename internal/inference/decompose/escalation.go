package decompose

import (
	"context"
	"time"
)

// Confider is implemented by micro-classifier outputs that report self-confidence.
// WithEscalation checks this interface to decide whether to invoke the LLM stage.
type Confider interface {
	Confidence() float64 // [0, 1]
	ConfStatus() Status  // StatusOK / StatusUnsure / StatusFail
}

// EscalationConfig controls when to escalate from micro to llm stage.
type EscalationConfig struct {
	// ConfidenceThreshold: micro Out is trusted if Status==OK and Confidence >= threshold.
	ConfidenceThreshold float64

	// EscalateOnError: if true, micro errors trigger llm fallback.
	// If false, micro errors propagate as Stage error (llm not invoked).
	EscalateOnError bool

	// RecordSkippedTrace: if true and llm is skipped, the returned StageTrace
	// still has Attempts=1 with TokensIn/Out=0. Useful for "saved tokens" accounting.
	RecordSkippedTrace bool
}

// WithEscalation composes (micro, llm) into a single Stage[In, Out].
//
// Run logic:
//  1. Run micro. If err and !EscalateOnError → propagate err.
//  2. If micro Out implements Confider and ConfStatus()==OK and Confidence()>=threshold
//     → return micro Out + micro trace (escalated=false, Attempts=1).
//  3. Else → run llm with same In; return llm Out + combined trace
//     (micro trace latency + llm trace; Attempts=2; tokens from llm only).
func WithEscalation[In, Out any](
	micro Stage[In, Out],
	llm Stage[In, Out],
	cfg EscalationConfig,
) Stage[In, Out] {
	return NewStage[In, Out](
		micro.Name()+"+escalation",
		func(ctx context.Context, in In) (Out, StageTrace, error) {
			// Run micro
			start := time.Now()
			microOut, microTrace, microErr := micro.Run(ctx, in)
			microTrace.LatencyMs = time.Since(start).Milliseconds()

			if microErr != nil {
				if !cfg.EscalateOnError {
					return microOut, microTrace, microErr
				}
				// escalate on error: fall through to llm
			} else {
				// check confidence
				if c, ok := any(microOut).(Confider); ok {
					if c.ConfStatus() == StatusOK && c.Confidence() >= cfg.ConfidenceThreshold {
						// micro is confident — short-circuit
						if cfg.RecordSkippedTrace {
							// Return a "saved tokens" trace: latency preserved, tokens zeroed.
							skipped := StageTrace{
								LatencyMs: microTrace.LatencyMs,
								Attempts:  1,
							}
							return microOut, skipped, nil
						}
						return microOut, microTrace, nil
					}
				}
				// not confident enough — fall through to llm
			}

			// run llm
			llmStart := time.Now()
			llmOut, llmTrace, llmErr := llm.Run(ctx, in)
			llmTrace.LatencyMs = time.Since(llmStart).Milliseconds()

			combined := StageTrace{
				LatencyMs: microTrace.LatencyMs + llmTrace.LatencyMs,
				TokensIn:  llmTrace.TokensIn,
				TokensOut: llmTrace.TokensOut,
				CostUSD:   llmTrace.CostUSD,
				Attempts:  2,
			}
			return llmOut, combined, llmErr
		},
	)
}
