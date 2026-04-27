---
title: SDP Product Layering — AI Fluency 4D Strategy Memo
status: v3 (post-IIP-council)
owner: Andrei
beads: sdplab-qgq1 (F150-01)
created: 2026-04-27
revised: 2026-04-27 (memo council R1+R2; then IIP council R1+R2)
councils:
  - docs/strategy/council/2026-04-27/synthesis.md (memo v2 layering council)
  - docs/strategy/council/2026-04-27-iip/synthesis.md (IIP — independent-value tools council)
supersedes: §"Layers" of `docs/plans/2026-04-27-f150-product-layering-release-readiness-design.md`
informs: F150 patch, SDP Toolbox manifest (TBD), `sdp-pr-gate` namespace lock (TBD), arch-snap / doc-tracer hypotheses (TBD)
---

# SDP Product Layering — AI Fluency 4D Strategy Memo (v3)

This memo is the deeper companion to [F150 design](../plans/2026-04-27-f150-product-layering-release-readiness-design.md). F150 stays as the executable program plan. This memo answers:

> What are the actual product surfaces, what are their names, how do they relate, what ships first, and how do we keep cold-start cost low across all of them?

It is grounded in the 2026-04-26 ChangePassport council outputs and Enterprise research, and in the current repo reality (`docs/reference/product-surface.md`, `docs/architecture/REPO-BOUNDARY.md`, the F120-F126 Toolkit lane). Memo v1 was challenged through a 5-model llm-council on 2026-04-27 (memo council). Memo v2 was challenged again the same day through a 5-model IIP council focused on tools with independent value (arch-snap, doc-tracer). Both councils' outputs and change ledgers live in `docs/strategy/council/2026-04-27/` and `docs/strategy/council/2026-04-27-iip/`.

## Revision History

- **v1 (2026-04-27)** — initial draft. Reframed F150 layering through 4D, introduced Standalone Tools as a "first-class new category", positioned Operator Mode as "advanced Toolkit feature", named the enterprise slot "Enterprise Perimeter Control Plane", set hallucination rate as a governance metric, deferred all repo-split / namespace decisions.
- **v2 (2026-04-27, post-memo-council)** — applied the 9 consensus changes from the first 5-model council (gemini-3.1-pro / grok-4.20 / kimi-k2.6 / deepseek-v4-pro / qwen3.6-plus). Major deltas:
  - Enterprise slot renamed `Enterprise Delivery Governance` (drops "Perimeter").
  - Internal technical namespace `sdp-pr-gate` locked immediately, decoupled from `ChangePassport` display name.
  - Discernment metric: `hallucination rate` → `evidence-mismatch rate`; pilot vs GA targets split.
  - Standalone Tools repositioned as `SDP Toolbox` (subordinate, freemium acquisition lever); first-class category claim removed.
  - Operator Mode reframed as default Toolkit Happy Path embodying governed delivery, with provisional pricing hypothesis and explicit SKU re-evaluation trigger.
  - Cascade AGENTS.md ≤60 lines: added executable migration plan and incremental CI lint.
  - Wedge B (ChangePassport) gated on committed pilot before parallel resource allocation.
  - Package-level isolation enforced now even though physical repo split is deferred.
  - Workstream acceptance criteria for AGENTS.md migration and SDP Toolbox registry made explicit (no renumbering of 00-150-01..10).
- **v3 (2026-04-27, post-IIP-council)** — applied 9 consensus changes from the second 5-model council (xiaomi/mimo-v2.5-pro / minimax/minimax-m2.7 / kimi-k2.6 / deepseek-v4-pro / qwen3.6-plus, `max_tokens=100000`). The council challenged the proposal to introduce a new "Incubated Independent Products" taxonomy row for tools with value outside SDP (arch-snap, doc-tracer). Major deltas:
  - **IIP is a FLAG inside Toolbox lifecycle, not a new top-level taxonomy row.** Toolbox tools carry `standalone: true | false`; tools with `standalone: true` follow strict IIP rules.
  - **arch-snap and doc-tracer reframed as hypotheses under evaluation, not flagship candidates.** Neither exists in code. Each requires named lead + commercial_hypothesis.md + ≥3 discovery interviews before IIP status.
  - **Standalone go.mod from day one** (mandatory, zero `replace` directives, zero `internal/sdp-*` imports). Distribution shares unified SDP CI during incubation; separate Homebrew formula and isolated CI matrix deferred to extraction.
  - **Manual dependency audit interim**: until WS 00-150-04 lint deploys, every PR runs mandatory manual `go list -deps` + grep audit; extraction is blocked until automated lint passes.
  - **Extraction criteria revised**: technical isolation lint pass + commercial signal (signed LOI / paying customer / co-maintainer-sponsor) + ≥3 discovery interviews + finalized name & license. **Domain dropped** as extraction requirement (only needed for commercial launch later).
  - **Named IIP lead** (individual, not team) required for every IIP epic. No lead = no IIP status.
  - **Brand ownership transfer plan** from inception: permissive license (Apache-2.0/MIT) by default; written commitment in `BRAND.md` to transfer to neutral entity at extraction; co-sponsor option (≥$50K/year or dedicated FTE).
  - **Downgrade criteria** for stalled IIPs: 60d zero external consumers OR 30d no maintainer OR 90d unpatched critical OR 12mo no commercial signal → downgrade to Lab or archive.
  - **`independent_value` flag is verifiable, not self-declared**: derived from machine evidence (`go list -deps` clean) AND a `commercial_hypothesis.md` artifact, not hand-written annotation.

## Cold Start Answers

1. F150 v0 listed five "layers" but conflated *product surfaces* with *operating modes* with *technical substrates*. This memo separates them and reflects the council's reframing of Operator Mode and Standalone Tools.
2. The first paid wedge is ChangePassport (renamed internally `sdp-pr-gate`); it begins parallel resource allocation only after a committed pilot. Toolkit + Toolbox via Homebrew is the parallel free wedge.
3. Russian sovereign / Enterprise Delivery Governance is out of F150 scope. A renamed slot is reserved.
4. Working names (`ChangePassport`, `Enterprise Delivery Governance`) stay working until rename criteria are met. Internal technical namespaces are locked NOW to prevent refactor debt.
5. The architectural meta-rule is the cascade `AGENTS.md` model: every product surface and every Toolbox tool must be cold-startable from `≤60-line root + ≤60-line module` files. Current root AGENTS.md (606 lines) is the migration target. Migration is incremental, CI-warn-then-enforce, scoped inside `00-150-03`.

## Sources

