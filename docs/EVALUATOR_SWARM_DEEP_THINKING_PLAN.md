# Evaluator Swarm Deep-Thinking Plan

This design defines the deep-thinking evaluation loop for `sdp_dev-hx0.1` and establishes persona-role collaboration rules before implementation and rubric tasks proceed.

## Objective

- Run recurring evaluator audits that challenge framework decisions from multiple lenses.
- Require explicit evidence and cross-role disagreement handling before recommendations become backlog items.
- Keep the workflow deterministic so trial runs can be regression-tested.

## Cycle Cadence and Entry Gate

- Cadence: `weekly-or-change-triggered`.
- Entry gate reuses baseline intake checks from `internal/evaluator/intake.go`:
  - `issue-selected`
  - `dependencies-clear`
  - `scope-baseline-defined`
  - `gate-command-declared`
  - `callback-contract-available`

Evaluator cycles do not start until all entry signals pass.

## Persona Roles

| Persona ID | Decision lens | Primary question | Required evidence | Escalation target |
| --- | --- | --- | --- | --- |
| `systems-architect` | Boundaries, coupling, maintainability | Does this preserve architecture integrity under roadmap growth? | `boundary-map`, `dependency-graph`, `upgrade-path` | `product-strategist` |
| `sre` | Reliability, operability, failure isolation | Can this survive production-like stress without paging instability? | `slo-impact`, `runbook-delta`, `rollback-plan` | `systems-architect` |
| `security-reviewer` | Abuse resistance and policy alignment | What is the worst abuse path and is it contained? | `threat-model`, `secret-handling-proof`, `policy-check-results` | `sre` |
| `dx-expert` | Maintainability ergonomics and flow clarity | Can maintainers execute and verify this without hidden context? | `contract-examples`, `cli-runbook`, `verification-latency` | `systems-architect` |
| `product-strategist` | Outcome impact and sequencing | Does this maximize near-term user value per engineering cost? | `outcome-hypothesis`, `adoption-signal`, `opportunity-cost` | `systems-architect` |

## Deep-Thinking Evaluation Phases

| Phase ID | Objective | Required signals | Output artifacts |
| --- | --- | --- | --- |
| `framing` | Restate issue intent, constraints, and success criteria. | `intent-brief`, `dependency-map` | `evaluation-frame`, `risk-register` |
| `persona-analysis` | Run role-specific audits in parallel. | `evaluation-frame`, `artifact-evidence-bundle` | `persona-findings` |
| `adversarial-review` | Run challenge rounds to expose contradictions. | `persona-findings` | `conflict-matrix`, `challenge-transcript` |
| `consensus-synthesis` | Rank recommendations and capture dissent. | `conflict-matrix`, `challenge-transcript` | `prioritized-recommendations`, `dissent-log` |
| `publish-handoff` | Convert accepted actions into implementable backlog and publish evidence. | `prioritized-recommendations` | `implementation-brief`, `trace-link` |

## Collaboration Protocol

- Conflict rule: `adversarial-double-pass` (each persona critiques once, then responds to strongest challenge).
- Consensus threshold: `4-of-5-persona-majority`.
- Fallback: `escalate-to-issue-owner-with-dissent-log` when no majority emerges.

## Contract and Test Hooks

- Contract helper: `internal/evaluator/swarm_plan.go`.
- Core API:
  - `DefaultDeepThinkingSwarmPlan()` for deterministic contract materialization.
  - `EvaluateSwarmPlanReadiness(...)` for gate checks across trigger signals, role coverage, and phase coverage.
- Regression tests: `internal/evaluator/swarm_plan_test.go`.

## Acceptance Mapping for `sdp_dev-hx0.1.1`

- Deep-thinking evaluator cycle and persona collaboration flow are defined.
- Roles include systems architect, SRE, security reviewer, DX expert, and product strategist.
- Deterministic contract helper and tests provide implementation-ready baseline for downstream persona library and protocol tasks.

## Runtime Bridge

- Runtime orchestration increment for `sdp_dev-hx0.1.2` is documented in `docs/EVALUATOR_SWARM_RUNTIME_ORCHESTRATION.md`.
- Implementation hooks are available in `internal/evaluator/swarm_runtime.go` for persona execution packets and score/report assembly.
