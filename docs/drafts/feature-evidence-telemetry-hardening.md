# FR-017: Evidence/Traces/Provenance/Telemetry Hardening

Priority: P0
Effort: 4d
Dependencies: FR-001 (adapter-controller works)

## Problem

Per codebase audit, the evidence/traces/provenance/telemetry systems are implemented at ~60%, but have critical gaps:

### Evidence System (70% → 100%)
| Gap | Impact |
|-----|--------|
| EvidenceProjector not integrated in reconcile loop | Evidence is not collected in operator path |
| No signing of evidence envelopes | No cryptographic verification |
| No completeness validation in post-reconcile | Incomplete evidence passes through |

### Trace System (60% → 100%)
| Gap | Impact |
|-----|--------|
| No OpenTelemetry export | Traces do not reach Tempo |
| No heartbeat traces for Running phase | No visibility in long runs |
| No completeness validation | Broken trace chains |
| No distributed context propagation | No correlation between agents |

### Provenance System (80% → 100%)
| Gap | Impact |
|-----|--------|
| No cryptographic signing | Hash chain without signatures |
| No verification CLI | Cannot verify provenance |

### Telemetry/Observability (40% → 90%)
| Gap | Impact |
|-----|--------|
| JSONL → stderr, no OTLP | Data does not reach collector |
| No Prometheus metrics | No alerting |
| No Grafana dashboards | No visibility |
| CRD events not in Loki | No log correlation |

## Design

### WS-017-01: Evidence Integration in Reconcile Loop

Wire EvidenceProjector into TaskReconciler post-reconcile:

```go
case Succeeded:
    evidence := r.evidenceProjector.ProjectFromIntent(task, intent)
    if err := r.evidenceValidator.Validate(evidence); err != nil {
        // Block transition, set condition
        return ctrl.Result{}, err
    }
    r.fsm.Transition(issueID, "review")
```

### WS-017-02: OpenTelemetry SDK Integration

Replace JSONL → stderr with OTLP export:

```go
import "go.opentelemetry.io/otel"

func setupTracing() (*sdktrace.TracerProvider, error) {
    exporter, _ := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint("otel-collector.sdp-observability:4318"),
    )
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

Components to instrument:
- adapter-controller (reconcile loop spans)
- swarm-worker (execute/verify/publish spans)
- swarm-reviewer (review spans)
- orchestrator (dispatch/monitor spans)

### WS-017-03: Prometheus Metrics

```go
var (
    agentRunsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "sdp_agent_runs_total"},
        []string{"project", "status", "model"},
    )
    agentRunDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "sdp_agent_run_duration_seconds"},
        []string{"project", "role"},
    )
    evidenceCompleteness = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{Name: "sdp_evidence_completeness_ratio"},
        []string{"project"},
    )
)
```

### WS-017-04: Grafana Dashboards

Create dashboards for:
1. Agent Runs Overview (success rate, latency, active runs)
2. Model Usage (tokens, cost, provider distribution)
3. Evidence Quality (completeness, violations, provenance chain)
4. System Health (NATS lag, pod restarts, resource usage)

### WS-017-05: Trace Completeness Validation

```go
type TraceValidator interface {
    ValidateChain(runID string) ([]TraceGap, error)
    RequiredPhases() []string // Pending → Running → Succeeded/Failed
}
```

## Acceptance Criteria

- [ ] EvidenceProjector integrated in TaskReconciler
- [ ] Evidence validated before FSM transition
- [ ] OTLP traces exported to collector → Tempo
- [ ] Distributed trace context propagated (W3C tracecontext)
- [ ] Prometheus metrics exposed by all components
- [ ] Grafana dashboards deployed (4 dashboards)
- [ ] Trace completeness validated before terminal
- [ ] Heartbeat traces emitted during Running phase
- [ ] Provenance chain verified on evidence write
- [ ] All existing JSONL emission preserved (backward compat)
- [ ] `go.opentelemetry.io/otel` added to go.mod
