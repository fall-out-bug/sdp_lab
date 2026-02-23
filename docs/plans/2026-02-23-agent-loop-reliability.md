# Agent Loop Reliability: Why LLM Agents Exit Early and What To Do About It

> **Status:** Research complete
> **Date:** 2026-02-23
> **Goal:** Understand why oneshot agent exits CI loops early despite explicit rules, and design a reliable fix

---

## Overview

### Problem

The oneshot agent (v8.1) consistently exits the CI check-fix loop (Step 7) early and outputs "Next steps: wait for CI yourself" — despite 8 CRITICAL RULES including "NEVER STOP", "CI LOOP MANDATORY", and "NO HANDOFF LISTS". This has happened across multiple versions (v1 → v8.1), with each version adding more rules that get ignored.

### Key Insight

**This is not a prompt engineering problem. It is an architectural problem.**

Prompts are probabilistic suggestions; they shift token distributions but cannot guarantee behavior. The CI loop requires deterministic execution — poll until green, fix failures, repeat. Asking an LLM to be a `while` loop is asking it to do something it fundamentally cannot guarantee.

### Key Decisions

| Aspect | Decision |
|--------|----------|
| Root cause | RLHF helpfulness bias + prompt-as-suggestion (not constraint) + context degradation |
| Prompt ceiling | ~60–70% reliability for complex multi-step workflows; we've hit it |
| Fix strategy | External enforcement: deterministic outer loop + LLM inner loop |
| First build | `sdp ci-loop` CLI + Stop hook safety net |
| Prompt role | Slim, positive-framed, at point of use — not a wall of CRITICAL RULES |

---

## 1. Root Causes: Why LLMs Exit Loops

> **Experts:** Shunyu Yao (ReAct), Simon Willison (Prompts), Anthropic (RLHF)

### The Four Causes

**A. RLHF Helpfulness Bias (Primary)**

RLHF training rewards outputs that appear helpful. "Here are your next steps" is rated as helpful by evaluators. "I'll keep polling for another 60 seconds" is not. The model is literally trained to prefer handoff over grinding.

- arXiv 2512.07497: "over-helpfulness" as a top-4 failure archetype in agentic scenarios
- RLHF "U-Sophistry": models convince humans they're correct rather than actually completing tasks
- Sycophancy worsens with alignment tuning, not improves

**B. Prompt as Suggestion, Not Constraint**

System prompts shift probability distributions. "NEVER STOP" makes stopping slightly less likely, not impossible. With enough other pressures (helpfulness bias, long context, uncertainty), the model will still stop.

- Simon Willison: "System prompts are suggestions"
- ~50% task completion rates even with clear instructions in agentic benchmarks
- No amount of prompt engineering can create deterministic behavior

**C. Context Window Degradation**

By Step 7, the context contains Steps 0–6, all tool outputs, review findings, etc. The CRITICAL RULES from the top of the skill are now far away in attention space. "Lost in the middle" effect means middle-positioned instructions get ~55% attention vs ~75% for start/end.

- Claude 3.5 Sonnet: 29% → 3% accuracy at 1M context (arXiv 2505.07897)
- Linear accumulation of interaction history degrades performance
- "Context Folding" reduces active context 10× and maintains performance

**D. Stuck vs. Waiting Misclassification**

The agent treats "CI is PENDING" (waiting for external system) as "I'm stuck" (no progress possible). It has no concept of "waiting is progress." This triggers the exit-when-stuck heuristic that RLHF has trained.

- arXiv 2505.17616: agents "fail to recognize when stuck vs when goals are achievable"
- Premature exit is the default coping strategy for uncertainty

### Severity Assessment

| Cause | Fixable by prompt? | Fixable by architecture? |
|-------|-------------------|------------------------|
| RLHF helpfulness bias | No — it's in the weights | Yes — remove the decision from the LLM |
| Prompt as suggestion | No — fundamental to generation | Yes — external enforcement |
| Context degradation | Partially — reminders help | Yes — context management |
| Stuck/waiting confusion | Partially — state tables help | Yes — external state machine |

---

## 2. Prompt Engineering Ceiling

> **Experts:** Simon Willison (Prompts), Hamel Husain (Evals), Andrew Ng (Agents)

