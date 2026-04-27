---
title: SDP Product Layering Council — R1+R2 Synthesis
date: 2026-04-27
input: ../../2026-04-27-sdp-product-layering-4d.md (memo v1 claims)
roles_models:
  - Architect: google/gemini-3.1-pro-preview
  - Critic: x-ai/grok-4.20
  - Technician: moonshotai/kimi-k2.6 (retried with max_tokens=12000)
  - Philosopher: deepseek/deepseek-v4-pro
  - Pragmatist: qwen/qwen3.6-plus
output: memo v2 + F150 patch list update
---

# Council Synthesis — SDP Product Layering, 2026-04-27

This synthesis aggregates the R1 (Socratic) and R2 (Council) outputs from five non-Anthropic, non-OpenAI models. The author commits to changing memo v1 where ≥3 roles converge on a specific revision, while preserving minority reports.

## Voting record (R2 final verdicts)

| Claim | Architect | Critic | Technician | Philosopher | Pragmatist | Net |
|---|---|---|---|---|---|---|
| C1 Operator Mode = Toolkit feature | AWR | REJECT | AWR | AWR | AWR | revise |
| C2 Standalone Tools = first-class category | AWR | REJECT | AWR | AWR | AWR | revise |
| C3 Cascade AGENTS.md ≤60 = meta-rule | AWR | REJECT | AWR | ACCEPT | ACCEPT | revise (add migration plan) |
| C4 Two parallel wedges | AWR | AWR | ACCEPT | ACCEPT | AWR | revise (pilot gate) |
| C5 Enterprise Perimeter slot | AWR | AWR | ACCEPT | AWR | AWR | revise (rename) |
| C6 Sovereign track separate | ACCEPT | ACCEPT | ACCEPT | ACCEPT | ACCEPT | accept |
| C7 ChangePassport working name | AWR | AWR | AWR | ACCEPT | AWR | revise (lock internal namespace) |
| C8 Substrates as semver packages | ACCEPT | ACCEPT | ACCEPT | ACCEPT | ACCEPT | accept |
| C9 Repo split = downstream event | AWR | ACCEPT | AWR | ACCEPT | ACCEPT | revise (logical isolation now) |
| C10 Keep 10 workstreams, no renumber | REJECT | REJECT | AWR | ACCEPT | AWR | revise (explicit acceptance criteria) |
| C11 Pilot metrics per surface | AWR | REJECT | AWR | AWR | AWR | revise (evidence-mismatch, split pilot/GA) |
| C12 Operator Mode = Toolkit Happy Path | REJECT | REJECT | ACCEPT | AWR | AWR | revise (provisional SKU + re-eval trigger) |

Legend: AWR = ACCEPT WITH REVISION.

Final overall: 4 of 5 ACCEPT WITH CHANGES; 1 (Critic) REJECT, with criticisms substantively aligned with the changes others propose.

## Consensus changes (≥3 roles agree)

The author accepts the following changes for memo v2.

### 1. Rename "Enterprise Perimeter Control Plane" to "Enterprise Delivery Governance"

Pushed by: Architect, Critic, Philosopher, Pragmatist.

Reason: "Perimeter" implies network security or appliance, mispositioning the future product against AppSec rather than delivery governance. The slot is reserved for the agent-neutral governed delivery layer; the name must reflect that.

Action: replace all "Enterprise Perimeter" references in memo and F150 patch list with "Enterprise Delivery Governance" (working hypothesis name; final brand TBD when ICP signs).

### 2. Lock internal technical namespace immediately, decoupled from display name

Pushed by: Architect, Critic, Technician, Pragmatist.

Reason: keeping `ChangePassport` as a fluid display name while letting it bleed into code, schemas, package paths, GitHub App IDs, and database tables creates irreversible refactor debt when (not if) the marketing rename happens. Lock the technical namespace now.

Action: introduce internal namespace `sdp-pr-gate` (manifesto v2 explicitly uses "GitHub PR Gate Loop" so the namespace matches existing language). All technical artifacts (Go packages, schemas, CLI slugs, GitHub App IDs, database tables, env vars) adopt `sdp-pr-gate` or a close variant. `ChangePassport` is the marketing/display name only; rename criteria stay.

### 3. Replace "hallucination rate" with "evidence-mismatch rate" in governance metrics

Pushed by: Architect, Technician, Philosopher, Pragmatist.

