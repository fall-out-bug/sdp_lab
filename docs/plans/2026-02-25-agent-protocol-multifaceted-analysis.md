# Многосторонний анализ: протокол, агент, пользователь

**Дата:** 2026-02-25  
**Метод:** @think — Stage 1 breakdown → Stage 2 parallel expert analysis (4 experts) → Stage 3 summary  
**Источники:** `cursor_build_00_53_16.md`, `cursor_markdown_file_build_discussion.md`, `cursor_project_design_and_code_analysis.md`

---

## Stage 1: Task Breakdown

| Аспект | Эксперт | Фокус |
|--------|---------|-------|
| Agent execution patterns | Kent C. Dodds + Martin Fowler + Troy Hunt | Тесты, дрейф состояния, размещение артефактов, OOS по умолчанию |
| User interaction patterns | Nir Eyal (UX) + Theo Browne (API) | Микроменеджмент, статус, «продолжай», drift/corrections |
| Protocol state management | Martin Kleppmann (distributed systems) | Beads ↔ WS ↔ INDEX ↔ checkpoint, единый источник истины |
| Protocol flow and review cycle | W. Edwards Deming (process) | review→design→build, guard/scope, форматы, Round accumulation |

---

## Stage 2: Expert Findings (Summary)

### Expert 1: Agent Execution Patterns

**Принципы:** Test behavior not implementation; small steps; defense in depth.

| # | Проблема | Доказательство | Рекомендация |
|---|----------|----------------|---------------|
| 1 | **Test shortcuts** | 00-053-20: "Simplifying tests... removing integration tests"; vague "Tests may be slow" | @build: контракт "Integration: t.Skip + -short; never delete; CI: go test -short" |
| 2 | **State drift** | 00-053-06..09 Done в review, status pending; beads уже исправлены, но WS созданы; beads не закрыты после build | Post-build `bd close`; pre-design verification (grep code для каждого bead) |
| 3 | **Artifact placement** | F053-REVIEW-SUMMARY в backlog/; idea-f053 и idea-phase4 — два плана | AGENTS.md: правила размещения; @design: `ls docs/drafts/idea-*` перед созданием |
| 4 | **OOS by default** | "никаких out of scope" | @design: beads из review → in scope; OOS только с обоснованием |

**Приоритет:** State drift (2) → Placement (3) → OOS (4) → Test shortcuts (1)

---

### Expert 2: User Interaction Patterns

**Принципы:** Triggers, actions, variable rewards; reduce cognitive load; one command per intent.

| # | Точка трения | Доказательство | Решение |
|---|--------------|----------------|---------|
| 1 | **22× sequential /build** | cursor_build: /build 00-53-16 … 37 | Batch `/build 00-053-16..25` |
| 2 | **Manual status** | "Проверь beads", "Найди оставшиеся", "А эти сделаны?" | `/status F053` — pending WS, open beads, next action |
| 3 | **Ambiguous "продолжай"** | "продолжай F053" — build? design? review? | Конвенция: = `sdp-orchestrate --next-action`; документировать |
| 4 | **Drift corrections** | "никаких OOS", "два плана сделал", "файл не в той папке" | Checklists в skills; proactive sync |

**Решение:** **Combined (D)** — batch + /status + continue convention + checklists. Порядок: /status → batch build → continue doc → checklists.

---

### Expert 3: Protocol State Management

**Принципы:** Eventual consistency; bounded context; declarative config.

| # | Проблема | Доказательство | Варианты |
|---|----------|----------------|----------|
| 1 | **INDEX vs WS drift** | INDEX показывает Pending; WS файлы — done | Checkpoint as primary; generate INDEX from checkpoint |
| 2 | **Status inconsistency** | done vs completed | Нормализовать к одному значению |
| 3 | **Beads for fixed code** | "А эти сделаны?" — te1h, ywv8, wp5a уже исправлены | Pre-check в @design: bd show + grep |
| 4 | **Beads not closed** | "закрой сделанные beads" | Post-build `bd close` в @build |

