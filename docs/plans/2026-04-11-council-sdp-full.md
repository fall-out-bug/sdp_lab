# LLM Council Report: SDP Architecture Review

**Date:** 2026-04-11  
**Rounds:** 2  
**Input:** Full project description (not a pre-framed crisis doc)  
**Consensus:** PARTIAL  
**Decision Owner:** PENDING

## Models

| Role | Model |
|------|-------|
| Architect | claude-sonnet-4-6 (codex:codex-rescue) |
| Critic | google/gemini-3.1-pro-preview |
| Technician | deepseek/deepseek-v3.2 |
| Philosopher | moonshotai/kimi-k2 |
| Pragmatist | minimax/minimax-m2.7 |
| Engineer | xiaomi/mimo-v2-pro |

---

## Consensus (≥5/6 agree)

### C1: Governance over opaque subprocess — нерешённая фундаментальная проблема (6/6)

Все шесть ролей назвали это главным структурным изъяном.

SDP заявляет "the pipeline is law", но реальный execution path — `exec.Command("opencode")` с ожиданием exit code + stdout. SDP не видит tool calls, не может остановить агента, не знает текущую фазу. Gates срабатывают постфактум.

Архитектор: *"A gate that fires after an opaque subprocess completes is an audit log, not governance."*  
Критик: *"'Pipeline is law' is a compelling slogan, but law requires enforcement."*  
Прагматик: *"SDP dispatches to opencode as a subprocess, receives only exit code + stdout, and calls that 'evidence.' This is the same blind spot the project exists to solve."*

---

### C2: agentloop — правильное решение, нулевая интеграция (6/6)

Consensus. `internal/agentloop` содержит корректные абстракции: FSM, GateEngine с circuit breaker, EvidenceAccumulator из tool outputs, SQLite WAL. 128 тестов, -race clean.

Но: `grep "github.com/fall-out-bug/sdp_lab/internal/agentloop" cmd/ internal/executor/` → 0 результатов. `ExecutorBridge` продолжает вызывать opencode subprocess.

Архитектор: *"The governance kernel is built but not wired, while the production path still delegates to an uncontrolled subprocess."*  
Прагматик: *"The team built the solution but forgot to use it."*

---

### C3: in-toto attestation на самоотчёте = криптографический театр (5/6)

Quasi-consensus. Migrating evidence format to in-toto — правильный шаг. Но если evidence собирается из stdout неуправляемого subprocess, то signing cryptographic proof поверх self-reported JSON не решает проблему.

Критик: *"A hallucinating or malicious LLM can simply generate a JSON string claiming that tests passed and scope was respected. You are applying enterprise-grade cryptographic signatures to entirely untrustworthy data."*

Контраргумент (Technician): EvidenceAccumulator в agentloop специально ограничен tool outputs (не self-report). Проблема — в текущем disconnected состоянии, не в intended архитектуре. Если agentloop интегрирован, evidence становится реальным.

---

### C4: CANONICAL_SDP_PIPELINE.md в статусе "Proposed" создаёт правовой вакуум (5/6)

"The pipeline is law" + Status: Proposed = противоречие. Никто не знает какие gates обязательны, а какие advisory. Невозможно строить compliance вокруг proposed standard.

Прагматик: *"If the pipeline is law, it cannot be proposed — law is enforced, not debated."*  
Архитектор: *"The integration seam needs an owner and a deadline, not a doc status of 'Proposed.'"*

---

### C5: StubGateway как постоянный placeholder — структурный риск (5/6)

Agentloop использует StubGateway. Нет defined interface contract или migration path. Gateway stub под давлением интеграции имеет тенденцию становиться load-bearing.

Архитектор: *"Define the ModelGateway interface contract now, before integration pressure forces a shortcut."*  
Прагматик (контраргумент): StubGateway — правильный первый шаг: доказать что phase routing, FSM, evidence accumulation, gate logic работают до введения LLM-недетерминизма. Проблема не в стабе, а в незаконченной интеграции.

---

## Split (нет consensus)

### S1: 22 бинаря — симптом или дизайн? (3:3)

**Simptom (Critic, Pragmatist, Philosopher):** Excessive binary decomposition — архитектурная фрагментация. `sdp-omc-guard`, `sdp-ready`, `sdp-a2a` — гиперспецифичные инструменты с неясным usage. Какие 3 команды покрывают 80% daily usage?

**Design (Technician):** Это не sprawling CLI, а decomposed control plane. Каждый компонент — standalone service для K8s-native orchestration. Deliberate design для platform, не user-facing tool.

**Philosopher refinement:** Вопрос не в количестве, а в *orthic decomposition*. Если бинари различаются lifecycle tempo (human-interactive vs CI vs daemon) — оправдано. Если только flag set — мусор.

---

### S2: Human gates — bottleneck или необходимость? (2:4)

**Bottleneck (Philosopher, Engineer частично):** Четыре human approval на карточку = bottleneck. При цели "fully autonomous" — противоречие. Артефакты для approve генерируются тем же агентом которого они полицируют → humans становятся mechanical rubber-stamps.

