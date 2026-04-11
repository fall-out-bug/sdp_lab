# Autonomous K8s Agent Swarm — Dream Product Design

> **Status:** Research complete
> **Date:** 2026-02-22
> **Goal:** Design the dream product — autonomous K8s agent swarm on opencode — from ecosystem parts + SDP evidence layer

---

## Overview

### The Dream

Issue goes in. PR with proof comes out. Fully autonomous. K8s-native. Multi-model. Auditable.

### Key Decisions

| Aspect | Decision |
|--------|----------|
| Architecture | **Evidence-as-a-Platform** — SDP is a service, not an orchestrator |
| Multi-role pipeline | **Fix AgentRunReconciler** — sequential analyst→coder→reviewer with handoff artifacts |
| Evidence flow | **NATS JetStream** — each pod publishes fragments, assembler materializes envelope |
| Intake | **CRD-only** — all sources create AgentRun CRD, no NATS for dispatch |
| Model selection | **ConfigMap policy** — controller resolves per-role, annotation overrides |
| What we build | **~6K LOC** of novel evidence code. Ecosystem handles the rest. |

---

## 1. Architecture: What Is Ours vs Ecosystem

> **Experts:** Sam Newman, Kelsey Hightower, Martin Kleppmann

### The Composition

```
┌───────────────────────────────────────────────────────┐
│                 ECOSYSTEM (not our code)               │
│                                                        │
│  beads ─── issue tracking, dependencies                │
│  kubeopencode ─── Task CRD, Agent CRD, pod lifecycle   │
│  opencode ─── agent runtime (tools, LLM, git)          │
│  OpenRouter ─── multi-model access                      │
│  NATS JetStream ─── evidence event stream               │
│                                                        │
└────────────────────────┬──────────────────────────────┘
                         │
┌────────────────────────┼──────────────────────────────┐
│         SDP CUSTOM CODE (~6K LOC)                      │
│                         │                              │
│  ┌──────────────────────▼──────────────────────────┐  │
│  │            adapter-controller                    │  │
│  │  AgentRunReconciler: analyst→coder→reviewer      │  │
│  │  IntentTranslator: beads issue → Task CRD spec   │  │
│  │  PolicyGate: model allowlist + budget check      │  │
│  │  EvidenceAssembler: NATS fragments → envelope    │  │
│  └──────────────────────────────────────────────────┘  │
│                                                        │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐  │
│  │ evidence/    │  │ artifact/    │  │ pr/         │  │
│  │ strict valid │  │ hash chain   │  │ PR gate +   │  │
│  │ 9-section    │  │ provenance   │  │ publish     │  │
│  └──────────────┘  └──────────────┘  └─────────────┘  │
│                                                        │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐  │
│  │ evaluator/   │  │ selfimprove/ │  │ policy/     │  │
│  │ multi-persona│  │ failure      │  │ model cfg   │  │
│  │ adversarial  │  │ patterns     │  │ budget      │  │
│  │ [research]   │  │ [research]   │  │             │  │
│  └──────────────┘  └──────────────┘  └─────────────┘  │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │            beads-bridge (CronJob)                │  │
│  │  bd ready per project → create AgentRun CRDs     │  │
│  │  ~50 lines of Go                                 │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────┘
```

### What Gets Deleted

| Package / Binary | LOC | Replaced By |
|-----------------|-----|-------------|
| orchestrator/ | 929 | kubeopencode + AgentRunReconciler |
| parallel/ | 499 | kubeopencode DependsOn |
| swarm/ | 107 | AgentRunReconciler phases |
| roles/ | 298 | opencode agent prompts |
| agent/ | 885 | opencode plugin system |
| swarm-worker | 1,573 | kubeopencode pods |
| swarm-orchestrator | 118 | beads-bridge CronJob |
| feature-orchestrator | 344 | beads-bridge + reconciler |
| autonomy-worker | 596 | kubeopencode |
| intake-gateway | 404 | AgentRun CRD directly |

**~5,753 LOC deleted.** Not wasted — it taught us how agents work. kubeopencode does it better.

### What Stays (Novel)

| Package | LOC | Why Novel |
|---------|-----|-----------|
| evidence/ | 384+264 | Strict 9-section validation. Nobody has this. |
| artifact/ | 1,177+811 | SHA256 hash chain provenance. Nobody has this. |
| adapter/ | 1,368+1,267 | AgentRunReconciler + EvidenceAssembler. Novel bridge. |
| pr/ | 697+597 | Evidence-gated PR gates. Nobody has this. |
| policy/ | 619+312 | Model config + budget. Thin but useful. |
| evaluator/ | 1,281+818 | Multi-persona adversarial review. Research. |
| selfimprove/ | 365+255 | Failure pattern detection. Research. |
| beads/ | 252+142 | Beads adapter. Thin glue. |

---

## 2. Multi-Role Pipeline: analyst → coder → reviewer

