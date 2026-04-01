# Real Feature-to-PR Runbook (Private)

This runbook executes the canonical operator path from ready `beads issue` to clean `PR` with early `draft PR`, typed findings, and `QA/UAT`.

## Preconditions

- remote origin is configured and has a default branch
- `gh auth status` is valid
- target task exists in Beads with `autonomy` and `strict-evidence` labels
- the task links back to one `feature` and one `workstream`

## Steps

1. Claim task (autonomous):

```bash
go run ./cmd/autonomy-worker
```

2. Ensure feature branch exists and open early `draft PR`:

```bash
git push -u origin <feature-branch>
gh pr create --draft --base main --title "FXXX: short-name"
```

If the `feature` already has an active `draft PR`, reuse it. Do not wait until the end of the feature to open the PR.

3. Validate transition to review:

```bash
go run ./cmd/beads-fsm --issue <issue-id> --to review --apply
```

4. Prepublish strict-evidence gate:

```bash
go run ./cmd/pr-gate --issue <issue-id> --prepublish
```

5. Publish or update PR evidence and write `trace.pr_url`:

```bash
go run ./cmd/pr-publish --issue <issue-id> --title "..." --head <feature-branch> --base main --body-file <body.md>
```

`pr-publish` now also writes `trace.run_context_link` + `trace.evidence_context_link` and appends a callback dispatch report note for `pr-callbacks.v1` recipients.

6. Validate publish gate:

```bash
go run ./cmd/pr-gate --issue <issue-id>
```

7. Convert review, CI, or `drift` findings into typed `beads issue` entries.

Each finding must capture:

- `source = review | ci | drift | qa`
- linked `feature`
- linked `workstream`
- `blocking = true|false`
- `PR` link or artifact reference

Use the canonical contract in [protocol/BEADS_FINDINGS_CONTRACT.md](protocol/BEADS_FINDINGS_CONTRACT.md).

Blocking findings return to the ready queue and must be closed before the `PR` is considered clean.

8. Run `QA/UAT` after engineering gates are clean.

`QA/UAT` returns one of two verdicts:

- `qa:pass` with `UAT evidence`
- `qa:fail` with new blocking `beads issue`

9. Complete flow and close task:

```bash
go run ./cmd/beads-fsm --issue <issue-id> --to verified --apply
go run ./cmd/beads-fsm --issue <issue-id> --to done --apply
```

## Policy notes

- merge is always manual (human gate)
- model allowlist remains `glm-5`, `glm-4.7`
- `PR` is opened early and stays active throughout execution
- merge-ready state requires clean engineering gates, complete `trace`, recorded `drift` verdict, and `QA/UAT` pass

## Callback retry semantics and policy controls

Before `pr-publish`, verify callback routing policy aligns with the default contract (`callback-routing-reliability/v1`):

- ack timeout `30s`
- deterministic retry delays `5s, 15s, 30s, 60s, 120s, 240s, 420s` (total budget `<= 15m`)
- dead-letter fallback enabled with reason `retry-window-exhausted`
- required recipient fallback order `issue-owner -> orchestrator-audit`

When policy override is required (incident or rollout), record the override and justification in issue notes before publish:

- `callback.route.mode`: `required-first` (default) or `fanout-all`
- `callback.retry.profile`: `standard-15m` (default) or `aggressive-5m`
- `callback.notify.watchers`: `enabled` (default) or `disabled`
- `callback.escalate.on.deadletter`: `enabled` (default) or `disabled`

If callback delivery exhausts retry budget, append an explicit escalation note with:

- issue id and run id
- last callback status/transport error
- dead-letter reason (`retry-window-exhausted`)
- follow-up action owner

## OneShot evidence packaging (PR body minimum)

When shipping `workstream:oneshot-swarm-orchestrator`, include a short artifact table in PR body:

- run artifact path: `.sdp/runs/<issue-id>.json`
- evidence artifact path: `.sdp/evidence/<issue-id>.json`
- verification fields confirmed:
  - `verification.go_test_passed`
  - `verification.oneshot.report`
  - `verification.oneshot.recovery_plan` (if verification failed)
- run summary fields confirmed:
  - `oneshot_verification`
  - `oneshot_recovery` (if recovery required)
- issue note proof: JSON payload with `kind=oneshot_verify`
- gate outputs captured: `go test ./cmd/swarm-worker`, `go test ./internal/oneshot`, `go test ./...`, and `pr-gate` output
- publish link: `trace.pr_url` and matching PR URL in Beads notes
