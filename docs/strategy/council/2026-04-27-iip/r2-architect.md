# R2 — Architect

Model: `xiaomi/mimo-v2.5-pro`  
Fallback used: `False`  
Elapsed: 71.9s  
OK: `True`  
Finish: `stop`  
Usage: `{'prompt_tokens': 15231, 'completion_tokens': 4381, 'total_tokens': 19612, 'cost': 0.0282204, 'is_byok': False, 'prompt_tokens_details': {'cached_tokens': 192, 'cache_write_tokens': 0, 'audio_tokens': 0, 'video_tokens': 0}, 'cost_details': {'upstream_inference_cost': 0.0282204, 'upstream_inference_prompt_cost': 0.0150774, 'upstream_inference_completions_cost': 0.013143}, 'completion_tokens_details': {'reasoning_tokens': 2394, 'image_tokens': 0, 'audio_tokens': 0}}`

---

# Architect Round 2 — Final Verdicts

---

## CLAIM VERDICTS

| # | Verdict | Revised Claim (if applicable) |
|---|---------|-------------------------------|
| D1 | ACCEPT WITH REVISION | Tools with plausible non-SDP ICPs carry `standalone: true` flag and are incubated without `sdp-` prefix; flag is revoked if no external consumer validates within 6 months. |
| D2 | ACCEPT WITH REVISION | IIP is a flag (`standalone: true`) within the existing Toolbox row, not a new taxonomy row; promoted to separate repo (and optionally a taxonomy row) only when D6 extraction criteria are met. |
| D3 | ACCEPT WITH REVISION | Replace self-declared annotation with CI-enforced dependency audit: module qualifies as independent if `go.mod` imports zero `internal/sdp-*` packages and ≤2 pinned substrate packages; AGENTS.md independence is a documentation target, not a gate. |
| D4 | ACCEPT WITH REVISION | Import isolation rules are architecturally correct but enforcement is aspirational until WS 00-150-04 ships; interim enforcement is mandatory manual dependency audit per PR, with extraction blocked until automated lint passes. |
| D5 | ACCEPT WITH REVISION | Each IIP must have a standalone `go.mod` from day one (no `replace` directives pointing at monorepo paths); defer separate Homebrew formula and landing page to extraction event; use monorepo-internal installer dispatch until then. |
| D6 | ACCEPT WITH REVISION | Extraction requires ≥10 weekly active external users OR ≥1 paying customer, ≥50% non-SDP usage at that scale, and a second organization willing to co-maintain or sponsor; drop domain requirement (domain needed for commercial launch, not extraction). |
| D7 | ACCEPT WITH REVISION | Toolbox tools with plausible independent value carry a `dual-track: true` flag; no hard binary reclassification until extraction criteria are met, preventing constant re-litigation of the boundary. |
| D8 | ACCEPT WITH REVISION | Two IIP hypotheses under evaluation: `arch-snap` and `doc-tracer`. Neither exists in code. ICPs are constructed personas requiring customer discovery validation before IIP infrastructure investment. |
| D9 | ACCEPT WITH REVISION | Each IIP has a designated IIP lead (named individual) who owns delivery; F150 provides architectural compliance checks only; IIP leads report capacity allocation to product council quarterly. |
| D10 | ACCEPT WITH REVISION | SDP team is initial maintainer; brand ownership must transfer to a separate legal vehicle or neutral foundation by extraction time; incubation uses a neutral GitHub org or explicit trademark assignment agreement; SDP receives no preferential marketing rights. |
| D11 | ACCEPT WITH REVISION | Each IIP has an explicit cost-allocation model (engineering hours, infra, support) reviewed quarterly; if an IIP cannot justify its cost within 12 months, it is archived or re-subordinated; pricing model defined at extraction event. |
| D12 | ACCEPT WITH REVISION | Add downgrade criteria: IIP downgraded to Lab if no external consumers for 60 days, no active maintainer for 30 days, or critical security unpatched for 90 days. |

---

## MIND CHANGES (R1 → R2)

**1. D4: STRONG → ACCEPT WITH REVISION**
The **Technician's veto** moved me. In R1 I endorsed the import restriction as "the most architecturally sound claim" without adequately flagging that the CI lint (WS 00-150-04) doesn't exist. Claiming active enforcement for unimplemented infrastructure is compliance theater. The principle is sound; the execution claim is false.

**2. D5: OK → ACCEPT WITH REVISION**
The **Technician's CI matrix argument** and the **Pragmatist's runway burn argument** moved me. In R1 I focused on go.mod standalone (correct) but underweighted the operational cost of separate Homebrew formulas for non-existent products. 12 parallel macOS builds per release for incubating tools is unsustainable. Defer distribution infrastructure to extraction.

