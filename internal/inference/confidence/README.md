# `internal/inference/confidence`

Generic confidence wrapper над LLM-инференсом. Композирует **self-check**, **redundancy** (N-sample) и **constraint-based** проверки в единый `Score ∈ [0, 1]` и `Status ∈ {OK, UNSURE, FAIL}`, чтобы вызывающий код мог явно маршрутизировать низкоуверенные ответы в retry / human review / conservative fallback.

> **Feature:** F144 — design в [docs/plans/2026-04-26-f144-inference-confidence-design.md](../../../docs/plans/2026-04-26-f144-inference-confidence-design.md). Trekается через beads epic `sdplab-sjdp`.

## When to use

Подключай этот пакет к LLM call-site'у, **где ошибка стоит дорого**:
- Решение «passed/failed» на артефакте (ws-verdict, gate prompts).
- Структурная классификация, где результат идёт в downstream-анализ (architect classify).
- Routing-решения с асимметричной ценой ошибки (dispatch classify — lite mode).

Не подключай для embedding-вызовов или генеративных задач, где «один правильный ответ» не определён.

## API surface

```go
// Decision shape.
type Status string                      // "ok" | "unsure" | "fail"
type Result[T any] struct {
    Answer    T
    Status    Status
    Score     float64                   // [0, 1] aggregate
    SubScores map[string]float64        // per-strategy
    Reasons   []string                  // human-readable
    Trace     Trace                     // latency, tokens, cost
    Attempts  int
}

// Per-call-site configuration.
type Policy struct {
    OKThreshold    float64               // default 0.8
    FailThreshold  float64               // default 0.5
    Weights        map[string]float64    // strategy name -> weight
    UnsureBehavior UnsureBehavior        // retry_once | human_handoff | conservative_fallback
    MaxLatencyMs   int64                 // 0 = unlimited
    MaxCostUSD     float64               // 0 = unlimited (reserved, F144-02+)
}

// Strategy contract.
type Strategy[T any] interface {
    Name() string
    Run(ctx context.Context, in StrategyInput[T]) (StrategyOutput, error)
}

// Orchestrator.
func NewChecker[T any](caller LLMCaller, strategies []Strategy[T], policy Policy) (*Checker[T], error)
func (c *Checker[T]) Check(ctx context.Context, req Request[T]) (Result[T], error)
```

## Canonical usage

```go
import (
    "context"
    "github.com/fall-out-bug/sdp_lab/internal/inference/confidence"
)

// 1. Build strategies appropriate for the call-site. F144-01 ships only
//    the orchestration layer; concrete strategies arrive in F144-02..04.
var strategies []confidence.Strategy[MyAnswer] = []confidence.Strategy[MyAnswer]{
    /* selfcheck.New(...), nsample.New(...), constraint.New(...) */
}

// 2. Tune policy per call-site. DefaultPolicy() is the starting point.
policy := confidence.DefaultPolicy()
policy.UnsureBehavior = confidence.UnsureHumanHandoff   // for ws-verdict
policy.MaxLatencyMs   = 5000                            // 5s soft budget

// 3. Construct the checker. caller is your LLM client (may be nil if all
//    strategies are pure validators).
checker, err := confidence.NewChecker(myLLMCaller, strategies, policy)
if err != nil { return err }

// 4. Run after the primary inference call has produced an answer.
res, err := checker.Check(ctx, confidence.Request[MyAnswer]{
    Input:  userPrompt,
    Answer: parsedAnswer,
    Raw:    rawResponseText,
})
if err != nil { return err }

switch res.Status {
case confidence.StatusOK:
    return res.Answer, nil
case confidence.StatusUnsure:
    // Route per policy.UnsureBehavior:
    // - UnsureRetryOnce: re-run with higher N
    // - UnsureHumanHandoff: bd human <id>
    // - UnsureConservativeFallback: emit safe default
case confidence.StatusFail:
    // Reject; do not use res.Answer.
}
```

## Defaults & tuning

`DefaultPolicy()` ships:

| Field | Value | Reason |
|---|---|---|
| `OKThreshold` | `0.8` | High enough that ≥0.8 means "all strategies broadly agreed" |
| `FailThreshold` | `0.5` | Below midpoint of weighted average — at least one strategy strongly disagreed |
| `Weights["self_check"]` | `0.4` | Critic catches semantic drift; not as reliable as consensus alone |
| `Weights["consensus"]` | `0.4` | N-sample agreement is the strongest signal of stability |
| `Weights["constraint"]` | `0.2` | Format/invariants are necessary but coarse — they say "didn't break", not "got it right" |
| `UnsureBehavior` | `UnsureRetryOnce` | Cheapest mitigation; switch to `HumanHandoff` for production-blocking gates |

Tune per call-site rather than mutating `DefaultPolicy`. Configurations live inline at the call-site (see F144-05/06/07 adapters) until > 5 sites need them, at which point migrate to `configs/inference-confidence.yaml` (F144 design §7 open question 2).

## Composition rules

`Policy.Compose(subscores)` is a **weighted average over present strategies**:

- Strategies present in `subscores` but absent from `Policy.Weights` are ignored (not zero-weighted).
- Strategies present in `Policy.Weights` but absent from `subscores` do not penalize the score (lite-mode skipping is free).
- A strategy with `HardFail = true` forces `Status = FAIL` regardless of composed score; the score is still computed and visible for debugging.
- Out-of-range subscores (`< 0`, `> 1`, `NaN`, `±Inf`) return an error rather than silently clamping.

## Budget enforcement

- `MaxLatencyMs` is a **soft budget** enforced before each strategy. Strategies whose start would exceed it are skipped with `subscore = 0.5` (neutral) and `reason = "<name>: skipped: budget"`. The composed score absorbs the neutral contribution.
- `MaxCostUSD` is **reserved** for F144-02+ where strategies report token usage and pricing is wired in.

## Roadmap (F144 children)

- **F144-01 (this package, P1)** — types, Policy, Checker. ✅ done.
- **F144-02** — `selfcheck` strategy: critic-prompt with role swap.
- **F144-03** — `nsample` strategy: temperature-jittered redundancy.
- **F144-04** — `constraint` strategy: schema + invariants.
- **F144-05** — adapter `cmd/sdp-ws-verdict-validate` second-opinion.
- **F144-06** — adapter `internal/architect/classify`.
- **F144-07** — adapter `internal/dispatch/classify` (lite).
- **F144-08** — replay corpus + metrics report.

## Test strategy

- All public surface is unit-tested via `*_test.go` (96%+ coverage as of F144-01 close).
- Strategies in F144-02/03/04 add their own tests; integration tests live under `-tags=integration` and require `OPENROUTER_API_KEY`.
- End-to-end replay corpus + metrics report ships in F144-08 under `-tags=replay`.
