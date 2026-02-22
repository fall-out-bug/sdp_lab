# feature-orchestrator: SRE Operations

## SEV levels and escalation (sdp_dev-k6d)

| SEV | Definition | Response |
|-----|------------|----------|
| SEV1 | feature-orchestrator down or no AgentRuns created for >15 min; NATS/aggregator unreachable | Page on-call; restart deployment; check NATS and K8s API |
| SEV2 | Dispatch latency >5 min p99; queue depth growing unbounded | Investigate within 2h; scale workers or increase maxConcurrent |
| SEV3 | Intermittent create failures; metrics gaps | Triage within 1 day; fix and add tests |

**Escalation:** SEV1 → SRE + TechLead; SEV2 → SRE; SEV3 → backlog.

---

## SLOs / SLIs (sdp_dev-21l)

| SLI | Target | Measurement |
|-----|--------|-------------|
| Poll success | 99% of poll cycles complete without error | `sdp_feature_orchestrator_dispatched_total` rate; log/alert on dispatch loop errors |
| Dispatch latency | p95 < 2 min from ready to AgentRun created | Trace or timestamp from aggregator ready to K8s Create |
| AgentRun creation rate | ≥1 per 10 min when ready backlog non-empty | `dispatched_total` delta; alert if 0 for 10 min with `active_runs` or queue depth > 0 |

**Alerting:** Prometheus alerts on `sdp_feature_orchestrator_active_runs` (stuck at 0 or very high), `dispatched_total` (no increase when expected).

---

## Incident runbook (sdp_dev-3dl)

### Symptoms

- No new AgentRuns created although Beads has ready issues.
- Pod CrashLoopBackOff or NotReady.
- NATS connection errors in logs.
- K8s API errors (RBAC, quota).

### Diagnosis

1. **Pod status:** `kubectl get pods -n sdp-control -l app=feature-orchestrator`
2. **Logs:** `kubectl logs -n sdp-control -l app=feature-orchestrator --tail=200`
3. **Metrics:** `GET http://<pod>:8080/metrics` — check `sdp_feature_orchestrator_active_runs`, `sdp_feature_orchestrator_dispatched_total`
4. **NATS:** Ensure NATS is up and `sdp.beads.*.ready` is being published (bridge/aggregator).
5. **Registry:** ConfigMap `project-registry` and env `SDP_REGISTRY_PATH`; ensure projects exist and workspaces can be ensured.

### Mitigation

| Issue | Action |
|-------|--------|
| Pod crash | `kubectl delete pod -n sdp-control -l app=feature-orchestrator` (let Deployment recreate); check NATS_URL and K8s config |
| RBAC | Verify Role/RoleBinding in sdp-workers for `agentruns` (create, get, list, watch, update, patch) |
| No ready events | Verify federation bridge and aggregator; ensure Beads ready snapshot is published to `sdp.beads.<project>.ready` |
| Lease lock stuck | Leases in sdp-workers namespace; delete Lease for stuck issue if safe: `kubectl delete lease -n sdp-workers sdp-ar-<issue-id>` |

### Post-incident

- Update this runbook if new failure mode found.
- Consider adding alert on dispatch loop error rate or queue depth.
