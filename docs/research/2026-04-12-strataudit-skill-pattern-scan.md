# StratAudit Skill Pattern Scan

Date: 2026-04-12
Scope: reusable skill patterns for a future `StratAudit` skill surface that can run in any harness, with OpenRouter as an optional model amplifier rather than a hard dependency.

## Why this scan exists

`StratAudit` is moving from "CLI module with an LLM backend" toward "portable audit capability with a usable skill surface". That shift changes the problem.

The hard part is no longer only extraction and traceability. The hard part is teaching a harness or agent:

- when `StratAudit` should trigger,
- what evidence bar it must enforce,
- how it should select runtimes,
- what outputs it must produce,
- and what it must refuse to claim.

This scan looks for patterns worth stealing from existing skill ecosystems, then reduces them into a concrete `StratAudit` design direction.

## Repositories scanned

High-signal:

- `/Users/fall_out_bug/projects/vibe_coding/agent-skills`
- `/Users/fall_out_bug/projects/vibe_coding/hyperpowers`
- `/Users/fall_out_bug/projects/vibe_coding/superpowers`
- `/Users/fall_out_bug/projects/vibe_coding/paperclip`
- `/Users/fall_out_bug/projects/vibe_coding/gastown`
- `/Users/fall_out_bug/projects/vibe_coding/gstack`
- `/Users/fall_out_bug/projects/vibe_coding/oh-my-openagent`
- `/Users/fall_out_bug/projects/vibe_coding/deer-flow`
- `/Users/fall_out_bug/projects/vibe_coding/BMAD-METHOD`
- `/Users/fall_out_bug/projects/vibe_coding/JavaGuide`

Lower-signal or mostly peripheral for this task:

- `/Users/fall_out_bug/projects/vibe_coding/graphify`
- `/Users/fall_out_bug/projects/vibe_coding/awesome-agent-skills`
- `/Users/fall_out_bug/projects/vibe_coding/AI-Research-SKILLs`
- `/Users/fall_out_bug/projects/vibe_coding/sgr-agent-core`
- `/Users/fall_out_bug/projects/vibe_coding/claude-code-main`

## Executive Summary

The best external patterns converge on five rules:

1. `Evidence before claims` must be a hard gate, not a suggestion.
2. `Source hierarchy` must be explicit: primary evidence beats model synthesis.
3. `Skill metadata` must describe trigger conditions, not internal workflow.
4. `Runtime selection` must be capability-aware and provider-neutral.
5. `Outputs` must separate conclusions, evidence, assumptions, and non-goals.

The strongest design implication for `StratAudit` is blunt:

`StratAudit` should not be one monolithic "analyze strategy" skill.

It should be a portable skill surface with a narrow trigger contract and a strict analyst workflow:

`scope -> ingest reality -> verify evidence -> trace -> report -> flag uncertainty`

If evidence is weak, the skill must fail closed. It must never quietly turn "insufficient provenance" into polished executive prose.

## Current Gap in SDP

In the current `main` checkout, `StratAudit` is documented in `docs/STRATAUDIT.md`, but there is no stable, discoverable skill surface in `skills/` and no usable public prompt surface under `sdp/` in this checkout. That matters.

Today the module is still optimized around the pipeline internals:

- ingest,
- extract,
- link,
- analyze,
- report.

But a reusable skill needs a different public contract:

- trigger conditions,
- evidence policy,
- runtime policy,
- output policy,
- refusal policy.

This scan is about that missing public contract.

## Adopt

These patterns should transfer almost directly.

### 1. Evidence-before-claims gate

Source patterns:

- `hyperpowers/skills/verification-before-completion/SKILL.md`
- `superpowers/skills/verification-before-completion/SKILL.md`

Why it matters:

`StratAudit` currently has the same failure mode those skills are trying to prevent: a system sounding finished before it has proof. In strategy traceability this is worse than in coding, because executives will trust the shape of the report.

What to adopt:

- hard rule: no coverage claim without verified denominator and numerator,
- no trace claim without inspectable evidence,
- no finding severity without backing evidence bundle,
- no "aligned / not aligned" summary if provenance grade is below threshold.

Direct implication:

The future `StratAudit` skill needs a visible `verification gate` section, not just implementation-time checks in Go.

### 2. Source-driven / authority hierarchy

Source pattern:

- `agent-skills/skills/source-driven-development/SKILL.md`

Why it matters:

This is the cleanest reusable pattern for "the model is not the source of truth". It maps almost perfectly onto strategy auditing.

What to adopt:

- explicit authority ladder,
- explicit conflict surfacing,
- explicit "unverified" state,
- visible source citation discipline.

Recommended `StratAudit` authority hierarchy:

1. exact document span or quote,
2. document metadata and section context,
3. verified derived entity/trace,
4. model-assisted inference,
5. analyst interpretation.

Rules:

- lower layers can summarize higher layers,
- lower layers cannot override higher layers,
- if quote and model disagree, the quote wins,
- if no quote exists, the report must say `inferred`, not `verified`.

### 3. Trigger metadata describes when to use, not how it works

Source pattern:

- `superpowers/skills/writing-skills/SKILL.md`
- `skill-creator/SKILL.md`

Why it matters:

This is one of the easiest places to get `StratAudit` wrong. If the skill description explains the workflow, agents will short-circuit and imitate the summary instead of reading the actual guardrails.

What to adopt:

- description focused on user signals and trigger situations,
- rich keyword coverage,
- progressive disclosure,
- slim main skill body with deeper references loaded only when needed.

Direct implication:

The future `StratAudit` skill description should look like:

`Use when auditing strategic documents for evidence-backed alignment, traceability gaps, cross-level coverage, or source-grounded initiative mapping across a messy document corpus.`

It should not say:

`Ingests documents, extracts entities, links them, and produces HTML reports.`

That is internal mechanics, not trigger logic.

### 4. Progressive disclosure

Source pattern:

- `skill-creator/SKILL.md`
- `JavaGuide/docs/ai/agent/skills.md`

Why it matters:

`StratAudit` has too much detail to dump into one prompt. If the skill becomes a giant instruction slab, agents will skip or blur the important parts.

What to adopt:

- short `SKILL.md` with routing and hard rules,
- separate references for:
  - evidence policy,
  - runtime/provider policy,
  - report schema,
  - corpus-quality taxonomy,
  - investigation playbooks.

Direct implication:

The top-level skill should teach behavior and routing. It should not contain the entire report spec.

### 5. Research and synthesis checklist

Source pattern:

- `deer-flow/skills/public/deep-research/SKILL.md`

Why it matters:

A strategy audit is often partly a research task: the corpus is incomplete, redundant, multilingual, noisy, or contradictory. DeerFlow's "broad exploration -> deep dive -> diversity -> synthesis check" pattern is highly portable.

What to adopt:

- explicit exploration pass before conclusions,
- diversity check across doc types and levels,
- synthesis checklist before final report,
- requirement to inspect both confirming and disconfirming evidence.

Direct implication:

The skill should force a pre-report checklist:

- did we inspect every configured level,
- did we verify document coverage,
- did we inspect both strong traces and missing traces,
- did we separate verified evidence from inferred links,
- did we quantify corpus noise.

## Adapt

These patterns are good, but need narrowing.

### 1. Runtime and model-family matching

Source pattern:

- `oh-my-openagent/docs/guide/agent-model-matching.md`

Why it matters:

This project already wants `OpenRouter` as an amplifier, not a monopoly. The right takeaway is not "copy their fallback chains". The right takeaway is that runtime policy must be explicit and capability-based.

What to adapt:

- choose runtime by required capability, not brand loyalty,
- separate text extraction reasoning, evidence verification, and synthesis needs,
- define a fallback order per phase,
- allow harness-native models when they meet the bar.

Recommended `StratAudit` runtime policy:

- `host-native` first if the harness offers a model with sufficient reasoning and structured output quality,
- `OpenRouter` second as capability expansion,
- provider-specific chains documented as examples, not as hidden magic.

Phases should have different bars:

- extraction needs structured recall and language discipline,
- verification needs conservative reasoning,
- report synthesis needs writing quality but must never outrank evidence.

### 2. Memory layering

Source pattern:

- `paperclip/skills/para-memory-files/SKILL.md`

Why it matters:

The PARA idea is too general to copy wholesale, but the layered memory model is useful.

What to adapt:

- raw timeline: pipeline logs and run artifacts,
- durable facts: verified entities, traces, coverage data,
- tacit knowledge: operator or corpus-level heuristics.

Direct implication:

`StratAudit` should distinguish:

- ephemeral run output,
- durable evidence pack,
- operator notes / project heuristics.

Do not mix those in one report blob.

### 3. Workflow menu / phase routing

Source pattern:

- `BMAD-METHOD/src/bmm/agents/analyst.agent.yaml`

Why it matters:

BMAD is useful because it exposes operator-facing workflow choices instead of hiding them behind one giant "analyst" instruction.

What to adapt:

Split `StratAudit` into a small menu of modes:

- `corpus-audit`
- `traceability-audit`
- `coverage-audit`
- `report-redraft`
- `evidence-pack`

This is better UX than one ambiguous "run strategy audit" skill.

### 4. Safety guards in plain language

Source pattern:

- `gastown/docs/skills/convoy/SKILL.md`

Why it matters:

The best part of `convoy` is not the command reference. It is the plain-language safety section. `StratAudit` needs the same thing.

What to adapt:

Write a compact `Safety guards` section for the future skill:

- never fabricate quotes,
- never translate initiative names unless explicitly asked,
- never claim a trace is verified if only similarity passed,
- never hide missing documents behind aggregate coverage,
- never summarize a report without provenance grade.

## Reject

These patterns should not be copied into `StratAudit`.

### 1. Giant preambles and session telemetry

Reject from:

- `gstack/investigate/SKILL.md`

Why reject:

This style is operationally heavy, noisy, and fragile. It makes sense for a batteries-included private environment with opinionated hooks. It is bad for a portable audit capability.

`StratAudit` needs clear behavior contracts, not ritual bootstrapping.

### 2. "Company brain" or agent persona theater

Reject from:

- parts of `paperclip`
- parts of `oh-my-openagent`
- parts of `BMAD-METHOD`

Why reject:

`StratAudit` is analyst tooling, not roleplay. It should be explicit about artifacts and evidence, not wrapped in personas or lore.

### 3. Hidden orchestration magic

Reject from:

- orchestration-heavy systems where decomposition is implicit and durable outputs are unclear

Why reject:

Strategy audit results must be reviewable and reproducible. If an agent fans out sub-work, the outputs still need to land as inspectable evidence. Hidden internal delegation is fine only if the public artifact remains deterministic enough to review.

## Proposed StratAudit Skill Contract

The future portable skill should be intentionally narrow.

### Trigger contract

Use when the user needs one of these:

- evidence-backed strategy alignment audit,
- cross-level traceability audit,
- coverage and gap analysis across strategic documents,
- evidence pack for strategy claims,
- report redraft grounded in existing audit artifacts.

Do not trigger for:

- generic business strategy advice,
- roadmap generation from scratch,
- freeform brainstorming without source documents,
- executive summary writing without evidence artifacts.

### Mandatory phases

1. `Scope`
   - identify corpus root, levels, exclusions, language, and output goal.
2. `Reality check`
   - inspect actual available documents and artifact health.
3. `Evidence pass`
   - verify quotes, spans, provenance, and corpus quality.
4. `Trace pass`
   - derive only inspectable traceability claims.
5. `Synthesis`
   - produce report sections with provenance grade.
6. `Refusal or warning`
   - if evidence is thin, say so directly and downgrade claims.

### Mandatory output contract

Every run should produce or expose these surfaces:

- `Executive summary`
- `Evidence quality summary`
- `Coverage by level and by document`
- `Trace explorer or trace table`
- `Findings with evidence`
- `Assumptions and unknowns`
- `Not covered / not claimed`

This is the biggest reusable pattern from the scan: good skills separate `what we know`, `how we know it`, and `what we are not claiming`.

### Runtime contract

The skill should be runtime-neutral.

Preferred order:

1. harness-native model if capability is sufficient,
2. configured external runtime,
3. OpenRouter as optional capability expansion.

The skill should declare runtime requirements, not vendors:

- structured extraction,
- conservative verification,
- multilingual fidelity,
- enough context handling for long documents.

### Refusal contract

The skill must refuse or downgrade output when:

- corpus root is missing,
- document provenance is incomplete,
- quotes cannot be verified,
- coverage denominator is unclear,
- traces are inferred but the user asks for verified alignment,
- the requested summary is broader than the evidence pack supports.

## What to Implement Next

These are the concrete changes suggested by the scan.

### 1. Create a real local skill surface

Add a real `StratAudit` skill in `sdp_lab`, not just `docs/STRATAUDIT.md`.

Recommended structure:

```text
skills/
  strataudit/
    SKILL.md
    references/
      evidence-policy.md
      runtime-policy.md
      report-contract.md
      corpus-quality-taxonomy.md
```

### 2. Keep the top-level skill short

`SKILL.md` should only contain:

- when to use,
- when not to use,
- safety guards,
- mandatory workflow,
- output contract,
- which reference file to load when.

### 3. Add evidence policy as a first-class reference

The strongest missing artifact today is a public, reusable evidence policy. It should explain:

- authority hierarchy,
- verification grades,
- quote verification rules,
- trace verification rules,
- confidence vs provenance distinction.

### 4. Add runtime policy as a first-class reference

Do not bury provider logic in code and env vars. Publish:

- required capabilities,
- host-native path,
- OpenRouter path,
- fallback behavior,
- fail-closed rules when runtime quality is insufficient.

### 5. Add explicit output modes

Do not expose only `run full audit`.

Expose at least:

- `audit-corpus`
- `audit-traceability`
- `audit-coverage`
- `build-evidence-pack`
- `redraft-report-from-artifacts`

## Final Recommendation

The right move is not to make `StratAudit` "more agentic".

The right move is to make it more explicit:

- explicit trigger,
- explicit evidence bar,
- explicit runtime policy,
- explicit refusal policy,
- explicit output contract.

The best borrowed patterns are the boring ones:

- verify before claiming,
- show the source hierarchy,
- keep trigger metadata clean,
- split references out of the main skill,
- make uncertainty visible.

That is what will make `StratAudit` portable across harnesses without turning it into a prompt-shaped dashboard hallucination.

## Source Files Read

- `/Users/fall_out_bug/projects/vibe_coding/agent-skills/skills/source-driven-development/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/agent-skills/skills/idea-refine/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/agent-skills/skills/spec-driven-development/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/hyperpowers/skills/brainstorming/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/hyperpowers/skills/verification-before-completion/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/hyperpowers/skills/dispatching-parallel-agents/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/superpowers/skills/brainstorming/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/superpowers/skills/verification-before-completion/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/superpowers/skills/writing-skills/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/paperclip/skills/paperclip/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/paperclip/skills/para-memory-files/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/gastown/docs/skills/convoy/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/gstack/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/gstack/design-review/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/gstack/investigate/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/oh-my-openagent/docs/guide/agent-model-matching.md`
- `/Users/fall_out_bug/projects/vibe_coding/oh-my-openagent/docs/guide/orchestration.md`
- `/Users/fall_out_bug/projects/vibe_coding/oh-my-openagent/AGENTS.md`
- `/Users/fall_out_bug/projects/vibe_coding/deer-flow/skills/public/deep-research/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/deer-flow/skills/public/bootstrap/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/deer-flow/skills/public/skill-creator/SKILL.md`
- `/Users/fall_out_bug/projects/vibe_coding/BMAD-METHOD/README.md`
- `/Users/fall_out_bug/projects/vibe_coding/BMAD-METHOD/docs/reference/workflow-map.md`
- `/Users/fall_out_bug/projects/vibe_coding/BMAD-METHOD/src/bmm/agents/analyst.agent.yaml`
- `/Users/fall_out_bug/projects/vibe_coding/JavaGuide/docs/ai/agent/skills.md`
- `/Users/fall_out_bug/projects/vibe_coding/JavaGuide/docs/ai/agent/context-engineering.md`
- `/Users/fall_out_bug/projects/vibe_coding/JavaGuide/docs/ai/agent/harness-engineering.md`