Reason: the merge-readiness product reviews evidence and renders a decision. It does not generate code or text where "hallucination" is the relevant failure mode. The right metric is whether the rendered passport accurately reflects observed evidence (no false claims about what tests passed, what scanners found, what reviewers said).

Action: in the discernment metrics table, swap `hallucination rate <5%` for `evidence-mismatch rate <5%` (claims in passport vs ground-truth evidence sources). Split pilot-stage targets from future GA SLOs explicitly.

### 4. Reposition Standalone Tools as subordinate "SDP Toolbox" / acquisition levers

Pushed by: Architect, Critic, Philosopher, Pragmatist.

Reason: calling them a "first-class new product category" without isolated packaging, semver, telemetry, external validation, or distinct ICP is brand fragmentation and process theater. They are top-of-funnel acquisition levers for the broader SDP family — useful, but subordinate.

Action: rename to `SDP Toolbox` (or `SDP Utilities`), reframe as "single-purpose utilities under the SDP brand functioning as freemium acquisition for Toolkit and ChangePassport". Keep the 60-line module rule and the extraction lifecycle (those are correct). Drop the "first-class product category" claim. Promotion to a separate product category requires: 2+ external consumers AND an isolated buyer ICP.

### 5. Reframe Operator Mode: default Toolkit Happy Path embodying governed delivery, with provisional SKU + re-evaluation trigger

Pushed by: Critic (REJECT both C1 and C12), Architect (REJECT C12, AWR C1), Philosopher (AWR with prominence), Technician (AWR architectural distinct), Pragmatist (AWR provisional pricing).

Reason: Operator Mode today IS the GA governed-delivery surface inside `sdp_lab`. Calling it "advanced Toolkit feature" buries the strongest existing proof point of governed delivery. It is also stateful orchestration (different topology from stateless Toolkit utilities). At the same time, the council does not endorse making it a paid SKU now (no validated buyer for Operator Mode in isolation).

Action:
- Position Operator Mode as the **default Toolkit Happy Path embodying governed delivery** (matching `product-surface.md` Happy Path 3).
- Acknowledge it as a stateful orchestration layer, not a stateless utility. Maintain isolated dependency graph.
- Do NOT mark as separate SKU now. But DO draft a provisional pricing hypothesis before any external pilot, so willingness-to-pay can be measured.
- Add explicit re-evaluation trigger: if 3+ buyers ask for Operator Mode in isolation OR a compliance-only buyer wants workstream evidence without coding agents, re-evaluate as standalone SKU.

### 6. Add executable migration plan to cascade AGENTS.md rule

Pushed by: Architect, Critic, Technician (the engineering-leaning roles).

Reason: a ≤60-line rule with a 606-line baseline and no migration plan is purity theater. PRs will fail or developers will route around it.

Action:
- Add: incremental CI linting (warn-only first, then enforce per-module after each module migrates).
- Add: 20% root-AGENTS.md line-reduction sprint goal as a deliverable inside `00-150-03` (module path migration) or a sibling subtask.
- Add: explicit carve-out plan: root → root + `docs/architecture/PLATFORM-INVARIANTS.md` + `docs/reference/operator-handbook.md` (which already exists for some content).
- Keep the rule as architectural target; mark it non-blocking for F150 close, but demand attached funded execution.

### 7. Add gate to Wedge B (ChangePassport): committed pilot before parallel resource allocation

Pushed by: Critic, Pragmatist.

Reason: declaring a parallel paid wedge without a validated buyer is commercially reckless. Replacement risk (70-85% of SDP can be replaced today by GitHub + Spec Kit/Kiro + Codex/Claude + CodeRabbit/Sonar) is unmitigated.

Action: Wedge B implementation begins only after one of:
- one ICP commits in writing to a 4-week paid pilot;
- a signed LOI with revenue commitment;
- explicit board/founder decision to pre-build the wedge despite no demand.

Until one of those triggers, Wedge B work stays at "Schema v1 + Evidence Provider API v1 + Decision Record v1 + override protocol" design level only.

### 8. Add explicit acceptance criteria for AGENTS.md migration and Standalone Tools registry within existing workstreams

Pushed by: Architect (REJECT C10), Critic (REJECT C10), Technician (AWR), Pragmatist (AWR).

Reason: the original C10 wanted to fold these into `00-150-09` (docs alignment) — Architect and Critic argue this is hiding architectural work in a docs bucket.