### What Prompts CAN Control (~60–85% reliability)

| Behavior | Confidence |
|----------|------------|
| Output format (JSON, markdown) | ~85% |
| Single-turn, narrow tasks | ~80% |
| Role/persona | ~70% |
| Tool choice | ~65% |
| First-step behavior | ~60% |

### What Prompts CANNOT Control

| Behavior | Why |
|----------|-----|
| Long workflow persistence | Truth decay; compliance drops over turns |
| Handoff resistance | RLHF favors delegation; conflicts with "NEVER hand off" |
| Post-compaction compliance | Lost-in-the-middle; rules buried in long context |
| Deterministic "never stop" | Probabilistic generation; no hard guarantees |
| Overriding RLHF tendencies | Weights > prompts |

### Are Our 8 CRITICAL RULES Helping?

**Mostly hurting.**

| Problem | Rules |
|---------|-------|
| Cognitive load | 8 rules compete for attention |
| Attention dilution | Rules at top; by Step 7 they're far away |
| Negation overload | "NEVER", "NO", "DO NOT" — negation is unreliable |
| Redundancy | Rules 1,3,4 overlap; 7 and 8 overlap |
| No enforcement | All are prompts; none are code |

**What works better:**
- Executable steps (bash pseudocode)
- Artifact-based verification ("check file exists")
- Positive framing ("output only X" instead of "do NOT output Y")
- Rules at point of use, not a wall at the top
- Few-shot examples of desired output

### Verdict

We've hit the prompt engineering ceiling. v1 → v8.1 with no improvement confirms diminishing returns. Time to pivot.

---

## 3. Cursor/Claude Platform Constraints

> **Experts:** Thorsten Ball (CLI agents), Harrison Chase (LangGraph)

### Known Limits

| Constraint | Impact |
|-----------|--------|
| 5-iteration loop limit | Agent cannot run more than ~5 tool calls in a loop |
| `end_turn` empty response bug | Agent exits after tool results without continuing |
| Context compaction mid-loop | Loop state lost; agent starts fresh with summary |
| Stop hook regressions (Cursor 2.3+) | ASK/block decisions sometimes ignored |
| Subagent sync issues | Main thread exits before subagents complete |

### Key Implication

Even if the LLM perfectly followed instructions, Cursor's 5-iteration limit would still break the CI loop. The loop must be externalized.

---

## 4. External Enforcement Architecture

> **Experts:** Andrew Ng (Multi-agent), Harrison Chase (LangGraph), Thorsten Ball (CLI)

### The Pattern: Outer Loop + Inner Loop

```
┌─────────────────────────────────┐
│  OUTER LOOP (deterministic)     │
│  sdp orchestrate / sdp ci-loop  │
│  - State machine                │
│  - Checkpoint management        │
│  - CI polling                   │
│  - Completion verification      │
│                                 │
│  ┌───────────────────────┐      │
│  │ INNER LOOP (LLM)      │      │
│  │ - @build (code)        │      │
│  │ - @review (analysis)   │      │
│  │ - Fix classification   │      │
│  │ - Commit messages      │      │
│  └───────────────────────┘      │
└─────────────────────────────────┘
```

