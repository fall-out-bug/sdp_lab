# R1 — Critic

Model: `x-ai/grok-4.20`  
Fallback used: `False`  
Elapsed: 24.7s  
OK: `True`

---

**C1: VERDICT: WEAK**  
**EVIDENCE:** Claim that Operator Mode is “just an advanced Toolkit feature” contradicts the repo reality note that Operator Mode loop is already GA inside sdp_lab while Toolkit is only GA and the multi-harness install is still Beta. Treating it as surfacing only after Beads+workstreams+evidence collection creates hidden coupling and raises process-theater risk: the team will declare victory on “Toolkit” while the actual operator loop remains the only thing customers care about.  
**PROPOSAL:** Operator Mode is a distinct product surface that can be cold-started independently; its current implementation inside Toolkit is a temporary co-location, not architectural subordination.

**C2: VERDICT: OPPOSE**  
**EVIDENCE:** Naming five concrete utilities (doc-tracer … doc-analyzer) “Standalone Tools — a first-class new product category” is classic process theater. They are currently scripts inside one monorepo; promising they “may eventually live in their own repos” is not evidence of productization. This broad framing hides the absence of packaging, versioning, separate telemetry, pricing, or customer validation.  
**PROPOSAL:** The five utilities remain experimental Toolkit extras until each has its own repo, semver contract, and at least one external pilot.

**C3: VERDICT: VETO**  
**EVIDENCE:** The “architectural meta-rule” of every surface cold-startable from ≤60-line AGENTS.md is elegant but completely unsubstantiated by any working example; the current 606-line root already violates it and no migration plan exists inside F150. This is pure process theater that will be used to reject any pragmatic deviation while the team quietly keeps the 606-line file.  
**PROPOSAL:** [DOMAIN VETO: architectural purity theater without executable proof]

**C4: VERDICT: WEAK**  
**EVIDENCE:** “Two parallel commercial wedges” assumes free dev adoption via Homebrew will automatically convert to paid ChangePassport GitHub PR Gate without any evidence of conversion funnel, pricing, or buyer validation. Prior council output already stated that “inside enterprise perimeter alone is NOT unique”; claiming Wedge B as first paid without addressing that risk is wishful framing.  
**PROPOSAL:** Paid wedge must be validated with at least one committed pilot before declaring parallel commercial strategy.

**C5: VERDICT: OK**  
**EVIDENCE:** Reserving a slot without including it in F150 scope matches prior enterprise-perimeter research that the real differentiator is governed protocol, not perimeter control. Acceptable containment.

**C6: VERDICT: STRONG**  
**EVIDENCE:** Russian sovereign adapters are correctly scoped as parallel F-track; mixing them into F150 would explode complexity and contradict the “neutral governed delivery protocol” strategy.

**C7: VERDICT: WEAK**  
**EVIDENCE:** Keeping placeholder name “ChangePassport” until domain, trademark, ICP, and council tests pass is fine, but the claim omits that the name itself carries strong negative connotations in security/governance contexts (“passport” implies border control). No evidence that buyer language test was run with actual enterprise delivery leads.  
**PROPOSAL:** Replace working name before external pilots; run language test with target ICP buyers immediately.

**C8: VERDICT: STRONG**  
**EVIDENCE:** Explicit semver packaging for shared substrates is the only responsible way to avoid future breakage; claim is evidence-based and aligns with replacement-risk analysis.

**C9: VERDICT: STRONG**  
**EVIDENCE:** Repo split as downstream event after Schema v1 + Evidence Provider API v1 + pilot is disciplined and matches prior council output. Correctly avoids upfront architecture theater.

**C10: VERDICT: OPPOSE**  
**EVIDENCE:** Keeping exactly 10 workstreams “without renumbering” and stuffing two new items into 00-150-09 is classic process theater that prioritizes tidy numbering over actual priority. The cascade AGENTS.md migration is architectural debt, not a side-car for workstream 9.  
**PROPOSAL:** Renumber or create two new workstreams; migration plan cannot be deferred inside an existing bucket.

**C11: VERDICT: WEAK**  
**EVIDENCE:** Mixing pilot-stage targets with GA SLOs in one list and claiming “discernment metrics” without baselines, measurement method, or statistical confidence is weak evidence. “Reviewer time -20%” and “useful suggestion ≥30-40%” lack any pilot data or competitive benchmark.  
**PROPOSAL:** Split pilot success metrics from GA SLOs and provide measurement plan for each before F150 starts.

**C12: VERDICT: OPPOSE**  
**EVIDENCE:** Treating Operator Mode as “Toolkit Happy Path” while repo reality shows it is already GA contradicts C1 and creates strategic risk: if buyers want the operator loop, we will be forced to productize it under the wrong name and packaging. Prior council emphasized governed delivery as core; Operator Mode is the clearest embodiment of that. Defaulting to “not a SKU” without customer signal plan is exactly the risk the author invited us to challenge.  
**PROPOSAL:** Operator Mode is a first-class surface and candidate SKU; decide SKU status before external pilots, not after “signals emerge”.

**MINORITY REPORT:**  
I will disagree with majority on C3 and C12. C3 is unacceptable architectural theater that will paralyze delivery. C12 misclassifies the actual product customers will buy.

**DOMAIN VETOES:**  
[DOMAIN VETO: C3 — architectural purity theater without executable proof]

**THREE BIGGEST RISKS** the document fails to address:  
1. No validated buyer demand or pilot for ChangePassport as paid wedge (replacement risk remains 70-85%).  
2. Operator Mode is already the real product surface but is being buried as “Toolkit feature” — classic feature-vs-product confusion.  
3. Absence of concrete migration plan for 606-line AGENTS.md root turns the elegant cascade rule into performative process theater.