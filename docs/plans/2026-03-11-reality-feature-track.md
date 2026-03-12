# Reality Feature Track

Updated: 2026-03-11

## Numbering Decision

`F059` and `00-059-*` are already occupied by OhMyOpenCode Evidence Integration.

To avoid branch-collision drift, the reality track is assigned a new range:

- `F090` — Reality OSS Baseline
- `F091` — Reality-Pro Consulting Surface

## Feature Split

| Feature | Scope | Delivery Rule |
|---|---|---|
| `F090` | Open-contract OSS baseline | Must remain executable in this repo and publishable through the `sdp/` CLI surface |
| `F091` | Pro-only orchestration and premium depth | May reuse open artifact contracts, but private logic stays out of public protocol guarantees |

## Phases

### Phase A: F090 contract baseline

Completed:

- `00-090-01` schema contract files
- `00-090-02` OSS emitter flow
- `00-090-03` artifact validation gate

First executable slice already exists:

- `sdp reality emit-oss`
- `go run ./cmd/sdp-reality-validate .`

### Phase B: F090 parity hardening

Completed:

- `00-090-04` close the gap between `OSS-SPEC.md`, `sdp/prompts/skills/reality/SKILL.md`, and the current runtime baseline

### Phase C: F091 planning and boundary freeze

Completed by this planning pass:

- `00-091-01` staged bootstrap backlog

### Phase D: F091 contract materialization

Completed:

- `00-091-02` pro-only schema surface

### Phase E: F091 reposet + memory substrate

Completed:

- `00-091-03` persistent memory and reposet ingestion

### Phase F: F091 consulting-grade orchestration

Backlog:

- `00-091-04` specialist review orchestration
- `00-091-05` pro report and artifact emitters

## Open vs Private Boundary

Open-contract work:

- artifact families and schema files
- claim/source enums
- OSS emitter and validator
- IDs and readiness semantics

Private/pro work:

- specialist selection heuristics
- persistent memory implementation details
- skepticism/arbitration strategy
- multi-source connector implementations
- consulting-grade synthesis logic

## Dependency Spine

`F090-01` -> `F090-02` -> `F090-03` -> `F090-04`

`F090-03` -> `F091-01` -> `F091-02` -> `F091-03` -> `F091-04` -> `F091-05`

## Source Documents

- [`OSS-SPEC.md`](../specs/reality/OSS-SPEC.md)
- [`PRO-SPEC.md`](../specs/reality/PRO-SPEC.md)
- [`ARTIFACT-CONTRACT.md`](../specs/reality/ARTIFACT-CONTRACT.md)
- [`2026-03-11-reality-gap-audit.md`](../reviews/2026-03-11-reality-gap-audit.md)
