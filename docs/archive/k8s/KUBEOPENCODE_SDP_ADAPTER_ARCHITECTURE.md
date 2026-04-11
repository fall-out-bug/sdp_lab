# KubeOpenCode SDP Adapter Architecture and CRD Mapping

<!-- markdownlint-disable MD013 -->

Status: design contract for `sdp_dev-2aq.7.1`

## Goals

Define a deterministic adapter layer on top of kubeopencode `Task` and `Agent` CRDs that preserves SDP semantics for:

- Beads lifecycle authority.
- FSM transition guardrails.
- Evidence envelope production.
- Policy enforcement and deny reasoning.

The adapter must isolate SDP-private behavior from generic operator capabilities so upstream divergence remains low.

## Adapter Surfaces

1. **Intent Translator**: Converts Beads issue/run intent into kubeopencode CRD payloads.
2. **Lifecycle Reconciler**: Maps CRD status/events to SDP FSM transitions and Beads updates.
3. **Evidence Projector**: Projects operator outputs into strict SDP evidence schema.
4. **Policy Gate**: Enforces model/risk/publish controls before terminal state publication.
5. **Run Lock Manager**: Provides idempotent issue-run locking and duplicate suppression.

## CRD to SDP Mapping

### Task CRD Mapping

| kubeopencode Task field/event | SDP artifact/flow | Adapter behavior | Determinism guarantee |
| --- | --- | --- | --- |
| `metadata.name` | `run_context.run_id` | Generate stable run id `<issue>-<attempt>` if absent. | Same Beads issue + attempt always resolves to same run id. |
| `metadata.labels[beads.issue]` | Beads issue correlation | Required label; reject create if missing. | No orphaned Task objects. |
| `spec.prompt` / `spec.objective` | `intent` evidence section | Normalize and hash into evidence envelope. | Intent hash is immutable for run lifetime. |
| `spec.agentRef` | role selection (`analyst`,`coder`,`reviewer`) | Resolve to SDP role binding matrix. | Role resolution is table-driven and versioned. |
| `status.phase=Pending` | FSM `in_progress` (entry) | Claim Beads issue if still `open`, then start execution window. | Single writer via lock manager. |
| `status.phase=Running` | FSM `in_progress` (active) | Emit heartbeat trace and execution evidence append. | Monotonic heartbeat sequence id. |
| `status.phase=Succeeded` | FSM `review` then `verified` candidate | Trigger verification policy checks before Beads close. | Terminal success requires explicit policy pass token. |
| `status.phase=Failed` | FSM `blocked`/`escalated` | Apply retry budget; escalate on budget exhaustion. | Failure reason normalized to fixed taxonomy. |
| `status.conditions[*]` | `trace` + `risk_notes` | Preserve raw condition payload in provenance attachment. | Full condition chain retained for audit. |

### Agent CRD Mapping

| kubeopencode Agent field | SDP artifact/flow | Adapter behavior | Boundary note |
| --- | --- | --- | --- |
| `metadata.name` | `execution.actor` | Bind actor id in evidence provenance. | Generic capability. |
| `spec.model` | policy allowlist gate | Validate against SDP model policy before run start. | SDP-private policy evaluation. |
| `spec.tools` | execution capability declaration | Persist declared tools in `plan` evidence section. | Generic metadata, SDP formatting. |
| `status.lastRun` | trace linkage | Attach run timestamps and controller version. | Generic input, SDP trace contract output. |

## Boundary Contracts

### Beads Contract (`Adapter <-> Beads`)

- **Inputs**: `issue_id`, current issue state, priority, dependency status.
- **Outputs**: claim/close/escalate state updates, evidence note pointer, terminal reason.
- **Rules**:
  - Beads remains source of truth for lifecycle state.
  - Adapter cannot force-close if Beads dependencies are unresolved.
  - Every terminal update includes `trace.run_context_link` and `trace.evidence_context_link`.

### FSM Contract (`Adapter <-> Transition Controller`)

- **Canonical path**: `open -> in_progress -> review -> verified -> done`.
- **Side states**: `blocked`, `escalated`, `cancelled`.
- **Rules**:
  - CRD event translation must be monotonic; no backward transitions without explicit reopen event.
  - Transition denials produce typed reasons: `policy_denied`, `verification_failed`, `dependency_blocked`, `runtime_failed`.

### Evidence Contract (`Adapter <-> Artifact Bus`)

- Required sections: `intent`, `plan`, `execution`, `verification`, `review`, `risk_notes`, `boundary`, `provenance`, `trace`.
- Adapter writes machine-readable envelope compatible with `specs/runtime/kubeopencode-sdp-adapter-contract.json`.
- Provenance requires CRD UID/resourceVersion chain and controller build fingerprint.

### Policy Contract (`Adapter <-> Policy Layer`)

- Gates enforced at three checkpoints:
  1. pre-dispatch model allowlist,
  2. pre-close risk threshold,
  3. publish/PR go-no-go.
- Denials are deterministic and persisted in both Task status annotations and Beads notes.
- Policy bundles are internal-only and not part of upstream kubeopencode API surface.

## Integration Scenarios

1. **Happy path**: Beads issue `open` -> Task succeeds -> policy passes -> evidence persisted -> issue `done`.
2. **Retry then escalate**: Task fails with retriable reason -> bounded retry budget consumed -> issue `escalated` with terminal reason.
3. **Policy denial**: Task succeeds but risk gate fails -> issue `blocked`; evidence includes deny reason and remediation hints.
4. **Duplicate dispatch**: second Task create for same issue/run rejected by lock manager; original run continues.

These scenarios validate AC2 bridge behavior across Beads/FSM/evidence/policy integration without modifying core kubeopencode reconciliation logic.

## Migration and Rollback

### Migration Plan

1. **Phase 0 - Shadow mode**: adapter reads Task/Agent events and produces evidence drafts without mutating Beads.
2. **Phase 1 - Controlled write**: enable Beads state updates for selected labels (`lane:adapter-canary`).
3. **Phase 2 - Full activation**: all kubeopencode-backed runs use adapter contracts.

### Rollback Plan

1. Disable adapter mutation flag (`SDP_ADAPTER_WRITE_ENABLED=false`) to stop Beads writes.
2. Revert run routing to baseline probe workflow in `docs/KUBEOPENCODE_MULTI_ROLE_PROBE_RUNBOOK.md`.
3. Preserve created evidence artifacts for audit; mark affected Beads issues `blocked` with rollback reason.
4. Reconcile in-flight Task objects by annotating `sdp.dev/rollback=true` and preventing auto-close.

Rollback is safe because adapter boundaries are additive and do not require CRD schema forks.

## Acceptance Mapping (`sdp_dev-2aq.7.1`)

- **AC1**: deterministic lifecycle translation specified in Task/Agent mapping tables and FSM rules.
- **AC2**: bridge contracts plus integration scenarios defined for beads/FSM/evidence/policy.
- **AC3**: boundary contracts isolate generic operator fields from SDP-private policy and evidence logic.

## Evidence Notes (Task Closure)

- Added architecture contract and CRD mapping design at `docs/KUBEOPENCODE_SDP_ADAPTER_ARCHITECTURE.md`.
- Added machine-readable adapter contract at `specs/runtime/kubeopencode-sdp-adapter-contract.json`.
- Added JSON Schema helper for validation at `specs/runtime/schemas/kubeopencode-sdp-adapter-contract.schema.json`.

<!-- markdownlint-enable MD013 -->
