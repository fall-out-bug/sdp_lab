# Cross-Review: Operating Rules vs. Harness/Model Landscape

**Reviewer:** A (harness/runtime engineer)
**Scope:** Whether Rules 1–12 are sufficient given the landscape document's findings
**Date:** 2026-05-15

---

## Verdict

**Conditional pass.** The rules are internally consistent and address most skill-authoring discipline. They are *insufficient* for the harness/runtime concerns the landscape document identifies as the defining 2026 shift: model routing as a first-class concern, long-horizon flow lifecycle, tool side-effect classification, degraded-evidence protocol, and harness adapter parity. These are not edge cases—they are the central claim of the landscape doc (§Executive Summary: "coding agents are becoming managed runtimes").

---

## Blocking Gaps

Gaps where a rule claims coverage it does not have, or where a landscape-mandated concern has no rule at all.

### B1. No rule addresses model routing as a structural requirement

The landscape identifies model routing by role (scout, planner, implementer, reviewer, security, synthesis, judge) as a harness-level concern. Rule 10 bans single-vendor reviewer panels but says nothing about:

- declaring model capability prerequisites in skill triggers;
- tracking endpoint provenance and model version in evidence artifacts;
- requiring model diversity for specific review planes;
- routing degraded or fallback behavior when a preferred provider is unavailable.

Rule 5 cites specific products for source verification but does not extend to model selection. Rule 12 enumerates runtime controls but omits model/provider allowlists.

**Impact:** Without this, skills can silently route all work to one model with no evidence trail and no diversity requirement. The landscape's consequences §items 4–6 are unaddressed.

### B2. No rule addresses long-horizon flow lifecycle

The landscape identifies budgets, checkpoints, compaction, and review stops as essential for long-horizon harness work. Rule 7 mandates small slices (good for leaf work) but says nothing about:

- when to start a fresh session vs. continue (landscape §3: within-session compliance drops);
- compaction or memory management policy;
- flow primitives (spawn, sequence, fork, join, loop) and their governance;
- harness session lifecycle (create, run, release) as a skill concern.

**Impact:** The landscape explicitly warns against "one giant autonomous run" and for bounded workflows with checkpoints. The rules have no structural countermeasure for sessions that run long beyond "slice smaller."

### B3. Tool side-effect classification is absent from all rules

The landscape §4 and consequences §item 9 call for classifying tools into five categories (perception/read, analysis, local write, external write, irreversible/identity-mediated) with separate permission and evidence policy. Rule 12 mentions per-agent tool permissions but does not require skills to declare which side-effect classes they exercise or to accumulate differentiated evidence per class.

**Impact:** A skill that performs external writes is governed by the same prose as one that only reads. The landscape identifies this as a growing risk category.

---

## Major Gaps

Important but not blocking synthesis; can be addressed as amendments.

### M1. Degraded-evidence states are under-specified

Rule 3 mentions `not_assessed` for empty/hung reviewer output. The landscape (consequences §item 8) enumerates six degraded states: failed provider, timeout, empty output, unavailable CLI, unverified benchmark, and not-assessed runtime. No rule makes these a general principle or requires skills to handle/report them.

### M2. Rule 9 (multi-plane review) missing two planes

Current planes: code correctness, requirements, evidence/tracing, security/PI, docs/runtime truth, ops/CI/release. Missing:

- **model/provider routing and provenance plane** — did the right model see the right slice with the right evidence chain?
- **tool-side-effect policy plane** — did the skill stay within its declared side-effect classes?

### M3. Rule 1 (triggers) missing runtime precondition declarations

Triggers specify *when* to use a skill but not *what runtime controls* the skill requires (sandbox mode, network access, specific tool classes, model capabilities). A skill whose workflow calls external APIs has different precondition requirements than one that only reads local files.

### M4. Rule 3 anti-rationalization table missing harness-specific entries

Absent rationalizations the landscape implies:

| Rationalization | Reality |
|---|---|
| "The harness sandbox covers it." | Sandbox is necessary but not sufficient; prompt-injection and capability-intent mismatch persist inside sandboxes. |
| "This model passed our evals." | Benchmark performance does not transfer to SDP task workflows without local measurement. |
| "The agent had network access so it verified." | Network access is a permission, not evidence of verification. |
| "Context window is large enough to skip compaction." | Within-session compliance degrades regardless of window size. |

### M5. No rule addresses harness adapter parity

The landscape (consequences §item 2) calls for separating static adapter parity (file presence) from runtime dispatch evidence (did the harness actually route as configured?). No rule requires verifying that an adapter's runtime behavior matches its spec.

---

## Useful Tensions

Tensions that are features, not bugs—worth preserving in synthesis.

### T1. Rule 12 vs. prompt-only minimalism

Rule 12 ("runtime beats prompt") directly contradicts any temptation to solve safety by writing better instructions. The landscape reinforces this repeatedly. This tension is correct and should be sharpened, not softened.

### T2. Rule 7 (small slices) vs. landscape long-horizon models

Rule 7 mandates thin-slice implementation. The landscape describes models claiming 8-hour autonomous runs. The tension is real: long-horizon capability exists, but SDP should still slice. The synthesis should acknowledge this explicitly and state that long-horizon models are for executing *many bounded slices in sequence*, not for eliminating slice discipline.

### T3. Rule 10 (no persona trees) vs. landscape swarm/flow patterns

Rule 10 restricts orchestration patterns. The landscape describes swarm capabilities (Kimi Claw Groups) and flow graphs (pi-agent-flow). The tension is healthy: SDP should adopt flow-graph primitives (spawn, fork, join) without adopting recursive agent trees. The synthesis should draw this line clearly.

### T4. Rule 5 (source-driven) product list vs. ecosystem volatility

Rule 5 hardcodes specific products (OpenAI/Codex, OpenCode, Pi). The landscape shows this ecosystem changes fast. Useful tension: keep the mandatory-source discipline but make the product list a versioned appendix, not a fixed rule body.

---

## Concrete Changes for Synthesis Document

1. **Add Rule 13 or extend Rule 12: Model routing and provenance.** Require skills to declare model capability prerequisites, require evidence to include model version/provider/endpoint, require model diversity for trust-sensitive review planes.

2. **Add Rule 14 or extend Rule 7: Long-horizon flow lifecycle.** Mandate session budgets, compaction triggers, checkpoint evidence, and explicit fresh-session criteria. State that long-horizon models execute bounded slices in sequence, not unbounded runs.

3. **Extend Rule 1: Runtime preconditions in triggers.** Add a `Requires` field covering sandbox mode, network access, tool side-effect classes, and minimum model capabilities.

4. **Extend Rule 2: Side-effect class declaration in workflows.** Require each workflow step to declare its tool side-effect class and for the skill's evidence to be differentiated by class.

5. **Extend Rule 3: Add harness-specific rationalizations** (sandbox sufficiency, benchmark transfer, network-access-as-verification, context-window sufficiency).

6. **Extend Rule 9: Add two review planes** — model/provider routing and provenance, and tool-side-effect policy.

7. **Extract degraded-evidence protocol as a cross-cutting concern** (applicable to all rules, not just Rule 3). Enumerate the six states from the landscape.

8. **Add Rule 15 or extend Rule 12: Harness adapter parity must be runtime-verified**, not just file-presence checked.

9. **Make Rule 5's product list a versioned appendix** with a review cadence, not a fixed rule body.

10. **Preserve tensions T1–T4 explicitly** in the synthesis with the framing given above.

---

## Evidence State

- Operating rules internal consistency: **assessed** (read in full).
- Landscape document accuracy against external sources: **not_assessed** (not in scope for this review).
- Whether proposed changes would be sufficient after adoption: **not_assessed** (requires implementation and measurement on real SDP tasks, per landscape consequence §item 10).
- Runtime feasibility of proposed changes in current Pi/Codex/OpenCode harnesses: **not_assessed** (would require adapter-specific prototyping).
