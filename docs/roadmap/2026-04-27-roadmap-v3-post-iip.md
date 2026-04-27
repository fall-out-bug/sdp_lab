---
title: SDP Roadmap v3 — Post-IIP Council
date: 2026-04-27
status: companion update to canonical ROADMAP.md
informs: ROADMAP.md (entries F151..F160), F150 acceptance criteria
council: docs/strategy/council/2026-04-27-iip/synthesis.md
memo: docs/strategy/2026-04-27-sdp-product-layering-4d.md (v3)
---

# SDP Roadmap v3 — Post-IIP Council (2026-04-27)

This document is the dated change-set that translates memo v3 (post-IIP-council) into trackable epics with beads. It is a companion to canonical [ROADMAP.md](ROADMAP.md), not a replacement.

## What's new vs prior roadmap

1. F150 still the active program (in progress); v3 acceptance criteria attached.
2. Ten new epic candidates (F151..F160) added, sequenced by gating dependencies.
3. arch-snap and doc-tracer are tracked as **gated IIP hypotheses**, not active products. No code work until each lands: named lead + `commercial_hypothesis.md` + ≥3 discovery interviews.
4. Enterprise Delivery Governance and Russian Sovereign tracks are **deferred**, only triggered by enterprise ICP signal.
5. `sdp-pr-gate` (ChangePassport internal namespace) implementation is **gated on committed pilot**. F151 ships only design artifacts (Schema v1, API, Decision Record, override, GitHub App flow, pilot measurement plan).

## Track map

### Now (in-flight)

| Epic | Status | Gate |
|---|---|---|
| F150 — Product layering and release readiness | in progress (`sdplab-nyr0`) | closes when v3 acceptance bar in memo §"Discernment Metrics" turns green |

### Immediate next (parallel after F150 design tasks unblock; some can start now)

| Epic | Depends on | Why now |
|---|---|---|
| F151 — `sdp-pr-gate` Design v1 | F150-04 (isolation lint) + F150-03 (namespace lock visible in code) | freezes Schema v1 + Evidence Provider API v1 + Decision Record v1 + Override protocol + GitHub App v1 flow + pilot measurement plan; all design, no implementation |
| F152 — Pricing Hypothesis (Operator + sdp-pr-gate) | F151 (need scope) | required before any external pilot per IIP-council Pragmatist; attaches WTP-measurement instrument |
| F153 — SDP Brand Architecture | none — can start now | required before first external launch; resolves family confusion (Toolkit / Toolbox / Operator / ChangePassport / EDG / IIPs) |
| F154 — Shared Substrates v1 | F150-04 (substrate package boundaries) | locks semver contracts for `sdp-{evidence,policy,modelgw,context,eval}-core`; documents per-substrate SDP-runtime assumptions |
| F158 — Go Import-Path Contamination Decision | none — can start now | blocks any active IIP (highest unaddressed structural risk per IIP-council, all 5 roles flagged) |

### Then (after design freezes)

| Epic | Depends on | Why |
|---|---|---|
| F155 — Evidence Persistence Architecture | F151 + F154 | Schema v1 must know storage backend; substrate `sdp-evidence-core` API depends on persistence shape |

### Hypotheses (beads tracking only; gated)

| Epic | Gate to activation |
|---|---|
| F156 — `arch-snap` Hypothesis | named lead + `commercial_hypothesis.md` + ≥3 discovery interviews + F158 decision applied |
| F157 — `doc-tracer` Hypothesis | same as F156 |

### Risk / advisory tracks (small artifacts, can start when capacity allows)

| Epic | Why |
|---|---|
| F159 — Competitive Positioning Artifact | flagged by IIP council Pragmatist; vs Copilot Workspace, CodeRabbit, GitLab Duo, Tabnine, Factory |
| F160 — Procurement / Compliance Install Profile | flagged by both councils; SOC2/SLA/indemnification stance for ICPs in regulated industries |

### Deferred (no work until ICP signal)

| Track | Trigger |
|---|---|
| F-EDG family (Enterprise Delivery Governance) | enterprise ICP signs LOI; multiple sub-epics |
| F-SOVEREIGN family (Russian sovereign model adapters) | sub-track of EDG; same trigger |
| `sdp-pr-gate` implementation | committed pilot per Wedge B gate; new F-track when triggered |

## Sequencing

```text
F150 (running)
  ├── F150-03 namespace lock visible ────┐
  ├── F150-04 isolation lint operational ┤
  └── F150-02 toolbox+IIP registry ──────┤
                                          │
                            ┌─────────────┴─────────────┐
                            │                           │
       F151 sdp-pr-gate     F154 Substrates v1
       Design v1            (semver + SDP-runtime
       (Schema, API,         assumption docs)
       Decision Record,        │
       Override,               │
       GitHub App,             │
       pilot plan) ────────────┘
       │
       ├── F152 Pricing Hypothesis ─── (gates external pilot)
       └── F155 Evidence Persistence Architecture

(parallel)
F153 SDP Brand Architecture       — ready to start now
F158 Import-path contamination    — ready to start now (blocks IIP)
F159 Competitive Positioning      — ready to start now (advisory)
F160 Procurement Profile          — depends on F151 (light)

(gated, hypothesis-only beads)
F156 arch-snap   ── waits for named lead + commercial_hypothesis.md + ≥3 interviews
F157 doc-tracer  ── same gate

(deferred)
F-EDG family            — enterprise ICP LOI
F-SOVEREIGN family      — sub-track of EDG
sdp-pr-gate impl track  — committed pilot
```

## Concrete deliverables per new epic

### F151 — `sdp-pr-gate` Design v1

Source: ChangePassport manifesto v2 §"Build Next". Six artifacts; design only, no implementation.

