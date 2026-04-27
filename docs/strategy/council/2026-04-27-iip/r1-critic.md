# R1 — Critic

Model: `minimax/minimax-m2.7`  
Fallback used: `False`  
Elapsed: 96.7s  
OK: `True`  
Finish: `stop`  
Usage: `{'prompt_tokens': 2198, 'completion_tokens': 5303, 'total_tokens': 7501, 'cost': 0.00701532, 'is_byok': False, 'prompt_tokens_details': {'cached_tokens': 32, 'cache_write_tokens': 0, 'audio_tokens': 0, 'video_tokens': 0}, 'cost_details': {'upstream_inference_cost': 0.00701532, 'upstream_inference_prompt_cost': 0.00065172, 'upstream_inference_completions_cost': 0.0063636}, 'completion_tokens_details': {'reasoning_tokens': 1985, 'image_tokens': 0, 'audio_tokens': 0}}`

---

# CRITIC REVIEW — Option D (Incubate-Then-Spin-Out)

---

## CLAIM-BY-CLAIM VERDICTS

### D1: "Inherent value OUTSIDE SDP"

**VERDICT: WEAK**

**EVIDENCE:**
- These tools **do not exist in code**. They are "research / lab-only hypotheses in memo v2." Asserting "inherent value" for non-existent artifacts is speculation, not evidence.
- The ICPs cited (due-diligence/M&A buyers, security architects, CTOs, tech writers) are personas constructed from assumption, not from customer discovery interviews, waitlist signups, or GitHub stars.
- The claim that `sdp-` prefix "actively repels" buyers is unvalidated. Search behavior studies show developers often search by problem ("architecture extraction tool"), not brand. The prefix problem is solvable via SEO and landing-page strategy without structural separation.
- Precedent: `gcloud-*` tools from Google, `fb-*` libraries from Meta, and `aws-*` SDKs from Amazon all used company prefixes and achieved massive adoption. The prefix is a packaging detail, not a barrier.
- "Inherent value" is a label, not a measurement. Value is demonstrated by usage, not declared in a memo.

**PROPOSAL:** Replace "inherent value" with "hypothesized value requiring validation via 3+ external users within 90 days."

---

### D2: New taxonomy row "Incubated Independent Products"

**VERDICT: WEAK**

**EVIDENCE:**
- Memo v2 already has 7 layers. Adding an 8th ("IIP") increases taxonomy complexity before validating the products exist.
- "No `sdp-` prefix from inception" is a **branding decision**, not a technical architecture decision. This belongs to the Philosopher role, not the Critic's domain of evidence validation.
- The claim that prefix-less naming improves adoption is unproven. The author provides no A/B testing, no search volume data, no conversion funnel comparison.
- Precedent: `kubectl`, `helm`, `docker` — these succeeded not because they lacked company prefixes, but because they solved real problems and had strong communities.
- The taxonomy row is premature infrastructure for hypothetical products.

**PROPOSAL:** Add IIP as a **sub-row under Lab**, not as a top-level taxonomy layer. Promote to top-level only when extraction criteria (D6) are met.

---

### D3: `independent_value: yes` annotation; AGENTS.md written as if SDP didn't exist

**VERDICT: OPPOSE**

**EVIDENCE:**
- This is **circular reasoning**: you prove independent value by writing documentation that assumes independent value. The annotation is a label, not evidence.
- "No `sdp-` references in 60-line cold-start text" is an arbitrary constraint. Many successful ecosystem tools (React, Vue, ESLint) have extensive docs tying them to their ecosystems without being "subordinate."
- The cascade AGENTS.md rule is a **target, not current state** (root AGENTS.md is 606 lines, not ≤60). Enforcing sub-60 rules on hypothetical IIPs when the parent rule isn't implemented is premature.
- The annotation creates a **governance illusion**: tools can be labeled "independent" without earning that status through usage or revenue.
- This claim substitutes documentation formatting for actual market validation.

**PROPOSAL:** Replace annotation with **measurable criteria**: "independent_value: validated" requires evidence of ≥2 external non-SDP consumers.

---

### D4: IIP must not import SDP-runtime modules; substrate imports via pinned semver

