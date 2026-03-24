# Beads SDP schema

Status: canon draft  
Date: 2026-03-24  
Scope: canonical SDP metadata schema, issue taxonomy, gate taxonomy, dependency model, status mapping for Beads-first SDP

## 1. Canonical metadata schema

SDP использует стандартный `beads/types.Issue` и расширяет его через `Issue.Metadata`.

### 1.1 Envelope

```json
{
  "schema": "sdp/beads-metadata/v1",
  "sdp": {
    "card_id": "feature-sdp_lab-2026-03-24-001",
    "project_id": "sdp_lab",
    "phase": "clarify",
    "task_type": "architecture",
    "execution_mode": "repo",
    "risk_level": "high",
    "intent": {
      "normalized": "Перевести SDP на beads-first operational truth"
    },
    "scope": {
      "in": ["docs/", "internal/control"],
      "out": ["переписывание beads core"]
    },
    "non_goals": ["строить вторую task graph модель"],
    "why_now": "нужно прекратить split-brain между FeatureCard и Beads",
    "target": {
      "repo": "sdp_lab",
      "area": "control"
    },
    "source_refs": [
      "docs/BEADS_FIRST_CONTROL_TOWER_ROADMAP.md",
      "docs/SDP_SPEC_DRIVEN_PIPELINE_CANON.md"
    ],
    "links": ["https://example.local/adr/123"],
    "clarification": {
      "open_questions": [],
      "last_classification": "additive"
    },
    "contract": {
      "id": "CTR-001",
      "ref": ".sdp/contracts/CTR-001.json",
      "hash": "sha256:abc",
      "required_artifacts": ["contract.json", "review.md"],
      "required_checks": ["go test ./...", "golangci-lint run"]
    },
    "executor": {
      "role": "omo-implementation",
      "dispatched_to": "opencode",
      "packet_ref": ".sdp/dispatch/packet-001.json",
      "session_id": "sess-123",
      "state": "running",
      "started_at": "2026-03-24T08:30:00Z",
      "last_heartbeat_at": "2026-03-24T08:42:00Z",
      "progress_summary": "Обновлена schema документация",
      "active_agents": ["orchestrator", "implementer"],
      "last_result": {
        "status": "success",
        "summary": "Документы созданы",
        "received_at": "2026-03-24T08:50:00Z",
        "artifacts": ["docs/BEADS_SDP_SCHEMA.md"],
        "findings": [],
        "open_risks": [],
        "recommended_next_step": "start_review"
      }
    },
    "review": {
      "state": "pending",
      "attempts": 0,
      "summary": "",
      "ref": ""
    },
    "delivery": {
      "state": "pending",
      "target": "staging",
      "summary": "",
      "ref": "",
      "delivered_at": "",
      "rollback_ref": "",
      "rollback_summary": ""
    },
    "gates": {
      "open": ["sdp-gate-h01"],
      "satisfied": [],
      "failed": []
    },
    "blocking": {
      "reasons": ["ожидается human approval"],
      "waiting_on": ["andrey"],
      "needs_feedback_from": ["andrey"],
      "feedback_request": ["Подтвердить cutover semantics"],
      "decision_required": ["Одобрить ADR"],
      "author_update": [],
      "admin_action_required": []
    },
    "artifacts": {
      "intake": [".sdp/control/projects/sdp_lab/intake/feature-sdp_lab-2026-03-24-001.md"],
      "linked": ["docs/FEATURECARD_BEADS_MAPPING.md"],
      "evidence": [".sdp/evidence/EVD-001/index.json"],
      "provenance": [".sdp/provenance/PROV-001.json"]
    },
    "followups": [],
    "workstreams": [],
    "policy": {
      "constitution_warnings": []
    },
    "orchestration": {
      "last_action": "mark_ready",
      "last_reason": "Все обязательные поля заполнены",
      "last_at": "2026-03-24T08:00:00Z"
    },
    "next": {
      "action": "dispatch",
      "step": "start_execution",
      "reason": "Нет открытых gate и blocking deps"
    },
    "counters": {
      "clarification_cycles": 1,
      "blocked_cycles": 0,
      "execution_attempt_count": 0,
      "review_fail_count": 0,
      "rollback_count": 0
    },
    "provenance": {
      "intake_hash": "sha256:111",
      "contract_hash": "sha256:222",
      "dispatch_packet_hash": "sha256:333",
      "prompt_hash": "sha256:444",
      "result_packet_hash": "sha256:555"
    }
  }
}
```

