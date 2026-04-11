# StratAudit v2 — Evidence-Backed Report Redesign Plan

**Date:** 2026-04-12
**Status:** Proposed
**Feature:** F109
**Spec:** [2026-04-12-strataudit-evidence-report-redesign-design.md](2026-04-12-strataudit-evidence-report-redesign-design.md)
**Goal:** довести StratAudit до состояния, где финальный пользовательский этап
есть переформатирование отчёта поверх честного evidence contract, а не попытка
скрыть слабые данные красивым HTML.

---

## Outcome

После выполнения плана StratAudit должен:

1. отделять `candidate` от `verified truth`;
2. сохранять язык источника;
3. показывать provenance до уровня section/quote;
4. раскрывать coverage по документам и разделам;
5. отдавать `report.v2.json` как главный продуктовый контракт;
6. генерировать HTML-отчёт нового формата без переизобретения data layer.

---

## Workstreams

### WS-01: Extraction Trust Gate

**Why:** пока модель может сохранить system prompt как `source_quote`, весь
остальной отчёт недостоверен.

**Changes:**

- validate `source_quote` against chunk/document text;
- reject or downgrade entities with quote mismatch;
- detect prompt leakage and boilerplate patterns;
- add `trust_grade` and `quality_flags` to entity pipeline.

**Acceptance:**

- prompt-leak strings не попадают в verified entities;
- quote mismatch отражается в diagnostics;
- `report.v2.json` содержит counters rejected/suspect/verified entities.

### WS-02: Source-Preserving Multilingual Policy

**Why:** скрытый перевод ломает доверие и мешает проверке.

**Changes:**

- extraction prompt forces source-language preservation;
- add `title_original`, `description_original`, `lang`, `language_mismatch`;
- disallow English rewrite for Russian source by default.

**Acceptance:**

- русскоязычные сущности отображаются на русском;
- language drift фиксируется как отдельный quality signal;
- display fields не меняют язык без explicit opt-in.

### WS-03: Provenance Layer — Documents, Sections, Quote Spans

**Why:** без section/span report cannot explain where an entity or trace came from.

**Changes:**

- add first-class `sections`;
- persist section metadata and quote spans;
- bind entities to sections and quotes.

**Acceptance:**

- каждая verified entity ссылается на section и quote span;
- coverage строится не только по level, но и по document/section;
- document links can point to exact source file.

### WS-04: Trace Evidence Contract

**Why:** similarity-only trace must stop pretending to be proof.

**Changes:**

- split `trace_candidates` and `verified_traces`;
- verified trace requires `trace_evidence`;
- persist verification mode, similarity score, trust grade;
- demote similarity-only links to diagnostics.

**Acceptance:**

- no verified trace without evidence bundle;
- report shows verification mode explicitly;
- trace explorer can open source and target evidence.

### WS-05: Findings And Coverage Regrouping

**Why:** one-entity-one-alert produces noise, not insight.

**Changes:**

- introduce grouped findings;
- add corpus-quality clusters;
- compute coverage at three levels: level/document/section.

**Acceptance:**

- default HTML shows clusters, not raw spam;
- operators can separate strategy gaps from corpus-quality failures;
- report highlights top blocking documents/sections.

### WS-06: Report JSON v2 Contract

**Why:** HTML must consume a stable report contract instead of reaching into ad hoc structs.

**Changes:**

- design and implement `report.v2.json`;
- include audit scope, trust summary, corpus quality, entities, traces,
  grouped findings, coverage, evidence pack;
- add schema versioning and golden tests.

**Acceptance:**

- HTML generator uses only JSON v2 data model;
- JSON has enough data for offline analysis without SQLite access;
- golden test on UBRIR-style sample verifies stable shape.

### WS-07: HTML Report Reformatting

**Why:** only after WS-01..WS-06 does visual redesign become honest product work.

**Changes:**

- build three-layer report IA;
- add drill-down navigation;
- add local document links and evidence cards;
- add corpus quality dashboard.

**Acceptance:**

- from any finding, user reaches document and quote in <=3 clicks;
- coverage and traces are explorable, not just counted;
- exec overview has explicit trust disclaimer.

### WS-08: Regression Harness On Real Corpus

**Why:** StratAudit is highly sensitive to corpus mess. Without regression on a
realistic Russian corpus, the redesign will drift back into toy demos.

**Changes:**

- add golden corpus fixture or reproducible audit fixture;
- snapshot JSON v2 invariants;
- add red-flag checks for prompt leak, language drift, missing document links.

**Acceptance:**

- regression fails if prompt/system strings reappear in entities;
- regression fails if verified traces lose evidence;
- regression fails if HTML/JSON drop document provenance fields.

---

## Execution Order

```
WS-01 → WS-02 → WS-03 → WS-04 → WS-05 → WS-06 → WS-07
  \                                             /
   └──────────────────────→ WS-08 ─────────────┘
```

Parallelism is limited on purpose. This is a trust-sensitive redesign, not a
"раскидаем по пяти агентам и потом склеим" задача.

---

## Delivery Slices

### Slice A: Trust Substrate

- WS-01
- WS-02
- WS-03

**Visible result:** trustworthy entities with source-preserving provenance.

### Slice B: Analytical Contract

- WS-04
- WS-05
- WS-06

**Visible result:** `report.v2.json` becomes the real audit product.

### Slice C: Final Report Reformat

- WS-07
- WS-08

**Visible result:** new HTML report with drill-down, links, coverage, traces,
and trust disclaimers.

---

## Explicit Stop Conditions

Stop and revisit the design if any of these happen:

1. verified traces still rely on similarity-only evidence;
2. multilingual policy still allows silent translation of titles;
3. JSON v2 still cannot reconstruct document/section provenance offline;
4. HTML reformatting starts before JSON v2 contract is stable;
5. grouped findings regress back to one-entity-one-alert spam.

---

## Recommended Commit Sequence

1. `design(strataudit): evidence-backed report redesign spec`
2. `plan(strataudit): workstreams for evidence-backed report redesign`
3. `research(strataudit): market landscape and product takeaways`

This keeps design intent, implementation slicing, and market rationale separate.
