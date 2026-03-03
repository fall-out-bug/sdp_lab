# A* State Alignment Stream

Status: active
Updated: 2026-03-03
Purpose: stabilize repository control plane before any new feature expansion

## 1. Stream mandate

No new feature development starts until A* stabilization gates pass.

Primary objective:

- make repo state trustworthy for planning and release decisions

Secondary objective:

- harden CI and tooling so drift cannot silently reappear

## 2. Role-model synthesis

Think persona baseline:

- strategic direction is coherent; operational control is the bottleneck
- key risk is control-plane drift, not roadmap intent

Review persona baseline:

- go/no-go must be gate-driven and measurable
- advisory checks are insufficient for state governance

Explorer/librarian evidence:

- consistency drift previously existed across roadmap/index/backlog
- CI contains soft-fail patterns and fake-input policy evaluation paths
- external OSS benchmark confirms strict lint/build gates, deterministic checks, and mandatory PR controls as baseline practice

## 3. Current A* baseline (evidence)

- consistency checker (non-strict): 0 errors, 6 warnings
- consistency checker (strict-ac): 6 errors
- system Go toolchain mismatch with project baseline blocks local `go run` authority for several commands

Independent review verdict artifacts:

- `.sdp/STATE_ALIGNMENT_AUDIT_2026-03-03.md`
- `.sdp/state_alignment_verdict.json` (`NO_GO`)

Warning set in strict mode:

- `docs/workstreams/backlog/00-026-01.md`
- `docs/workstreams/backlog/00-059-01.md`
- `docs/workstreams/backlog/00-059-02.md`
- `docs/workstreams/backlog/00-061-01.md`
- `docs/workstreams/backlog/00-061-02.md`
- `docs/workstreams/backlog/00-067-01.md`

## 4. Stream phases

### Phase A — State authority lock

- reconcile roadmap/index/backlog/mapping status mismatches
- remove phantom workstream references
- keep one source-of-truth policy active

Exit criteria:

- status mismatch errors = 0
- phantom references = 0
- mapping count mismatch = 0

### Phase B — Done semantics hardening

- resolve done-with-unchecked-AC warnings by evidence completion or status correction
- move strict AC check to blocking in CI

Exit criteria:

- strict-ac errors = 0

### Phase C — CI hardening

- eliminate soft-pass behavior on governance checks
- remove fake policy input paths and wire real gate outputs
- pin unstable tooling versions

Exit criteria:

- no governance-critical check can pass on command failure
- policy gate input fields are derived from real checks

### Phase D — Tooling parity

- ensure authoritative local verification path independent of global PATH assumptions
- require reproducible local check command for contributors

Exit criteria:

- documented one-command local verification path works on clean env

## 5. Go/No-Go criteria for new feature work

Go only if ALL conditions are true:

1. `python3 scripts/check_repo_consistency.py --strict-ac --json` returns `ok=true`
2. CI consistency gate is green on PR
3. policy/evidence/scope checks have no soft-pass bypass for blocking conditions
4. backlog/index/roadmap statuses are synchronized
5. ready queue prioritized with explicit stabilization lane completed for current week

If any condition is false: No-Go for new feature start.

## 6. Operating rule: one-feature polish discipline

When feature work resumes:

- only one feature receives primary delivery focus at a time
- full timebox is used to polish that feature to release-grade quality
- no parallel feature starts until current feature reaches done with evidence

Definition of polished feature:

- acceptance criteria complete
- evidence/gate checks pass
- docs and usage flow updated
- no open P1/P2 regressions introduced

## 7. Weekly execution template

Monday:

- run A* baseline checks and publish state snapshot

Tuesday-Thursday:

- execute one stabilization objective (not feature expansion)

Friday:

- enforce go/no-go decision for next week
- if no-go, keep stabilization stream active

## 8. Mitigation-first execution sequence

Sequence order is strict:

1. Toolchain parity and compile baseline
2. Done-with-unchecked-AC cleanup to enable strict consistency pass
3. CI gate hardening (remove soft-pass behavior)
4. Working tree entropy reduction into auditable slices
5. Re-run strict go/no-go gates

Feature throughput resumes only after step 5 is green.

## 9. Related documents

- `docs/roadmap/CONSISTENCY_MITIGATION_POLICY.md`
- `docs/roadmap/CRITICAL_ROADMAP_REVIEW_2026-03-03.md`
- `docs/roadmap/IMPLEMENTATION_DRIFT_AUDIT_2026-03-03.md`
- `docs/roadmap/UNIFIED_VISION_ROADMAP_2026-03-03.md`
- `docs/research/MARKET_LANDSCAPE_AI_TRUST_TOOLS_2026-03-03.md`
- `docs/plans/2026-03-03-oss-growth-playbook-12weeks.md`
