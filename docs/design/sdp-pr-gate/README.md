# sdp-pr-gate (ChangePassport) Design v1

Design-only artifacts for the ChangePassport merge-readiness product.

**Internal namespace:** `sdp-pr-gate`
**Display name:** ChangePassport
**Status:** Design v1 — no implementation

## Artifacts

| WS | Artifact | Description |
|---|---|---|
| 00-151-01 | [Passport Schema v1](passport-schema-v1.md) | JSON Schema for the Change Passport |
| 00-151-02 | [Evidence Provider API v1](evidence-provider-api-v1.md) | Provider contract for evidence ingestion |
| 00-151-03 | [Decision Record v1](decision-record-v1.md) | Decision schema and lifecycle |
| 00-151-04 | [Override Protocol v1](override-protocol-v1.md) | Comment-trigger override mechanism |
| 00-151-05 | [GitHub App v1 Flow](github-app-v1-flow.md) | End-to-end GitHub integration flow |
| 00-151-06 | [Pilot Measurement Plan v1](pilot-measurement-plan-v1.md) | Metrics, sample sizes, stop/go rules |

## Schemas

| Schema | Path |
|---|---|
| Passport | `schema/sdp-pr-gate/passport.schema.json` |
| Evidence Event | `schema/sdp-pr-gate/evidence-event.schema.json` |
| Decision Record | `schema/sdp-pr-gate/decision-record.schema.json` |

## Source Documents

- ChangePassport manifesto v2 (2026-04-26)
- SDP Product Layering 4D Strategy Memo v3 (2026-04-27)
- SDP Roadmap v3 — Post-IIP Council (2026-04-27)

## Design Principles

1. **Observed facts are immutable.** Manual annotations are separate and cannot overwrite evidence.
2. **Missing evidence is visible.** Degraded or absent evidence is surfaced, not silently passed.
3. **Decisions are accountable.** Every decision has an owner, a reason, and a timestamp.
4. **Overrides are auditable.** Every override produces an append-only audit entry.
5. **Provider-agnostic.** Schemas and APIs work across platforms, not just GitHub.
6. **Idempotent ingestion.** Evidence events can be safely replayed without duplication.
