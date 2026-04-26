# CI Gates Map

Reference map of all SDP CI gates: purpose, owner, failure semantics, and local reproduce commands.

> **See also:** [trust-guarantees.md](./trust-guarantees.md), [quality-gates.md](./quality-gates.md), [maturity-matrix.md](./maturity-matrix.md)

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

| Gate | CI Job Name | Blocks Merge | Default Mode | Timeout | Local Reproduce |
|------|-------------|-------------|-------------|---------|-----------------|
| Build & Test | `build-test` | Yes | Required | 10 min | `./scripts/run_go_quality_gates.sh` |
| Snapshot Tests | `snapshot-test` | Yes | Required | 15 min | `go test -run TestSnapshot ./...` |
| Push Protection | `push-protection` | Yes (main only) | Required | 1 min | `sdp ready` (checks branch protection) |
| Evidence Gate | `evidence-gate` | Yes | Fail-closed | 5 min | `sdp-evidence verify .sdp/evidence/` |
| Scope Gate | `scope-gate` | Yes | Fail-closed | 5 min | `sdp guard check` |
| Consistency Gate | `consistency-gate` | Yes | Fail-closed | 5 min | `sdp verify` |
| Coverage Gate | `coverage-gate` | Yes | Fail-open (pilot) | 10 min | `go test -coverprofile=cover.out ./... && go tool cover -func=cover.out` |
| Policy Gate | `policy-gate` | Yes | Fail-closed | 5 min | `sdp gate status` |
| Secret Scan | `secretscan` | Yes | Fail-closed | 2 min | `git log -p \| grep -iE '(password|secret|token|api.key)' \| head` |
| Final Gate | `ci-pass` | Yes | Required | 1 min | `./scripts/run_go_quality_gates.sh` (all gates) |

## Gate Details

### build-test
- **Owner**: platform
- **Triggers**: push, PR to main/master/feature/*
- **Steps**: `go build`, `go test`, `golangci-lint`
- **Tags**: `sqlite_fts5`
- **Failure semantics**: Blocks merge. All tests must pass, zero vet/lint errors.
- **Local reproduce**: `./scripts/run_go_quality_gates.sh`
- **Output**: stdout (pass/fail + test names); no file artifacts.

### snapshot-test
- **Owner**: platform
- **Triggers**: push, PR
- **Steps**: `go test -run TestSnapshot`
- **Special**: `UPDATE_SNAPSHOTS=1` mode for local updates (fails in CI)
- **Failure semantics**: Blocks merge on snapshot diff. Must update snapshots intentionally.
- **Local reproduce**: `go test -run TestSnapshot ./...`
- **Output**: Snapshot diff in test output; updated files in `testdata/`.

### push-protection
- **Owner**: platform
- **Triggers**: push to main/master only
- **Purpose**: Prevent direct pushes bypassing PR review
- **Allows**: Merge commits, squash commits with PR reference
- **Failure semantics**: Rejects direct push to protected branches.
- **Local reproduce**: `sdp ready` (verifies branch protection is active)
- **Output**: Git hook message with rejection reason.

### evidence-gate
- **Owner**: platform
- **Triggers**: PR with `.sdp/evidence/*.json` files in diff
- **Steps**: Validates each evidence file against `schema/evidence.schema.json`
- **Required fields**: `id`, `type`, `timestamp`, `ws_id`
- **Valid types**: `plan`, `generation`, `verification`, `approval`, `decision`, `lesson`
- **Skip condition**: No evidence files in diff -> PASS
- **Failure semantics**: Blocks merge. Invalid evidence schema = hard failure.
- **Local reproduce**: `sdp-evidence verify .sdp/evidence/`
- **Output schema**: `schema/evidence.schema.json`
- **Artifact path**: `.sdp/evidence/*.json`

### scope-gate
- **Owner**: platform
- **Triggers**: PR with `.sdp/checkpoints/*.json` files in diff
- **Steps**: Validates changed files match declared workstream scope
- **Skip condition**: No checkpoint files -> PASS
- **Failure semantics**: Blocks merge on scope violation. Out-of-scope files detected.
- **Local reproduce**: `sdp guard check`
- **Output**: stdout listing out-of-scope files; no file artifacts.

### consistency-gate
- **Owner**: platform
- **Triggers**: Every PR
- **Steps**: `sdp verify` -- checks guard-rules, schema conformance, file hygiene
- **Never skipped**
- **Failure semantics**: Blocks merge. Schema or rule conformance violation.
- **Local reproduce**: `sdp verify`
- **Output**: stdout with violation list; no file artifacts.

### coverage-gate
- **Owner**: platform
- **Triggers**: Every PR
- **Steps**: `go test -coverprofile`, `sdp coverage check --minimum=60`
- **Configurable**: Minimum threshold in `.sdp/config.yml`
- **Failure semantics**: Warn-only during pilot (fail-open). Will transition to fail-closed.
- **Local reproduce**: `go test -coverprofile=cover.out ./... && go tool cover -func=cover.out`
- **Output schema**: Go cover profile format
- **Artifact path**: `cover.out` (local), CI artifact (CI)

### policy-gate
- **Owner**: platform
- **Triggers**: Every PR (after evidence, consistency, coverage gates)
- **Dependencies**: `evidence-gate`, `consistency-gate`, `coverage-gate`
- **Steps**: Aggregates all gate results, runs auto-attestation if configured
- **Produces**: Policy summary JSON with gate results
- **Failure semantics**: Blocks merge if any dependency gate failed.
- **Local reproduce**: `sdp gate status`
- **Output schema**: Policy summary JSON
- **Artifact path**: `.sdp/gates/policy-summary.json`

### secretscan
- **Owner**: platform
- **Triggers**: Every PR
- **Steps**: Pattern-based scan for leaked credentials, tokens, and API keys
- **Failure semantics**: Blocks merge. Hard gate -- always block on detected secrets.
- **Local reproduce**: `git log -p | grep -iE '(password|secret|token|api.key)' | head`
- **Output**: stdout with matched lines; no file artifacts.

### ci-pass
- **Owner**: platform
- **Triggers**: Every PR (after ALL other gates)
- **Dependencies**: All gates listed above
- **Purpose**: Final merge gate -- ALL must pass
- **Failure semantics**: Blocks merge until all gates green.
- **Local reproduce**: `./scripts/run_go_quality_gates.sh` (runs all checks)
- **Output**: Aggregate pass/fail; no separate artifacts.

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
