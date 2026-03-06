# Ideas F086-F089: Long-Horizon Parking Lot

This draft captures Direction 3 ideas parked for later cycles.

## F086: Cross-Project Evidence Federation

Hypothesis: one evidence layer can serve multiple repositories while preserving per-repo trust boundaries and ownership.

Potential outcomes:

- reduce duplicated policy plumbing across repositories
- enable cross-repo incident traceability

## F087: Adversarial Reviewer Quorum

Hypothesis: requiring dissent-aware multi-reviewer verdicts reduces false positives in AI-generated code approvals.

Potential outcomes:

- better defect detection in high-risk changes
- explicit disagreement tracking for audit

## F088: Autonomous Backlog Synthesis

Hypothesis: recurring findings patterns can be converted into candidate backlog tasks with high precision.

Potential outcomes:

- faster remediation loop from CI signal to task creation
- lower manual triage overhead

## F089: Adaptive Gate Tuning

Hypothesis: gate thresholds tuned by historical signal quality can reduce false positives without weakening controls.

Potential outcomes:

- lower review friction on healthy modules
- stricter checks where drift risk is higher

## Parking Criteria

These ideas remain parked until:

- Direction 1 workstreams are materially complete
- owner and scope are assigned
- measurable pilot success criteria are defined
