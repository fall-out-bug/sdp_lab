# StratAudit — Claim-Centric Trace Explorer Design

**Date:** 2026-04-13
**Status:** Proposed
**Feature:** F117
**Owner:** Андрей
**Module:** `internal/strataudit`
**CLI:** `cmd/sdp-strataudit`

---

## Problem

Текущий StratAudit после `F109` стал честнее по evidence contract, но как
аналитический продукт всё ещё не решает главную задачу: **понятный анализ
документов и явная трассировка между слоями**.

Наблюдения по реальному корпусу и ручному просмотру отчёта:

1. Один экран пытается быть одновременно summary, document reader, trace viewer
   и debug console.
2. Пользователь не видит, что чему соответствует: между документами, сущностями
   и уровнями нет ясной карты соответствий.
3. При `0` verified traces отчёт деградирует в перегруженный список, а не
   показывает, где именно оборвалась трасса.
4. Сравнение прогонов загрязняет основной user flow. Это debug-задача, не
   продуктовый первый экран.
5. Документ остаётся слишком крупной единицей объяснения. Для реальной
   трассировки нужен путь `document -> claim -> link -> claim -> document`.

Вывод: `F109` закрыл trust и provenance, но не закрыл **trace UX model**.

---

## Product Goal

Сделать StratAudit инструментом, где:

1. аналитик начинает с карты документов, а не с простыни алертов;
2. трасса строится вокруг claim/entity с цитатой, а не вокруг документа как
   неделимого блока;
3. при отсутствии подтверждённых связей видно место разрыва и причина разрыва;
4. debug-диагностика изолирована от основного аналитического отчёта;
5. интерфейс раскладывается по табам с разными задачами, а не по одной длинной
   странице.

---

## Scope

В scope `F117` входят:

- claim-centric trace model поверх уже существующего provenance;
- first-class `trace_nodes`, `trace_edges`, `trace_gaps`;
- document correspondence read model;
- табовая IA отчёта;
- waterfall-представление разрывов трассировки;
- separation between analyst view and diagnostics/debug view;
- regression/UX acceptance на реальном или воспроизводимом корпусе.

Вне scope `F117`:

- исправление recall на strategy deck'ах;
- новая extraction ontology;
- multi-run compare как часть основного отчёта;
- collaborative moderation UI;
- отдельный web backend.

Open bugs вроде `sdplab-qkyt` и `sdplab-h2qm` остаются отдельной линией:
`F117` не обещает magically создать хорошие traces из плохих данных. Он обязан
сделать состояние системы наблюдаемым и аналитически понятным.

---

## Design Thesis

### 1. Document Is The Entry Point, Claim Is The Trace Unit

Пользователь мыслит документами. Система должна открываться картой документов.
Но единица трассы — не документ, а claim/entity с конкретной цитатой.

Документный view должен быть агрегатом над claim graph, а не подменой graph.

### 2. No Silent Empty State

Если трасса не построилась, интерфейс обязан сказать:

- на каком claim это произошло;
- между какими слоями;
- в каком stage;
- по какой причине.

`0 traces` как итоговая цифра без объяснения — плохой продукт.

### 3. Report Tabs Must Separate Jobs

Один экран не должен обслуживать четыре разных сценария. Нужны отдельные
поверхности под:

- краткий сводный readout;
- документы;
- трассировку;
- разрывы;
- диагностику.

### 4. Compare Mode Is Secondary

Сравнение прогонов полезно для engineering/debug, но это не основная форма
анализа корпуса. По умолчанию его в продуктовой поверхности быть не должно.

### 5. Russian UX By Default

Если корпус русскоязычный, chrome отчёта и аналитические заголовки тоже должны
быть русскоязычными. Английский допустим только для исходного контента или
технических кодов/идентификаторов.

---

## Design Decisions

### AD-1: Extend `report.v2.json`, Do Not Invent A New Contract Prematurely

`F109` уже закрепил `report.v2.json` как основной контракт. `F117` не должен
ломать его только потому, что текущий UI неудобен.

Правило:

- новые trace explorer поля добавляются **аддитивно** в `report.v2.json`;
- major bump допускается только если старые семантики ломаются необратимо;
- HTML читает только report contract, не лезет обратно в SQLite/store.

Новые top-level/read-model блоки:

```json
{
  "trace_graph": {
    "nodes": [],
    "edges": [],
    "paths": []
  },
  "trace_gaps": [],
  "document_views": [],
  "report_modes": {
    "default": "analyst",
    "compare_available": false
  }
}
```

### AD-2: `trace_node` Is The Minimal Explainable Unit

Каждый `trace_node` описывает конкретную admitted claim/entity с evidence.

Минимальные поля:

- `id`
- `entity_id`
- `document_id`
- `section_id`
- `level_id`
- `title`
- `source_quote`
- `trust_grade`
- `lang`