**3. D8: OK → ACCEPT WITH REVISION**
The **Critic's factual accuracy veto** moved me. In R1 I treated "flagship candidates today" as aspirational language. The Critic is right: in a strategy document, "today" means current state. Current state: zero lines of code. Calling non-existent hypotheses "flagship" is strategic misrepresentation that undermines the proposal's credibility.

**4. D6: OK → ACCEPT WITH REVISION**
The **Philosopher's domain veto** moved me. In R1 I focused on raising the consumer bar (which I still endorse) but didn't flag that requiring a finalized domain at extraction is premature brand lock-in. Domain acquisition should gate commercial launch, not repo extraction. Many successful spin-outs operated for years without dedicated domains.

**5. D2: WEAK → ACCEPT WITH REVISION (direction changed)**
The **Philosopher's argument** about organizational signaling gave me pause, but the **Architect's own R1 argument** (which I now endorse more strongly in R2) convinced me: adding a taxonomy row immediately after memo v2 consensus signals instability. A flag within Toolbox preserves the taxonomy while capturing the intent. The Philosopher's concern about brand identity is addressed by the naming convention (no `sdp-` prefix), not by the taxonomy row.

---

## 3 MOST IMPORTANT UNADDRESSED RISKS

**1. Import-path contamination via monorepo module path**
Even with no `sdp-` prefix, the Go module path `github.com/sdp-lab/sdp_lab/arch-snap` permanently associates the tool with SDP in every import statement. Non-SDP teams evaluating the tool will see `sdp-lab` in their `go.mod` and question independence. The proposal must specify whether a separate repo or neutral module path can be adopted during incubation (e.g., `github.com/arch-snap/arch-snap` from day one, with monorepo as a development convenience only).

**2. Substrate transitive coupling**
D4 forbids importing `internal/sdp-*` but allows substrate cores (`sdp-evidence-core`, `sdp-policy-core`). However, substrate cores may themselves carry transitive SDP assumptions (context objects, SDP-specific types, SDP runtime hooks). If `sdp-evidence-core` assumes an SDP context object, importing it transitively couples the IIP to SDP's runtime model. No claim addresses substrate API surface auditing or SDP-assumption documentation in substrate AGENTS.md files.

**3. Chicken-and-egg adoption suppression**
Extraction criteria (D6) require external non-SDP consumers, but the incubation environment (monorepo under `sdp-lab`, no separate Homebrew, no landing page) actively suppresses non-SDP adoption. The tool may never prove independence because the incubation environment itself signals "this is an SDP tool." This creates permanent incubation limbo — too SDP-associated to attract independent users, too independent to get full SDP marketing support.

---

## 3 MOST IMPORTANT CORRECTIONS BEFORE SHIPPING

**1. D8: Fix factual accuracy immediately**
Change "Two flagship IIP candidates today" to "Two IIP hypotheses under evaluation." This is the single most credibility-damaging claim in the proposal. Stakeholders will discover the tools don't exist and lose trust in the entire memo.

**2. D4: Add implementation dependency and interim enforcement**
State explicitly: "CI lint enforcement requires completion of WS 00-150-04. Until then, import isolation is enforced via mandatory manual dependency audit in every PR, with a checklist of forbidden packages." This prevents the false confidence that isolation is already enforced.

**3. D5: Standalone go.mod from day one, defer distribution to extraction**
Mandate standalone `go.mod` (no `replace` directives) as a non-negotiable architectural constraint. Defer Homebrew formula, separate landing page, and separate installer to extraction event. This eliminates the monorepo exit problem (Architect's domain veto) while avoiding premature CI overhead (Technician's matrix concern).

---

## FINAL RECOMMENDATION

**ACCEPT WITH CHANGES**

The proposal's direction is sound: tools with plausible independent value should be incubated without subordination. But the current claims conflate aspiration with evidence, assert enforcement for unimplemented infrastructure, and create governance gaps. The following changes are required before shipping:

1. **D2**: IIP is a flag in Toolbox, not a new taxonomy row
2. **D4**: Acknowledge lint is aspirational; add manual audit interim
3. **D5**: Standalone go.mod mandatory; defer Homebrew to extraction
4. **D6**: Raise consumer bar; drop domain requirement
5. **D8**: Fix "flagship candidates today" → "hypotheses under evaluation"
6. **D9**: Name IIP leads; specify delivery ownership
7. **D10**: Add governance/brand ownership transfer plan
8. **D11**: Add cost-allocation model and archival trigger
9. **D12**: Add explicit downgrade criteria

With these changes, Option D becomes a disciplined incubation model rather than premature infrastructure for nonexistent code.