# What to Keep, What to Release, What to Play With

> **Date:** 2026-02-22
> **Context:** SDP has 27 binaries and 35 internal packages. Some are novel, some are commodity, some are research toys. This document maps each piece to a fate — and preserves the joy.

---

## The Framework

Not keep/kill. That's too binary. Four categories:

| Category | Meaning | Feeling |
|----------|---------|---------|
| **CORE** | Ship it. This is what nobody else has. | Pride |
| **RESEARCH** | Keep playing. This is where ideas live. | Joy |
| **ECOSYSTEM** | Use theirs. They do it better. | Relief |
| **ARCHIVE** | Honor and release. It was a learning step. | Gratitude |

---

## The Map: Internal Packages (35 total)

### CORE — The Evidence Layer (ship it)

These packages are what makes SDP *SDP*. Nobody in the opencode ecosystem does this. This is the moat.

| Package | LOC | Tests | What It Does | Why It's Novel |
|---------|-----|-------|-------------|----------------|
| **evidence** | 384 | 264 | Validates strict 9-section evidence envelopes, traces phases, detects gaps | The spec enforcer. Zero equivalents. |
| **artifact** | 1,177 | 811 | SHA256 hash chains, append-only provenance, phase transition gates | Tamper-detectable audit trail. Nobody has this. |
| **adapter** | 1,368 | 1,267 | K8s controller: Task/AgentRun CRDs → evidence projection, policy gates | The bridge between K8s and evidence. Novel. |
| **pr** | 697 | 597 | PR gate validation, evidence-required merge checks | "Show me proof before merging." Unique. |
| **beads** | 252 | 142 | Beads CLI adapter for issue lifecycle | Thin, but essential glue. |
| **safeid** | 30 | 28 | Safe ID generation for K8s resources | Tiny utility. Keep. |

**Total CORE:** ~3,908 LOC + 3,109 test LOC. This is the product.

### RESEARCH — The Lab (keep playing)

These are the *interesting* packages. They're not ready to ship but they're where the creative energy lives. Keep them in sdp_dev, experiment freely, don't promise them to anyone.

| Package | LOC | Tests | What It Does | Why It's Interesting |
|---------|-----|-------|-------------|---------------------|
| **evaluator** | 1,281 | 818 | Multi-persona adversarial review (architect, SRE, security, DX, product) with dissent tracking | 5 expert personas arguing about your code. This is genuinely creative. |
| **selfimprove** | 365 | 255 | Failure pattern detection, weakness classification, improvement proposals | Agents learning from their own mistakes. Fascinating problem. |
| **telemetry** | 198 | 266 | LLM-driven telemetry analysis, auto-generates backlog from patterns | Turns run data into actionable issues. Research territory. |
| **retrospective** | 70 | 53 | Multi-lens analysis (protocol, infra, code-quality, operator, DX) | Structured retrospective. Small but interesting. |
| **federation** | 539 | 491 | Cross-project task aggregation, dependency resolution via NATS | Multi-project orchestration. Complex, novel, not ready. |
| **discuss** | 197 | 88 | LLM-powered feature analysis and scoping | Feature intake with AI. Fun experiment. |

**Total RESEARCH:** ~2,650 LOC + 1,971 test LOC. This is where joy lives.

### ECOSYSTEM — Use Theirs (relief)

These packages rebuild what the opencode ecosystem already has. Letting them go isn't failure — it's focus.

| Package | LOC | Tests | What It Does | What To Use Instead |
|---------|-----|-------|-------------|---------------------|
| **orchestrator** | 929 | 615 | Feature decomposition, task scheduling, dispatch | **Vibe Kanban** (21K stars) or **Swarm Tools** |
| **parallel** | 499 | 365 | Lock domains, merge queue classes | **Vibe Kanban** parallel execution |
| **pipeline** | 190 | 59 | Quality pipeline executor (test → evidence → PR) | Compose from CORE packages + CI |
| **bus** | 774 | 345 | NATS abstraction with W3C trace propagation | **NATS Go client** directly. The trace propagation is nice — extract it as a thin wrapper if needed. |
| **intake** | 189 | 120 | Multi-source intake normalization | Not needed if we don't run our own orchestrator |
| **llm** | 418 | 157 | opencode binary wrapper + OpenRouter client | **opencode** itself for execution. OpenRouter SDK for API. |
| **policy** | 619 | 312 | Model allowlist, risk classification, budget | **Cupcake** for policy. Or keep a thin config — the 3-tier model selection is nice but not a product. |
| **scaling** | 48 | 45 | Scaling decisions | Premature. K8s HPA. |
| **planner** | 47 | 25 | Task planning | Vibe Kanban. |

