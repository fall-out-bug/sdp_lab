# Discovery Validation

## 🔄 Final Verdict: PIVOT

The evidence is mixed: each validated claim remains INSUFFICIENT_DATA rather than clearly supported or contradicted, so the core product thesis is not strong enough for a GO, but not weak enough for a KILL. There is a plausible market direction around standardized consulting workflows and configurable agent operations, yet the data does not show enough confidence that non-technical protocol definition, standardized project archetypes, or workflow flexibility are broadly compelling enough to justify full commitment.

**Pivot suggestion:** Narrow the product to a specific consulting subsegment and a small set of repeatable engagement types, then position the product as a configurable orchestration layer with templated agent teams and high-level protocol presets rather than a general multi-agent communication platform.

> ⚠️ **Needs experiment:** one or more claims have insufficient desk-research data — Phase 4b recommended.

## Claim Validation

### Rank 1 — insufficient_data (confidence 62%)

**Claim:** Product managers can define communication protocols without deep technical expertise

**Notes:** High-level protocol definition by PMs is plausible when using abstractions and standards, but research and industry guidance also indicate that robust agent communication design usually needs technical expertise, so the evidence is mixed.

| Direction | Evidence | Estimate? |
|-----------|----------|-----------|
| FOR | [The MACH Alliance’s composable-commerce guidance shows that business/product stakeholders can define interaction and API contracts at the business level, while implementation details are handled by architects and engineers, implying PMs can meaningfully specify communication protocols without deep technical expertise.](https://machalliance.org/) | no |
| FOR | [The OpenAPI Specification was explicitly designed to be human- and machine-readable for describing REST APIs, and teams commonly use it as a shared contract across product, design, and engineering; this lowers the technical bar for non-engineers to define message formats and endpoints.](https://www.openapis.org/) | no |
| FOR | In product discovery practice, user stories and acceptance criteria are often written in business language and then translated into technical implementations, which supports the idea that PMs can define what communication should achieve even if they do not specify lower-level protocol mechanics. | yes |
| FOR | Modern agent frameworks increasingly expose high-level abstractions such as tool schemas, routing policies, and workflow graphs, so PMs can often choose among prebuilt patterns (for example, request/response vs. event-driven handoffs) without needing to understand underlying networking or serialization internals. | yes |
| AGAINST | [A survey of 2,000 executives in IBM’s 2023 Global AI Adoption Index found that lack of AI talent and skills was one of the top barriers to AI adoption, suggesting that non-technical PMs may struggle to define effective agent protocols without specialist support.](https://www.ibm.com/reports/ai-adoption) | no |
| AGAINST | [MIT CISR research on enterprise digital transformation has long shown that strong business-technology collaboration, not business-only specification, is associated with better outcomes; this implies protocol design quality likely depends on technical expertise as well as product judgment.](https://cisr.mit.edu/) | no |
| AGAINST | [NIST AI Risk Management Framework emphasizes mapping and managing interactions, dependencies, and context-specific risks in AI systems; defining communication protocols for agents therefore requires technical risk understanding beyond typical PM training.](https://www.nist.gov/itl/ai-risk-management-framework) | no |
| AGAINST | In multi-agent systems research, communication protocols materially affect coordination quality, emergent behavior, and failure modes; because these effects are often non-obvious, non-technical definition alone is likely insufficient for robust protocol design. | yes |

### Rank 2 — insufficient_data (confidence 66%)

**Claim:** Consulting firms have standardized project types that could use predefined agent team configurations

**Notes:** Consulting clearly has recurring delivery patterns and roles, but a substantial share of work is bespoke and dynamic, so the evidence supports partial standardization rather than a strong universal standardization claim.

| Direction | Evidence | Estimate? |
|-----------|----------|-----------|
| FOR | [The Project Management Institute’s 2021 Pulse of the Profession reported that organizations manage work through repeatable project-management processes and “project types” that can be standardized; PMOs are explicitly used to create common methods, templates, and governance across recurring engagements, which fits the idea that consulting delivery can be packaged into predefined team configurations.](https://www.pmi.org/learning/thought-leadership/pulse/pulse-of-the-profession-2021) | no |
| FOR | [McKinsey’s work on enterprise operating models describes professional-services firms as organizing delivery around repeatable offerings and reusable assets (for example, diagnostics, workshops, implementation playbooks), implying that many consulting engagements follow stable patterns that can be mapped to standard roles and workflows.](https://www.mckinsey.com/capabilities/people-and-organizational-performance/our-insights) | no |
| FOR | In IT and management consulting, firms commonly use staffing archetypes such as partner/engagement lead, manager, SME, analyst, and industry specialist; this role stack appears repeatedly across proposal templates and delivery methodologies, indicating a de facto standardized team topology for many project classes. | yes |
| FOR | Large consultancies invest heavily in knowledge-management and “accelerators” (reusable tools, decks, benchmarks, and code), suggesting many engagements are sufficiently similar that a predefined agent team could mirror those repeatable work packages rather than being designed from scratch each time. | yes |
| AGAINST | [A Harvard Business Review article on consulting emphasizes that client problems differ materially by industry, function, geography, and maturity, and that successful consultants adapt the engagement model rather than apply a fixed formula; this implies standardized project types are only part of the market, not the whole market.](https://hbr.org/2004/09/the-consulting-industry) | no |
| AGAINST | Firm-level studies of consulting show strong variation in project scope and duration—from short diagnostic studies to long transformation programs—making it hard to cover the space with a small number of predefined team configurations. | yes |
| AGAINST | Many consulting engagements are bespoke because client constraints, politics, data quality, and stakeholder alignment materially change the work plan; that variability reduces the share of projects that are truly standardized enough to benefit from a fixed agent team design. | yes |
| AGAINST | The same consulting firms often reconfigure teams dynamically as projects evolve (e.g., adding legal, data, cybersecurity, or industry experts late in the engagement), which means an initially standardized team configuration can become obsolete quickly. | yes |

### Rank 3 — insufficient_data (confidence 64%)

**Claim:** System architects value flexible operational modes enough to change their current workflow

**Notes:** There is credible evidence that engineers embrace workflow changes when benefits are clear, but enterprise inertia, pilot-stage adoption, and contextual variability make it unclear that system architects value flexible operational modes enough to change current practice.

| Direction | Evidence | Estimate? |
|-----------|----------|-----------|
| FOR | [In a large Stack Overflow Developer Survey, 70%+ of respondents said they use AI tools in their development workflow, and a sizable share reported using them for multiple tasks; this suggests technical practitioners are willing to adjust established workflows when a tool clearly improves task performance.](https://survey.stackoverflow.co/2024/ai/) | no |
| FOR | [Google’s 2024 DORA report found that generative AI adoption in software delivery was broad and associated with time savings for many developers, indicating that engineering organizations are already experimenting with workflow changes to gain efficiency.](https://dora.dev/research/2024/dora-report/) | no |
| FOR | [The 2024 State of DevOps / DORA research reports that high-performing teams emphasize tooling and automation as ways to reduce friction and improve flow, which is consistent with architects valuing operational flexibility when it improves system throughput.](https://dora.dev/research/2024/dora-report/) | no |
| FOR | Kubernetes and other orchestration platforms have become standard in many enterprises precisely because architects want flexible deployment and runtime modes; this historical adoption pattern suggests system architects will consider switching workflows for operational control and adaptability. | yes |
| AGAINST | [The DORA research also found that the benefits of AI tools vary widely by context and that organizational constraints can limit realized gains, implying that many architects may not see enough benefit to change their current workflow.](https://dora.dev/research/2024/dora-report/) | no |
| AGAINST | [McKinsey’s 2024 State of AI reports that most organizations are still in experimentation or pilot stages for gen AI rather than fully redesigning core workflows, which suggests reluctance to change established operating models.](https://www.mckinsey.com/capabilities/quantumblack/our-insights/the-state-of-ai) | no |
| AGAINST | System architects are typically risk-sensitive because workflow changes can affect reliability, security, and compliance; in practice this often creates inertia unless the new mode delivers a clear, immediate advantage over current processes. | yes |
| AGAINST | Adoption studies of enterprise software repeatedly show that switching costs, retraining burden, and integration work are major barriers to process change, so flexibility alone may not be enough to induce architects to alter their current workflow. | yes |

---

*Cost: $0.01141*
