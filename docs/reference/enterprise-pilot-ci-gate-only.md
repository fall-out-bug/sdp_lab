# CI-Gate-Only Enterprise Pilot Quickstart

> **Goal**: Integrate SDP as CI governance in under 30 minutes — no runtime orchestration changes required.

## Overview

SDP ships a suite of CI gates that enforce evidence-backed development at merge time. This pilot path activates those gates without touching your agent stack, orchestration scripts, or production runtime. It is the lowest-risk adoption path for teams evaluating SDP.

**What you get:**
- Evidence validation on every PR
- Protocol compliance checking (guard-rules, schema, file hygiene)
- Scope gate (workstream drift detection)
- Coverage enforcement
- Secret scanning (hard gate on deploy)

**What you don't need:**
- No changes to how your agents run
- No migration from existing CI pipelines
- No runtime schema enforcement
- No orchestration bus or event streaming

## Prerequisites (5 min)

| Requirement | Version | Check |
|-------------|---------|-------|
| GitHub repository | any | `git remote -v` |
| GitHub Actions | enabled | `.github/workflows/` writable |
| Go | 1.26+ | `go version` |
| `sdp` CLI | latest | `sdp version` |

## Step-by-step (25 min)

### Step 1: Install SDP CLI (2 min)

```bash
go install ./cmd/sdp@latest
sdp version
```

### Step 2: Initialize SDP in your repo (3 min)

```bash
cd /path/to/your/repo
sdp init
```

This creates:
- `.sdp/config.yml` — project configuration
- `.sdp/log/events.jsonl` — evidence event log (tracked in git)
- `.sdp/guard-rules.yml` — quality gate rules (tracked in git)

Verify:
```bash
git status
# Should show: .sdp/config.yml, .sdp/log/events.jsonl, .sdp/guard-rules.yml
```

### Step 3: Add CI workflow gates (10 min)

Copy the SDP reusable CI gates into your workflow. The gates are designed to slot into any existing `ci.yml`:

```yaml
# .github/workflows/ci.yml — add these jobs

jobs:
  # Your existing build/test/lint jobs go here
  # ...

  # SDP Gates (add after your existing jobs)
  evidence-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Validate evidence files
        run: |
          EVIDENCE_FILES=$(git diff --name-only origin/${{ github.base_ref }}...HEAD | grep '^\.sdp/evidence/.*\.json$' || true)
          if [ -z "$EVIDENCE_FILES" ]; then
            echo "No evidence files changed — skipping validation"
            exit 0
          fi
          for f in $EVIDENCE_FILES; do
            go run ./cmd/sdp-evidence validate --require-pr-url=false "$f"
          done

  scope-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Check scope compliance
        run: |
          CHANGED=$(git diff --name-only origin/${{ github.base_ref }}...HEAD)
          CHECKPOINTS=$(echo "$CHANGED" | grep '^\.sdp/checkpoints/.*\.json$' || true)
          if [ -z "$CHECKPOINTS" ]; then
            echo "No checkpoint files — scope gate not applicable"
            exit 0
          fi
          go run ./cmd/sdp scope check

  consistency-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Protocol consistency
        run: go run ./cmd/sdp verify

  coverage-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Coverage enforcement
        run: |
          go test -tags "sqlite_fts5" -coverprofile=coverage.out ./...
          go run ./cmd/sdp coverage check --minimum=60

  policy-gate:
    runs-on: ubuntu-latest
    needs: [evidence-gate, consistency-gate, coverage-gate]
    steps:
      - uses: actions/checkout@v6
      - name: Policy gate
        run: go run ./cmd/sdp policy check

  # Final gate: all must pass
  ci-pass:
    runs-on: ubuntu-latest
    needs: [build-test, evidence-gate, scope-gate, consistency-gate, policy-gate, coverage-gate]
    if: always()
    steps:
      - name: Check all gates passed
        run: |
          for gate in "build-test" "evidence-gate" "scope-gate" "consistency-gate" "policy-gate" "coverage-gate"; do
            result="${{ needs.${gate}.result }}"
            if [ "$result" != "success" ]; then
              echo "FAIL: $gate ($result)"
              exit 1
            fi
            echo "PASS: $gate"
          done
          echo "All CI gates passed"
```

### Step 4: Record your first evidence event (5 min)

Create an evidence file for your next PR to see the gates in action:

```bash
# Record a plan evidence event
sdp skill record --skill build --type plan \
  --ws-id 00-001-01 \
  --data '{"scope_files":["cmd/app/main.go"],"action":"add health endpoint","feature_id":"F001"}'
```

This writes to `.sdp/log/events.jsonl`:
```json
{
  "id": "a1b2c3d4-...",
  "type": "plan",
  "timestamp": "2026-04-25T09:00:00Z",
  "ws_id": "00-001-01",
  "commit_sha": "abc1234",
  "prev_hash": "sha256-of-previous-event",
  "data": {"scope_files":["cmd/app/main.go"],"action":"add health endpoint","feature_id":"F001"}
}
```

### Step 5: Open a PR and verify gates (5 min)

