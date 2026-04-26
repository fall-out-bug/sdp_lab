# decompose — multi-stage inference pipeline

Sequential typed pipeline for LLM inference decomposition. Each stage receives
the previous stage's output as its typed input. Per-stage failure policies,
timeouts, and trace aggregation are built-in.

## Canonical 3-stage usage

```go
// Stage types
type ExtractOut struct { Files []string }
type ClassifyOut struct { Verdict string }
type AggregateOut struct { Score float64; Summary string }

// Stage implementations (plug in real LLM calls)
extract := decompose.NewStage[Diff, ExtractOut]("extract", func(ctx context.Context, d Diff) (ExtractOut, decompose.StageTrace, error) {
    // call Haiku, parse JSON response
    return ExtractOut{Files: parsed}, trace, nil
})

classify := decompose.NewStage[ExtractOut, ClassifyOut]("classify", func(ctx context.Context, e ExtractOut) (ClassifyOut, decompose.StageTrace, error) {
    // call Sonnet with enum constraint
    return ClassifyOut{Verdict: verdict}, trace, nil
})

aggregate := decompose.NewStage[ClassifyOut, AggregateOut]("aggregate", func(ctx context.Context, c ClassifyOut) (AggregateOut, decompose.StageTrace, error) {
    // call Haiku, emit TOON table
    return AggregateOut{Score: score, Summary: summary}, trace, nil
})

// Build pipeline
p := decompose.New[AggregateOut]("ws-verdict")
decompose.Then(p, extract, decompose.StageConfig{
    Timeout:   5 * time.Second,
    OnFailure: decompose.RetryOnce,
})
decompose.Then(p, classify, decompose.StageConfig{
    OnFailure: decompose.Abort,
})
decompose.Then(p, aggregate, decompose.StageConfig{
    OnFailure: decompose.Fallback,
    FallbackOut: AggregateOut{Score: 0, Summary: "aggregate failed"},
})

// Execute
result, err := p.Run(ctx, inputDiff)
if err != nil {
    // terminal error — see result.StageResults for partial evidence
}
// result.Answer is AggregateOut from last stage
// result.Status: OK / UNSURE / FAIL (aggregate of all stages)
// result.Score: mean SubScore of stages (default 1.0 for OK, 0.0 for FAIL)
// result.Trace: aggregate tokens / cost / latency
```

## Failure policies

| Policy      | Behaviour on stage error                                        |
|-------------|-----------------------------------------------------------------|
| `Abort`     | Pipeline stops; returns error to caller. (default)              |
| `RetryOnce` | Stage retried once. Second failure → Abort.                     |
| `Fallback`  | Error swapped for `StageConfig.FallbackOut`; pipeline continues.|

## Composition with F144/F145

Confidence and cascade wrapping per stage is added in `00-146-03`
(`internal/inference/decompose/integration.go`). The core pipeline has no
direct dependency on those packages; it accepts any `Stage[In,Out]` impl.
