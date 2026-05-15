# Cross-Review: Agent Skill Operating Rules × Harness Engineering Landscape

**Reviewer:** C (product/runtime adoption)  
**Date:** 2026-05-15  
**Verdict:** `needs_work` — both docs are useful research, but neither is ready for adoption without prioritization cuts, existence proofs, and vendor-claim quarantine.

---

## Blocking Gaps (must close before adoption)

### B1 — Skill existence proof for Rule 3
Rule 3 says to add anti-rationalization tables to `build`, `review`, `ship`, `debug`, `delivery-loop`, and `spec-interrogate`. None of these skills are referenced by path in the operating-rules doc. The adoption backlog mentions Rule 3 but does not confirm these skills exist in the SDP skill inventory (presumably `docs/reference/skills.md`). **Status:** `not_assessed` — need a manifest scan to confirm which exist, which are aspirational, which are package-provided.

### B2 — No adoption priority
The adoption backlog is a flat list of 7 items across both documents. A maintainer cannot triage without knowing: what blocks what, what is a one-hour change versus a week-long spike, and what is cosmetic versus structural. **Status:** `needs_priority_map`.

### B3 — Runtime-permission semantics conflict with Pi's design posture
Harness doc consequence #1 says "add runtime permission semantics to the manifest." The harness doc itself acknowledges Pi's core is intentionally minimal and expects permissions to come from packages or external process controls. The operating-rules doc Rule 12 says "policy belongs in the runtime." No reconciliation is provided: does the manifest declare permission requirements that a Pi package enforces externally, or does the manifest itself become a runtime policy engine? **Status:** architectural conflict, `not_resolved`.

### B4 — Multi-plane review (Rule 9) vs. cost
Rule 9 proposes up to six review planes (correctness, requirements, evidence, security, docs, ops/CI). Rule 4 adds a doubt cycle with adversarial review and three maximum cycles. Combined, a single non-trivial claim could spawn 18 review invocations before a verdict. No cost, latency, or model-selection budget is defined. This is unusable for day-to-day SDP work as written. **Status:** `not_feasible_without_scoping`.

---

## Major Gaps

### M1 — Doubt cycle overlap
Rule 4 (doubt-driven: CLAIM/EXTRACT/DOUBT/RECONCILE/STOP) and Rule 8 (tests are concrete doubt) and Rule 9 (multi-plane review) are all doubt mechanisms for overlapping situations. The doc does not define when to use which, or whether they compose. A bug fix, for instance, might trigger all three simultaneously. **Status:** `not_disambiguated`.

### M2 — "Degraded evidence" is undefined
The harness doc consequence #8 says "encode degraded evidence explicitly" and lists examples (failed provider, timeout, empty output, etc.). The operating-rules doc mentions `not_assessed` state. No schema, type, or storage location is specified. Is `not_assessed` a Beads field, a review verdict tag, or a manifest state? **Status:** `not_assessed`.