**VERDICT: OK (premature)**

**EVIDENCE:**
- Technical isolation principle is sound architecture.
- **However**: package-level isolation lint is "planned for WS 00-150-04" — not implemented. `extractable: yes` annotation is "planned for WS 00-150-02" — not implemented.
- Enforcing isolation rules on infrastructure that doesn't exist is building compliance theater for hypothetical products.
- The CI lint enforcement is a promise, not a deployed control.
- This claim is architecturally correct but operationally hollow until the lint is built.

**PROPOSAL:** Add implementation dependency: "CI lint enforcement requires completion of WS 00-150-04. IIP isolation rules are aspirational until lint is deployed."

---

### D5: Each IIP gets own go.mod, semver tag, Homebrew formula, README

**VERDICT: WEAK**

**EVIDENCE:**
- Homebrew formula for **non-existent products** is premature operational overhead. Formula maintenance requires ongoing CI, versioning, and user support.
- Own semver tag prefix assumes these will be **versioned packages** — reasonable but not proven necessary at this stage.
- The operational burden of maintaining separate release infrastructure (Homebrew taps, separate README sites, separate landing pages) for 3 hypothetical IIPs is non-trivial.
- Precedent: Many tools start as internal modules before graduating to packages. Building separate release pipelines before validating demand is **build-it-and-they-will-come** thinking.
- The claim assumes extraction will happen. D6's extraction criteria are strict and hypothetical.

**PROPOSAL:** Defer Homebrew formula and separate README to extraction event (when D6 criteria are met). For incubation phase, use internal module with `// internal/ii p/arch-snap` path.

---

### D6: Extraction criteria stricter than Toolbox promotion

**VERDICT: WEAK**

**EVIDENCE:**
- Criteria sound reasonable in theory but are **unenforceable for non-existent products**.
- ">= 2 external consumers using it weekly" — how do you measure this for tools in Lab? No analytics infrastructure specified.
- ">= 50% of those consumers do NOT use SDP at all" — this requires **cross-product identity tracking** that doesn't exist. How do you attribute a GitHub clone to "SDP user" vs "non-SDP user"?
- "Distinct revenue or adoption signal independent of SDP funnel" — if these tools live in `sdp_lab` monorepo, how is the funnel separated?
- The criteria assume external consumers exist. They don't yet.
- No mechanism specified for **verifying** these criteria. Who counts? How?

**PROPOSAL:** Add verification mechanism: "Extraction criteria verified by: (1) GitHub stars from non-SDP accounts, (2) explicit opt-in survey question 'Do you use SDP?', (3) revenue attribution via separate Stripe product."

---

### D7: SDP Toolbox narrows to only SDP-tied tools

**VERDICT: OK**

**EVIDENCE:**
- This is a reasonable taxonomy cleanup. Moving "anything else with independent value" to IIP is acceptable.
- **However**: The claim assumes we know what "has independent value." The author is simultaneously arguing that doc-tracer and arch-snap have independent value (D1) while also saying we don't know yet (D6 criteria).
- The Toolbox narrowing is contingent on the IIP infrastructure being built. If D1-D6 are weak, this claim is also hollow.
- The claim is directionally correct but depends on unvalidated premises.

**PROPOSAL:** No revision needed, but flag as **contingent on D1-D6 validation**.

---

### D8: Two flagship IIP candidates today: arch-snap and doc-tracer

**VERDICT: VETO**

**EVIDENCE:**
- **FACTUAL ERROR**: "Today" these are **hypotheses, not products**. They don't exist in code. You cannot have "flagship candidates" for non-existent artifacts.
- The ICPs listed (due-diligence/M&A, security architects, CTOs, compliance teams) are **constructed personas**, not validated customers. No evidence of waitlist, no evidence of search volume, no evidence of competitor analysis.
- "Flagship" implies flagship *status* — a product with demonstrated adoption. These are Lab experiments.
- This claim **undermines the proposal's credibility** by asserting as current reality what is clearly future aspiration.
- The author is conflating "we plan to build these" with "these are flagship candidates today."

