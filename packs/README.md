# SDP Stage Packs

Status: v1 canonical stage-pack index
Date: 2026-03-23
Scope: thin, practical process packs for the SDP control tower

## Purpose

Stage packs make the control-tower lifecycle easier to enter without redefining it.

They are:
- a **stage-oriented operator/implementation guide**
- a way to make the existing control-store flow discoverable
- a thin layer over the current canon, contracts, and CLI

They are **not**:
- a replacement for `CONTROL_TOWER_CANON.md`
- a replacement for `docs/CONTROL_STORE_SKELETON.md`
- a second architecture
- a duplicate of OmO

If the control tower canon explains **what the system is**, packs explain **how to work one stage cleanly**.

---

## Available v1 packs

### [Intake Pack](intake/PACK.md)
Scope: raw request → card in control store

Use for:
- creating cards from raw requests
- preserving intake truth
- confirming intake artifacts/snapshots exist

### [Shaping Pack](shaping/PACK.md)
Scope: inbox / clarifying / needs_input → ready

Use for:
- clarifying intent and scope
- deciding when to ask for human input
- applying the ready gate without over-specifying

### [Execution Bridge Pack](execution-bridge/PACK.md)
Scope: ready → executing → result ingestion

Use for:
- bridging a ready card into Beads-backed execution
- keeping linkage via `linked_beads_ids`
- ingesting executor outcomes back into the card lifecycle

### [Feedback Loop Pack](feedback-loop/PACK.md)
Scope: cross-cutting human/admin feedback loop

Use for:
- generating normalized feedback packets
- exporting/importing feedback via external systems
- resuming a blocked or input-waiting card without losing orchestrator ownership

---

## When to read which pack

Read the pack that matches the work slice:

- `sdp card create` / intake artifact / intake hygiene → `packs/intake/PACK.md`
- `sdp card clarify`, `sdp card needs-input`, `sdp card ready`, parking, ready-gate work → `packs/shaping/PACK.md`
- `sdp card execute`, `sdp dispatch next`, `sdp orchestrate once`, `sdp result ingest` → `packs/execution-bridge/PACK.md`
- `sdp card feedback`, `sdp card feedback-export`, `sdp card resume`, `sdp card resume-import` → `packs/feedback-loop/PACK.md`

If the work spans multiple stages, start from the most immediate stage and then follow references.

---

## Relationship to canon

Read order for new control-tower implementation work:

1. `/home/fall_out_bug/.openclaw/workspace/CONTROL_TOWER_CANON.md`
2. `docs/CONTROL_TOWER_IMPLEMENTATION_ROADMAP.md`
3. `docs/CONTROL_STORE_SKELETON.md`
4. relevant stage pack from this directory

That order matters.
A pack assumes the reader already understands the basic control-tower model.

---

## Pack contract

Each pack should stay thin and practical.

Expected contents:
- stage purpose
- entry/exit boundaries
- canonical commands already supported today
- minimal rules and anti-patterns
- references to deeper docs and real canonical templates instead of duplicating them

Avoid in packs:
- speculative future framework
- fake command flags
- pack-local template systems unless they actually exist
- references to nonexistent template paths
- long restatements of contracts already defined elsewhere

## Template discipline for packs

Packs sit on top of the Phase 3 template/generator discipline:
- canonical source templates live in `docs/templates/`
- generated outputs belong in runtime/project output paths, not in `docs/templates/`
- CLI renderings, snapshots, and transport packets are delivery surfaces, not canonical templates

Use packs to point at real templates when that helps the stage.
Do not imply that every stage has its own hidden template system.

Reference: `docs/TEMPLATE_INVENTORY_AND_CLASSIFICATION.md`

---

## Current non-goals

These packs do not yet formalize:
- verification/review as a separate pack
- release orchestration as a separate pack
- retro/process-improvement packs
- pack-driven code generation
- pack-specific validators beyond existing `sdp doctor control`

That is intentional. v1 is about formalizing the current slice, not overbuilding it.

---

## Related docs

- `docs/SESSION_START_CANON.md`
- `docs/CONTROL_STORE_SKELETON.md`
- `docs/SDP_OPERATIONAL_LAYER_ROADMAP.md`
- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `docs/ORCHESTRATOR_ACTIONS_AND_FEEDBACK_CONTRACT.md`
- `docs/BEADS_GASTOWN_SDP_CONTROL_TOWER_INTEGRATION_PLAN.md`

---

## Status

| Pack | Scope | Status |
|---|---|---|
| intake | request → card | v1 |
| shaping | inbox/clarifying/needs_input → ready | v1 |
| execution-bridge | ready → executing → result ingestion | v1 |
| feedback-loop | cross-cutting feedback/resume | v1 |

Phase 2 outcome: the control tower now has an official stage-pack doc slice for the currently implemented lifecycle surface.
