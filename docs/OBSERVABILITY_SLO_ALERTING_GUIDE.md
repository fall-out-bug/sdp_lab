# Observability SLO, Alerting, and Escalation Guide

This guide defines baseline SLOs and operational alert guidance for the observability stack and telemetry schema delivered in `sdp_dev-2aq.20.2` and `sdp_dev-2aq.20.3`.

## 1. Scope and data contract alignment

These SLOs are built from fields that are guaranteed by the unified intake contract in `docs/OBSERVABILITY_METRICS_TRACE_SCHEMA_INTAKE.md`:

- `event.protocol.phase`
- `event.protocol.status`
- `event.system.component`
- `event.model.name`
- `metric.name=protocol_phase_latency_ms`

For Grafana/Loki queries, these map to Promtail labels:

- `phase`, `status`, `component`, `model_name`, `metric_name`, `record_type`, `issue_id`, `run_id`

## 2. Service level objectives

Use these as starting points. Tighten after two weeks of baseline data.

### SLO-A: End-to-end protocol success rate

- Objective: `>= 99%` successful protocol transitions over 30m windows.
- Signal (Loki): ratio of `status="success"` events against all terminal outcomes (`success|failed|fallback|escalated|blocked`).
- Suggested alert threshold: page if `< 97%` for 15m.

Example query pair:

```logql
sum(count_over_time({namespace=~"sdp-(workers|control)",record_type="event",status="success"}[30m]))
```

```logql
sum(count_over_time({namespace=~"sdp-(workers|control)",record_type="event",status=~"success|failed|fallback|escalated|blocked"}[30m]))
```

### SLO-B: Protocol phase latency

- Objective: p95 phase latency `< 1500ms` for `protocol_phase_latency_ms` (15m windows).
- Signal (Loki metric records): `record_type="metric"` and `metric_name="protocol_phase_latency_ms"`.
- Suggested alert threshold: warn when p95 `> 1500ms` for 15m, page when p95 `> 2500ms` for 10m.

Example query:

```logql
quantile_over_time(0.95, {namespace=~"sdp-(workers|control)",record_type="metric",metric_name="protocol_phase_latency_ms"} | json metric_value_ms="metric.value_ms" | unwrap metric_value_ms [15m])
```

### SLO-C: Escalation pressure

- Objective: escalated transitions `< 1%` of total terminal outcomes over 30m windows.
- Signal: `status="escalated"` relative to terminal outcomes.
- Suggested alert threshold: page if `>= 3%` for 10m or if any single issue/run escalates repeatedly.

Example query pair:

```logql
sum(count_over_time({namespace=~"sdp-(workers|control)",record_type="event",status="escalated"}[30m]))
```

```logql
sum(count_over_time({namespace=~"sdp-(workers|control)",record_type="event",status=~"success|failed|fallback|escalated|blocked"}[30m]))
```

## 3. Alerting and escalation policy

Use a two-level policy:

- **Warn** (Slack/on-call channel): SLO drift but no severe outage signal.
- **Page** (primary on-call): sustained breach or rapid degradation.

Escalation ladder:

1. On-call operator triages in Grafana dashboards (`SDP Model Comparison`, `SDP End-to-End Flow Metrics`).
2. If issue is model-localized, route to model/inference owner with affected `model_name` and `component`.
3. If issue is cross-component, escalate to platform owner with `phase` and `issue_id` evidence.
4. If unresolved after 30 minutes for page-level incident, escalate to incident commander.

Required payload for handoff:

- impacted window (UTC)
- top affected `phase` / `status`
- affected `model_name`
- example `run_id` and `issue_id`
- dashboard snapshot URL and raw Loki query

## 4. Recommended Prometheus-style alert names

Use these names even if implemented in Grafana Alerting first:

- `SDPProtocolSuccessRateLow`
- `SDPProtocolP95LatencyHigh`
- `SDPEscalationRateHigh`
- `SDPObservabilityIngestionSilent` (no event/metric records observed in expected window)

## 5. Operational review cadence

- Daily: check previous 24h SLO trend and top failing phases.
- Weekly: tune thresholds with observed baseline and deployment changes.
- Release gate: for changes touching protocol routing/model policy, validate all three SLOs in staging before prod rollout.