**PROPOSAL:** Revise to "Two IIP hypotheses under evaluation: arch-snap and doc-tracer. Neither exists in code. Validation required before IIP status."

---

### D9: Each IIP gets own beads epic outside F150

**VERDICT: WEAK**

**EVIDENCE:**
- Creating separate epics for **non-existent products** adds tracking overhead without delivery value.
- "F150 does NOT ship any IIP" — but if IIPs aren't in F150, **who owns their delivery**? The proposal creates a governance gap.
- F150 is the SDP delivery vehicle. If IIPs are outside F150, they have no committed delivery owner, no sprint allocation, no capacity planning.
- This is **governance theater**: separate epics with no clear ownership or delivery mechanism.
- The claim that F150 "ensures architectural rules" is vague. What does "ensures" mean? Audit? Automated check? Manual review?

**PROPOSAL:** Specify delivery ownership: "IIPs are owned by designated IIP leads outside F150. F150 provides architectural compliance checks only. IIP leads report to product council for capacity allocation."

---

### D10: IIP brand strategy — SDP team is maintainer, not brand owner

**VERDICT: WEAK**

**EVIDENCE:**
- "May eventually be cross-licensed, sponsored, or spun out" — these are **future possibilities**, not current strategy. The proposal doesn't specify triggers or decision criteria for these outcomes.
- If SDP team maintains the tool but can't claim it as their own, **what's the incentive**? Open-source maintainer burnout is well-documented. Without brand ownership, maintainer engagement degrades.
- "SDP gets no preferential treatment in marketing positioning" — but SDP team is the maintainer. In practice, SDP will be the marketing channel by default.
- The claim assumes a clean separation between maintainer and brand owner that rarely exists in practice.
- Precedent: Many "neutral" foundation projects struggle with brand identity because no single organization has skin in the game.

**PROPOSAL:** Add brand ownership criteria: "IIP brand ownership transfers when: (1) ≥100 external contributors, (2) ≥2 corporate sponsors with >$50K/year commitment, (3) dedicated IIP lead hired."

---

### D11: IIP pricing independent of SDP pricing

**VERDICT: OK**

**EVIDENCE:**
- This is reasonable in principle.
- **However**: "Some IIPs may stay free/open-source forever" is a non-strategy, not a strategy. It provides no decision framework.
- No mechanism specified for **when** pricing decisions are made. At extraction? At first external consumer? At revenue signal?
- The claim is directionally correct but operationally empty.
- "Independent willingness-to-pay hypothesis" — who builds this hypothesis? How? When?

**PROPOSAL:** Add pricing decision trigger: "Pricing model defined at extraction event (D6 criteria met). Free tier required for initial validation. Commercial track considered only after ≥10 paying customers in any category."

---

### D12: Cap of 3 active IIPs; explicit founder/owner approval required

**VERDICT: OK**

**EVIDENCE:**
- The cap is sensible governance to prevent fragmentation.
- "Explicit founder/owner approval" — good principle, but **who are these founders/owners**? The proposal doesn't specify selection criteria or capacity implications.
- "Existing IIP slot exits when a tool is extracted, archived, or downgraded" — what are the **downgrade criteria**? No mechanism specified.
- The cap prevents proliferation but doesn't address the core problem: we don't have validated IIPs to proliferate.
- Precedent: Open-source foundations use "incubation" stages precisely to manage this (Apache TLP, CNCF, etc.). The proposal reinvents governance without adopting proven frameworks.

**PROPOSAL:** Add downgrade criteria: "IIP downgraded to Lab if: (1) no external consumers for 60 days, (2) no active maintainer for 30 days, (3) critical security unpatched for 90 days."

---

## MINORITY REPORT

**I predict disagreement with majority on D8 (VETO).**

The majority may accept "flagship IIP candidates today" as aspirational language. I disagree. In a strategy document, "today" means current state. Current state: these tools don't exist. Using "flagship" language for non-existent products is **strategic misrepresentation** that will undermine credibility when the council presents this to stakeholders.

Rationale: The proposal's strongest argument is "we might be wrong, we're willing to change." But claiming non-existent tools are "flagship" contradicts that humility. It signals overconfidence precisely where the evidence is weakest.

---

