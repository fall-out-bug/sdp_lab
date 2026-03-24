# FeatureCard ↔ Beads mapping

Status: canon draft  
Date: 2026-03-24  
Scope: canonical mapping from `internal/control.FeatureCard` to `beads/types.Issue` + `Issue.Metadata.sdp` + external artifacts

## 1. Принцип mapping

После Beads-first pivot:

- **Bead issue** = canonical operational work item
- **FeatureCard** = semantic projection / intake-spec artifact
- **External artifacts** = большие или специализированные документы (`contract.json`, intake markdown, review report, evidence bundle)

### Нормы

1. Если поле имеет естественный typed field в `Issue` — использовать его.
2. Если поле SDP-специфично, но operationally важно — хранить в `Issue.Metadata.sdp.*`.
3. Если поле является документом, отчётом или blob-артефактом — хранить как внешний artifact, а в metadata держать ref/path/hash.
4. `LinkedBeadsIDs` удаляется: **один FeatureCard соответствует одному canonical Beads issue**.

---

## 2. Canonical envelope

### Beads Issue

```json
{
  "id": "sdp-ab12cd",
  "title": "Pivot Control Tower to beads-first source of truth",
  "description": "Raw request + normalized intent summary",
  "status": "open",
  "issue_type": "feature",
  "priority": 1,
  "owner": "human@local",
  "assignee": "sdp-orchestrator",
  "metadata": {
    "schema": "sdp/beads-metadata/v1",
    "sdp": {
      "card_id": "feature-sdp_lab-2026-03-24-001",
      "phase": "clarify",
      "task_type": "architecture",
      "execution_mode": "repo",
      "risk_level": "high",
      "scope": {
        "in": ["docs/", "internal/control"],
        "out": ["beads storage engine rewrite"]
      },
      "contract": {
        "ref": ".sdp/contracts/CTR-001.json",
        "id": "CTR-001",
        "hash": "sha256:abc"
      }
    }
  }
}
```

### FeatureCard после pivot

FeatureCard может сохраняться как удобная проекция, но не должен вводить собственное lifecycle truth. Минимально он должен содержать:
- card id
- project id
- canonical bead id
- semantic summary
- artifact refs
- derived state snapshot для UI/CLI

---

## 3. Полный mapping полей FeatureCard

