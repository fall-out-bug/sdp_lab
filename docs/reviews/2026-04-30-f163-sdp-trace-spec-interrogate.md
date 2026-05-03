# F163 Dogfood: sdp-trace SpecKit Interrogation

Date: 2026-04-30
Feature: F163
Workstream: 00-163-01
Bead: sdplab-n7a9.1

## Target

Repository: `/Users/fall_out_bug/projects/vibe_coding/sdp-trace`

Artifacts reviewed:

- `specs/001-sdp-trace-time-series-evidence-substrate/spec.md`
- `specs/001-sdp-trace-time-series-evidence-substrate/plan.md`
- `specs/001-sdp-trace-time-series-evidence-substrate/tasks.md`
- `specs/001-sdp-trace-time-series-evidence-substrate/research.md`
- `specs/001-sdp-trace-time-series-evidence-substrate/data-model.md`
- `specs/001-sdp-trace-time-series-evidence-substrate/quickstart.md`
- `specs/001-sdp-trace-time-series-evidence-substrate/contracts/sdp-trace-sdp-gate-boundary.md`

## Invocation

Critic provider: `zai/glm-5.1`

Clean-context flags:

```bash
pi --provider zai --model glm-5.1 --no-tools --no-context-files --no-session -p ...
```

Rubrics:

- problem and goal
- system boundary and non-goals
- roles and actors
- primary scenarios
- assumptions and dependencies
- edge cases and failure behavior
- security and access
- observability and metrics
- testability and acceptance
- rollout, migration, and backward compatibility
- open questions and risks

## Round 1: GLM Critic

Verdict: `REWORK`

Reason: the critic returned 23 rubric-scoped questions with 4 blocking, 14 major, and 5 minor items. The output obeyed the protocol: JSON object, questions only, no patches, no rewritten spec, no implementation plan.

Blocking questions:

1. `Q3` — `data-model.md` has `Evidence Event.strength`, but the spec does not say who assigns evidence strength or whether that crosses the `sdp-trace` / `sdp-gate` policy boundary.
2. `Q9` — JSON Schema draft/version is selected after schema authoring tasks, so five schema tasks can be written against the wrong validator contract.
3. `Q13` — evidence/provenance references can point to prompts, command output, or context containing secrets/PII, but the self-trace plan commits examples without a content-review/scanning rule.
4. `Q19` — `schema/trace.schema.json` may be modified or replaced, but the specs do not state whether existing consumers require backward compatibility.

Major clusters:

- movement/degradation representation is not structurally pinned down enough for independent consumers
- external verdicts need a stronger structural distinction from internal policy judgments
- evidence producers and actor type constraints are underspecified
- OpenCode + MiniMax/Kimi/GLM pilot evidence may collapse into documentation unless a live artifact requirement is stated
- `pending` evidence status creates a third state not covered by `evidence` vs `not_assessed`
- example-to-schema validation is not explicitly in scope despite success criteria implying it
- Kotlin+Bazel "gap closed" conflicts with a planned `not_assessed` placeholder
- provenance hash-chain fields imply integrity semantics without defining algorithm or verification

## Provider Rotation Test

Two additional clean-context critic passes used the same SpecKit bundle and no previous critique context.

| Provider | Result | Blocking | Major | Minor | Protocol Notes |
|---|---:|---:|---:|---:|---|
| `zai/glm-5.1` | `REWORK` | 4 | 14 | 5 | Valid JSON in fenced block; questions only |
| `minimax/MiniMax-M2.7` | `REWORK` | 3 | 10 | 2 | Valid JSON; strongest on product boundary and validation timing |
| `kimi-coding/k2p6` | `REWORK` | 2 | 5 | 6 | Valid JSON in fenced block; strongest on integrity, migration, and negative testing |

Provider comparison:

- GLM produced the broadest rubric coverage and the highest number of actionable questions.
- MiniMax produced the clearest challenge to the core product promise: answering the CTO degradation question without crossing into `sdp-gate` verdict policy.
- Kimi surfaced the strongest missing trust-model issue: provenance hash fields imply integrity, but the specs do not define trust anchors, signing, write authorization, or tamper detection.

Converged blockers across providers:

1. `sdp-trace` must define how it answers the CTO's degradation question as evidence/movement data without producing a gate verdict.
2. External verdicts, evidence strength, and policy-adjacent fields need structural separation from native trace observations.
3. JSON Schema validator and schema draft/version must be selected before schema authoring and example validation claims.
4. Evidence and provenance references need a security/integrity model before self-trace or pilot artifacts are committed.
5. Schema versioning and backward compatibility must be explicit before `sdp-gate` inherits the contracts.

## Assessment

The updated `spec-interrogate` concept is useful. A clean-context critic found real implementation blockers that were not obvious from author-side discussion. The current `sdp-trace` specs are good enough to discuss but not yet good enough to implement without hidden assumptions.

The next correct action is not to add code. The author should revise the `sdp-trace` SpecKit artifacts for the converged blockers, add explicit resolution notes for major items, and then run a judge pass with a provider different from the latest critic provider.

## Follow-Up Judge Pass

The `sdp-trace` SpecKit artifacts were revised on 2026-04-30. The author added resolution notes and a committed judge result under the target SpecKit package:

- `/Users/fall_out_bug/projects/vibe_coding/sdp-trace/specs/001-sdp-trace-time-series-evidence-substrate/socratic-resolution-notes.md`
- `/Users/fall_out_bug/projects/vibe_coding/sdp-trace/specs/001-sdp-trace-time-series-evidence-substrate/socratic-judge-result.json`

Judge provider: `minimax/MiniMax-M2.7`

Judge verdict: `PASS`

Judge assessment:

- all 5 converged blockers resolved
- no new contradictions
- no scope creep

## Local Checks

- `go test ./internal/workstream -count=1` — passed, 47 tests.
- `go run ./cmd/sdp-protocol-check --lint-skills --format json` — exited 0 with four pre-existing warnings outside this change.
- `jq empty schema/*.json` — passed.
- `jq empty docs/reviews/2026-04-30-f163-sdp-trace-spec-interrogate-evidence.json` — passed.
- `go run ./cmd/sdp doctor backlog` — passed, 0 findings.
- `sdp-trace`: `jq empty schema/*.json specs/001-sdp-trace-time-series-evidence-substrate/socratic-judge-result.json` — passed.
- `sdp-trace`: `npx --yes ajv-cli@5.0.0 validate --spec=draft2020 -s schema/trace.schema.json -d examples/github-speckit/trace.json` — passed.
