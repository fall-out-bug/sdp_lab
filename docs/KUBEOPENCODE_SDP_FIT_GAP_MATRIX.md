# KubeOpenCode Fit-Gap Matrix Against SDP Requirements

<!-- markdownlint-disable MD013 -->

Status: baseline intake for `sdp_dev-2aq.7.3`

## Scope

This matrix compares SDP runtime requirements to the currently validated kubeopencode baseline (Task/Agent CRDs plus probe orchestration in `docs/KUBEOPENCODE_MULTI_ROLE_PROBE_RUNBOOK.md`).

Severity scale:

- `critical`: missing control can violate task-state correctness, policy, or production safety.
- `high`: missing control can break deterministic automation or auditability.
- `medium`: missing control reduces reliability/operability but has bounded blast radius.
- `low`: optimization or maintainability gap with low immediate risk.

Disposition mapping:

- `adapter extension`: implemented in SDP adapter/controller layer on top of kubeopencode.
- `upstream PR candidate`: generic capability suitable for upstream kubeopencode contribution.
- `internal patch`: SDP-private control not suitable for upstream.

## Fit-Gap Matrix

| Requirement Area | SDP Requirement | KubeOpenCode Baseline Fit | Gap | Severity | Disposition | Notes / Follow-on Task Input |
| --- | --- | --- | --- | --- | --- | --- |
| Beads workflow | Beads remains source of truth for claim/close/escalate lifecycle. | No native Beads integration in Task/Agent reconciliation path. | Missing deterministic mapping between Beads issue state and CRD events. | high | adapter extension | Primary design input for `sdp_dev-2aq.7.1`. |
| FSM transitions | Enforce canonical progression (`open -> in_progress -> review -> verified -> done`) and side states (`blocked`, `escalated`, `cancelled`). | Task phases (`Completed/Failed`) are available but not mapped to SDP FSM contract. | No transition guardrail layer aligned with SDP state-machine semantics. | critical | adapter extension | Reuse transition policy patterns from `internal/artifact/transition_controller.go` in adapter design (`sdp_dev-2aq.7.1`). |
| Evidence capture | Strict evidence envelope with required sections and provenance keys must be emitted per run. | Probe captures task logs and role artifacts; strict SDP evidence contract is not native. | Missing contract-complete evidence packing (`intent`, `plan`, `execution`, `verification`, `review`, `risk_notes`, `boundary`, `provenance`, `trace`). | critical | adapter extension | Adapter must project role outputs into `.sdp/evidence/<issue>.json` and provenance schema (`sdp_dev-2aq.7.1`). |
| Policy enforcement | Runtime model allowlist and go/no-go gates must be policy-enforced before publish. | Model routing can be configured per Agent; no built-in SDP policy contract enforcement. | Missing deterministic deny reasons and policy-gate audit artifacts. | high | internal patch | Keep SDP policy controls private (`docs/MODEL_POLICY.md`, `docs/RISK_POLICY.md`); implement as internal guard rails in `sdp_dev-2aq.7.4`. |
| Operational controls | Idempotent dispatch and duplicate-run prevention for same issue/run-id. | Baseline operator can execute tasks; duplicate suppression is not guaranteed by SDP semantics. | No lock-domain or explicit idempotency guard keyed by Beads issue/run context. | high | adapter extension | Implement run lock domain in adapter control plane (`sdp_dev-2aq.7.1`) and align with prior orchestrator safeguards. |
| Retry and escalation | Bounded retries with escalation policy and terminal reason recording. | Task failures are observable, but retry budget/escalation policy are external today. | Missing native retry budget + escalation contract aligned to SDP. | medium | upstream PR candidate | Candidate upstream enhancement if kept generic (retry budget and terminal reason fields) for `sdp_dev-2aq.7.2`. |
| Observability and traceability | Every run must link run context, evidence context, and PR linkage for audit. | Probe collects logs and phases; does not natively emit SDP trace keys. | Missing first-class trace fields (`trace.run_context_link`, `trace.evidence_context_link`, `trace.pr_url`). | high | adapter extension | Adapter should persist these fields into Beads notes/evidence (`sdp_dev-2aq.7.1`). |
| Multi-role dependency semantics | Reviewer must run only after analyst/coder artifacts pass verification. | Prototype orchestrator sequences roles correctly, but rule is not enforced as reusable operator contract. | Missing generic DAG/dependency gating semantics in operator APIs. | medium | upstream PR candidate | Upstreamable as generic task dependency contract in `sdp_dev-2aq.7.2`; keep SDP-specific strict evidence checks internal. |
| Security and tenancy boundaries | SDP-private policy bundles and boundary constraints must stay internal. | Upstream operator is generic and should not embed private policy/tenant assumptions. | Need explicit boundary split to avoid leaking private controls upstream. | high | internal patch | Define non-upstream patch boundary and rationale in `sdp_dev-2aq.7.4`. |

## Sequencing Recommendations

1. `sdp_dev-2aq.7.1` first (P1): implement adapter contracts for Beads/FSM/evidence/trace and idempotency controls.
2. `sdp_dev-2aq.7.2` second (P2): propose upstream-safe enhancements extracted from adapter findings (retry budget primitives, dependency semantics, generic status fields).
3. `sdp_dev-2aq.7.4` third (P2): apply SDP-private hardening after adapter contract is stable and upstream scope boundary is explicit.

Execution rationale:

- Adapter-first reduces ambiguity by proving exact integration seams before upstream split.
- Upstream planning should consume concrete generic deltas observed during adapter implementation.
- Internal patches should be minimized and isolated after upstream candidacy is evaluated.

## Evidence Notes (Task Closure)

- Produced fit-gap matrix with per-requirement severity and disposition mapping.
- Captured follow-on sequencing inputs for `sdp_dev-2aq.7.1`, `sdp_dev-2aq.7.2`, and `sdp_dev-2aq.7.4`.
- Added machine-readable companion artifact at `specs/runtime/kubeopencode-sdp-fit-gap.json`.

<!-- markdownlint-enable MD013 -->
