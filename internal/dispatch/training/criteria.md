# SDP Complexity Classifier — Evaluation Criteria

Issue: `sdplab-e447` (Day 6 challenge — F133 fine-tune dataset).

## Task

Three-axis classifier for SDP dispatch routing.

| Axis        | Values                                            |
|-------------|---------------------------------------------------|
| complexity  | `low` / `medium` / `high`                         |
| task_type   | `feature` / `bugfix` / `refactor` / `test` / `docs` |
| risk        | `low` / `high`                                    |

Input: `Title: ...\nGoal: ...` (≤ 4000 chars).
Output: single-line JSON, no prose.

## Dataset

- Source: 209 ws-frontmatter rows + 11 beads issues = 220 deduped real samples
- Split: 80/20 → 176 train, 44 eval (seed 42, deterministic)
- 100% real (no synthetic generation needed at this size)

## Baseline criteria (vs `qwen2.5:3b` raw)

Recorded in `baseline.json`. Tracked metrics:

| Metric                 | Definition                                       | Target uplift after FT |
|------------------------|--------------------------------------------------|-------------------------|
| `parse_ok`             | response contains a parseable label JSON         | ≥ 90 % (baseline likely 50-70 %) |
| `complexity_accuracy`  | predicted complexity == expected                 | ≥ +15 pp                 |
| `task_type_accuracy`   | predicted task_type == expected                  | ≥ +15 pp                 |
| `risk_accuracy`        | predicted risk == expected                       | ≥ +10 pp                 |
| `all_match_accuracy`   | all three axes match exactly                     | ≥ +20 pp                 |

## "Better" definition

Fine-tune is judged successful if it satisfies **all** of:

1. `parse_ok` reaches **100 %** on eval (no malformed JSON, no prose)
2. `all_match_accuracy` is **at least +20 pp** vs baseline
3. No regression on any individual axis (each accuracy ≥ baseline)
4. Inference latency on Ollama M-series stays **< 1 s / sample** (≈ feasible for in-loop F133 routing)

## Out-of-scope (not measured here)

- Generalisation to non-SDP tasks (the model is intentionally domain-specific)
- Calibration / probability estimates (output is deterministic enum)
- Long-context inputs (capped at 4000 chars in dataset)