### M3 — Model routing claims unevaluated on SDP tasks
The harness doc has routing hypotheses for five model families but explicitly says (consequence #10) "measure harness behavior on SDP tasks; do not assume benchmark rankings transfer." The doc does not contain any such measurements. Every routing hypothesis should carry a "validated: yes/no/not_assessed" flag before being actionable. **Status:** `not_assessed` for all five.

### M4 — Context layers (Rule 6) lack concrete triggers
Rule 6's five context layers describe *what* to load progressively but not *when* to promote to the next layer or how to detect starvation vs. flood. The harness doc discusses context compaction but does not offer thresholds or signals. **Status:** `operationalized_not_feasible`.

---

## Vendor Hype Quarantine

These claims are flagged `vendor_claim` and should not drive SDP policy until independently validated:

| Claim | Source | Risk if accepted uncritically |
|---|---|---|
| GLM-5.1 "8-hour sustained autonomous work" | Z.AI docs | Could justify too much unattended execution; contradicts bounded-workflow doctrine |
| Kimi "agent swarm" and "Claw Groups" | Kimi product page | Swarm coordination is orthogonal to SDP's Beads model; no evidence of interoperability |
| DeepSeek V4 "competitive but not uniformly SOTA" | Hugging Face blog | Third-party interpretation, not the primary source; DeepSeek-V4 official English docs noted as `not_assessed` |
| Qwen3.6-35B-A3B "optimized for agentic coding" | AWS/Alibaba docs | Marketing characterization; the 3B active parameter count may produce brittle long-range reasoning |
| GPT-5.5 "strongest fit for hard synthesis" | OpenAI migration guidance | Migration docs say this; independent evals on SDP tasks are `not_assessed` |

The harness doc already says "vendor claims are useful for routing hypotheses, not for merge or release authority." This discipline must be carried forward into any synthesis.

---

## Useful Tensions

These are productive disagreements worth preserving in synthesis:

1. **Pi minimalism vs. runtime safety** (Harness doc vs. Rule 12): Pi intentionally lacks built-in MCP, subagents, and permission popups. Rule 12 demands runtime-enforced safety. The synthesis doc should say explicitly: Pi packages like `pi-agents`/`pi-agent-flow` are the intended enforcement point, not Pi core.

2. **Small slices vs. long-horizon models** (Rule 7 vs. Harness §3): The harness doc identifies long-horizon execution as the new boundary, while Rule 7 insists on small revertable slices. These are compatible — small slices *within* a long-horizon bounded flow with budgets and review stops — but the synthesis doc must articulate this reconciliation.

3. **Skill workflow vs. module-local convention** (Rule 2 vs. AGENTS.md): Rule 2 says skills should not contain module-local facts (package APIs, import rules, etc.) — those go in `AGENTS.md`. This is correct but introduces a loading-order dependency: the agent must read `AGENTS.md` before skill execution. The operating-rules doc should specify this explicitly.

---

## Concrete Changes for the Synthesis Document

These are specific edits to make in whichever doc becomes the synthesis:

1. **Add an existence table** for every skill named in Rule 3: `build`, `review`, `ship`, `debug`, `delivery-loop`, `spec-interrogate`. Columns: `exists_in_repo` (yes/no), `path_if_exists`, `status` (stable/experimental/missing). If missing, move from adoption to "design needed."

2. **Replace the flat adoption backlog** with a phased table:
   - Phase 1 (this week): Rule 1 description template enforcement, manifest lint skeleton.
   - Phase 2 (this sprint): Anti-rationalization in `build` and `review` only.
   - Phase 3 (next sprint): Simplification as a review dimension; degraded-evidence schema draft.
   - Phase 4+: Remaining items, dependent on Phase 2 measurements.

3. **Add a `routing_confidence` column** to the model landscape table. Values: `vendor_only`, `local_spike`, or `validated_on_sdp_tasks`. All current entries should be `vendor_only`.

4. **Resolve B3 explicitly**: Add a paragraph stating manifest declares permission *requirements*, and enforcement is delegated to runtime (Pi packages, OpenCode permissions, or external gate). No new runtime engine is being built in SDP manifest.

5. **Scope Rule 9 multi-plane review**: Specify a default of 2-3 planes per workstream. Security and evidence planes are mandatory only for write-capable actions. The six-plane matrix is a reference library, not a default checklist.

6. **Disambiguate doubt mechanisms**: Add a decision tree to the synthesis doc:
   - Behavioral code change → Rule 8 (tests first)
   - Prompt/agent/model/skill/policy change → Rule 4 (doubt cycle)
   - Trust-sensitive workstream → Rule 4 + subset of Rule 9 planes
   - Everything else → standard review (single plane)

7. **Mark all GLM/Kimi/DeepSeek routing hypotheses** with a `vendor_claim` flag and remove them from the "Consequences for SDP" section. They belong in a separate routing-research doc, not in a document claiming to state consequences.

8. **Define `not_assessed`**: Add one paragraph or a small schema describing where `not_assessed` appears (review verdict, Beads state, evidence artifact), how it displays in reports, and that it is not equivalent to `pass`.

---

## Summary

| Dimension | Assessment |
|---|---|
| Operating rules quality | Good operational discipline; needs scoping and prioritization |
| Harness landscape quality | Good survey; routing hypotheses are under-evaluated for SDP |
| Adoption readiness | Neither doc is ready as-is; synthesis needs cuts |
| Vendor hype exposure | Managed but not quaranted; routing claims need confidence tags |
| Internal consistency | Three unresolved tensions (B3, B4, M1) |
| **Overall verdict** | `needs_work` → synthesize with priority map, scope cuts, and `not_assessed` markings |