- `/Users/fall_out_bug/Documents/changepassport-manifesto-v2-2026-04-26.md`
- `/Users/fall_out_bug/Documents/changepassport-council-report-2026-04-26.md`
- `/Users/fall_out_bug/Documents/enterprise-perimeter-agentic-delivery-research-2026-04-26.md`
- `/Users/fall_out_bug/Documents/enterprise-perimeter-agentic-delivery-gpt55-analysis-2026-04-26.md`
- `/Users/fall_out_bug/Documents/sdp-strategy-research-ai-fluency-2026-04-26.md`
- `/Users/fall_out_bug/Documents/ai-sdlc-pdlc-vs-sdp-april-2026-report.md`
- `docs/reference/product-surface.md`
- `docs/architecture/REPO-BOUNDARY.md`
- `docs/plans/2026-04-27-f150-product-layering-release-readiness-design.md`
- `docs/strategy/council/2026-04-27/synthesis.md` (R1 + R2 voting record)

## Architectural Meta-Rule: Cascade AGENTS.md ≤60

Every separable surface (product, Toolbox tool, substrate package) MUST be cold-startable from at most:

- root `AGENTS.md` (≤60 lines, repo-wide invariants only);
- module `AGENTS.md` (≤60 lines, surface-specific operating contract).

A worker entering `cmd/sdp-spec/` reads exactly two files to be productive: root + module. No third hop into "you may also want to read".

This rule is the anti-corruption layer for the lab. It forces every promoted surface to declare its boundary in plain text. If a tool cannot fit, it is not yet a separable surface; it stays inside `sdp_lab` as research.

Module `AGENTS.md` MUST declare:

- one-line purpose;
- inputs and outputs (interfaces, files, env);
- runtime dependencies (allowlist; absence = "no other SDP surface required");
- maturity label (`experimental | beta | ga`);
- `extractable: yes | no` (whether the module is a candidate for separate repo);
- quick commands (build, test, run, smoke);
- escalation pointer (where to read more).

### Executable migration plan (added in v2)

The current 606-line root AGENTS.md cannot be slimmed to 60 in one PR. Migration is incremental:

1. **Carve-out** root content into `docs/architecture/PLATFORM-INVARIANTS.md` (long-form architectural rules) and extend `docs/SDP_OPERATOR_WORKFLOW.md` for operator details.
2. **CI lint warn-only**: a check counts non-blank lines in every `AGENTS.md`; warns when >60.
3. **20% line-reduction sprint goal** as a deliverable subtask of `00-150-03` — root AGENTS.md drops below 480 lines by end of F150.
4. **Per-module enforce**: each new `AGENTS.md` written under this memo MUST be ≤60 from day one; the lint becomes blocking for those new files.
5. **Full-cascade enforce** is post-F150. Not a gate for F150 close.

This is the council-required answer to "rule without plan = process theater" (Architect, Critic, Technician).

## AI Fluency 4D Reframing

### Delegation

What can be delegated to agents/tools, what stays human, per surface.

| Surface | Delegated to agents/tools | Stays human |
|---|---|---|
| SDP Lab | research, prompt experiments, dogfood orchestration, eval runs | research direction, kill criteria |
| SDP Toolbox (subordinate to Toolkit) | repo scan, metric extraction, index building, spec recovery, doc tracing, architecture extraction, token accounting, local-model dispatch | tool selection, output interpretation, action |
| SDP Toolkit | bundling and consistent invocation, manifest parity, adapter generation | install decision, config commit |
| Operator Mode (default Toolkit Happy Path, stateful orchestration) | workstream tracking, evidence collection, findings routing, QA hand-off | shape decisions, scope confirmation, override |
| ChangePassport (display) / `sdp-pr-gate` (technical) | scope seeding from PR/issue/labels, evidence ingestion, draft passport, readiness signal | scope confirmation, override with reason, decision |
| Enterprise Delivery Governance (hypothesis) | model routing, context compilation, evidence normalization, audit log emission | policy authoring, override, compliance sign-off |

The operating principle: **delegate observation, retain decision.** No surface promotes a model claim into a pass without a deterministic gate or a human signature.

### Description

Every surface MUST replace free-form chat with a structured input artifact.

| Surface | Input artifact | Output artifact |
|---|---|---|
| SDP Toolbox | tool config + target repo path | structured report (JSON/Markdown) |
| Toolkit | `sdp.manifest.yaml` | adapter set + parity report |
| Operator Mode | feature → workstream → beads issue | workstream verdict, evidence bundle |
| `sdp-pr-gate` (ChangePassport) | PR + issue + labels + evidence events | Passport (Markdown + JSON) + Decision Record |
| Enterprise Delivery Governance | Change Request → Scope Contract → Evidence Plan → Model Routing Plan | ChangePassport + audit trail |

The 60-line cascade rule is itself a Description artifact: it tells humans and agents what each surface does without requiring chat.

### Discernment

How quality is decided per surface. Numbers are pilot-stage targets. GA SLOs are post-F150.

| Surface | Primary metric | Pilot target (post-PMF GA SLO TBD) | Stop / kill |
|---|---|---|---|
| SDP Toolbox tool | install-to-first-useful-output | ≤5 min | tool ignored after 2 weeks |
| Toolkit | install time, manifest parity | ≤30 min, parity green | install requires author help |
| Operator Mode | useful workstream completion rate | ≥70% completion without manual reroute | reviewers bypass evidence |
| ChangePassport / `sdp-pr-gate` | useful decision rate | ≥70% passports drive reviewer action without manual reconstruction | hallucinate-style misclassification of evidence makes reviewers distrust |
| ChangePassport / `sdp-pr-gate` | install + first decision | install ≤30 min, passport ≤60 sec post-checks | install >30 min repeatedly |
| ChangePassport / `sdp-pr-gate` | reviewer time delta | median −20% in 4-week pilot | no time reduction |
| ChangePassport / `sdp-pr-gate` | **evidence-mismatch rate** (passport claim vs ground-truth source) | <5% | systematic mismatch |
| ChangePassport / `sdp-pr-gate` | false-block rate | <5% | blocking PRs that should ship |
| ChangePassport / `sdp-pr-gate` | post-merge incident rate | not above baseline | ready decisions correlate with incidents |
| Enterprise Delivery Governance | install in target perimeter | ≤2-4 weeks | >8-12 weeks per install |
| Enterprise Delivery Governance | useful suggestion rate | ≥30-40% | <30% sustained |

> Council change vs v1: hallucination rate replaced with evidence-mismatch rate. Reason: the merge-readiness product reviews evidence and renders a decision; it is not a content generator. The relevant failure is the passport asserting evidence that does not exist or contradicts ground truth.

