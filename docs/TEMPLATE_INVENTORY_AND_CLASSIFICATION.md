# Template Inventory and Classification

Status: working inventory
Date: 2026-03-23
Phase: 3 first thin slice
Scope: classify the current SDP template-like assets without pretending a larger generator framework already exists

## Purpose

This inventory makes the current template surface explicit so new implementation sessions do not have to guess:
- which files are canonical source templates
- which things are generated/operator outputs
- which things are renderers or delivery surfaces
- which assets are ambiguous and should not be oversold

This is a classification pass, not a migration plan.

---

## Classification rules

### Canonical template
Hand-maintained source structure for a recurring artifact.

### Generator or generated-operator-output source
Code or template source used to produce run-specific artifacts/configs.
This category includes generator code and the source templates those generators consume.
It does **not** mean the generated files themselves belong in `docs/templates/`.

### User-facing rendering surface
A renderer, presentation contract, or export shape optimized for delivery/UX rather than canonical authorship.

### Ambiguous / needs cleanup
Template-like or template-named assets that are real, but should not be casually treated as canonical Phase 3 control-tower templates.

---

## A. Canonical source templates

These are the current Phase 3-safe canonical templates for recurring SDP artifacts.

### Control-tower / artifact templates in `docs/templates/`
- `docs/templates/task-brief.template.md`
- `docs/templates/implementation-plan.template.md`
- `docs/templates/verification-note.template.md`
- `docs/templates/review-note.template.md`
- `docs/templates/handoff-note.template.md`
- `docs/templates/control-tower-launch-brief.template.md`

Why they belong here:
- hand-maintained
- generic and reusable
- referenced by canon/packs/docs
- not tied to one specific run

### Adjacent canonical structured template
- `specs/strict-evidence-template.json`

Why it belongs here:
- it is a stable source structure for strict evidence artifacts
- it is not a rendering surface
- it should be treated as source, even though it is JSON instead of markdown

---

## B. Generator sources and generated-operator-output lineage

These assets are real, but they are **not** the same thing as the canonical control-tower templates above.

### Workstream generation
- `internal/workstream/template.go`
- `sdp/templates/workstream.md`
- `sdp/templates/workstream-v2.md`
- `sdp/templates/workstream-frontmatter.md`

Classification:
- `internal/workstream/template.go` = generator code
- `sdp/templates/workstream*.md` = generator source templates / legacy template assets
- generated workstream markdown under `docs/workstreams/` = generated or instantiated operator artifacts, not canonical source templates

### Profile/config generation
- `internal/profile/config_templates.go`

Classification:
- generator code for file outputs like `config.yaml`, `adapters.yaml`, `guard.yaml`
- useful template lineage, but not part of the Phase 3 control-tower canonical template set

### Other reusable source templates under `sdp/templates/`
- `sdp/templates/breaking-changes.md`
- `sdp/templates/idea-draft.md`
- `sdp/templates/migration-guide.md`
- `sdp/templates/PROJECT_CONVENTIONS.md`
- `sdp/templates/release-notes.md`
- `sdp/templates/skill-template.md`
- `sdp/templates/uat-guide.md`

Classification:
- reusable source templates for older/broader SDP surfaces
- real assets, but not part of the new `docs/templates/` control-tower canon by default
- should be referenced explicitly by use case, not implicitly lumped into Phase 3 packs

---

## C. User-facing rendering surfaces

These surfaces present state or instructions to humans, but they are not canonical source templates.

### CLI rendering code
- `internal/cli/status_view.go`
- `internal/cli/instructions_view.go`

Why:
- they produce delivery-oriented text/JSON views
- they are presentation contracts for humans/operators/automation
- they should not be documented as canonical templates

### Board snapshots and feedback payloads
- project/portfolio snapshot JSON under `.sdp/control/.../snapshots/`
- exported feedback packets and resume/import payloads
- result-ingest packets

Why:
- these are run-specific transport/rendering surfaces
- they are derived from state or used to move state
- they are not the hand-authored source template layer

---

## D. Ambiguous / needs cleanup / honesty notes

### `sdp/templates/` as a whole
This directory is real and useful, but it is a mixed bag:
- some files are genuine reusable templates
- some belong to older workstream flows
- some are not relevant to the current control-tower slice

Rule: do not describe `sdp/templates/` as if it were the canonical Phase 3 template home.

### Nonexistent `sdp-plugin/templates/`
The roadmap/spec lineage mentioned `sdp-plugin/templates/`, but that path is not present in the current repo.

Rule:
- do not reference it as an active template surface
- do not build pack guidance around it
- if such a path returns later, classify it explicitly instead of hand-waving

### `docs/workstreams/` files
Many markdown files there look template-ish because they share structure.
They are better treated as instantiated/generated work artifacts, not canonical templates.

### Delivery examples in docs
Some docs include “template” sections, examples, or message snippets.
Unless they are promoted into a stable source file, treat them as documentation examples rather than canonical templates.

---

## Phase 3 working rules

1. `docs/templates/` is the canonical home for the current thin control-tower/source-artifact templates.
2. Do not put generated run output in `docs/templates/`.
3. Packs may reference canonical templates, but only when the file actually exists and is useful for the stage.
4. CLI renderings, packets, and board snapshots are output/rendering surfaces, not canonical templates.
5. Do not imply a bigger generator framework than the repo actually implements today.
6. Do not treat all historical `sdp/templates/` files as part of the new control-tower canon automatically.

---

## Immediate practical mapping for new implementation sessions

If the session needs a recurring control-tower artifact structure, start here:
- intake/shaping task framing → `docs/templates/task-brief.template.md`
- execution planning → `docs/templates/implementation-plan.template.md`
- verification evidence → `docs/templates/verification-note.template.md`
- review wrap-up → `docs/templates/review-note.template.md`
- handoff/continuation → `docs/templates/handoff-note.template.md`
- new implementation-session bootstrap → `docs/templates/control-tower-launch-brief.template.md`

If the session needs a generated runtime artifact, config output, board snapshot, or CLI display, do **not** pretend that `docs/templates/` is the primary source of that output.
