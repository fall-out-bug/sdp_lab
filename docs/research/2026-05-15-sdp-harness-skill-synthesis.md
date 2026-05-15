# SDP Harness And Skill Operating Strategy

Status: research synthesis
Date: 2026-05-15

This document synthesizes two inputs:

- `2026-05-15-agent-skill-operating-rules.md`
- `2026-05-15-harness-engineering-landscape.md`

It also incorporates independent cross-review from:

- `docs/reviews/2026-05-15-harness-cross-review-glm.md`
- `docs/reviews/2026-05-15-harness-cross-review-kimi.md`
- `docs/reviews/2026-05-15-harness-cross-review-qwen.md`

The cross-reviews are model-generated review evidence, not validation authority.
This synthesis treats them as adversarial input curated by the author. Any
recommendation below still needs normal repo adoption, implementation, and
measurement before it becomes policy.

## Executive Position

SDP should not copy `addyosmani/agent-skills` as a command model. SDP already
has a stronger control plane: manifest inventory, Beads, workstreams, evidence,
multi-harness parity, prompt-injection boundaries, and explicit degraded states.

The useful import is narrower:

- clearer skill anatomy;
- trigger and exclusion discipline;
- anti-rationalization tables;
- doubt-driven review for high-risk claims;
- source-driven checks for external APIs and harness behavior;
- explicit separation between prompt policy and runtime enforcement.

The 2026 harness shift is bigger than "better coding models". Coding agents are
becoming managed runtimes: sandboxing, network policy, tool permissions, model
routing, context lifecycle, telemetry, and evidence trails now matter as much as
prompt quality.

SDP should therefore optimize for governed harness execution, not for a larger
instruction pile.

## Non-Negotiable Design Lines

### 1. Skills Are Workflows, Not Libraries

Skills should say what to do, in order, with inputs, outputs, stop conditions,
and verification. They should not absorb module-local facts, provider secrets,
package APIs, or historical roadmap context.

Module facts stay in the nearest `AGENTS.md`. Stable reference belongs in
`docs/reference/`. Runtime inventory belongs in `sdp.manifest.yaml` or generated
adapter metadata.

### 2. Runtime Beats Prompt Policy

Prompt text can instruct. Runtime controls enforce.

For SDP, that means:

- declared writable roots;
- network allowlists or approval gates;
- per-agent tool permissions where the harness supports them;
- read-only reviewer roles by default;
- explicit gates for `git push`, publish, merge, external issue creation, and
  other identity-mediated actions;
- append-only evidence for model outputs, tool calls, failures, and degraded
  coverage.

### 3. Long-Horizon Models Do Not Remove Slice Discipline

GLM, Kimi, Qwen, DeepSeek, and GPT/Codex-class models now advertise stronger
long-horizon coding and tool use. SDP should treat those claims as routing
hypotheses, not as permission for unbounded execution.

Long-horizon capability is useful for executing many bounded slices in sequence.
It is not a reason to skip workstreams, checkpoints, review stops, or evidence.

### 4. Vendor Claims Are Quarantined

Model launch claims and benchmark tables can inform experiments. They cannot
drive merge, release, or trust-sensitive approval.

Every model-routing claim should carry:

- provider and endpoint;
- model id or snapshot when available;
- source class: official docs, vendor claim, third-party analysis, local eval;
- `routing_confidence`: `vendor_only`, `local_spike`, or
  `validated_on_sdp_tasks`.

All current model-family routing notes in the source landscape are
`vendor_only` unless separately measured on SDP tasks.

## Skill Authoring Standard

Every SDP skill should converge toward this shape:

```yaml
---
name: <kebab-case>
description: Does X. Use when Y. Do not use when Z if Z is a common false trigger.
version: <semver>
compatibility: [claude-code, opencode, cursor, codex, pi]
requires_cli: []
requires_mcp: []
tool_risk_classes: [perception, analysis]
runtime_requires:
  sandbox: read_only | workspace_write | full_access
  network: none | allowlisted | approval_required
  approvals: []
---
```

Required body sections:

- `Purpose`
- `Use When`
- `Do Not Use When`
- `Inputs`
- `Outputs`
- `Process`
- `Verification`
- `Degraded Evidence`

For high-risk skills, also add:

- `Common Rationalizations`
- `Runtime Preconditions`
- `Model Routing`
- `Stop Conditions`

Initial high-risk skills confirmed in the manifest:

| Skill | Canonical path | Status for adoption |
|---|---|---|
| `build` | `prompts/skills/build/SKILL.md` | exists; high-priority |
| `review` | `prompts/skills/review/SKILL.md` | exists; high-priority |
| `ship` | `prompts/skills/ship/SKILL.md` | exists; high-priority |
| `debug` | `prompts/skills/debug/SKILL.md` | exists; reconcile with `.agents/skills/debug.md` deprecation note |
| `delivery-loop` | `prompts/skills/delivery-loop/SKILL.md` | exists; high-priority |
| `spec-interrogate` | `prompts/skills/spec-interrogate/SKILL.md` | exists; already closest to target |

`compatibility` is a claim only after runtime dispatch evidence exists for that
harness. Until then it means intended portability and should be reported as
`not_assessed_runtime` per harness.

## Tool Risk Classes

Skills and harness adapters should classify tool use by side-effect class:

| Class | Meaning | Default policy |
|---|---|---|
| `perception` | read files, list paths, inspect logs, browse docs | allowed for most roles |
| `analysis` | local compute without writes or external side effects | allowed with evidence |
| `local_write` | edit files, generate artifacts, modify local state | implementer only; scoped |
| `external_write` | push, publish, create remote issue, update remote system | explicit gate |
| `irreversible` | merge, deploy, delete, spend money, rotate credentials | explicit human/workflow authorization |

This should live in manifest/runtime metadata, not prose alone. Pi may enforce
through packages or external wrappers; OpenCode can map to permission keys;
Codex can map to sandbox, approvals, and managed config. The manifest declares
requirements. The harness enforces them where it can.

If a harness cannot enforce `external_write` or `irreversible` gates at runtime,
the skill may not claim runtime support for that action class. It can only run
under an explicit workflow-level authorization path, and evidence must say
`not_assessed_runtime` or `manual_gate_only`.

## Degraded Evidence Taxonomy

Do not collapse missing evidence into green.

Canonical states:

| State | Meaning |
|---|---|
| `passed` | deterministic or reviewer evidence completed and supports the claim |
| `failed` | deterministic or reviewer evidence contradicts the claim |
| `not_assessed` | the plane was not run |
| `failed_provider` | provider returned an explicit error |
| `timeout` | run exceeded the bounded window |
| `empty_output` | run completed with no useful content |
| `off_task` | output did not address the requested plane |
| `unavailable_cli` | required tool was missing or could not run |
| `unverified_benchmark` | vendor/third-party claim not validated on SDP tasks |
| `not_assessed_runtime` | static files exist, but runtime behavior was not proven |

Evidence artifacts and review summaries should report these states directly.
`not_assessed` is not a weak pass. It is missing coverage.

Assignment rule: deterministic harness/tool output wins over model prose. If no
deterministic owner exists yet, the human/operator or orchestrator assigns the
state and records the reason. Conflicting evidence states are resolved toward
the more conservative degraded state until inspected.

## Model Routing Policy

Role-based routing is useful only if it is measured and auditable.

Recommended target roles:

| Role | Primary concern | Routing rule |
|---|---|---|
| scout | fast repo/context mapping | low-cost model acceptable after eval |
| planner | decomposition and acceptance criteria | stronger reasoning model |
| implementer | code changes | model with proven local gate performance |
| reviewer | independent critique | different family from implementer where possible |
| security | prompt-injection and side-effect risk | stronger model; strict evidence |
| synthesis | reconcile conflicting reviews | high-reasoning model |
| judge | final spec/review adjudication | different provider from critic |

Evidence for every model-generated artifact should include:

- `model_id`;
- provider or endpoint;
- harness name and version when available;
- prompt/context source;
- tool permissions enabled;
- degraded evidence state if coverage is partial.

Default review posture: for trust-sensitive work, at least one reviewer should
come from a different model family or provider than the implementer. All slots
from one vendor family are not independent enough for high-risk claims.

This rule requires implementer provenance to be known. If the implementer model
is unknown, the review must record `not_assessed_runtime` for model-family
independence rather than claiming diversity.

