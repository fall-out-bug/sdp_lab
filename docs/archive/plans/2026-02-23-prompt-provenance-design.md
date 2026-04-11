# Prompt Provenance: From "Prompts as Code" to Evidence-Native Trust

> **Date:** 2026-02-23
> **Status:** Research complete — decision reached
> **Origin:** Explored dynamic prompt generation ("Prompts as Code") as a Blueprint evolution; reframed into protocol-level prompt provenance after critical analysis

---

## TL;DR

"Prompts as Code" (a Go framework for dynamic prompt generation) is premature abstraction for 6 prompts in 75 lines of code — and contradicts SDP's own thesis that prompts can't reliably control agent behavior.

But "Prompt Provenance" (recording what the agent was told to do) is a natural strengthening of the evidence layer. **What matters isn't generating smarter prompts — it's proving what prompt the agent saw.**

**Decisions:**

| Idea | Verdict | When |
|------|---------|------|
| `internal/promptgen/` framework | **No** — premature abstraction, 6 variants | Revisit when variants > 15 |
| Prompt Provenance in evidence | **Yes** → F001 enhancement | Evidence Schema (Phase 1) |
| Consolidate prompt builders | **Yes** → F025 | Phase 0 cleanup |
| Pre-hydration context packet | **Yes** → F022 | Phase 0 enhancement |
| Phase Hooks | **Yes** → F024 | Post-F016 |
| Composable Phases pipeline | **Later** | When 3+ adopters need it |
| Prompt DSL | **No** | Never for this project |
| Meta-prompting (LLM → prompt → LLM) | **No** | Violates evidence-based philosophy |

---

## Current State: Prompts in the Codebase

### Go-Generated Prompts

| Location | Function | LOC |
|----------|----------|-----|
| `orchestrate/invoke_opencode.go` | `"Execute @build %s"` — one-liner triggers | 2 |
| `llm/prompt.go` | `BuildPrompt()` — task + boundary | 30 |
| `roles/reviewer.go` | `buildReviewPrompt()` — persona + evidence | 18 |
| `orchestrator/decomposer.go` | `buildDecomposePrompt()` — feature → subtasks | 22 |
| `llm/prompt.go` | `BuildPromptShort()` — one-liner | 3 |

**Total: 5 functions, ~75 lines of Go, 6 distinct prompt variants.**

### IDE-Loaded Prompts

43 markdown files in `.opencode/skills/` and `sdp/prompts/skills/`. Loaded by the IDE (Cursor/Claude Code/opencode) at session start. Go code doesn't interact with them — it says `"Execute @build"` and the IDE loads the skill.

### Key Finding

For the dominant path (skill-triggered), the Go prompt is **0.18% of the agent's context**. The remaining 99.8% comes from static SKILL.md files. Optimizing the 0.18% while the 99.8% stays unchanged is false precision.

---

## Why NOT to Build a Prompt Generation Framework

### Contradiction #1: SDP's Own Thesis

Phase 0 research (Agent Loop Reliability) concludes:

> *"Prompt engineering hits a ceiling at ~60-70% reliability. Prompts are probabilistic suggestions — they cannot guarantee behavior. Move control from prompts to deterministic code."*

`sdp orchestrate` already does this: Go code controls WHEN to stop, WHICH phase is next, WHETHER CI is green. Prompts just say "do this thing." Building "smarter prompts" means returning control to the probabilistic layer that we deliberately moved away from.

### Contradiction #2: The Numbers

6 prompts. 75 lines. A framework for 6 things is a CMS for a 3-page website.

### Contradiction #3: Wrong Bottleneck

The Go prompt is 0.18% of the signal. The rest — SKILL.md (hundreds of lines) + model capability + codebase state + tool access — dominates. Optimizing 0.18% while 99.8% is unchanged is optimization theater.

### Contradiction #4: Wrong Direction

The project is 178K LOC aiming for 6K. A new `internal/promptgen/` package increases the codebase, adds an abstraction layer, and creates a new debugging surface — the opposite of the goal.

---

## The Reframe: Prompt Provenance

### Core Idea

Not "generate smarter prompts" → but **"record what the agent was told to do."**

