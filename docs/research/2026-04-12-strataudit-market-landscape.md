# StratAudit Market Landscape — Strategy Traceability And Report UX

**Date:** 2026-04-12
**Status:** Working research note
**Purpose:** зафиксировать, кто уже продаёт strategy/traceability продукты,
что именно они обещают, и где остаётся окно для StratAudit.

---

## Market Buckets

### 1. Enterprise Architecture / Transformation

Игроки:

- Ardoq
- SAP LeanIX
- Bizzdesign

Что продают:

- strategy-to-execution visibility;
- capability/application landscape;
- roadmap and dependency planning;
- impact analysis for transformation programs.

Сильные стороны:

- хорошая executive-упаковка;
- зрелые модели зависимостей и heatmaps;
- понятный story для CIO/enterprise architecture teams.

Ограничения для нашего кейса:

- обычно стартуют со структурированных fact sheets, imports и ручного моделирования;
- плохо решают messy corpus ingestion из PDF/DOC/Confluence exports;
- доказательная связь до цитаты/раздела не является центром продукта.

### 2. Strategy Execution / OKR Platforms

Игроки:

- Cascade
- WorkBoardAI
- Quantive

Что продают:

- alignment;
- strategic execution;
- KPI/OKR cadence;
- AI assistance for operating rhythm and leadership visibility.

Сильные стороны:

- отличный executive UX;
- heatmaps, alignment views, dashboards;
- narrative around "source of truth for strategy execution".

Ограничения для нашего кейса:

- эти продукты предполагают, что структура целей уже заведена в систему;
- они не умеют честно добывать смысл из грязного корпуса документов;
- evidence layer до source quote почти не является differentiator.

### 3. Requirements / Traceability Platforms

Игроки:

- Jama Connect
- IBM DOORS Next
- Siemens Polarion
- PTC Codebeamer
- Visure

Что продают:

- end-to-end traceability;
- impact analysis;
- suspect links;
- auditability and compliance readiness.

Что у них стоит брать:

- trace explorer;
- матрицы трассировки;
- явное разделение verified links и suspect links;
- coverage как first-class объект.

Где они слабее нас:

- ориентированы на already-structured requirements artifacts;
- слабая история для multilingual office-document mining;
- poor fit for "throw a folder of PDF/DOC files and get evidence-backed strategy map".

### 4. OSS / Docs-As-Code Traceability

Игроки:

- Sphinx-Needs
- StrictDoc
- Doorstop
- Eclipse Capra

Что у них ценно:

- link-as-data mindset;
- прозрачные IDs;
- exportable graph/matrix;
- проверяемость и реплицируемость.

Ограничения:

- почти все требуют ручной дисциплины;
- ingestion слабее enterprise document reality;
- UX ориентирован на engineers, не на mixed exec+analyst audience.

---

## Common Market Promises

Почти все игроки обещают одно из трёх:

1. **Visibility** — "мы покажем, как стратегия связана с исполнением"
2. **Alignment** — "мы найдём disconnects и orphan work"
3. **Governance** — "мы дадим auditability и impact analysis"

Почти никто не обещает одновременно:

- ingestion грязного русского/многоязычного офисного корпуса;
- traceability до цитаты и раздела;
- явную trust-модель для extracted entities and traces;
- честное разделение strategy findings и corpus quality problems.

---

## What To Borrow

### Borrow From Strategy Execution Tools

- executive overview first;
- alignment heatmap;
- grouped view of disconnects instead of flat alert feed.

### Borrow From Requirements Traceability Tools

- suspect link semantics;
- trace matrix / explorer;
- impact-style drill-down;
- coverage as structured data, not decorative KPI.

### Borrow From OSS Traceability

- IDs and explicit links;
- exportable machine-readable contract;
- evidence visible without hidden server logic.

---

## White Space

На рынке почти пусто место для продукта с таким сочетанием:

1. **messy document corpus ingestion**
2. **multilingual extraction with source preservation**
3. **evidence-backed strategy traceability**
4. **report UX for both executives and analysts**

Именно это окно и стоит занимать StratAudit.

Не "ещё один dashboard для strategy execution", а:

> evidence-backed strategy mining and traceability over messy enterprise documents

---

## Product Implications For StratAudit

### 1. Compete On Trust, Not On Dashboard Cosmetics

Красивая HTML-страница сама по себе не бьёт LeanIX/WorkBoard. Наше отличие —
доказуемость и provenance.

### 2. Keep The Analyst Path Strong

Рынок любит executive storytelling. Нам нельзя потерять analyst drill-down:
документ, section, quote, trace evidence.

### 3. Surface Corpus Quality Explicitly

У конкурентов source corpus обычно уже очищен и структурирован. Для нас это
неверно, значит диагностика корпуса должна быть частью продукта.

### 4. Make JSON Contract A Real Asset

В OSS и compliance-like tools сильна идея exportable truth. `report.v2.json`
должен быть полноценным артефактом, а не побочным дампом для HTML.

---

## Bottom Line

Если StratAudit будет просто "рисовать отчёт", он проиграет зрелым strategy and
EA tools по polish и breadth.

Если StratAudit станет **trustworthy evidence layer over messy strategy corpus**,
он зайдёт в нишу, которую рынок закрывает слабо или не закрывает вообще.
