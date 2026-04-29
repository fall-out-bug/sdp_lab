# CI Gates Map

Reference map of all SDP CI gates: purpose, owner, failure semantics, and local reproduce commands.

> **See also:** [trust-guarantees.md](./trust-guarantees.md), [quality-gates.md](./quality-gates.md), [maturity-matrix.md](./maturity-matrix.md)
> **CI workflow source:** `.github/workflows/ci.yml`

## Gate Dependency Graph

```
build-test ─────────────────────┐
snapshot-test ──────────────────┤
push-protection ────────────────┤
architect-tests ────────────────┤
contract-compat ────────────────┤
                                │
evidence-gate ───────────┐      │
scope-gate ──────────────┤      │
protocol-compliance ─────┤      │  (needs: build-test)
consistency-gate ────────┤      │
                         │      │
coverage-gate ───────────┤      │  (needs: build-test; blocking baseline delta + advisory maturity tiers)
                         │      │
policy-gate ◄────────────┘      │
   (needs: evidence,            │
    protocol-compliance,        │
    consistency)                │
auto-attestation ───────────────┤  (needs: build-test)
                                │
required-checks ◄───────────────┴──┘
   (needs: ALL gates must pass)
```

## Gate Reference

| Gate | CI Job Name | Blocks Merge | Mode | Local Reproduce |
|------|-------------|-------------|------|-----------------|
| Build & Test | `build-test` | Yes | Required | `./scripts/run_go_quality_gates.sh` |
| Snapshot Tests | `snapshot-test` | Yes | Required | `go test -tags sqlite_fts5 -run TestSnapshot ./internal/snapshot/ ./cmd/sdp/ -v` |
| Push Protection | `push-protection` | Yes (main only) | Required | N/A (commit message check, see details) |
| Architect Tests | `architect-tests` | Yes | Required | `go test -tags sqlite_fts5 ./tests/architect/... -v` |
| Contract Compat | `contract-compat` | Yes | Required | `go test -tags sqlite_fts5 ./tests/contracts/... -v` |
| Evidence Gate | `evidence-gate` | Yes | Fail-closed | `go run ./cmd/sdp-evidence validate --require-pr-url=false <file>` |
| Scope Gate | `scope-gate` | Yes | Fail-closed | `go run ./cmd/sdp-guard --ws <ws-id>` |
| Protocol Compliance | `protocol-compliance` | Yes | Fail-closed | `go run ./cmd/sdp-guard --check-contract --contract <file> --snapshot <file>` |
| Consistency Gate | `consistency-gate` | Yes | Fail-closed | `python3 scripts/check_repo_consistency.py --strict-ac --json` |
| Coverage Gate | `coverage-gate` | Yes | Blocking baseline delta; maturity-tiered minimums advisory | `go test -tags sqlite_fts5 -coverprofile=cover.out ./... && go tool cover -func=cover.out` |
| Policy Gate | `policy-gate` | Yes | Advisory (configurable) | See details (OPA eval) |
| Auto-Attestation | `auto-attestation` | Yes | Required | `go run ./internal/evidence/cmd/auto-attest --branch <branch>` |
| Required Checks | `required-checks` | Yes | Required | Verify all gate jobs pass |

## Gate Details