**Решение:** **A (checkpoint primary) + B (sync rules)**. Phase 1: `sdp index --feature F053` генерирует INDEX. Phase 2: нормализация status. Phase 3: sync rules в skills.

**Риски:** Checkpoint может отсутствовать; orchestrate vs ciloop checkpoint — совместимость.

---

### Expert 4: Protocol Flow and Review Cycle

**Принципы:** PDCA; reduce waste; make work visible.

| # | Проблема | Решение |
|---|----------|---------|
| 1 | **Review → design handoff** | @review: блок "If CHANGES_REQUESTED: Run @design phase4-remediation with findings" |
| 2 | **Guard/scope contract** | @design + @guard: документировать "scope_files supports directory prefix; guard uses prefix matching" |
| 3 | **Format contracts** | @build: TDD markers, ws-verdict, Execution Report; post-build: jq schema validation для ws-verdict |
| 4 | **Round accumulation** | @design: checklist "bead already fixed?" перед созданием WS |

**Выбор:** Все A/B — skill-level изменения, без новых инструментов.

---

## Stage 3: Unified Summary

### Сводная матрица (эксперт → проблема → действие)

| Действие | Где | Effort | Источник |
|----------|-----|--------|----------|
| Post-build `bd close` | @build SKILL.md | 0.5d | E1, E3, E4 |
| Pre-design "bead fixed?" check | @design skill | 0.5d | E1, E3, E4 |
| Artifact placement rules | AGENTS.md | 0.5d | E1 |
| Pre-draft `ls docs/drafts/idea-*` | @design skill | 0.5d | E1 |
| Default-in-scope for review beads | @design skill | 0.5d | E1, E4 |
| Integration test contract | @build SKILL.md | 0.5d | E1 |
| `/status F053` | skill или sdp status | 1–2d | E2 |
| Batch `/build 16..25` | @build skill | 0.5d | E2 |
| "Продолжай" = next-action | AGENTS.md | 0.5d | E2 |
| Review handoff block | @review skill | 0.5d | E4 |
| Guard/scope doc | @design, @guard | 0.5d | E4 |
| ws-verdict schema validation | post-build hook | 0.5d | E4 |
| `sdp index --feature F053` | cmd/sdp-orchestrate | 1–2d | E3 |
| Checkpoint as primary | design doc + impl | 3–5d | E3 |

### Приоритизированный план

**Quick wins (1–2 дня):**
1. @build: post-build `bd close`, integration test contract
2. @design: pre-draft check, default-in-scope, bead-fixed checklist
3. @review: handoff block
4. AGENTS.md: placement rules, "продолжай" convention
5. @guard: scope prefix documentation

**Medium (2–4 дня):**
6. `/status F053` command
7. Batch `/build 16..25` в skill
8. ws-verdict schema validation в hook
9. `sdp index --feature F053` (INDEX generation)

**Longer (отдельный WS):**
10. Checkpoint as primary source of truth

### Консенсус экспертов

- **Все 4 эксперта** сходятся: post-build `bd close` и pre-design bead verification — must-have.
- **E1, E2, E4** сходятся: checklists и контракты в skills — низкий effort, высокий impact.
- **E2, E3** сходятся: `/status` и batch build — снижают cognitive load и coordination.
- **E3, E4** сходятся: checkpoint/INDEX generation — следующий шаг после quick wins.

---

## Маршрутизация: F053 (техника) vs F054 (протокол)

**Решение:** Технические находки → F053. Протокол и промпты → поиск в существующих WS; если нет — F054 (непрерывное улучшение протокола, часть будущего роя).

### Технические → F053

