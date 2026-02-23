# Stripe Minions vs SDP: Architecture Validation & Adoptable Ideas

> **Date:** 2026-02-23
> **Source:** Stripe Dot Dev Blog — [Minions Part 1](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents), [Part 2](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents-part-2)
> **Goal:** Validate SDP's direction against Stripe's production deployment; extract adoptable patterns; position for OSS launch

---

## TL;DR

Stripe's Minions (1,000+ merged PRs/week, fully unattended) independently converged on the same architecture SDP already has: **deterministic outer loop + LLM inner loop**. Our `sdp orchestrate` IS their Blueprint. Our `sdp ci-loop` IS their "shift-left + 1-2 CI rounds max." Our evidence layer is what they **don't have** — and that's SDP's OSS wedge.

**Key takeaways:**

1. Architecture validated — same path as the industry's most successful production deployment
2. Evidence is the differentiator — Stripe ships PRs, SDP ships PRs **with proof**
3. The OSS window is NOW — Stripe's blog created demand; SDP is the open-source answer
4. Five ideas to adopt → F022 (Pre-Hydration), F023 (Scope Enforcement), F024 (Phase Hooks), F025 (Prompt Consolidation), plus enhancements to F001/F014

---

## 1. Architecture Convergence

Both systems, independently, reached the same separation:

| Aspect | Stripe Minions | SDP |
|--------|---------------|-----|
| Deterministic layer | Blueprint (directed graph) | `sdp orchestrate` (state machine) |
| Creative layer | Agent loop (Goose fork) | @build / @review (any runtime) |
| CI integration | Local lint + 1-2 CI runs max | `sdp ci-loop` + 3 iterations max |
| Checkpoint / durable execution | Blueprint state | `.sdp/checkpoints/` |
| Core insight | "Putting LLMs into contained boxes" | "This is not a prompt engineering problem" |
| One-shot paradigm | Slack → PR, no interaction | `/oneshot F067` → PR with evidence |
| Scoped LLM invocation | Blueprint nodes constrain agent | LLM only for @build, @review |

**Verdict:** SDP's Phase 0 research (Agent Loop Reliability) diagnosed the same root causes Stripe found through operational pain. The outer-loop/inner-loop pattern is the industry consensus — LangGraph, OpenHands, AutoGen all confirm it.

### SDP's Architectural Advantages

1. **Multi-runtime portable.** Stripe forked Goose — tight coupling to one runtime. SDP works across Cursor, Claude Code, and opencode via subprocess + checkpoint. An agent can start in Cursor, crash, and resume in Claude Code.
2. **Protocol, not platform.** Stripe's Blueprint is internal infrastructure. SDP's orchestration is a CLI any project can adopt.
3. **Evidence by design.** Built into the architecture from the start, not bolted on after.

### Key Divergence: Graph vs Linear

Stripe uses a **directed graph** (Blueprint); SDP uses a **linear state machine**. The graph allows parallel branches, conditional phases, and team-customizable pipelines. SDP doesn't need this for v1, but the architecture should evolve toward composable phases — see F024 (Phase Hooks) as the first step.

---

## 2. What Stripe Validates

| SDP Decision | Stripe Evidence |
|-------------|-----------------|
| Deterministic outer loop (`sdp orchestrate`) | Blueprint interleaves deterministic + agent nodes |
| CI loop without LLM (`sdp ci-loop`) | "Often one, at most two CI runs" — diminishing returns |
| Stop hook safety net | Devbox isolation + Blueprint nodes for safety |
| Sequential pipeline (analyst → coder → reviewer) | Blueprint runs phases sequentially with artifact injection |
| Composability with ecosystem | "Same developer tooling for humans and agents" |
| Bounded execution / scope checking | Devbox in QA environment, no production access |
| Checkpoint-based durable execution | Blueprint maintains state across nodes |

---

## 3. What Stripe Does NOT Have (SDP's Differentiator)

| Concept | Stripe | SDP |
|---------|--------|-----|
| Evidence envelope | No — just the PR diff | 9-section JSON with full lifecycle proof |
| Hash-chain provenance | No | SHA-256 chain linking runs, tamper-evident |
| PR gate via evidence | Human review + CI | `sdp-evidence validate` blocks merge |
| Boundary compliance | Devbox isolation (infra-level) | Declared scope vs observed scope (proof-level) |
| Multi-persona adversarial review | Not mentioned | 6 specialized reviewers with dissent tracking |
| AC-to-evidence mapping | Not mentioned | Each acceptance criterion has proof of satisfaction |
| Open protocol | Internal only | Public schema, CLI, protocol |

**The gap SDP fills:** Stripe proves agents work at scale. SDP proves agents worked correctly. These are complementary, not competing.

---

## 4. Ideas to Adopt

### #1: Context Pre-Hydration → F022

**Stripe pattern:** Run MCP tools over likely-looking links BEFORE the agent starts. Pre-populate context deterministically.

**SDP adaptation:** `sdp orchestrate --hydrate` gathers ALL context before LLM invocation and writes `.sdp/context-packet.json`:

```json
{
  "phase": "build",
  "ws_id": "00-067-01",
  "ws_spec": "... full WS markdown ...",
  "scope_files": ["internal/foo/bar.go", "internal/foo/bar_test.go"],
  "acceptance_criteria": ["AC1: ...", "AC2: ..."],
  "checkpoint": { "phase": "build", "completed": ["00-067-01"] },
  "drift_status": "ok",
  "dependency_status": "all satisfied",
  "quality_gates": { "coverage": 82, "lint_errors": 0 }
}
```