Discernment principle: a surface earns "ready" only when it has measurable threshold + stop rule. A surface that cannot define both stays in Lab.

### Diligence

Ownership, audit, perimeter, residency, debt — per surface.

| Surface | Owner | Audit | Perimeter / residency | Debt protocol |
|---|---|---|---|---|
| SDP Lab | research | dogfood logs only | local | open backlog, no SLA |
| SDP Toolbox | per-tool maintainer | tool-local logs | local by default, network only with consent | each tool tracks its own beads epic |
| Toolkit | toolkit maintainer | manifest + adapter parity reports | local | F125, F126 lanes |
| Operator Mode | operator | beads + workstream verdict | local | F091-F096 platform reset |
| ChangePassport / `sdp-pr-gate` | product owner (TBD) | append-only decision/override log, signed | repo-scoped + tenant-scoped | Schema v1 freeze plan + pilot gate |
| Enterprise Delivery Governance | enterprise lead (TBD) | immutable audit + OTEL | air-gapped capable, no-egress mode | dedicated F-track |
| Shared Substrates | substrate owner per package | per-package release notes | n/a | semver, deprecation policy |

Diligence principle: anything ship-facing must have an explicit owner, audit, and debt protocol. If any of three is missing, the surface is Lab.

## Revised Layer Taxonomy (v2 + v3 IIP flag)

| # | Surface | Kind | Working name | Internal namespace | Status today | First commercial role |
|---|---|---|---|---|---|---|
| 1 | SDP Lab | research workspace | `sdp_lab` | `sdp_lab` | active | none — feeds others |
| 2 | SDP Toolbox (with IIP flag) | subordinate freemium funnel for SDP family; tools with `standalone: true` follow IIP rules (see §"Incubated Independent Products") | `SDP Toolbox` for SDP-funnel tools; IIPs use **no** `sdp-` prefix (e.g. `arch-snap`) | `sdp-toolbox-*` for SDP-funnel; bare tool names for IIPs | partial: F120-F124 done as Toolbox; arch-snap / doc-tracer are IIP hypotheses (no code yet) | free dev adoption (Toolbox); independent products at extraction (IIP) |
| 3 | SDP Toolkit | meta-distribution | `sdp` CLI | `sdp` | GA inside `sdp_lab`, F125 in progress | free dev adoption |
| 4 | Operator Mode | default Toolkit Happy Path embodying governed delivery; stateful orchestration layer; not a paid SKU now (provisional pricing required pre-pilot, re-evaluation trigger defined below) | `sdp` operator commands (`orchestrate`, `ready`, `ci-loop`) | `sdp-operator` | GA inside `sdp_lab`; lab-only by REPO-BOUNDARY | team adoption (free with provisional pricing hypothesis attached) |
| 5 | ChangePassport (display) | merge-readiness product | `ChangePassport` (working display) | **`sdp-pr-gate`** (locked from now) | direction; Schema v1 not yet locked | first paid wedge — gated on committed pilot |
| 6 | Enterprise Delivery Governance | enterprise-grade governed delivery control plane (hypothesis; **not** "Perimeter") | TBD | `sdp-edg-*` | hypothesis, out of F150 scope | enterprise paid wedge (separate F-track) |
| 7 | Shared Substrates | versioned packages with semver contracts | individually named | `sdp-evidence-core`, `sdp-policy-core`, `sdp-modelgw-core`, `sdp-context-core`, `sdp-eval-core` | implicit today; not formally versioned | n/a — internal contracts |

### Why this taxonomy beats v1

- **Operator Mode is the default Toolkit Happy Path embodying governed delivery.** It is the strongest existing proof of governed AI delivery in `sdp_lab`. Burying it as "advanced feature" hides our most credible demonstration. It is a stateful orchestration layer — different topology from stateless Toolbox utilities.
- **Standalone Tools repositioned as SDP Toolbox** — explicitly subordinate, freemium acquisition. Promotion to a distinct product category requires 2+ external consumers AND a distinct buyer ICP. This stops process-theater categorization.
- **Enterprise Delivery Governance** replaces "Perimeter Control Plane" — naming aligned with delivery governance, not network appliance. Enterprise wedge is a delivery governance product hosted inside the customer perimeter, not a perimeter security tool.
- **ChangePassport** is display name. Internal technical namespace is `sdp-pr-gate` from day one (see §"Internal Namespace Lock" below).
- **Shared Substrates** are explicitly semver-versioned packages, not vague "technical assets". Promotion criteria below.
- **(v3) IIP is a flag inside Toolbox, not a new taxonomy row.** Tools with `standalone: true` follow strict IIP rules: no `sdp-` prefix from inception, standalone go.mod, named lead, brand transfer plan. Adding a top-level row after every change signals strategic instability; the flag captures the intent without restructuring. See §"Incubated Independent Products" for full rules.

### Naming and identity strategy

- **Working names stay working** until all four rename criteria are met: domain available, no trademark collision, ICP recognizes the name, council/buyer language test passes.
- **SDP-prefixed names** for tooling (`sdp-scout`, `sdp-spec`, `sdp-orchestrate`, `sdp-toolbox-*`). They imply "part of the SDP toolkit family".
- **Independent display names** for products with distinct ICP (`ChangePassport`). Marketing rebrand is permitted; internal namespace is decoupled and locked.
- **Internal namespace decoupling** (council change): the marketing display name is allowed to evolve; the internal technical namespace is locked when the product surface is named for the first time. See §"Internal Namespace Lock".
- **Enterprise Delivery Governance** stays as a category until the first ICP signs and gives it a real name. The enterprise GTM team picks the brand. The internal namespace `sdp-edg-*` is reserved.
- **SDP Toolbox** can have any tool-level name. The rule: tool name must survive without `sdp-` prefix in case the tool is extracted to its own repo.
- **Avoid these names** (per research): "AI software engineer", "local Copilot", "on-prem coding assistant", "sovereign coding agent", and now also "Perimeter Control Plane". They force a feature war or position SDP as something it is not.

## Internal Namespace Lock (added in v2)

Council consensus (Architect + Critic + Technician + Pragmatist): marketing display names are allowed to evolve; internal technical namespaces must be locked early and decoupled.

### Lock list (effective immediately)

| Surface | Display name (working) | Internal namespace (locked) |
|---|---|---|
| ChangePassport | `ChangePassport` | `sdp-pr-gate` |
| Enterprise Delivery Governance | TBD | `sdp-edg-*` (reserved) |
| Operator Mode | Operator Mode | `sdp-operator-*` |
| Shared Substrates | individual | `sdp-{evidence,policy,modelgw,context,eval}-core` |

### Scope of the lock

The internal namespace governs:

- Go package paths (`internal/sdp-pr-gate/...`, `pkg/sdp-evidence-core/...`);
- CLI slugs (`sdp pr-gate ...`);
- GitHub App ID and webhook paths;
- database tables (`sdp_pr_gate_decisions`);
- env vars (`SDP_PR_GATE_*`);
- semver tags (`sdp-pr-gate/v0.1.0`).

The display name (`ChangePassport`) continues to live in:

- README and product-surface docs;
- README banners, blog posts, marketing pages;
- buyer-facing UI strings.

### Rename triggers

If the display name changes, the lock prevents code-side migration. If the technical namespace changes (rare, large reason required), it requires a versioned migration with semver bump and deprecation window.

## SDP Toolbox — narrowed (v2 reposition + v3 narrowing)

v2 repositioned "Standalone Tools" as `SDP Toolbox` (subordinate freemium funnel, not a parallel category). v3 narrows the row further: **only tools whose value is fundamentally tied to SDP onboarding/adoption stay in Toolbox**. Tools with independent value get the IIP flag (see next section) and follow stricter rules.

### Definition

`SDP Toolbox` is a collection of single-purpose utilities under the SDP brand whose primary purpose is to acquire users into the SDP family (Toolkit → Operator Mode → ChangePassport). Each Toolbox tool:

- has a 60-line module `AGENTS.md`;
- has its own tests, CI, maturity label, and owner;
- functions as **freemium acquisition for SDP**, not as a standalone product;
- may carry `extractable: yes` in module AGENTS.md if the tool could one day live in its own repo;
- carries `standalone: false` (default) — see IIP flag below if `standalone: true`.

### Toolbox examples (SDP-funnel tools only after v3 narrowing)

| Tool | Current location | Status | `standalone` | `extractable` |
|---|---|---|---|---|
| `sdp-scout` (repo card) | `cmd/sdp` subcommand | GA | false | yes (low priority) |
| `sdp-metrics` (git process health) | `cmd/sdp` subcommand | GA | false | yes |
| `sdp-index` (codebase memory) | `cmd/sdp` subcommand | GA | false | yes |
| `sdp-spec` (spec recovery) | `cmd/sdp` subcommand | GA | false | yes |
| `sdp-bootstrap` (brownfield setup) | `cmd/sdp` subcommand | GA | false | yes |
| `sdp-doc-sync` (doc drift / staleness) | partial today | beta | false | yes |

Tools previously listed (`doc-tracer`, `arch-snap`, `tok-economy`, `local-model-router`) **moved to the IIP flag** in v3 — see next section. They are NOT Toolbox tools.

### Toolbox-tool promotion criteria (unchanged from v2)

A Toolbox tool may be promoted to a separately distributed product (own landing page, own ICP messaging, possible separate repo) only when ALL of:

1. ≥ 2 external consumers using it weekly;
2. distinct buyer ICP (different from Toolkit / ChangePassport buyer);
3. own metrics dashboard (install, retention, useful-output rate);
4. own substrate stability review passed.

Until that bar is cleared, every Toolbox tool stays subordinate.

### Toolbox-tool dependency rule

A Toolbox tool MUST NOT import:

- `internal/sdp-operator/*` (Operator Mode);
- `internal/sdp-pr-gate/*` (ChangePassport);
- `internal/sdp-edg/*` (Enterprise Delivery Governance, when it exists).

A Toolbox tool MAY import:

- `internal/sdp-toolkit-core/*` (shared CLI helpers);
- a Shared Substrate package (`sdp-evidence-core`, etc.) at a pinned semver.

Enforced via `go vet` allowlist or a custom linter (in `00-150-04` experimental isolation).

## Incubated Independent Products (IIP) — Flag Inside Toolbox (v3, post-IIP-council)

Some tools have value **outside SDP** and would lose adoption if subordinated to SDP funnel framing. Examples (all hypothetical today; no code yet):

- `arch-snap` (architecture extraction from code) — ICPs: due-diligence/M&A buyers, security architects, new CTOs, tech writers.
- `doc-tracer` (docs↔code traceability) — ICPs: docs-as-code shops, compliance teams (FDA/ISO 13485), regulated industries, audit functions.
- `tok-economy` (token accounting + cascade routing primitives) — ICPs: AI platform teams independent of SDP.
- `local-model-router` (vLLM/NIM/Ollama dispatch) — ICPs: any team running local inference.

The IIP council (5 models, 2026-04-27) endorsed this direction with constraints. **IIP is a flag inside Toolbox, NOT a new top-level taxonomy row** — adding rows after every change signals strategic instability; the flag captures intent without restructuring.

### IIP definition

A Toolbox tool carries `standalone: true` when ALL of:

- a `commercial_hypothesis.md` exists in the module documenting target non-SDP ICPs, top 3 competitors, willingness-to-pay range, kill criteria;
- machine-verifiable independence: `go list -deps ./...` shows zero `internal/sdp-*` imports; `go.mod` has zero `replace` directives pointing to monorepo paths;
- a named IIP lead (individual, not team) is assigned and accountable;
- module name has **no** `sdp-` prefix from inception (`arch-snap`, NOT `sdp-arch-snap`);
- module `AGENTS.md` (≤60 lines) is written as if SDP did not exist (no `sdp-` references in cold-start text);
- license is permissive by default (Apache-2.0 or MIT) unless a specific commercial track requires otherwise;
- `BRAND.md` documents the trademark transfer plan (transfer to neutral entity at extraction).

### IIP hypotheses currently under evaluation (NOT flagship)

The two flagship-sounding examples below are **hypotheses, not products today**. Neither exists in code. Each requires named lead + commercial_hypothesis.md + ≥3 discovery interviews before earning IIP status.

| IIP hypothesis | Tool concept | Hypothesized non-SDP ICPs | Status |
|---|---|---|---|
| `arch-snap` | architecture extraction from code | due-diligence / M&A buyers; security architects; new CTOs onboarding to legacy codebases; tech writers | hypothesis only — needs lead, hypothesis doc, discovery |
| `doc-tracer` | docs↔code traceability | docs-as-code shops; compliance (FDA, ISO 13485, GxP); regulated industries; audit functions | hypothesis only — needs lead, hypothesis doc, discovery |

`tok-economy` and `local-model-router` are also IIP candidates but earlier in the funnel (no named hypothesis sponsor yet).

### IIP architectural rules (mandatory from inception)

