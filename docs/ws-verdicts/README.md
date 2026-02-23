# Workstream Verdicts

Workstream verdicts record completion and acceptance-criteria evidence for each workstream.

## Location

- **Runtime:** `.sdp/ws-verdicts/{ws-id}.json` (gitignored; local/CI)
- **Committed (optional):** `docs/ws-verdicts/{ws-id}.json` for shared or audit use

## Schema

Each verdict file should contain at least:

- `ws_id` — workstream ID (e.g. `00-015-01`)
- `status` — `done` | `in_progress` | `blocked`
- `completed_at` — ISO8601 when marked done (optional)
- `ac_evidence` — array of `{ "ac": "description", "evidence": "file or command" }` per acceptance criterion

## When to Create

- When a workstream is marked **Done** in INDEX.md
- When running `/review` or `@build` with AC coverage checks
- After completing all ACs for a workstream

## Example

```json
{
  "ws_id": "00-015-01",
  "status": "done",
  "ac_evidence": [
    { "ac": "Hook fires when Cursor agent finishes", "evidence": "scripts/oneshot-stop-gate.sh + .cursor/hooks.json" }
  ]
}
```
