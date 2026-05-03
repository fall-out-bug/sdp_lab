# F165 Workstreams Spec Interrogate

Date: 2026-05-03
Artifacts:

- `docs/workstreams/backlog/00-165-00.md`
- `docs/workstreams/backlog/00-165-01.md`
- `docs/workstreams/backlog/00-165-02.md`
- `docs/workstreams/backlog/00-165-03.md`
- `docs/workstreams/backlog/00-165-04.md`
- `docs/workstreams/backlog/00-165-05.md`
- `docs/workstreams/INDEX.md`
- `docs/roadmap/ROADMAP.md`
- `.beads-sdp-mapping.jsonl`

Feature: F165
Mode: Socratic review
Verdict: PASS

## Provider Coverage

| Round | Role | Provider | Result |
|---|---|---|---|
| 1 | critic | `kimi-coding/k2p6` | 15 plan questions |
| 1 | judge | `minimax/MiniMax-M2.7` | PASS |

## Findings And Resolution

Kimi challenged the plan on:

- F164 ownership and possible overlap
- linear dependency graph
- whether the F165 core is platform scope or feature-local scope
- Beads/workstream mapping ownership
- dotted Beads child IDs versus strict parser expectations
- advisory report fields being mistaken for delivery gates
- technical isolation of unsafe demo machinery
- "Day-12" meaning and scope
- unsafe oracle meaning
- Normalize rule scope creep
- proof that no live Beads/Git/network/model calls occur
- provider degradation threshold
- ingestion path assumptions
- `blocked_reason` scope creep
- `.github/workflows` scope while forbidding blocking CI

The plan was revised to:

- state F165 builds on F164 by reference and does not reopen F164
- document the linear dependency rationale
- scope the core as shared inside F165 only
- explain strict-parser-compatible leaf IDs
- mark report verdicts as demo/eval verdicts only
- require fake tool registry assertions and static/runtime checks
- define the unsafe oracle as a deterministic stub
- treat Normalize expansion as scope change or follow-up
- set provider coverage threshold to three successful providers or two plus
  recorded provider failure and maintainer acceptance
- remove `.github/workflows` from WS05 scope

MiniMax judged all 15 questions resolved with no new contradictions and no scope
creep.

## Protocol Check

`go run ./cmd/sdp-protocol-check --format json --strict-beads` exits 0 after the
F165 fixes. Remaining output contains pre-existing warnings for other historical
features and no F165 errors.

