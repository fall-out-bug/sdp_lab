# SDP CLI Control Tower Pack

> **Status:** Draft design
> **Date:** 2026-03-08
> **Goal:** Turn the SDP CLI from a large command collection into a coherent operator-facing control tower

---

## Core Idea

The SDP CLI should answer four questions from one coherent surface:

1. **What is the current state?**
2. **What should I do next?**
3. **If something is wrong, how do I fix it?**
4. **Why did SDP make that recommendation or decision?**

Today the command surface already has most of the raw capability, but it is fragmented across many entrypoints:

- `sdp status`
- `sdp next`
- `sdp doctor`
- `sdp health`
- `sdp quality`
- `sdp diagnose`
- `sdp log`
- `sdp verify`
- `sdp decisions`
- `sdp metrics`

The CLI improvement pack should not add random new power. It should make existing power feel like one product.

---

## Why This Pack Matters

The public CLI already registers a large command tree in `sdp/sdp-plugin/cmd/sdp/main.go`, but the product story is weaker than the implementation surface.

Observed issues:

- root help mentions only a small subset of commands while `main.go` registers dozens
- `sdp status` and `sdp next` are not yet one canonical guidance system
- `sdp doctor`, `sdp health`, `sdp quality`, and `sdp diagnose` overlap without a clear hierarchy
- `sdp log`, `sdp verify`, `sdp decisions`, and `sdp metrics` expose evidence, but not as one explainability lane
- docs and completion lag the actual CLI surface
- implemented features can be effectively hidden when docs and shell completion lag the actual command tree, even after commands like `demo` are registered

This is exactly the kind of product debt that makes an OSS CLI feel more complex than it really is.

---

## Product Thesis

The SDP CLI should behave like a **control tower**.

That means:

- one canonical state model
- one canonical next-step resolver
- one canonical diagnostics entrypoint
- one canonical explainability path
- one discoverable command map for humans and machines

If a command does not strengthen one of those five things, it does not belong in this pack.

---

## The Pack

### 1. Guided State Pack

**Goal:** unify `status`, `next`, and help-driven guidance.

Scope:

- make `sdp status` the canonical state surface
- make `sdp next` use the same underlying recommendation source as `status`
- add stable machine-readable `status` and `instructions` contracts
- distinguish `ready`, `blocked`, `recovery`, and `planning` states explicitly
- include rationale in every next-step recommendation

Why this matters:

- `sdp/sdp-plugin/cmd/sdp/status_text.go` still uses naive counting and manual JSON printing
- `sdp/sdp-plugin/cmd/sdp/next.go` is promising, but under-promoted and not yet the center of the UX

Primary work:

- `docs/workstreams/backlog/00-068-04.md`
- `docs/workstreams/backlog/00-069-04.md`
- `docs/workstreams/backlog/00-069-05.md`

### 2. Actionable Diagnostics Pack

**Goal:** make failure handling feel like one lane instead of four separate utilities.

Scope:

- define `doctor` as the primary operator entrypoint
- clarify the relationship between `doctor`, `health`, `quality`, and `diagnose`
- standardize output structure: issue, severity, why it matters, exact fix
- ensure common failure paths point back to `status` or `next`
- add `--json` where missing or inconsistent

Design rule:

- `doctor` = environment and setup
- `health` = project/runtime integrity
- `quality` = code/test gates
- `diagnose` = failure playbooks

But users should not have to guess that model from source code. The CLI must teach it.

### 3. Explainability Pack

**Goal:** make SDP explain what happened and why, without turning this pack into a full observability rewrite.

Scope:

- connect `log`, `verify`, `decisions`, and `metrics` into one explainability story
- improve “why did this fail/pass?” and “what happened in this run?” workflows
- make `verify` point clearly to supporting evidence and decisions
- keep `metrics` as summary and routing, not as a separate platform

Design rule:

- explainability starts from a run, workstream, or decision and fans out to evidence
- it should not require users to know the entire evidence subsystem first

### 4. Discoverability Pack

**Goal:** make the real CLI surface visible and learnable.

Scope:

- repair root help in `sdp/sdp-plugin/cmd/sdp/main.go`
- register and promote `sdp demo`
- align shell completion with the actual command tree
- expand `sdp/docs/CLI_REFERENCE.md` to cover real top-level commands and important subcommands
- update `sdp/README.md`, `sdp/docs/QUICKSTART.md`, and `sdp/sdp-plugin/README.md` with a CLI-first path

Evidence for the need:

- root help under-describes the command tree
- zsh and fish completion lag the live command tree even when bash completion is updated
- help and integration coverage can drift from the current command surface

### 5. Command IA and Release Polish Pack

**Goal:** clean up command architecture without starting a disruptive rename project.

Scope:

- define essential vs advanced command groups
- improve help grouping and progressive disclosure
- decide which commands are primary entrypoints vs specialist tools
- tighten integration tests so the CLI does not silently drift from docs and help
- keep aliases/additional grouping minimal and purposeful

Design rule:

- this slice exists to support slices 1-4, not to create a separate cleanup epic

---

## What This Pack Is Not

- not OpenSpec import
- not memory subsystem expansion
- not `sdp-evidence` packaging
- not a telemetry platform rebuild
- not a mass command rename campaign

Those may matter, but they are not part of the CLI control tower story for this cycle.

---

## Release Order

### Phase 1 - State and discoverability foundation

Ship first:

- guided state contracts and next-step foundation
- root help repair
- `sdp demo` registration
- completion refresh
- `CLI_REFERENCE` expansion for the core operator path

Why first:

- this is the highest user-visible value
- it fixes the biggest OSS problem: discoverability and first success

### Phase 2 - Diagnostics lane

Ship second:

- clear role split across `doctor`, `health`, `quality`, `diagnose`
- shared reporting structure
- help and docs that route users into the right diagnostic command

Why second:

- once state and next-step exist, diagnostics can route cleanly back into them

### Phase 3 - Explainability lane and release polish

Ship third:

- evidence-linked explainability for `verify` / `log` / `decisions`
- tighter integration tests for help/docs drift
- final README and quickstart polish

Why third:

- explainability is most useful after the control tower entrypoints are stable

---

## Pack Guardrails

- Every slice must strengthen the same control tower contract.
- New output shapes should be stable and machine-readable where practical.
- Docs, completion, help, and tests must describe the same command surface.
- No slice may depend on OpenSpec import, memory expansion, or a new `sdp-evidence` release artifact.
- This pack should reduce user decision load, not create more command choice.

---

## Success Criteria

The pack succeeds when all of these are true:

1. A new user can run `sdp status` and understand the current state and next step.
2. A failing user can reach the right fix path without guessing between four overlapping commands.
3. A user can answer “what happened?” and “why?” through the CLI without reading internals first.
4. Root help, completion, CLI reference, quickstart, and actual commands all agree.
5. The CLI feels smaller and clearer even if the number of commands does not shrink.

---

## Recommended First Implementation Slice

If we start tomorrow, we should begin with:

1. `00-068-04` status and instructions contracts
2. `00-069-04` guided next-step status and help
3. `00-069-05` structured failure guidance and walkthrough
4. root help, completion, and `CLI_REFERENCE` repair
5. promote `sdp demo` across help, completion, and walkthrough docs

This gives the pack a visible product win before touching deeper command IA questions.

---

## References

- `sdp/sdp-plugin/cmd/sdp/main.go`
- `sdp/sdp-plugin/cmd/sdp/status.go`
- `sdp/sdp-plugin/cmd/sdp/status_text.go`
- `sdp/sdp-plugin/cmd/sdp/next.go`
- `sdp/sdp-plugin/cmd/sdp/doctor.go`
- `sdp/sdp-plugin/cmd/sdp/health.go`
- `sdp/sdp-plugin/cmd/sdp/diagnose.go`
- `sdp/sdp-plugin/cmd/sdp/log.go`
- `sdp/sdp-plugin/cmd/sdp/quality.go`
- `sdp/sdp-plugin/cmd/sdp/demo.go`
- `sdp/sdp-plugin/internal/ui/completion_bash.go`
- `sdp/docs/CLI_REFERENCE.md`
- `sdp/docs/QUICKSTART.md`
- `sdp/README.md`
- `docs/reviews/sdp-cli-ux-exploration-report.md`
- `docs/plans/2026-03-08-single-release-feature-selection.md`