- F151-01 Passport Schema v1 (JSON Schema)
- F151-02 Evidence Provider API v1
- F151-03 Decision Record v1
- F151-04 Override protocol
- F151-05 GitHub App v1 flow design (auth, webhook, check, comment override)
- F151-06 Pilot measurement plan (install time, evidence-mismatch rate, useful-decision rate, false-block rate, reviewer time delta, post-merge incident rate)

### F152 — Pricing Hypothesis (Operator + sdp-pr-gate)

- F152-01 Operator Mode pricing hypothesis doc (per-active-repo or per-team monthly base; included volume; expansion path)
- F152-02 `sdp-pr-gate` pricing hypothesis doc (decision volume, overage, expansion)
- F152-03 ≥3 discovery interviews with target ICP for `sdp-pr-gate` (boutique consulting / agency / fractional CTO)

### F153 — SDP Brand Architecture

- F153-01 Brand family map (Lab / Toolbox / Toolkit / Operator / ChangePassport / EDG / IIPs)
- F153-02 Naming policy doc (when `sdp-` prefix; when not; rename criteria)
- F153-03 Trademark and domain check on key working names (`ChangePassport`, `arch-snap`, `doc-tracer`)

### F154 — Shared Substrates v1

Five substrates; each gets API v1 + SDP-runtime-assumption doc.

- F154-01 `sdp-evidence-core` v1 (semver contract + assumption doc)
- F154-02 `sdp-policy-core` v1
- F154-03 `sdp-modelgw-core` v1
- F154-04 `sdp-context-core` v1
- F154-05 `sdp-eval-core` v1

### F155 — Evidence Persistence Architecture

Single decision artifact. Storage backend (git LFS vs object storage vs local SQLite vs MCP server), retention policy, backup, privacy. Required before Schema v1 freeze.

### F156 — `arch-snap` Hypothesis

Single gating activity. Cannot proceed to code without:

- named IIP lead;
- `commercial_hypothesis.md` (target non-SDP ICPs, top-3 competitors, WTP range, kill criteria);
- ≥3 discovery interviews with due-diligence / M&A / security architect / tech writer personas;
- F158 decision applied (Go import path).

When all four land: epic is promoted to active, child workstreams are then carved.

### F157 — `doc-tracer` Hypothesis

Same gate as F156. Discovery interviews target docs-as-code / FDA / ISO 13485 / GxP / audit functions.

### F158 — Go Import-Path Contamination Decision

Single decision artifact. Three options analyzed:

- A. Neutral GitHub org from inception (`incubator-org/arch-snap`) while SDP team is maintainer.
- B. Accept contamination during incubation; path changes at extraction (`git filter-repo`).
- C. Hybrid: monorepo at very early phase, neutral org at v0.1.

Decision must be applied before any IIP earns active status. Blocking F156, F157.

### F159 — Competitive Positioning Artifact

Single doc. Differentiator analysis vs Copilot Workspace, CodeRabbit, GitLab Duo Self-Hosted, Tabnine, Factory, OpenHands Enterprise, Sourcegraph. Output: positioning statement and battle card.

### F160 — Procurement / Compliance Install Profile

Single doc. Install profile that survives basic security review (no egress by default, scoped GitHub App permissions, SOC2 stance, SLA template, indemnification template). Required before any regulated-industry pilot.

## What is NOT in this roadmap

- ChangePassport implementation (`sdp-pr-gate` runtime). Triggered by committed pilot; new F-track when triggered.
- Enterprise Delivery Governance. Triggered by enterprise ICP LOI; multiple sub-epics under a future F-EDG family.
- Russian sovereign model adapters. Sub-track of EDG.
- IIP code (`arch-snap`, `doc-tracer`). Hypothesis-only until each gate lands.
- Promotion of any Toolbox tool to a separate product category (requires ≥2 external consumers + distinct ICP).

## Decision dependencies (what blocks what)

| Decision | Blocks |
|---|---|
| F158 Go import-path | F156, F157 active status; first-active IIP |
| F151 Schema v1 freeze | F155, future `sdp-pr-gate` impl, committed-pilot search |
| F154 substrate semver | F151 stability; future EDG; F155 |
| F150 close (namespace lock + isolation lint + cascade AGENTS.md ≥20% root reduction) | F151, F154 prerequisites |
| Committed pilot | `sdp-pr-gate` impl track creation |
| Enterprise ICP LOI | EDG track creation |
| Named lead per IIP | F156 / F157 promotion to active |

## Beads epics created on 2026-04-27

The following beads epics and children are created together with this roadmap. Each beads epic carries the priority shown.

| Epic | Priority | Children |
|---|---|---|
| F151 — `sdp-pr-gate` Design v1 | P2 | 6 (F151-01..F151-06) |
| F152 — Pricing Hypothesis | P2 | 3 (F152-01..F152-03) |
| F153 — SDP Brand Architecture | P2 | 3 (F153-01..F153-03) |
| F154 — Shared Substrates v1 | P2 | 5 (F154-01..F154-05) |
| F155 — Evidence Persistence Architecture | P3 | 1 (F155-01) |
| F156 — `arch-snap` Hypothesis | P3 (gated) | 1 gating workstream (F156-01) |
| F157 — `doc-tracer` Hypothesis | P3 (gated) | 1 gating workstream (F157-01) |
| F158 — Go Import-Path Contamination Decision | P2 | 1 (F158-01) |
| F159 — Competitive Positioning Artifact | P3 | 1 (F159-01) |
| F160 — Procurement / Compliance Install Profile | P3 | 1 (F160-01) |

Total new beads: 10 epics + 23 children = 33 issues.
