# Control Tower Implementation Roadmap

Status: working roadmap
Date: 2026-03-23 (updated)
Owner: Клавдий (orchestration) + OmO (implementation)
Canon: `SDP_SPEC_DRIVEN_PIPELINE_CANON.md`

## Goal

Полный spec-driven pipeline от intent до deploy:
- Constitution constrains everything
- Intent → Specify → Contract → Dispatch → Execute → Verify → Result
- Provenance + Evidence + Trace на каждом шаге
- Beads = execution graph, Board = visualization surface

---

## Current baseline

**Completed phases:**
- ✅ Phase 0: Constitution foundation (project-registry, ADR, persona registry)
- ✅ Phase 1: Control Store MVP (FeatureCard lifecycle, snapshots, doctor, attention)
- ✅ Phase 2: Beads Bridge (card-execute, Beads linkage, dispatch packet)
- ✅ Phase 3: Orchestrator Loop Integration (orchestrate once, dispatch next, result ingest, feedback/resume)
- ✅ Phase 4: Human/Admin Surface (board views, executive summary, CLI board/attention/doctor)
- ✅ Contract layer (TaskContract, ClarificationGate, EnforceContractGate, drift detection)
- ✅ Provenance layer (prompt hash, context sources, artifact hash chain)
- ✅ Old orchestrate loop (Hydrate, InvokeOpenCode, ContractGate, Review, CI, QA, PR)

**Critical gap identified (2026-03-23):**
Control tower and orchestrate loop are disconnected. See `DISPATCH_BRIDGE_SPEC.md`.

---

## Completed Phases (Archive)

### Phase 1 — Control Store MVP ✅

### Goal
Turn the current skeleton into a usable local control-state engine.

### Scope
1. Add card update/load helpers beyond current lifecycle basics
2. Add stronger ready-gate and transition validation
3. Add snapshot rebuild convenience helpers
4. Add richer status rendering for CLI use
5. Add explicit intake artifact writing/updating helpers

### Deliverable
A reliable local control-state layer that can:
- persist FeatureCards
- derive project/portfolio snapshots
- expose human/admin-relevant queues

### Exit criteria
- can create, clarify, request input, ready, and park cards reliably
- snapshots reflect queues correctly
- CLI output is usable for orchestration and human visibility

---

### Phase 2 — Beads Bridge ✅

### Goal
Bridge `ready` FeatureCards into Beads execution objects.

### Scope
1. `card-execute` / `card-bridge` action
2. Create feature-level Beads issue from a ready card
3. Write back `linked_beads_ids`
4. Seed labels/description/spec references from card data
5. Optionally attach initial workstream hints or placeholders

### Deliverable
First working path:
`FeatureCard (ready)` -> `Beads feature issue`

### Exit criteria
- ready card can create linked Beads issue
- control store reflects the linkage
- snapshot can show execution linkage

---

### Phase 3 — Orchestrator Loop Integration ✅

### Goal
Make orchestration actions operate on the control store as a first-class system.

### Scope
1. Orchestrator helpers over card lifecycle
2. Recommendation engine for next action
3. Feedback packet generation for author/admin
4. Resume flow after feedback/decision arrival
5. Blocked / waiting-on-human / ready-to-execute prioritization

### Deliverable
The orchestrator can autonomously move cards and surface only meaningful exceptions.

### Exit criteria
- orchestrator can own state transitions by default
- author/admin only get targeted feedback requests or updates
- feedback answers can resume flow automatically

---

### Phase 4 — Human/Admin Surface ✅

### Goal
Expose control state as a useful board/dashboard without making it the system of record.

### Scope
1. CLI-friendly board/status views
2. thin board rendering over snapshots
3. project board view
4. portfolio control tower view
5. waiting_on_human / blocked / ready_to_execute visibility
6. show active agents and human/admin requests clearly

### Deliverable
A usable visualization surface for humans/admins.

### Exit criteria
- portfolio and project views are readable
- can see what is happening, which agent is doing what, and what is needed from humans/admins

---

## Next Phases (Prioritized)

Priorities based on critical gap analysis in `SDP_SPEC_DRIVEN_PIPELINE_CANON.md`.

### Phase 5 (P0) — Dispatch Bridge

**Goal**: Connect control tower to OmO executor. Close the single critical gap.

**Spec**: `DISPATCH_BRIDGE_SPEC.md`

**Scope**:
1. `internal/executor/bridge.go` — ExecutorBridge with DispatchAndRun
2. Read ExecutionPacket → build ContextPacket → prompt → InvokeOpenCode
3. Write executor session metadata to FeatureCard (runtime state)
4. Translate opencode output → ExecutorResultPacket → executor-results/
5. `sdp dispatch next --execute` CLI command
6. Auto-ingest via existing OrchestrateOnce flow
7. Prompt provenance on every dispatch

**Exit criteria**:
- `sdp dispatch next --execute` runs end-to-end
- FeatureCard updated with session metadata
- Result auto-ingested
- Prompt provenance written
- Tests pass, no regression

---

### Phase 6 (P1) — Auto-Generate TaskContract from FeatureCard

**Goal**: TaskContract derived automatically from FeatureCard on ready gate.

**Scope**:
1. On `MarkReady`: generate TaskContract from card fields
2. normalized_intent → objective
3. acceptance_shape → acceptance_criteria (structured)
4. scope_out → required_evidence
5. risk_level → constraints
6. Write to `.sdp/contracts/<card-id>.json`

**Exit criteria**:
- Every ready card has an auto-generated TaskContract
- Contract gates work on auto-generated contracts

---

### Phase 7 (P2) — Unify Provenance

**Goal**: Full provenance chain from intent to artifact in the unified pipeline.

**Scope**:
1. Extend prompt provenance to dispatch bridge path
2. Chain: contract hash → packet hash → prompt hash → artifact hash
3. Provenance query CLI (`sdp provenance show --card <id>`)

**Exit criteria**:
- Every dispatched card has complete provenance chain
- Provenance queryable

---

### Phase 8 (P3) — A2A Interface

**Goal**: Expose SDP as A2A-compliant agent for external orchestration (OpenClaw, others).

**Scope**:
1. A2A HTTP server wrapping DispatchBridge
2. Agent Card at `/.well-known/agent.json`
3. Operations: tasks/send, tasks/get, tasks/list, tasks/cancel
4. Streaming support for long-running tasks

**Reference**: A2A Protocol v1.0.0 (https://a2a-protocol.org)

---

### Phase 9 (P4) — Constitution as Explicit Layer

**Goal**: Formalize project-registry + ADR into explicit Constitution document.

**Scope**:
1. Constitution template: vision, principles, non-negotiable constraints
2. Per-project Constitution derived from project-registry
3. Architecture fit check as part of ClarificationGate

---

### Phase 10 (P5) — Advanced Execution

**Goal**: Mature the unified pipeline.

**Scope**:
1. Richer Beads decomposition from TaskContract
2. Findings loop back to cards
3. Release/review gating
4. Multi-repo parallel execution

---

## Immediate Next Step

**Architectural pivot before more glue code: adopt Beads-first control tower roadmap.**

See:
- `BEADS_FIRST_CONTROL_TOWER_ROADMAP.md`
- `DISPATCH_BRIDGE_SPEC.md`

Dispatch bridge work is important, but the next correct move is to stop treating control-state files as a parallel truth system and instead formalize:
1. Beads as operational source of truth
2. FeatureCard as semantic artifact / projection
3. repository boundary inside `internal/control`
4. exact mapping for contracts, evidence, provenance, and gates

Only after that should the remaining bridge and cutover work continue.
