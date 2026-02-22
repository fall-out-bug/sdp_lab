# Observability Metrics+Trace Schema Intake

This intake defines the unified event schema used by system metrics and protocol traces for `sdp_dev-2aq.20`.

## Contract

- Contract version: `observability-metrics-trace/v1`
- Canonical helper: `internal/observability/intake_contract.go`
- Regression tests: `internal/observability/intake_contract_test.go`

## Required Fields

The schema captures system, protocol, and model tags in one envelope so every event can be joined across metrics, traces, and evidence.

| Path | Type | Domain | Notes |
| --- | --- | --- | --- |
| `trace.run_id` | `string` | `protocol` | Global run key for joins and replay correlation. |
| `protocol.issue_id` | `string` | `protocol` | Issue key for per-task and per-epic slicing. |
| `protocol.phase` | `string` | `protocol` | Protocol phase (`intake`, `plan`, `execute`, `verify`, `review`, `publish`). |
| `protocol.status` | `string` | `protocol` | Phase state marker (`running`, `success`, `failed`, `blocked`, `retrying`, `fallback`, `escalated`). |
| `system.component` | `string` | `system` | Emitting component (`opencode-agent`, `swarm-worker`, `swarm-reviewer`). |
| `system.agent_role` | `string` | `system` | Emitting role (`orchestrator`, `worker`, `reviewer`). |
| `model.name` | `string` | `model` | Selected model for this event. |
| `metrics.latency_bucket` | `string` | `system` | Bucketed latency label for SLI/SLO aggregation. |
| `resilience.retry_count` | `integer` | `protocol` | Retry attempts used by the event. |
| `resilience.fallback_used` | `boolean` | `model` | Fallback route/model marker. |
| `resilience.escalated` | `boolean` | `protocol` | Escalation marker for operator/manual path. |
| `linkage.evidence_context_link` | `string` | `protocol` | Link to run/evidence artifact context. |
| `linkage.pr_url` | `string` | `protocol` | Link to published PR when present. |

## Latency Buckets

`metrics.latency_bucket` must be one of:

- `le_50ms`
- `le_100ms`
- `le_250ms`
- `le_500ms`
- `le_1000ms`
- `le_2500ms`
- `le_5000ms`
- `gt_5000ms`

## Contract Helper Expectations

`ValidateUnifiedMetricsTraceEvent` enforces:

- all required fields must be present and non-empty for string values;
- `protocol.status` must use the allowed status set;
- `metrics.latency_bucket` must use the allowed latency bucket labels.

This helper is intended for ingestion gates and CI tests before telemetry publish.

## Component Instrumentation Behavior

- `cmd/opencode-agent`, `cmd/swarm-worker`, and `cmd/swarm-reviewer` emit JSONL telemetry records to `stderr`.
- Each emit writes two records with a shared schema payload:
  - `record_type=event` for protocol transition events;
  - `record_type=metric` for phase latency (`protocol_phase_latency_ms`) with the same event tags.
- Emitted events always include required model and resilience tags:
  - `model.name`
  - `resilience.retry_count`
  - `resilience.fallback_used`
  - `resilience.escalated`
- Linkage fields are filled from run/evidence context when available:
  - `linkage.evidence_context_link` from evidence file path or trace value;
  - `linkage.pr_url` from trace value when known, otherwise `unknown` sentinel.
