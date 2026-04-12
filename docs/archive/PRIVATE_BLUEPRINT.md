# Private Blueprint: L3 Autonomy + Repo Split

Status: active draft
Owner: SDP core team
Date: 2026-02-19

## 1. Objective

Build an L3-first autonomous development system with strict controls:

- 3 OSS repositories: `sdp-protocol`, `sdp-plugin`, `sdp-orchestrator`
- 1 private repository: `sdp-enterprise` (security + self-evolution + commercial loop)
- Autonomy scope: full path to PR, merge stays manual (human gate)

## 2. Confirmed decisions (interview lock)

1. Brain is private and controlled by our team.
2. Brain is deeply integrated with OpenCode.
3. OpenClaw is integrated through adapter contract and later deployed in k8s.
4. Delivery target is PR creation, never auto-merge.
5. Git model is feature branches.
6. Security features and agent self-evolution stay private.
7. Implementation order is A -> C -> B:
   - A: OpenCode autonomy first
   - C: repo split
   - B: unified runtime contract and OpenClaw parity
8. Evidence policy is Strict (mandatory for every autonomous PR).
9. Swarm runtime will run on k8s on a neighboring machine; setup via SSH.
10. Orchestrator model scope is temporarily restricted to `glm-4.7` and `glm-5`.
11. Stack policy is Go-first for production path; Python is research-only (`docs/ADR-0001-go-first-stack.md`).

## 3. Private system architecture

### 3.1 Control model

- `Brain` (private): policy engine, routing, risk classification, approval logic, self-evolution pipeline.
- `Swarm Runtime`: agent workers, scheduler, verifier, reviewer, release bot.
- `Beads Control Graph`: source of execution truth (dependencies, status, acceptance criteria, trace links).

### 3.2 Runtime topology

- Local machine:
  - OpenCode control surface
  - Private planning/config repository (`sdp_dev`)
  - SSH orchestration client for remote cluster
- Remote machine (same network):
  - k8s cluster for swarm execution
  - later: OpenClaw deployment

### 3.3 k8s deployment layers

- Namespace `sdp-control`:
  - Brain API (private)
  - Scheduler
  - Policy service
- Namespace `sdp-workers`:
  - Builder workers
  - Verifier workers
  - Reviewer workers
- Namespace `sdp-observability`:
  - metrics collector
  - evidence indexer
  - audit exporter
- Namespace `sdp-openclaw` (later):
  - OpenClaw runtime + adapter sidecar

### 3.4 Model policy (current phase)

- Allowed models for orchestration runtime:
  - `glm-5` (primary)
  - `glm-4.7` (fallback)
- Any attempt to schedule another model is a policy violation and must be escalated.
- Model expansion is a separate private policy change, not a runtime default.

## 4. Strict evidence bundle (mandatory)

Every PR created by agents must include all sections below:

1. `intent`
   - feature/task id
   - user or system trigger
   - acceptance criteria
   - risk class
2. `plan`
   - workstream decomposition
   - dependency order and rationale
3. `execution`
   - claimed beads IDs
   - branch name
   - file changes summary
4. `verification`
   - test results
   - lint/type/contract checks
   - coverage delta
5. `review`
   - self-review findings
   - adversarial review findings
6. `risk_notes`
   - known residual risks
   - excluded scope
7. `trace`
   - beads -> branch -> commits -> PR linkage

No section = no PR publication.

## 5. Beads state machine for autonomy

Canonical flow:

`open -> in_progress -> review -> verified -> done`

Supporting states:

- `blocked` (must contain dependency link, not only notes)
- `escalated` (human decision needed)
- `cancelled`

Hierarchy model in Beads:

- `epic -> feature -> task`
- autonomous workers claim only `task`

Transition rules:

- `open -> in_progress`: only after claim lock and branch assignment
- `in_progress -> review`: only after local quality precheck pass
- `review -> verified`: only after strict evidence bundle complete
- `verified -> done`: only after PR opened and linked

## 6. Autonomous scenarios (north-star)

### Scenario A: User-driven feature -> PR

1. User provides feature request.
2. Brain creates/decomposes beads tasks.
3. Swarm executes on feature branch.
4. Strict evidence generated.
5. PR created and linked to beads.

Success condition: user receives PR with complete evidence and clear risk notes.

### Scenario B: Agent-initiated improvement -> PR

1. Agent detects improvement opportunity.
2. Brain validates value/risk threshold.
3. Task is created in beads with explicit rationale.
4. Swarm executes and verifies.
5. PR created for human merge decision.

Success condition: autonomous value discovery without violating policy boundaries.

## 7. Repo split contract (private view)

### OSS repositories

1. `sdp-protocol`
   - schemas, protocol docs, public contracts, public gates
2. `sdp-plugin`
   - lightweight CLI + generated adapters for tooling integration
3. `sdp-orchestrator`
   - public orchestrator core and execution primitives

### Private repository

4. `sdp-enterprise`
   - security policy packs
   - self-evolution engine and private tuning data
   - commercial and governance automation

## 8. Execution sequence (quality-first, no deadlines)

### Stage A (first): OpenCode autonomy

- implement private brain integration in OpenCode
- run end-to-end path `feature -> beads -> branch -> strict evidence -> PR`
- stabilize operational runbooks

### Stage C (second): repository split

- split protocol/plugin/orchestrator boundaries
- preserve compatibility bridges
- ensure no private leakage in exported OSS artifacts

### Stage B (third): shared runtime contract + OpenClaw parity

- finalize `AutonomousRuntimeModule` contract
- implement OpenClaw adapter
- deploy OpenClaw in remote k8s and validate parity with OpenCode

## 9. Self-evolution loop (private-only)

For each closed autonomous task:

1. classify failures and retries
2. detect policy or prompt weakness
3. propose patch to policy/prompt/routing
4. run safety simulation against regression set
5. either apply patch or escalate for human approval

This loop remains private and is never directly published to OSS.

## 10. Risk controls

- Merge remains human-only.
- Security-sensitive changes auto-escalate.
- Budget or provider anomalies trigger degraded mode.
- Any missing evidence section blocks PR creation.
- Private/OSS export requires redaction check from `REDACTION_RULES.md`.
- Candidate public artifact must pass `go run ./cmd/redaction-check --file <draft>`.

## 11. Next private artifacts to produce

Completed:

1. `docs/BEADS_AUTONOMY_SPEC.md` - fields, labels, transitions, blockers, risk gates.
2. `docs/RISK_POLICY.md` - risk classes and mandatory gate stack.
3. `docs/RETRY_ESCALATION_POLICY.md` - retry classes and escalation behavior.
4. `docs/BEADS_SCENARIO_BACKLOG_TEMPLATE.md` - scenario templates for `feature -> PR` and `agent-initiated -> PR`.
5. `docs/MODEL_POLICY.md` - allowed models and fallback chain for current phase.

Next:

1. `docs/OPENCODE_BRAIN_INTEGRATION_PLAN.md` - stage A implementation packet.
2. `docs/K8S_SWARM_BOOTSTRAP.md` - SSH setup, cluster namespaces, deployment order.
3. `docs/OPENCLAW_ADAPTER_PLAN.md` - parity and compatibility strategy.

## 12. Export note

When publishing any part to OSS, use `docs/OSS_EXPORT_TEMPLATE.md` and perform redaction review first.
