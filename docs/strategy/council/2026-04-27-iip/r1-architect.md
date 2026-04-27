# R1 — Architect

Model: `xiaomi/mimo-v2.5`  
Fallback used: `True`  
Elapsed: 39.8s  
OK: `True`  
Finish: `stop`  
Usage: `{'prompt_tokens': 2474, 'completion_tokens': 3530, 'total_tokens': 6004, 'cost': 0.00798816, 'is_byok': False, 'prompt_tokens_details': {'cached_tokens': 192, 'cache_write_tokens': 0, 'audio_tokens': 0, 'video_tokens': 0}, 'cost_details': {'upstream_inference_cost': 0.00798816, 'upstream_inference_prompt_cost': 0.00092816, 'upstream_inference_completions_cost': 0.00706}, 'completion_tokens_details': {'reasoning_tokens': 38, 'image_tokens': 0, 'audio_tokens': 0}}`

---

# Architect Review — Option D Claims D1–D12

---

## D1: Inherent outside-SDP value, no subordination

**VERDICT: OK**

**EVIDENCE:** The claim is directionally correct — arch-snap and doc-tracer have plausible non-SDP buyer personas. However, "inherent value" is asserted, not demonstrated. Today these tools do not exist in code. We are pre-assigning standalone value to hypotheses. The risk is premature optimization: we build isolation scaffolding for tools that may never find product-market fit outside SDP, while the SDP funnel itself remains unvalidated. That said, the cost of naming without `sdp-` prefix at inception is near-zero, and renaming later is expensive. So the precautionary principle supports this claim even without proof of independent demand.

**PROPOSAL:** None needed — but the claim should carry a caveat that if neither tool reaches 2 external weekly consumers within 6 months, the IIP status is revoked and the tool is archived or re-subordinated.

---

## D2: New taxonomy row "Incubated Independent Products"

**VERDICT: WEAK**

**EVIDENCE:** Adding a taxonomy row between Lab and Toolbox creates a 3-tier incubation ladder (Lab → IIP → Extraction) where today a 2-tier ladder (Lab → Toolbox) suffices. The problem is organizational: memo v2 just landed with the Toolbox row as a deliberate simplification. Introducing a new row immediately after council consensus signals instability in the taxonomy itself. More concretely, the distinction between "Lab experiment" and "IIP" is blurry when neither tool has code. What makes a Lab experiment an IIP? The `independent_value: yes` annotation (D3) — but that's a self-declared boolean, not a validated signal. This row risks becoming a prestige category that tools enter but never leave.

**PROPOSAL:** Fold IIP status into the existing Toolbox row as a flag (`standalone: true`) rather than creating a new taxonomy row; promote to separate repo only when extraction criteria (D6) are met.

---

## D3: `independent_value: yes` annotation in AGENTS.md

**VERDICT: WEAK**

**EVIDENCE:** A self-declared boolean in a 60-line AGENTS.md is not a meaningful architectural constraint. It's metadata theater. The real question is whether the code actually has no SDP runtime dependencies — and that's enforced by lint (D4), not by an annotation. Furthermore, writing AGENTS.md "as if SDP did not exist" is a good aspiration but unverifiable: if the tool imports `sdp-evidence-core`, the AGENTS.md is already lying by omission. The annotation adds cognitive overhead without adding enforcement.

**PROPOSAL:** Replace the annotation with a CI-enforced dependency audit: if a module's `go.mod` imports zero `internal/sdp-*` packages and ≤2 `sdp-*-core` substrate packages, it qualifies as independent; no human annotation needed.

---

## D4: Import restrictions + CI lint enforcement

**VERDICT: STRONG**

**EVIDENCE:** This is the most architecturally sound claim. The three-tier import model (forbidden: `internal/sdp-pr-gate`, `internal/sdp-operator`, `internal/sdp-edg`; allowed-with-pinning: substrate cores; forbidden-by-default: everything else) is enforceable and aligns with Go module boundaries. The pinned semver requirement for substrate imports prevents silent breakage. Precedent: Kubernetes subprojects (e.g., client-go, controller-runtime) enforce similar import boundaries via `go.mod` and CI checks. The risk is that substrate cores themselves carry transitive SDP assumptions (e.g., `sdp-evidence-core` may assume an SDP context object), but that's a substrate design problem, not an IIP taxonomy problem.

**PROPOSAL:** None — but substrate authors must be required to document SDP-assumptions in their own AGENTS.md so IIP consumers know what they're pulling in transitively.

---

## D5: Own go.mod, semver prefix, Homebrew formula, README

**VERDICT: OK**