1. **Standalone go.mod from day one.** Zero `replace` directives. Zero `internal/sdp-*` imports. Each IIP runs `go list -deps ./...` clean. This makes future extraction mechanical.
2. **Independent semver tag prefix** from inception (`arch-snap/v0.1.0`, `doc-tracer/v0.1.0`).
3. **No `sdp-` prefix** in module name, repo path inside monorepo, README banners, or AGENTS.md cold-start text.
4. **Substrate imports only via pinned semver** (`sdp-evidence-core@v0.X.Y`). Substrate AGENTS.md must document any SDP-runtime assumptions; IIP imports require sign-off from substrate owner.
5. **Distribution during incubation**: shares unified SDP CI pipeline (subcommand or build tag dispatch). Separate Homebrew formula, dedicated landing page, isolated CI matrix are **deferred to extraction**.
6. **Manual dependency audit** until WS 00-150-04 lint deploys: every PR runs `make audit-iip-imports` (a Makefile target running `go list -deps` + grep against forbidden packages); CI fails on forbidden imports. After WS 00-150-04 deploys: lint becomes mandatory; manual audit deprecated.

### IIP extraction criteria (council-revised)

To extract an IIP from monorepo to its own repo, ALL of:

- **Technical**: lint pass on isolation; `go mod tidy` clean with no `replace` directives.
- **Demand**: at least ONE of (a) signed LOI / committed paying customer; (b) ≥10 weekly active external users with ≥50% non-SDP attribution by survey; (c) co-maintainer organization or corporate sponsor with ≥$50K/year or ≥1 dedicated FTE.
- **Discovery**: ≥3 documented customer-discovery interviews validating ICP and willingness-to-pay.
- **Identity**: finalized name + license + visual identity. **Domain NOT required** for extraction (only for later commercial launch).

### IIP cap and downgrade

- **Cap of 3 active IIPs simultaneously** (matches incubation throughput; protects engineering bandwidth).
- **Promotion to IIP requires founder/owner approval** (not a default path from Lab).
- **Downgrade** to Lab or archive if ANY of:
  - 60 days with zero external consumers;
  - 30 days without active maintainer;
  - 90 days with unpatched critical security finding;
  - 12 months without commercial signal AND lead cannot justify IIP cost.

### IIP brand and ownership

- **Initial maintainer**: SDP team. **Initial brand holder**: SDP organization (with explicit transfer commitment).
- **Co-sponsor option**: any organization contributing ≥$50K/year OR ≥1 dedicated maintainer-FTE may be listed as co-sponsor with brand-decision input. Documented in `BRAND.md`.
- **At extraction**: brand ownership transfers automatically to a neutral entity (separate legal vehicle, foundation, or new org). SDP receives no preferential marketing rights post-extraction.
- **License**: permissive (Apache-2.0 or MIT) by default. Commercial track may require dual-licensing — decided per IIP at promotion.

### IIP pricing

Each IIP gets its own willingness-to-pay hypothesis when commercial extraction is considered. There is no requirement that IIP pricing align with SDP pricing. Some IIPs may stay free/open-source forever; others may become commercial standalone products with their own pricing model. Each IIP's `commercial_hypothesis.md` documents the pricing-model hypothesis.

### IIP unaddressed risks (tracked outside immediate memo update)

1. **Go import-path contamination**: `github.com/<sdp-lab-org>/sdp_lab/arch-snap` permanently associates the tool with SDP in every Go import statement. Mitigation options to study (separate decision):
   - Use a neutral GitHub org for incubation from day one (separate `incubator-org/arch-snap` while still developed by SDP team).
   - Accept the contamination as an incubation cost; the import path changes at extraction (mechanical via `git filter-repo`).
   - Hybrid: monorepo sub-path during very early phase, neutral org from v0.1 onwards.
2. **Substrate transitive coupling**: substrate packages may carry implicit SDP-runtime assumptions. Mitigation: substrate `AGENTS.md` must document SDP-runtime assumptions; IIP imports require sign-off from substrate owner.
3. **Chicken-and-egg adoption suppression**: incubation environment broadcasts subordination. Mitigation: addressed via the brand-transfer plan and commercial-signal-based extraction; further mitigation possible if neutral-org incubation is taken.
4. **Procurement/compliance friction**: target ICPs (FDA, ISO, M&A) require SOC2, SLAs, indemnification. Permissive OSS defaults are insufficient. Tracked outside memo for IIP-by-IIP commercial track.
5. **Maintainer incentive misalignment**: SDP engineers maintain IIPs they don't own. Mitigation: each IIP epic must include a maintainer-incentive plan before promotion (revenue share, dedicated FTE, sponsor-funded role, etc.). Tracked outside immediate memo update.

### Beads epics for IIPs

Each IIP gets its OWN beads epic (`F-ARCH-SNAP`, `F-DOC-TRACER`) tracked outside F150. F150 only ensures the architectural rules are in place (cascade AGENTS.md, package-level isolation lint via WS 00-150-04, manual audit interim, `extractable` annotation, `standalone` flag mechanism via WS 00-150-02 registry).

**No IIP epic is created until** the IIP has: (a) named lead, (b) `commercial_hypothesis.md`, (c) at least 3 discovery interviews. Until then the IIP candidate is a hypothesis line in this memo.

## Operator Mode — Reframed (council change)

**Operator Mode is the default Toolkit Happy Path embodying governed delivery. It is a stateful orchestration layer. It is NOT a separate paid SKU now, but it requires a provisional pricing hypothesis BEFORE pilot launch and an explicit re-evaluation trigger.**

### Why this reframing

- Critic, Architect: Operator Mode IS the GA governed-delivery surface inside `sdp_lab`. Calling it "advanced Toolkit feature" is misclassification that buries our strongest proof.
- Architect, Technician: Operator Mode is stateful orchestration with isolated dependency graphs. Different topology from stateless Toolbox utilities.
- Pragmatist: deferring SKU decision until after pilot makes willingness-to-pay measurement impossible. Provisional pricing required upfront.
- Philosopher: governance buyer (engineering manager) needs to see Operator Mode prominently to reach the paid `sdp-pr-gate` wedge.

### Provisional pricing hypothesis (required before any pilot)

Before any external pilot of Operator Mode or `sdp-pr-gate`, draft:

- per-active-repo per-month base price hypothesis;
- included monthly governed-decision volume;
- overage pricing per decision;
- expansion path: Operator Mode → ChangePassport → Enterprise Delivery Governance.

The hypothesis is not commitment. It is the measurement instrument that lets the pilot answer "does the buyer value this enough to pay?".

### Re-evaluation trigger for separate SKU

Operator Mode is reconsidered as a paid SKU when ANY of:

- 3+ buyers ask for Operator Mode in isolation (without `sdp-pr-gate`);
- a compliance-only buyer wants workstream evidence without coding agents;
- pilot data shows Operator Mode usage outside the merge-readiness flow;
- internal Architect signal that orchestration topology divergence forces operational separation.

