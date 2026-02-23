# FR-014: Telemetry Analyzer Agent (Continuous Backlog Generation)

Priority: P1
Effort: 5d
Dependencies: FR-017 (telemetry hardened), FR-011 (cross-project registry)

## Problem

Currently tasks are created only manually. An agent is needed that:
- Continuously analyzes telemetry of closed tasks (latency, success rate, failure patterns)
- Examines closed Beads issues (what was done, what patterns)
- Generates new features/tasks for the backlog
- Proposes improvements for k8s orchestrator and SDP protocol

## Existing Code

| Component | Status | Location |
|-----------|--------|----------|
| `retro-agent` | ✅ Exists | `cmd/retro-agent/` — subscribes to `sdp.beads.*.closed` |
| `self-improve-agent` | ✅ Exists | `cmd/self-improve-agent/` — self-improvement telemetry |
| `internal/retrospective/` | ✅ Exists | Retrospective analysis (2 files) |
| `internal/selfimprove/` | ✅ Exists | Self-improvement metrics (5 files) |
| `internal/observability/` | ✅ Exists | Unified telemetry schema |
| `internal/discuss/openrouter_analyzer.go` | ✅ Exists | Feature analysis via OpenRouter |

## Design

### Architecture

```
Telemetry Sources                   Analyzer Agent                    Output
─────────────────                   ──────────────                    ──────
NATS sdp.beads.*.closed ──┐
                          ├──► telemetry-analyzer ──► LLM Analysis ──► Beads Issues
NATS sdp.lifecycle.*.* ───┤    (continuous loop)       (claude/o3)     (type: feature/task)
                          │                                           
.sdp/evidence/*.json ─────┤    Aggregation:                          ──► NATS sdp.intake.*
                          │    - Success/failure rates                   (for orchestrator)
Observability JSONL ──────┘    - Latency distributions
                               - Common failure patterns
                               - Coverage gaps
                               - Model performance
```

### Agent Loop

```go
func (a *TelemetryAnalyzer) Run(ctx context.Context) error {
    ticker := time.NewTicker(a.interval) // default: 30min
    for {
        select {
        case <-ticker.C:
            // 1. Collect recent closed issues (last interval)
            closed := a.beads.ListClosed(since)
            
            // 2. Collect telemetry for those runs
            telemetry := a.collectTelemetry(closed)
            
            // 3. Analyze patterns via LLM
            analysis := a.llm.Analyze(telemetry, a.projectContext)
            
            // 4. Generate feature/task proposals
            proposals := a.generateProposals(analysis)
            
            // 5. Create Beads issues (type: feature, priority: P2-P3)
            for _, p := range proposals {
                a.beads.Create(p)
            }
            
            // 6. Emit telemetry about own analysis
            a.trace.Emit("telemetry-analysis", analysis.Summary)
        }
    }
}
```

### Analysis Dimensions

1. **Reliability**: success rate, failure taxonomy, retry effectiveness
2. **Performance**: latency distribution, model response times
3. **Quality**: evidence completeness, review pass rate, boundary violations
4. **Coverage**: workstream coverage, untested paths
5. **Cost**: model usage, token consumption, provider distribution
6. **Process**: bottleneck detection, blocked task patterns

### Output Types

- **Feature proposals** (P2-P3): "Add retry for OpenRouter timeout failures"
- **Bug reports** (P1-P2): "Evidence projection fails for multi-file PRs"
- **Improvement tasks** (P2): "Reduce analyst latency by switching to faster model"
- **SDP protocol updates**: "Add new evidence section for cost tracking"

## Acceptance Criteria

- [ ] Agent runs as K8s Deployment in sdp-control namespace
- [ ] Subscribes to NATS `sdp.beads.*.closed` and `sdp.lifecycle.*.*`
- [ ] Reads .sdp/evidence/ from workspace
- [ ] LLM analysis via configurable model (FR-012)
- [ ] Generates Beads issues via `internal/beads` adapter
- [ ] Proposals tagged with `source:telemetry-analyzer`
- [ ] Rate limiting: max N proposals per cycle
- [ ] Duplicate detection: does not create duplicates of existing issues
- [ ] Self-telemetry: emits own metrics to observability pipeline
- [ ] ConfigMap: analysis interval, max proposals, LLM model, analysis dimensions
