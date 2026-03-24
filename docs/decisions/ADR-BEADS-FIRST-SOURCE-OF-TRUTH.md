# ADR: Beads-first operational source of truth

- **Status:** Proposed
- **Date:** 2026-03-24
- **Related:** `../BEADS_FIRST_CONTROL_TOWER_ROADMAP.md`, `../SDP_SPEC_DRIVEN_PIPELINE_CANON.md`, `../FEATURECARD_BEADS_MAPPING.md`, `../BEADS_SDP_SCHEMA.md`

## Context

В текущей модели SDP одновременно держит lifecycle state в двух местах:

1. `FeatureCard` в `.sdp/control/...` как YAML/JSON control-store
2. `Beads Issue` как dependency-aware durable graph

Это создаёт **split-brain**.

### Почему это проблема

| Проблема | Проявление |
|---|---|
| Дублирование статуса | `FeatureCard.status=ready`, а Bead уже `blocked` или `in_progress` |
| Дублирование блокировок | `waiting_on`, `blocking_reasons`, `needs_feedback_from` в card конкурируют с deps/gates в Beads |
| Дублирование orchestration semantics | control store повторно моделирует ready queue, dispatch eligibility, follow-up routing |
| Неясный truth boundary | непонятно, чему верить: YAML, snapshot, packet state или Beads graph |
| Слабая объяснимость | человеку и агенту трудно ответить на вопросы `что сейчас реально происходит?`, `почему blocked?`, `что ready next?` |
| Сложная federation/governance | A2A, provenance, evidence и policy труднее строить поверх двух независимых lifecycle stores |

### Корень проблемы

`FeatureCard` исторически вырос из удобного orchestration-объекта в фактический lifecycle store. Но Beads уже решает именно тот класс задач, для которого lifecycle store и нужен:

- durable identity work item'а
- statuses
- dependencies
- readiness/blocking
- graph traversal
- gate-like coordination through issues + deps + await fields

Поддерживать поверх этого второй полноценный store состояния — архитектурная ошибка. Это увеличивает стоимость миграций, reconciliation, board projection и execution routing.

## Decision

### 1. Beads = единственный operational source of truth

Для SDP operational truth хранится в `Beads Issue` и графе зависимостей Beads.

**В Beads canonical live data:**
- work item identity
- issue type
- status
- priority
- dependencies / blockers
- assignment / claim / owner
- gate beads
- parent-child topology
- operational metadata в `Issue.Metadata.sdp.*`

### 2. FeatureCard = semantic artifact / projection

`FeatureCard` сохраняется, но перестаёт быть самостоятельным lifecycle store.

Его новая роль:
- intake/spec artifact
- semantic projection для человека и orchestration слоя
- carrier для нормализованного intent, scope, why-now, non-goals
- reference envelope к Bead ID, contract artifact, provenance/evidence artifacts

`FeatureCard` больше не является источником истины для:
- текущего execution status
- blocked/ready semantics
- active execution ownership
- gate state
- retry/review/delivery counters как независимой истины

Эти данные должны либо жить в самом Beads issue, либо вычисляться из него и связанных artifacts.

### 3. Control store = derived views, not truth

Control Tower остаётся, но как **thin orchestration shell**:
- policy
- routing
- view generation
- board/snapshot projections
- dispatch packet assembly
- result ingestion
- evidence/provenance indexing

`.sdp/control` может хранить:
- projections
- snapshots
- packets
- artifacts
- caches

Но не должен быть автономным lifecycle engine.

### 4. SDP-specific semantics хранятся в Beads metadata

Все SDP-специфичные operational поля, которые не имеют прямого typed-поля в `Issue`, переносятся в `Issue.Metadata.sdp.*`.

Примеры:
- `sdp.card_id`
- `sdp.phase`
- `sdp.contract.*`
- `sdp.review.*`
- `sdp.delivery.*`
- `sdp.executor.*`
- `sdp.provenance.*`
- `sdp.gates.*`

### 5. External artifacts остаются файлами

Не всё должно жить внутри Beads.

Файловыми artifact'ами остаются:
- `contract.json`
- intake markdown
- dispatch/result packets
- evidence bundles
- provenance records
- review/qa reports
- board snapshots

