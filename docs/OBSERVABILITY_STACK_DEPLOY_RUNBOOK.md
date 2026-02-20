# Observability Stack Deploy Runbook

This runbook covers deployment and sanity checks for the `sdp-observability` stack:

- Prometheus for metrics
- Loki for JSONL log telemetry
- Tempo for traces
- Grafana for dashboards/datasource access
- Promtail for Kubernetes log collection and JSONL parsing
- OpenTelemetry Collector for OTLP ingest + export fanout

## 1. Preconditions

- remote host has kubectl access to target cluster
- namespace bootstrap already done (`sdp-observability`)
- local machine can SSH into remote host

Recommended precheck:

```bash
./scripts/check_remote_k8s.sh --host <user@ip-or-host> --port <port>
```

## 2. Apply manifests

Deploy the stack from this repo to the remote cluster:

```bash
./scripts/apply_observability_manifests.sh --host <user@ip-or-host> --port <port>
```

This script copies `deploy/k8s/observability/` to remote `/tmp/sdp-dev-observability`, applies kustomize resources, waits for rollouts, and prints pod/service state.

## 3. Telemetry ingestion path (JSONL)

Runtime components emit JSONL telemetry to stderr (see `docs/OBSERVABILITY_METRICS_TRACE_SCHEMA_INTAKE.md`).

In-cluster path:

1. container stdout/stderr -> Kubernetes pod logs (`/var/log/pods/...`)
2. `promtail` daemonset tails pod logs in `sdp-workers` and `sdp-control`
3. Promtail pipeline parses JSON fields:
   - `record_type`
   - `event.trace.run_id`
   - `event.protocol.issue_id`
   - `event.protocol.phase`
   - `event.protocol.status`
   - `event.system.component`
   - `event.system.agent_role`
   - `event.model.name`
   - `metric.name`
4. parsed values are promoted to Loki labels for query slices by run, issue, phase, and model
5. Grafana reads Loki datasource for protocol/event timelines

OTLP path for future native metrics/traces:

- clients -> `otel-collector:4317/4318`
- traces -> Tempo
- metrics -> Prometheus scrape endpoint from OTel collector

## 4. Sanity-check commands

Quick end-to-end check:

```bash
./scripts/sanity_check_observability_remote.sh --host <user@ip-or-host> --port <port>
```

Manual checks:

```bash
ssh <user@ip-or-host> -p <port> "kubectl -n sdp-observability get deploy,ds,pod,svc"
ssh <user@ip-or-host> -p <port> "kubectl -n sdp-workers logs deploy/opencode-agent --tail=200"
ssh <user@ip-or-host> -p <port> "kubectl -n sdp-observability logs ds/promtail --tail=200"
```

Grafana access (from remote host):

```bash
ssh <user@ip-or-host> -p <port> "kubectl -n sdp-observability port-forward svc/grafana 3000:3000"
```

Default credentials are in `deploy/k8s/observability/grafana-admin-secret.yaml` and should be replaced for shared environments.

## 5. Provisioned dashboards

Grafana is provisioned with dashboard definitions from:

- `deploy/k8s/observability/grafana-dashboard-provider-configmap.yaml`
- `deploy/k8s/observability/grafana-dashboards-configmap.yaml`

Expected dashboards:

- `SDP Model Comparison`
- `SDP End-to-End Flow Metrics`

These dashboards are aligned with Promtail labels populated from the unified intake schema:

- `phase`, `status`, `component`, `model_name`, `metric_name`, `issue_id`, `run_id`, `record_type`

## 6. Operator incident workflow

When alerts fire or operators observe degraded flow:

1. Open `SDP End-to-End Flow Metrics` and identify impacted `phase/status`.
2. Pivot to `SDP Model Comparison` to isolate affected `model_name` and escalation/fallback volume.
3. Capture at least one `run_id` and `issue_id` from Loki query results.
4. Correlate with application logs from `sdp-workers` / `sdp-control` and link to PR/evidence context when present.
5. Route to owning team (model, platform, or protocol) with dashboard snapshot and query.

For SLO definitions and escalation thresholds, see `docs/OBSERVABILITY_SLO_ALERTING_GUIDE.md`.
