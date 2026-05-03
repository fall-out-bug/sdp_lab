# F165 Design Spec Interrogate

Date: 2026-05-03
Artifact: `docs/plans/2026-05-03-f165-indirect-prompt-injection-through-task-data-design.md`
Feature: F165
Mode: Socratic review
Verdict: PASS

## Provider Coverage

| Round | Role | Provider | Result |
|---|---|---|---|
| 0 | critic | `zai/glm-5.1` | provider/run failure; no stdout captured from first invocation form |
| 1 | critic | `minimax/MiniMax-M2.7` | 12 questions, 4 blocking |
| 1 | judge | `kimi-coding/k2p6` | PASS after revision |
| 2 | critic | `zai/glm-5.1` | 15 questions, 7 blocking |
| 2 | judge | `minimax/MiniMax-M2.7` | REWORK; 14 resolved, 1 partially resolved |
| 2b | point judge | `kimi-coding/k2p6` | PASS for remaining mapping question |

## Critic Findings And Resolution

### MiniMax Round 1

MiniMax found that the initial design was directionally correct but not
operational enough. Blocking issues:

- root cause mixed model behavior, trust boundary failure, and missing state gates
- strict parsing and parser TCB were undefined
- pass/fail criteria were qualitative
- naive vulnerable runner containment was underspecified

Resolution in the design:

- reframed the problem as data-plane to control-plane boundary failure
- split mechanisms into model susceptibility, data/control separation, and state gates
- defined strict parser as TCB with fail-closed semantics
- added measurable failure criteria per vector
- added unsafe demo containment requirements
- added residual-risk/unsupported-case reporting
- split evidence/finding poisoning from optional handoff poisoning

Kimi judged all 12 MiniMax questions resolved with no new contradictions.

### GLM Round 2

GLM accepted the direction but found remaining execution ambiguities. Blocking
issues:

- trusted state source was non-circular only by assertion
- parser rejection semantics were not precise
- Normalize stage did not distinguish stripping from classification
- naive runner determinism versus live-model behavior was ambiguous
- workstream decomposition was not specified
- the report itself could become a downstream injection vector
- challenge mapping lacked an evidentiary standard

Resolution in the design:

- trusted state is now a pre-ingestion snapshot
- parser rejection halts as `parse_error`; malformed optional narrative remains untrusted
- conflict checks live in `Validate`, not `Parse`
- naive runner is a scripted oracle using only in-memory mock Beads/tool state
- Normalize rules are explicit for zero-width characters, HTML comments, link targets, and ordinary prose
- `blocked_reason` and residual-risk categories are closed sets
- report automation may consume typed fields only; narrative is not authority
- workstream decomposition is stage-first, not one pipeline per vector
- challenge mapping is justified by trust-boundary isomorphism

MiniMax judged 14 of 15 GLM questions resolved and left only the challenge
mapping standard partially resolved. Kimi then judged the added mapping standard
PASS.

## Final Design State

The design is now clear enough to move to feature planning. Required next step:

1. create F165 epic,
2. create aggregate and leaf workstreams,
3. create Beads issues and mapping,
4. run Socratic review again over the workstream/Beads plan before implementation.

## Residual Risks

- Handoff poisoning is optional. If not implemented, it must appear as
  `unsupported_surface` in the final report.
- MCP resource injection remains covered by F164 and is not primary F165 scope.
- Live model behavior is advisory only; deterministic pass/fail is based on
  scripted fixtures, trusted-state snapshots, and validation traces.