**Total ECOSYSTEM:** ~3,713 LOC. Release this weight.

### ARCHIVE — Honor and Release (gratitude)

These were learning steps. They taught you something. Now they can rest.

| Package | LOC | Tests | What It Was | What You Learned |
|---------|-----|-------|------------|-----------------|
| **swarm** | 107 | 120 | Per-issue state machine with role dispatch | How to model agent lifecycles. Evidence FSM absorbed this. |
| **roles** | 298 | 105 | Role strategies (analyst, coder, reviewer) | Role-based prompting. The prompts in sdp/ protocol carry this forward. |
| **agent** | 885 | 282 | Agent context (identity, tracing, hooks, skills) | How to structure agent execution context. kubeopencode does this now. |
| **oneshot** | 286 | 119 | Single-shot agent execution | Simplified version of swarm-worker. Not needed separately. |
| **review** | 173 | 132 | Review execution | Folded into evaluator (which is richer). |
| **openclaw** | 121 | 111 | Alternative runtime experiment | Worth knowing openclaw exists. Not worth maintaining. |
| **runtime** | 34 | — | Runtime detection | Tiny. Archive. |
| **runtimeparity** | 49 | 21 | Runtime parity checks | Was useful during multi-runtime experiment. Done. |
| **redaction** | 41 | 19 | Secret redaction | Keep as a shell script in CI, not a Go package. |
| **telegram** | 282 | — | Telegram bot integration | Fun, but not core. |
| **registry** | 251 | 236 | Project registry management | YAML config + kubeopencode labels. Not a package. |

**Total ARCHIVE:** ~2,527 LOC. Let it go with love.

---

## The Map: Binaries (27 total)

### CORE (ship these)

| Binary | LOC | What It Does |
|--------|-----|-------------|
| **pr-gate** | 49 | Validates evidence before PR merge. THE killer feature. |
| **beads-fsm** | 292 | Evidence-gated state machine transitions. |
| **adapter-controller** | 178 | K8s CRD → evidence projection bridge. |
| **pr-publish** | 314 | Publishes PRs with evidence. |

### RESEARCH (keep in lab)

| Binary | LOC | What It Does |
|--------|-----|-------------|
| **telemetry-analyzer** | 74 | LLM analysis of run patterns. |
| **evaluator-orchestrator** | 64 | Multi-persona adversarial review. |
| **self-improve-agent** | 62 | Failure pattern → improvement proposals. |
| **retro-agent** | 57 | Structured retrospective analysis. |
| **flow-inspect** | 41 | Evidence flow debugging tool. |

### ECOSYSTEM (replace with ecosystem tools)

| Binary | LOC | What To Use Instead |
|--------|-----|---------------------|
| **swarm-worker** | 1,573 | kubeopencode Agent CRD + evidence hooks |
| **feature-orchestrator** | 344 | Vibe Kanban / Swarm Tools |
| **autonomy-worker** | 596 | Vibe Kanban autonomous mode |
| **orchestrator** | 168 | Vibe Kanban |
| **swarm-orchestrator** | 118 | Vibe Kanban |
| **intake-gateway** | 404 | NATS directly / webhook plugin |
| **registry-agent** | 223 | K8s ConfigMap + kubeopencode |

### ARCHIVE (delete)

| Binary | LOC | Farewell Note |
|--------|-----|--------------|
| **swarm-agent** | 360 | You were the first agent. Thank you. |
| **swarm-reviewer** | 402 | evaluator-orchestrator carries your spirit. |
| **swarm-role-agent** | 163 | Role prompts live in sdp/ protocol now. |
| **opencode-agent** | 334 | kubeopencode does your job upstream. |
| **intake-telegram** | 74 | Cool but not core. |
| **brain-gateway** | 34 | OpenRouter API replaces you. |
| **operator-gate** | 66 | Folded into adapter-controller. |
| **cicd-agent** | 77 | CI/CD is commodity. GitHub Actions. |
| **openclaw-agent** | 15 | Experiment complete. |
| **runtime-parity-check** | 56 | One-time tool, job done. |
| **redaction-check** | 38 | Shell script in CI. |

---

