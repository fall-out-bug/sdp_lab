# F150 Product Layering And Release Readiness Design

Status: active design (v2, post-council)
Owner: Andrei
Beads epic: `sdplab-nyr0`
Created: 2026-04-27
Revised: 2026-04-27 (council R1 + R2)
Companion memo: [docs/strategy/2026-04-27-sdp-product-layering-4d.md](../strategy/2026-04-27-sdp-product-layering-4d.md)
Council synthesis: [docs/strategy/council/2026-04-27/synthesis.md](../strategy/council/2026-04-27/synthesis.md)

> **v2 note.** §"AI Fluency 4D Reframe" and §"Layers" below are superseded by the companion memo. This file keeps the executable program plan and the patch trail; the canonical product-layering decision lives in the memo.

## Cold Start Answers

1. This is platform work, not "use SDP in my project" onboarding.
2. The owner is `F150` / `sdplab-nyr0`.
3. Canonical context: `docs/reference/project-map.md`, `docs/reference/product-surface.md`, `docs/reference/maturity-matrix.md`, `docs/MULTI-REPO-WORKFLOW.md`, the 2026-04-26 ChangePassport / Enterprise research documents, plus the **2026-04-27 layering memo v2** at `docs/strategy/2026-04-27-sdp-product-layering-4d.md` and the **council synthesis** at `docs/strategy/council/2026-04-27/synthesis.md`.
4. This document is Discovery output. The linked workstreams are Delivery units.
5. Protocol artifact publishing is not automatic. Publish to the public `sdp` repo only if a later workstream changes `prompts/`, `schema/`, `templates/`, hooks, or harness entrypoints that external users need.

## Problem

The request "global review and refactor the repo" is too broad to execute safely.

It mixes at least five different jobs:

1. product-layering and market framing;
2. public install and Homebrew packaging;
3. Go module, build, dependency, and duplicate-code cleanup;
4. telemetry and consent;
5. explicit debt capture.

Treating those as one refactor would create a large diff with weak acceptance criteria. F150 turns the request into a product architecture decision plus executable readiness lanes.

## AI Fluency 4D Reframe