| FeatureCard field | Куда маппится | Canonical target | Примечание |
|---|---|---|---|
| `ID` | `metadata.sdp.card_id` | Metadata | Card ID сохраняется как semantic reference |
| `ProjectID` | `metadata.sdp.project_id` | Metadata | При необходимости также в label/prefix discipline |
| `Title` | `Issue.Title` | Typed | Основной title bead issue |
| `Status` | `Issue.Status` + `metadata.sdp.phase` | Typed + Metadata | Card status как самостоятельный store убирается |
| `RawRequest` | `Issue.Description` + intake artifact | Typed + Artifact | Полный intake лучше хранить в `intake.md`; в description — summary или raw excerpt |
| `CreatedAt` | `Issue.CreatedAt` | Typed | Каноническое время создания bead |
| `UpdatedAt` | `Issue.UpdatedAt` | Typed | Каноническое время последнего обновления |
| `SourceRefs` | `metadata.sdp.source_refs[]` | Metadata | Ссылки на контекст/источники |
| `LastOrchestratorAction` | `metadata.sdp.orchestration.last_action` | Metadata | Derived operational trace |
| `LastOrchestratorReason` | `metadata.sdp.orchestration.last_reason` | Metadata | Почему был сделан последний шаг |
| `LastOrchestratorAt` | `metadata.sdp.orchestration.last_at` | Metadata | Timestamp |
| `RecommendedNextAction` | `metadata.sdp.next.action` | Metadata | Подсказка orchestration/view layer |
| `RecommendedNextReason` | `metadata.sdp.next.reason` | Metadata | Объяснение |
| `ClarificationCycles` | `metadata.sdp.counters.clarification_cycles` | Metadata | Counter |
| `BlockedCycles` | `metadata.sdp.counters.blocked_cycles` | Metadata | Counter |
| `ExecutionAttemptCount` | `metadata.sdp.counters.execution_attempt_count` | Metadata | Counter |
| `ReviewFailCount` | `metadata.sdp.counters.review_fail_count` | Metadata | Counter |
| `RollbackCount` | `metadata.sdp.counters.rollback_count` | Metadata | Counter |
| `ReviewState` | `metadata.sdp.review.state` | Metadata | review pending/approved/changes_requested |
| `ReviewSummary` | `metadata.sdp.review.summary` | Metadata | Краткий summary; полный отчёт — artifact |
| `ReviewRef` | `metadata.sdp.review.ref` | Metadata | Path/URI на review report |
| `DeliveryState` | `metadata.sdp.delivery.state` | Metadata | pending/deployed/rolled_back/... |
| `DeliveryTarget` | `metadata.sdp.delivery.target` | Metadata | staging/prod/local |
| `DeliverySummary` | `metadata.sdp.delivery.summary` | Metadata | Краткий итог |
| `DeliveryRef` | `metadata.sdp.delivery.ref` | Metadata | Ссылка на deploy/report artifact |
| `DeliveredAt` | `metadata.sdp.delivery.delivered_at` | Metadata | Timestamp |
| `RollbackRef` | `metadata.sdp.delivery.rollback_ref` | Metadata | Ref на rollback evidence/artifact |
| `RollbackSummary` | `metadata.sdp.delivery.rollback_summary` | Metadata | Summary rollback |
| `FollowupRefs` | `metadata.sdp.followups[]` | Metadata | Ref на follow-up artifacts/issues |
| `NormalizedIntent` | `metadata.sdp.intent.normalized` | Metadata | Семантический intent |
| `TaskType` | `metadata.sdp.task_type` | Metadata | Например: feature/docs/architecture/fix |
| `ExecutionMode` | `metadata.sdp.execution_mode` | Metadata | repo/spec/docs/etc. |
| `TargetRepo` | `metadata.sdp.target.repo` | Metadata | Repo identifier/path |
| `TargetArea` | `metadata.sdp.target.area` | Metadata | Подсистема/директория |
| `ScopeIn` | `metadata.sdp.scope.in[]` | Metadata | In-scope |
| `ScopeOut` | `metadata.sdp.scope.out[]` | Metadata | Out-of-scope |
| `NonGoals` | `metadata.sdp.non_goals[]` | Metadata | Не цели |
| `RiskLevel` | `metadata.sdp.risk_level` | Metadata | low/medium/high |
| `WhyNow` | `metadata.sdp.why_now` | Metadata | Причина срочности/приоритета |
| `Links` | `metadata.sdp.links[]` | Metadata | Related URLs/docs |
| `OpenQuestions` | `metadata.sdp.clarification.open_questions[]` | Metadata | Пока не закрытые вопросы |
| `AcceptanceShape` | `Issue.AcceptanceCriteria` + `metadata.sdp.acceptance_shape[]` | Typed + Metadata | Human-readable список и/или compiled text |
| `RecommendedNext` | `metadata.sdp.next.step` | Metadata | Семантический next step |
| `IntakeArtifact` | `metadata.sdp.artifacts.intake[]` | Metadata | Фактический markdown хранится как file artifact |
| `LinkedBeadsIDs` | **удаляется** | — | bead = сам card; отдельная связь больше не нужна |
| `LinkedWorkstreams` | `metadata.sdp.workstreams[]` | Metadata | Ссылки на derived execution workstreams |
| `RequiredArtifacts` | `metadata.sdp.contract.required_artifacts[]` | Metadata | Канон лучше брать из `contract.json`; metadata — индекс |
| `RequiredChecks` | `metadata.sdp.contract.required_checks[]` | Metadata | Аналогично |
| `LinkedArtifacts` | `metadata.sdp.artifacts.linked[]` | Metadata | Список artifact refs |
| `ActiveAgents` | `metadata.sdp.executor.active_agents[]` | Metadata | Derived runtime state |
| `BlockingReasons` | `metadata.sdp.blocking.reasons[]` | Metadata | Только как explanation; реальная блокировка выражается deps/gates/status |
| `WaitingOn` | `metadata.sdp.blocking.waiting_on[]` | Metadata | Explanation index, не primary blocker model |
| `NeedsFeedbackFrom` | `metadata.sdp.blocking.needs_feedback_from[]` | Metadata | Обычно коррелирует с gate:human |
| `FeedbackRequest` | `metadata.sdp.blocking.feedback_request[]` | Metadata | Human/admin questions |
| `DecisionRequired` | `metadata.sdp.blocking.decision_required[]` | Metadata | Список требуемых решений |
| `AuthorUpdate` | `metadata.sdp.blocking.author_update[]` | Metadata | Что должен сделать author/human |
| `AdminActionRequired` | `metadata.sdp.blocking.admin_action_required[]` | Metadata | Например approval/deploy credential step |
| `DispatchedAt` | `metadata.sdp.executor.dispatched_at` | Metadata | Timestamp dispatch |
| `DispatchedTo` | `metadata.sdp.executor.dispatched_to` | Metadata | Role / runtime destination |
| `DispatchedPacketPath` | `metadata.sdp.executor.packet_ref` | Metadata | Ссылка на dispatch packet |
| `ExecutorSessionID` | `metadata.sdp.executor.session_id` | Metadata | Runtime session id |
| `ExecutorStartedAt` | `metadata.sdp.executor.started_at` | Metadata | Timestamp |
| `LastExecutorHeartbeatAt` | `metadata.sdp.executor.last_heartbeat_at` | Metadata | Timestamp |
| `ExecutorRuntimeState` | `metadata.sdp.executor.state` | Metadata | pending/running/stuck/completed |
| `ExecutorProgressSummary` | `metadata.sdp.executor.progress_summary` | Metadata | Summary |
| `ExecutorResult` | `metadata.sdp.executor.last_result.*` + artifact refs | Metadata + Artifact | Полный result packet — внешний artifact |
| `ConstitutionWarnings` | `metadata.sdp.policy.constitution_warnings[]` | Metadata | Policy evaluation outcome |