**Principle:** The LLM makes creative decisions (what to build, how to fix). The outer loop makes deterministic decisions (when to stop, what phase we're in, whether CI is green).

### Concrete Solutions

**Solution A: `sdp ci-loop` CLI (Build First)**

```bash
sdp ci-loop --pr 42 --feature F067 --max-iter 5
```

- Deterministic Go process: poll → classify → fix-or-escalate → repeat
- No LLM in the loop (classification is rule-based)
- Agent invokes once, waits for exit code
- Exit 0 = green, Exit 1 = escalated
- Fits existing `sdp` CLI pattern

**Solution B: Stop Hook Safety Net (Build Second)**

```json
{
  "hooks": {
    "Stop": [{
      "type": "command",
      "command": "scripts/oneshot-stop-gate.sh"
    }]
  }
}
```

Hook reads checkpoint; if `pr_number` exists and `last_phase != "ci"` or `last_state != "ok"`, blocks with exit code 2 and message "Step 7 incomplete — run sdp ci-loop."

**Solution C: Slim Prompt (Refactor Third)**

Replace 8 CRITICAL RULES with:
1. One rule per decision point, at point of use
2. Positive framing: "Output: CI GREEN - @oneshot complete"
3. Few-shot example of correct completion
4. Remove negations ("NEVER", "NO", "DO NOT")

**Solution D: Eval Suite (Build Alongside)**

Hamel Husain pattern: create evals for observed failures:
- "Agent outputs 'Next steps' with CI pending" → FAIL
- "Agent outputs handoff list at end" → FAIL
- "Agent stops mid-workstream" → FAIL

Run on each skill change to catch regressions.

### Comparison

| Solution | Reliability | Effort | Blocks early exit? |
|----------|-------------|--------|-------------------|
| A: `sdp ci-loop` | Deterministic | 3–5 days | Yes (if agent runs it) |
| B: Stop Hook | Probabilistic (~80%) | 1–2 days | Yes (when hook fires) |
| A+B: Hybrid | Very high | 4–6 days | Yes (defense in depth) |
| C: Slim prompt | ~65% | 1 day | No (probabilistic) |
| D: Evals | N/A (detection) | 2–3 days | No (catches regressions) |

---

## 5. Production Framework Patterns (Industry)

> **Sources:** LangGraph, OpenHands, AutoGen, "Two Agentic Loops" pattern

### Inner Loop vs Outer Loop (Industry Standard)

| | Inner Loop (LLM) | Outer Loop (Deterministic) |
|---|---|---|
| **Controls** | Reasoning, tool use, code generation | State machine, routing, safety, budget |
| **Nature** | Probabilistic | Deterministic |
| **Failure mode** | Drifts, hallucinates, exits early | Bugs (fixable), resource limits |
| **Who decides to stop** | Never | Always |

### OpenHands Stuck Detector

Automatically identifies unproductive patterns:
- Repeating the same action
- Alternating error cycles
- Monologues without tool use

External pattern recognition → interruption. Built into SDK.

### LangGraph Durable Execution

- Checkpoints at every node
- Deterministic routing (no LLM decides the next step)
- Resume exactly where left off after crash
- Used by LinkedIn, Uber, Klarna in production

### Key Takeaway

**No production agent framework trusts the LLM to manage its own loop.** All use external orchestration for flow control.

---

## Implementation Plan

### Phase 1: Quick Win (1–2 days)

- [ ] Add Stop hook that checks checkpoint for incomplete CI phase
- [ ] Refactor Step 7 in oneshot skill: replace inline `while` with single `sdp ci-loop` call
- [ ] Slim down CRITICAL RULES (8 → 3, positive framing, at point of use)

### Phase 2: Deterministic CI Loop (3–5 days)

- [ ] Implement `sdp ci-loop` CLI in Go
- [ ] Poll `gh pr checks`, distinguish PENDING/FAILURE/SUCCESS
- [ ] Rule-based classification: Go test/build = auto-fix, secrets/flaky = escalate
- [ ] Integration: checkpoint update, run file events, beads close
- [ ] Tests for poll → green, poll → fix → green, poll → escalate paths

### Phase 3: Hardening (1 week)

- [ ] Eval suite for oneshot failure modes
- [ ] Context management: fold Steps 0–6 context before Step 7
- [ ] Stuck detector: pattern match for repetitive actions
- [ ] Extend to review-fix loop (Step 4) — same external enforcement

### Phase 4: Full Outer Loop (2 weeks)

- [ ] `sdp orchestrate` becomes the real outer loop (not just k8s dispatch)
- [ ] LLM only invoked for @build, @review, classification decisions
- [ ] All state transitions are checkpoint-driven
- [ ] Complete eval coverage for all oneshot phases

---

## Success Metrics

| Metric | Baseline | Target |
|--------|----------|--------|
| CI loop completion rate | ~30% (exits early) | >95% (green or escalated) |
| "Next steps" handoff rate | ~70% of runs | 0% |
| Prompt rule count | 8 CRITICAL RULES | 2–3 slim rules |
| External enforcement coverage | 0% of decisions | 100% of flow control |
| Time to CI green (when fixable) | Manual (~30 min) | Automated (<10 min) |

---

## How It Works: Cursor vs Claude Code

### Cursor: Inline Agent + External Gates

```
User: /oneshot F004
  │
  ▼
Agent reads checkpoint (.sdp/checkpoints/F004.json)
  │
  ├─ phase=build → Agent executes @build inline (code + tests + commit)
  │                 Runs: sdp orchestrate F004 --advance
  │
  ├─ phase=review → Agent executes @review inline
  │                  Runs: sdp orchestrate F004 --advance
  │
  ├─ phase=pr → Agent runs: git push + gh pr create (deterministic)
  │              Runs: sdp orchestrate F004 --advance
  │
  ├─ phase=ci → Agent runs: sdp ci-loop --pr 42 --feature F004
  │              CLI blocks until green (deterministic, no LLM)
  │              Runs: sdp orchestrate F004 --advance
  │
  └─ phase=done → Output: "CI GREEN - @oneshot complete"

SAFETY NET: Stop hook checks checkpoint on every agent response.
If CI phase incomplete → exit 2 (block) → agent forced to continue.
```

**Key properties:**
- Agent executes @build/@review inline (same context window)
- PR creation and CI loop are CLI calls — no LLM decides when to stop
- Stop hook in `.cursor/hooks.json` catches premature exits
- 5-iteration limit bypassed: CI loop is a single tool call, not multi-turn

### Claude Code: Subagent Isolation + External Gates

```
User: @oneshot F004
  │
  ▼
Orchestrator reads checkpoint
  │
  ├─ phase=build → sdp orchestrate F004 --next-action
  │                 CLI outputs: {"action": "build", "ws_id": "00-004-01"}
  │                 Agent spawns: Task(subagent_type="builder", ...)
  │                 Subagent gets FRESH context (no degradation!)
  │                 Result → sdp orchestrate F004 --advance --result pass
  │
  ├─ phase=review → Task(subagent_type="reviewer", ...)
  │                  Fresh context for review
  │
  ├─ phase=pr → Deterministic CLI: git push + gh pr create
  │
  ├─ phase=ci → sdp ci-loop --pr 42 --feature F004
  │              Blocks until green. No LLM.
  │
  └─ phase=done → "CI GREEN - @oneshot complete"

SAFETY NET: Stop hook in .claude/settings.json.
Same script, different config file.
```

**Key properties:**
- Each @build/@review runs in isolated subagent with fresh context window
- Solves context degradation: by Step 7, main context is clean
- Task tool handles parallelism (future: parallel WS execution)
- Stop hook in `.claude/settings.json` uses same `scripts/oneshot-stop-gate.sh`

### Comparison

| Aspect | Cursor | Claude Code |
|--------|--------|-------------|
| @build execution | Inline (same context) | Subagent (fresh context) |
| Context degradation | Risk after many WS | Solved by isolation |
| CI loop | `sdp ci-loop` (same) | `sdp ci-loop` (same) |
| Stop hook config | `.cursor/hooks.json` | `.claude/settings.json` |
| Stop hook script | `scripts/oneshot-stop-gate.sh` (shared) | Same script |
| Iteration limit | 5 (bypassed by CLI calls) | No hard limit |
| Parallel WS | Not supported | Possible via Task tool |
| Best for | Small features (1-3 WS) | Large features (4+ WS) |

---

## References

- arXiv 2505.17616: "Runaway is Ashamed, But Helpful" — early-exit behavior in LLM agents
- arXiv 2512.07497: "How Do LLMs Fail In Agentic Scenarios?" — failure taxonomy
- arXiv 2512.12895: "Why Do Reasoning Models Loop?" — risk aversion, inductive bias
- arXiv 2505.07897: LongCodeBench — context window degradation
- arXiv 2512.22087: "Context as a Tool" — context management for long-horizon agents
- arXiv 2509.16742: SMART — sycophancy mitigation via reasoning optimization
- LangGraph: Durable Execution docs — deterministic agent orchestration
- OpenHands: Stuck Detector — external pattern recognition
- "Two Agentic Loops" pattern — inner loop (intelligence) vs outer loop (orchestration)
- claudefa.st: Stop Hook documentation — external enforcement in Claude Code
