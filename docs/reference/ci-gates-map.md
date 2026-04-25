# CI Gates Map

Reference map of all SDP CI gates, their purpose, dependencies, and configuration.

## Gate Dependency Graph

```
build-test ──────────────────────┐
snapshot-test ───────────────────┤
push-protection ─────────────────┤
                                 │
evidence-gate ───────────┐       │
scope-gate ──────────────┤       │
consistency-gate ────────┼──┐    │
                         │  │    │
coverage-gate ───────────┤  │    │
                         │  │    │
policy-gate ◄────────────┘  │    │
   (needs: evidence,        │    │
    consistency, coverage)  │    │
                            │    │
ci-pass ◄───────────────────┴────┘
   (needs: ALL gates must pass)
```

## Gate Reference

| Gate | CI Job Name | Blocks Merge | Default Mode | Timeout |
|------|-------------|-------------|-------------|---------|
| Build & Test | `build-test` | Yes | Required | 10 min |
| Snapshot Tests | `snapshot-test` | Yes | Required | 15 min |
| Push Protection | `push-protection` | Yes (main only) | Required | 1 min |
| Evidence Gate | `evidence-gate` | Yes | Fail-closed | 5 min |
| Scope Gate | `scope-gate` | Yes | Fail-closed | 5 min |
| Consistency Gate | `consistency-gate` | Yes | Fail-closed | 5 min |
| Coverage Gate | `coverage-gate` | Yes | Fail-open (pilot) | 10 min |
| Policy Gate | `policy-gate` | Yes | Fail-closed | 5 min |
| Final Gate | `ci-pass` | Yes | Required | 1 min |

## Gate Details

### build-test
- **Triggers**: push, PR to main/master/feature/*
- **Steps**: `go build`, `go test`, `golangci-lint`
- **Tags**: `sqlite_fts5`

### snapshot-test
- **Triggers**: push, PR
- **Steps**: `go test -run TestSnapshot`
- **Special**: `UPDATE_SNAPSHOTS=1` mode for local updates (fails in CI)

### push-protection
- **Triggers**: push to main/master only
- **Purpose**: Prevent direct pushes bypassing PR review
- **Allows**: Merge commits, squash commits with PR reference

### evidence-gate
- **Triggers**: PR with `.sdp/evidence/*.json` files in diff
- **Steps**: Validates each evidence file against `schema/evidence.schema.json`
- **Required fields**: `id`, `type`, `timestamp`, `ws_id`
- **Valid types**: `plan`, `generation`, `verification`, `approval`, `decision`, `lesson`
- **Skip condition**: No evidence files in diff → PASS

### scope-gate
- **Triggers**: PR with `.sdp/checkpoints/*.json` files in diff
- **Steps**: Validates changed files match declared workstream scope
- **Skip condition**: No checkpoint files → PASS

### consistency-gate
- **Triggers**: Every PR
- **Steps**: `sdp verify` — checks guard-rules, schema conformance, file hygiene
- **Never skipped**

### coverage-gate
- **Triggers**: Every PR
- **Steps**: `go test -coverprofile`, `sdp coverage check --minimum=60`
- **Configurable**: Minimum threshold in `.sdp/config.yml`

### policy-gate
- **Triggers**: Every PR (after evidence, consistency, coverage gates)
- **Dependencies**: `evidence-gate`, `consistency-gate`, `coverage-gate`
- **Steps**: Aggregates all gate results, runs auto-attestation if configured
- **Produces**: Policy summary JSON with gate results

### ci-pass
- **Triggers**: Every PR (after ALL other gates)
- **Dependencies**: All gates listed above
- **Purpose**: Final merge gate — ALL must pass

## Configuration

### .sdp/guard-rules.yml

```yaml
gates:
  evidence:
    enabled: true
    mode: fail-closed
    schema: schema/evidence.schema.json
  scope:
    enabled: true
    mode: fail-closed
  consistency:
    enabled: true
    mode: fail-closed
  coverage:
    enabled: true
    mode: fail-open  # warn during pilot
    minimum: 60
  policy:
    enabled: true
    mode: fail-closed
    auto_attest: false
  secretscan:
    enabled: true
    mode: fail-closed  # hard gate — always block on secrets
```

### .sdp/config.yml

```yaml
project:
  name: my-project
  go_version: "1.26"

gates:
  enabled: true

runtime:
  mode: ci-only  # or "contracted"

coverage:
  minimum: 60

evidence:
  log_path: .sdp/log/events.jsonl
  tracked: true
```

## Disabling Gates

See [enterprise-pilot-rollback.md](./enterprise-pilot-rollback.md) for full disable procedures.

Quick disable:
```bash
sdp config set gates.enabled false          # all gates
sdp config set gates.coverage.enabled false  # specific gate
```