### Implication for F150

- F150-09 (product docs alignment) treats Operator Mode as the prominent Happy Path inside Toolkit (matches `product-surface.md` §"Run Operator Mode");
- the layer count is 7 with kind labels; Operator Mode is still a distinct row with kind = "stateful orchestration layer / default Toolkit Happy Path";
- provisional pricing hypothesis is added as a deliverable inside the future `sdp-pr-gate` pilot prep (separate F-track, not F150).

## Repo Topology — Cascade Plan

### Today

`sdp_lab` is a monorepo. `sdp` is a distilled mirror published via `scripts/sdp-publish.sh`. This stays.

### Near term (within F150)

- root `AGENTS.md` slimmed by 20% via carve-out (see migration plan above);
- each `cmd/sdp-*` and each Toolbox tool gains its own `AGENTS.md` ≤60 lines;
- `extractable: yes/no` annotation added per module;
- Shared Substrates declared explicitly as `pkg/sdp-*-core` packages with semver contracts;
- internal namespace `sdp-pr-gate` reserved and used in any new code under `internal/sdp-pr-gate/`;
- **package-level isolation enforced now** (council change): a CI lint check forbids cross-imports between `internal/sdp-pr-gate/`, `internal/sdp-operator/`, and `internal/sdp-toolkit-core/` (one-way: Toolkit-core may be imported by others, never the reverse).

### Mid term (post F150)

- ChangePassport: when Schema v1 + Evidence Provider API v1 + Decision Record v1 freeze AND committed pilot lands, evaluate splitting `internal/sdp-pr-gate` into its own repo (`fall-out-bug/sdp-pr-gate` or branded variant). Because of package-level isolation enforced from F150, the split is mechanical (`git filter-repo`).
- High-adoption Toolbox tools: extract to own repo when promotion criteria are met (≥2 external consumers + distinct ICP).

### Far term (only if commercial pull)

- Enterprise Delivery Governance: separate repo organization, separate SemVer line, separate compliance posture. No commitment now.

### Cascade `AGENTS.md` topology

```
/AGENTS.md                          ≤60 lines, repo-wide invariants (target: by end of F150 down to ≤480 from 606)
/cmd/sdp/AGENTS.md                  ≤60 lines, Toolkit operating contract
/cmd/sdp-orchestrate/AGENTS.md      ≤60 lines, Operator Mode commands
/cmd/sdp-evidence/AGENTS.md         ≤60 lines, evidence CLI
/internal/sdp-pr-gate/AGENTS.md     ≤60 lines, ChangePassport runtime (when exists)
/internal/sdp-operator/AGENTS.md    ≤60 lines, Operator Mode runtime
/pkg/sdp-evidence-core/AGENTS.md    ≤60 lines, substrate contract
...
```

Cold start = root AGENTS.md + the module AGENTS.md the worker is about to touch. Two files. Always.

## Wedge Ordering — Recommendation (council-revised)

Two parallel wedges, sequenced. Wedge B is gated on a committed pilot (council change).

1. **Wedge A — free / dev adoption (now, parallel to F150).**
   Toolkit + selected SDP Toolbox tools, distributed via Homebrew (F150-08).
   Goal: developer trust, dogfood breadth, brand awareness, funnel into Wedge B.
   Paid object: none.
   Risk: this wedge alone does not validate commercial PMF.

2. **Wedge B — first paid (after Schema v1 lock AND committed pilot).**
   `sdp-pr-gate` GitHub PR Gate Loop v1 (per ChangePassport manifesto v2). Marketed as `ChangePassport`.
   Goal: first paying ICP — boutique consulting / agency, 10-50 engineers, ≥8 AI-assisted PRs/week.
   Paid object: governed readiness decision + override trail + reviewer-readable passport.
   Prerequisites:
   - Schema v1 + Evidence Provider API v1 + Decision Record v1 + Override protocol (manifesto v2 §"Build Next");
   - **at least one ICP committed in writing to a 4-week paid pilot** OR signed LOI with revenue commitment OR explicit founder pre-build decision (council change).
   Until any one of those triggers, Wedge B implementation stays at design level (Schema v1 etc.) and does NOT take parallel implementation resources.

3. **Wedge C — enterprise (later, separate F-track).**
   Enterprise Delivery Governance on top of GitLab Self-Managed.
   Goal: regulated enterprise pilot.
   Out of F150 scope.

### What this means for F150

F150-08 (Homebrew dry run) stays. It serves Wedge A.

F150 does NOT ship `sdp-pr-gate`. It keeps the boundary clean so Wedge B is unblocked when the pilot signals: cascade AGENTS.md rule, dependency rules, package-level isolation, Shared Substrates versioning, internal namespace lock. `sdp-pr-gate` implementation is a separate F-track that starts when Schema v1 + committed pilot are both in.

### What this means for product framing

Public messaging during F150 must NOT promise ChangePassport as "available". Per manifesto v2: it is product direction, not GA. The current `product-surface.md` line ("ChangePassport — Product direction, separate boundary") stays correct.

## Sovereign / Enterprise Delivery Governance — Reserved Slot

Per user direction, this is a separate track for several future epics.

What the slot guarantees in F150:

- the layer model has explicit row 6 with kind = "hypothesis", named **Enterprise Delivery Governance** (not "Perimeter Control Plane");
- internal namespace `sdp-edg-*` reserved;
- dependency rules already block lab-only experiments from entering the EDG product;
- Shared Substrates that EDG will need (model gateway, context compiler, evidence provider mesh, OTEL) are kept versionable in F150;
- no implementation, no naming finalization, no commercial promise during F150.

What is explicitly NOT in F150:

- adapters for GigaChat / YandexGPT / MWS;
- model routing logic;
- on-prem deployment blueprints;
- ICP qualification.

Those go to a future F-track (`F-EDG-*`) when an enterprise pilot is on the table.

## Risks Surfaced By Council (added in v2)

The council surfaced risks the v1 memo did not address. The author accepts these and tracks them outside F150.