**EVIDENCE:** Go module sub-paths (`github.com/org/sdp_lab/arch-snap`) with independent semver tags are well-supported by Go tooling — this is how golang.org/x/ packages and Kubernetes sub-modules work. The Homebrew formula separation is correct for brand independence. However, "own README and landing copy that does not reference SDP unless contextually necessary" is vague and creates ongoing editorial burden. More importantly, the monorepo-to-multi-repo extraction path (when D6 criteria are met) requires splitting `go.mod` files, which is non-trivial if substrate imports have accumulated. The longer a tool lives in the monorepo, the harder extraction becomes — this is the classic monorepo exit problem (see: Bazel's internal-to-external migration at Google, still incomplete after 8+ years).

**PROPOSAL:** Mandate that each IIP's `go.mod` is a standalone module from day one (not a `replace` directive pointing at monorepo paths), so extraction is a repo-move, not a module-refactor.

---

## D6: Extraction criteria (2+ external consumers, 50% non-SDP, revenue signal, brand finalized)

**VERDICT: OK**

**EVIDENCE:** The criteria are specific and measurable, which is good. However, ">= 2 external consumers using it weekly" is an extremely low bar — two people could be the same company's two employees. ">= 50% of those consumers do NOT use SDP" with a denominator of 2 means 1 non-SDP consumer triggers extraction eligibility. This is statistically meaningless. The "distinct revenue or adoption signal" criterion is also circular: if the tool is free and in a monorepo, how do you measure adoption independent of SDP? Download counts? GitHub stars? These are vanity metrics. The real signal is someone willing to pay or contribute.

**PROPOSAL:** Raise the bar: extraction requires (a) ≥10 weekly active external users OR ≥1 paying customer, (b) ≥50% non-SDP usage at that scale, (c) a second organization (not the author's) willing to co-maintain or sponsor.

---

## D7: Toolbox row narrows to SDP-onboarding-only tools

**VERDICT: OPPOSE**

**EVIDENCE:** This claim retroactively reclassifies tools that were just placed in Toolbox by memo v2 consensus. Specifically, `doc-tracer` and `arch-snap` were listed as Toolbox examples in memo v2. Moving them to IIP immediately after council consensus undermines the stability of the taxonomy we just agreed on. More substantively, the claim assumes we can cleanly separate "SDP-onboarding value" from "independent value" — but many tools will have both. `sdp-scout` could be useful for non-SDP codebases. `sdp-metrics` could be a standalone observability tool. Drawing a hard line at "fundamentally tied to SDP onboarding" creates an arbitrary boundary that will need constant re-litigation.

**PROPOSAL:** Toolbox tools should be classified by their PRIMARY go-to-market, not by an absolute value judgment; allow tools to carry a `dual-track: true` flag if they have plausible independent value, without requiring a new taxonomy row.

---

## D8: Two flagship ICP candidates

**VERDICT: OK**

**EVIDENCE:** The ICP lists are plausible and well-researched. Due-diligence/M&A buyers for arch-snap and FDA/ISO 13485 compliance teams for doc-tracer are genuine buyer personas with budget authority. However, these are hypotheses, not validated segments. The risk is that we're building product strategy around ICPs that sound good in a memo but haven't been interviewed. "Tech writers building docs" is a notoriously low-budget persona. "Security architects" may already have tools (SonarQube, CodeScene, Structure101) that do architecture extraction. The competitive landscape is not addressed.

**PROPOSAL:** Before committing to IIP infrastructure, run 5 customer discovery interviews per tool with the named ICPs to validate willingness-to-pay and current alternatives.

---

## D9: Own beads epic outside F150

**VERDICT: STRONG**

**EVIDENCE:** This is correct project management. F150 should enforce the architectural rules (isolation lint, AGENTS.md cascade, annotations) but should not be responsible for shipping IIP features. Mixing IIP feature work into F150 would create scope creep and muddy the sprint. Keeping epics separate also makes it easier to kill an IIP without affecting the core SDP roadmap. Precedent: Google's Area 120 incubator runs parallel to core product sprints; Alphabet's Other Bets are tracked separately from Google Cloud.

**PROPOSAL:** None — this is well-structured.

---

## D10: SDP team as maintainer, not brand owner; permissive license default

**VERDICT: WEAK**

**EVIDENCE:** "SDP team is the maintainer, not the brand owner" is contradictory in practice. If the SDP team writes the code, reviews the PRs, and triages the issues, they ARE the brand owner in every meaningful sense — regardless of what the LICENSE file says. The claim that "SDP gets no preferential treatment in marketing positioning" is aspirational but unenforceable: who decides what the landing page says? The SDP team. The claim about cross-licensing or spinning out to a new entity is premature — we don't have a tool yet, let alone a licensing strategy. Apache-2.0 default is fine, but the "unless a specific commercial track requires otherwise" escape clause could be used to re-subordinate the tool under SDP commercial terms at any time.

**PROPOSAL:** If the intent is genuine independence, commit to a governance model: IIPs with >50% external contributors get an independent steering committee; SDP team retains maintainer role but not unilateral brand control.

---

## D11: Independent pricing per IIP

**VERDICT: OK**

**EVIDENCE:** Correct in principle. Pricing independence prevents cross-subsidization distortions and allows each tool to find its own willingness-to-pay curve. The risk is organizational: if the SDP team is the only team working on IIPs, they will implicitly optimize for SDP revenue even if the IIP pricing is "independent." There's also a practical concern: if an IIP stays free/open-source forever (as D11 allows), what's the business case for the engineering investment? This claim needs a companion claim about how IIP engineering costs are allocated.

**PROPOSAL:** Add: "Each IIP must have an explicit cost-allocation model (engineering hours, infra, support) reviewed quarterly; if an IIP cannot justify its cost within 12 months, it is archived or re-subordinated."

---

## D12: Cap of 3 active IIPs; founder approval required

**VERDICT: STRONG**

**EVIDENCE:** The cap of 3 is sensible for an organization that doesn't yet have a single shipped IIP. It prevents the "innovation theater" pattern where every Lab experiment gets promoted to IIP status to signal activity. Founder/owner approval as a gate is appropriate at this stage — democratic promotion processes are for organizations with 50+ engineers, not for a small team. The exit criteria (extracted, archived, or downgraded) are clear. The only risk is that the cap becomes a political football: "we have 3 IIPs but #2 is dying, can we promote #4 before killing #2?" The answer should be no.

**PROPOSAL:** None — well-designed constraint.

---

## MINORITY REPORT

**D7 (Toolbox narrowing):** I predict I will disagree with the majority on this claim. The majority may accept D7 because it seems clean and logical. But D7 is the most dangerous claim in the set because it forces a binary classification (SDP-dependent vs. independent) on tools that exist on a spectrum. It will create constant re-classification debates and undermine the taxonomy's stability. The majority will likely accept it because it's presented as a natural consequence of D1-D2, but it's actually a separate and more contentious decision.

---

## DOMAIN VETOES

**[DOMAIN VETO: D5 — go.mod sub-path without standalone module]**
If IIP tools use `replace` directives pointing at monorepo-internal paths, extraction to a separate repo becomes a module-refactoring exercise, not a repo-move. This is architecturally unacceptable. The go.mod must be standalone from day one, or extraction will be delayed by months of dependency untangling.

---

## THREE BIGGEST RISKS

1. **Premature infrastructure for nonexistent code.** Arch-snap and doc-tracer do not exist yet. We are designing a sophisticated incubation-to-extraction pipeline for hypotheses. The entire IIP taxonomy, lint rules, AGENTS.md conventions, and semver mechanics could be built and then never used if the tools don't find PMF. The cost of this scaffolding is non-trivial: new lint rules, new CI checks, new AGENTS.md templates, new semver tag conventions, new beads epics. All of this is overhead for zero shipped code.

2. **Monorepo exit problem.** The longer IIP tools live in `sdp_lab` monorepo, the more they accumulate implicit dependencies on monorepo conventions, CI pipelines, shared tooling, and developer habits. Extraction becomes harder with every commit. The D6 criteria (2+ external consumers) may take 12-18 months to meet, by which time the tool is deeply entangled. Precedent: Bazel, Buck, and Pants all struggled with internal-to-external extraction despite being built by companies with far more resources than an SDP lab.

3. **Organizational fragmentation without organizational capacity.** Three IIPs with independent epics, independent pricing, independent branding, and independent AGENTS.md files require independent product management attention. If the SDP team is 3-5 people, they cannot simultaneously run SDP core, SDP Toolbox, and 3 IIPs. The cap of 3 (D12) helps, but even 1 IIP is a significant distraction from the core SDP product, which itself is unvalidated.

---

## PRECEDENT REFERENCES

1. **etcd (from CoreOS):** Started as CoreOS's internal cluster coordination tool. Extracted to独立项目, became CNCF graduated project. **Lesson:** etcd succeeded because it solved a universal problem (distributed key-value store) that was genuinely independent of CoreOS's container Linux product. The `sdp-` prefix equivalent ("coreos-etcd") was dropped early. **Key factor:** etcd had massive external adoption (Kubernetes depended on it) before formal extraction. Arch-snap and doc-tracer have zero external adoption today.

2. **containerd (from Docker):** Spun out of Docker's monolithic runtime into a CNCF project. **Lesson:** containerd extraction was forced by competitive pressure (Kubernetes ecosystem rejected Docker's proprietary runtime). The extraction was painful and took 2+ years because of deep internal coupling. **Key factor:** containerd had a clear external forcing function (Kubernetes adoption). IIP tools have no such forcing function — extraction will be purely voluntary, which means it will be perpetually deprioritized.

3. **Ginkgo/Gomega (from onsi/ginkgo, Cloud Foundry ecosystem):** Ginkgo testing framework grew out of the Cloud Foundry Go ecosystem but was always a separate repo. **Lesson:** Ginkgo succeeded as an independent tool because it was never subordinated to a parent product's branding or funnel. It was a standalone tool from day one. **Key factor:** The "incubate in monorepo then extract" pattern was never needed because the tool was always independent. This suggests that if arch-snap and doc-tracer truly have independent value, they should be standalone from day one (Option A), not incubated under SDP (Option D). Option D may be the worst of both worlds: enough independence to create overhead, not enough to capture the standalone market.