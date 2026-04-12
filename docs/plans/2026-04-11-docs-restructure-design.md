# Docs Restructure Design — SDP Documentation Overhaul

**Date:** 2026-04-11
**Status:** Approved
**Owner:** Андрей

---

## Проблема

Проект не может ответить на вопрос "что такое SDP" из-за:
1. Vision нигде не написан явно — четыре конкурирующих фрейминга
2. Discovery реализована в коде, но не существует как фаза в документации
3. Код и документация описывают разные системы (aspirational vs. real)
4. 683 MD файла, из которых ~600 — рабочий дневник, не документация

## Решение

Новая структура с принципом **глубины, а не ширины**.
Один вход на аудиторию. Фазы системы отражены в структуре docs.

## Аудитория

| Аудитория | Вход | Путь |
|-----------|------|------|
| Андрей (product owner) | `VISION.md` | VISION → ARCHITECTURE → happy-paths |
| Агент (строит SDP) | `AGENTS.md` | AGENTS → phases/ → guides/ → reference/ |
| Discovery агент | `docs/guides/agent-discovery.md` | guide → phases/DISCOVERY → skills/ |
| Delivery агент | `docs/guides/agent-delivery.md` | guide → phases/DELIVERY → workstream |
| Внешний разработчик | `sdp/QUICKSTART.md` | QUICKSTART → sdp/docs/ |

## Структура (sdp_lab — private)

```
sdp_lab/
├── VISION.md                        ← NEW. Что такое SDP. 1 страница.
├── AGENTS.md                        ← MIGRATE. Добавить: SDP overview, Discovery/Delivery, llm-council, agentloop.
│
└── docs/
    ├── ARCHITECTURE.md              ← NEW. Компоненты + потоки.
    ├── TERMS.md                     ← NEW (на базе GLOSSARY.md). Единый глоссарий.
    │
    ├── phases/
    │   ├── DISCOVERY.md             ← NEW. Что такое Discovery: шаги, агенты, артефакты.
    │   └── DELIVERY.md              ← NEW. Что такое Delivery: фазы agentloop, gates, агенты.
    │
    ├── happy-paths/
    │   ├── greenfield.md            ← NEW. Новый проект с нуля.
    │   ├── new-feature.md           ← MIGRATE из canonical-happy-path.md
    │   ├── brownfield.md            ← NEW. Встройка в legacy.
    │   └── cold-start.md            ← NEW. Первый запуск SDP.
    │
    ├── critical-paths/
    │   ├── evidence-chain.md        ← MIGRATE из critical-paths + evidence docs.
    │   └── gate-enforcement.md      ← MIGRATE из quality-gates.md
    │
    ├── roadmap/
    │   └── ROADMAP.md               ← KEEP (обновить направление)
    │
    ├── decisions/                   ← KEEP (ADRs без изменений)
    │
    ├── reference/
    │   ├── components.md            ← NEW. Каталог cmd/ + internal/ (реальное состояние).
    │   ├── skills.md                ← MIGRATE. Добавить llm-council.
    │   ├── agent-catalog.md         ← KEEP.
    │   └── quality-gates.md         ← KEEP.
    │
    ├── guides/
    │   ├── agent-discovery.md       ← NEW. Полный протокол Discovery агента.
    │   ├── agent-delivery.md        ← NEW. Полный протокол Delivery агента.
    │   └── operator.md              ← NEW. Как человек запускает и следит.
    │
    └── archive/                     ← MOVE. ~400 файлов из docs/plans/, docs/ root ALLCAPS.
        ├── plans/                   ← все plans/ старше 2026-04-01 (кроме активных)
        ├── k8s/                     ← все K8s документы
        ├── next-steps/              ← все *_NEXT_STEP.md
        └── council-rounds/          ← все council round artifacts
```

## Структура (sdp/ — public)

```
sdp/
├── VISION.md                        ← NEW (публичная выжимка).
├── QUICKSTART.md                    ← MIGRATE. Добавить Discovery/Delivery явно.
├── CLAUDE.md                        ← MIGRATE. Синхронизировать со skills.md.
└── docs/
    ├── DISCOVERY.md                 ← NEW. Публичный гайд.
    ├── DELIVERY.md                  ← NEW. Публичный гайд.
    └── TERMS.md                     ← NEW. Публичный глоссарий.
```

## Что создать (новые документы)

| Документ | Приоритет | Основа |
|----------|-----------|--------|
| `VISION.md` | P0 | С нуля (vision нигде не написан) |
| `ARCHITECTURE.md` | P0 | С нуля + PRIVATE_BLUEPRINT (устарел) |
| `TERMS.md` | P0 | На базе GLOSSARY.md (2026-01-29) + обновить |
| `docs/phases/DISCOVERY.md` | P0 | discovery-pipeline-design.md + code audit |
| `docs/phases/DELIVERY.md` | P0 | CANONICAL_SDP_PIPELINE + code audit |
| `docs/reference/components.md` | P0 | Из code audit (что реально работает) |
| `docs/guides/agent-discovery.md` | P1 | С нуля |
| `docs/guides/agent-delivery.md` | P1 | С нуля |
| `docs/guides/operator.md` | P1 | Из SDP_OPERATOR_WORKFLOW.md |
| `docs/happy-paths/*.md` | P1 | Из canonical-happy-path + новые сценарии |
| `docs/critical-paths/*.md` | P2 | Из существующих evidence/gate docs |

## Что мигрировать

| Документ | Действие |
|----------|----------|
| `AGENTS.md` | Добавить преамбулу о системе, Discovery/Delivery, llm-council, agentloop |
| `MANIFESTO.md` | Обновить: убрать "evidence layer", добавить AI PDLC+SDLC |
| `CANONICAL_SDP_PIPELINE.md` | Разделить реальное vs aspirational, добавить Discovery |
| `sdp/PRODUCT_VISION.md` | Переписать: "evidence layer" → AI PDLC+SDLC |
| `docs/reference/skills.md` | Добавить llm-council, обновить @idea/@design |
| `sdp/QUICKSTART.md` | Добавить явный Discovery/Delivery раздел |
| `sdp/CLAUDE.md` | Синхронизировать с canonical skills |

## Что архивировать

- ~400 файлов: council rounds, plans/ до 2026-04-01, K8s docs, *_NEXT_STEP.md
- `PRIVATE_BLUEPRINT.md` (L3 + 3 OSS repos — устарело)
- `prd/PRD.md` (K8s-driven FR-001..FR-005)
- Старые findings (оставить только финальные версии)
- Дубликаты roadmap (UNIFIED_VISION_ROADMAP_2026-03-03.md и др.)

## Порядок работы

1. **P0: Создать критические документы** — VISION, ARCHITECTURE, TERMS, phases/
2. **P0: Обновить входные точки** — AGENTS.md, sdp/PRODUCT_VISION.md
3. **P1: Создать guides и happy-paths**
4. **P1: Создать reference/components.md** (актуальный каталог кода)
5. **P2: Архивировать** — переместить ~400 файлов в docs/archive/
6. **P2: Обновить sdp/** — публичные docs
