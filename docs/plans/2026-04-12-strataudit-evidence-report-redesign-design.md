# StratAudit v2 — Evidence-Backed Report Redesign Design

**Date:** 2026-04-12
**Status:** Proposed
**Feature:** F109
**Owner:** Андрей
**Module:** `internal/strataudit`
**CLI:** `cmd/sdp-strataudit`

---

## Problem

Текущий StratAudit уже умеет собирать документы, сущности, трассы и findings, но
как продукт он не выполняет главное обещание: **доказуемый аудит стратегической
трассировки**.

Наблюдения по реальному прогону на корпусе УБРиР:

1. В сущности попадает мусор из system prompt и служебного boilerplate.
2. Названия и описания переводятся или переписываются на английский, хотя
   исходный документ на русском.
3. Отчёт не даёт перехода от finding/trace к документу, разделу и цитате.
4. Coverage показывается как одна цифра по уровню без раскрытия по документам
   и разделам.
5. Similarity и LLM verification выглядят как доказательства, хотя по сути это
   только механизм генерации кандидатов.
6. HTML-отчёт сводит всё к плоскому списку findings. Это не аналитический
   интерфейс, а длинная витрина алертов.

Пока это не исправлено, переформатирование HTML само по себе будет косметикой.

---

## Product Goal

Сделать StratAudit инструментом, где:

1. руководитель видит честную картину выравнивания стратегии с явным уровнем доверия;
2. аналитик может провалиться из любой проблемы до конкретного документа, раздела
   и цитаты;
3. оператор понимает качество корпуса и может отличить плохую стратегию от
   плохого извлечения;
4. финальная пользовательская итерация действительно сводится к
   **переформатированию отчёта**, потому что данные и trust-модель уже правильные.

---

## Why Not "Just Reformat HTML"

Это слабая постановка задачи. Причина:

- если entity title переведён моделью, ссылка на красивую карточку не делает его правдой;
- если trace построен только по similarity, граф не становится evidence chain;
- если finding не показывает document path и quote, пользователь не может
  проверить вывод;
- если coverage не раскрывается по документам и разделам, нельзя понять, это
  стратегический разрыв или просто плохой source corpus.

Вывод: **сначала data contract и evidence model, потом report IA, потом HTML layout**.

---

## Scope

В scope v2 входят:

- trust-модель для entity/trace/finding;
- сохранение исходного языка и запрет скрытого перевода;
- first-class provenance до уровня раздела/чанка/цитаты;
- JSON contract v2 для отчёта;
- новая информационная архитектура HTML-отчёта;
- grouped findings и раскрытие coverage по документам и разделам;
- регрессионная проверка на реальном русскоязычном корпусе.

Вне scope v2:

- отдельный web backend;
- collaborative moderation UI;
- cross-run diff как first-class feature;
- ручная разметка/approval workflow;
- новая ontology DSL.

---

## Product Principles

### 1. Truth Over Beauty

Отчёт обязан показывать не только вывод, но и основание вывода.

### 2. Source Language Is Sacred

По умолчанию показываем то, что написано в документе. Никаких скрытых переводов
или "улучшенных" англоязычных названий.

### 3. Similarity Is Discovery, Not Proof

Similarity и LLM verification помогают найти кандидатов, но не считаются
достаточным основанием для сильной стратегической связи.

### 4. Quality Of Corpus Must Be Visible

Если документ загрязнён HTML export, MIME header, prompt leak или boilerplate,
это должно быть видно в отчёте отдельно от стратегических findings.

### 5. Findings Must Be Grouped

Один entity = один gap — плохой UX. Пользователю нужны кластеры проблем:
документ, раздел, тема, уровень, тип дефекта.

---

## Design Decisions

### AD-1: Verified Evidence Is The Only Report Truth

В продуктовый отчёт попадают только те объекты, для которых можно показать
проверяемое основание.

Новая trust-шкала:

| Grade | Meaning | Visible in report by default |
|---|---|---|
| `verified` | quote/span реально найден в документе; trace опирается на реальное evidence | yes |
| `supported` | данные достаточно сильные для аналитики, но есть ограничения | yes |
| `suspect` | объект создан, но evidence или язык вызывают сомнение | only in diagnostics |
| `rejected` | объект отброшен пайплайном | only in diagnostics |

Правило:

- `trace_candidates` могут опираться на similarity и LLM;
- `verified_traces` для пользовательского отчёта обязаны иметь evidence bundle;
- similarity-only связь остаётся диагностикой, а не стратегическим фактом.

### AD-2: Source-Preserving Multilingual Extraction

Сущность должна хранить оригинал и только потом derived-представления.

Новые поля entity/report contract:

| Field | Purpose |
|---|---|
| `title_original` | исходный title из документа или verbatim extraction |
| `description_original` | исходное описание |
| `lang` | язык фрагмента |
| `display_title` | нормализованное отображение без перевода |
| `display_description` | нормализованное отображение без смены языка |
| `language_mismatch` | флаг рассинхрона между документом и entity |

Жёсткое правило для extraction:

- prompt требует сохранять язык исходного фрагмента;
- title/description не переводятся;
- если модель вернула язык, которого нет в source quote, entity получает
  `suspect` или отбрасывается.

### AD-3: Sections And Chunks Become First-Class Provenance Units

Текущий документ слишком крупный объект. Для настоящей трассировки нужен слой
разделов/чанков.

Новый provenance layer:

| Object | Purpose |
|---|---|
| `document` | файл, путь, тип, ingest metadata |
| `section` | логический раздел или fallback chunk |
| `quote_span` | точная цитата и диапазон внутри section |
| `entity_evidence` | связь entity с quote_span |
| `trace_evidence` | bundle из source/target evidence и verification mode |

Минимальный контракт для `section`:

- `id`
- `document_id`
- `ordinal`
- `heading`
- `char_start`, `char_end`
- `preview`
- `content_hash`
- `quality_flags[]`

### AD-4: Corpus Hygiene Is A First-Class Output

Отдельный слой диагностики корпуса обязателен. Он не должен маскироваться под
стратегические выводы.

Новые качества, которые считаются отдельно:

- `prompt_leak`
- `mime_header_noise`
- `html_export_noise`
- `language_drift`
- `quote_not_found`
- `boilerplate_repetition`
- `unsupported_format`
- `section_parse_fallback`

Отчёт обязан показывать:

- сколько entity rejected/suspect;
- в каких документах больше всего noise;
- какие файлы/разделы тянут coverage вниз;
- где finding нельзя считать стратегическим из-за качества источника.

### AD-5: Report JSON v2 Becomes The Real Product Contract

HTML больше не источник истины. Истина — `report.v2.json`.

Топ-level contract:

```json
{
  "schema_version": "2",
  "audit_scope": {},
  "trust_summary": {},
  "corpus_quality": {},
  "levels": [],
  "documents": [],
  "sections": [],
  "entities": [],
  "trace_candidates": [],
  "verified_traces": [],
  "findings_grouped": [],
  "coverage": {},
  "evidence_pack": {}
}
```

Минимальные требования:

- у entity есть document path, section id, source quote, trust grade;
- у verified_trace есть source/target evidence refs, verification mode,
  similarity score, justification, trust grade;
- coverage раскрывается по уровню, документу и разделу;
- findings сгруппированы и ссылаются на evidence, а не только на entity IDs.

### AD-6: HTML Report Gets Three Layers

Новая IA отчёта:

#### Layer 1: Executive Overview

- scope и ограничения прогона;
- trust summary;
- heatmap покрытия по уровням;
- top grouped findings;
- список blocking data-quality problems.

#### Layer 2: Analyst Explorer

- coverage by level -> by document -> by section;
- trace explorer with source and target evidence;
- grouped findings with drill-down;
- filters by level, document, trust grade, finding type.

#### Layer 3: Evidence Pack

- link to JSON export;
- link to source documents;
- exact quotes/spans;
- candidates rejected from final truth model;
- diagnostics and counters for corpus quality.

### AD-7: Findings Move From Alert Spam To Problem Clusters

Новый finding model:

| Group | Example |
|---|---|
| `strategic_gap_cluster` | цели документа X не поддержаны документами architecture |
| `orphan_cluster` | раздел implementation живёт без связи со strategy |
| `corpus_quality_cluster` | документ загрязнён export boilerplate |
| `language_drift_cluster` | entity title rewritten into English |
| `trace_ambiguity_cluster` | несколько кандидатов без сильного evidence |

Entity-level findings остаются доступными, но по умолчанию идут внутри cluster drill-down.

---

## Report UX Requirements

### Mandatory

1. Каждая entity card показывает:
   - title в исходном языке;
   - тип;
   - уровень;
   - документ;
   - раздел;
   - source quote;
   - trust grade.
2. Каждая trace card показывает:
   - source entity;
   - target entity;
   - relation;
   - verification mode;
   - similarity score;
   - evidence refs с обеих сторон.
3. Каждый finding cluster показывает:
   - почему это проблема;
   - сколько объектов затронуто;
   - какие документы/разделы вовлечены;
   - confidence/trust disclaimer.
4. Все документы кликабельны из HTML-отчёта.
5. Coverage раскрывается до документа и section, а не только до уровня.

### Explicitly Forbidden

- показывать similarity-only связь как strong trace;
- переводить русский title на английский без явного opt-in;
- строить итоговый summary только из counts без data-quality disclaimer;
- скрывать rejected/suspect объекты полностью.

---

## Acceptance Criteria

1. В `report.v2.json` каждая отображаемая entity содержит `document_path`,
   `section_id`, `source_quote`, `trust_grade`, `lang`.
2. В `report.v2.json` нет verified trace без `trace_evidence`.
3. HTML-отчёт даёт переход от finding -> entity -> document/section -> quote.
4. HTML-отчёт показывает отдельный блок `Corpus Quality`, а не смешивает
   source noise со strategy findings.
5. На русскоязычном корпусе title/description по умолчанию остаются на русском,
   если source quote на русском.
6. Prompt/system leakage не попадает в `verified` entity set.
7. Coverage можно посмотреть на трёх уровнях:
   - level
   - document
   - section
8. Финальная HTML-реализация сводится к переформатированию поверх уже готового
   JSON contract v2, без повторного redesign data model.

---

## Deferred

- manual analyst approval workflow;
- side-by-side diff двух audit runs;
- online collaborative viewer;
- graph database migration;
- ontology enrichment beyond the current entity types.

---

## Recommended Next Artifact

Следующий артефакт после этой design-spec:

- [docs/plans/2026-04-12-strataudit-evidence-report-redesign-plan.md](2026-04-12-strataudit-evidence-report-redesign-plan.md)

Его цель: разрезать дизайн на последовательные workstreams так, чтобы финальный
user-facing slice действительно был `report reformatting`, а не очередной
architecture rewrite без visible outcome.
