# Orchestrator + Beads Operating Model

Status: working model
Date: 2026-03-22
Scope: Beads-first execution model for SDP with orchestrator-driven routing and status synthesis

## Goal

Make Beads the canonical execution graph and make the orchestrator the primary runtime that:
- selects work
- routes work to the right executor role
- ingests results
- advances state
- surfaces only meaningful exceptions to humans/admins

This is the model required if we want to stop asking manually for status updates.

---

## 1. Core principle

### Beads is the canonical execution graph
All real execution tasks should live in Beads.

That means:
- backlog lives in Beads
- dependencies live in Beads
- ready/blocked/in-progress work lives in Beads
- follow-up findings loop lives in Beads

### SDP is the protocol/process layer over that graph
SDP defines:
- trace and artifacts
- stage expectations
- routing logic
- handoff rules
- verification/review expectations
- escalation conditions

### Orchestrator is the runtime state mover
The orchestrator reads Beads, chooses what moves next, assigns execution, ingests results, and updates state.

### Board is a visualization surface
Humans/admins should see status through board/read models, not by interrogating agents manually.

---

## 2. Target execution flow

```text
FeatureCard / intake
  -> SDP trace starts
  -> Beads feature/task graph exists or is created
  -> orchestrator selects eligible work from Beads
  -> orchestrator routes work to executor role
  -> executor returns result packet
  -> orchestrator updates Beads + SDP artifacts
  -> board/snapshots reflect current state
```

### Human/admin involvement
Humans/admins are brought into the loop only when:
- clarification blocks safe progress
- approval is needed
- policy/risk threshold is crossed
- acceptance or product decision is required

---

## 3. Beads as canonical work substrate

### What belongs in Beads
- feature-level execution nodes
- child execution tasks
- dependencies
- findings/follow-up tasks
- execution-ready state
- blocked state
- ownership/claim state if applicable

### What does NOT replace Beads
- FeatureCard
- board snapshots
- raw chat context
- artifact docs

Those may reference Beads, but they do not replace it as execution truth.

---

## 4. Role of FeatureCard in this model

`FeatureCard` still matters, but primarily as:
- intake/shaping object
- human-facing project control object
- bridge into execution

Once execution begins, the canonical work graph is Beads.
FeatureCard remains useful for:
- trace from intake
- board-level visibility
- feedback/decision loops
- mapping human request to execution graph

---

## 5. Orchestrator runtime loop

The orchestrator loop should be thought of as:

### Step 1 — Select eligible work
Read Beads and find:
- ready work
- unclaimed work
- work whose dependencies are satisfied
- work requiring a routing decision

### Step 2 — Classify stage and task type
Determine:
- task class
- risk level
- repo context
- whether clarification/review/release sensitivity applies

### Step 3 — Route to executor role
Choose the correct executor role/agent.

### Step 4 — Build execution packet
Produce a narrow task packet with:
- beads task id
- parent feature id if relevant
- repo/project
- objective
- scope/constraints
- required outputs/artifacts
- routing metadata

### Step 5 — Dispatch execution
Launch the executor.

### Step 6 — Ingest result
Accept a result packet with:
- status
- summary
- artifacts produced
- follow-up findings
- whether more work/review/escalation is needed

### Step 7 — Update state
Write back to:
- Beads execution state
- SDP trace/artifact links
- board/snapshot projections

### Step 8 — Surface exceptions only when needed
If needed, create:
- author feedback request
- admin action request
- blocked state
- next-action recommendation

---

## 6. Routing matrix (initial skeleton)

This should later become a machine-readable matrix, but the canonical initial logic is:

### Intake / clarification / ambiguity
Route to:
- orchestrator clarification logic
- planner/analyst role

### Generic implementation work
Route to:
- OmO implementation role
- repo-local augmentation only when repo truth is needed

### Repo architecture questions
Route to:
- repo-local architecture agent
- e.g. `backend-core` in `opencode_server`

### Review
Route to:
- reviewer role
- repo-local review augmentation when needed

### Release / migration impact
Route to:
- release-check role
- plus review/verification if needed

### Docs / translation / local repo policy
Route to:
- repo-local docs/translator/triage/etc. where appropriate

### Escalation / uncertainty / risk threshold
Route to:
- human/admin feedback loop

---

## 7. Execution packet shape (initial skeleton)

A future machine-readable execution packet should minimally include:

```yaml
beads_task_id: <id>
parent_feature_id: <id or null>
project_id: <project>
target_repo: <repo>
executor_role: <role>
objective: <goal>
scope_in:
  - ...
scope_out:
  - ...
constraints:
  - ...
required_artifacts:
  - ...
required_checks:
  - ...
next_handoff_target: <role or stage>
```

This is what the orchestrator gives to an executor.

---

## 8. Result packet shape (initial skeleton)

A future result packet should minimally include:

```yaml
beads_task_id: <id>
executor_role: <role>
status: success|blocked|needs_review|needs_input|failed
summary: <short summary>
artifacts:
  - <artifact refs>
findings:
  - <follow-up findings or issues>
open_risks:
  - ...
recommended_next_step: <next>
```

This is what an executor gives back to the orchestrator.

---

## 9. Status synthesis model

We should not need manual status interrogation in the happy path.

Status should be synthesized from:
- Beads graph state
- active execution claims/runs
- linked artifacts
- feedback-needed state
- blocked reasons
- orchestrator next-action logic

### Human/admin views should answer
- what is executing now?
- what is blocked now?
- who is working on what?
- what needs human/admin response now?
- what can execute immediately?
- what is the next recommended system action?

---

## 10. Exception gates

The orchestrator should pause autonomy only when:
- product/intent ambiguity blocks safe progress
- approval is required
- policy/risk threshold is exceeded
- unresolved dependency blocks movement
- review/acceptance requires a human decision

Everything else should move automatically.

---

## 11. Immediate next implementation slice

The best next implementation slice is:

### routing matrix + execution packet skeleton

Why:
- Beads-first runtime needs explicit routing logic
- execution packets are required for consistent executor dispatch
- this is the minimum structure needed before a richer orchestrator loop can exist

### Concretely
Implement in `sdp_lab`:
- a first routing matrix skeleton
- a first execution packet struct/contract
- thin CLI/helper path to emit a packet for a Beads/control task

---

## 12. Short formula

- Beads = canonical execution graph
- SDP = process/routing/trace protocol
- Orchestrator = runtime state mover and allocator
- Executors = OmO + repo-local augmentation
- Board = human/admin visibility only

That is the operating model we want.
