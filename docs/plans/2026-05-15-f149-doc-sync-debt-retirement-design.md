# F149: strict doc-sync debt retirement

Status: draft for review

Owner issue: `sdplab-t5k3`

Parent workstream: `00-149-01`

## Problem

`sdp-doc-sync --mode check --strict` is a local blocking check, but the same class
of findings is advisory in CI while historical backlog drift is reconciled. That
split made "known repo-wide backlog/link debt" a reusable handoff note instead
of owned work.

Current baseline on 2026-05-15:

- strict `sdp-doc-sync` exits 2
- 56 findings
- buckets: broken local links, missing ROADMAP/INDEX feature rows, scaffold
  workstreams without required `## Beads` or `## Acceptance Criteria`, and
  workstreams whose Beads section has no concrete `sdplab-*` reference

## Goal

Retire the strict `sdp-doc-sync` backlog so the local strict check exits 0, then
change the repo policy so future "known debt" notes must either point at an open
owner issue or fail the relevant gate.

This work does not add a doc-sync allowlist. That would preserve the failure
mode under a more formal name. After this branch, strict doc-sync findings are
blocking unless a future PR explicitly changes the tool or CI contract with its
own reviewed policy.

## Non-goals

- Reopen product design for F083/F084/F085/F101/F133/F145/F146/F147.
- Complete the F145/F146/F147 implementation workstreams.
- Rewrite historical archive docs outside the active docs checked by
  `sdp-doc-sync`.
- Merge without a separate PR review and explicit merge authorization.

## Slices

### Slice 1: ownership and spec

- Add this design.
- Add executable workstream `00-149-02`.
- Link `00-149-02` to `sdplab-t5k3` in the workstream and mapping.
- Run independent spec review before broad cleanup.

Acceptance:

- `00-149-02` has explicit scope, non-goals, acceptance criteria, and Beads link.
- Review artifacts record at least requirements, evidence/tracing, and DX/gate
  review planes.
- Unusable reviewer output is recorded as `not_assessed` with reviewer, model,
  attempt, and reason; it is not counted as review approval.

### Slice 2: factual doc debt cleanup

- Fix broken local links or intentionally mark links historical in active docs.
- Add missing ROADMAP/INDEX entries for features referenced by active backlog
  files, or archive/supersede the backlog files if they are not active backlog.
- Normalize scaffold workstreams so they satisfy the current section contract
  while still stating `design-pending` or non-executable status.
- Replace placeholder Beads text with concrete `sdplab-*` references where the
  Beads issue exists.

Acceptance:

- `go run ./cmd/sdp-doc-sync --mode check --strict` exits 0.
- Before/after doc-sync evidence is recorded:
  - baseline strict output with finding count;
  - final strict output with exit 0;
  - per-finding resolution notes grouped by bucket.
- Any non-executable scaffold keeps all concrete non-executable markers:
  - frontmatter `status: design-pending` or another non-buildable status;
  - an explicit `## Status` section saying it must not be built yet;
  - acceptance criteria that describe design/readiness gates, not implementation
    completion;
  - no language that claims implementation readiness.
- No workstream is made runnable by accident: for every touched
  `design-pending` workstream, reviewers must check that its status and next
  action still route to `/design` or `/feature`, not `/build`.
- Archiving/superseding is allowed only when the resolution note proves the file
  is not linked as active from ROADMAP/INDEX and names the replacement or reason.

### Slice 3: rule and gate cleanup

- Update repo policy to say strict doc-sync debt may be advisory only when it has
  an explicit owner issue, expiry/revisit path, CI artifact, and a reviewed
  tool/CI contract that names the advisory class. A handoff note is not enough.
- Remove stale language that treats historical backlog/doc drift as an indefinite
  advisory class after Slice 2 is green.
- Keep CI advisory/reporting semantics only for newly discovered findings that
  meet the advisory-eligibility criteria above, not for already-retired debt.

Acceptance:

- `docs/reference/ci-gates-map.md`, `docs/reference/quality-gates.md`, and
  `AGENTS.md` agree on current doc-sync semantics.
- CI no longer documents the retired F143/F149 backlog debt as a standing reason
  to ignore doc-sync findings.
- No new allowlist or ignore file is introduced by this remediation.

## Review Plan

Spec review:

- requirements plane: does the spec resolve the real problem without inventing
  product scope?
- evidence/tracing plane: does it preserve `not_assessed` and prevent accidental
  green claims?
- DX/gate plane: will future agents know when doc-sync is blocking vs advisory?

Slice review:

- after Slice 1: review spec and ownership only
- after Slice 2: review the cleanup diff plus strict doc-sync evidence
- after Slice 3: review policy/gate diff and final PR readiness

Review lane states:

- `assessed`: reviewer produced specific findings or explicit approval with
  citations to the embedded target.
- `not_assessed`: reviewer output is empty, asks to call tools instead of
  reviewing the embedded target, is off-task, lacks findings/verdict, or depends
  on unavailable artifacts.
- `degraded`: reviewer completes the plane but misses some requested evidence;
  record what was and was not assessed.

## Verification

Minimum branch gates before PR readiness:

- `go run ./cmd/sdp-doc-sync --mode check --strict`
- `go run ./cmd/sdp-protocol-check --format json --strict`
- `./scripts/run_go_quality_gates.sh` or documented host fallback
- PR checks matching the branch protection set are green; absent, skipped, or
  advisory-only checks are recorded by name and not treated as passing evidence.
