# Critical Roadmap Review (2026-03-03)

Status: active
Scope: critical assessment of current roadmap versus repo reality and earlier strategic viewpoints

## 1. Headline assessment

Roadmap direction is strategically strong but execution hygiene is weak. The main risk is not wrong vision, but governance drift between planning artifacts and live backlog.

## 2. What is working

- Strong thesis consistency around trust/provenance/evidence
- Correct standards trajectory (in-toto, OPA, Sigstore)
- Healthy ecosystem integration intent (OpenCode, Beads, Gas Town)
- Clear ambition for phased evolution from CLI trust layer to K8s runtime

## 3. Critical issues

### C1. Planning source drift

- ROADMAP, INDEX, backlog frontmatter, and Beads disagree on status for multiple features/workstreams.
- This creates false confidence and undermines prioritization.

### C2. Legacy queue pressure vs strategic queue

- Ready queue includes many legacy-migration tasks (F002/F004/F005/etc.) while strategic narrative pushes dual-surface productization.
- Without explicit lane policy, weekly releases can be consumed by historical debt instead of strategic differentiation.

### C3. Repo-split ambiguity

- Different docs prescribe different split targets and naming structures.
- Missing trigger-based governance can cause premature repo fragmentation.

### C4. Launch narrative fragmentation

- Promotion and positioning are spread across many docs with overlaps and occasional contradictions.
- No single authoritative outward story for external adopters.

### C5. Market-analysis loop under-operationalized

- Market awareness exists in research docs, but there is no strict recurring process that forces roadmap adaptation cadence.

## 4. Missed insights from earlier viewpoints

- Keep split minimal until external pull is demonstrated
- Enforce two-mode product lens (Light vs Full) in planning decisions
- Keep evidence/trust layer as primary differentiation; avoid commodity orchestration drift

## 5. Recommended correction priorities

Priority 1 (this week):

- Fix roadmap/index/backlog/beads status mismatches
- Lock one canonical repo-boundary policy with trigger-based milestones

Priority 2 (next 2 weeks):

- Protect one-feature-per-week cadence with strict definition of done
- Allocate weekly capacity explicitly: strategic feature lane + risk/debt lane

Priority 3 (this month):

- Collapse outward narrative into a single promotion vision and launch sequence
- Run weekly market loop and tie each release to an external signal or explicit thesis defense

## 6. Decision standard for weekly planning

Each weekly feature candidate must answer:

1. Does it strengthen the trust/provenance/evidence moat?
2. Is it externally demonstrable in one release cycle?
3. Does it avoid commodity duplication unless needed for integration?
4. Is source-of-truth status clean across roadmap/index/backlog/beads?

If any answer is no, defer or re-scope.
