# UX Improvement Addendum: Gaps, Counterpoints, and Extensions

**Date:** 2026-04-05  
**Status:** Supplementary — does not replace or modify other UX plan documents  
**Related (read-only references):**

- [UX Audit Results](2026-04-05-ux-audit-results.md)
- [UX Improvement Proposals](2026-04-05-ux-improvement-proposals.md)
- [UX Improvement Specifications](2026-04-05-ux-improvement-specs.md)

This addendum records an **alternative framing**, **deliberate gaps** in the three documents above, **items to add** before or during execution, and **areas to expand** without reopening the agreed SPEC list.

---

## 1. Purpose

The audit and SPEC set optimize for **individual IDE workflows** and **discipline enforcement**. They under-specify **measurement**, **organizational adoption**, **risk-based quality policy**, **non-interactive/CI paths**, and **cross-SPEC consistency**. This file is the single place to track those extensions until they are folded into workstreams or new SPECs.

---

## 2. Alternative Framing (Counterpoint)

SDP’s friction is not only “too much text” or “too many questions.” It is also:

1. **Contract ambiguity** — Users cannot tell what tier of guarantees they get (Claude vs Cursor vs Codex) without reading internals.
2. **Implicit cost model** — Spawn-heavy flows trade money and latency for depth; that trade is rarely surfaced.
3. **Brownfield as default reality** — Greenfield paths are over-fit; most value is in incremental adoption without pretending the repo already matches SDP ideals.

The existing proposals address (1) via harness parity and (2) partially via scaled review; (3) is the focus of SPEC-03. This addendum argues those fixes need **explicit success criteria and governance** so UX wins do not erode trust in evidence and review.

---

## 3. Deliberate Gaps in the Current Document Set

| Gap | Why it matters | Suggested direction |
|-----|----------------|-------------------|
| **No measurement plan** | “&lt;5 min read” and similar targets are not falsifiable without baselines. | Define 5–10 metrics, how to capture them (timed runs, scripted journeys), and a lightweight pre/post check. |
| **Review policy vs UX** | Scaled `@review` and `@review --override` touch security and audit expectations. | Document risk signals (paths, secrets, CI/auth) alongside LOC tiers; tie override to justification + visibility in PR/evidence. |
| **Team / org UX** | Personas are individual-centric. | Add “team lead / platform” scenarios: shared templates, branch protection, multi-user checkpoints. |
| **CI and headless** | Happy paths assume interactive IDE. | One canonical headless story: orchestrator + exit codes + artifacts + no auto-commit conflicts with branch rules. |
| **Privacy and context** | Hydrate/adoption can pull large or sensitive trees into prompts. | Brownfield appendix: what to exclude, minimal hydrate, secrets hygiene. |
| **Observability** | “Silent failure” is a recurring theme; repair helps after the fact. | First-class `sdp doctor` (or equivalent): harness tier, hooks, beads, orchestrator, last errors. |
| **Versioning and upgrades** | Default output flip JSON→text and skill renames break scripts. | Compatibility matrix + migration notes per minor release. |
| **SPEC overlap** | `sdp uninstall` appears in SPEC-03 and SPEC-05; graduation “one-way” vs explicit level down is ambiguous across text. | Single owner doc per command; one rule for adoption level changes. |
| **Competitive trade-offs** | Benchmarks name faster tools but not SDP’s non-negotiables. | Short “when SDP is slower on purpose” paragraph for roadmap/onboarding. |

---

## 4. Items to Add (New Artifacts, Not Yet in SPECs)

These are **additive**. They can become separate SPECs, workstream sections, or checklist items when executing existing SPECs.

1. **Success metrics & UAT plan** — Baseline time-to-first-success, question counts, manual steps per journey, optional user walkthrough script.
2. **Governance snippet for review** — Who may use `--override`, required fields, relation to branch protection (human gate).
3. **Non-happy-path matrix** — Partial install, network loss, merge conflicts on `settings.json`, concurrent editors, corrupted checkpoint mid-run.
4. **Machine-readable completion manifest** — Small JSON (or frontmatter) emitted or documented after `@vision` / `@feature` for tooling, in addition to human “Created / Next” text (SPEC-06 style).
5. **Release/migration guide** — Orchestrator `--json` default change, `@deploy` → `@ship`, merged hooks behavior.
6. **Leakage guard for public sdp** — Procedure ensuring public `AGENTS.md` / harness docs never pull in private lab-only content (called out in audit; procedural gap in execution).
7. **Stop hook scope (F40)** — Config or heuristic: “landing” checklist only after `@oneshot` (or similar), not every chat end; document trade-off with discipline.

---

## 5. Areas to Expand Within Existing SPEC Intent

| Area | Current intent (SPEC) | Extension |
|------|----------------------|-----------|
| **Adaptive `@feature`** | Skip sub-skills when output files exist. | Scope by **feature id** in paths or metadata; stale/wrong-draft detection; explicit `--redo discovery` style escapes. |
| **Scaled `@review`** | LOC buckets. | Add **risk-based** triggers: auth, crypto, deps, infra, workflow files → bump reviewer set even for small diffs. |
| **Harness parity** | `sdp-capabilities.yml` + fallbacks. | Ensure **tier is injected into system context** (generated one-liner in `.cursorrules` / `AGENTS.md`) so the model does not skip reading the manifest. |
| **Brownfield trackers** | Adapters: beads, github, none. | **Migration / coexistence**: dual-write period, id mapping, or explicit “SDP issues are separate” policy. |
| **Orchestrator** | Human status, repair, auto-commit. | Document interaction with **squash merges**, **signed commits**, **amend/rebase**; optional `--no-commit` for CI. |
| **`sdp doctor`** | Mentioned as mitigation in SPEC-03 risks. | Expand into a small cross-cutting spec: one command, stable output, harness + config summary. |

---

## 6. Cross-SPEC Consistency Checklist (For Implementers)

Use this when opening PRs against multiple SPECs:

- [ ] **Uninstall:** One canonical behavior; the other SPEC references it without duplicating semantics.
- [ ] **Adoption level:** Document whether downgrades are allowed via explicit command and how that interacts with evidence/guard history.
- [ ] **Auto-commit:** Default on/off per context (oneshot vs standalone `@build`); documented escape for users who forbid bot commits.
- [ ] **JSON vs text:** Every consumer of orchestrator output has a documented flag and a deprecation window.
- [ ] **Public vs private repo:** Changes to `AGENTS.md`, install merge, and sample configs reviewed for sdp_lab-only content.

---

## 7. Suggested Reading Order With This File

1. Audit results — what is broken.  
2. Proposals + specs — what will be built.  
3. **This addendum** — measurement, governance, CI, and consistency before cutting workstreams.

---

## 8. Non-Goals of This Addendum

- Replacing or editing [2026-04-05-ux-audit-results.md](2026-04-05-ux-audit-results.md), [2026-04-05-ux-improvement-proposals.md](2026-04-05-ux-improvement-proposals.md), or [2026-04-05-ux-improvement-specs.md](2026-04-05-ux-improvement-specs.md).
- Committing to a new numbered SPEC; items here are **candidates** until promoted via `@design` or roadmap.