**Necessary (Pragmatist, Engineer, Technician, Architect):** Immediate problem — не human throughput, а то что automated gates не имеют ничего легитимного для оценки. Сначала надо сделать систему где автоматический gate реально видит tool execution, потом обсуждать замену human gates на stochastic sampling.

---

### S3: Beads (Dolt) — оправдан или лишний? (3:3)

**Оправдан (Technician, Pragmatist, Architect):** Git-backed SQL store с immutable history, branching, mergeable schema — правильный выбор для governance platform. Реальный риск — silent fork между sdp_lab и sdp.

**Лишний (Philosopher, Pragmatist):** Dolt — elegant answer to a question no user asked. Issue history is append-only; cheaper с WORM object storage + merkle snapshots. Если Beads ломается, весь governance парализован.

---

## Новые наблюдения R2 (не были в R1)

**N1: Отсутствует reconciliation loop** (Technician)  
Нет defined процесса для reconciliation между Beads state и physical outcomes при сбое агента. Если агент крашится после частичных изменений, Beads не знает о dirty filesystem state. Beads становится optimistic ledger, не reliable source of truth.

**N2: PhaseRouter recovery logic не определён** (Technician)  
`RecoveryNext` states задокументированы, но нет spec для: что именно triggers recovery, как rollback evidence, что делать с filesystem state. Выбор между retry/rollback/escalation не специфицирован архитектурно.

**N3: Время — неучтённое измерение** (Philosopher)  
Wall-clock time, agent ThinkTime, human latency не appear в evidence или policy. Агент может сжигать calendar time (и бюджет) оставаясь в рамках tool-count и scope rules. "Budgeted time" должен быть first-class gate criterion.

**N4: Human reviewers approving agent-generated artifacts без independent synthesis** (Philosopher)  
Contract, scope diff, staging report — всё генерируется тем же агентом. Без semantic diff tooling или spot-check, humans — механические rubber stamps. Либо дать reviewers semantic diff engine, либо признать что процесс automated и убрать theatrical sign-offs.

**N5: Failure ownership между слоями не определён** (Architect)  
SDP/OmO/Beads — чистое разделение в теории. Но при сбое OmO mid-phase (crash, timeout, partial edit): мост видит exit code, Beads записывает state, SDP не имеет rollback signal. GateEngine escalation paths существуют в agentloop, но agentloop не в pipeline. Кто отвечает за recovery?

---

## Minority Reports

**Technician (deepseek/deepseek-v3.2):**
> "This [22 binaries] criticism misses the architectural intent. The 22 binaries are not a sprawling CLI but a decomposed control plane, where each component is a standalone service that can be orchestrated independently in a K8s-native workflow."

**Pragmatist (minimax/minimax-m2.7):**
> "StubGateway is not the problem — it's the right first step. Building a loop with deterministic stub responses is correct engineering: prove the phase routing, state machine, evidence accumulation, and gate logic work before introducing real LLM nondeterminism. StubGateway is fine for Phase 1; not integrating the loop is the failure."

**Philosopher (moonshotai/kimi-k2):**
> "Even a post-hoc record is legally operable if (a) the record is immutable+signed and (b) the cost of falsifying it outweighs the value of the violation. SDP's real sin is not opacity per se, but that it provides no economic penalty: exit-code parsing is trivial to spoof, so the deterrence is zero. Add verifiable side-effect witnesses (eBPF, container diff, external test runner receipts) and the same black box suddenly becomes a tamper-evident crime scene."

---

## Рекомендации (для Decision Owner)

По unanimous/majority consensus:

**1. Интеграция agentloop — единственный следующий шаг высшего приоритета**

До этого: все governance claims — на бумаге. После: SDP реально видит tool calls, enforces gates, собирает evidence из реальных действий.

Конкретно: заменить `openCodeInvoker` в `ExecutorBridge.DispatchAndRun()` на `agentloop.RestoreHarness` + `h.RunPhase()`. Параллельно: определить `ModelGateway` interface contract (не реализовывать — только контракт).

**2. Заморозить CANONICAL_SDP_PIPELINE.md как v1.0**

Пока pipeline в статусе "Proposed" — нет basis для enforcement. Заморозить или явно пометить SDP как research platform с незафиксированными правилами.

**3. Написать один failing integration test**

До agentloop интеграции: один end-to-end test с реальным LLM call (не StubGateway), real tool execution, gate evaluation, evidence в `.sdp/evidence/`. Пусть он будет failing — это цель разработки.

**4. Определить failure ownership между SDP/OmO/Beads**

Кто отвечает за recovery при crash mid-phase? Reconciliation loop — кто его запускает? Это не code question, это protocol question. Нужен документ, не PR.

**Отложить:** K8s Phase 9, Platform Reset F091-F096, in-toto Phase 5-7 — пока agentloop не интегрирован и pipeline не заморожен.

---

## Round Convergence

| Round | Что делали | Ключевые находки |
|-------|-----------|-----------------|
| R1 | Blind review проекта (без наводящих вопросов) | C1-C5 consensus сформирован |
| R2 | Роли отвечают друг другу | N1-N5 новые наблюдения; S1-S3 split зафиксированы |