---

## 4. Что уходит во внешние artifacts

| Семантика | Artifact |
|---|---|
| полный raw intake | `intake.md` |
| executable spec | `contract.json` |
| dispatch payload | `dispatch-packet.json` |
| executor result | `result-packet.json` |
| review report | `review.md` / `review.json` |
| qa report | `qa.md` / `qa.json` |
| evidence bundle | `evidence/` + index |
| provenance chain | `provenance.json` |

Правило простое: **в metadata — ссылка, в artifact — тело документа**.

---

## 5. Lifecycle mapping

### 5.1 Card status → Beads status + SDP phase

`FeatureCard.status` больше не живёт как независимая FSM. Вместо этого используется пара:

- `Issue.Status` — operational state
- `metadata.sdp.phase` — pipeline phase

| Старый `FeatureCard.status` | `Issue.Status` | `metadata.sdp.phase` | Комментарий |
|---|---|---|---|
| `inbox` | `open` | `intake` | Карточка создана, ещё не нормализована |
| `clarifying` | `open` / `blocked` | `clarify` | `blocked`, если ждём человека или внешнее решение |
| `ready` | `open` | `ready` | Готово к dispatch, blockers закрыты |
| `executing` | `in_progress` | `build` / `implement` | Активный runtime |
| `reviewing` | `in_progress` / `blocked` | `review` | Может блокироваться gate:pr / gate:human |
| `blocked` | `blocked` | любой | Operationally blocked независимо от phase |
| `needs_input` | `blocked` | `clarify` / `review` / `release` | Обычно с gate:human |
| `done` | `closed` | `done` / `release` | Закрыто после завершения pipeline |
| `parked` | `deferred` | `deferred` | Осознанно отложено |

### 5.2 Gate model в lifecycle

Готовность к переходу определяется не только статусом, но и отсутствием незакрытых gate/issues и blocking deps.

| SDP lifecycle condition | Beads representation |
|---|---|
| нужно уточнение от человека | `Issue.Status=blocked` + child gate issue type `gate:human` |
| ждём CI | child gate `gate:ci` или dependency на соответствующий gate bead |
| ждём PR/merge | child gate `gate:pr` |
| ждём таймер/defer | `gate:timer` или `defer_until`/`await_type=timer` |
| можно исполнять | `Issue.Status=open`, `phase=ready`, нет blocking deps, нет open gates |

---

## 6. LinkedBeadsIDs: что меняется

### Было
`LinkedBeadsIDs` связывал один FeatureCard с одним или несколькими bead IDs.

