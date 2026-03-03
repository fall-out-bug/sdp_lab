# Unified Vision and Roadmap (2026-03-03)

Status: active
Owner: SDP core
Scope: unify roadmap, workstream index, Beads reality, and launch direction into one execution model

## 1. Why this document exists

SDP has strong ideas and many documents, but execution signals are split across ROADMAP, INDEX, backlog frontmatter, and Beads. This file defines one operating model so weekly delivery and strategic direction stay aligned.

## 2. Current reality snapshot

Data source snapshot (2026-03-03):

- Open Beads issues: 64
- Priority distribution: P1=19, P2=37, P3=8
- Open issue buckets: code-review=17, legacy-migration=21, protocol=17, ecosystem=4, other=5
- Ready queue (`bd ready`): 10 issues (mostly F004/F005/F002 legacy + F074/F075/F076/F077/F063)
- Backlog workstreams: 107 files, mapping count matches 107

Consistency findings:

- ROADMAP references F053/F054 workstreams that do not exist in INDEX or backlog
- 20+ backlog frontmatter statuses conflict with INDEX
- Several features show mixed done/backlog state across sources

Reference mismatch report: `docs/ROADMAP_INDEX_BEADS_MISMATCH_REPORT.md`

## 3. Canonical source-of-truth rules

Use these rules to avoid drift:

1. `docs/roadmap/ROADMAP.md` = feature and phase authority
2. `docs/workstreams/INDEX.md` = workstream status authority
3. `docs/workstreams/backlog/*.md` frontmatter status must match INDEX
4. Beads status must map from INDEX (not from ad-hoc edits)
5. A workstream cannot be marked done if acceptance criteria are unchecked
6. ROADMAP cannot mention workstreams that do not exist in INDEX/backlog

## 4. Critical assessment of current roadmap

### Strengths

- Correct strategic core: trust/provenance/evidence and standards-based enforcement
- Strong technical base: outer-loop architecture, CI gates, evidence tooling
- Good ecosystem awareness: OpenCode, Beads, Gas Town, K8s runtime path

### Weak points

- Operational drift between roadmap/index/backlog/beads reduces planning trust
- Legacy migration work (F002-F013/F004/F005) competes with current product narrative
- Launch narrative is fragmented across many docs; no single external storyline
- Repo split strategy is inconsistent across documents (2-repo vs 4/7-repo futures)
- No explicit weekly feature-release governance tied to a strict definition of done

### What was missed from early viewpoints

- Early recommendation to keep repo count minimal until external pull (2-repo bias) was partially lost
- Two-mode product model (Light mode for humans, Full mode for autonomy) is not enforced as a planning lens
- Market-loop discipline (continuous competitor tracking + strategic adaptation) is under-specified in execution artifacts

## 5. Weekly release operating model

Target: one complete, user-meaningful feature per week.

Definition of a weekly feature release:

- Has a clear user-facing capability (not internal refactor only)
- Has acceptance criteria and evidence of completion
- Has docs update (what changed, how to use)
- Passes build/test/gates in CI
- Has migration notes if behavior changed

Weekly cadence:

- Monday: lock scope and success metrics
- Tuesday-Thursday: implementation + validation
- Friday: release artifact + docs + promotion output

## 6. 12-week execution lanes

This plan balances product momentum with debt/risk retirement.

Lane A (Core trust product):

- F068 Unified Integration Contracts
- F070 OSS Observability and Explainability
- F077 CI to Local Bridge

Lane B (Governance and reliability):

- Resolve open P1/P2 security/concurrency/error-handling issues from code review
- Complete branch protection and protocol-compliance enforcement

Lane C (Ecosystem and adoption):

- F069 OSS Combine Bootstrap
- F063 memory integration completion
- F060 Gas Town bridge (staged)

## 7. Repo split milestones

Do not split by aspiration. Split only when triggers fire.

Milestone M1: protocol boundary hardening (now)

- Keep protocol artifacts cleanly separable
- Remove contradiction between boundary docs

Milestone M2: first external adoption trigger

- Trigger: repeated external demand for standalone evidence tooling
- Action: isolate and release stable OSS surface for evidence validation

Milestone M3: orchestrator/runtime extraction trigger

- Trigger: sustained usage requiring independent release cadence and ownership
- Action: split runtime/orchestrator repo with strict contract compatibility

Milestone M4: enterprise boundary formalization

- Trigger: concrete enterprise requirements (compliance/RBAC/SIEM) with maintenance capacity
- Action: separate enterprise pack with strict provenance/signature defaults

## 8. Immediate alignment actions (next 7 days)

1. Fix roadmap/index/backlog mismatches (including F053/F054 references)
2. Decide and document one canonical repo-boundary strategy
3. Freeze next 3 weekly feature releases with explicit DoD and KPIs
4. Publish single external narrative from canonical docs only

## 9. KPI set for roadmap governance

- Weekly complete feature releases (target: 1/week)
- Roadmap consistency score (roadmap/index/backlog/beads alignment)
- Gate pass rate (protocol-compliance + evidence + policy)
- Open P1 issue burn-down rate
- Time-to-adoption metrics for released OSS capabilities

## 10. Weekly feature release train (initial)

Target: one finished feature every week.

Initial 8-week sequence (adjust weekly via market loop):

1. Week 1: F068 contract unification slice (L1 protocol contracts)
2. Week 2: F059/F064 governance enforcement slice (contract gate + CI policy hardening)
3. Week 3: F070 observability slice (evidence/trace explainability artifact)
4. Week 4: F077 CI-to-local bridge slice (findings to Beads closed loop)
5. Week 5: F069 OSS bootstrap slice (light-mode onboarding + examples)
6. Week 6: F063 memory integration slice (session continuity with evidence constraints)
7. Week 7: F060 bridge slice (Gas Town adapter checkpoint with scope policy)
8. Week 8: F074 enterprise trust slice (signed envelope enforcement profile)

Rule: if a week is consumed by urgent P1 fixes, release a standalone reliability/security feature and re-sequence the remaining train.

## 11. Market-backed prioritization inputs

Primary research artifacts:

- `docs/research/MARKET_LANDSCAPE_AI_TRUST_TOOLS_2026-03-03.md`
- `docs/plans/2026-03-03-oss-growth-playbook-12weeks.md`

These must be reviewed in weekly planning before freezing the next feature release.

Related:

- `docs/roadmap/CRITICAL_ROADMAP_REVIEW_2026-03-03.md`
- `docs/roadmap/MARKET_INTELLIGENCE_OPERATING_LOOP.md`
- `docs/vision/REPO_PROMOTION_VISION.md`
