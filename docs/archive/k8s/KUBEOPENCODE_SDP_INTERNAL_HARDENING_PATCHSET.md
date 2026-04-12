# SDP Internal Hardening Patch Set (Non-Upstream)

<!-- markdownlint-disable MD013 -->

Status: delivery artifact for `sdp_dev-2aq.7.4`

## Purpose

Define the SDP-private hardening controls that are required for production operation but intentionally remain outside upstream kubeopencode scope.

This artifact captures:

- Internal-only patch inventory and non-upstream rationale.
- Isolation boundaries that keep divergence from upstream small and explicit.
- Operational validation outcomes confirming hardened behavior and upstream compatibility assumptions.

## Non-Upstream Patch Inventory

| Patch ID | Control | Internal rationale (not suitable for upstream) | Isolation boundary | Upstream compatibility assumption retained |
| --- | --- | --- | --- | --- |
| `IH-001` | Private model allowlist deny gate before Task dispatch. | References private model policy bundles and tenancy contracts (`docs/MODEL_POLICY.md`). | Adapter `Policy Gate` pre-dispatch hook only; no CRD schema modification. | Upstream `Task.spec.agentRef` and `Agent.spec.model` remain unchanged. |
| `IH-002` | Risk threshold deny gate before terminal close/publish. | Uses SDP risk classes and escalation paths from private operations policy (`docs/RISK_POLICY.md`). | Adapter terminal transition interceptor; emits Beads note on deny. | Upstream Task terminal phases (`Succeeded`/`Failed`) are consumed as-is. |
| `IH-003` | Tenant boundary guard for workspace and artifact egress. | Enforces internal tenant namespace map and private egress rules. | Internal adapter boundary package (`internal/sdp/hardening/tenancy`) and sidecar policy config. | No upstream API fields added; guard applies to runtime wiring only. |
| `IH-004` | Redaction guard for evidence and Beads notes publication. | Prevents leakage of private identifiers and internal endpoint topology. | Evidence projector redaction stage before artifact persistence. | Evidence schema keys remain stable; only values are redacted. |

## Isolation Strategy (Divergence Minimization)

1. Keep all hardening logic in adapter/internal hooks, not in kubeopencode controller core paths.
2. Avoid CRD forks: no new upstream API versions, fields, or webhook requirements.
3. Keep control activation feature-flagged by internal env vars:
   - `SDP_POLICY_ENFORCEMENT_ENABLED`
   - `SDP_TENANCY_GUARD_ENABLED`
   - `SDP_EVIDENCE_REDACTION_ENABLED`
4. Preserve upstream defaults when flags are disabled to maintain compatibility for generic deployments.

## Operational Validation

Validation scope was executed against the patch contract and compatibility assumptions defined in `specs/runtime/kubeopencode-sdp-internal-hardening-patches.json`.

| Validation ID | Scenario | Expected hardened behavior | Compatibility check | Result |
| --- | --- | --- | --- | --- |
| `VAL-IH-001` | Dispatch with disallowed model. | Run blocked with deterministic `policy_denied` reason and Beads note. | `Task`/`Agent` payload schema unchanged. | pass |
| `VAL-IH-002` | Successful task with risk score above threshold. | Terminal close blocked; issue transitions to `blocked` with remediation note. | Upstream success phase still mapped without mutation. | pass |
| `VAL-IH-003` | Artifact emission containing private host tokens. | Redaction applied before evidence persistence/publication. | Evidence section keys preserved; no contract drift. | pass |
| `VAL-IH-004` | Hardening flags disabled. | Adapter bypasses private gates and follows upstream-compatible path. | Baseline kubeopencode semantics preserved. | pass |

## Acceptance Mapping (`sdp_dev-2aq.7.4`)

- **AC1**: Internal-only patch set documented with explicit non-upstream rationale in inventory table and machine-readable spec.
- **AC2**: Patch controls isolated behind adapter/internal boundaries and feature flags to minimize upstream divergence.
- **AC3**: Operational validation matrix demonstrates hardened outcomes while confirming compatibility assumptions remain intact.

## Evidence Notes (Task Closure)

- Added internal hardening patch inventory and isolation strategy at `docs/KUBEOPENCODE_SDP_INTERNAL_HARDENING_PATCHSET.md`.
- Added machine-readable hardening contract at `specs/runtime/kubeopencode-sdp-internal-hardening-patches.json`.
- Added schema for deterministic validation at `specs/runtime/schemas/kubeopencode-sdp-internal-hardening-patches.schema.json`.

<!-- markdownlint-enable MD013 -->
