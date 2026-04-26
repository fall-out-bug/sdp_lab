# `selfcheck` — critic-prompt self-check strategy

Asks the model (or a separate critic instance of the same model) to review its own answer. Two modes trade signal strength against round-trip cost.

## Modes

### `ModeFull` (default)

One extra LLM call. The critic receives the original input plus the candidate raw answer and is asked to return strict JSON:

```json
{"verdict": "agree" | "disagree" | "unsure", "confidence": 0.0-1.0, "issues": ["..."]}
```

Critic temperature is forced to **0.0** (we want a stable verdict, not a sample). `MaxTokens` is 256 — enough for short verdicts and a few issues.

SubScore mapping:

| Verdict | SubScore |
|---|---|
| `agree` | `confidence` |
| `disagree` | `1 - confidence` |
| `unsure` | `0.5` |
| unknown / malformed | `0.5` |

Use full mode for **critical paths** like ws-verdict where one extra round-trip is worth the stronger signal.

### `ModeLite`

No LLM calls. A caller-supplied extractor reads a `self_score: X` annotation from the primary call's raw text. Strategy returns that score directly. Annotation missing → SubScore=0.5.

Use lite mode for **hot paths** (dispatch classify) where doubling round-trips would blow the latency budget. The trade-off: signal is weaker — the model is grading its own first-pass output without the role-swap that full mode enforces.

## Construction

```go
// Full mode — relies on a Caller injected at Run time via StrategyInput.
s, _ := selfcheck.New[MyAnswer](selfcheck.Options[MyAnswer]{
    Mode: selfcheck.ModeFull,
})

// Full mode with custom prompt template:
s, _ := selfcheck.New[MyAnswer](selfcheck.Options[MyAnswer]{
    Mode: selfcheck.ModeFull,
    CriticPromptTemplate: func(input, raw string) string {
        return "Domain-specific critic instructions for input " + input + " ..."
    },
})

// Lite mode:
s, _ := selfcheck.New[MyAnswer](selfcheck.Options[MyAnswer]{
    Mode: selfcheck.ModeLite,
    LiteScoreExtractor: func(raw string) (float64, bool) {
        // Parse `self_score: 0.85` line out of raw, etc.
        return parsed, found
    },
})
```

## Graceful degradation

Bad critic output (malformed JSON, unknown verdict, out-of-range confidence) **never hard-fails**. The strategy returns a neutral `0.5` with a descriptive `Reason`. Rationale: a flaky critic shouldn't override a correct primary answer — that's what `constraint` and `nsample` are for.

Caller errors from full-mode critic call DO propagate (wrapped with `%w`) — those are infrastructure failures, not signal noise.

## Output

```go
out.SubScore   // 0..1 per the table above
out.HardFail   // always false — selfcheck never hard-fails
out.Reason     // "critic agreed (conf 0.92)" / "lite mode: self_score=0.7" / etc.
out.Tokens     // usage from the critic call (full mode), zero (lite)
out.Log        // selfcheck.Log{Mode, Verdict, CriticConfidence, Issues, RawCritic}
```