| Находка | Суть | F053 WS | Действие |
|---------|------|---------|----------|
| ws-verdict schema validation | Post-build hook: jq + schema validation | — | Новый WS 00-053-44 или добавить в 00-053-31 |
| `sdp index --feature F053` | Генерация INDEX из checkpoint/WS | — | Новый WS 00-053-45 |
| Checkpoint as primary | Единый источник истины для phase, WS status | 00-053-18 (merge-safe) | Расширить 00-053-18 или новый WS 00-053-46 |
| Integration test contract | CI: `go test -short`; t.Skip для integration | — | Добавить в 00-053-16 (hooks) или новый WS |
| Post-build `bd close` | Вызов bd CLI после build | — | **Протокол** (skill) → см. F054 |
| Pre-design bead verification | grep кода перед созданием WS | — | **Протокол** → F054 |

**Итого F053:** 00-053-44 (ws-verdict validation), 00-053-45 (sdp index), 00-053-46 (checkpoint primary) или расширить 00-053-18; integration contract — в hooks или отдельно.

### Протокол/промпты → поиск в F053 → иначе F054

| Находка | Суть | Существующий F053 WS? | Решение |
|---------|------|----------------------|---------|
| Post-build `bd close` | @build skill: после success — bd close | 00-053-37 (PromptOps) — Done | F054: @build checklist |
| Pre-design bead-fixed check | @design: bd show + grep перед созданием WS | — | F054 |
| Artifact placement rules | AGENTS.md: docs/reviews/, docs/drafts/ | — | F054 |
| Pre-draft `ls docs/drafts/idea-*` | @design: не дублировать draft | — | F054 |
| Default-in-scope | @design: beads из review → in scope | — | F054 |
| `/status F053` | skill или sdp status | — | F054 |
| Batch `/build 16..25` | @build skill: итерация по диапазону | 00-053-37 — Done | F054: @build batch |
| «Продолжай» = next-action | AGENTS.md конвенция | — | F054 |
| Review handoff block | @review: «Run @design with findings» | — | F054 |
| Guard/scope doc | scope_files prefix в @design/@guard | 00-053-36 — Done | Уже есть |

**Итого F054:** Все протокольные находки, для которых нет подходящего F053 WS. F054 = «Continuous Protocol Improvement» — раздел будущего роя.

### F054: Continuous Protocol Improvement

**Описание:** Непрерывное улучшение протокола SDP — skills, AGENTS.md, workflow, handoffs. Часть будущего роя (dream swarm).

**Планируемые WS (черновик):**

| WS | Название | Источник |
|----|----------|----------|
| 00-054-01 | @build: post-build bd close + batch /build | E1, E2 |
| 00-054-02 | @design: pre-draft check, bead-fixed, default-in-scope | E1, E4 |
| 00-054-03 | @review: handoff block | E4 |
| 00-054-04 | AGENTS.md: placement rules, «продолжай» convention | E1, E2 |
| 00-054-05 | /status F053 command | E2 |
| 00-054-06 | AGENTS.md / CLAUDE.md sync sdp ↔ sdp_lab | — |

---

## Следующие шаги

1. ~~**F053:** Создать 00-053-44, 45, 46~~ — **Сделано** (через @design)
2. ~~**F054:** Добавить в ROADMAP; draft; workstreams 00-054-01..05~~ — **Сделано**
3. ~~**INDEX:** Обновить~~ — **Сделано**

**Выполнено (2026-02-25):**
- Drafts: `idea-f053-technical-phase5.md`, `idea-f054-protocol-improvement.md`
- F053: 00-053-44, 45, 46 (beads: vxef, rnz2, utmd)
- F054: 00-054-01..05 (beads: hryg, 0hy8, u1ni, jbbi, ju7v)
- Mapping: 105 = 105 backlog
- ROADMAP, INDEX обновлены; bd sync выполнен

**Команды для продолжения:** `@build 00-053-44` или `@build 00-054-01`

---

*Документ создан по @think: 4 expert subagents. Маршрутизация F053/F054 — по решению пользователя. Внесено через @design skill (drafts → workstreams → beads → mapping → INDEX).*