1. **No pricing model / willingness-to-pay hypothesis** for ChangePassport (Pragmatist). Mitigation: provisional pricing hypothesis required before any pilot; tracked under future `F-PR-GATE-*` track.
2. **No validated buyer demand** for paid ChangePassport wedge (Critic). Mitigation: pilot gate added to Wedge B (above).
3. **Procurement / compliance friction** for dev-led adoption converting to manager-paid (Pragmatist). Mitigation: design `sdp-pr-gate` install profile that survives basic security review (no egress by default, scoped GitHub App permissions); future track.
4. **Competitive moat erosion window** — Copilot Workspace, CodeRabbit, GitLab Duo are commoditizing PR review/governance (Pragmatist). Mitigation: SDP differentiator is agent-neutral cross-tool evidence + override trail + Operator Mode + Toolbox; tracked as a positioning artifact, separate F-track.
5. **CI matrix and artifact registry proliferation** across Toolkit (Homebrew), Toolbox, `sdp-pr-gate` (GitHub App) (Technician). Mitigation: shared reusable workflows for Toolkit + Toolbox; `sdp-pr-gate` may diverge later. F150-08 covers Homebrew; CI consolidation is a separate engineering F-track.
6. **Schema v1 freeze collision with module path migration** — `00-150-03` may break `sdp-evidence-core` after Schema v1 freeze (Technician). Mitigation: Schema v1 freeze is post-F150 by design; `00-150-03` lands first; substrates carry semver from F150 onward.
7. **Evidence persistence architecture undefined** — no decision on git LFS vs object storage vs MCP server vs SQLite, no retention/backup/privacy policy (Technician). Mitigation: substrate `sdp-evidence-core` design doc must answer this before Schema v1; tracked as a future F-track prerequisite.
8. **SDP brand architecture missing** — Toolkit, Toolbox, ChangePassport, future EDG lack a coherent brand family; ICPs will be confused (Philosopher). Mitigation: explicit brand architecture artifact required before first external launch; tracked outside F150.
9. **Governance buyer blind spot** — if Operator Mode is buried, engineering managers who buy governance will not reach the paid `sdp-pr-gate` wedge (Philosopher). Mitigation: Operator Mode reframing above; F150-09 docs alignment surfaces it as default Happy Path.

## Discernment Metrics — F150 Acceptance Bar

F150 itself ships when:

| Metric | Target | Source |
|---|---|---|
| Cascade AGENTS.md ≤60 lines | adopted for: root reduced ≥20% (606 → ≤480), `cmd/sdp` ≤60, one Toolbox tool ≤60, one substrate ≤60 | new |
| Module `extractable` annotation | present on every `cmd/sdp-*` and every Toolbox tool | new |
| Active Go imports use `sdp_dev` | 0 occurrences | F150-03 |
| Experimental code in stable Homebrew formula | 0 binaries | F150-04, F150-08 |
| Telemetry export without consent | 0 paths | F150-07 |
| Coverage policy | maturity-aligned thresholds in CI | F150-06 |
| Public docs match v2 layer taxonomy + Operator Mode reframing | yes | F150-09 |
| Open beads debt without owner | 0 | F150-10 |
| Internal namespace lock implemented | `sdp-pr-gate`, `sdp-operator`, `sdp-edg-*` reserved across packages, env vars, schemas | new |
| Package-level isolation lint | green; no cross-imports between `internal/sdp-pr-gate/`, `internal/sdp-operator/`, `internal/sdp-toolkit-core/` (except declared one-way) | new |

This is the F150 release readiness contract. If any row is red, F150 does not close.

## F150 Patch List (council-aligned)

The following changes are proposed for `docs/plans/2026-04-27-f150-product-layering-release-readiness-design.md`. Mechanical edits the F150 owner can apply.

1. **§"Layers"**: replace the 6-layer list with a one-line reference to this memo's §"Revised Layer Taxonomy (v2)"; keep the dependency rules table.
2. **§"Layers" / SDP Operator Mode**: change kind from "Role" to "default Toolkit Happy Path embodying governed delivery; stateful orchestration layer; not a separate SKU now, provisional pricing required before pilot, re-evaluation trigger defined".
3. **§"Layers"**: insert new row before SDP Toolkit: "SDP Toolbox — subordinate freemium acquisition collection; see this memo §"SDP Toolbox"".
4. **§"Layers" / Shared Substrates**: replace prose with reference to this memo §"Shared Substrates" and require versioned package contracts (semver) — substrate names locked: `sdp-evidence-core`, `sdp-policy-core`, `sdp-modelgw-core`, `sdp-context-core`, `sdp-eval-core`.
5. **§"Layers" / Enterprise Perimeter Control Plane**: rename to **Enterprise Delivery Governance**. Remove all "Perimeter" references in F150 design doc.
6. **§"Open Decisions"**:
   - #2 (Homebrew installs only `sdp` or also helpers) — RESOLVED: helpers via opt-in formula tap or build tag, not default. Defer details to F150-08.
   - #4 (build tags vs GoReleaser allowlists) — RESOLVED: GoReleaser allowlists are primary; build tags only for compile isolation when import boundaries cannot enforce.
   - #1 (ChangePassport in this repo or new repo) — KEPT, with criterion: split when Schema v1 + Evidence Provider API v1 + Decision Record v1 freeze AND first committed pilot lands. Package-level isolation enforced from F150 so split is mechanical.
   - #3 (long-term Go module path) — KEPT, with criterion: stay on `github.com/fall-out-bug/sdp_lab` until repo split events occur.
7. **§"AI Fluency 4D Reframe"**: replace with a one-line pointer to this memo (this memo is the authoritative 4D reframing, v2).
8. **§"Execution Plan"**: do NOT renumber the 10 workstreams. Add explicit acceptance criteria per workstream:
   - WS 00-150-02 (release surface inventory) — adds deliverable "SDP Toolbox registry: every standalone-utility module with `extractable` flag, `standalone` flag, and 60-line AGENTS.md".
   - WS 00-150-03 (module path migration) — adds CI-gated subtask "AGENTS.md cascade migration: root reduced ≥20%, ≥5 modules ≤60 lines, incremental CI lint warn-only".
   - WS 00-150-04 (experimental isolation) — explicitly enforces package-level isolation lint between `internal/sdp-pr-gate/`, `internal/sdp-operator/`, `internal/sdp-toolkit-core/`. **(v3)** Adds Makefile target `audit-iip-imports` running `go list -deps` against forbidden-package list; runs in CI; fails on forbidden imports.
   - WS 00-150-09 (product docs alignment) — consumes outputs from WS 02 and 03; surfaces Operator Mode as default Happy Path.
9. **§"Non-goals"**: append "Implement `sdp-pr-gate` runtime in F150" and "Build any Enterprise Delivery Governance component in F150" and "Ship `ChangePassport` to external pilots before Schema v1 + committed pilot". **(v3)** Append "Promote any IIP candidate (arch-snap, doc-tracer) to active IIP status without named lead + commercial_hypothesis.md + ≥3 discovery interviews".
10. **§"Success Criteria"**: append the F150 acceptance bar from this memo §"Discernment Metrics — F150 Acceptance Bar" — including the new internal namespace lock and package-level isolation rows. **(v3)** Append "IIP audit Makefile target operational; `standalone` flag mechanism documented in `00-150-02` Toolbox registry".