> **Experts:** Martin Kleppmann, Kelsey Hightower, Sam Newman, Martin Fowler

### The Flow

```
                    AgentRun created
                         │
                   ┌─────▼──────┐
                   │  "" phase   │ Controller creates analyst Task
                   └─────┬──────┘
                         │
                   ┌─────▼──────┐
                   │ Analyzing  │ kubeopencode runs analyst pod
                   │            │ analyst writes .sdp/handoff/<id>/analyst.json
                   └─────┬──────┘
                         │
                   ┌─────▼──────┐
                   │ Coding     │ Controller creates coder Task
                   │            │ coder reads analyst.json, implements
                   │            │ coder writes .sdp/handoff/<id>/coder.json
                   └─────┬──────┘
                         │
                   ┌─────▼──────┐
                   │ Reviewing  │ Controller creates reviewer Task
                   │            │ reviewer reads analyst.json + coder.json
                   │            │ reviewer writes verdict
                   └─────┬──────┘
                         │
                    ┌────▼────┐
                    │Approved?│
                    │         │
               YES  │    NO   │
                ┌───┘    └────┐
                ▼             ▼
          ┌──────────┐  ┌──────────┐
          │Succeeded │  │ Rework   │ → back to Coding
          │ PR gate  │  │ (max 2)  │
          │ PR pub   │  └──────────┘
          └──────────┘
```

### Handoff Artifacts

Each role writes a structured JSON file that the next role reads:

```
.sdp/handoff/<issue_id>/
  analyst.json   ← risk notes, decomposed steps, recommended approach
  coder.json     ← changed files, test results, implementation notes
  reviewer.json  ← verdict, findings, suggestions
```

The handoff contract is strict: each file has a JSON Schema. The controller injects the path into the next role's prompt via Task CRD annotations.

### Why Not kubeopencode DependsOn?

DependsOn creates all Tasks upfront. This breaks:
- **Rework loops** — reviewer rejects, coder retries. Can't do with upfront Tasks.
- **Artifact injection** — coder prompt depends on analyst output, which doesn't exist at creation time.
- **Budget checks** — controller can check budget before each step, not just at the start.

The AgentRunReconciler is a thin state machine (~200 LOC) that manages 6 phases. kubeopencode handles the hard part (running pods, git, tools).

---

## 3. Evidence Flow

> **Experts:** Martin Kleppmann, Kelsey Hightower, Troy Hunt

### When Each Section Gets Filled

| Section | When | Which Pod | How |
|---------|------|-----------|-----|
| intent | Dispatch | Controller | AgentRun spec → NATS fragment |
| plan | After analyst | Analyst pod | Published via bus.Publish() |
| execution | After coder | Coder pod | Changed files, branch, commands |
| verification | After coder | Coder pod | Test results, coverage |
| review | After reviewer | Reviewer pod | Verdict, findings |
| risk_notes | After reviewer | Reviewer pod | Residual risks |
| boundary | Dispatch + after coder | Controller + Coder | Declared (controller), observed (coder) |
| provenance | Each step | All pods | Hash chain: sequence, hash_prev |
| trace | Accumulated | Assembler | beads_ids, branch, commits, pr_url |

### NATS JetStream Evidence Stream

```
Agent pod publishes:
  sdp.evidence.<issueID>.plan          (analyst)
  sdp.evidence.<issueID>.execution     (coder)
  sdp.evidence.<issueID>.verification  (coder)
  sdp.evidence.<issueID>.review        (reviewer)
  sdp.evidence.<issueID>.risk_notes    (reviewer)

EvidenceAssembler subscribes:
  sdp.evidence.<issueID>.>

Assembles 9-section envelope → validates → writes .sdp/evidence/<issueID>.json
```

**Why NATS, not shared PVC?**
- Each fragment is individually hash-chained (tamper-evident by construction)
- JetStream provides replay, at-least-once delivery, retention
- Only needs RWO PVC (assembler writes, pr-gate reads) — works on any K8s cluster
- Multi-cluster ready via NATS superclusters
- Existing `bus.Bus` and `TraceEmitter` already publish to NATS — natural extension

**Why not CRD status?**
- etcd 1.5MB object limit — large evidence payloads won't fit
- CRD status is mutable — weaker tamper-evidence than append-only NATS stream
- No replay/retention

### Hash Chain Across Runs

The `BusService.Ingest()` already enforces:
- `sequence = len(existing_artifacts_for_issue)`
- `hash_prev = last_artifact.Hash`
- `ValidateAppend()` rejects broken chains

If an issue spawns a second AgentRun (retry), the new run's first fragment has `hash_prev = head_hash_of_previous_run`. The chain extends, never restarts.

---

## 4. Intake & Model Selection

> **Experts:** Kelsey Hightower, Martin Kleppmann, Troy Hunt

### Intake: CRD-Only

