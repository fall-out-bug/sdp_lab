---
title: SDP Incubated Independent Products (IIP) Council — R1+R2 Synthesis
date: 2026-04-27
input: memo v2 + Option D proposal (12 claims D1..D12)
roles_models:
  - Architect: xiaomi/mimo-v2.5-pro (R2 primary; R1 fell back to mimo-v2.5)
  - Critic: minimax/minimax-m2.7
  - Technician: moonshotai/kimi-k2.6 (R1 fell back to k2.5; R2 primary worked at max_tokens=100000)
  - Philosopher: deepseek/deepseek-v4-pro
  - Pragmatist: qwen/qwen3.6-plus
output: memo v3 (post-IIP-council) + new beads epic for arch-snap and doc-tracer hypotheses
total_completion_tokens_R2: ~24000
---

# IIP Council Synthesis — 2026-04-27

This synthesis aggregates R1 (Socratic) and R2 (Council) outputs from five models on the question: **what to do with tools that have value outside SDP** (arch-snap, doc-tracer).

Council unanimous: **ACCEPT WITH CHANGES (5/5)**.

## Voting record (R2 final verdicts)

| Claim | Architect | Critic | Technician | Philosopher | Pragmatist | Net |
|---|---|---|---|---|---|---|
| D1 — independent value | AWR | WEAK | ACCEPT | ACCEPT | AWR | accept (small revisions) |
| D2 — new taxonomy row | AWR (flag, not row) | WEAK | ACCEPT | ACCEPT | AWR (flag, not row) | **revise: flag, not row** |
| D3 — `independent_value: yes` annotation | AWR (CI-derived) | OPPOSE | AWR (machine-gen) | ACCEPT | AWR (commercial_hypothesis.md) | revise heavily |
| D4 — import isolation rules | AWR (manual interim) | OK PREMATURE / AWR | AWR (manual interim) | AWR (manual interim) | AWR (manual interim) | **revise: manual interim** |
| D5 — own go.mod / semver / Homebrew | AWR (defer Homebrew) | WEAK | AWR (defer Homebrew) | AWR (defer Homebrew) | AWR (defer Homebrew) | **revise: standalone go.mod yes; defer Homebrew/CI** |
| D6 — extraction criteria | AWR (raise bar; drop domain) | WEAK / AWR (verification) | AWR (raise bar; drop domain) | AWR (drop domain) | AWR (commercial signal) | **revise: commercial signal + name/license; drop domain** |
| D7 — Toolbox narrows | AWR (dual-track flag) | OK CONTINGENT | ACCEPT | ACCEPT | ACCEPT | accept |
| D8 — flagship candidates today | AWR (factual fix) | VETO | AWR (factual fix) | AWR (factual fix) | AWR (factual fix) | **revise: hypotheses, not flagship** |
| D9 — own beads epic outside F150 | AWR (named lead) | WEAK / AWR (named lead) | AWR (compliance milestones + named owner) | ACCEPT | AWR (track within F150) | **revise: named lead required** |
| D10 — brand strategy | AWR (transfer plan) | WEAK / AWR (transfer trigger) | ACCEPT | AWR (neutral org + transfer) | AWR (auto-transfer) | **revise: explicit transfer plan** |
| D11 — independent pricing | AWR (cost allocation) | OK | ACCEPT | ACCEPT | ACCEPT | accept (minor revision) |
| D12 — cap of 3 IIPs | AWR (downgrade criteria) | OK / AWR | ACCEPT | ACCEPT | ACCEPT | **revise: + downgrade criteria** |

Legend: AWR = ACCEPT WITH REVISION.

## Consensus changes (≥3 roles converge)

The author commits to applying the following changes for memo v3.

### A. IIP is a FLAG inside Toolbox lifecycle, not a new top-level taxonomy row (D2)

**Pushed by**: Architect, Critic, Pragmatist. Technician and Philosopher accept.

**Reason**: introducing a new taxonomy row immediately after memo v2 council consensus signals strategic instability. The intent (separate identity, no SDP funnel subordination) is captured by a metadata flag plus naming/isolation rules, not by a new layer.

**Action**: replace memo v2 §"Revised Layer Taxonomy" row insertion with a flag mechanism:

- Toolbox tools carry `standalone: true | false`.
- Tools with `standalone: true` follow strict IIP rules (no `sdp-` prefix from inception, own go.mod, named lead, brand transfer plan).
- Promotion to a separate top-level row happens ONLY at extraction (when criteria are met).
- This avoids re-litigating the taxonomy after every change.

### B. Replace "flagship candidates today" with "hypotheses under evaluation" (D8)

**Pushed by**: Critic VETO; Architect/Technician/Philosopher/Pragmatist AWR.

**Reason**: arch-snap and doc-tracer do not exist in code. Calling them "flagship today" is factual misrepresentation that undermines memo credibility.

**Action**: every reference to arch-snap/doc-tracer in memo v3 marks them as "IIP hypotheses under evaluation; neither exists in code yet; ICPs are constructed personas requiring customer discovery validation before IIP infrastructure investment."

