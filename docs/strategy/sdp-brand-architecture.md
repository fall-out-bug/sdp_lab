---
title: SDP Brand Architecture
status: v1
owner: Andrei
beads: sdplab-mbhg (F153)
created: 2026-04-27
source: docs/strategy/2026-04-27-sdp-product-layering-4d.md
---

# SDP Brand Architecture

One-page map of the SDP product family: names, audiences, namespaces, and naming rules.

> Authoritative source for product taxonomy: [SDP Product Layering 4D Memo](2026-04-27-sdp-product-layering-4d.md) (v3, post-IIP-council).
> This document derives from the memo and makes the brand structure actionable.

## Brand Family Map

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SDP Product Family                               │
├───────────────┬──────────────────┬──────────────┬───────────────────────┤
│ Surface       │ Working name     │ Audience     │ Paid                  │
├───────────────┼──────────────────┼──────────────┼───────────────────────┤
│ Lab           │ sdp_lab          │ Researchers  │ No (feeds others)     │
│ Toolbox       │ SDP Toolbox      │ Developers   │ No (freemium funnel)  │
│  └ IIP flag   │ (bare name)      │ Non-SDP ICPs │ Independent at extract│
│ Toolkit       │ sdp CLI          │ Developers   │ No                    │
│ Operator Mode │ sdp operator     │ Eng managers │ No (price hypothesis) │
│ ChangePassport│ ChangePassport   │ Reviewers    │ Yes (first paid wedge)│
│  └ Internal   │ sdp-pr-gate      │ —            │ —                     │
│ Ent. Delivery │ TBD              │ Enterprise   │ Yes (hypothesis)      │
│ Substrates    │ sdp-*-core       │ Internal     │ No (semver contracts) │
└───────────────┴──────────────────┴──────────────┴───────────────────────┘
```

## Surface Details

| # | Surface | Kind | Working name | Internal namespace | Audience | Paid status |
|---|---------|------|-------------|-------------------|----------|-------------|
| 1 | SDP Lab | Research workspace | `sdp_lab` | `sdp_lab` | Researchers | No — feeds others |
| 2 | SDP Toolbox (with IIP flag) | Subordinate freemium funnel | `SDP Toolbox` for SDP tools; bare names for IIPs | `sdp-toolbox-*` / bare | Developers (funnel); non-SDP ICPs (IIP) | Free (Toolbox); per-IIP pricing at extraction (some may stay OSS) |
| 3 | SDP Toolkit | Meta-distribution | `sdp` CLI | `sdp` | Developers | Free |
| 4 | Operator Mode | Default Toolkit Happy Path; stateful orchestration | `sdp` operator commands | `sdp-operator-*` | Engineering managers | Free; provisional pricing hypothesis required before pilot |
| 5 | ChangePassport (display) | Merge-readiness product | `ChangePassport` | `sdp-pr-gate` | Reviewers, boutique consulting | First paid wedge |
| 6 | Enterprise Delivery Governance | Enterprise hypothesis | TBD | `sdp-edg-*` | Enterprise | Future paid |
| 7 | Shared Substrates | Versioned packages | Individually named | `sdp-{evidence,policy,modelgw,context,eval}-core` | Internal | No (contracts) |

## Display vs Internal Namespace

Marketing display names may evolve. Internal technical namespaces are locked when first named.

| Surface | Display name (may change) | Internal namespace (locked) | Lock scope |
|---------|--------------------------|----------------------------|------------|
| ChangePassport | `ChangePassport` | `sdp-pr-gate` | Go packages, CLI slugs, env vars, semver tags, DB tables |
| Enterprise Delivery Governance | TBD | `sdp-edg-*` | Reserved |
| Operator Mode | Operator Mode | `sdp-operator-*` | Go packages, CLI slugs |
| Substrates | individual | `sdp-*-core` | Go packages, semver tags |

## IIP Flag Mechanism

Tools inside SDP Toolbox with value outside SDP carry `standalone: true` and follow Incubated Independent Product rules:

- No `sdp-` prefix from inception
- Standalone `go.mod` (zero `replace` directives, zero `internal/sdp-*` imports)
- Module `AGENTS.md` ≤60 lines, written as if SDP did not exist (no `sdp-` references in cold-start text)
- Named IIP lead (individual, not team) required
- `commercial_hypothesis.md` required (target non-SDP ICPs, top 3 competitors, willingness-to-pay range, kill criteria)
- At least 3 documented customer-discovery interviews before earning `standalone: true`
- Founder/owner approval required for IIP promotion
- Permissive license (Apache-2.0 / MIT)
- `BRAND.md` with trademark transfer plan

Current IIP hypotheses (neither has earned `standalone: true`; no code yet): `arch-snap`, `doc-tracer`. Each requires named lead + hypothesis doc + discovery interviews before IIP status.

---

## Naming Policy

### When `sdp-` Prefix Is Required

| Category | Rule | Examples |
|----------|------|---------|
| Toolkit CLI | Always `sdp` | `sdp scout`, `sdp metrics` |
| Toolbox tools (SDP-funnel) | `sdp-` prefix | `sdp-scout`, `sdp-metrics`, `sdp-index` |
| Internal namespaces | `sdp-` prefix | `sdp-pr-gate`, `sdp-operator`, `sdp-edg-*` |
| Shared Substrates | `sdp-*-core` | `sdp-evidence-core`, `sdp-policy-core` |
| CLI subcommands | `sdp <verb>` | `sdp orchestrate`, `sdp ready` |

### When `sdp-` Prefix Is Forbidden

| Category | Rule | Examples |
|----------|------|---------|
| IIP candidates | No `sdp-` prefix from inception | `arch-snap`, NOT `sdp-arch-snap` |
| IIP module names | Bare name only | `doc-tracer`, NOT `sdp-doc-tracer` |
| IIP AGENTS.md | Written as if SDP did not exist | No `sdp-` references in cold-start text |

### When `sdp-` Prefix Is Optional

| Category | Rule | Notes |
|----------|------|-------|
| Display names | Decoupled from namespace | `ChangePassport` (display) vs `sdp-pr-gate` (internal) |
| Working names | Stay working until rename criteria met | `Enterprise Delivery Governance` is a category, not a final name |

### Working-Name Rename Criteria

A working name becomes a final name when ALL four criteria are met:

1. **Domain available** — primary TLD (.com or relevant) is available or acquired
2. **No trademark collision** — USPTO and common-law search returns clear
3. **ICP recognizes the name** — target buyer understands it without explanation
4. **Council/buyer language test passes** — at least 2 external buyers confirm the name fits the product

Until all four are met, the name stays "working" and may change without migration cost.

### Display-vs-Internal Namespace Rules

1. **Display names** live in: README, product-surface docs, marketing pages, blog posts, buyer-facing UI strings.
2. **Internal namespaces** live in: Go package paths, CLI slugs, GitHub App IDs, webhook paths, database tables, environment variables, semver tags.
3. **Lock scope**: once an internal namespace is named, it is locked. Changes require a versioned migration with semver bump and deprecation window.
4. **Display name changes** do NOT trigger internal namespace changes (decoupled by design).

### Toolbox Tool Naming Rules

1. Tool name must survive without `sdp-` prefix in case the tool is extracted to its own repo.
2. CLI subcommand format: `sdp <tool-name>` (prefix handled by CLI routing, not tool name).
3. Module `AGENTS.md` uses tool name without prefix.
4. If a Toolbox tool is promoted to IIP (`standalone: true`), it MUST already have been named without `sdp-` from inception (rule 1). Retroactive renaming is not an allowed path — tools that acquired an `sdp-` prefix cannot become IIPs without a full name change.

### IIP Naming Rules

1. No `sdp-` prefix in: module name, repo path, README banners, AGENTS.md cold-start text.
2. Standalone `go.mod` from day one.
3. Independent semver tag prefix (`arch-snap/v0.1.0`, `doc-tracer/v0.1.0`).
4. `BRAND.md` in module root documents trademark transfer plan.
5. Permissive license (Apache-2.0 or MIT) by default.

## Names to Avoid

Per competitive research and council consensus:

- "AI software engineer" — forces a feature war
- "local Copilot" — positions as knockoff
- "on-prem coding assistant" — wrong category
- "sovereign coding agent" — ambiguous geopolitics
- "Perimeter Control Plane" — wrong category (network appliance, not delivery governance)

## Sources

- [SDP Product Layering 4D Memo](2026-04-27-sdp-product-layering-4d.md) (v3, authoritative)
- [F150 Product Layering Design](../plans/2026-04-27-f150-product-layering-release-readiness-design.md)
- [Product Surface Reference](../reference/product-surface.md)
- [REPO-BOUNDARY](../architecture/REPO-BOUNDARY.md)
- Memo council synthesis: `council/2026-04-27/synthesis.md`
- IIP council synthesis: `council/2026-04-27-iip/synthesis.md`

## Trademark & Domain Evidence

Per-name trademark and domain availability checks:
- [ChangePassport](naming/changepassport.md) — TM medium-high, .com taken (broker)
- [arch-snap](naming/arch-snap.md) — TM low, all hyphenated domains free
- [doc-tracer](naming/doc-tracer.md) — TM low-medium, doctracer.com taken since 2010
- [sdp-pr-gate](naming/sdp-pr-gate.md) — TM low, all 4 domains free
- [Enterprise Delivery Governance](naming/enterprise-delivery-governance.md) — TM high (descriptive), all free
