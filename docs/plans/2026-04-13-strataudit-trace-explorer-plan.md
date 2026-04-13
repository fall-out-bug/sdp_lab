# StratAudit — Claim-Centric Trace Explorer Plan

**Date:** 2026-04-13
**Status:** Proposed
**Feature:** F117
**Spec:** [2026-04-13-strataudit-trace-explorer-design.md](2026-04-13-strataudit-trace-explorer-design.md)

---

## Outcome

После выполнения `F117` StratAudit должен:

1. объяснять трассировку через claim graph, а не через длинный report dump;
2. показывать doc-to-doc correspondence как агрегат над trace model;
3. материализовывать gap reasons, а не молчаливый `0 traces`;
4. разделять analyst report и diagnostics по разным табам;
5. оставлять compare-mode вне основного отчёта.

---

## Workstreams

### WS-01: Claim-Centric Trace Graph Contract

**Why:** пока отчёт знает только про entities/traces/findings, он не умеет
объяснить trace path и trace failure как first-class объекты.

**Changes:**

- add `trace_graph.nodes`, `trace_graph.edges`, `trace_graph.paths` в report contract;
- materialize edge status `verified|candidate|rejected`;
- keep contract additive on top of `report.v2.json`.

**Acceptance:**

- каждый trace node ссылается на entity, document, section и quote evidence;
- каждый edge несёт status, verification mode, confidence, reason;
- HTML может строить trace explorer без прямого чтения SQLite.

### WS-02: Gap Taxonomy And Link Diagnostics

**Why:** `0 traces` без объяснения бесполезен. Нужен явный waterfall причин.

**Changes:**

- add first-class `trace_gaps`;
- persist link-stage rejection reasons и candidate counts;
- compute stage-aware gap diagnostics.

**Acceptance:**

- boundary claim получает либо edge, либо gap;
- gap фиксирует stage, reason, expected next level и top candidates;
- default analytics can explain why a path stopped.

### WS-03: Document Correspondence Read Model

**Why:** пользователь мыслит документами, но correspondence нельзя вычислять
ручным prose после рендера.

**Changes:**

- add `document_views` as aggregate over claim graph;
- compute upstream/downstream document correspondence;
- expose blockers, coverage and key claims per document.

**Acceptance:**

- у каждого документа есть explainable list соответствующих upstream/downstream docs;
- correspondence выводится из nodes/edges/gaps, а не из UI heuristics;
- report can answer "что чему соответствует" по документам.

### WS-04: Tabbed Analyst Report IA

**Why:** перегруженный single-page report всегда будет ломать UX.

**Changes:**

- rebuild HTML as tabs: `Сводка`, `Документы`, `Трассировка`, `Разрывы`, `Диагностика`;
- move compare/debug surfaces out of default flow;
- normalize Russian chrome and navigation labels.

**Acceptance:**

- tabs are explicit and stable;
- diagnostics no longer pollute summary/doc views;
- default report opens in analyst mode, not compare mode.

### WS-05: Real-Corpus UX Acceptance Harness

**Why:** synthetic smoke tests не ловят перегрузку отчёта и silent empty states.

**Changes:**

- add fixture/assertions for tab presence, document correspondence, trace gap waterfall;
- add red flags for mixed-language chrome and fake empty graph;
- validate report against realistic Russian corpus shape or minimized reproducible fixture.

**Acceptance:**

- regression fails if report loses tabs or document correspondence;
- regression fails if zero-trace run lacks gap explanation;
- regression fails if main report reintroduces compare-first flow.

---

## Execution Order

```text
WS-01 → WS-02
WS-01 → WS-03
WS-01 + WS-02 + WS-03 → WS-04
WS-04 → WS-05
```

`WS-04` deliberately waits for the data/read-model slices. Иначе UI снова
превратится в подрисовку поверх неполных структур.

---

## Delivery Slices

### Slice A: Trace Substrate

- WS-01
- WS-02

**Visible result:** claim graph и gap reasons становятся first-class model.

### Slice B: Document Analysis Surface

- WS-03

**Visible result:** понятная correspondence-карта документов.

### Slice C: Analyst Report

- WS-04
- WS-05

**Visible result:** tabbed report, который объясняет trace paths и trace breaks.

---

## Explicit Stop Conditions

Stop and revisit the design if any of these happen:

1. document view снова становится заменой claim graph, а не агрегатом над ним;
2. zero-trace state всё ещё сводится к пустому графу или общему alert count;
3. compare mode возвращается в основной analyst flow;
4. diagnostics снова смешиваются с product summary;
5. report contract требует SQLite/store access для базового trace explorer.

---

## Recommended Commit Sequence

1. `design(strataudit): claim-centric trace explorer spec`
2. `plan(strataudit): workstreams for trace explorer`
3. `docs(strataudit): register F117 in roadmap, index, backlog, and beads`
