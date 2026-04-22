# Workstream Verdicts

Workstream verdicts record completion and acceptance-criteria evidence for each workstream.

## Location

- **Runtime:** `.sdp/ws-verdicts/{ws-id}.json` (gitignored; local/CI)
- **Committed (optional):** `docs/ws-verdicts/{ws-id}.json` for shared or audit use

## Schema

Each verdict file must conform to `schema/ws-verdict.schema.json`. Required fields:

- `ws_id` — workstream ID (e.g. `00-015-01`)
- `feature_id` — feature ID (e.g. `F001`)
- `verdict` — `PASS` | `FAIL` | `PARTIAL`
- `quality_gates` — object with `tests_pass`, `lint_clean` (boolean)
- `existing_work_summary` — one-line summary of pre-existing code found before implementation
- `ac_evidence` — array of `{ "ac": "description", "met": true|false, "evidence": "file or command" }` per acceptance criterion

## When to Create

- When a workstream is marked **Done** in INDEX.md
- When running `/review` or `@build` with AC coverage checks
- After completing all ACs for a workstream

## Example

```json
{
  "ws_id": "00-015-01",
  "feature_id": "F015",
  "verdict": "PASS",
  "quality_gates": { "tests_pass": true, "lint_clean": true },
  "existing_work_summary": "Pre-existing hooks in .cursor/hooks.json",
  "ac_evidence": [
    { "ac": "Hook fires when Cursor agent finishes", "met": true, "evidence": "scripts/oneshot-stop-gate.sh + .cursor/hooks.json" }
  ]
}
```