> **Superseded by [companion memo §"AI Fluency 4D Reframing"](../strategy/2026-04-27-sdp-product-layering-4d.md#ai-fluency-4d-reframing).** The memo is the canonical 4D analysis; it covers Delegation, Description, Discernment, and Diligence per surface with council-approved metrics (incl. evidence-mismatch rate replacing hallucination rate).

Original draft kept for the patch trail:

- **Delegation** lanes: taxonomy, release surface, module path migration, experimental isolation, dependency audit, coverage policy, telemetry consent, Homebrew dry run, product docs, debt ledger. → Memo expands per surface.
- **Description**: prepare selected SDP surfaces for public installation/evaluation by defining layers, boundaries, build scope, coverage, telemetry, debt. → Memo locks internal namespaces (`sdp-pr-gate`, `sdp-edg-*`, `sdp-operator-*`, `sdp-{evidence,policy,modelgw,context,eval}-core`).
- **Discernment**: evidence-driven gates for tests, dependencies, coverage, telemetry, Homebrew. → Memo adds package-level isolation lint, AGENTS.md ≤60 cascade lint, evidence-mismatch metric for governance surfaces.
- **Diligence**: every non-release-ready surface labeled `experimental | beta | ga | retired`. → Memo adds `extractable: yes/no` annotation and per-surface owner / audit / debt protocol.

## Product Layering Decision

> **Superseded by [companion memo §"Revised Layer Taxonomy (v2)"](../strategy/2026-04-27-sdp-product-layering-4d.md#revised-layer-taxonomy-v2).** The memo is the canonical taxonomy. Summary below for the patch trail.

The model: **parallel product surfaces over shared technical substrates, with strict dependency rules**, plus an explicit subordinate Toolbox tier and a stateful Operator Mode that is the default Toolkit Happy Path.

## Layers (summary; canonical version in memo)

The 7-row v2 taxonomy:

1. **SDP Lab** — research and platform workspace; never a customer runtime dependency.
2. **SDP Toolbox** *(NEW row, council-renamed from "Standalone Tools")* — subordinate freemium acquisition collection (`sdp-toolbox-*`); single-purpose utilities under SDP brand. Promotion to a separate product category requires ≥2 external consumers + distinct ICP. Memo §"SDP Toolbox" defines the lifecycle.
3. **SDP Toolkit** — installable developer surface (`sdp` CLI, multi-harness install, adapter generation, `scout`, `metrics`, `index`, `spec`, `bootstrap`, manifest doctor). Useful without ChangePassport, Operator Mode, EDG, or local model routing.
4. **Operator Mode** *(council-reframed)* — default Toolkit Happy Path embodying governed delivery; stateful orchestration layer (`sdp-operator-*`); not a separate paid SKU now, but **provisional pricing hypothesis required before any pilot**, plus an explicit re-evaluation trigger (3+ buyers in isolation, compliance-only buyer, etc.).
5. **ChangePassport (display) / `sdp-pr-gate` (internal namespace, locked)** — merge-readiness product. Schema v1, Evidence Provider API v1, Decision Record v1, override protocol, GitHub PR Gate Loop v1. Paid object: governed readiness decision + override trail + reviewer-readable passport. **Implementation gated on committed pilot.**
6. **Enterprise Delivery Governance** *(council-renamed; was "Enterprise Perimeter Control Plane")* — enterprise-grade governed delivery control plane (`sdp-edg-*` reserved); hypothesis; out of F150 scope. The new name avoids "Perimeter" mispositioning against AppSec.
7. **Shared Substrates** — versioned semver packages: `sdp-evidence-core`, `sdp-policy-core`, `sdp-modelgw-core`, `sdp-context-core`, `sdp-eval-core`. Promotion requires docs, tests, owner, maturity label, release contract.

Dependency-rule highlights (memo holds the full table):

- **Toolbox** must NOT import `internal/sdp-pr-gate/`, `internal/sdp-operator/`, or `internal/sdp-edg/`.
- **`sdp-pr-gate`** must NOT import `sdp_lab` workstreams, Beads, lab-only binaries.
- **EDG** must NOT depend on unversioned lab experiments.
- **Package-level isolation lint** (council-added) enforces these now even though physical repo split is deferred.

## Dependency Rules

| Product surface | May depend on | Must not depend on |
|---|---|---|
| SDP Toolbox | `internal/sdp-toolkit-core/*`, Shared Substrates at pinned semver | `internal/sdp-pr-gate/*`, `internal/sdp-operator/*`, `internal/sdp-edg/*` |
| SDP Toolkit | stable CLI helpers, manifest/adapters, read-only analysis packages | operator Beads workflow, `sdp-pr-gate`, EDG runtime |
| Operator Mode | Toolkit primitives, Beads, evidence, workstreams | `sdp-pr-gate` product schema as a hard dependency |
| `sdp-pr-gate` (ChangePassport) | evidence provider contracts, Git provider adapters, renderer, decision store | `sdp_lab` workstreams, Beads, lab-only binaries |
| Enterprise Delivery Governance | `sdp-pr-gate`, model gateway, telemetry, deployment blueprints | unversioned lab experiments |
| SDP Lab | everything experimental | customer-facing stability claims |

**Council-added: package-level isolation lint** — a CI check (in WS 00-150-04) forbids cross-imports between `internal/sdp-pr-gate/`, `internal/sdp-operator/`, and `internal/sdp-toolkit-core/` (one-way only: Toolkit-core may be imported by others, never the reverse). This makes the future repo split mechanical (`git filter-repo`-grade) without forcing it now.

## Release Surface Policy

F150 treats `sdp` as the first formula target.

The formula should not install every `cmd/sdp-*` binary by default. The release inventory must decide which helpers are:

- bundled and linked;
- bundled but hidden;
- source-only;
- lab-only;
- excluded by build tags;
- retired.

The current module path `sdp_dev` must be replaced. The default migration target is:

```text
github.com/fall-out-bug/sdp_lab
```

Reason: Go source currently lives in `sdp_lab`. Using `github.com/fall-out-bug/sdp` would point Go users at the distilled protocol mirror, not the source module.

## Experimental Isolation

Experimental code should be isolated, not deleted.

Allowed mechanisms:

- release allowlist in GoReleaser;
- explicit build tags such as `sdp_experimental`;
- package-level maturity annotations;
- docs marking lab-only commands;
- separate formula or cask later if needed.

Not allowed:

- hiding compile failures by excluding tests silently;
- deleting integration tests to make gates green;
- making unstable functionality part of the stable formula by accident.

## Coverage Policy

Accepted policy:

- happy paths: `>=80%`;
- GA components: `>=60%`;
- Beta components: `>=50%`;
- Experimental components: measured only when useful, not gate-blocking.

This supersedes stronger blanket claims in older docs. The denominator must be explicit: package, command, product surface, or happy-path scenario.

## Telemetry Policy

Telemetry export must be opt-in.

Default:

- local trace storage may exist;
- no external export;
- no collector endpoint called.

Export requires:

- explicit consent;
- explicit OTEL collector endpoint;
- visible command/config path;
- schema allowlist;
- no content export unless consent level permits it;
- documented disable path.

## Execution Plan

No renumbering. Council added explicit acceptance hints to several lanes.

| WS | Purpose | First useful output | Council acceptance hint (v2) |
|---|---|---|---|
| 00-150-01 | product layering and SKU boundary | this design + memo v2 + synthesis | satisfied by memo v2 + synthesis |
| 00-150-02 | release surface inventory | keep/exclude matrix for CLI/packages | + SDP Toolbox registry: every standalone-utility module with `extractable` flag and 60-line AGENTS.md |
| 00-150-03 | module path migration | `sdp_dev` removed from active Go imports | + AGENTS.md cascade migration subtask: root reduced ≥20% (606 → ≤480), ≥5 modules ≤60 lines, incremental CI lint warn-only |
| 00-150-04 | experimental isolation | release build excludes lab-only code | + package-level isolation lint between `internal/sdp-pr-gate/`, `internal/sdp-operator/`, `internal/sdp-toolkit-core/` |
| 00-150-05 | dependency and duplicate audit | concrete removals or filed debt | unchanged |
| 00-150-06 | coverage policy | maturity-aligned checks/docs | unchanged |
| 00-150-07 | telemetry consent/OTEL | opt-in export contract | unchanged |
| 00-150-08 | Homebrew formula dry run | local formula install/test evidence | unchanged |
| 00-150-09 | product docs alignment | README/product-surface match layers | + surface Operator Mode as default Happy Path; consume outputs of WS-02 / WS-03 |
| 00-150-10 | readiness report | shipped/blocked/deferred ledger | + record internal-namespace lock status and isolation-lint status |

## Non-goals

- Release a tagged version in this workstream.
- Publish to the public `sdp` repo before relevant PRs merge.
- Build a full `sdp-pr-gate` (ChangePassport) implementation before Schema/API v1 are locked.
- Build any Enterprise Delivery Governance component (renamed from "Enterprise Perimeter") before the product boundary is stable.
- Delete experimental code as a shortcut.
- Ship `ChangePassport` to external pilots before Schema v1 + a committed pilot land *(council-added)*.
- Make Standalone Tools / Toolbox a parallel paid product category before promotion criteria are met *(council-added)*.

## Open Decisions (post-council)

1. **(KEPT)** Whether ChangePassport / `sdp-pr-gate` lives in this repo initially or starts as a separate repo when Schema/API v1 begins. **Criterion:** split when Schema v1 + Evidence Provider API v1 + Decision Record v1 freeze AND first committed pilot lands. Package-level isolation enforced from F150 makes the split mechanical.
2. **(RESOLVED)** Homebrew scope: helpers via opt-in formula tap or build tag, **not default**. Defer details to F150-08.
3. **(KEPT)** Long-term Go module path: stay on `github.com/fall-out-bug/sdp_lab` until repo-split events occur (per #1 and Toolbox extractions per memo §"SDP Toolbox").
4. **(RESOLVED)** Release isolation: GoReleaser allowlists are primary; build tags only for compile isolation when import boundaries cannot enforce.

Council-added open items (tracked outside F150, see memo §"Open Items"):

5. Pricing model and willingness-to-pay hypothesis for `sdp-pr-gate` and Operator Mode.
6. SDP brand architecture artifact across Toolkit / Toolbox / ChangePassport / EDG.
7. Evidence persistence architecture decision (storage backend, retention, backup, privacy).
8. Procurement/compliance install profile for dev-led-to-manager-paid path.
9. Competitive positioning artifact vs Copilot Workspace, CodeRabbit, GitLab Duo.

## Success Criteria

- Product layers are understandable without reading historical plans (canonical taxonomy lives in companion memo v2).
- A user can install the first formula without receiving lab-only commands as a product promise.
- Active Go imports no longer use `sdp_dev`.
- Experimental code is isolated from stable release builds.
- Telemetry export is impossible without explicit consent.
- Remaining debt is in Beads with owners and not hidden in prose.
- **Internal namespaces locked** *(council-added)*: `sdp-pr-gate`, `sdp-operator-*`, `sdp-edg-*` reserved across packages, env vars, schemas; display name `ChangePassport` decoupled.
- **Package-level isolation lint green** *(council-added)*: no cross-imports between `internal/sdp-pr-gate/`, `internal/sdp-operator/`, `internal/sdp-toolkit-core/` (except declared one-way Toolkit-core → others).
- **Cascade AGENTS.md migration started** *(council-added)*: root reduced ≥20% (606 → ≤480 lines); at least one module under 60 lines per layer (Toolkit / Operator / Toolbox / substrate); incremental CI lint warn-only enabled.
- **Public docs surface Operator Mode as default Happy Path** *(council-added)*: `product-surface.md` and README align with memo v2 reframing.
- **Hallucination metric replaced** *(council-added)*: `evidence-mismatch rate` is the governance-decision accuracy metric for `sdp-pr-gate`. Pilot vs GA targets explicitly split in docs.
