# Platform Backlog Reset (2026-03-31)

> **Status:** Active
> **Goal:** Rebuild the executable backlog around the agent-platform direction so the next work is kernel-first, adapter-aware, augmentation-ready, and eval-driven.

Related:

- `docs/roadmap/AGENT_PLATFORM_ROADMAP_2026-03-31.md`
- `docs/plans/2026-03-31-agent-platform-kernel-boundary-spec.md`
- `docs/workstreams/INDEX.md`
- `docs/roadmap/ROADMAP.md`

## Decision

Effective immediately, feature picking should follow the platform lane first:

1. `F091` platform backlog reset and doc sync
2. `F092` kernel contract surface
3. `F093` adapter gateway layer
4. `F094` augmentation engine
5. `F095` behavioral eval system

Trust and evidence work still matters.
It is no longer the default execution path.
It is the `trust lane`.

## Why this reset is necessary

The old backlog still optimizes for:

- trust and evidence enforcement
- ecosystem bridges
- enterprise governance packs

The new architecture direction requires the opposite order:

- first define the reusable kernel
- then isolate adapters
- then build augmentation
- then make evals behavior-level
- only then expand trust and enterprise hardening

Without this reset, the queue keeps dragging execution back into trust-lane work before the substrate exists.

## Execution rules

- `F091`–`F095` are the current priority lane.
- Old backlog items remain valid unless explicitly superseded, but they are secondary unless they directly unblock `F091`–`F095`.
- `F090` remains foundational and should be treated as completed prerequisite context, not the active strategy.
- New work should prefer extending `internal/kernel`, `internal/adapters`, `internal/augmentation`, and `internal/evals` before adding more trust-only surfaces.

## Existing feature triage

### Absorb into the new platform lane

- `F068` Unified Integration Contracts → split across `F091` and `F092`
- `F072` Advanced Agent Architecture for AI SDLC → split across `F094` and `F095`
- `F073` BYOM Model Gateway → becomes a slice inside `F093`

### Keep as secondary trust lane

- `F064`–`F067` auto-attestation and discrepancy features
- `F074`, `F078`, `F079`, `F080`, `F082`, `F083`, `F084`, `F085`

### Keep as ecosystem or research lane

- `F060`–`F063`
- `F069`–`F071`
- `F075`
- `F081`

## First executable workstream train

| Order | WS | Purpose |
|------|----|---------|
| 1 | `00-091-01` | Make the planning surface reflect the new priority lane |
| 2 | `00-092-01` | Complete the first kernel contract boundary |
| 3 | `00-093-01` | Extract provider and routing contracts into kernel ownership |
| 4 | `00-092-02` | Move session, trace, and artifact consumers further onto kernel types |
| 5 | `00-093-02` | Isolate runtime adapters behind a kernel-facing interface |
| 6 | `00-094-01` | Introduce workflow-pack manifest and lazy loading |
| 7 | `00-094-02` | Add role, hook, and approval augmentation surfaces |
| 8 | `00-095-01` | Replace transcript-matching as the primary eval model |
| 9 | `00-095-02` | Add behavior-level regression suites |

## Exit criteria for the reset

- `docs/workstreams/INDEX.md` shows `F091`–`F095` as the active execution lane
- backlog files exist for the first platform workstreams
- beads contains matching tasks for the new workstreams
- roadmap explicitly states that trust is a lane, not the whole platform story
