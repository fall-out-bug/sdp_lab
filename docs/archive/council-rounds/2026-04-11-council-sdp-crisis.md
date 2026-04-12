# LLM Council Report: SDP Crisis — Governance Architecture

**Date:** 2026-04-11  
**Rounds:** 2  
**Consensus:** PARTIAL  
**Decision Owner:** PENDING (user)

## Models

| Role | Model |
|------|-------|
| Architect | claude-sonnet-4-6 (via codex:codex-rescue) |
| Critic | google/gemini-3.1-pro-preview |
| Technician | deepseek/deepseek-v3.2 |
| Philosopher | moonshotai/kimi-k2 |
| Pragmatist | minimax/minimax-m2.7 |
| Engineer | xiaomi/mimo-v2-pro |

---

## Recommendations (для Decision Owner)

### RESOLVED — consensus ≥80%, no vetoes

**R1: "Governance over opaque subprocess" — нерабочая модель (6/6)**

Consensus. opencode получает prompt, возвращает exit code + stdout. SDP не видит tool calls, не может остановить агента, не знает текущую фазу. Evidence = неструктурированный stdout. Фазы и гейты — только конвенция, trivially spoofable.

Action required: не строить на этой модели. Любой wiring, который использует openCodeInvoker без изменений, не решает проблему.

---

**R2: agentloop архитектурно корректен, операционально неполон (6/6)**

Consensus. FSM, GateEngine, PhaseRouter, EvidenceAccumulator — правильные абстракции. Проблема в том, что это isolated library с нулём caller'ов вне пакета. Текущий статус: design artifact, не operational component.

Action required: не переписывать agentloop. Подключать через wiring в ExecutorBridge.

---

**R3: 128 unit tests ≠ интеграционное покрытие (6/6)**

Consensus. StubGateway делает тест-сьют нерелевантным для реального LLM. Дают ложную уверенность.

Action required: написать один real failing integration test:
```
sdp run --card=<real-card>
  → реальный LLM call (не StubGateway)
  → agentloop harness с real ModelGateway
  → evidence накоплена в .sdp/evidence/<card>.json
  → gate check → phase transition
```
Пока этот тест не проходит — никакой другой тест не даёт уверенности в интеграции.

---

**R4: CANONICAL_SDP_PIPELINE.md должен быть закрыт или продвинут (5/6)**

Quasi-consensus. "Proposed" статус = authority vacuum. Каждое решение, принятое против него, имплицитно provisional.

Action required: либо промоутнуть до Accepted с конкретным маппингом на agentloop constructs, либо архивировать и считать agentloop код каноническим spec. Два конкурирующих авторитета ведут к drift.

---

**R5: EvidenceAccumulator tool-only constraint нужно сохранить (4/6)**

Majority. Tool outputs (не self-report) — единственное правильное основание для evidence. Любая попытка включить LLM narrative в evidence коллапсирует audit trail в self-attestation.

Action required: при wiring OpenRouterGateway убедиться, что evidence capture происходит в harness layer, а не внутри model response handler.

---

**R6: PhaseRouter + GateEngine должны быть HARD preconditions, не convention (4/6, architect domain veto)**

Raised by Architect. GateEngine 5s circuit breaker намекает на concurrent gates, а не blocking checkpoints. Неясно: gate failure = rollback, halt, или escalation? И что из этого enforced в коде vs что convention?

Action required: до написания `sdp run` CLI — задокументировать и протестировать gate-blocked phase transitions под `-race`. Иначе система выглядит governed, но не является.

---

### DEFERRED — нужно решение Decision Owner

**D1: Путь A (runtime) vs Путь B (protocol)**

Split 3:3 (с философом в minority).

**Путь A: SDP becomes the runtime**
- Support: Technician, Engineer, Pragmatist
- Аргументы: полный контроль, достижимо за недели, доказуемо работающее
- Риски: SDP отвечает за все инструменты, портирование, совместимость, sandboxing

