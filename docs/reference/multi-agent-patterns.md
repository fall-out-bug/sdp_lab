# Multi-Agent Orchestration Patterns -- SDP Playbook

> **Scope:** When and how SDP agents delegate work to each other.
> **Source:** Adaptation of [Anthropic -- Multi-agent coordination patterns](https://claude.com/blog/multi-agent-coordination-patterns) (April 2026) to the SDP FSM.
> **Policy:** F127-04 (`docs/plans/2026-04-16-f127-multi-harness-modernization-design.md`).

---

## 1. Introduction: Why Patterns Matter for SDP

SDP's execution kernel (`internal/agentloop` FSM) already mixes several coordination models --
but without explicit selection criteria. Different phases (Discover, Plan, Build, Review, Eval)
demand fundamentally different semantics. Without formalization, three failure modes recur:

1. **Orchestrator bottleneck.** The lead agent accumulates all sub-agent outputs into its own
   context. Beyond ~80K tokens the model degrades, producing shallow analysis and missing
   edge cases. This is the single most common multi-agent failure in SDP.

2. **False parallelism.** Parallel dispatch applied to tasks that share hidden state (shared
   test fixtures, overlapping file regions, interdependent acceptance criteria). Wall-clock
   time improves, but merge conflicts or semantic conflicts erase the gain.

3. **Recursive delegation.** An orchestrator delegates to sub-agent A, which delegates to B,
   which delegates to C. Context tax compounds at each level (1000-2000 tokens per hop),
   latency multiplies, and the original intent becomes a game of telephone.

This document formalizes five coordination patterns from the Anthropic blog post with SDP-specific
examples, selection criteria, anti-patterns, and cost estimates. The goal: any SDP operator or
agent can unambiguously pick the right pattern before dispatching work.

**How to use this document:**
- Start with the [Decision Tree](#2-decision-tree) to pick a pattern.
- Read the pattern's section for SDP examples and pitfalls.
- Check [Choosing the Right Pattern for SDP Phases](#8-choosing-the-right-pattern-for-sdp-phases)
  for phase-specific defaults.

---

## 2. Decision Tree

### Textual form

1. **Need delegation?** If the task is a single edit in one file or a trivial lookup -- do not
   delegate. Sub-agent bootstrap costs 1000-2000 tokens of context initialization. If the task
   is smaller than the bootstrap tax, keep it in the current agent.

2. **Is the goal adversarial review?** Generator produces an artifact; a separate agent
   independently critiques it without seeing the generator's reasoning. If yes, use
   **Generator-Verifier** (Pattern 1).

3. **Are sub-tasks bounded and independent?** Each sub-task has clear inputs, clear outputs, and
   no shared mutable state. If yes, proceed to question 4. If sub-tasks are sequential or
   interdependent, use **Orchestrator-Subagent** (Pattern 2) in sequential mode.

4. **Will sub-agents persist across tasks?** If sub-agents are one-shot (spawn, do work, return
   result, die), use **Orchestrator-Subagent** (Pattern 2) with parallel dispatch. If sub-agents
   form a persistent worker pool that survives multiple task assignments, use
   **Agent Teams** (Pattern 3).

5. **Event-driven with many producers/consumers?** Multiple independent agents publish events;
   multiple consumers react asynchronously. If yes, use **Message Bus** (Pattern 4).

6. **Multi-agent coordination via shared store?** Agents do not message each other directly.
   They read and write a common persistent store (filesystem, database). If this is the primary
   coordination mechanism, use **Shared State** (Pattern 5).

### Quick reference table

```
  Condition                                    Pattern
  ─────────────────────────────────────────── ──────────────────────
  No delegation needed                         Keep in current agent
  Adversarial review (pass/fail)               Generator-Verifier
  Bounded + independent + one-shot             Orchestrator-Subagent
  Bounded + independent + persistent pool      Agent Teams
  3+ independent tasks, no shared state        Orchestrator-Subagent (parallel)
  Event-driven, many producers/consumers       Message Bus
  Coordination via shared persistent store     Shared State
```

### Parallel dispatch rules

Parallel dispatch is a specialization of Orchestrator-Subagent. **All five** conditions must hold:

1. **3+ independent tasks.** With fewer tasks, bootstrap tax exceeds the wall-clock benefit.
2. **No shared mutable state.** Sub-agents do not write to the same files.
3. **Clear file boundaries.** Task A touches `foo.go`, task B touches `bar.go`.
4. **Clear goal per agent.** One sub-agent = one acceptance criterion passed.
5. **Bounded output.** Sub-agent returns a summary, not a full log.

If **any** condition fails, run sequentially.

---

## 3. Pattern 1 -- Generator-Verifier

**Idea:** One agent generates an artifact (code, spec, review plan). A second agent independently
critiques it, never seeing the generator's reasoning context. The verifier receives only the
artifact and the evaluation criteria.

### SDP examples

| Phase | Generator | Verifier | Artifact |
|-------|-----------|----------|----------|
| Build | `implementer` (writes code via TDD) | `spec-reviewer` (checks against AC) | Diff + test output |
| Review | `tech-lead` (writes review plan) | `qa` + `security` + `sre` (multi-axis critique) | Review findings |
| Discovery / council | One LLM model | Another LLM model (blind review) | Architecture proposal |
| Bootstrap | `sdp bootstrap` (generates CLAUDE.md) | Human or second agent (checks conventions) | CLAUDE.md content |

**Concrete SDP scenario -- `@build` + `@review` cycle:**

1. Lead agent dispatches `implementer` to realize workstream `00-FFF-SS` via TDD.
2. `implementer` returns: diff, test output, list of modified files.
3. Lead dispatches `spec-reviewer` with **only** the diff and the workstream's acceptance criteria.
   The reviewer does not see the implementer's chain-of-thought.
4. If reviewer finds blocking issues, lead sends them back to a fresh `implementer` sub-agent.
5. Cycle repeats until zero P0/P1 findings.

### When to use

- The task has a binary or scored outcome (pass/fail, approve/reject, 0-10 quality rating).
- The generator is prone to self-consistency -- it will rationalize its own mistakes.
- There is a clear evaluation criterion (spec, acceptance criteria, threat model, SLO).

### Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Same model + same context | This is self-review, not adversarial review | Verifier must have a separate context with only the artifact |
| Verifier sees full generative chat | Verifier is no longer "blind" -- it will anchor on the generator's reasoning | Pass only the artifact and the evaluation rubric |
| No tie-breaker | After two rounds of disagreement, neither agent converges | Designate a Decision Owner (human or Opus-tier agent) |
| Verifier is lower-tier than generator | Haiku reviewing Opus output misses subtle issues | Match or exceed generator's tier for verification |

### Cost estimate

- **Token multiplier:** ~2x relative to single-agent execution (generator run + verifier run).
- **Defect reduction:** 20-40% fewer defects in practice (based on SDP `llm-council` retrospectives).
- **Wall-clock:** ~1.5-2x (verifier can start immediately after generator finishes).
- **Net ROI:** Positive when the cost of a defect escaping to the next phase exceeds the
  verification cost. For production code and security reviews, this is almost always the case.

---

## 4. Pattern 2 -- Orchestrator-Subagent (one-shot)

**Idea:** The lead agent (orchestrator) delegates a bounded task to a sub-agent. The sub-agent
returns a result in one final message and terminates. The orchestrator retains no sub-agent state
after return. This is the most common SDP pattern in production today.

### SDP examples

| Orchestrator | Sub-agent | Bounded task |
|---|---|---|
| `orchestrator` (lead session) | `scout` | "Scan repo, return `scout.json`" |
| `planner` | `analyst` | "Decompose feature into 3-5 workstreams" |
| `reviewer` | `security` | "Produce threat model for this PR diff" |
| sdp-harness FSM | `implementer` | "Realize WS 00-FFF-SS, return diff + evidence" |
| `orchestrator` | `sdp architect` | "Analyze architecture, return `report.json`" |

**Concrete SDP scenario -- `sdp orchestrate --feature F050`:**

1. Orchestrator reads ROADMAP.md, identifies 8 leaf workstreams for F050.
2. For each workstream, checks `bd ready` -- finds 5 are unblocked.
3. Dispatches 5 sub-agents in parallel (via Claude Code Task tool or Codex CLI), each
   receiving exactly one workstream file and the feature's acceptance criteria.
4. Each sub-agent runs the Build phase independently: implement, test, collect evidence.
5. Sub-agents return summaries (not full context). Orchestrator assembles results.
6. Orchestrator dispatches a Review sub-agent against the combined diff.

### When to use

- The task is **bounded** -- clear input, clear output, no open-ended loops.
- The sub-agent does not need to persist state after returning its result.
- There is no shared mutable state between sub-agents.
- For 3+ independent bounded tasks, use parallel dispatch (see rules above).

### Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Orchestrator-bottleneck | Lead collects all sub-outputs into its context, exceeding 80K tokens | Summarize sub-outputs before assembling; or switch to Agent Teams |
| Recursive dispatch (A->B->C) | Context tax multiplies; latency compounds; intent degrades at each hop | Flatten: orchestrator delegates to all sub-agents directly |
| Mixed responsibilities in one sub-agent | Sub-agent gets "implement X and also review Y" -- produces shallow work on both | One sub-agent = one bounded task |
| Orchestrator micro-manages | Lead sends follow-up messages to sub-agent, turning one-shot into a conversation | Accept the sub-agent's result or re-dispatch a new sub-agent |

### Cost estimate

- **Per sub-agent:** 1x execution cost + 500-1500 tokens for context bootstrap.
- **Parallel N tasks:** ~Nx sub-agents in token cost, but wall-clock ~= 1x longest task.
- **Sequential N tasks:** Nx in both tokens and wall-clock.
- **Recommendation:** Parallel dispatch for 3+ tasks; sequential for 1-2 or when tasks are interdependent.

---

## 5. Pattern 3 -- Agent Teams (persistent workers)

**Idea:** A stable pool of worker agents that survive across multiple task assignments. Workers
share a task list, can message each other directly, and maintain their own context windows across
assignments. Workers claim tasks from a shared backlog.

### SDP examples

| Team | Workers | Lifecycle |
|---|---|---|
| Delivery squad | implementer, spec-reviewer, qa | Lives for the duration of one PR: cycles through build, review, fix |
| Discovery squad | scout, analyst, architect | Lives during cold-start onboarding of a brownfield repo |
| Review team | qa, security, sre, devops, tech-lead | Lives for the duration of a review cycle on a large PR |

**Concrete SDP scenario -- parallel workstream dispatch (F050 with 13 workstreams):**

1. Lead spawns 4-6 implementer workers as persistent Agent Team members.
2. Workers pull from a shared task list (backed by `bd ready` output).
3. Worker-1 claims WS-01 (isolated in git worktree). Worker-2 claims WS-02. And so on.
4. As workers finish, they claim the next unblocked workstream.
5. Workers can message each other: "WS-03 needs the interface from WS-01 -- is it stable?"
6. Lead synthesizes completed workstreams into the final PR.

**Note:** As of April 2026, production SDP uses one-shot Orchestrator-Subagent. Agent Teams
are a roadmap item (F126 + Claude Code `AGENT_TEAMS` experimental flag). The `.agents/skills/`
directory and `docs/AGENT_TEAMS.md` contain the planned configuration.

### When to use

- Workers are **re-usable** across multiple tasks within the same workflow.
- There is genuine **parallelism** (3+ concurrent tasks) that justifies the bootstrap cost.
- Workers benefit from **accumulated context** (learning about the codebase across tasks).
- The token savings from avoiding repeated context-bootstrap (3-4 tasks each paying 1500 tokens)
  exceeds the ongoing cost of maintaining persistent worker contexts.

### Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Team for 1-2 tasks | Bootstrap cost dominates; one-shot dispatch is cheaper | Use Orchestrator-Subagent instead |
| No shared task list schema | Workers claim duplicate tasks or skip tasks | Implement `bd ready` as the canonical task source |
| No lead / no tie-breaker | Workers disagree without resolution mechanism | Designate a lead worker or fall back to orchestrator for disputes |
| Team members with identical roles | Redundant workers add cost without diversity | Give each worker a distinct role (implementer vs reviewer vs qa) |

### Cost estimate

- **Token multiplier:** 3-4x vs one-shot Orchestrator-Subagent (persistent context is larger).
- **Wall-clock:** 2-3x faster for pipelines with 5+ tasks (parallelism amortizes bootstrap).
- **Break-even:** ~4-5 tasks. Below that, one-shot dispatch is cheaper.
- **Recommendation:** Use for features with 5+ independent workstreams and complex cross-WS
  dependencies. Stick with Orchestrator-Subagent for simpler features.

---

## 6. Pattern 4 -- Message Bus

**Idea:** Event-driven pub/sub coordination. Agents publish events to a bus. Subscribed agents
react to events asynchronously. Producers and consumers are decoupled -- neither knows about
the other directly.

### SDP examples

| Bus | Producers | Consumers | Event |
|---|---|---|---|
| CI findings bus | GitHub Actions (sensor) | Local analysis agent | Workflow run completed (pass/fail) |
| Beads queue | Any agent creating work | `beads:task-agent`, human reviewers | New issue created |
| Review findings bus | reviewer, qa, security | orchestrator (collects blocking findings) | Finding filed (P0/P1/P2) |
| Evidence bus | implementer (tool outputs) | `EvidenceAccumulator` in agentloop FSM | `file_modified`, `test_passed`, `tool_error` |
| Drift events | `sdp drift detect` | orchestrator, human | Workstream-code mismatch detected |

**Concrete SDP scenario -- beads as event bus:**

1. Reviewer sub-agent files a P0 finding as a beads issue: `bd create "Security: SQL injection in user.go"`.
2. The beads queue is the bus. Orchestrator subscribes: `bd ready --json` on each cycle.
3. Finding appears as a blocking dependency on the PR's feature.
4. Implementer claims the finding: `bd update <id> --claim`.
5. Fix is pushed. CI runs. If CI passes, the finding is closed: `bd close <id>`.
6. Transport layer (`scripts/beads_transport.sh export`) persists the event stream to git.

SDP already uses this model through the beads queue + transport scripts. The beads queue acts
as a durable event store with replay capability (via Dolt or git-backup).

### When to use

- **Many producers, many consumers** with non-1:1 relationships.
- Events are **asynchronous** -- findings appear at unpredictable times.
- **Persistence/replay** is required -- events must outlive individual sub-agent sessions.
- **Loose coupling** is valuable -- producers and consumers evolve independently.

### Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Bus for 2 agents with 1:1 relationship | Over-engineering; direct delegation is simpler | Use Orchestrator-Subagent instead |
| No ordering/causality guarantees | Race conditions in consumers (stale state) | Add sequence numbers or use Dolt's SQL transaction ordering |
| No dead-letter policy | Stuck events block the queue indefinitely | Define TTL + escalation path for unprocessed events |
| Bus-as-polling | Consumers run `ls` in a loop checking for new events | Use event triggers (git hooks, MCP subscriptions) instead |

### Cost estimate

- **Per-event:** Low (tokens for the event payload only).
- **Infrastructure:** Requires a persistent store (beads/Dolt, git log, file system).
- **SDP already pays this cost:** The beads queue is the bus. No additional infrastructure needed.
- **Recommendation:** This is SDP's default coordination mechanism for findings and CI events.
  Do not introduce a separate message broker -- extend the existing beads-based bus.

---

## 7. Pattern 5 -- Shared State

**Idea:** Agents do not send messages to each other. Instead, they read and write a shared
persistent store (filesystem, database, ConfigMap). Coordination happens implicitly through
the store's state.

### SDP examples

| Store | Writers | Readers | State shape |
|---|---|---|---|
| `.sdp/evidence/` | implementer, qa | reviewer, orchestrator, CI | JSON per tool output (structured) |
| `.sdp/checkpoints/` | `sdp-orchestrate` FSM | Recovery on restart | Phase state + accumulated evidence |
| `.sdp/dispatch/profiles/*.json` | dispatch router | Future dispatch decisions | Capability profiles per harness |
| `.agents/skills/` | Skill authors (human or agent) | All harnesses (Claude Code, OpenCode, Cursor, Codex) | Skill markdown files |
| `AGENTS.md` | Human operators | All agents in all harnesses | Agent instructions |
| Git worktrees | Lead agent (creates) | Sub-agents (work in isolation) | Isolated filesystem trees |
| `.sdp/scout.json` | `sdp scout` | `@understand`, `@architect`, `@metrics` | Quick repo facts |

**Concrete SDP scenario -- git worktree isolation for parallel work:**

1. Orchestrator identifies 3 independent workstreams that touch overlapping file areas.
2. Orchestrator creates 3 git worktrees: `git worktree add .claude/worktrees/ws-01`.
3. Each sub-agent operates in its own worktree (no shared mutable state).
4. Sub-agents write evidence to `.sdp/evidence/{ws-id}/` within their worktree.
5. Orchestrator merges worktrees back: one at a time, running quality gates after each.
6. If merge conflict occurs, orchestrator resolves using context from both sub-agents' evidence.

**Concrete SDP scenario -- `.sdp/checkpoints/` for crash recovery:**

1. `sdp-orchestrate --feature F050` enters the Build phase.
2. After each tool call, `EvidenceAccumulator` writes to `.sdp/checkpoints/F050-build.json`.
3. Context compaction or crash occurs mid-build.
4. Next invocation reads the checkpoint, resumes from the last recorded tool call.
5. No re-work needed -- evidence from prior tool calls is preserved in the checkpoint.

### When to use

- There is **no direct messaging** between agents -- state is long-lived and read by different
  agents at different times.
- **Crash resilience** is required (checkpoint/replay).
- The store is the **single source of truth**, not a cache.
- Agents need to **coordinate without being online simultaneously** (write now, read later).

### Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Two agents writing same file without lock | Race condition: last-write-wins, data loss | Use git worktrees for isolation, or file-level locking |
| Store becomes implicit bus | Agents poll the filesystem for changes | If agents need real-time notification, use Message Bus instead |
| Schema evolves without versioning | Old artifacts become unreadable | Version all `.sdp/` schemas; include `schema_version` field |
| Store grows unbounded | Stale checkpoints and evidence accumulate | Implement retention policy; archive completed feature evidence |

### Cost estimate

- **CPU/token:** Low. No per-message token overhead -- agents only pay for reads and writes.
- **Discipline cost:** High. Requires strict versioning, concurrency-safe writes, and retention
  policies. Without discipline, the store rots.
- **Recommendation:** Use Shared State for all persistent artifacts (evidence, checkpoints,
  profiles, skills). Use Message Bus for real-time coordination on top of the store.

---

## 8. Choosing the Right Pattern for SDP Phases

| SDP Phase | Primary pattern | Secondary pattern | Rationale |
|-----------|----------------|-------------------|-----------|
| Discovery | Orchestrator-Subagent | Generator-Verifier (llm-council) | Scout/architect/spec are bounded one-shot tasks; council uses adversarial review for key decisions |
| Plan | Orchestrator-Subagent | Shared State (`.sdp/plan/`) | Workstream decomposition is bounded; plan artifacts persist in the store |
| Build | Orchestrator-Subagent | Generator-Verifier (implementer vs spec-reviewer) | Implementation is bounded per workstream; review cycle uses adversarial verification |
| Review | Generator-Verifier | Message Bus (findings to beads) | Review is inherently adversarial; findings flow into the beads bus for tracking |
| Eval | Shared State | Message Bus (metrics to dashboards) | Test outputs are evidence in the store; CI events flow through the bus |

### Phase-specific defaults for operators

**When in doubt, default to Orchestrator-Subagent.** It is the safest pattern for most SDP
operations. Upgrade to Agent Teams only when the task count exceeds 5 and workers genuinely
benefit from accumulated context. Upgrade to Generator-Verifier when the task is explicitly
about quality assurance or adversarial review.

---

## 9. Model Tiering (cost optimization)

Anthropic (and Claude Code) support model tiering through `CLAUDE_CODE_SUBAGENT_MODEL`:

| Tier | Model | Typical SDP role | When to use |
|------|-------|-----------------|-------------|
| Opus | Claude Opus | Main orchestrator, final reviewer, architect | Complex planning, final decisions, architectural analysis |
| Sonnet | Claude Sonnet | Implementer, spec-reviewer, qa, reviewer | Bulk execution, code generation, test writing |
| Haiku | Claude Haiku | Scout, discovery fan-out, lookups, analyst | Lightweight queries, repo scanning, file searches |

**SDP recommendation:**
- **Opus** for `orchestrator` + `architect` + `tech-lead` (complex reasoning).
- **Sonnet** for `implementer` + `qa` + `reviewer` + `spec-reviewer` (balanced cost/quality).
- **Haiku** for `scout` + `analyst` discovery fan-out (maximize breadth, minimize cost).

Estimated savings: 40-60% in tokens relative to uniform Opus deployment, with minimal quality
impact on implementation tasks.

---

## 10. Anti-Pattern Catalog

This section consolidates all anti-patterns across the five coordination patterns for quick
reference.

| Anti-pattern | Symptom | Pattern(s) | Fix |
|---|---|---|---|
| Orchestrator-bottleneck | Lead context exceeds 80K tokens; lead produces shallow output | Orchestrator-Subagent, Agent Teams | Summarize sub-outputs before assembling; switch to Shared State or Teams |
| Recursive dispatch | Sub-agent spawns sub-agent spawns sub-agent (3+ levels) | Orchestrator-Subagent | Flatten: orchestrator delegates to all sub-agents directly |
| False parallel | 3 "independent" sub-agents touch the same files | Orchestrator-Subagent (parallel) | Serialize or use isolated git worktrees per sub-agent |
| Bus-as-polling | Consumers poll filesystem in a loop instead of reacting to events | Message Bus | Use event triggers (git hooks, MCP subscriptions) |
| Team-for-two | Persistent team spawned for 1-2 tasks | Agent Teams | Use one-shot Orchestrator-Subagent instead |
| Silent verifier | Verifier sees generator's full reasoning context | Generator-Verifier | Pass only the artifact and evaluation rubric to the verifier |
| Implicit bus | Shared store used for real-time signaling (agents poll changes) | Shared State | Upgrade to Message Bus when real-time notification is needed |
| Schema drift | `.sdp/` artifact schemas evolve without versioning | Shared State | Add `schema_version` to all artifacts; maintain backward-compatible readers |

---

## 11. Parallel Dispatch Examples

### Example 1 -- parallel (good)

Three independent skill files need YAML frontmatter added. Each file is self-contained.

```
Dispatch:
  Sub-agent-1: add frontmatter to .agents/skills/llm-council.md
  Sub-agent-2: add frontmatter to .agents/skills/strataudit.md
  Sub-agent-3: add frontmatter to .agents/skills/review-readiness.md

Result: all three complete in parallel; wall-clock ~= max(single edit time).
```

### Example 2 -- sequential (good)

Fix a bug, then add a regression test, then update the changelog. Each step depends on the
previous one completing successfully.

```
Step 1: implementer fixes bug in user.go
Step 2: qa writes regression test (depends on fix being in place)
Step 3: ops updates CHANGELOG.md (depends on knowing the fix details)
```

### Example 3 -- parallel, subtle failure

"Fix 5 failing tests." Sounds independent, but tests share a common fixture file (`testdata/setup.json`).

```
Sub-agent-1 fixes test_a.go -- modifies setup.json
Sub-agent-2 fixes test_b.go -- also modifies setup.json
Result: merge conflict in setup.json; one agent's fix breaks the other's test.

Fix: serialize, or give each sub-agent its own worktree with a copy of the fixture.
```

---

## 12. References

### External

- [Anthropic -- Multi-agent coordination patterns](https://claude.com/blog/multi-agent-coordination-patterns)
  (canonical source for the five patterns)
- [ClaudeFast -- Sub-agent best practices](https://claudefa.st/blog/guide/agents/sub-agent-best-practices)
- [ClaudeFast -- Agent Teams](https://claudefa.st/blog/guide/agents/agent-teams)
- [AGENTS.md specification](https://agents.md/) -- multi-harness agent instruction standard

### Internal SDP

- `docs/plans/2026-04-16-f127-multi-harness-modernization-design.md` -- F127 design doc
- `docs/plans/2026-04-13-sdp-toolkit-vision-design.md` -- SDP toolkit pipeline vision
- `docs/plans/2026-04-13-sdp-skill-architecture-design.md` -- Skill architecture (intent routing)
- `docs/phases/DELIVERY.md` -- Delivery phase FSM description
- `docs/AGENT_TEAMS.md` -- Planned Agent Teams configuration for SDP
- `docs/reference/harness-integration.md` -- Harness status and invocation patterns
- `.agents/skills/llm-council.md` -- LLM council skill (uses Generator-Verifier pattern)
- `.agents/skills/parallel-dispatch.md` -- Parallel dispatch skill
- `AGENTS.md` -- Agent instructions (sub-agent dispatch policy section)
