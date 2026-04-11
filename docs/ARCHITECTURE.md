# SDP Architecture

> Статусы: 🟢 Production | 🟡 Beta (работает, MVP) | 🔴 Planned (не реализован) | ⚠️ Needs wiring

**Дата:** 2026-04-11  
**Это:** честная карта компонентов, а не roadmap.

---

## Обзор

```
User / Operator
     │
     ├── sdp discover / sdp architect  ──►  Discovery Layer
     │                                           │
     │                                      GO verdict + spec
     │                                           │
     └── sdp-harness / sdp-orchestrate  ──►  Delivery Layer
                                                 │
                                            Evidence + PR
                                                 │
                                         sdp deploy ──► Deploy
```

SDP — две первоклассные фазы:
- **Discovery:** исследование идеи → validated spec + scope decision
- **Delivery:** реализация → PR с gate-enforced evidence

---

## Discovery Layer

| Компонент | Назначение | Команда | Статус |
|-----------|-----------|---------|--------|
| `internal/discovery` | 4-phase LLM pipeline (Frame→Hypothesize→Scan→Validate) | `sdp discover` | 🟢 |
| `internal/architect` | C4 + код-анализ существующего кода | `sdp architect analyze` | 🟢 |
| `internal/strataudit` | Стратегический LLM-аудит | `sdp-strataudit run` | 🟢 |
| `skills/llm-council.md` | Multi-model deliberation для ключевых решений | (skill, не бинарь) | 🟢 |

Все компоненты Discovery активно вызывают OpenRouter через нативный HTTP client.

**Производит:** `docs/discovery/<slug>/` — frame.md, hypothesis.md, scan.md, validation.md, experiment.md

---

## Delivery Layer

| Компонент | Назначение | Команда | Статус |
|-----------|-----------|---------|--------|
| `internal/agentloop` | Phase FSM (Discover→Plan→Build→Review→Eval) + GateEngine + EvidenceAccumulator | `sdp-harness new/run` | ⚠️ нужен LiveGateway (F106) |
| `internal/executor` | ServeBridge → Harness orchestration, DispatchAndRun | (library) | 🟢 |
| `internal/orchestrate` | Feature-level phase orchestration | `sdp-orchestrate` | 🟢 |
| `internal/gate` | Gate filesystem (`.sdp/gates/`) | (library) | 🟢 |
| `internal/evidence` | in-toto attestation, EvidenceStore | `sdp-evidence` | 🟢 |
| `internal/deploy` | Docker Compose wrapper + staging/prod gates | (library) | 🟡 |
| `internal/guard` | Scope enforcement, out-of-scope detection | `sdp-guard` | 🟢 |
| `internal/ciloop` | CI feedback loop | `sdp-ci-loop` | 🟢 |

### agentloop подробнее

`internal/agentloop` — ядро Delivery. Фазовый FSM:

```
Discover → Plan → Build → Review → Eval
    │          │       │        │      │
    └──gates───┴───────┴────────┴──────┘
              GateEngine (5s circuit breaker)
```

- **GateEngine:** оценивает gate из tool call evidence (EvidenceAccumulator)
- **EvidenceAccumulator:** извлекает доказательства только из real tool outputs — не из текста агента
- **SQLite WAL sessions:** каждая сессия — `.sdp/sessions/<cardID>.db`
- **Статус:** логика завершена; нет production callers пока не будет F106 (LiveGateway)

---

## Infrastructure Layer

| Компонент | Назначение | Статус |
|-----------|-----------|--------|
| `internal/modelgateway` | LLM adapters (Anthropic/OpenAI/selfhosted), PolicyRouter | 🟡 0 production callers |
| `internal/control` | FeatureCard store, ExecutorResultPacket | 🟢 |
| `internal/beads` | Beads/Dolt issue tracker интеграция | 🟢 |
| `internal/a2a` | Agent-to-agent protocol | 🟡 |
| `internal/authz` | RBAC | 🔴 0 callers |
| `internal/planner` | Task graph planner | 🔴 0 callers |
| `internal/monitor` | Metrics/monitoring | 🟡 |
| `internal/session` | Session store abstractions | 🟢 |
| `internal/docsync` | Doc sync tooling | `sdp-doc-sync` | 🟢 |

---

## CLI Binaries (cmd/)

