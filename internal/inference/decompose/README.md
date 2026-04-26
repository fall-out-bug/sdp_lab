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

## Stitchers

Stitchers enforce a strict wire format on stage outputs. Use `StageConfig.Stitcher`
(wired in 00-146-03) to attach one per stage.

### EnumStitcher — closed-set classification

```go
s := decompose.NewEnumStitcher("verdict", []string{"passed", "partial", "failed"})
// Validate("passed") → nil
// Marshal("passed")  → "passed" (no quoting)
```

### JSONStitcher — schema-validated structured extraction

```go
s, _ := decompose.NewJSONStitcherFromBytes("extract", schemaBytes)
// Validate(map[string]any{...}) → nil if schema passes
// Marshal(map[string]any{...}) → indented JSON; round-trip guaranteed
```

### TOONStitcher — token-optimized flat table (≥40% saving vs JSON)

```go
cols := []decompose.TOONColumn{
    {Name: "file",   Type: "string"},
    {Name: "status", Type: "string"},
    {Name: "score",  Type: "float"},
}
s := decompose.NewTOONStitcher("aggregate", cols)
// Marshal([]map[string]any{...}) produces:
//   # file | status | score
//   internal/foo.go | pass | 0.9
//   internal/bar.go | warn | 0.5
```

**Token saving** — on a 20-row fixture the TOON output is ~43% the length of
indented JSON (measured in characters, a proxy for BPE token count).

**Round-trip:** `Marshal → decompose.ParseTOON → Validate` is guaranteed valid.

**v1 limitation:** nested objects in cell values are not supported — Validate
returns an error.

## Composition with F144/F145

Confidence and cascade wrapping per stage is added in `00-146-03`
(`internal/inference/decompose/integration.go`). The core pipeline has no
direct dependency on those packages; it accepts any `Stage[In,Out]` impl.
