# Reality / Reality-Pro Gap Audit

Date: 2026-03-11
Scope: `reality` and `reality-pro` execution readiness in `sdp_lab` + `sdp/` submodule.

## Executive Summary

Current state is spec-heavy and execution-light.

- `reality` has documented contracts and prompt definitions.
- `reality-pro` has conceptual product spec boundaries.
- There is no materialized execution track (no runtime emitters, no schema files in `schema/reality/`, no generated reality artifacts in repo outputs).
- There are no active Beads tasks dedicated to this area in the current ready queue.

Result: direction exists, but it is not currently executable as a feature stream.

## Observed Evidence

### Present

- `docs/specs/reality/OSS-SPEC.md`
- `docs/specs/reality/PRO-SPEC.md`
- `docs/specs/reality/ARTIFACT-CONTRACT.md`
- `sdp/prompts/skills/reality/SKILL.md`
- `sdp/prompts/commands/reality.md`

### Missing or not materialized

- `schema/reality/*.schema.json` files referenced by artifact contract
- runtime pipeline that emits:
  - `docs/reality/*.md`
  - `.sdp/reality/*.json`
- explicit workstream backlog for `reality` and `reality-pro` execution

## Main Gaps

1. Contract is defined, but artifact schemas are absent in repo.
2. Prompt/command entry exists, but no implementation layer emits contract artifacts.
3. `reality-pro` boundary is specified, but no staged bootstrap backlog exists for delivery.

## Recommended Execution Track

1. Materialize artifact schemas in `schema/reality/`.
2. Implement OSS `reality` emitter flow for required outputs.
3. Add validation/CLI checks that fail on malformed reality artifacts.
4. Create staged `reality-pro` bootstrap backlog (contract-first, implementation later).

## Delivery Decision

Create an epic with execution tasks in Beads and start from OSS `reality` contract materialization first.
