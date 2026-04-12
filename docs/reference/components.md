# SDP Components Catalog

> Статусы: 🟢 Production | 🟡 Beta (работает, MVP) | ⚠️ Needs wiring | 🔴 Planned (0 callers / не реализован)

**Дата:** 2026-04-11  
**Источник:** code audit + docs/ARCHITECTURE.md

---

## CLI Binaries (cmd/)

| Бинарь | Назначение | Фаза | Статус | Примечание |
|--------|-----------|------|--------|------------|
| `sdp` | Главный CLI (discover, architect, gate, ws, …) | All | 🟢 | Основная точка входа |
| `sdp-harness` | Запуск/восстановление agentloop сессий | Delivery | ⚠️ | Нужен LiveGateway (F106) |
| `sdp-orchestrate` | Feature-level orchestration | Delivery | 🟢 | |
| `sdp-orchestrate-daemon` | Daemon-вариант orchestrate | Delivery | 🟡 | |
| `sdp-guard` | Scope enforcement, out-of-scope detection | Delivery | 🟢 | |
| `sdp-ci-loop` | CI feedback loop (test → fix cycle) | Delivery | 🟢 | |
| `sdp-strataudit` | Стратегический LLM-аудит | Discovery | 🟢 | |
| `sdp-eval` | Eval runner | Delivery | 🟡 | |
| `sdp-evidence` | Evidence management CLI | Delivery | 🟢 | in-toto attestations |
| `sdp-doc-sync` | Doc link checker + sync | Infra | 🟢 | `--mode check --strict` |
| `sdp-beads-bridge` | Beads ↔ SDP bridge | Infra | 🟢 | |
| `sdp-control` | Control plane operations | Infra | 🟢 | |
| `sdp-dispatch` | Dispatch layer | Infra | 🟡 | |
| `sdp-a2a` | Agent-to-agent protocol binary | Infra | 🟡 | |
| `sdp-gh-findings-sync` | GitHub findings → Beads | Infra | 🟢 | |
| `sdp-ready` | Ready check (pre-flight) | Infra | 🟢 | |
| `sdp-up` | Project bootstrap | Infra | 🟢 | |
| `sdp-ws-verdict-validate` | Workstream verdict validation | Delivery | 🟢 | |
| `sdp-omc-guard` | OMC scope guard | Delivery | 🟡 | |
| `sdp-protocol-check` | Protocol validation | Infra | 🟢 | |

---

## Internal Packages (internal/)

### Discovery

| Пакет | Назначение | Статус | Примечание |
|-------|-----------|--------|------------|
| `internal/discovery` | 4-phase LLM pipeline: Frame→Hypothesize→Scan→Validate | 🟢 | Вызывает OpenRouter нативно |
| `internal/architect` | C4-анализ + runtime coupling detection | 🟢 | `sdp architect analyze` |
| `internal/strataudit` | Стратегический аудит с provider-neutral runtime | 🟢 | `sdp-strataudit run`, portable skill/harness integration |

### Delivery

| Пакет | Назначение | Статус | Примечание |
|-------|-----------|--------|------------|
| `internal/agentloop` | Phase FSM + GateEngine + EvidenceAccumulator + SQLite sessions | ⚠️ | Нужен LiveGateway (F106-WS01) |
| `internal/executor` | ServeBridge → DispatchAndRun → Harness | 🟢 | Точка входа для Delivery |
| `internal/orchestrate` | Feature-level phase orchestration | 🟢 | |
| `internal/gate` | Gate filesystem (`.sdp/gates/`) | 🟢 | |
| `internal/evidence` | in-toto attestations, EvidenceStore | 🟢 | |
| `internal/deploy` | Docker Compose wrapper, staging/prod gates | 🟡 | |
| `internal/guard` | Scope enforcement logic | 🟢 | |
| `internal/ciloop` | CI feedback loop logic | 🟢 | |
| `internal/eval` | Evaluation framework | 🟡 | |
| `internal/harness` | Harness lifecycle (вспомогательный) | 🟢 | |
| `internal/kernel` | Execution kernel primitives | 🟢 | |

### Infrastructure