```bash
git checkout -b feature/pilot-test
git add .sdp/
git commit -m "feat: initialize SDP CI gates"
git push -u origin feature/pilot-test
gh pr create --title "SDP CI gate pilot" --body "First PR with SDP CI gates active"
```

**Expected PR checks:**
- `build-test` — your existing tests
- `evidence-gate` — validates any `.sdp/evidence/*.json` files
- `scope-gate` — checks checkpoint scope alignment
- `consistency-gate` — protocol consistency
- `policy-gate` — runs after evidence + consistency + coverage
- `coverage-gate` — enforces minimum coverage
- `ci-pass` — final gate, all must pass

## PR Walkthrough: Evidence Generate → Validate → Gate Decision

Here is a complete end-to-end walkthrough for a single PR:

```
Developer workflow:
  1. Make code changes
  2. Record evidence: sdp skill record --skill build --type generation --ws-id 00-001-01 --data '{...}'
  3. Commit + push + open PR

CI pipeline:
  4. evidence-gate:
     - Finds .sdp/evidence/*.json in diff
     - Runs: go run ./cmd/sdp-evidence validate --require-pr-url=false <file>
     - Checks: id present, type valid enum, timestamp ISO 8601, ws_id matches pattern
     - Result: PASS or FAIL with specific validation errors
  5. scope-gate:
     - Reads checkpoint files from diff
     - Validates workstream scope matches changed files
     - Result: PASS or FAIL with scope drift report
  6. consistency-gate:
     - Runs: sdp verify
     - Checks guard-rules, schema conformance, file hygiene
     - Result: PASS or FAIL with violation list
  7. policy-gate:
     - Aggregates: evidence-gate ✓ + consistency-gate ✓ + coverage-gate ✓
     - Runs auto-attestation if configured
     - Result: PASS or FAIL with policy summary
  8. ci-pass:
     - All 6 jobs must be "success"
     - Result: merged ✅ or blocked ❌

Gate decision: If any gate fails, the PR cannot merge. Each gate provides a clear
failure reason in the GitHub Actions log. Fix the issue and re-push.
```

## Expected Outputs

After completing the pilot, your repository will have:

| Artifact | Location | Purpose |
|----------|----------|---------|
| SDP config | `.sdp/config.yml` | Project settings |
| Guard rules | `.sdp/guard-rules.yml` | Quality gate rules |
| Evidence log | `.sdp/log/events.jsonl` | Append-only audit trail |
| CI gates | `.github/workflows/ci.yml` | Automated gate checks |

Each PR will produce:
- GitHub Actions check results per gate
- Evidence validation output in CI logs
- Scope compliance report (if checkpoints used)

## Troubleshooting

### evidence-gate fails: "invalid evidence file"

**Cause:** Evidence JSON doesn't match the schema.

**Fix:** Validate locally before pushing:
```bash
go run ./cmd/sdp-evidence validate .sdp/evidence/my-evidence.json
```

Common issues:
- Missing required field (`id`, `type`, `timestamp`, `ws_id`)
- `type` not in enum: must be one of `plan`, `generation`, `verification`, `approval`, `decision`, `lesson`
- `ws_id` doesn't match pattern `XX-XXX-XX`

### scope-gate fails: "scope drift detected"

**Cause:** Files changed in the PR don't match the workstream checkpoint scope.

**Fix:** Either:
1. Update the checkpoint scope to include the new files, or
2. Move the unrelated changes to a separate PR

### consistency-gate fails: "protocol violation"

**Cause:** File doesn't conform to SDP protocol rules (guard-rules.yml).

**Fix:** Run locally:
```bash
sdp verify
```

### coverage-gate fails: "below minimum"

**Cause:** Test coverage dropped below the configured minimum (default 60%).

**Fix:** Add tests or adjust the minimum in `.sdp/config.yml`:
```yaml
coverage:
  minimum: 50
```

### policy-gate fails: "auto-attestation required"

**Cause:** The policy gate requires an attestation step before merge.

**Fix:** Run:
```bash
go run ./internal/evidence/cmd/auto-attest --help
```

### All gates pass locally but fail in CI

**Cause:** CI uses `origin/base...HEAD` diff, which may differ from local `main...HEAD`.

**Fix:** Always test against the actual base branch:
```bash
git fetch origin main
sdp verify --base origin/main
```

## Migration Path

After the CI-gate-only pilot, teams can optionally migrate to contracted runtime mode (see [enterprise-pilot-contracted-runtime.md](./enterprise-pilot-contracted-runtime.md)):

1. **CI-only (this document)** — gates enforce at merge time
2. **Contracted runtime** — schema validation as ingest precondition, runtime decisions, handoff events
3. **Full orchestration** — event-driven agent loop with findings and runtime decisions

Each step is incremental. No step requires undoing the previous one.

## Reference

- [Evidence schema](../../schema/evidence.schema.json)
- [CI workflow](../../.github/workflows/ci.yml)
- [Guard rules](../../.sdp/guard-rules.yml)
- [Contract workflow](./CONTRACT-WORKFLOW.md)
- [Evidence coverage](./EVIDENCE-COVERAGE.md)