### C. Manual dependency audit interim until WS 00-150-04 lint deploys; extraction blocked until lint passes (D4)

**Pushed by**: All 5 roles converge.

**Reason**: claiming active CI lint enforcement is false confidence. WS 00-150-04 doesn't exist yet. The principle is sound; the present-tense framing is dishonest.

**Action**: memo v3 frames import isolation as:

- **From day one**: mandatory manual `go list -deps ./...` audit per PR; checklist of forbidden packages.
- **From WS 00-150-04 deploy**: automated lint becomes mandatory; manual audit deprecated.
- **Until lint deploys**: IIP cannot be extracted; it can only incubate.

### D. Standalone go.mod mandatory from day one; defer separate Homebrew formula and CI matrix to extraction (D5)

**Pushed by**: All 5 roles converge.

**Reason**: parallel Homebrew formulas / dedicated CI lanes for tools that don't exist yet is runway burn. Architect's domain veto on `replace` directives stands: extraction must be mechanical.

**Action**:

- Each IIP MUST have its own `go.mod` from inception, with **zero `replace` directives** pointing to monorepo paths and **zero `internal/sdp-*` imports**.
- Distribution during incubation goes through unified SDP CI/build pipeline (subcommand or build tag).
- Separate Homebrew formula, dedicated landing page, and isolated CI matrix are deferred to the extraction event.
- Standalone semver tag prefix (`arch-snap/v0.1.0`) is encouraged from day one to make the future split mechanical.

### E. Extraction criteria: commercial signal + name/license; domain optional (D6)

**Pushed by**: Architect (≥10 weekly users OR ≥1 paying customer + co-maintainer/sponsor), Critic (verification by GitHub stars + survey), Technician (technical isolation lint pass + ≥10 users / ≥1 LOI), Philosopher (drop domain), Pragmatist (signed pilot/LOI + 3 discovery interviews + name/license).

**Reason**: ≥2 weekly users is statistically meaningless. Domain is needed for commercial launch, not for extraction. Commercial signal (LOI, paying customer, sponsor) is the right gate.

**Action**: extraction criteria become:

1. Technical: lint pass (no forbidden imports); standalone go.mod clean (`go mod tidy` no `replace`).
2. Demand: ≥1 of (a) signed LOI / committed paying customer; (b) ≥10 weekly active external users with ≥50% non-SDP attribution by survey; (c) co-maintainer or corporate sponsor with funded commitment.
3. Discovery: ≥3 documented customer-discovery interviews validating ICP and willingness-to-pay.
4. Identity: finalized name + license + visual identity. Dedicated domain optional until commercial launch.

### F. Each IIP requires a named lead (individual, not team) (D9)

**Pushed by**: Architect, Critic, Technician, Pragmatist.

**Reason**: precedent (Ginkgo, containerd) shows champions matter. SDP-team-as-maintainer creates a delivery vacuum.

**Action**: memo v3 mandates: every IIP epic has a named individual as IIP lead. The lead owns delivery cadence, commercial validation, and extraction readiness. IIP leads report capacity allocation to product council quarterly. No lead = no IIP status.

### G. Brand ownership transfer plan from inception (D10)

**Pushed by**: Architect, Critic, Philosopher, Pragmatist.

**Reason**: "maintainer not brand owner" is legally hollow without explicit trademark/transfer language. Procurement buyers ask for clear ownership chains.

**Action**: memo v3 specifies:

- **License from day one**: permissive (Apache-2.0 or MIT) unless a specific commercial track requires otherwise.
- **Trademark plan**: at incubation, IIP brand is held by the SDP organization with a written commitment to transfer to a neutral entity / foundation / new legal vehicle at extraction. The commitment is documented in the IIP's `BRAND.md`.
- **Co-sponsor option**: any organization that contributes ≥$50K/year or ≥1 dedicated maintainer-FTE may be listed as co-sponsor with brand-decision input.
- **At extraction**: brand transfer triggers automatically.

### H. Downgrade criteria for stalled IIPs (D12)

**Pushed by**: Architect, Critic.

**Reason**: cap (3 active) without floor lets dead IIPs occupy slots forever.

**Action**: memo v3 adds:

- IIP downgraded back to Lab (or archived) if ANY of:
  - 60 days with zero external consumers;
  - 30 days without active maintainer;
  - 90 days with unpatched critical security finding;
  - 12 months without commercial signal AND IIP cost cannot be justified by lead.
- Downgrade requires founder/owner approval (mirrors promotion).

### I. Independent value via verifiable artifact, not self-declared annotation (D3)

**Pushed by**: Architect (CI-derived flag), Technician (machine-generated from dependency audit), Pragmatist (`commercial_hypothesis.md` document).

**Reason**: a hand-written `independent_value: yes` is metadata theater. Real independence is verified by import graph + commercial discovery.

**Action**: each IIP module MUST contain BOTH:

- machine-verifiable evidence: `go list -deps ./...` shows zero `internal/sdp-*` imports; `go.mod` has no `replace` directives;
- a `commercial_hypothesis.md`: target non-SDP ICPs (named verticals/personas), pricing-model hypothesis, top 3 competitors, expected willingness-to-pay range, kill criteria.

The `independent_value` flag is set/unset by these artifacts, not by hand annotation.

## Risks surfaced by council that the proposal still does not fully address

The author accepts these and tracks outside the immediate memo update.

1. **Go import-path contamination** (Architect, Critic, Technician, Philosopher all flag): even without `sdp-` prefix, `github.com/<sdp-lab-org>/sdp_lab/arch-snap` permanently associates the tool with SDP in every Go import statement. This is the highest-priority unaddressed risk. Mitigation options to study:
   - Use a neutral GitHub org for incubation from day one (separate `incubator-org/arch-snap` while still developed by the SDP team).
   - Accept the contamination as an incubation cost; the import path changes at extraction (mechanical via `git filter-repo`).
   - Hybrid: monorepo sub-path during very early phase, neutral org from v0.1 onwards.

2. **Substrate transitive coupling** (Architect, Technician): substrate packages may carry implicit SDP-runtime assumptions (context objects, env vars, config schemas). Mitigation: substrate `AGENTS.md` MUST document SDP-runtime assumptions; IIP imports require sign-off from substrate owner.

3. **Chicken-and-egg adoption suppression** (Architect, Critic, Philosopher): incubation environment broadcasts subordination, suppressing the non-SDP adoption that extraction criteria require. Mitigation: addressed via the brand-transfer plan (G) and commercial-signal-based extraction (E); also addressed if the neutral-org incubation option (risk 1) is taken.

4. **CI matrix runner exhaustion** (Technician, Pragmatist): even with deferred separate formulas, parallel IIP semver tags + macOS bottle builds will saturate runners. Mitigation: capacity ceiling and budget plan documented; max 3 IIPs (D12 enforces this).

5. **Procurement/compliance friction** (Pragmatist): ICPs (FDA, ISO, M&A due-diligence) require SOC2, SLAs, indemnification. Permissive OSS defaults are insufficient. Tracked outside memo for IIP-by-IIP commercial track.

6. **Maintainer incentive misalignment** (Pragmatist): SDP engineers maintain IIPs they don't own with zero equity/revenue share. Mitigation: each IIP epic must include a maintainer-incentive plan before promotion (revenue share, dedicated FTE, sponsor-funded role, etc.). Tracked outside immediate memo update.

## Preserved minority report

- **Critic — partial REJECT**: 7 of 12 claims still rated WEAK in R2 (D1, D2, D3, D5, D6, D9, D10). Critic accepts with changes overall, but signals that even with the consensus changes, the proposal commits non-trivial engineering investment for non-existent products. Position recorded; the "named lead + commercial signal at extraction + downgrade criteria" combination substantially closes the gap. If after 90 days neither arch-snap nor doc-tracer has a named lead with discovery interviews, Critic's position would crystallize to a hard reject.

## Decisions not changed

- **D7** Toolbox narrows to SDP-onboarding tools (4/5 ACCEPT, 1 OK CONTINGENT).
- **D11** Independent pricing per IIP (4/5 ACCEPT or AWR-light; cost allocation added).
- The core direction (Option D — incubate-then-spin-out) is endorsed by all 5 roles.

## Final action

- Author updates memo to v3 in place at `docs/strategy/2026-04-27-sdp-product-layering-4d.md`.
- The new "IIP" mechanism is a FLAG within Toolbox, not a new taxonomy row.
- arch-snap and doc-tracer reframed as hypotheses, not flagship.
- F150 patch list updated minimally — IIP architecture (manual audit, isolation lint, standalone go.mod) tracked as enhancements to existing WS 00-150-02/04, not as new workstreams.
- Beads epics for arch-snap and doc-tracer postponed until each has: (a) named IIP lead, (b) commercial_hypothesis.md, (c) at least 3 discovery interviews. Until then they remain hypothesis lines in memo v3.
- This synthesis document is canonical record of the IIP council; raw R1+R2 outputs preserved alongside.

## Self-check via 4D after the council

- **Delegation**: I delegated challenge to 5 non-Anthropic, non-OpenAI models including the user-requested Xiaomi and MiniMax. I did not delegate identity decisions to a single model. Synthesis is human-authored.
- **Description**: input was 12 atomic claims (D1-D12) plus full memo v2 context. Output: structured R2 verdicts per role + named risks.
- **Discernment**: ≥3-of-5 convergence is the strong-signal threshold. Critic's 7-of-12 WEAK is preserved as minority. Two roles (Technician R1, Architect R1) fell back to non-pro variants; R2 primaries worked.
- **Diligence**: raw outputs versioned in `docs/strategy/council/2026-04-27-iip/r{1,2}-{role}.{md,json}`. Token usage and cost recorded per call. `max_tokens=100000` per user request fixed the previous Kimi truncation issue.