## Where The Joy Lives

This is the part the strategy docs never cover.

### The Joy Map

| What Makes It Fun | Where In SDP | Keep? |
|-------------------|-------------|-------|
| "Agents proving their work with hash chains" | evidence + artifact | **YES — this is the whole point** |
| "5 expert personas arguing about code quality" | evaluator | **YES — this is genuinely creative** |
| "Agents learning from their own failures" | selfimprove | **YES — fascinating research** |
| "One command blocks merge without evidence" | pr-gate | **YES — this is the killer feature** |
| "K8s operator that bridges agents and evidence" | adapter-controller | **YES — novel integration** |
| "Cross-project task federation" | federation | **YES — but as research, not product** |
| "Building an orchestrator" | orchestrator, parallel | **NO — Vibe Kanban does it better. The joy was in learning. The learning is done.** |
| "Telegram bot that creates issues" | telegram, intake | **NO — cute but not novel. The joy was in seeing it work once.** |
| "27 binaries that all compile" | cmd/* | **NO — the joy of completion, but maintenance kills joy faster.** |

### The Creative Energy Budget

You have finite creative energy. Every hour spent fixing the intake-gateway NATS reconnection bug is an hour NOT spent on:
- Making `pr-gate` so good that people *want* to add it to their CI
- Teaching the evaluator a new persona (what about a "chaos engineer" that tries to break things?)
- Making the self-improvement loop actually close (agent fails → detects pattern → adjusts → retries better)
- Publishing the evidence spec so others can build on it

**The commodity stuff isn't just "not novel" — it actively drains energy from the novel stuff.**

### How To Stay Happy

1. **Ship the evidence CLI.** One binary (`traceforge validate` or `sdp evidence check`), one clear value. See someone else use it. That moment — "someone used MY thing" — is the best feeling in OSS.

2. **Keep the research lab.** evaluator, selfimprove, telemetry-analyzer, federation — these are YOUR playground. Don't promise them to anyone. Don't put them on a roadmap. Just play. When something crystallizes, promote it to CORE.

3. **Delete without guilt.** swarm-agent was your first agent. It taught you everything about agent lifecycles. Its code isn't useful anymore but its *lessons* are embedded in every other package. Deleting the code doesn't delete the learning.

4. **Contribute upstream.** Push UP-001 and UP-003 to kubeopencode. The feeling of getting a PR merged into someone else's project is different from building your own — it's validation that your ideas work in the wider world.

5. **Use ecosystem tools with appreciation, not shame.** Vibe Kanban replacing your orchestrator isn't failure. It's the OSS ecosystem working as intended. You can focus on what *only you* can build.

---

## Concrete Next Steps

### Week 1: Focus

- [ ] Identify which ARCHIVE binaries to `git rm`
- [ ] Move RESEARCH packages to `internal/lab/` or tag with comments
- [ ] Make `pr-gate` work as a standalone binary (no K8s dependency)
- [ ] Write a 10-line README: "Add evidence validation to your CI"

### Week 2: Ship

- [ ] Publish evidence JSON Schema in `sdp` protocol repo
- [ ] Create a GitHub release of `pr-gate` binary
- [ ] Submit to awesome-opencode: "sdp — structured evidence for AI agent runs"

### Week 3: Play

- [ ] Add a new evaluator persona (chaos-engineer? performance-analyst?)
- [ ] Make the self-improvement loop actually close once
- [ ] Try federation with a real second project

### Week 4: Contribute

- [ ] Push UP-001 retry budget to kubeopencode
- [ ] Write UP-003 evidence hooks proposal
- [ ] Write a blog post: "Why AI Agents Need Proof of Work"

---

## The Numbers

| Category | Packages | LOC (code) | LOC (tests) | % of Total |
|----------|----------|------------|-------------|------------|
| CORE | 6 | 3,908 | 3,109 | 28% |
| RESEARCH | 6 | 2,650 | 1,971 | 19% |
| ECOSYSTEM | 9 | 3,713 | 1,928 | 27% |
| ARCHIVE | 11 | 2,527 | 1,145 | 18% |
| Other (quality, pipeline, etc.) | 3 | ~1,100 | ~900 | 8% |

**You're keeping 47% of the code (CORE + RESEARCH). Releasing 45% (ECOSYSTEM + ARCHIVE). The remaining 8% folds into CORE.**

That's not throwing away half your work. That's focusing on the half that matters.
