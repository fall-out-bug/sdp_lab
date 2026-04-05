# UX Workstream Delegation by Harness

**Date:** 2026-04-05
**Source:** [Council Synthesis](2026-04-05-ux-council-synthesis.md)
**Method:** LLM Council — each harness reviews and executes work suited to its strengths

---

## Delegation Principles

1. **Each harness works on what it knows best.** Cursor writes .cursorrules. Codex writes AGENTS.md. OpenCode ports hooks. Claude Code handles Go orchestrator and complex CLI.
2. **Doc-only tasks go to any harness.** Writing docs, skill file edits, and cleanup don't need specific IDE features.
3. **Go code tasks go to Claude Code.** sdp_lab is a Go project, Claude Code has full enforcement.
4. **Harness-specific artifacts go to that harness.** Best way to validate is to create from inside.
5. **Council review at milestones.** After each Phase completion, all harnesses review the result.

---

## Stream A: Product Identity (F097 -> F098 -> F102)

### F097: Product Truth and Activation Loop

| WS | Beads | Task | Harness | Rationale |
|---|---|---|---|---|
| 00-097-01 | sdplab-13 | Product Contract Document | **Any** (Codex recommended) | Pure docs. Codex proved strategic clarity in council review. |
| 00-097-02 | sdplab-14 | Activation Loop CLI (assess/try) | **Claude Code** | Go CLI implementation in sdp-plugin. Needs full build/test chain. |
| 00-097-03 | sdplab-15 | Metrics Baseline | **Claude Code** | Go telemetry code, schema definition. |

### F098: Simplified Progressive Disclosure

| WS | Beads | Task | Harness | Rationale |
|---|---|---|---|---|
| 00-098-01 | sdplab-16 | Three-Path CLAUDE.md | **Any** (OpenCode recommended) | Doc restructure. OpenCode showed strong UX critique in council. |
| 00-098-02 | sdplab-17 | Explicit @feature Modes | **Any** | Skill file edits. Any harness can edit SKILL.md. |
| 00-098-03 | sdplab-18 | Risk-Aware @review | **Any** | Skill file edits + config schema. |

### F102: Harness Quick Fixes (P1)

| WS | Beads | Task | Harness | Rationale |
|---|---|---|---|---|
| 00-102-01 | sdplab-25 | Cursor .cursorrules | **Cursor** | Cursor creates its own system prompt. Validate from inside. |
| 00-102-02 | sdplab-26 | Codex AGENTS.md | **Codex** | Codex creates its own agent instructions. Validate from inside. |
| 00-102-03 | sdplab-27 | OpenCode Hook Port | **OpenCode** | OpenCode ports its own hooks from sdp_lab. Validate from inside. |
| 00-102-04 | sdplab-28 | Fallback Doc + commands.yml | **Any** | Docs + YAML. Any harness. |

---

## Stream B: Trust and Reliability (F101 + F103 + F104)

### F101: Write Boundaries

| WS | Beads | Task | Harness | Rationale |
|---|---|---|---|---|
| 00-101-01 | sdplab-24 | Write Plan Emission | **Any** (OpenCode recommended) | Skill file edits. OpenCode proposed this spec — let them implement. |

### F103: Resilient Orchestrator (P1)

| WS | Beads | Task | Harness | Rationale |
|---|---|---|---|---|
| 00-103-01 | sdplab-29 | Human-Readable Output | **Claude Code** | Go code in cmd/sdp-orchestrate. Needs build + test. |
| 00-103-02 | sdplab-30 | Checkpoint Resilience | **Claude Code** | Go code in internal/orchestrate. Critical error handling. |
| 00-103-03 | sdplab-31 | Inline Progress + Resume | **Claude Code** | Go code. Integrates with orchestrate loop. |
| 00-103-04 | sdplab-32 | Rename @deploy to @ship | **Any** | Skill file rename + doc updates. |

### F104: Escape Hatches (P1)

| WS | Beads | Task | Harness | Rationale |
|---|---|---|---|---|
| 00-104-01 | sdplab-33 | @feature --design-only + @review post-retry | **Any** | Skill file edits. |
| 00-104-02 | sdplab-34 | sdp reset + Recovery Sections | **Claude Code** (reset) + **Any** (recovery docs) | Reset is Go CLI. Recovery sections are skill file edits. |

---

## Stream C: Adoption and Discipline (F100 + F099)

### F100: Release Discipline Gates

| WS | Beads | Task | Harness | Rationale |
|---|---|---|---|---|
| 00-100-01 | sdplab-22 | Reference Integrity CI Gate | **Claude Code** | Shell script + GitHub Actions YAML. Needs CI validation. |
| 00-100-02 | sdplab-23 | One-Time Reference Cleanup | **Any** (parallelize across harnesses) | Many small file fixes. Split across harnesses for speed. |

