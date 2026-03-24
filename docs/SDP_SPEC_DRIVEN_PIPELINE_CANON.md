# SDP Spec-Driven Pipeline Canon

Status: canon
Date: 2026-03-23
Scope: foundational design document for SDP as a spec-driven pipeline from intent to deploy
Owner: Клавдий (orchestration) + Андрей (architecture)

## Core Principle

> **Code is derivable. Contracts are not.**

SDP treats specifications, not code, as the source of truth. Code can be rewritten, refactored, or regenerated from specs. Contracts — once declared — enforce constraints that execution must satisfy.

This aligns with the broader AI-native development shift identified in industry research (GitHub Spec Kit, Thoughtworks Five Building Blocks, Patrick Debois Four Patterns):

- «We're moving from 'code is the source of truth' to 'intent is the source of truth'.» — GitHub
- «The bottleneck of development has shifted from raw coding to the precise articulation of requirements.» — Thoughtworks
- «From Implementation to Intent: developers specify WHAT is needed.» — Debois

## SDP Philosophy

Three pillars that make SDP different from generic AI-dev tools:

### 1. Provenance

Every artifact carries a traceable lineage:
- **Prompt provenance**: what was sent to the LLM (prompt hash + context source hashes)
- **Artifact provenance**: hash chain (sha256, append-only, sequence-based, hash_prev linked)
- **Execution provenance**: which agent, which model, which run, which contract version

Nothing is trusted without provenance. Nothing enters the pipeline without being traceable.

### 2. Evidence

Execution must prove it satisfied the contract:
- **TaskContract** declares what evidence is required (acceptance criteria, metrics, artifacts, quality gates)
- **TaskSnapshot** records what evidence was actually produced
- **Compliance evaluation** detects drift between contract and reality (AC drop, metric drop, scope weaken, quality gate fail)
- Evidence is mandatory, not optional. A task without evidence is an incomplete task.

### 3. Trace

The full chain from intent to deploy is observable:
- **FeatureCard** is the trace spine — every lifecycle transition leaves orchestrator trace
- **Dispatch trace** — what was dispatched, to whom, when, with what packet
- **Executor trace** — session ID, heartbeat timestamps, runtime state, progress
- **Result trace** — status, summary, artifacts, findings, open risks

---

## Pipeline: Intent to Deploy

SDP is a spec-driven pipeline. Each layer constrains the next. Upper layers are immutable unless explicitly revised through ADR.

