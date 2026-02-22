# Evaluator Swarm Runtime Orchestration

This document defines the first runtime increment for persona execution orchestration in `sdp_dev-hx0.1.2`.

## Runtime Deliverables

- Persona execution packet assembly for a concrete issue run.
- Deterministic persona score ingestion.
- Consensus and dissent report assembly for publish-handoff phases.

## Persona Execution Packet Contract

Implemented in `internal/evaluator/swarm_runtime.go` via `BuildPersonaExecutionPacket(...)`.

Packet fields:

- `ContractVersion`: `deep-thinking-evaluator-runtime/v1`
- `IssueID`: target issue under evaluation.
- `Cadence`: carried from deep-thinking swarm plan.
- `PhaseOrder`: deterministic phase list for role execution sequencing.
- `Units`: one execution unit per persona role, including:
  - persona ID and decision lens
  - primary question
  - required evidence list
  - escalation target
  - phase focus and entry gate signals

## Score + Report Assembly

Implemented in `AssembleSwarmScoreReport(...)`.

Behavior:

- Accepts persona scores and maps them to known packet personas.
- Clamps score inputs into `0..100`.
- Ignores duplicate persona responses after keeping the highest sorted score.
- Tracks unknown persona IDs for evidence hygiene.
- Produces:
  - response coverage (`responded`, `missing`)
  - aggregate metrics (`average`, `min`, `max`)
  - dissent roster (`score < 70`)
  - consensus status (>= 80 percent of configured personas and no missing responders)
  - ranked recommendation list for handoff

## Validation

- `internal/evaluator/swarm_runtime_test.go` verifies:
  - deterministic packet generation
  - contract validation failures (missing issue ID, incomplete plan)
  - complete 5-persona score assembly with 4-of-5 consensus
  - missing/unknown persona handling and score clamp behavior

## Follow-on Hooks

- `sdp_dev-hx0.1.3` consumes packet and score-report structures via `internal/evaluator/audit_protocol.go` and documents the periodic checkpoint/escalation contract in `docs/EVALUATOR_PERIODIC_COMPONENT_AUDIT_PROTOCOL.md`.
- Later rubric work can score by artifact class while preserving this contract as the orchestration envelope.