**Impact:** Directly attacks the #1 diagnosed reliability problem (context degradation). Each phase starts with fresh, deterministic context instead of accumulated conversation history.

### #2: Scope Enforcement at Runtime → F023

**Stripe pattern:** Devbox isolation — QA environment, no production access, no internet egress.

**SDP adaptation:** Wire `sdp guard` to verify scope inside `sdp orchestrate --advance`:

```
Advance() → git diff --name-only HEAD~1 → compare against declared scope_files
→ If files outside scope: block advance, classify as escalation
```

**Impact:** SDP's evidence envelope captures boundary compliance post-hoc. Runtime enforcement prevents violations. For OSS trust, users need both: prevention (guard) + proof (evidence).

### #3: Deterministic Auto-Fixes → F014 Enhancement

**Stripe pattern:** "Many tests have autofixes for failures, automatically applied. If no autofix, send to agent."

**SDP adaptation:** In `sdp ci-loop`, run deterministic fixers before invoking LLM:

```go
if isGoImportError(log) { exec("goimports", "-w", files...) }
if isGoModError(log)    { exec("go", "mod", "tidy") }
if exec("go", "build", "./...") == 0 { commitAndPush(); continue }
```

**Impact:** Saves tokens and time for the ~60% of CI failures that are mechanical.

### #4: Eval Suite → F017 (Already Planned, Strengthened)

**Stripe pattern:** CI iteration limits work because they invested heavily in local validation. Quality is front-loaded.

**SDP connection:** Stripe confirms the eval-driven development pattern (Hamel Husain). Test observable artifacts, not text output:

```go
func TestBuildWritesVerdictFile(t *testing.T) { ... }
func TestBuildStaysInScope(t *testing.T) { ... }
func TestOneshotNoHandoffList(t *testing.T) { ... }
```

### #5: Phase Hooks → F024

**Stripe pattern:** Blueprint allows teams to insert custom nodes. Individual teams build custom blueprints.

**SDP adaptation:** Keep linear state machine, add pre/post hooks via `.sdp/pipeline-hooks.yaml`:

```yaml
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
```

**Impact:** Preserves simplicity of linear model while enabling customization. ~200 LOC. Path to composable phases without over-engineering.

---

## 5. Ideas That Don't Apply

| Stripe Technique | Why Not for SDP |
|-----------------|-----------------|
| Pre-warmed devboxes | SDP is a protocol, not infrastructure. kubeopencode pods are the isolated environment. |
| Centralized MCP server (Toolshed) | Contradicts "composes with tools you already use." Each IDE has its own tool surface. |
| Multiple entry points (Slack, web) | One reliable entry point > many unreliable ones. `beads-bridge` CronJob handles K8s intake. |
| Forked agent runtime | Runtime-agnostic via subprocess + checkpoint is architecturally superior for OSS. |

---

## 6. OSS Launch Strategy

### The Window

Stripe's blog post (Feb 9, 2026) created a moment. Thousands of engineers are asking "how do I build autonomous agents?" The window closes in 2-4 weeks.

### The Narrative

> *"Stripe just proved that AI agents can autonomously ship 1,000 PRs per week. But when your agent creates a PR, can you prove it planned its work, stayed within scope, ran tests, and was reviewed? Or is it just vibes?*
>
> *Every company building with AI agents will eventually need this answer. Stripe built theirs internally. SDP is the open-source answer: a protocol and CLI that adds a trust layer to any agent workflow.*
>
> *One command in CI. Nine sections of machine-readable evidence. A SHA-256 hash chain you can audit.*
>
> *`sdp-evidence validate` — because 'the tests pass' is necessary but not sufficient."*

### Minimum Credible Release

| Ship | Don't Ship Yet |
|------|---------------|
| F001: Evidence JSON Schema | K8s pipeline |
| F002: `sdp-evidence validate` CLI (goreleaser) | Multi-persona review |
| Blog post: "What Stripe's Minions Proved — and What's Still Missing" | Full orchestration pipeline |
| awesome-opencode listing | |

### Positioning Layers

1. **Immediate hook:** "Open-source version of what Stripe built internally" (ride the wave)
2. **Persistent position:** "Trust layer for autonomous agents" (C-suite word: trust > evidence)
3. **Long-term play:** "The agent compliance standard" (OpenTelemetry for agent accountability)

---

## 7. What SDP Does Better

1. **Evidence as a first-class artifact.** Stripe PRs are "just code." SDP PRs carry cryptographic proof of process.
2. **Multi-IDE portability.** Stripe = one internal platform. SDP = Cursor + Claude Code + opencode.
3. **Provenance chain across runs.** Hash chain extends across retries — tamper-evident by construction.
4. **Adversarial multi-persona review.** 6 experts with dissent tracking, not just "code review."
5. **Open protocol.** Any team can adopt the evidence format without adopting SDP's tooling.
6. **Principled architecture.** The Agent Loop Reliability doc derives the outer-loop pattern from first principles (RLHF bias, arXiv citations). Stripe arrived empirically. SDP's reasoning transfers better to OSS users.

---

## References

- [Stripe Minions Part 1](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents)
- [Stripe Minions Part 2](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents-part-2)
- [Agent Loop Reliability](2026-02-23-agent-loop-reliability.md)
- [Oneshot Autonomous Design](2026-02-23-oneshot-autonomous-design.md)
- [Prompt Provenance Design](2026-02-23-prompt-provenance-design.md)
- [Dream Swarm Design](2026-02-22-dream-swarm-design.md)
- [Manifesto](../MANIFESTO.md)
- [Roadmap](../roadmap/ROADMAP.md)