| Бинарь | Назначение | Фаза | Статус |
|--------|-----------|------|--------|
| `sdp` | Главный CLI (discover, architect, gate, ws…) | All | 🟢 |
| `sdp-harness` | Запуск agentloop сессий | Delivery | ⚠️ нужен LiveGateway |
| `sdp-orchestrate` | Feature-level orchestration | Delivery | 🟢 |
| `sdp-orchestrate-daemon` | Daemon вариант orchestrate | Delivery | 🟡 |
| `sdp-guard` | Scope enforcement | Delivery | 🟢 |
| `sdp-ci-loop` | CI feedback loop | Delivery | 🟢 |
| `sdp-strataudit` | Стратегический аудит | Discovery | 🟢 |
| `sdp-eval` | Eval runner | Delivery | 🟡 |
| `sdp-evidence` | Evidence management CLI | Delivery | 🟢 |
| `sdp-doc-sync` | Doc link checker + sync | Infra | 🟢 |
| `sdp-beads-bridge` | Beads ↔ SDP bridge | Infra | 🟢 |
| `sdp-control` | Control plane operations | Infra | 🟢 |
| `sdp-dispatch` | Dispatch layer | Infra | 🟡 |
| `sdp-a2a` | Agent-to-agent protocol binary | Infra | 🟡 |
| `sdp-gh-findings-sync` | GitHub findings → Beads | Infra | 🟢 |
| `sdp-ready` | Ready check | Infra | 🟢 |
| `sdp-up` | Project bootstrap | Infra | 🟢 |
| `sdp-ws-verdict-validate` | Workstream verdict validation | Delivery | 🟢 |
| `sdp-omc-guard` | OMC scope guard | Delivery | 🟡 |
| `sdp-protocol-check` | Protocol validation | Infra | 🟢 |

---

## Интеграции

| Интеграция | Способ | Статус |
|-----------|--------|--------|
| OpenRouter | HTTP (native) — Discovery и будущий LiveGateway | 🟢 |
| Beads / Dolt | `bd` CLI + git-backed Dolt DB | 🟢 |
| GitHub | `sdp-gh-findings-sync`, `sdp-ci-loop` | 🟢 |
| Harnesses (Claude Code, Cursor) | MCP tools (поверхность взаимодействия) | 🟡 |
| Docker / Docker Compose | `internal/deploy` | 🟡 |

**Важно о harnesses:** Claude Code и Cursor — внешние инструменты с подписками. SDP не перехватывает их LLM-трафик (нельзя потерять подписку). Интеграция через MCP: SDP экспортирует tools → harness вызывает их через MCP protocol.

---

## Что работает сейчас vs. Что planned

### Работает сейчас 🟢

- Discovery pipeline (`sdp discover`) — полный 4-phase LLM run
- Architect analysis (`sdp architect analyze`) — C4 + runtime coupling
- Strategic audit (`sdp-strataudit run`) — high-level LLM audit
- LLM Council (skill) — multi-model deliberation
- Gate filesystem — enforcement через файловую систему
- Evidence store — in-toto attestations
- Scope guard — out-of-scope detection
- CI loop — continuous integration feedback

### Работает, нужна доработка ⚠️

- **agentloop + sdp-harness** — FSM логика готова, нужен LiveGateway (F106)
- **modelgateway** — адаптеры написаны, 0 production callers до F106

### Не реализовано 🔴

- `internal/authz` — RBAC (0 callers)
- `internal/planner` — task graph planner (0 callers)
- MCP server для harness интеграции

---

## Ключевые потоки данных

### Discovery flow
```
sdp discover "идея"
  → internal/discovery.Run()
  → Frame() → Hypothesize() → Scan() → Validate()
  → docs/discovery/<slug>/*.md
  → GO verdict → Beads feature card
```

### Delivery flow
```
bd ready → workstream backlog
  → sdp-harness new --session=<id>
  → internal/executor.ServeBridge.DispatchAndRun()
  → internal/agentloop.Harness.RunPhase()
  → ModelGateway → OpenRouter (F106)
  → GateEngine evaluates from EvidenceAccumulator
  → .sdp/sessions/<id>.db (SQLite WAL)
  → gate pass → next phase
```

---

Детальный каталог компонентов с метаданными: [reference/components.md](reference/components.md)
