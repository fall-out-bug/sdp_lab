---
title: F162 System-Level Prompt Bundle Packaging
status: design
owner: Andrei
feature: F162
beads: sdplab-wuaa
created: 2026-04-29
---

# F162 System-Level Prompt Bundle Packaging

## Problem

F141 shipped multi-harness install and adapter parity: SDP can install skills,
commands, and agents for Claude Code, Codex, OpenCode, and Cursor from
`sdp.manifest.yaml`. F162 expands the packaging target set to Claude Code,
Codex, OpenCode, Cursor, Pi, and Kilo.

That is necessary but not sufficient. It answers "what files get installed", not
"which exact prompt contract does this model/agent/task receive at startup".
Without a resolver, each harness sees too much context and the agent has to infer
which rules apply. That produces prompt soup: duplicated instructions,
cross-harness leakage, model-specific rules in universal files, and weak
evidence when a run fails.

## Product Principle

SDP should not ask agents to choose their own system prompt from a pile of
settings. The launcher/adapter layer must resolve the prompt bundle before the
agent starts.

Resolution flow:

```text
task intent
  -> harness: claude-code | codex | opencode | cursor | pi | kilo
  -> model/profile: frontier | mini | local-small | local-fast
  -> role: implementer | reviewer | planner | orchestrator | ...
  -> task_class: build | bugfix | review | design | ...
  -> prompt bundle id + hash
  -> harness-native entrypoint
```

Important caveat: repo files cannot overwrite the vendor's hidden system prompt.
The product can only install the highest available project/user-level entrypoint
for each harness. For practical use, that is still the right target: Claude Code
loads `CLAUDE.md` and `.claude/agents`, Codex loads project instructions such as
`AGENTS.md`/`.codex`, OpenCode loads `.opencode` agents and prompts, Cursor
loads `.cursorrules` and `.cursor/rules`, Kilo loads `AGENTS.md` plus
`kilo.jsonc`/`.kilocodemodes`/`.kilo/*`, and Pi must declare its exact runtime
entrypoint in a capability profile before dispatch claims support.

## Desired Packaging

### Canonical prompt source

Keep prompt logic in canonical source files, not harness output trees:

- `prompts/base/sdp-discipline.md`
- `prompts/base/ai-fluency-4d-intake.md`
- `prompts/harness/claude-code.md`
- `prompts/harness/codex.md`
- `prompts/harness/opencode.md`
- `prompts/harness/cursor.md`
- `prompts/harness/pi.md`
- `prompts/harness/kilo.md`
- `prompts/models/openai-coding.md`
- `prompts/models/anthropic-coding.md`
- `prompts/models/local-coder.md`
- `prompts/agents/*.md`
- `prompts/commands/*.md`
- `prompts/skills/*/SKILL.md`

Generated harness files are adapters only. They must not become the source of
truth.

### Manifest extension

Add `prompt_bundles` to `sdp.manifest.yaml`.

```yaml
prompt_bundles:
  - id: codex-openai-implementer-build
    harnesses: [codex]
    models: [gpt-5.4, gpt-5.4-mini]
    roles: [implementer]
    task_classes: [build, bugfix, refactor]
    sections:
      - prompts/base/sdp-discipline.md
      - prompts/base/ai-fluency-4d-intake.md
      - prompts/harness/codex.md
      - prompts/models/openai-coding.md
      - prompts/agents/implementer.md
      - prompts/commands/build.md
    output:
      path: .sdp/generated/prompts/codex/implementer-build.md
```

Empty selectors are forbidden. A bundle must say who it is for.

### Resolver contract

Introduce a resolver behind `sdp prompt resolve` and `sdp generate-adapters`:

```bash
sdp prompt resolve \
  --harness codex \
  --model gpt-5.4-mini \
  --role implementer \
  --task-class build
```

Output:

- bundle id
- bundle hash
- resolved output path
- ordered section list
- warnings/errors

Failure modes must be explicit:

- no matching bundle: error
- more than one equally specific bundle: error
- missing section file: error
- model-specific content in universal files: doctor error

### Harness entrypoints

Claude Code:

- `CLAUDE.md` stays thin and imports universal repo policy plus the generated
  Claude default bundle.
- `.claude/agents/<role>.md` uses the resolved role bundle.
- Claude-specific behavior stays in `prompts/harness/claude-code.md`.

Codex:

- Root `AGENTS.md` stays universal and small.
- `.codex/AGENTS.md` is the Codex-specific project entrypoint.
- `.codex/skills/*` remain generated adapters.
- For non-interactive role/task execution, `sdp dispatch` passes the resolved
  bundle path and records the bundle id/hash in evidence.