В Beads хранятся **ссылки и state anchors**, а не большие blob-артефакты.

## Consequences

### Что меняется

| Было | Станет |
|---|---|
| `FeatureCard.status` = рабочая истина | `Issue.Status` = рабочая истина |
| `LinkedBeadsIDs` связывает card с bead | bead и есть canonical card/work item; отдельный список не нужен |
| `waiting_on`, `blocking_reasons`, `needs_feedback_from` живут в YAML | блокировки выражаются deps, gate beads и `metadata.sdp.gates` |
| board может интерпретировать lifecycle сам | board только проецирует Beads + artifacts |
| dispatch eligibility считается control-store логикой | readiness выводится из Beads status/dependencies/gates |
| counters и review/delivery state размазаны по card | counters/state живут в metadata.sdp и/или derived views |

### Что не меняется

| Инвариант | Комментарий |
|---|---|
| SDP остаётся spec-driven pipeline | intent/spec/contract/evidence/provenance никуда не исчезают |
| FeatureCard остаётся полезным | но как semantic artifact, а не durable workflow DB |
| TaskContract остаётся executable spec | operational truth ≠ semantic contract; контракт по-прежнему обязателен |
| Control Tower остаётся | но как orchestration/policy/view shell |
| opencode остаётся runtime | Beads не исполняет код, а фиксирует work graph/state |
| A2A остаётся transport/API boundary | pivot не меняет transport surface, только truth layer |

### Migration path

#### Phase 1. Canonical mapping
- Зафиксировать mapping `FeatureCard -> Beads Issue + Metadata + Artifacts`
- Зафиксировать taxonomy issue types / gate types / dep types
- Описать status/phase mapping

#### Phase 2. Storage boundary extraction
- Выделить repository boundary в `internal/control`
- Оставить file-backed implementation без изменения внешнего API

#### Phase 3. Beads adapter
- Ввести Beads-backed repository / adapter
- Научить создавать, читать, обновлять и искать SDP work items через Beads

#### Phase 4. Dual-write / shadow-read
- File control store временно сохраняется как projection/cache
- Operational updates пишутся в Beads
- Snapshot equivalence проверяется через comparison

#### Phase 5. Cutover
- Read path переключается на Beads
- Ready/blocked/next-action перестают вычисляться из YAML как из primary store
- `FeatureCard` остаётся derived artifact

#### Phase 6. Cleanup
- Удалить или депрекейтнуть поля `FeatureCard`, которые имитируют lifecycle store
- Упростить board/snapshot код до projection-only behavior

### Risks

| Риск | Деталь | Митигация |
|---|---|---|
| Миграционная рассинхронизация | dual-write период может дать divergent state | shadow-read diff, строгий cutover plan, ограниченный период dual-write |
| Перегрузка metadata | слишком много SDP state может превратить `metadata.sdp` в свалку | каноническая schema, versioning, чёткое деление: typed field vs metadata vs artifact |
| Слабая выразительность status | temptation напихать phase в custom status | использовать `Issue.Status` для operational state, `metadata.sdp.phase` для SDP phase |
| Потеря удобства human-readable card | после cutover card может стать слишком тонким | сохранить FeatureCard как projection/spec summary |
| Неявные gate semantics | разные компоненты будут по-разному трактовать gates | зафиксировать taxonomy gate issue types и auto-close rules |
| Старый код продолжит писать в YAML как в truth | скрытый legacy path сломает инвариант | repository boundary, ADR enforcement, audit существующих write paths |

## Decision Drivers

1. Один durable graph лучше двух конкурирующих stores
2. Readiness и blocking должны вычисляться там, где уже живут deps
3. Governance, provenance и A2A проще строить поверх одного operational truth layer
4. Derived views должны быть производными, а не скрытой альтернативной БД
5. SDP должен определять meaning, constraints и trust, а не переизобретать task graph

## Resulting invariant

> **Beads stores operational reality. SDP adds semantic meaning, contracts, evidence, and policy.**

Следовательно:

> **Если Beads и FeatureCard расходятся, прав Beads.**