## DOMAIN VETOES

**[DOMAIN VETO: D8 — Factual inaccuracy]**
Claiming non-existent products are "flagship candidates today" is factually incorrect. This undermines the entire proposal's credibility. A strategy document cannot assert as current reality what is clearly future aspiration.

**[DOMAIN VETO: D3 — Circular reasoning]**
The `independent_value: yes` annotation proves nothing. You cannot establish independence by labeling it. This is governance theater that substitutes formatting for evidence.

---

## THREE BIGGEST RISKS THE PROPOSAL FAILS TO ADDRESS

**1. Premature infrastructure commitment for unvalidated products**
The proposal builds release pipelines (Homebrew, separate go.mod, separate semver, separate README sites) for products that don't exist. This creates operational debt before demand is validated. The "build it and they will come" assumption is the most common failure mode in developer tooling.

**2. Governance gap between F150 and IIP delivery ownership**
F150 doesn't ship IIPs. IIPs get separate epics. But who owns delivery? Who allocates capacity? Who is accountable for IIP milestones? The proposal creates a structural gap where no team has end-to-end ownership. This is how projects go to die.

**3. Brand dilution via maintainer misaligned incentives**
SDP team maintains tools they can't claim as their own. Without brand ownership, maintainer engagement degrades over time. The proposal acknowledges spin-out as a possibility but provides no trigger, no criteria, and no transition plan. This leaves IIPs in limbo — too independent to get SDP support, too SDP-maintained to have true independence.

---

## PRECEDENT REFERENCES

**1. etcd (CoreOS → CNCF)**
- Started as internal infrastructure at CoreOS, proven in production at scale before spin-out.
- CoreOS had commercial incentive to maintain etcd as a product. The spin-out to CNCF happened after etcd was battle-tested and had clear governance.
- **Lesson**: Spin-out when product is proven, not when it's a hypothesis. etcd had millions of downloads before CNCF incubation.

**2. containerd (Docker → CNCF)**
- Docker donated containerd after it was the runtime for thousands of production deployments.
- Docker maintained containerd as a core component *before* donation, ensuring quality during incubation.
- **Lesson**: The maintainer (Docker) had skin in the game. SDP team maintaining IIPs without brand ownership lacks equivalent incentive.

**3. Ginkgo (Cloud Foundry → Independent)**
- Started as internal testing framework at Cloud Foundry, extracted when it had internal users and clear value.
- The extraction was driven by a specific maintainer (Onsi Fakhouri) who took ownership, not by a governance decision.
- **Lesson**: Successful spin-outs require a **champion**, not just a governance structure. The proposal doesn't address champion identification.

**4. Tailwind CSS (Laravel ecosystem → Independent)**
- Started as a plugin for a Laravel conference talk, validated via Twitter demos before becoming a product.
- The creator (Adam Wathan) maintained it as his primary project, not as a side effort under a parent brand.
- **Lesson**: Independent tools need dedicated maintainers, not shared teams with divided attention.

---

## SUMMARY VERDICT TABLE

| Claim | Verdict | Critical Issue |
|-------|---------|----------------|
| D1 | WEAK | "Inherent value" asserted, not demonstrated |
| D2 | WEAK | Premature taxonomy layer for hypothetical products |
| D3 | OPPOSE | Circular reasoning via annotation |
| D4 | OK (premature) | Infrastructure not yet built |
| D5 | WEAK | Homebrew for non-existent products |
| D6 | WEAK | Criteria unenforceable without measurement infrastructure |
| D7 | OK | Contingent on D1-D6 validation |
| D8 | VETO | Factual error: tools don't exist |
| D9 | WEAK | Governance gap in delivery ownership |
| D10 | WEAK | Maintainer incentive misalignment |
| D11 | OK | Directionally correct, operationally empty |
| D12 | OK | Good governance, missing downgrade criteria |

**Bottom line**: Option D's *direction* is sound — tools with independent value should be treated independently. But the proposal conflates aspiration with evidence, builds infrastructure for non-existent products, and creates governance gaps that will undermine delivery. The core claim (D8) is factually incorrect. Fix D8 first, then revisit the structural proposals.