# Market Intelligence Operating Loop

Status: active
Updated: 2026-03-03
Purpose: make market analysis a recurring input into weekly feature prioritization

## 1. Objectives

- Detect shifts in AI agent tooling, governance, provenance, and CI trust solutions
- Convert external signals into roadmap decisions within 1 week
- Avoid building commodity layers where ecosystem tools already lead

## 2. Scope

Market scan coverage:

- Agent orchestration platforms
- Policy/gate engines
- Provenance/attestation ecosystems
- Runtime guardrails and agent governance frameworks
- OSS adoption and launch patterns for developer tools

## 3. Cadence

Weekly (Friday):

- Update one market pulse report (top changes, competitor moves, new standards)
- Classify impact: ignore / monitor / adapt / accelerate

Monthly:

- Deep landscape refresh
- Update differentiation map and deprecate stale assumptions

Quarterly:

- Strategic thesis review and roadmap rebalancing

## 4. Output artifacts

Weekly outputs:

- `docs/market/WEEKLY_PULSE_YYYY-MM-DD.md`
- `docs/market/DECISION_LOG.md` update (what changed in roadmap and why)

Monthly outputs:

- `docs/market/LANDSCAPE_QUADRANT_YYYY-MM.md`

Quarterly outputs:

- `docs/market/THESIS_REVIEW_YYYY-QN.md`

## 5. Decision framework

For each external signal:

1. Relevance to SDP core thesis (trust/provenance/evidence)
2. Threat vs complement vs distribution channel
3. Build vs integrate vs ignore decision
4. Expected impact in 4-12 weeks

## 6. Required KPIs

- Number of roadmap decisions backed by market evidence
- Time from external signal to documented decision
- % of delivered features tied to validated market need
- % of roadmap items tagged commodity vs differentiation

## 7. Minimum workflow

1. Collect signals (releases, blog posts, standards updates, major OSS movements)
2. Score impact with a fixed rubric
3. Propose one roadmap adjustment per week
4. Record accepted/rejected decisions with rationale

## 8. Integration with weekly release model

Every weekly release planning session must include:

- top 3 market signals from the current week
- explicit answer: "does this week feature strengthen differentiation or duplicate commodity?"
- one sentence connecting the feature to market context

Baseline reference set:

- `docs/research/MARKET_LANDSCAPE_AI_TRUST_TOOLS_2026-03-03.md`
- `docs/plans/2026-03-03-oss-growth-playbook-12weeks.md`