Action: keep 10 workstreams but attach explicit acceptance criteria:
- `00-150-03` (module path migration): adds CI-gated subtask "AGENTS.md cascade migration: at least root + 5 modules under 60 lines, incremental CI lint warn-only".
- `00-150-02` (release surface inventory): adds deliverable "SDP Toolbox registry: list every standalone-utility module with `extractable` flag and 60-line AGENTS.md".
- `00-150-09` (product docs alignment): consumes the above outputs, does not own them.

### 9. C9 — enforce package-level isolation NOW even though physical split is deferred

Pushed by: Architect, Technician.

Reason: deferring the physical split is fine, but only if the future split is mechanical (`git filter-repo`-grade). That requires zero cross-imports between ChangePassport and Toolkit internals from day one.

Action: add to memo §"Repo Topology": package-level isolation enforced via `internal/sdp-pr-gate/` isolation rule and a CI lint check forbidding cross-imports. Update F150 patch list.

## Risks surfaced by council that memo v1 fails to address

The author accepts these risks and adds a dedicated section in memo v2.

1. **No pricing model / willingness-to-pay hypothesis** for ChangePassport (Pragmatist).
2. **No validated buyer demand** for paid ChangePassport wedge — replacement risk 70-85% remains (Critic).
3. **Procurement / compliance friction** for dev-led adoption converting to manager-paid (Pragmatist).
4. **Competitive moat erosion window**: Copilot Workspace, CodeRabbit, GitLab Duo are commoditizing PR review/governance (Pragmatist).
5. **CI matrix and artifact registry proliferation** across Toolkit (Homebrew), Toolbox, ChangePassport (GitHub App) — no shared build matrix specified (Technician).
6. **Schema v1 freeze collision with module path migration** — `00-150-03` may break `sdp-evidence-core` after Schema v1 freeze (Technician).
7. **Evidence persistence architecture undefined** — no decision on git LFS vs object storage vs MCP server vs SQLite, no retention/backup/privacy policy (Technician).
8. **SDP brand architecture missing** — Toolkit, Toolbox, ChangePassport, future Enterprise lack a coherent brand family; ICPs will be confused (Philosopher).
9. **Governance buyer blind spot** — if Operator Mode is buried, engineering managers who buy governance will not reach the paid ChangePassport wedge (Philosopher).

## Preserved minority reports

- **Critic (REJECT overall)**: even with all 9 consensus changes applied, the document still carries unsubstantiated claims about willingness-to-pay and competitive moat. Position recorded; not blocking unless a 10th change is added (commercial proof point as a hard prerequisite).
- **Architect on C12**: REJECT (hard) — Operator Mode classified as "Toolkit Happy Path" is "an architectural lie". Position recorded; partially addressed by reframing (now: stateful orchestration layer, not just feature) but Architect would still prefer it called an explicit orchestration surface. Author decision: accept "stateful orchestration layer" framing without renaming the layer; will revisit if a buyer signal emerges.

## Decisions not changed

- **C6** Sovereign track separate (5/5 ACCEPT).
- **C8** Substrates as semver packages (5/5 ACCEPT).
- Working-name policy for ChangePassport (display) and Enterprise Delivery Governance (slot) — kept as working names with rename criteria.
- 60-line module AGENTS.md target — kept as architectural target.
- The two-wedge ordering (Toolkit/Toolbox first as free, ChangePassport as first paid) — kept, with pilot gate added.

## Final action

- Update memo to v2 in place at `docs/strategy/2026-04-27-sdp-product-layering-4d.md`.
- Update F150 patch list inside the memo to reflect council changes.
- Preserve original v1 in git history (one commit before this synthesis lands).
- This synthesis document is canonical record; raw R1+R2 outputs preserved alongside.

## Self-check via 4D after the council

- **Delegation**: I delegated challenge to non-Anthropic, non-OpenAI models. I did not delegate keep/kill voting to a single model. Synthesis is human-authored.
- **Description**: input was 12 atomic claims (C1-C12). Output: structured verdicts per role, signed by model name + timestamp.
- **Discernment**: ≥3-of-5 convergence is the strong-signal threshold; minority preserved with rationale; domain vetoes (Architect on C10/C12, Critic on C1/C3/C10/C11/C12) recorded.
- **Diligence**: raw outputs versioned in `docs/strategy/council/2026-04-27/r{1,2}-{role}.{md,json}`; one model failure (Technician kimi-k2.6 truncated by reasoning budget) caught and retried with `max_tokens=12000`; record of fallback policy preserved in `retry_technician.py`.
