# Template and Generator Discipline Spec

Status: working spec
Date: 2026-03-23
Phase: 3
Scope: separate canonical templates, generated operator outputs, and user-facing renderings for SDP

## Purpose

Phase 2 gave SDP stage-oriented packs.
Phase 3 should make those packs sit on top of a cleaner artifact-generation model instead of a loose pile of templates and ad hoc outputs.

This spec defines the minimum useful discipline.

---

## Problem

Right now SDP has:
- canonical-ish markdown templates under `docs/templates/`
- other templates under `sdp/templates/`
- process packs under `packs/`
- various generated/operator-facing artifacts implied by docs and workflows

Without discipline, these collapse into one mess:
- templates get treated as generated outputs
- generated outputs drift from source templates
- user-facing renderings and machine-facing source material get mixed together
- packs start duplicating template structure instead of referencing it

That is exactly the kind of entropy SDP should be preventing.

---

## Goal

Introduce a clear three-layer model:

1. **Canonical templates**
   - source-of-truth structures
   - stable, reusable, hand-maintained
   - live in a predictable place

2. **Generated operator outputs**
   - produced from state + templates + workflow context
   - intended for operators/agents to act on
   - reproducible, not hand-authored as the primary source

3. **User-facing renderings**
   - formatted for messaging, chat, dashboards, or brief delivery
   - optimized for readability and transport
   - can be lossy compared to operator outputs

---

## Core rule

**Do not let one file pretend to be all three things at once.**

If a thing is the canonical source template, treat it as source.
If a thing is a generated operator artifact, label and place it like output.
If a thing is a human-facing rendering, treat it as a rendering surface.

---

## Layer definitions

### 1. Canonical templates

Canonical templates define the expected structure of recurring artifacts.

Examples already present:
- `docs/templates/task-brief.template.md`
- `docs/templates/implementation-plan.template.md`
- `docs/templates/verification-note.template.md`
- `docs/templates/review-note.template.md`
- `docs/templates/handoff-note.template.md`

Properties:
- hand-maintained
- generic and reusable
- not tied to one specific project run
- referenced by packs and runbooks

### 2. Generated operator outputs

These are concrete outputs created for a specific card/work item/session.

Examples:
- a generated task brief for a specific FeatureCard
- a generated implementation-plan draft for a dispatchable card
- a generated launch brief for a new implementation session
- a generated operator checklist from control-store state

Properties:
- specific to a run, card, or session
- derived from canonical templates + state
- may be regenerated
- should not become the hidden source-of-truth if the underlying state changes

### 3. User-facing renderings

These are delivery forms for humans on a particular surface.

Examples:
- short Telegram-friendly feedback message
- compact board summary in CLI
- one-screen operator digest
- email/slack formatted payload

Properties:
- presentation-oriented
- surface-specific
- may omit detail
- should link back to operator output or source state when fidelity matters

---

## Proposed directory discipline

### Canonical source templates
Keep canonical templates under:
- `docs/templates/`

Use this directory for generic source templates that describe recurring SDP artifacts.

### Generated operator outputs
Do not store these in `docs/templates/`.
Generated outputs should live in runtime/project output locations, for example under:
- project-local `.sdp/` control paths when tied to a card/project
- dedicated generated-output folders if a future workflow needs them

This phase does **not** require a big new output tree.
It only requires not confusing generated output with source templates.

### User-facing renderings
Do not treat transport/rendered forms as canonical templates.
They belong either:
- in messaging/export logic
- in CLI rendering logic
- or in explicit delivery/export docs

---

## Relationship to packs

Packs should:
- reference canonical templates
- explain when a template is used
- explain what kind of output is expected
- avoid inventing pack-local template systems unless those files really exist

Packs should **not**:
- duplicate whole template bodies for no reason
- imply fake generated files that do not exist
- mix canonical template definitions with transport renderings

Short version:
- packs explain stage practice
- templates define artifact structure
- generators produce concrete artifacts
- renderers deliver them to humans

---

## Minimum implementation target for Phase 3

Do a thin first slice.

### Slice A — template inventory + classification
Produce a reference doc that classifies current template-like assets into:
- canonical templates
- generated/operator outputs
- user-facing renderings
- ambiguous / needs cleanup

### Slice B — launch brief template
Add one explicit canonical template for implementation-session launch briefs, because session-start discipline already exists but the generated brief structure is still implicit.

Suggested file:
- `docs/templates/control-tower-launch-brief.template.md`

### Slice C — pack/template references
Update stage packs to reference canonical templates where appropriate, but only when the template actually exists and is useful.

### Slice D — anti-confusion rules
Document explicit rules such as:
- never put generated run output in `docs/templates/`
- never document fake templates in packs
- do not use transport renderings as source templates

---

## Non-goals for this phase

Do not do these yet:
- full generator engine
- big templating framework
- pack-driven codegen
- dashboard rendering system
- broad file layout migration across the whole repo
- new universal artifact registry

Phase 3 is about discipline first, not machinery.

---

## Acceptance criteria

Phase 3 first slice is good enough when:
- current template assets are classified clearly
- launch brief becomes an explicit canonical template
- packs reference real canonical templates instead of imagined ones
- docs clearly separate source templates from generated outputs and renderings
- no architecture rewrite is required

---

## Recommended next implementation slice

1. Create `docs/TEMPLATE_INVENTORY_AND_CLASSIFICATION.md`
2. Add `docs/templates/control-tower-launch-brief.template.md`
3. Update `packs/README.md` and relevant pack docs to reference real templates only
4. Update `docs/SESSION_START_CANON.md` or adjacent canon docs to mention launch-brief template discipline
5. Keep the implementation doc-heavy and thin

---

## Short formula

- **templates** = source
- **generated outputs** = reproducible working artifacts
- **renderings** = delivery surfaces
- **packs** = stage guidance over the top of all that
