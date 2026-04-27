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

Per-stage call order: **cascade → confidence → stitcher**.

```
Cascade (F145, provider escalation)   ← optional, wraps Stage.Run thunk
  → Stage.Run (raw LLM call)
Confidence (F144, quality gate)       ← optional, wraps stage output
  → SubScore / Status derived
Stitcher (format gate)                ← optional, validates out shape
```

### Wiring confidence (F144)

```go
strat, _ := constraint.New[MyOut](constraint.Options[MyOut]{...})
checker, _ := confidence.NewChecker[MyOut](llmCaller, []confidence.Strategy[MyOut]{strat}, policy)

runner := decompose.NewConfidenceRunner(checker)   // type-erased wrapper
decompose.Then(p, myStage, decompose.StageConfig{
    Confidence: runner,
})
// StageResult.SubScore ← confidence.Result.Score
// StageResult.Status  ← confidence.Result.Status (OK/UNSURE/FAIL)
// StageTrace.ConfidenceLog populated
```

### Wiring cascade (F145 — forward-compatible narrow interface)

```go
// F145 not yet merged (sdplab-5ii8). Use decompose.CascadeInvoker interface.
// After F145 merges, swap in the real cascade.Invoker — API is identical.
type myCascade struct{}
func (m *myCascade) Invoke(ctx context.Context, fn func() (any, decompose.StageTrace, error)) (any, decompose.StageTrace, decompose.CascadeTrace, error) {
    // escalate provider, call fn, retry on failure
}
decompose.Then(p, myStage, decompose.StageConfig{
    Cascade: &myCascade{},
})
// StageTrace.CascadeLog populated with Provider, Attempts
```

### Status propagation

| Stage outcome             | StageResult.Status | StageResult.SubScore |
|---------------------------|--------------------|----------------------|
| OK (no confidence)        | OK                 | 1.0                  |
| Confidence OK             | OK                 | confidence.Score     |
| Confidence UNSURE         | UNSURE             | confidence.Score     |
| Confidence FAIL / error   | FAIL               | 0.0                  |
| Stage error (Fallback)    | FAIL               | 0.0                  |

Pipeline `Result.Status` = aggregated: any FAIL → FAIL; any UNSURE (no FAIL) → UNSURE; all OK → OK.

## MicroFirst tier (F147)

`WithEscalation` composes a cheap micro-classifier and a full LLM stage into a
single `Stage[In, Out]`. The micro stage runs first; the LLM is only invoked
when the micro output lacks confidence.

```go
// Define an output type that reports its own confidence.
type ClassifyOut struct {
    Label      string
    conf       float64
    confStatus decompose.Status
}

func (c ClassifyOut) Confidence() float64          { return c.conf }
func (c ClassifyOut) ConfStatus() decompose.Status { return c.confStatus }

// Build stages.
micro := decompose.NewStage[Diff, ClassifyOut]("micro-heuristic", func(ctx context.Context, d Diff) (ClassifyOut, decompose.StageTrace, error) {
    // fast regex / rule-based classifier
    return ClassifyOut{Label: label, conf: score, confStatus: status}, trace, nil
})

llm := decompose.NewStage[Diff, ClassifyOut]("llm-sonnet", func(ctx context.Context, d Diff) (ClassifyOut, decompose.StageTrace, error) {
    // full LLM call
    return ClassifyOut{Label: label, conf: 1.0, confStatus: decompose.StatusOK}, trace, nil
})

// Compose: micro runs first; if confidence >= 0.85 and status OK, llm is skipped.
composed := decompose.WithEscalation(micro, llm, decompose.EscalationConfig{
    ConfidenceThreshold: 0.85,
    EscalateOnError:     true,   // micro error → try llm
    RecordSkippedTrace:  true,   // trace Attempts=1, TokensIn/Out=0 when skipped
})

// Drop into a pipeline as any other stage.
p := decompose.New[ClassifyOut]("ws-classify")
decompose.Then(p, composed, decompose.StageConfig{OnFailure: decompose.Abort})
result, err := p.Run(ctx, diff)
// result.Trace reflects only the tokens actually consumed.
```

### Escalation decision table

| micro result                        | EscalateOnError | Action          | Attempts |
|-------------------------------------|-----------------|-----------------|----------|
| Confider OK, confidence >= threshold | any             | return micro    | 1        |
| Confider OK, confidence < threshold  | any             | invoke llm      | 2        |
| Confider status != OK               | any             | invoke llm      | 2        |
| Out does not implement Confider     | any             | invoke llm      | 2        |
| error                               | true            | invoke llm      | 2        |
| error                               | false           | propagate error | —        |