```
Telegram bot → creates AgentRun CRD
GitHub webhook → creates AgentRun CRD
beads-bridge CronJob → bd ready → creates AgentRun CRDs
Manual → kubectl apply AgentRun
```

No NATS for dispatch. K8s API server is the "message bus." The throughput math: at $50/day budget, maximum ~100 runs/day. K8s API handles 100 QPS trivially.

NATS stays only for evidence streaming (JetStream) and optional lifecycle events. Intake is pure CRD.

### Model Selection: ConfigMap + Controller Resolution

```yaml
# model-policy ConfigMap
roles:
  analyst:
    primary: glm-5
    fallback: openrouter/anthropic/claude-sonnet-4.6
    economy: openrouter/deepseek/deepseek-v3.2
  coder:
    primary: glm-4.7
    fallback: openrouter/moonshotai/kimi-k2.5
    economy: openrouter/deepseek/deepseek-v3.2
  reviewer:
    primary: glm-5
    fallback: openrouter/anthropic/claude-sonnet-4.6
    economy: openrouter/minimax/minimax-m2.5
budget:
  daily_limit_usd: 50.0
  per_run_limit_usd: 5.0
  auto_downgrade_at_pct: 80
```

Controller resolves: AgentRun.spec.workstream → role → ConfigMap policy → resolved model.
Writes `status.resolvedModel` for audit. Passes to Task pod as env var.
Budget tracked in a `budget-status` ConfigMap, checked before each Task creation.

---

## Implementation Plan

### Phase 1: Evidence Foundation (F001 + F002 — already planned)

- [ ] Formalize evidence JSON Schema
- [ ] Extract sdp-evidence CLI (validate + inspect)
- [ ] Goreleaser binary releases

### Phase 2: Pipeline Fix

- [ ] Make AgentRunReconciler sequential (analyst → coder → reviewer)
- [ ] Define handoff artifact schema (.sdp/handoff/<id>/<role>.json)
- [ ] Inject handoff artifacts into prompts via Task annotations
- [ ] Add rework loop (reviewer rejects → coder retries, max 2)

### Phase 3: Evidence Stream

- [ ] NATS JetStream evidence stream (sdp.evidence.<issueID>.<section>)
- [ ] Evidence fragment publisher in agent pods
- [ ] EvidenceAssembler in adapter-controller
- [ ] Hash chain validation on ingest

### Phase 4: Intake Simplification

- [ ] beads-bridge CronJob (bd ready → AgentRun CRD)
- [ ] Delete swarm-orchestrator, swarm-worker, intake-gateway, feature-orchestrator
- [ ] Wire model-policy ConfigMap into AgentRunReconciler
- [ ] Budget tracking in ConfigMap

### Phase 5: Ecosystem Integration (F004 + F005)

- [ ] kubeopencode upstream PRs (UP-001 retry budget, UP-003 evidence hooks)
- [ ] awesome-opencode submission
- [ ] Blog post: "Evidence for Autonomous Agent Swarms"

---

## Bill of Materials

| Component | Source | Status |
|-----------|--------|--------|
| Agent runtime | **opencode** (OSS) | Stable |
| K8s operator | **kubeopencode** (OSS) | Stable, contributing upstream |
| Issue tracking | **beads** (OSS) | Stable |
| Multi-model access | **OpenRouter** (SaaS) | Stable |
| Event streaming | **NATS JetStream** (OSS) | Stable |
| Evidence validation | **SDP** (our code) | Implemented |
| Provenance chain | **SDP** (our code) | Implemented |
| PR gate | **SDP** (our code) | Implemented |
| Pipeline controller | **SDP** (our code) | Needs fixing |
| Evidence assembler | **SDP** (our code) | Needs building |
| Model policy | **SDP** (our code) | 80% done |
| Multi-persona review | **SDP** (research) | Working, not production |
| Self-improvement | **SDP** (research) | Working, not production |
| beads-bridge | **SDP** (our code) | ~50 LOC, needs building |

### What We DON'T Build

| Function | Don't Build | Use Instead |
|----------|-------------|-------------|
| Agent orchestration | -- | kubeopencode |
| Parallel execution | -- | kubeopencode DependsOn |
| Agent tooling (git, shell, search) | -- | opencode |
| LLM API routing | -- | OpenRouter |
| Policy enforcement | -- | ConfigMap (simple) or Cupcake (complex) |
| Session management | -- | opencode |
| Dashboard | -- | kubectl + optional Grafana |

---

## Success Metrics

| Metric | Baseline | Target |
|--------|----------|--------|
| Custom LOC | ~25K | ~6K |
| Binaries | 27 | 4 (adapter-controller, pr-gate, beads-fsm, beads-bridge) |
| Evidence CLI released | No | Yes |
| E2E issue→PR with evidence | Partial | 10 consecutive runs |
| kubeopencode upstream PRs | 0 merged | >= 1 |
| awesome-opencode listing | No | Yes |
| External user of evidence CLI | 0 | >= 1 |