### Стало
Это поле **убирается**.

### Причина
После pivot bead issue — и есть canonical operational representation карточки. Нельзя хранить ссылку на truth, если сам объект уже является truth.

### Замена
Если нужны дополнительные связи, использовать:
- сам `Issue.ID` как canonical ID work item'а
- dependencies (`parent-child`, `blocks`, `related`, `tracks`)
- `metadata.sdp.workstreams[]` и `metadata.sdp.followups[]` для non-blocking semantic refs

---

## 7. Примеры beads issues

### 7.1 Feature bead

```json
{
  "id": "sdp-a1b2c3",
  "title": "Beads-first pivot for Control Tower",
  "description": "Сделать Beads единственным operational source of truth для SDP.",
  "status": "open",
  "priority": 1,
  "issue_type": "feature",
  "owner": "andrey",
  "assignee": "sdp-orchestrator",
  "metadata": {
    "schema": "sdp/beads-metadata/v1",
    "sdp": {
      "card_id": "feature-sdp_lab-2026-03-24-001",
      "project_id": "sdp_lab",
      "phase": "clarify",
      "task_type": "architecture",
      "execution_mode": "repo",
      "risk_level": "high",
      "intent": {
        "normalized": "Перевести SDP Control Tower на beads-first truth model"
      },
      "scope": {
        "in": ["docs/", "internal/control"],
        "out": ["переписывание beads storage"]
      },
      "clarification": {
        "open_questions": [
          "Нужен ли dual-write период?",
          "Какие поля FeatureCard останутся в projection?"
        ]
      },
      "contract": {
        "id": "CTR-001",
        "ref": ".sdp/contracts/CTR-001.json",
        "hash": "sha256:111"
      },
      "artifacts": {
        "intake": [".sdp/control/projects/sdp_lab/intake/feature-sdp_lab-2026-03-24-001.md"],
        "linked": ["docs/BEADS_FIRST_CONTROL_TOWER_ROADMAP.md"]
      },
      "counters": {
        "clarification_cycles": 1,
        "execution_attempt_count": 0,
        "review_fail_count": 0,
        "rollback_count": 0
      }
    }
  }
}
```

### 7.2 Human gate bead

```json
{
  "id": "sdp-gate-h01",
  "title": "Approve Beads-first cutover semantics",
  "status": "open",
  "priority": 1,
  "issue_type": "gate:human",
  "await_type": "human",
  "assignee": "andrey",
  "metadata": {
    "schema": "sdp/beads-metadata/v1",
    "sdp": {
      "parent_bead_id": "sdp-a1b2c3",
      "phase": "clarify",
      "gate": {
        "kind": "human",
        "state": "waiting",
        "prompt": "Подтвердить, что control store больше не хранит lifecycle truth",
        "auto_close_condition": "explicit_human_approval"
      }
    }
  }
}
```

### 7.3 Review bead

```json
{
  "id": "sdp-r01",
  "title": "Review Beads-first ADR and schema docs",
  "status": "in_progress",
  "priority": 2,
  "issue_type": "review",
  "assignee": "review-agent",
  "metadata": {
    "schema": "sdp/beads-metadata/v1",
    "sdp": {
      "parent_bead_id": "sdp-a1b2c3",
      "phase": "review",
      "review": {
        "state": "pending",
        "attempts": 1,
        "ref": ".sdp/reviews/review-sdp-r01.md"
      },
      "executor": {
        "session_id": "sess-123",
        "state": "running",
        "started_at": "2026-03-24T08:30:00Z"
      }
    }
  }
}
```

---

## 8. Практические правила для кода

1. Любой новый lifecycle write path сначала спрашивает: это typed Issue field, metadata или artifact?
2. Нельзя добавлять в FeatureCard новое operational поле, если оно уже выражается через `Issue.Status`, deps, gates или `metadata.sdp`.
3. Board/snapshot/UI не имеют права изобретать state; только проецировать.
4. Если `FeatureCard` и Beads расходятся, читать нужно Beads.
5. Для migration допустим projection cache, но не новая параллельная truth model.

## 9. Bottom line

`FeatureCard` остаётся полезным как слой смысла.  
`Beads Issue` становится слоем operational reality.  
`Artifacts` держат тяжёлые документы и доказательства.
