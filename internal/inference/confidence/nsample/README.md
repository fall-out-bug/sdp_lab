# `nsample` — N-sample redundancy strategy

Re-queries the LLM with N temperature-jittered prompts in parallel and scores confidence by **mean pairwise agreement** over the parsed answers. The signal: if the model is genuinely confident, samples agree; instability is uncertainty.

## Signal

`SubScore = Agreement(parsedSamples) ∈ [0, 1]`

The `Agreement` function is **caller-supplied** — this package is agnostic to the metric (exact equality, structural match, semantic similarity, etc.). Pick what fits your `T`.

A typical equality-based agreement is mean pairwise match over the upper triangle:

```go
func equalityAgreement[T comparable](s []T) float64 {
    n := len(s)
    if n < 2 { return 1 }
    pairs, matches := 0, 0
    for i := 0; i < n; i++ {
        for j := i+1; j < n; j++ {
            pairs++
            if s[i] == s[j] { matches++ }
        }
    }
    return float64(matches) / float64(pairs)
}
```

## Configuration

```go
s, _ := nsample.New[MyAnswer](nsample.Options[MyAnswer]{
    Temperatures: []float64{0.0, 0.3, 0.7}, // N=3 default
    Parser:       parseAnswer,
    Agreement:    answerAgreement,
    MaxTokens:    1024,        // default
    Name:         "consensus", // default
})
```

`N = len(Temperatures)`. Constructor rejects empty `Temperatures` and nil `Parser` / `Agreement`.
Leave `BasePrompt` empty for reusable checkers; `Run` will re-sample the current `Request.Input`. Use `BasePrompt` only for a single-use strategy pinned to one prompt.

## Concurrency & errors

All `N` calls run in parallel via `errgroup`. The first call-level error cancels siblings and is returned wrapped with `%w`. Context cancellation propagates as `context.Canceled` / `context.DeadlineExceeded`.

## Parse-failure handling

Parser failures are sample-level, not strategy-level:

- Failed parses are excluded from the `Agreement` set.
- `Reason` mentions the parse-failure count.
- If a **strict majority** of samples fail to parse, the strategy short-circuits with `SubScore = 0` and `Reason = "majority of samples failed to parse (k/N)"`. Otherwise reporting agreement over a tiny minority is misleading.

## Cost

`N×` baseline tokens. Default `N=3` is the canonical starting point. Higher `N` tightens the estimate at proportional cost; `N=2` is the cheap variant.

## What this DOES NOT catch

Temperature jitter explores a **local** sampling neighborhood. Failure modes invisible to it:

- **Confidently wrong answers** the model gives consistently — agreement will be high.
- **Calibration drift** under domain shift.
- **Prompt-injection or instruction-hijacking** that reproduces deterministically.

For those, layer in `selfcheck` (critic-prompt) or `constraint` (schema/rule validation).

## Output

```go
out.SubScore   // mean pairwise agreement in [0,1]
out.Reason     // "N=3 samples, agreement=0.92"
out.Tokens     // summed across all N calls
out.Log        // nsample.Log{Samples, Agreement, ParseFailures}
```

`Log.Samples[i].RawText` is truncated to 256 bytes for trace compactness.
