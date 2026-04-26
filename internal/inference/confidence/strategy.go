package confidence

import "context"

// TokenUsage tracks LLM token consumption from one or more calls. Strategies
// report what they spent so the Checker can aggregate cost into Trace.
type TokenUsage struct {
	In  int
	Out int
}

// Add returns the element-wise sum of two TokenUsage values without mutating
// either receiver.
func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	return TokenUsage{In: u.In + other.In, Out: u.Out + other.Out}
}

// LLMCaller is the minimum LLM interface a strategy needs: send a prompt and
// get text back with token accounting. The Checker injects a concrete caller
// (typically wrapping internal/architect/llm_client.go or modelgateway) and
// strategies invoke it for self-check critic prompts or N-sample redundancy.
type LLMCaller interface {
	Call(ctx context.Context, prompt string, opts CallOptions) (text string, usage TokenUsage, err error)
}

// CallOptions are the per-call knobs strategies adjust. Temperature is the
// primary lever for N-sample jitter; MaxTokens guards critic prompts that
// only need a short verdict.
type CallOptions struct {
	Temperature float64
	MaxTokens   int
}

// Request is the per-Check input: original user prompt plus the answer the
// primary call already produced. Strategies validate / re-sample this.
type Request[T any] struct {
	// Input is the original prompt or structured input that produced Answer.
	Input string
	// Answer is the parsed response from the primary inference call.
	Answer T
	// Raw is the unparsed text the primary call returned. Strategies that
	// re-parse (e.g. constraint schema check on raw JSON) need this in
	// addition to the parsed Answer.
	Raw string
}

// StrategyInput is what each Strategy.Run receives — the request plus the
// shared LLMCaller (which may be nil for strategies that don't need it,
// e.g. pure constraint validators).
type StrategyInput[T any] struct {
	Request Request[T]
	Caller  LLMCaller
}

// StrategyOutput is what each Strategy.Run returns. SubScore is in [0, 1].
// HardFail forces Status=FAIL regardless of composed score; use it for
// invariant violations where semantic confidence has no meaning (e.g.
// schema-broken JSON, type mismatch).
type StrategyOutput struct {
	SubScore float64
	HardFail bool
	Reason   string
	Tokens   TokenUsage
	// Log holds strategy-specific evidence (sample texts, critic verdicts,
	// schema errors). It is opaque to the Checker and surfaces verbatim in
	// Trace via the telemetry sink.
	Log any
}

// Strategy is one confidence-scoring technique. Implementations are
// stateless across calls; per-call state lives in Run's locals.
type Strategy[T any] interface {
	Name() string
	Run(ctx context.Context, in StrategyInput[T]) (StrategyOutput, error)
}