OpenCode:

- `.opencode/opencode.json` agent prompts point at resolved generated bundle
  files, not raw source fragments.
- Non-interactive runs use explicit agents, for example
  `opencode run --agent implementer`, to avoid the Sisyphus background-delegation
  deadlock documented in `docs/reference/harness-integration.md`.
- OpenCode-specific behavior stays in `prompts/harness/opencode.md`.

Cursor:

- `.cursorrules` stays a concise Cursor entrypoint for universal repo policy.
- `.cursor/rules/*.mdc` are generated from resolved bundles, not hand-maintained
  copies of source prompts.
- Cursor-specific behavior stays in `prompts/harness/cursor.md`.

Pi:

- Pi is included in the product scope as a dispatch target, but its exact shape
  must be defined by `00-162-06` before implementation claims parity.
- If Pi has a native project-instruction file, SDP generates that adapter.
- If Pi has no native project-instruction file, SDP supports Pi only through the
  SDP launcher path, where the resolved bundle is passed at invocation time.
- Pi-specific behavior stays in `prompts/harness/pi.md`.

Kilo:

- Kilo supports project instruction files and project-specific agent/mode
  configuration. F162 treats it as a first-class packaging target.
- Generated files may include `kilo.jsonc`, `.kilocodemodes`, `.kilo/agents/*.md`,
  and `.kilo/rules-{mode}/` entries as appropriate.
- Kilo-specific behavior stays in `prompts/harness/kilo.md`.

## 4D and SDP Discipline

AI Fluency 4D belongs in the bundle as an internal operating checklist, not as a
public response format.

- Delegation: what to do locally, what to ask, what to delegate, what cannot be
  delegated.
- Description: task contract, inputs, outputs, scope, acceptance criteria.
- Discernment: tests, evidence, source checks, review criteria.
- Diligence: safety, privacy, write boundaries, audit, ownership.

SDP discipline belongs in the same base bundle:

- no implementation without a sufficient task contract
- task contract depth scales with risk
- evidence before completion claims
- findings become beads/workstream follow-up, not forgotten chat text
- generated adapter drift is a failing condition

## Doctor Gates

Add prompt-specific checks under `sdp doctor prompts` or extend
`sdp doctor adapters`:

- every bundle resolves deterministically
- every generated entrypoint references the expected bundle hash
- universal files do not contain harness/model-specific rules
- harness files do not import another harness's section
- role prompts do not contain task-specific commands unless routed through a
  command/workflow section
- project overlays are explicit and validated
- generated files differ only when manifest/source sections change

## Non-Goals

- Do not rewrite all existing prompt content in this feature.
- Do not make the repo overwrite vendor-hidden system prompts.
- Do not add another free-form prompt directory.
- Do not make agents choose prompt bundles after startup.
- Do not treat an unsupported Pi or Kilo invocation shape as working; capability
  profiles and smoke tests are required before dispatch support is marked green.

## Workstreams

| WS | Bead | Purpose |
|---|---|---|
| 00-162-01 | sdplab-wuaa.1 | Bundle manifest schema and resolver contract |
| 00-162-02 | sdplab-wuaa.2 | Harness-native entrypoint generation |
| 00-162-03 | sdplab-wuaa.3 | Doctor gates and prompt isolation lint |
| 00-162-04 | sdplab-wuaa.4 | Runtime dispatch/model-profile integration |
| 00-162-05 | sdplab-wuaa.5 | Docs, onboarding, and F141 migration guide |
| 00-162-06 | sdplab-wuaa.6 | Cursor, Pi, and Kilo harness profiles and smoke contracts |

DAG:

```text
00-162-01 -> 00-162-06
00-162-01 -> 00-162-03
00-162-06 -> {00-162-02, 00-162-04}
{00-162-02, 00-162-03, 00-162-04, 00-162-06} -> 00-162-05
```

## Feature Acceptance

- `sdp.manifest.yaml` can declare prompt bundles for Claude Code, Codex,
  OpenCode, Cursor, Pi, and Kilo.
- `sdp prompt resolve` returns one deterministic bundle for a harness/model/role
  /task-class tuple.
- `sdp generate-adapters` emits harness-native project entrypoints that reference
  generated bundles for each supported harness profile.
- Direct harness startup loads only universal plus harness-appropriate policy.
- `sdp dispatch` can launch every green harness profile with the exact resolved
  bundle and records bundle id/hash in evidence.
- `sdp doctor` fails on bundle drift, missing sections, or cross-harness/model
  leakage.
- Downstream installation docs explain direct harness behavior, launcher behavior,
  update, rollback, and migration from F141.