### 1.2 JSON schema (logical)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "sdp/beads-metadata/v1",
  "type": "object",
  "required": ["schema", "sdp"],
  "properties": {
    "schema": {
      "const": "sdp/beads-metadata/v1"
    },
    "sdp": {
      "type": "object",
      "required": ["card_id", "project_id", "phase"],
      "properties": {
        "card_id": {"type": "string"},
        "project_id": {"type": "string"},
        "parent_bead_id": {"type": "string"},
        "phase": {
          "type": "string",
          "enum": ["intake", "clarify", "ready", "contract", "build", "review", "qa", "release", "done", "deferred"]
        },
        "task_type": {"type": "string"},
        "execution_mode": {"type": "string"},
        "risk_level": {
          "type": "string",
          "enum": ["low", "medium", "high"]
        },
        "intent": {
          "type": "object",
          "properties": {
            "normalized": {"type": "string"}
          }
        },
        "scope": {
          "type": "object",
          "properties": {
            "in": {"type": "array", "items": {"type": "string"}},
            "out": {"type": "array", "items": {"type": "string"}}
          }
        },
        "non_goals": {"type": "array", "items": {"type": "string"}},
        "clarification": {"type": "object"},
        "contract": {"type": "object"},
        "executor": {"type": "object"},
        "review": {"type": "object"},
        "delivery": {"type": "object"},
        "gates": {"type": "object"},
        "blocking": {"type": "object"},
        "artifacts": {"type": "object"},
        "followups": {"type": "array", "items": {"type": "string"}},
        "workstreams": {"type": "array", "items": {"type": "string"}},
        "policy": {"type": "object"},
        "orchestration": {"type": "object"},
        "next": {"type": "object"},
        "counters": {"type": "object"},
        "provenance": {"type": "object"}
      },
      "additionalProperties": true
    }
  },
  "additionalProperties": true
}
```

### 1.3 Обязательные vs optional поля

| Поле | Статус | Назначение |
|---|---|---|
| `schema` | required | Версия metadata schema |
| `sdp.card_id` | required | Stable ref к semantic card |
| `sdp.project_id` | required | Привязка к проекту |
| `sdp.phase` | required | SDP pipeline phase |
| `sdp.intent.normalized` | strongly recommended | Человеко-читаемый intent |
| `sdp.contract.*` | required начиная с phase=`contract` | Executable spec refs |
| `sdp.executor.*` | required во время dispatch/execute | Runtime trace |
| `sdp.review.*` | required для review beads и review phase | Review state |
| `sdp.delivery.*` | required для release/delivery | Delivery trace |
| `sdp.gates.*` | required если есть gates | Gate index |
| `sdp.provenance.*` | recommended | Traceability |

---

## 2. Issue type taxonomy для SDP

SDP использует Beads issue types как operational categories.

| Issue type | Назначение | Кто создаёт | Когда закрывается |
|---|---|---|---|
| `feature` | Главный work item / feature-level execution unit | intake/control tower | Когда весь pipeline завершён |
| `clarify` | Уточнение требований, scope, policy-sensitive change | orchestrator / human loop | Когда вопросы закрыты или merged into parent |
| `contract` | Генерация или ревизия executable contract | contract generator / orchestrator | Когда contract утверждён и зафиксирован |
| `review` | Reviewer pass поверх результата | review agent | Когда review approved или заменён новой итерацией |
| `qa` | QA/verifier pass | qa agent | Когда QA_PASS или зафиксирован QA_FAIL |
| `release` | Deploy/release activity | release orchestration | Когда deploy завершён или отменён |
| `gate:human` | Human approval / clarification gate | orchestrator | После explicit human resolution |
| `gate:ci` | Ожидание CI статуса | PR/CI integration | После green/red terminal CI outcome |
| `gate:pr` | Ожидание PR state / mergeability | PR integration | После merge/close/supersede |
| `gate:timer` | Time wait / cooldown / scheduled retry | orchestrator | После наступления времени или ручной отмены |

### Правила

1. У parent feature обычно `issue_type=feature`.
2. Подзадачи pipeline могут быть отдельными beads с `parent-child` зависимостями.
3. Gate'ы — это не просто флаг в metadata, а отдельные operational beads, если у них есть собственный lifecycle и причина существовать в graph.
4. Если gate элементарный и не требует отдельного узла, допустим metadata-only gate index, но канонический предпочтительный вариант для durable coordination — **отдельный gate issue**.

---

## 3. Gate taxonomy

## 3.1 Gate types

| Gate type | `issue_type` | Назначение | Типичный trigger |
|---|---|---|---|
| Human gate | `gate:human` | Ждём решения/ответа человека | approve/reject/clarify |
| CI gate | `gate:ci` | Ждём completion внешнего CI run | PR checks / pipeline run |
| PR gate | `gate:pr` | Ждём mergeability, review state, PR existence | draft/open/merge/close |
| Timer gate | `gate:timer` | Ждём времени/cooldown/retry window | retry after / defer until |

## 3.2 Lifecycle gate bead

| Этап | Что происходит |
|---|---|
| Create | parent bead создаёт gate bead и связывает `blocks` или `parent-child` |
| Wait | gate bead находится в `open` или `blocked` |
| Observe | orchestrator/integration обновляет metadata и/или status |
| Satisfy | наступает условие закрытия |
| Close | gate bead закрывается, блокировка снимается |
| Escalate | при timeout или failure создаётся follow-up / escalation bead |

## 3.3 Canonical gate metadata

```json
{
  "schema": "sdp/beads-metadata/v1",
  "sdp": {
    "parent_bead_id": "sdp-a1b2c3",
    "phase": "review",
    "gate": {
      "kind": "ci",
      "state": "waiting",
      "created_at": "2026-03-24T09:00:00Z",
      "await": {
        "type": "gh:run",
        "id": "123456789"
      },
      "timeout_at": "2026-03-24T10:00:00Z",
      "auto_close_condition": "ci_terminal_success",
      "failure_condition": "ci_terminal_failure"
    }
  }
}
```

## 3.4 Auto-close conditions

| Gate | Auto-close condition | Failure / alternative close |
|---|---|---|
| `gate:human` | человек дал явный approve/answer | reject/cancel/supersede/manual close |
| `gate:ci` | внешний CI перешёл в terminal success | terminal failure → gate closes as failed and parent becomes blocked/rework |
| `gate:pr` | PR merged, либо state стал acceptable для следующего шага | PR closed/unmergeable/superseded |
| `gate:timer` | наступило `due_at` / `await_id` time reached | manual cancel, superseded by new retry plan |

## 3.5 Когда нужен отдельный gate issue, а когда хватит metadata

| Сценарий | Подход |
|---|---|
| Нужен durable blocker с наблюдаемым lifecycle | отдельный gate issue |
| Нужен timeout/escalation/history | отдельный gate issue |
| Нужно показать причину blocked на board | отдельный gate issue |
| Это просто derived summary already backed by another gate issue | metadata index на parent |

---

## 4. Dependency types для SDP

SDP не должен изобретать собственные blocking semantics мимо Beads deps. Используются стандартные dependency types.

| Dependency type | Для чего используется в SDP |
|---|---|
| `blocks` | Жёсткая блокировка следующего шага/родителя до завершения prerequisite |
| `parent-child` | Иерархия feature → clarify/review/qa/release/gate beads |
| `waits-for` | Ожидание fanout-набора дочерних шагов или gates |
| `related` | Семантическая связь без блокировки |
| `discovered-from` | Артефакт/подзадача появилась из parent investigation/review/qa |
| `tracks` | Non-blocking reference на внешний convoy/epic/project-level tracking |
| `validates` | Review/QA/approval bead подтверждает parent work item |
| `caused-by` | Инцидент, rollback, follow-up вызван конкретным bead/result |
| `supersedes` | Новый contract/review/release bead заменяет старый |
| `delegated-from` | Отражает delegation chain при передаче работы агенту/подпроцессу |

### Recommended usage patterns

#### Feature → clarify
- `parent-child`
- при необходимости ещё `blocks`, если без clarify нельзя идти дальше

#### Feature → contract
- `parent-child`
- `blocks`, если contract не готов

#### Feature → review / qa / release
- `parent-child`
- `validates` для результата проверки

#### Parent → gate
- `parent-child`
- `blocks`, если gate реально держит parent

#### Follow-up after failure
- `caused-by`
- иногда `supersedes`, если новый bead заменяет старую попытку

---

## 5. Status mapping: Beads statuses → SDP phases

`Issue.Status` отвечает за operational condition. `sdp.phase` отвечает за смысл pipeline-этапа.

## 5.1 Base mapping

| Beads status | SDP interpretation | Допустимые фазы |
|---|---|---|
| `open` | work item существует и может быть ready/awaiting depending on deps/gates | `intake`, `clarify`, `ready`, `contract` |
| `in_progress` | активное исполнение или активная проверка | `build`, `review`, `qa`, `release` |
| `blocked` | дальнейшее движение остановлено внешним или внутренним blocker'ом | любая фаза кроме `done` |
| `deferred` | сознательно отложено | `deferred` |
| `closed` | operationally complete / terminated | `done`, иногда terminal `release` |
| `pinned` | persistent reference bead, обычно не используется как feature lifecycle | служебно |
| `hooked` | work attached to agent hook; не основной feature lifecycle state | служебно |

## 5.2 Recommended phase transitions

| Phase | Typical `Issue.Status` | Exit condition |
|---|---|---|
| `intake` | `open` | intent captured and normalized |
| `clarify` | `open` / `blocked` | open questions resolved |
| `ready` | `open` | contractable/executable and no blocking gates |
| `contract` | `open` / `in_progress` | contract artifact produced and accepted |
| `build` | `in_progress` | implementation done, evidence captured |
| `review` | `in_progress` / `blocked` | review approved or changes requested |
| `qa` | `in_progress` / `blocked` | QA pass/fail resolved |
| `release` | `in_progress` / `blocked` | release completed or rolled back |
| `done` | `closed` | terminal completion |
| `deferred` | `deferred` | resumed manually or by timer |

## 5.3 Rules

1. Не кодировать pipeline phase через кастомные statuses, если хватает `sdp.phase`.
2. `blocked` — cross-cutting operational condition, а не отдельная pipeline phase.
3. `closed` означает, что work item завершён как operational entity; semantic summary остаётся в artifacts/projections.
4. Gate beads обычно закрываются раньше parent bead и снимают blockers.

---

## 6. Minimal examples

### 6.1 Feature bead in ready state

```json
{
  "title": "Create Beads-first ADR",
  "status": "open",
  "issue_type": "feature",
  "metadata": {
    "schema": "sdp/beads-metadata/v1",
    "sdp": {
      "card_id": "feature-sdp_lab-2026-03-24-002",
      "project_id": "sdp_lab",
      "phase": "ready",
      "contract": {
        "id": "CTR-002",
        "ref": ".sdp/contracts/CTR-002.json",
        "hash": "sha256:aaa"
      },
      "gates": {
        "open": [],
        "satisfied": ["sdp-gate-h01"],
        "failed": []
      }
    }
  }
}
```

### 6.2 CI gate bead

```json
{
  "title": "Wait for PR CI",
  "status": "open",
  "issue_type": "gate:ci",
  "await_type": "gh:run",
  "await_id": "987654321",
  "metadata": {
    "schema": "sdp/beads-metadata/v1",
    "sdp": {
      "parent_bead_id": "sdp-a1b2c3",
      "phase": "review",
      "gate": {
        "kind": "ci",
        "state": "waiting",
        "auto_close_condition": "ci_terminal_success"
      }
    }
  }
}
```

### 6.3 Release bead after deploy

```json
{
  "title": "Release Control Tower cutover",
  "status": "closed",
  "issue_type": "release",
  "metadata": {
    "schema": "sdp/beads-metadata/v1",
    "sdp": {
      "parent_bead_id": "sdp-a1b2c3",
      "phase": "done",
      "delivery": {
        "state": "deployed",
        "target": "staging",
        "ref": ".sdp/releases/REL-001.json",
        "delivered_at": "2026-03-24T11:00:00Z"
      }
    }
  }
}
```

---

## 7. Operational invariants

1. **Beads is the source of truth for operational lifecycle.**
2. **`FeatureCard` is not allowed to outvote `Issue.Status` or deps/gates.**
3. **Heavy documents stay in artifacts; metadata stores references and compact operational state.**
4. **Gates that matter operationally should be explicit beads, not invisible booleans.**
5. **Board/snapshot/Control Tower are projections over Beads + artifacts, not an alternative state machine.**