**Путь B: Protocol enforcement (JSON-RPC stream)**
- Support: Philosopher (explicit), Pragmatist (conditionally)
- Аргументы: SDP как IETF (protcol), не GNU/Linux (runtime); harnesses свободны
- Риски: требует adoption от сторонних harnesses (6-12 месяцев minimum); spec не существует

**Позиция Pragmatist:** Path A is pragmatic for MVP; extract protocol once you understand the problem space.

**Позиция Engineer:** Path B — пока "JSON-RPC stream" это 2-sentence idea без схемы, версионирования, error recovery.

Decision Owner decision needed: **accept Path A for v1, extract protocol later** OR **invest in protocol spec first**.

---

**D2: Реалистичный timeline**

| Scope | Estimate |
|-------|---------|
| MVP с StubGateway (уже готово) | Done |
| OpenRouterGateway + wiring | 1-2 недели |
| Production bash tool (PTY, timeout, sandboxing) | 2-4 недели |
| Full integration test passing | 3-6 недель |
| Stable (4-8 weeks оценка) | Technician: "multi-month" для production-grade |

Decision Owner decision needed: каков приемлемый timeline? Что считать "done"?

---

### UNRESOLVED — split vote, no consensus

**U1: EvidenceAccumulator и intent gap (3:3)**

- Philosopher, Pragmatist: tool outputs ≠ reasoning trace; governance без intent = аудит постфактум, не enforcement
- Technician, Architect, Engineer, Critic: для v1 tool outputs достаточно; intent tracking = v2

Crux: нужен ли reasoning trace для enforcement SDP фаз в v1, или post-hoc evidence достаточно?

---

## Minority Reports

**Philosopher (moonshotai/kimi-k2) — verbatim:**

> "Making SDP the runtime is a power grab disguised as architecture. Owning the runtime gives you enforcement today, but it also makes SDP the bottleneck for every future capability (new model, new modality, new OS). Ask whether you are ready to become the GNU/Linux of agent infrastructure—maintaining tool ports, GPU drivers, sandboxing, etc.—instead of the IETF of agent governance: small, stubborn protocol that outlives implementations."

> "A runtime pivot embeds one worldview; a protocol pivot keeps the space open for rival harnesses. The artifact flips from 3-4 days to 2-8 weeks without asking: 'Are we solving enforcement or lock-in?'"

> "If the single integration test passes with Claude Code or Codex through a protocol stream, Path B is already viable. The council risks confirmation bias by writing the test only inside Path A assumptions."

**Critic (google/gemini-3.1-pro-preview):**

> "Replicating the reliability, context window management, and error-handling robustness of dedicated agents is a different product entirely. You are trading a governance problem for a distributed systems and prompt engineering nightmare."

> "A real bash tool requires stateful session management, PTY handling, and infinite loop protection, while edit_file requires robust diff parsing to handle inevitable LLM syntax hallucinations. Your MVP tools will catastrophically fail on their first real-world mid-sized codebase refactor."

---

## Round Convergence

| Round | Resolved | New | Notes |
|-------|----------|-----|-------|
| R1 (blind review) | 3 | 4 | governance broken, agentloop sound, 128 tests insufficient + intent gap, path A/B split, timeline |
| R2 (v2 review) | 3 | 2 | timeline updated, pipeline blocker confirmed; gate enforcement + protocol spec gap surfaced |

---

## Recommended Next Actions (Decision Owner)

По unanimous/majority consensus:

1. **Сегодня:** Принять решение Path A vs Path B. Зафиксировать в CANONICAL_SDP_PIPELINE.md.

2. **Эта неделя:** Написать failing integration test (без StubGateway). Пусть он будет failing — это цель разработки.

3. **Эта неделя (только Path A):** OpenRouterGateway → wiring в agentloop → один real LLM call через harness.

4. **До написания `sdp run` CLI:** Задокументировать и протестировать gate enforcement contract — что происходит при gate failure (rollback/halt/escalation) и что из этого enforced в коде.

5. **Позже:** Production-grade bash tool. Не до этого.