## Doubt Mechanisms: Decision Tree

Use the cheapest sufficient doubt mechanism:

| Situation | Mechanism |
|---|---|
| Behavioral code change | test-first or regression test evidence |
| Bug fix | prove failing case first, then prove pass |
| Prompt, skill, agent, review, eval, Beads, or model-routing change | bounded doubt cycle |
| Trust-sensitive workstream | doubt cycle plus selected multi-plane review |
| External API/framework/harness claim | source-driven verification from official docs |
| Release, publish, merge, or external side effect | runtime gate plus explicit authorization |
| One-line mechanical edit | standard local verification only |

Bounded doubt cycle:

1. `CLAIM`: state the claim and why it matters.
2. `EXTRACT`: isolate artifact and contract; strip author explanation.
3. `DOUBT`: send artifact and contract to an adversarial reviewer.
4. `RECONCILE`: classify findings as actionable, trade-off, misread, or noise.
5. `STOP`: stop after trivial findings, three cycles, or explicit owner override.

Do not combine every review plane with every doubt cycle by default. That turns
discipline into latency theater.

Budget: one doubt cycle should fit in a bounded review window and produce a
short finding list. If the reviewer returns a broad redesign request, split the
claim or reject the output as off-scope. Three cycles is a hard ceiling, not a
target.

## Multi-Plane Review Scope

The six-plane matrix is a library, not the default checklist.

Available planes:

- code correctness and maintainability;
- requirements vs implementation;
- evidence and tracing;
- security and prompt-injection boundary;
- docs/runtime truth;
- operations, CI, and release readiness;
- model/provider routing and provenance;
- tool-side-effect policy.

Default for ordinary work: one to two planes.

Default for trust-sensitive SDP work: two to three planes, selected by risk.

Mandatory planes:

- write-capable prompt/agent/skill changes: requirements, security/PI,
  evidence/tracing;
- release/publish/merge readiness: operations/CI/release, evidence/tracing,
  requirements;
- harness adapter support claims: docs/runtime truth, runtime dispatch evidence,
  tool-side-effect policy.

## Harness Adapter Parity

Static parity is not runtime support.

Static parity means:

- adapter file exists;
- manifest entry exists;
- generated docs or symlink points to the expected location.

Runtime dispatch evidence means:

- the harness loaded the intended skill/agent/command;
- the intended model was selected;
- permissions or sandbox settings matched the declaration;
- denied actions were actually denied or escalated;
- tool calls and failures were logged;
- output artifacts include evidence status.

Do not mark a harness as `supported` for a capability when only static parity is
verified. Use `not_assessed_runtime`.

## Source-Driven Development

When the task depends on current behavior of a hosted API, CLI, coding harness,
model, or framework, verify current official sources first.

Mandatory for:

- OpenAI/Codex API, Codex CLI, Codex app, or model behavior;
- OpenCode permissions, agent config, or delegation semantics;
- Pi packages, flow behavior, model routing, and context loading;
- external model capabilities, migrations, or deprecations;
- GitHub Actions or CI behavior when debugging CI.

Use current docs, release notes, model cards, source repositories, or installed
CLI versions. Mark conflicts and unverified claims explicitly.

## Adoption Plan

### Phase 1: One-Week Foundation

Goal: make authoring discipline enforceable without changing runtime.

- Update `docs/reference/skill-authoring.md` with required sections:
  `Do Not Use When`, `Verification`, `Degraded Evidence`.
- Add `Common Rationalizations` to `build` and `review`.
- Extend lint to warn on missing `Do Not Use When` and `Verification`.
- Add a small reference doc for tool risk classes and degraded evidence.

Acceptance gate:

- the current skill linter exists and runs, or the phase explicitly creates the
  minimal lint path first;
- `build` and `review` contain `Common Rationalizations`, `Verification`, and
  `Degraded Evidence`;
- the reference doc defines tool risk classes and degraded evidence states;
- all changed docs/skills pass the existing docs and skill lint checks;
- any runtime support claim added in this phase is marked `not_assessed_runtime`
  unless dispatch evidence exists.

### Phase 2: One-Sprint Harness Metadata

Goal: manifest can describe runtime needs even before every harness enforces
them.

