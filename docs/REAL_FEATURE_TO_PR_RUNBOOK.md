# Real Feature-to-PR Runbook (Private)

This runbook executes the full operator path from task claim to live PR.

## Preconditions

- remote origin is configured and has a default branch
- `gh auth status` is valid
- target task exists in Beads with `autonomy` and `strict-evidence` labels

## Steps

1. Claim task (autonomous):

```bash
go run ./cmd/autonomy-worker
```

2. Validate transition to review:

```bash
go run ./cmd/beads-fsm --issue <issue-id> --to review --apply
```

3. Prepublish strict-evidence gate:

```bash
go run ./cmd/pr-gate --issue <issue-id> --prepublish
```

4. Push feature branch:

```bash
git push -u origin <feature-branch>
```

5. Publish PR and write `trace.pr_url`:

```bash
go run ./cmd/pr-publish --issue <issue-id> --title "..." --head <feature-branch> --base master --body-file <body.md>
```

`pr-publish` now also writes `trace.run_context_link` + `trace.evidence_context_link` and appends a callback dispatch report note for `pr-callbacks.v1` recipients.

6. Validate publish gate:

```bash
go run ./cmd/pr-gate --issue <issue-id>
```

7. Complete flow and close task:

```bash
go run ./cmd/beads-fsm --issue <issue-id> --to verified --apply
go run ./cmd/beads-fsm --issue <issue-id> --to done --apply
```

## Policy notes

- merge is always manual (human gate)
- model allowlist remains `glm-5`, `glm-4.7`

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