| Пакет | Назначение | Статус | Примечание |
|-------|-----------|--------|------------|
| `internal/modelgateway` | LLM adapters: Anthropic/OpenAI/selfhosted, PolicyRouter | ⚠️ | Библиотека готова, 0 production callers до F106 |
| `internal/control` | FeatureCard store, ExecutorResultPacket | 🟢 | |
| `internal/beads` | Beads/Dolt integration | 🟢 | |
| `internal/session` | Session store abstractions (SQLite WAL) | 🟢 | |
| `internal/a2a` | Agent-to-agent protocol | 🟡 | |
| `internal/dispatch` | Dispatch routing | 🟡 | |
| `internal/monitor` | Metrics/monitoring | 🟡 | |
| `internal/docsync` | Doc sync + link checker | 🟢 | |
| `internal/workstream` | Workstream parsing + validation | 🟢 | |
| `internal/router` | Request routing | 🟢 | |
| `internal/runtime` | Runtime utilities | 🟢 | |
| `internal/profile` | Profile management | 🟡 | |
| `internal/policy` | Policy engine | 🟡 | |
| `internal/prompt` | Prompt construction utilities | 🟢 | |
| `internal/augmentation` | Context augmentation | 🟡 | |
| `internal/bridge` | Bridge abstractions | 🟢 | |
| `internal/cli` | CLI helpers | 🟢 | |
| `internal/sdputil` | SDP utilities | 🟢 | |
| `internal/gitutil` | Git utilities | 🟢 | |
| `internal/executil` | Exec utilities | 🟢 | |
| `internal/verify` | Verification tools | 🟢 | |
| `internal/tower` | Tower (orchestration layer) | 🟡 | |

### Не реализовано (0 callers)

| Пакет | Назначение | Статус | Примечание |
|-------|-----------|--------|------------|
| `internal/authz` | RBAC, authorization | 🔴 | Написан, 0 callers нигде |
| `internal/planner` | Task graph planner | 🔴 | Написан, 0 callers нигде |

---

## Skills (не бинари, не пакеты)

| Skill | Назначение | Статус |
|-------|-----------|--------|
| `skills/llm-council.md` | Multi-model deliberation | 🟢 |
| `skills/strataudit.md` | Portable strategy traceability audit skill | 🟢 |
| `skills/agent-dispatching.md` | Agent dispatch protocol | 🟢 |
| `AGENTS.md` | Agent instructions + SDP overview | 🟢 |

---

## Зависимости между слоями

```
Discovery ──────────────────────────────────────────────────────────
  internal/discovery  →  OpenRouter (HTTP)
  internal/architect  →  OpenRouter (HTTP)
  internal/strataudit →  injected host runtime | configured OpenAI-compatible runtime | OpenRouter
  skills/llm-council  →  OpenRouter (через agent)
  skills/strataudit   →  internal/strataudit | sdp-strataudit

Delivery ────────────────────────────────────────────────────────────
  internal/executor   →  internal/agentloop
  internal/agentloop  →  ModelGateway (интерфейс)
                      →  internal/gate
                      →  internal/evidence
                      →  internal/session (SQLite)
  ModelGateway impl   →  LiveGateway (F106) → OpenRouter

Infrastructure ──────────────────────────────────────────────────────
  internal/control    →  Beads (external CLI)
  internal/docsync    →  filesystem
  internal/ciloop     →  GitHub (via git + gh CLI)
```

---

## Точки входа по аудитории

| Аудитория | Команды | Документация |
|-----------|---------|-------------|
| Operator (запуск Discovery) | `sdp discover`, `sdp architect analyze` | [phases/DISCOVERY.md](../phases/DISCOVERY.md) |
| Operator (запуск Delivery) | `sdp-harness new/run`, `sdp-orchestrate` | [phases/DELIVERY.md](../phases/DELIVERY.md) |
| Agent (Discovery) | `sdp discover`, `sdp architect` | [guides/agent-discovery.md](../guides/agent-discovery.md) |
| Agent (Delivery) | `sdp-harness run`, `sdp-guard` | [guides/agent-delivery.md](../guides/agent-delivery.md) |
| Developer (setup) | `sdp-up`, `sdp-ready` | QUICKSTART.md |

---

*Полная архитектурная карта: [../ARCHITECTURE.md](../ARCHITECTURE.md)*
