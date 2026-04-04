# Superpowers Curated Intake Wave 1

> **Status:** Draft
> **Date:** 2026-04-01
> **Scope:** `sdp_lab` only
> **Why now:** `faust-workspace` now treats `superpowers` as external upstream and `sdp_lab` as the only curated intake layer for generic dev substrate.

---

## 1. Problem

`superpowers` is a real upstream repo with three things SDP can learn from:

- mature generic skills for planning and execution;
- cross-platform plugin/bootstrap adapters;
- visual brainstorming tooling for browser-based mockups and diagrams.

But importing `superpowers` whole would break SDP in two ways:

1. it would create a second public workflow next to the canonical SDP path;
2. product repos like `faust-workspace` would start depending on `superpowers` directly instead of consuming curated SDP capabilities.

So the rule has to be strict:

- `superpowers` is upstream;
- `sdp_lab` curates what to absorb;
- product repos consume only what SDP has accepted.

---

## 2. Current Overlap Map

| `superpowers` capability | Current SDP equivalent | Decision |
|---|---|---|
| `brainstorming` | `@vision` + `@feature` + `@ux` | Do **not** adopt as a public skill. Absorb only the visual companion subsystem. |
| `writing-plans` | `@feature` + `@design` + workstream/beads graph | Keep upstream-only. SDP intentionally prefers workstreams over giant implementation plans. |
| `executing-plans` | `@oneshot` | Keep upstream-only as a public skill. SDP already has an outer loop. |
| `subagent-driven-development` | `@oneshot` + `@build` + `@review` | Selectively absorb the internal two-stage review pattern, not the public skill. |
| `requesting-code-review` / `receiving-code-review` | `@review` + findings loop | Keep upstream-only. Reuse ideas only if they tighten SDP review quality. |
| `systematic-debugging` | `@debug` | No immediate intake. Audit later for prompt-quality gaps only. |
| `test-driven-development` | `@tdd` | No immediate intake. Audit later for prompt-quality gaps only. |
| `using-git-worktrees` | no first-class SDP equivalent | Adopt as a conditional local-execution helper, not a public stage. |
| `finishing-a-development-branch` | `@deploy` + PR flow | Keep upstream-only. Borrow checklist ideas later if needed. |
| `using-superpowers` | SDP bootstrap/install docs | Never adopt. This is upstream branding/bootstrap, not SDP. |
| `writing-skills` | SDP prompt authoring docs | Keep upstream-only. |

---

## 3. Intake Rules

Any capability coming from `superpowers` must satisfy all of these:

1. **No direct product dependency**
   `faust-workspace` and other product repos must not point to `superpowers` as canonical tooling.

2. **No second public workflow**
   SDP keeps one public path:
   `@vision -> @feature -> @oneshot -> @review -> @qa -> @deploy`

3. **Adopt capability, not branding**
   We take the behavior or tooling, not the `superpowers` namespace, docs layout, or prompt taxonomy.

4. **Curate in `sdp_lab` first**
   Experiment and stabilize in `sdp_lab`. Publish to `sdp/` only if the capability becomes part of the public protocol.

5. **Ephemeral outputs stay ephemeral**
   Browser mockup/session outputs must not become tracked canon by default.

---

## 4. Wave 1: Adopt Now

### 4.1 Visual Companion

**Source in `superpowers`:**

- `skills/brainstorming/visual-companion.md`
- `skills/brainstorming/scripts/start-server.sh`
- `skills/brainstorming/scripts/server.cjs`
- `skills/brainstorming/scripts/frame-template.html`
- `skills/brainstorming/scripts/helper.js`

**Why Wave 1:**

- SDP does not currently have an equivalent visual companion.
- This is the highest UX leverage for `@vision`, `@feature`, and `@ux`.
- It improves design and architecture conversations without changing the public stage model.

**How to absorb it:**

- bring the browser companion into `sdp_lab` as an internal capability;
- expose it only as optional support inside `@vision`, `@feature`, and `@ux`;
- store outputs under an SDP-owned path like `.sdp/visual/` or another ignored session directory, not `.superpowers/brainstorm/`;
- treat generated mockups as ephemeral unless explicitly promoted into `docs/mockups/` or another canonical artifact path.

**Explicit non-goal:**

- do **not** import the full `brainstorming` skill as a public replacement for `@feature` or `@vision`.

### 4.2 Local Worktree Isolation Helper

**Source in `superpowers`:**

- `skills/using-git-worktrees/SKILL.md`

**Why Wave 1:**

- SDP has branch rules and PR rules, but not a first-class local isolation helper.
- This is useful for local execution outside K8s/orchestrator flows.
- It is valuable as operator support, but it does not deserve its own public stage.

**How to absorb it:**

- implement as a conditional helper/runbook for local execution;
- keep it behind operator workflows, not on the canonical public path;
- align it with SDP branch policy instead of copying the `superpowers` wording verbatim.

**Explicit non-goal:**

- do **not** make worktrees a mandatory prerequisite for every SDP feature flow.

### 4.3 Two-Stage Internal Review Pattern

**Source in `superpowers`:**

- `skills/subagent-driven-development/SKILL.md`

**Why Wave 1:**

- SDP already has `@oneshot` and `@review`, so the public skill is redundant.
- But the internal pattern is strong: implementation, then spec compliance review, then code-quality review.
- This can tighten SDP execution quality without adding new public surface area.

**How to absorb it:**

- integrate the pattern into `@oneshot` / orchestrator internals;
- keep the public entrypoint unchanged;
- model it as an internal execution policy, not a user-facing skill.

**Explicit non-goal:**

- do **not** expose `subagent-driven-development` as a new top-level SDP skill.

---

## 5. Leave Upstream-Only For Now

These stay in `superpowers` until there is a specific gap and a curated rewrite:

- full `brainstorming` public skill;
- `writing-plans`;
- `executing-plans`;
- `finishing-a-development-branch`;
- `using-superpowers`;
- `writing-skills`;
- `requesting-code-review` and `receiving-code-review` as standalone public skills.

Reason:

- they either duplicate the canonical SDP path,
- or they encode a different artifact model,
- or they are upstream-branding/bootstrap rather than reusable substrate.

---

## 6. Consequences For Product Repos

For `faust-workspace` this means:

- product docs and prompts stay local;
- any reusable capability from `superpowers` must appear first as an SDP-owned capability;
- leaked namespaces like `docs/superpowers/*` and `.superpowers/brainstorm/*` must be cleaned up rather than normalized as product canon.

---

## 7. First Follow-Ups

1. Create an SDP task for visual companion intake and define target placement in `sdp_lab`.
2. Decide whether visual outputs live under `.sdp/visual/` or another ignored session path.
3. Define the operator contract for local worktree setup.
4. Specify how the two-stage internal review loop hooks into `@oneshot` without adding public skill sprawl.

---

## 8. Success Criteria

- SDP has one written intake policy for `superpowers`.
- Wave-1 adopted capabilities are explicit.
- Public SDP surface stays unchanged.
- Product repos can point to SDP, not to `superpowers`, for accepted generic dev capabilities.
