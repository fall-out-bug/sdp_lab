# F144 Confidence Replay Report

Generated automatically by `internal/inference/confidence/replay` against the fixture corpus under `internal/inference/confidence/testdata/`.

Acceptance gates (from F144 design §9):
- Adversarial rejection rate ≥ 0.80
- Correct false-FAIL rate ≤ 0.02

## ws-verdict

| Category | N | OK | UNSURE | FAIL | Rejection rate | Unsure rate | p50 ms | p95 ms | Mean tokens |
|---|---|---|---|---|---|---|---|---|---|
| correct | 2 | 2 | 0 | 0 | 0.00 | 0.00 | 0 | 0 | 0 |
| edge | 2 | 2 | 0 | 0 | 0.00 | 0.00 | 0 | 0 | 0 |
| adversarial | 4 | 0 | 0 | 4 | 1.00 | 0.00 | 0 | 0 | 0 |

Verdict: **PASS** — adversarial rejection 1.00 ≥ 0.80, correct false-FAIL 0.00 ≤ 0.02