## Workstream Hints (no re-numbering inside F150)

Mapping the existing 10 workstreams to layers:

| WS | Primary layer affected | New acceptance hint (council change) |
|---|---|---|
| 00-150-01 | All — taxonomy decision | this memo + synthesis |
| 00-150-02 | Toolkit, SDP Toolbox | + Toolbox registry deliverable |
| 00-150-03 | All Go modules | + AGENTS.md cascade migration subtask |
| 00-150-04 | Lab boundary vs Toolkit / Toolbox | + package-level isolation lint |
| 00-150-05 | Shared Substrates, Toolbox | unchanged |
| 00-150-06 | All maturity-labeled surfaces | unchanged |
| 00-150-07 | Toolkit, future EDG | unchanged |
| 00-150-08 | Toolkit Homebrew formula | unchanged |
| 00-150-09 | All — product docs | + surface Operator Mode as default Happy Path |
| 00-150-10 | All — debt ledger | unchanged |

## Open Items — to Resolve Outside F150

1. ChangePassport / `sdp-pr-gate` repo split: separate F-track once Schema v1 freezes AND committed pilot lands.
2. Enterprise Delivery Governance: separate F-track (multiple epics) when ICP signs.
3. Toolbox extraction events: each tool tracked individually when promotion criteria are met.
4. Naming finalization for ChangePassport display and Enterprise Delivery Governance: brand/legal/ICP work, not F150 scope.
5. Operator Mode UX surfacing: how to expose advanced operator commands as Happy Path without confusing first-time Toolkit users; F125 lane.
6. **Pricing model and willingness-to-pay hypothesis** for `sdp-pr-gate` and Operator Mode (memo-council-added).
7. **SDP brand architecture artifact** (memo-council-added).
8. **Evidence persistence architecture decision** (memo-council-added).
9. **Procurement/compliance install profile** for dev-led-to-manager-paid path (memo-council-added).
10. **Competitive positioning artifact** vs Copilot Workspace, CodeRabbit, GitLab Duo (memo-council-added).
11. **(v3, IIP)** `arch-snap` IIP hypothesis: needs named lead + `commercial_hypothesis.md` + ≥3 discovery interviews before earning `standalone: true` and a beads epic. Until then it remains a hypothesis line in this memo.
12. **(v3, IIP)** `doc-tracer` IIP hypothesis: same gate as arch-snap. Likely candidate ICPs: docs-as-code shops, regulated industries (FDA, ISO 13485, GxP), audit/compliance.
13. **(v3, IIP)** `tok-economy` and `local-model-router`: even earlier IIP candidates; need full hypothesis package before they're considered.
14. **(v3, IIP)** Go import-path contamination decision: neutral GitHub org from inception vs incubator-org-at-v0.1 vs accept-during-incubation. Architect, Critic, Philosopher all flagged this as the highest unaddressed structural risk.
15. **(v3, IIP)** Substrate transitive coupling audit: substrate `AGENTS.md` must document SDP-runtime assumptions; IIP imports require sign-off from substrate owner.
16. **(v3, IIP)** Maintainer incentive plan template (revenue share, dedicated FTE, sponsor-funded role) before any IIP promotion.

## Preserved Minority Reports

### Memo council (v2)

- **Critic (REJECT overall in R2)**: even with the 9 consensus changes, the document still carries unsubstantiated commercial assumptions (willingness-to-pay, competitive moat). Position recorded; not blocking unless a 10th change adds a hard prerequisite of "commercial proof point committed before any Wedge B implementation". Author judgment: the new pilot gate (Wedge B blocked until a committed pilot signs) substantially addresses Critic's core concern; the gap closes when the pilot lands.
- **Architect on C12**: hard REJECT — Operator Mode classified as "Toolkit Happy Path" is "an architectural lie". Position recorded; partially addressed by reframing as "default Toolkit Happy Path embodying governed delivery; stateful orchestration layer". Architect would still prefer renaming the layer to make the orchestration nature explicit. Author decision: keep current naming, revisit if a buyer signal or operational event forces it.

### IIP council (v3)

- **Critic (partial REJECT in R2)**: 7 of 12 D-claims still rated WEAK. Critic accepts with changes overall, but signals that even with the consensus changes the proposal commits non-trivial engineering investment for non-existent products. Position recorded; the named-lead + commercial-signal-at-extraction + downgrade-criteria combination substantially closes the gap. If after 90 days neither arch-snap nor doc-tracer has a named lead with discovery interviews, Critic's position would crystallize to a hard reject. Author judgment: the gate works; we will know within 90 days whether the IIP framework was right.
- **All 5 roles flagged Go import-path contamination** as the highest unaddressed structural risk (`github.com/<sdp-lab-org>/sdp_lab/arch-snap` permanently associates with SDP). This is recorded as Open Item 14 and requires a separate decision before either IIP earns active status.

## Decision Trail

- 2026-04-27 (morning): memo v1 authored as 4D companion to F150 design.
- 2026-04-27 (afternoon): 5-model memo council R1+R2 (gemini-3.1-pro / grok-4.20 / kimi-k2.6 / deepseek-v4-pro / qwen3.6-plus). Synthesis at `docs/strategy/council/2026-04-27/synthesis.md`; memo v2 published incorporating 9 consensus changes.
- 2026-04-27 (afternoon): user pushback "what about tools that live great OUTSIDE SDP — Архитектор, Трассировка". Author proposed Option D (incubate-then-spin-out).
- 2026-04-27 (evening): 5-model IIP council R1+R2 with user-selected models (xiaomi/mimo-v2.5-pro / minimax/minimax-m2.7 / kimi-k2.6 / deepseek-v4-pro / qwen3.6-plus, `max_tokens=100000`). Synthesis at `docs/strategy/council/2026-04-27-iip/synthesis.md`; memo v3 published incorporating 9 IIP consensus changes (IIP as flag inside Toolbox; standalone go.mod; manual audit interim; commercial-signal extraction; named lead; brand transfer plan; downgrade criteria; verifiable independence; reframe arch-snap/doc-tracer as hypotheses).
- 2026-04-27: F150-01 (`sdplab-qgq1`) acceptance criterion satisfied by memo v3 + F150 patch.
- Next: F150 owner applies patch; F150-02..10 proceed under v3 taxonomy with IIP architectural rules attached. Each IIP candidate stays a hypothesis until named lead + commercial_hypothesis.md + ≥3 discovery interviews land.