This strengthens the evidence layer (SDP's core value) rather than diverting into prompt engineering.

### Concrete Design

Add two fields to the evidence envelope's `provenance` section:

```json
{
  "provenance": {
    "artifact_id": "...",
    "orchestrator": "sdp-orchestrate",
    "runtime": "cursor",
    "model": "claude-sonnet-4-5-20250514",
    "role": "coder",
    "phase": "build",
    "prompt_hash": "sha256:a1b2c3d4e5f6...",
    "context_sources": [
      "issue:sdp_dev-abc",
      "boundary:internal/orchestrate/",
      "skill:build/SKILL.md",
      "handoff:.sdp/handoff/sdp_dev-abc/analyst.json",
      "checkpoint:.sdp/checkpoints/F004.json"
    ]
  }
}
```

- `prompt_hash` — SHA-256 of the rendered prompt (not the full text — it can be large)
- `context_sources` — list of all sources that entered the agent's context

### Why Provenance > Generation

| Question | promptgen answers | Prompt Provenance answers |
|----------|-------------------|---------------------------|
| What did the agent see? | No (generates, doesn't record) | Yes — hash + sources |
| Agent went off-track — why? | Maybe (if logged) | Yes — compare intent vs prompt |
| Two runs gave different results — same prompt? | Unclear | Yes — compare hashes |
| Did the prompt include review context? | Hidden in Go code | Explicit in `context_sources` |
| Works with third-party orchestrators? | No (our code) | Yes (protocol-level) |

### Elevator Pitch

> *"When an AI agent writes code, three things matter: what it was asked to do, what it actually did, and what it was told to do. Most tools capture the first two. SDP captures all three."*

---

## What IS Worth Doing with Prompts

### 1. Consolidate Prompt Builders → F025

Move 5 prompt-building functions into `internal/prompt/sections.go`. Extract shared sections (TaskSection, BoundarySection) as pure functions. Net result: ~80 LOC, likely fewer than today.

```go
package prompt

func TaskSection(id, title, desc, ac string) string { ... }
func BoundarySection(allowed, forbidden []string) string { ... }
func EvidenceSection(content string) string { ... }

func ForBuild(wsID string) string {
    return "Execute @build " + wsID +
        ". Output only code and commit message. After commit, output the commit hash."
}

func ForReview(issue IssueInput, evidence, persona string) string {
    return "You are a " + persona + " reviewer.\n\n" +
        TaskSection(issue.ID, issue.Title, issue.Description, issue.AC) +
        EvidenceSection(evidence) +
        "\nRespond with JSON: {\"verdict\": ..., \"comments\": [...]}\n"
}
```

**Benefit:** DRY without abstraction tax. `grep "prompt."` finds everything. Each section is independently testable. No new types.

### 2. Prompt Hash in Evidence → F001 Enhancement

Add to evidence-envelope schema:

```json
{
  "provenance": {
    "prompt_hash": { "type": "string", "pattern": "^sha256:[a-f0-9]{64}$" },
    "context_sources": { "type": "array", "items": { "type": "string" } }
  }
}
```

Implementation: `ExecuteResult.Prompt` (already saves the prompt) → compute hash → write to evidence. One line of code.

### 3. Pre-Hydration → F022

`sdp orchestrate --hydrate` writes `.sdp/context-packet.json` with WS spec, acceptance criteria, scope files, drift status, checkpoint state, dependency status, quality gate results.

This is not "prompt generation" — it's "context preparation." The prompt stays a one-liner: `"Read .sdp/context-packet.json and execute @build 00-004-01"`.

---

## Revisit Triggers

Upgrade to full PromptContext builder (Level 3) when 3+ of these trigger:

| Trigger | Threshold | Current |
|---------|-----------|---------|
| Prompt variants | > 15 | 6 |
| Runtime-specific prompts needed | 3+ runtimes with different prompt structure | 2 (opencode + Cursor), same prompt |
| Model-specific prompt adjustments | Regular model switches needing different format | 1 model |
| Prompt-related bugs | > 3 bugs caused by "wrong prompt for context" | 0 known |
| OSS contributors asking for customization | > 5 requests to change prompt behavior | 0 |

---

## Where the Blueprint Idea Actually Goes

The Blueprint model is not about prompts. It's about **composable pipelines with extension points**.

### Phase Hooks → F024 (The Real Blueprint Evolution)

```yaml
# .sdp/pipeline-hooks.yaml
hooks:
  pre-build:
    - command: "trivy fs ."
      on_fail: halt
  post-build:
    - command: "sdp drift detect ${WS_ID}"
      on_fail: warn
  pre-pr:
    - command: "sdp-evidence validate .sdp/evidence/"
      on_fail: halt
  post-ci:
    - command: "slack-notify #eng 'PR ready for review'"
      on_fail: ignore
```

This gives users extensibility (insert custom quality gates) without a prompt framework, without changing the state machine, and without writing Go. Hooks are shell commands.

### Composable Phases (Future — Not Now)

```yaml
# .sdp/pipeline.yaml (future)
phases:
  - name: build
    type: llm
    parallel: true
    timeout: 30m
    retry: 2
  - name: security-scan
    type: command
    command: "trivy fs ."
    on_fail: halt
  - name: review
    type: llm
    timeout: 15m
  - name: pr
    type: deterministic
  - name: ci
    type: deterministic
    timeout: 60m
```

This is the real Blueprint — a declarative pipeline with custom phases. It arrives when SDP has 3+ adopters who need customization.

---

## OSS Narrative

Don't call it "Prompts as Code." Call it **"Prompt Provenance."**

> *"Prompts as Code" sounds like a developer tool feature. "Prompt Provenance" sounds like a trust property.*
>
> *SDP evidence captures not just what agents did, but what they were told to do. The `prompt_hash` and `context_sources` in the evidence envelope create a verifiable chain: intent → instruction → execution → proof.*
>
> *If you use `sdp orchestrate`, you get optimized context-aware prompts for free. If you use your own orchestrator — fine, just capture the prompt hash. The protocol doesn't care how you build prompts. It cares that you can prove what the agent saw.*

---

## References

- [Stripe Minions Comparison](2026-02-23-stripe-minions-comparison.md) — Blueprint pattern analysis
- [Agent Loop Reliability](2026-02-23-agent-loop-reliability.md) — prompt ceiling at 60-70%
- [Oneshot Autonomous Design](2026-02-23-oneshot-autonomous-design.md) — pre-hydration and checkpoint design
- [Dream Swarm Design](2026-02-22-dream-swarm-design.md) — architecture decisions
