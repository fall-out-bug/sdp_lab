# CI to Local Bridge Runbook

## Purpose

Connect GitHub-hosted CI findings with local Beads-driven autonomous agents.

## Architecture

```text
GitHub CI (Sensor Layer)
  ├─ Runs: sdp-protocol-check, sdp-doc-sync --mode check
  ├─ Publishes: findings artifacts + optional GitHub issues
  └─ Emits: normalized findings payloads (schema v1)

Local Sync Daemon (Transport Layer)
  ├─ Pulls latest findings from GitHub
  ├─ Normalizes + hashes findings
  ├─ Deduplicates idempotently
  └─ Creates/updates local Beads issues

Local Agents (Actuator Layer)
  ├─ Analysis Agent: classifies and enriches findings
  ├─ Improvement Agent: implements fixes from Beads ready queue
  └─ Documentation Agent: updates changelog + docs consistency

Back-sync
  └─ Local Beads state pushes progress to GitHub labels/comments
```

## Data Contracts

- CI findings must follow `schema/findings/*` definitions (F077-01).
- Each finding must include stable key fields: `source`, `path`, `rule_id`, `message`, `run_ref`.
- Beads task metadata must include source GitHub run identifier and finding hash.

## Back-sync Mapping Policy

- Canonical Beads-to-GitHub mapping is stored in `external_ref` using `gh:owner/repo#<issue_number>`.
- Parser accepts `owner/repo#<issue_number>` directly and `gh-<issue_number>` only when a default repository is configured.
- Deterministic Beads status to GitHub label mapping:
  - `open` -> `sdp/open`
  - `in_progress` -> `sdp/in-progress`
  - `blocked` -> `sdp/blocked`
  - `deferred` -> `sdp/deferred`
  - `closed` -> `sdp/done`
- Every back-sync operation posts a deterministic status comment:
  - `SDP Beads issue <id> status synchronized to <status>.`

## Operating Modes

- `polling`: daemon checks GitHub every N minutes.
- `oneshot`: daemon runs once (manual or cron-triggered).

## Execution Order (F077 Implementation)

1. **00-077-01 Findings Schema**
   - Finalize JSON schemas and fixtures.
   - Wire schema validation into CI report generation.

2. **00-077-02 Sync Daemon**
   - Implement GitHub artifact + issue source adapters.
   - Map findings into Beads issue create/update payloads.

3. **00-077-03 Dedup/Idempotency**
   - Add stable finding hash strategy.
   - Guarantee reprocessing updates existing Beads tasks.

4. **00-077-04 Status Back-sync**
   - Map Beads states to GitHub labels/comments.
   - Add retry and outage recovery policy.

## Failure Handling

- If GitHub API is unavailable, store pending sync tasks locally and retry with backoff.
- If Beads write fails, keep finding in retry queue; never drop findings silently.
- If back-sync fails, mark issue with `backsync-pending` label for next attempt.

## Outage Recovery Procedure

1. Verify GitHub token scope and API availability (`gh auth status`, API health).
2. Re-run one-shot sync against the affected scope first, then re-enable polling.
3. Confirm retry audit entries include prior failures and final recovery success.
4. Remove `backsync-pending` only after labels/comments are synchronized.
5. If retries continue to fail, keep the Beads issue open and attach the latest audit error in notes.

## Security and Access

- Use least-privilege GitHub token for read artifacts/issues and write comments/labels only.
- Keep local Beads DB private to local environment; never expose it directly to CI.
- Redact sensitive path/content details before publishing GitHub comments.

## Verification Checklist

- CI produced findings artifact for a failing run.
- Sync daemon imported finding into Beads exactly once.
- Improvement agent closed Beads issue after fix.
- Back-sync updated the linked GitHub issue/check.
