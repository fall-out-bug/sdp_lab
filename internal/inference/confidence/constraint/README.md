# `constraint` — schema + invariant validation strategy

Cheapest of the F144 strategies: zero LLM calls, pure-Go validation. Runs first in most call-site stacks so other strategies don't waste tokens on already-broken output.

## Signal

- **Schema fail → hard-fail.** Forces `Status = FAIL` regardless of composed score, because semantic confidence has no meaning on malformed format.
- **Invariant fail → linear penalty.** `SubScore = 1 − failed / total` over the invariant set. Invariant failures **never** hard-fail — they're "answer is well-formed but misses a rule"; the composer decides whether the score is bad enough to block.

## Construction

```go
s, _ := constraint.New[MyAnswer](constraint.Options[MyAnswer]{
    SchemaValidator: func(raw string) error {
        // e.g. validate raw JSON against schema/wsverdict.json
        return jsonschema.Validate(rawSchema, raw)
    },
    Invariants: []constraint.Invariant[MyAnswer]{
        {Name: "verdict-known", Check: func(a MyAnswer) (bool, string) {
            switch a.Verdict {
            case "passed", "failed", "blocked":
                return true, ""
            default:
                return false, "unknown verdict " + a.Verdict
            }
        }},
        {Name: "score-in-range", Check: func(a MyAnswer) (bool, string) {
            if a.Score < 0 || a.Score > 1 {
                return false, fmt.Sprintf("got %v", a.Score)
            }
            return true, ""
        }},
    },
})
```

`SchemaValidator` is optional. `Invariants` is optional. Constructor rejects nil `Check` callbacks (caller bug — fail fast).

## Output shape

```go
out.SubScore   // 1.0, 0.0 (schema fail), or 1 - failed/total
out.HardFail   // true iff schema validator returned non-nil
out.Reason     // "schema valid" | "schema invalid: ..." | "2/5 invariants failed: name(detail); ..."
out.Tokens     // always zero (no LLM calls)
out.Log        // constraint.Log{SchemaError, Failures []Failure}
```

## When to use

Always, if you have either a JSON-Schema for the response shape or any structural rule you can express as a predicate. The cost is negligible and the signal — especially `HardFail` — protects downstream strategies from wasting LLM budget on broken output.

## Sequencing in a Checker

When mixed with `selfcheck` / `nsample`, place `constraint` first in the strategy slice. Hard-fail short-circuits the composed verdict to `FAIL` regardless of what the LLM-based strategies report — but they still run for trace evidence. The `Checker.Check` order is preserved as listed.