Это позволяет одинаково строить document view, trace view и gap view.

### AD-3: `trace_edge` Must Express Status, Not Just Success

Нам нужны не только подтверждённые связи. Иначе пустой граф ничего не объяснит.

Минимальный контракт edge:

- `from_node_id`
- `to_node_id`
- `status`: `verified | candidate | rejected`
- `verification_mode`
- `confidence`
- `similarity`
- `reason`
- `source_evidence_ref`
- `target_evidence_ref`

Verified и rejected должны жить в одной модели, но с разным статусом и
визуальным режимом.

### AD-4: `trace_gap` Is A First-Class Product Object

`trace_gap` нужен там, где связь не дошла даже до candidate или candidate не
перешёл admission.

Минимальные поля:

- `node_id`
- `from_level`
- `expected_to_level`
- `stage`: `candidate_search | verification | admission | upstream_missing`
- `gap_type`
- `reason`
- `candidate_count`
- `top_candidate_ids`

Примеры `gap_type`:

- `no_candidates`
- `all_candidates_rejected`
- `low_confidence`
- `missing_upstream_entities`
- `quote_evidence_missing`
- `language_mismatch`

### AD-5: Document Correspondence Is An Aggregated Read Model

Пользователь всё равно хочет понять: "какой документ чему соответствует?".
Это нельзя оставлять на уровне ручной догадки по entity list.

Нужен `document_view`:

- document metadata;
- extracted claim count;
- inbound/outbound trace counts by level;
- upstream/downstream corresponding documents;
- top blockers and gap counts;
- critical quality flags.

Это не замена trace graph. Это читабельная сводка над ним.

### AD-6: Default Report IA Uses Tabs

Новая обязательная IA:

1. `Сводка`
2. `Документы`
3. `Трассировка`
4. `Разрывы`
5. `Диагностика`

Назначение:

- `Сводка` — KPI, scope, ограничения, top blockers;
- `Документы` — карта корпуса и correspondence по документам;
- `Трассировка` — путь selected claim вверх/вниз по слоям;
- `Разрывы` — waterfall причин, почему путь не собрался;
- `Диагностика` — rejected/candidate/debug artifacts.

### AD-7: No Fake Graph When There Are No Verified Traces

Если verified traces == 0:

- `Трассировка` не рисует декоративную сеть;
- `Разрывы` становится основным trace screen;
- каждый слой показывает точку отваливания и причину;
- summary честно пишет, что система видит candidate/gap landscape, а не
  подтверждённую стратегическую цепочку.

---

## UX Requirements

### Mandatory

1. Пользователь может начать с документа и увидеть:
   - какие claims извлечены;
   - какие upstream/downstream документы с ним связаны;
   - какие связи подтверждены;
   - какие связи оборвались.
2. Пользователь может начать с claim и увидеть:
   - source quote;
   - переходы вверх и вниз по слоям;
   - verification status каждого шага;
   - причину разрыва, если шаг не состоялся.
3. Default report не содержит compare-mode и не смешивает diagnostics c summary.
4. Все основные подписи отчёта русскоязычные.
5. `Разрывы` показывает причину, stage и affected documents/claims, а не просто
   число `0 traces`.

### Explicitly Forbidden

- делать документ единственной единицей traceability;
- рисовать "graph-looking" заглушку без evidence или gap reasons;
- оставлять compare diff в первом экране аналитического отчёта;
- смешивать rejected/candidate diagnostics с executive summary;
- переключать chrome отчёта между русским и английским без явной причины.

---

## Acceptance Criteria

1. `report.v2.json` содержит аддитивные блоки `trace_graph`, `trace_gaps`,
   `document_views`, достаточные для offline HTML without SQLite.
2. Для каждого admitted claim на boundary level существует хотя бы одно из:
   - verified edge,
   - candidate/rejected edge,
   - trace gap с явной причиной.
3. HTML-отчёт раскладывается на табы `Сводка`, `Документы`, `Трассировка`,
   `Разрывы`, `Диагностика`.
4. Default report mode не показывает cross-run compare.
5. При `0` verified traces пользователь видит waterfall разрывов, а не пустой
   псевдограф.
6. `Документы` объясняет correspondence между документами через claims и links,
   а не через ручной prose summary.
7. Основной chrome отчёта русскоязычный и не смешивает языки без нужды.

---

## Deferred

- исправление recall/gating на strategy slides;
- auto-healing trace suggestions;
- multi-run compare dashboard;
- interactive graph canvas;
- human approval workflow for disputed links.

---

## Recommended Next Artifact

Следующий артефакт после этой design-spec:

- [docs/plans/2026-04-13-strataudit-trace-explorer-plan.md](2026-04-13-strataudit-trace-explorer-plan.md)

Его задача — разрезать `F117` на workstreams так, чтобы сначала появился честный
trace substrate, потом document correspondence, и только потом tabbed HTML.
