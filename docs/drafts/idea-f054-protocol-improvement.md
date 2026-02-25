# Idea: F054 — Continuous Protocol Improvement

**Source:** [2026-02-25-agent-protocol-multifaceted-analysis.md](../plans/2026-02-25-agent-protocol-multifaceted-analysis.md)  
**Feature:** F054 (new)  
**Design:** `/design f054-protocol-improvement`

---

## Context

Multifaceted analysis identified protocol/prompts findings that don't fit existing F053 workstreams. F054 = "Continuous Protocol Improvement" — часть будущего роя (dream swarm). Skills, AGENTS.md, workflow handoffs.

---

## Deliverables → Workstreams

| WS | Title | Scope | Source |
|----|-------|-------|--------|
| **00-054-01** | @build: post-build bd close + batch /build | sdp/prompts/skills/build/SKILL.md, .cursor/skills/build/ | E1, E2 |
| **00-054-02** | @design: pre-draft check, bead-fixed, default-in-scope | sdp/prompts/skills/design/, .cursor/skills/design/ | E1, E4 |
| **00-054-03** | @review: handoff block | sdp/prompts/skills/review/, .cursor/skills/review/ | E4 |
| **00-054-04** | AGENTS.md: placement rules, «продолжай» convention | AGENTS.md | E1, E2 |
| **00-054-05** | /status F053 command | skill or sdp status —feature F053 | E2 |
| **00-054-06** | AGENTS.md / CLAUDE.md sync sdp ↔ sdp_lab | AGENTS.md, sdp/CLAUDE.md | — |

---

## Scope Files (per WS)

- **00-054-01:** sdp/prompts/skills/build/SKILL.md, .cursor/skills/build/SKILL.md
- **00-054-02:** sdp/prompts/skills/design/, .cursor/skills/design/
- **00-054-03:** sdp/prompts/skills/review/, .cursor/skills/review/
- **00-054-04:** AGENTS.md
- **00-054-05:** cmd/sdp-orchestrate/ or .cursor/commands/, docs/
- **00-054-06:** AGENTS.md, sdp/CLAUDE.md, sdp/AGENTS.md

---

## Dependencies

- 00-054-01..05: no cross-deps; can run in parallel
- 00-054-06 depends on 00-054-04 (content to sync)
- 00-054-05 may use sdp-orchestrate --next-action output format