### build-test
- **Owner**: platform
- **Triggers**: push, PR to main/master/feature/*
- **Steps**: `go build -tags sqlite_fts5`, `go test -tags sqlite_fts5`, `golangci-lint`
- **Failure semantics**: Blocks merge. All tests must pass, zero lint errors.
- **Local reproduce**: `./scripts/run_go_quality_gates.sh`
- **Output**: stdout (pass/fail + test names); no file artifacts.

### snapshot-test
- **Owner**: platform
- **Triggers**: push, PR
- **Steps**: `go test -tags sqlite_fts5 -run TestSnapshot ./internal/snapshot/ ./cmd/sdp/`
- **Special**: `UPDATE_SNAPSHOTS=1` mode for local updates (fails in CI)
- **Failure semantics**: Blocks merge on snapshot diff. Must update snapshots intentionally.
- **Local reproduce**: `go test -tags sqlite_fts5 -run TestSnapshot ./internal/snapshot/ ./cmd/sdp/ -v`
- **Output**: Snapshot diff in test output; updated files in testdata/.

### push-protection
- **Owner**: platform
- **Triggers**: push to main/master only
- **Purpose**: Prevent direct pushes bypassing PR review
- **Allows**: Merge commits (`Merge pull request #N`), squash commits with PR reference (`(#N)`), merge-prefixed commits
- **Failure semantics**: Rejects direct push to protected branches.
- **Local reproduce**: Not applicable — this gate checks commit message patterns on push to main. Verify locally by checking commit message format.
- **Output**: Error message with rejection reason.

### architect-tests
- **Owner**: platform
- **Triggers**: push, PR
- **Steps**: `go test -tags sqlite_fts5 ./tests/architect/...`
- **Failure semantics**: Blocks merge. Architect regression detected.
- **Local reproduce**: `go test -tags sqlite_fts5 ./tests/architect/... -v`
- **Output**: stdout (test results); no file artifacts.

### contract-compat
- **Owner**: platform
- **Triggers**: push, PR
- **Steps**: `go test -tags sqlite_fts5 ./tests/contracts/...`
- **Failure semantics**: Blocks merge. Contract compatibility regression detected.
- **Local reproduce**: `go test -tags sqlite_fts5 ./tests/contracts/... -v`
- **Output**: stdout (test results); no file artifacts.

### evidence-gate
- **Owner**: platform
- **Triggers**: PR with `.sdp/evidence/*.json` files in diff
- **Steps**: Validates each evidence file via `go run ./cmd/sdp-evidence validate`; validates review verdict JSON if present
- **Required fields**: `id`, `type`, `timestamp`, `ws_id`
- **Valid types**: `plan`, `generation`, `verification`, `approval`, `decision`, `lesson`
- **Skip condition**: No evidence files in diff -> PASS
- **Failure semantics**: Blocks merge. Invalid evidence schema = hard failure.
- **Local reproduce**: `go run ./cmd/sdp-evidence validate --require-pr-url=false <file>`
- **Output schema**: `schema/evidence.schema.json`
- **Artifact path**: `.sdp/evidence/*.json`

### scope-gate
- **Owner**: platform
- **Triggers**: PR with `.sdp/checkpoints/*.json` files in diff
- **Steps**: Reads checkpoint workstream IDs, runs `go run ./cmd/sdp-guard --ws <ws-id>` per workstream
- **Skip condition**: No checkpoint files -> PASS
- **Failure semantics**: Blocks merge on scope violation. Out-of-scope files detected.
- **Local reproduce**: `go run ./cmd/sdp-guard --ws <ws-id>`
- **Output**: stdout listing scope violations; no file artifacts.

### protocol-compliance
- **Owner**: platform
- **Triggers**: PR with `.sdp/contracts/F*.json` files in diff
- **Steps**: Validates each contract has an adjacent snapshot; runs `go run ./cmd/sdp-guard --check-contract`
- **Skip condition**: No contract files changed -> PASS
- **Dependencies**: `build-test`
- **Failure semantics**: Blocks merge. Contract compliance violation.
- **Local reproduce**: `go run ./cmd/sdp-guard --check-contract --contract <file> --snapshot <file>`
- **Output**: stdout with compliance report; no file artifacts.

### consistency-gate
- **Owner**: platform
- **Triggers**: Every PR
- **Steps**: `python3 scripts/check_repo_consistency.py --strict-ac --json` + `go run ./cmd/sdp-protocol-check --format json` + `go run ./cmd/sdp-doc-sync --mode check --format json` + `go run ./cmd/sdp-protocol-check --lint-skills --format json`
- **Never skipped**
- **Failure semantics**: Blocks merge on repo consistency failure. Protocol check and doc-sync are non-blocking advisory. Skill-lint is non-blocking in advisory rollout.
- **Local reproduce**: `python3 scripts/check_repo_consistency.py --strict-ac --json`
- **Output schema**: JSON findings file
- **Artifact path**: `.sdp/findings/*.json` (uploaded as CI artifact)

### coverage-gate
- **Owner**: platform
- **Triggers**: push, PR
- **Steps**: `go test -tags sqlite_fts5 -coverprofile=cov.out ./...` + two-phase check:
  1. **Baseline delta**: Compare repo-total coverage against `.sdp/metrics/coverage.txt` with -2pp threshold (blocking).
  2. **Maturity-tiered minimums**: Check per-package coverage against maturity-appropriate targets (advisory rollout; see [Coverage Tier Policy](#coverage-tier-policy)).
- **Dependencies**: `build-test`
- **Failure semantics**: Blocking for total coverage drops more than 2pp below baseline. Maturity-tier minimum failures are reported but non-blocking during the advisory rollout. Experimental packages are exempt.
- **Local reproduce**: `go test -tags sqlite_fts5 -coverprofile=cover.out ./... && go tool cover -func=cover.out | grep total`
- **Output**: Coverage percentage in stdout; per-package tier results; `cov.out` locally.
- **Baseline**: `.sdp/metrics/coverage.txt` (auto-updated on push to main)

#### Coverage Tier Policy

Coverage expectations are tiered by component maturity. See [maturity-matrix.md](./maturity-matrix.md) for the canonical maturity classification.

| Tier | Maturity | Target | Enforced | Denominator |
|------|----------|--------|----------|-------------|
| Happy-path | GA + on happy-path surface | >= 80% | Advisory rollout | Per-package line coverage for packages implementing the canonical happy-path |
| GA | GA (not on happy-path) | >= 60% | Advisory rollout | Per-package line coverage |
| Beta | Beta | >= 50% | **Advisory** (non-blocking) | Per-package line coverage |
| Experimental | Experimental | None | **Exempt** | N/A |

**Happy-path packages** (>= 80% target): `internal/scout`, `internal/metrics`, `internal/index`, `internal/bootstrap`, `internal/control`, `internal/orchestrate`, `internal/cli`, `internal/manifest`, `internal/evidence`, `internal/guard`, `internal/discovery`, `internal/build`.

**GA (non happy-path) packages** (>= 60% target): All other GA-maturity packages in `internal/` and `cmd/`.

**Beta packages** (>= 50% advisory): `internal/tower`, `internal/dispatch`, `internal/a2a`, `internal/monitor`, `internal/profile`, `internal/policy`, `internal/augmentation`, `internal/mcp`, `internal/evals`, `internal/deploy`, `internal/trace`, `internal/sessionaudit`, and Beta-maturity binaries.

**Experimental packages** (exempt): `internal/agentloop`, `internal/modelgateway`, `internal/inference`, `internal/llmclient`, `internal/localmodel`, `internal/memory`, `internal/mutation`, `internal/finetune`, `internal/planner`, `internal/authz`, `internal/stream`, `internal/secretscan`, `internal/provenance`, `internal/flaky`, `internal/glob`.

**Measurement**: `go test -tags sqlite_fts5 -coverprofile=cov.out ./...` + `go tool cover -func=cov.out` (per-package line coverage).

### policy-gate
- **Owner**: platform
- **Triggers**: Every PR
- **Dependencies**: `evidence-gate`, `protocol-compliance`, `consistency-gate`
- **Steps**: Collects policy input from PR diff + gate results, evaluates OPA policies in `.sdp/policies/`
- **Enforcement mode**: Configurable via `SDP_POLICY_ENFORCEMENT_MODE` env var. Default: `advisory` (denials logged but non-blocking). Set to `blocking` to enforce denials.
- **Failure semantics**: Blocks merge only when `SDP_POLICY_ENFORCEMENT_MODE=blocking` and denials exist. Otherwise advisory (warnings logged).
- **Local reproduce**: `opa eval --data .sdp/policies/ --input /tmp/policy-input.json 'data.sdp.policies.effective_deny'`
- **Output**: Policy evaluation results in CI log.

### auto-attestation
- **Owner**: platform
- **Triggers**: Every PR
- **Dependencies**: `build-test`
- **Steps**: Runs `go run ./internal/evidence/cmd/auto-attest` to generate attestation, signs with Sigstore keyless
- **Failure semantics**: Blocks merge. Attestation generation failure = hard failure.
- **Local reproduce**: `go run ./internal/evidence/cmd/auto-attest --branch <branch> --base-branch main --output .sdp/attestations/ci-auto.json`
- **Output schema**: Attestation JSON + Sigstore bundle
- **Artifact path**: `.sdp/attestations/ci-auto.json`, `.sdp/attestations/ci-auto.bundle` (uploaded as CI artifact, 90-day retention)

### required-checks
- **Owner**: platform
- **Triggers**: Every PR (after ALL other gates)
- **Dependencies**: All 12 gates listed above
- **Purpose**: Final merge gate -- ALL must pass (success or skipped)
- **Failure semantics**: Blocks merge until all dependencies are green.
- **Local reproduce**: Verify each gate individually using local reproduce commands above.
- **Output**: Per-gate pass/fail summary in CI log.

## Policy Configuration

Policies are defined as OPA/Rego files in `.sdp/policies/`. The default enforcement mode is `advisory`.

To enforce blocking policy:
```bash
# In CI environment
export SDP_POLICY_ENFORCEMENT_MODE=blocking
```

## Disabling Gates

See [enterprise-pilot-rollback.md](./enterprise-pilot-rollback.md) for full disable procedures.
