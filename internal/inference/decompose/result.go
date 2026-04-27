// Package decompose provides the generic multi-stage inference pipeline:
// Pipeline[Final], Stage[In,Out], Result[Final], and supporting types.
// Use New[Final](name).Then(stage, cfg).Then(...).Run(ctx, input) to compose
// a sequential chain of typed stages with per-stage failure policies.
package decompose

// Result is the output of a Pipeline.Run call.
type Result[Final any] struct {
	Answer       Final
	Status       Status
	Score        float64
	StageResults []StageResult
	Trace        AggregateTrace
	Reasons      []string
}

// StageResult captures the output and evidence of a single stage execution.
type StageResult struct {
	Name     string
	Status   Status
	SubScore float64
	Out      any
	Trace    StageTrace
	Err      error
}

// StageTrace captures telemetry for one stage invocation.
type StageTrace struct {
	LatencyMs     int64
	TokensIn      int
	TokensOut     int
	CostUSD       float64
	Attempts      int
	ConfidenceLog *ConfidenceLog
	CascadeLog    *CascadeTrace
}

// ConfidenceLog records the F144 confidence check result for a stage.
type ConfidenceLog struct {
	Score   float64
	Status  Status
	Reasons []string
}

// AggregateTrace is the sum of all stage traces.
type AggregateTrace struct {
	LatencyMs int64
	TokensIn  int
	TokensOut int
	CostUSD   float64
}

func (t *AggregateTrace) add(s StageTrace) {
	t.LatencyMs += s.LatencyMs
	t.TokensIn += s.TokensIn
	t.TokensOut += s.TokensOut
	t.CostUSD += s.CostUSD
}

func meanScore(results []StageResult) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range results {
		sum += r.SubScore
	}
	return sum / float64(len(results))
}
