# Socratic Review: SDP Harness And Skill Operating Strategy

Status: review evidence
Date: 2026-05-15
Target: `docs/research/2026-05-15-sdp-harness-skill-synthesis.md`

## Reviewer Runs

| Lane | Model | Status | Notes |
|---|---|---|---|
| initial qwen | `openrouter/qwen/qwen3.6-plus` | `off_task` | Returned a fake tool-call preamble instead of reviewing the document. Not counted as evidence. |
| initial glm | `zai/glm-5.1` | `off_task` | Critiqued claims not present in the synthesis. Not counted as evidence. |
| socratic critic | `zai/glm-5.1` with attached file | `assessed` | Produced blocking and major questions tied to the synthesis. |
| adoption-risk reviewer | `kimi-coding/kimi-for-coding` with attached file | `assessed` | Produced feasibility and process-risk questions tied to the synthesis. |

## Blocking Questions Accepted

### B1. Cross-review provenance may self-contaminate the synthesis

The synthesis cited model-generated cross-reviews while also saying vendor/model
claims are quarantined. The document now states that cross-reviews are review
evidence curated by the author, not validation authority.

Disposition: accepted and fixed.

### B2. Phase 1 had no pass/fail gate

The one-week foundation phase described deliverables but not completion
criteria. The document now has an explicit Phase 1 acceptance gate.

Disposition: accepted and fixed.

### B3. Phase 1 declares runtime fields before runtime enforcement exists

The reviewers flagged that Phase 1 can only create authoring discipline; runtime
claims remain unproven. The document now requires `not_assessed_runtime` when
dispatch evidence does not exist.

Disposition: accepted and fixed.

## Major Questions Accepted

### M1. Model-family diversity needs implementer provenance

The review asked how "different model family" can be verified. The synthesis now
states that unknown implementer provenance makes diversity `not_assessed_runtime`.

Disposition: accepted and fixed.

### M2. `compatibility` was ambiguous

The review asked whether `compatibility` is a verified portability claim or an
intent. The synthesis now says it is only a claim after runtime dispatch
evidence exists.

Disposition: accepted and fixed.

### M3. Doubt cycles needed a per-cycle bound

The synthesis now says a doubt cycle must stay inside a bounded review window,
with short findings; broad redesign output is off-scope.

Disposition: accepted and fixed.

### M4. Routing promotion criteria were undefined

The synthesis now defines `vendor_only`, `local_spike`, and
`validated_on_sdp_tasks` promotion criteria.

Disposition: accepted and fixed.

### M5. Irreversible action gates may not exist in every harness

The synthesis now says a harness that cannot enforce `external_write` or
`irreversible` runtime gates cannot claim runtime support for that action class;
it must use workflow-level authorization and degraded evidence.

Disposition: accepted and fixed.

## Questions Deferred

### Trigger thresholds

The Socratic review asked for quantified trigger thresholds to replace "invoke
at 1% chance." This is deferred. The current recommendation is trigger-rich
prose plus lint, not numerical confidence scoring. Numerical thresholds would
likely create false precision until SDP has routing telemetry.

Disposition: deferred.

### Phase 4 provider procurement and rate limits

The adoption-risk reviewer noted that measuring all listed providers may require
API contracts and budget. This is valid but belongs in a future measurement
workstream, not this synthesis.

Disposition: deferred.

## Verdict

The review found real adoption blockers, and the synthesis was updated for the
ones that affect immediate decision quality. Remaining deferred items should be
handled during implementation planning, not by expanding this research document.