- Draft manifest metadata for `tool_risk_classes` and `runtime_requires`.
- Map OpenCode permissions, Codex sandbox/approval concepts, and Pi wrapper or
  package enforcement points.
- Add evidence fields for `model_id`, provider, endpoint, harness version, and
  evidence status.
- Define `not_assessed_runtime` in review and adapter reports.

Acceptance gate:

- manifest schema impact is documented before fields are treated as canonical;
- at least one harness has a concrete mapping from manifest metadata to runtime
  behavior or explicit `manual_gate_only` fallback;
- evidence output can carry model/provider/harness metadata without manual
  copy-paste for the common path.

### Phase 3: Review And Doubt Integration

Goal: reduce skipped-risk rationalization without turning every task into a
research project.

- Add bounded doubt cycle to `spec-interrogate` and high-risk `review` modes.
- Add model/provider diversity rule for trust-sensitive review lanes.
- Add simplification as a review dimension, not as default feature cleanup.
- Add source-driven checks for OpenAI/Codex, OpenCode, Pi, and external model
  claims.

Acceptance gate:

- doubt cycle is limited to high-risk prompt/agent/skill/policy changes;
- ordinary daily work keeps a one- or two-plane review default;
- provider-diversity rules are advisory unless implementer provenance and model
  availability are known.

### Phase 4: Measurement

Goal: turn routing hypotheses into SDP evidence.

- Build a small SDP task suite: scout, spec critique, bug fix, review, harness
  adapter check.
- Run GPT/Codex, GLM, Kimi, Qwen, DeepSeek, and MiniMax lanes where available.
- Track cost, latency, correctness, evidence quality, and failure modes.
- Promote only measured routing rules from `vendor_only` to `local_spike` or
  `validated_on_sdp_tasks`.

Promotion criteria:

- `vendor_only`: source exists, no SDP run.
- `local_spike`: at least one successful bounded SDP task with recorded
  provider/model/harness metadata and failure notes.
- `validated_on_sdp_tasks`: repeated runs across at least three representative
  SDP task types with acceptable cost, latency, deterministic gate results, and
  evidence quality. The threshold must be recorded with the measurement packet;
  one lucky run is not validation.

## What Not To Adopt

- Do not replace SDP's intent surface with `/spec -> /plan -> /build -> /test ->
  /review -> /ship`.
- Do not invoke a skill at "1% chance"; it is too noisy for this repo.
- Do not replace SDP's multi-plane review with a generic three-persona review.
- Do not copy generic security checklists without SDP prompt-injection and
  evidence-boundary semantics.
- Do not let model benchmark rankings determine role routing without local SDP
  measurements.
- Do not treat swarms or long-horizon claims as architecture. Treat them as
  possible implementation mechanisms inside bounded flows.

## Immediate Recommendation

Adopt the operating discipline, not the external command stack.

The first concrete product move should be:

1. strengthen skill authoring and lint;
2. add degraded evidence and tool-risk vocabulary;
3. add anti-rationalization to `build` and `review`;
4. draft manifest runtime metadata;
5. measure model routing before changing default roles.

This is enough to improve UX and DX now without pretending the harness runtime
is already more enforceable than it is.

## Source Notes

Current public sources checked during this research include:

- OpenAI, GPT-5.5 model docs: `https://developers.openai.com/api/docs/models/gpt-5.5`
- OpenAI, running Codex safely: `https://openai.com/index/running-codex-safely/`
- OpenAI, Windows Codex sandbox: `https://openai.com/index/building-codex-windows-sandbox/`
- OpenCode agents and permissions: `https://opencode.ai/docs/agents/`
- Pi docs and packages: `https://pi.dev/docs/latest`,
  `https://pi.dev/packages/pi-agents`, `https://pi.dev/packages/pi-agent-flow`
- Z.AI GLM-5.1 docs: `https://docs.z.ai/guides/llm/glm-5.1`
- Kimi K2.6 official blog/help: `https://www.kimi.com/blog/kimi-k2-6`
- Qwen3.6 official repository: `https://github.com/QwenLM/Qwen3.6`
- Hugging Face DeepSeek-V4 analysis:
  `https://huggingface.co/blog/deepseekv4`

External model claims in these sources remain routing hypotheses until validated
on SDP tasks.