```
┌─────────────────────────────────────────────────────────────────┐
│  CONSTITUTION                                                    │
│  project-registry.yaml, architecture principles, ADR             │
│  Constraints everything below. Immutable unless ADR.             │
├─────────────────────────────────────────────────────────────────┤
│  INTENT                                                          │
│  Natural language request from human.                            │
│  "Сделай X". Raw, unstructured, but preserved as intake truth.  │
├─────────────────────────────────────────────────────────────────┤
│  SPECIFY → FeatureCard (inbox → clarifying)                      │
│  Structured intent: normalized intent, scope, non-goals,         │
│  risk level, acceptance shape, why now                           │
│  Gate: ClarificationGate (classify additive/reductive/policy)    │
├─────────────────────────────────────────────────────────────────┤
│  READY → FeatureCard (ready)                                     │
│  Validates required fields. Ready gate.                          │
│  Auto-generates: TaskContract (derived spec from intent)         │
├─────────────────────────────────────────────────────────────────┤
│  CONTRACT → TaskContract                                         │
│  THE executable specification. Objective, AC, metrics, evidence, │
│  quality gates, constraints. This is source of truth for exec.   │
├─────────────────────────────────────────────────────────────────┤
│  DISPATCH → ExecutionPacket                                       │
│  Route to executor role, build packet from contract + card       │
│  Provenance: hash contract, record context sources               │
├─────────────────────────────────────────────────────────────────┤
│  BRIDGE → OmO Executor                                           │
│  Read packet → hydrate context → write prompt provenance         │
│  → InvokeOpenCode with agent + prompt                            │
│  → heartbeat on FeatureCard                                      │
├─────────────────────────────────────────────────────────────────┤
│  EXECUTE → opencode subprocess                                   │
│  implementer agent, workstream-by-workstream                     │
├─────────────────────────────────────────────────────────────────┤
│  CONTRACT GATE → EnforceContractGate                             │
│  Load contract + snapshot → EvaluateCompliance                   │
│  5 gates: requirement_integrity, evidence, metric_parity,        │
│          quality, process                                        │
│  BLOCKED = halt. Drift = logged.                                 │
├─────────────────────────────────────────────────────────────────┤
│  REVIEW → InvokeOpenCode (reviewer agent)                        │
│  APPROVED / CHANGES_REQUESTED                                    │
│  Changes → ClarificationGate → possible contract revision        │
├─────────────────────────────────────────────────────────────────┤
│  PR → git push + gh pr create (draft)                            │
├─────────────────────────────────────────────────────────────────┤
│  CI → sdp-ci-loop (poll PR CI status)                            │
├─────────────────────────────────────────────────────────────────┤
│  QA → InvokeOpenCode (qa agent)                                  │
│  QA_PASS / QA_FAIL → findings → reroute if needed               │
├─────────────────────────────────────────────────────────────────┤
│  RESULT → ExecutorResultPacket                                   │
│  auto-ingest → update FeatureCard → update snapshots            │
├─────────────────────────────────────────────────────────────────┤
│  DONE → trace complete                                           │
│  Full provenance chain from intent to deploy                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## Layer Descriptions

### Constitution

The immutable foundation. Defines:
- **Project registry**: what projects exist, their repos, their conventions
- **Architecture principles**: non-negotiable constraints (e.g., «every agent is opaque», «no shadow state»)
- **ADR**: documented architectural decisions with rationale

Constitution constrains every layer below. Changes to constitution require explicit ADR.

Reference: `specs/project-registry.yaml`, `docs/decisions/`

### Intent

Raw human input. Preserved verbatim as `raw_request` on FeatureCard. Never modified, never interpreted — only used as reference.

### Specify (FeatureCard lifecycle: inbox → clarifying)

Structures the intent into machine-actionable form:
- `normalized_intent`: what the feature achieves
- `scope_in` / `scope_out`: what's included and expected outputs
- `non_goals`: explicit exclusion boundaries
- `risk_level`: low/medium/high
- `acceptance_shape`: human-readable acceptance description
- `why_now`: rationale for timing
- `open_questions`: unresolved ambiguities

**Gate**: ClarificationGate classifies changes as additive, reductive, or policy-sensitive. Reductive and policy-sensitive changes require explicit approval before proceeding.

Reference: `internal/control/control.go` (FeatureCard), `internal/harness/clarification.go` (ClarificationGate)

### Ready Gate

Transition from clarifying to ready. Validates:
- Required fields are present
- `normalized_intent` is non-empty
- `scope_in` is non-empty
- Acceptance criteria are articulated

**On ready**: auto-generate TaskContract from FeatureCard. Contract becomes the executable spec.

Reference: `internal/control/update.go` (MarkReady), `internal/harness/types.go` (TaskContract)

### Contract (TaskContract)

The executable specification. Source of truth for execution:
- `objective`: derived from normalized_intent
- `acceptance_criteria`: structured, ID'd, prioritized
- `required_metrics`: measurable success indicators
- `required_evidence`: artifacts that must be produced
- `quality_gates`: build, test, lint, typecheck requirements
- `constraints`: allow_scope_reduction, security_policy, performance_budget

Contract can be amended via ClarificationChange with classification and approval tracking.

Reference: `internal/harness/types.go`, `internal/harness/clarification.go`

### Dispatch (ExecutionPacket)

Translates contract + card into executor-routing packet:
- `executor_role`: determined by RouteToExecutor (omo-implementation, review, clarification, etc.)
- `objective`: from contract
- `scope_in` / `scope_out`: from card
- `constraints`: derived from card risk/mode
- `required_artifacts` / `required_checks`: from contract
- `next_handoff_target`: routing metadata

**Provenance**: contract hash, context source hashes recorded before dispatch.

Reference: `internal/control/routing.go`, `internal/orchestrate/invoke_opencode.go` (provenance)

### Bridge (Dispatch Bridge)

Connects control tower to OmO executor. This is the transport layer:
1. Read ExecutionPacket
2. Create ContextPacket (workstream, AC, scope files, dependencies, quality gates, drift status)
3. Write prompt provenance (prompt hash + context sources)
4. InvokeOpenCode with correct agent role and prompt
5. Write heartbeat to FeatureCard (executor_runtime_state: pending → running)
6. On completion: parse result, translate to ExecutorResultPacket
7. Write result to executor-results/ for auto-ingest

**This layer currently has a gap.** Control tower writes ExecutionPacket to disk but nobody reads it. The old orchestrate loop has InvokeOpenCode but bypasses ExecutionPacket entirely. See `DISPATCH_BRIDGE_SPEC.md`.

Reference: `internal/orchestrate/invoke_opencode.go`, `internal/orchestrate/hydrate.go`

### Execute (opencode subprocess)

OmO executor runs inside opencode. The agent (implementer, reviewer, qa) receives:
- Hydrated context via prompt injection
- Workstream specification with AC and scope files
- Quality gates from AGENTS.md
- Drift status from git

Returns: stdout/stderr with commit hashes, approval verdicts, QA status.

Reference: `internal/orchestrate/llm.go` (LLMInvoker interface), `internal/orchestrate/invoke_opencode.go`

### Contract Gate (EnforceContractGate)

Evaluates compliance between contract and runtime snapshot:
1. **requirement_integrity**: AC not dropped, scope not weakened
2. **evidence**: all required evidence produced
3. **metric_parity**: all required metrics present and met
4. **quality**: build, test, lint, typecheck all pass
5. **process**: report completeness (contract coverage, gate results, evidence index, decision log)

Blocked gate halts pipeline. Drift is logged as violations.

Reference: `internal/harness/evaluate.go`, `internal/orchestrate/contract_gate.go`

### Review → PR → CI → QA → Result

Sequential verification pipeline:
- **Review**: InvokeOpenCode (reviewer), check for APPROVED
- **PR**: git push + gh pr create --draft
- **CI**: sdp-ci-loop polls PR status
- **QA**: InvokeOpenCode (qa), check for QA_PASS
- **Result**: ExecutorResultPacket → auto-ingest → update card + snapshots

Reference: `internal/orchestrate/loop.go`, `internal/control/orchestrate_once.go`

---

## Two Execution Paths (Current State)

SDP currently has two parallel execution paths that are not connected:

### Path A: Control Tower (new)
```
FeatureCard → DispatchCard → ExecutionPacket (on disk) → [GAP]
```
Capable of: intake, shaping, routing, packet building, result ingestion, board snapshots.
Missing: transport to executor.

### Path B: Orchestrate Loop (old)
```
Feature ID + Checkpoint → Hydrate → InvokeOpenCode → ContractGate → Review → PR → CI → QA → Done
```
Capable of: real execution, provenance, contract gates, full pipeline.
Missing: control tower integration, FeatureCard lifecycle, dispatch routing.

### Target State: Unified Pipeline
```
FeatureCard → DispatchCard → ExecutionPacket → Bridge → InvokeOpenCode → ContractGate → Review → PR → CI → QA → ExecutorResultPacket → auto-ingest → FeatureCard update
```

Both paths merged. Control tower owns lifecycle and routing. Orchestrate loop owns execution and verification. Bridge connects them.

---

## Research Alignment

SDP's design aligns with and extends industry-standard spec-driven development:

| Concept | Industry | SDP |
|---------|----------|-----|
| Spec as source of truth | GitHub Spec Kit | TaskContract |
| Constitution / project DNA | Spec Kit Constitution | project-registry + ADR |
| Specify → Clarify → Plan → Tasks → Implement | Spec Kit pipeline | FeatureCard lifecycle |
| Context engineering | Thoughtworks Building Block #5 | Hydrate + ContextPacket |
| Test-Driven AI (TDA) | Thoughtworks BMAD/TDA | ContractGate (quality gates) |
| Prompt provenance | Spec Kit (basic) | PromptProvenance + ContextSources + ArtifactHashChain |
| Agent opacity | A2A Protocol | ExecutorRole routing + ExecutionPacket contract |
| Stateful task lifecycle | A2A Tasks | FeatureCard FSM + Checkpoint FSM |
| Continuous discovery | Teresa Torres OST | ClarificationGate + feedback loop |

SDP goes beyond these by combining provenance + evidence + trace into a single pipeline with formal drift detection.

---

## Non-Goals

- Building a daemon/scheduler (orchestrate once = one step, not a loop)
- Replacing Beads as execution graph
- Building a web dashboard (CLI + board snapshots are sufficient)
- Replacing opencode as executor runtime
- Supporting non-opencode executors (future, not now)

---

## Related Documents

- `DISPATCH_BRIDGE_SPEC.md` — P0 specification for connecting control tower to OmO
- `CONTROL_TOWER_IMPLEMENTATION_ROADMAP.md` — updated roadmap with priorities
- `ORCHESTRATOR_BEADS_OPERATING_MODEL.md` — execution model (Beads = graph, SDP = protocol)
- `CONTROL_STORE_SKELETON.md` — current control store implementation
- `FEATURE_CARD_CONTRACT_WORKING_MODEL.md` — FeatureCard lifecycle
- `EXECUTION_HEARTBEAT_RUNTIME_RECONCILIATION_SPEC.md` — heartbeat spec