### F099: Brownfield Safe Overlay

| WS | Beads | Task | Harness | Rationale |
|---|---|---|---|---|
| 00-099-01 | sdplab-19 | Safe Install | **Any** | Shell scripts. Any harness can edit install.sh. |
| 00-099-02 | sdplab-20 | Disabled Gates on Adoption | **Claude Code** | Go code in sdp-plugin gate logic. |
| 00-099-03 | sdplab-21 | Adoption Guide Document | **Codex** | Pure docs. Codex has best brownfield perspective from council. |

---

## Summary: Harness Workload

| Harness | Primary | Can-Do | Total |
|---|---|---|---|
| **Claude Code** | 00-097-02, 00-097-03, 00-099-02, 00-100-01, 00-103-01, 00-103-02, 00-103-03 | All "Any" tasks | 7 primary + 15 any |
| **Cursor** | 00-102-01 | Doc tasks | 1 primary |
| **OpenCode** | 00-098-01, 00-101-01, 00-102-03 | Doc tasks | 3 primary |
| **Codex** | 00-097-01, 00-099-03, 00-102-02 | Doc tasks | 3 primary |
| **Any** | 00-098-02, 00-098-03, 00-099-01, 00-100-02, 00-101-01, 00-102-04, 00-103-04, 00-104-01, 00-104-02 | — | 9 tasks |

---

## Council Review Points

After each phase completion, all 4 harnesses review the result:

| Phase | Review Trigger | What to Review |
|---|---|---|
| Phase 1 (Honest) | F097 + F098 + F100 done | Does the product story make sense? Are docs consistent? |
| Phase 2 (Safe) | F101 + F099 done | Can each harness try SDP safely on a brownfield project? |
| Phase 3 (Recoverable) | F103 + F104 done | Does recovery work from each harness? |
| Phase 4 (Harnesses) | F102 done | Does each harness have a reliable minimum experience? |

Each review produces a short document in `docs/plans/YYYY-MM-DD-phase-N-council-review.md`.

---

## How to Delegate

### For Claude Code tasks (Go code)
```bash
# In Claude Code session:
@build 00-097-02   # Activation Loop CLI
@build 00-103-01   # Human-Readable Orchestrator Output
```

### For Cursor tasks
```bash
# In Cursor session:
# Read: docs/plans/2026-04-05-ux-council-synthesis.md (SPEC-02 Phase 1)
# Read: docs/workstreams/backlog/00-102-01.md
# Execute: create .cursorrules per acceptance criteria
```

### For OpenCode tasks
```bash
# In OpenCode session:
# Read: docs/plans/2026-04-05-ux-council-synthesis.md (SPEC-07)
# Read: docs/workstreams/backlog/00-101-01.md
# Execute: add write plan sections to all stateful skills
```

### For Codex tasks
```bash
# In Codex session:
# Read: docs/plans/2026-04-05-ux-council-synthesis.md (SPEC-00)
# Read: docs/workstreams/backlog/00-097-01.md
# Execute: write PRODUCT_CONTRACT.md
```

---

## Execution Order Within Each Stream

### Stream A (can start immediately)
```
00-097-01 (Codex: Product Contract)
    |
    +-> 00-098-01 (OpenCode: CLAUDE.md restructure)
    |       |
    |       +-> 00-102-01 (Cursor: .cursorrules)
    |       +-> 00-102-02 (Codex: AGENTS.md)
    |       +-> 00-102-03 (OpenCode: hook port)
    |       +-> 00-102-04 (Any: fallback doc)
    |
    +-> 00-097-02 (Claude: assess/try CLI)
            |
            +-> 00-097-03 (Claude: metrics)

Parallel: 00-098-02 (Any), 00-098-03 (Any) — no deps
```

### Stream B (can start immediately)
```
00-101-01 (OpenCode: write boundaries) — no deps
00-103-01 (Claude: human-readable output) — no deps
00-103-02 (Claude: checkpoint resilience) — no deps
00-103-03 (Claude: inline progress) — no deps
00-103-04 (Any: @deploy rename) — no deps
00-104-01 (Any: --design-only + post-retry) — no deps
00-104-02 (Claude + Any: reset + recovery) — no deps
```

### Stream C (can start immediately)
```
00-100-01 (Claude: CI gate) — no deps
    |
    +-> 00-100-02 (Any: cleanup)

00-099-01 (Any: safe install) — no deps
    |
    +-> 00-099-03 (Codex: adoption doc)

00-099-02 (Claude: disabled gates) — depends on 00-097-02
```
