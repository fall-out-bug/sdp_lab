# MicroFirst Inference Tier — Design

> **Status:** Design (2026-04-27) · **Owner:** Andrei · **Target feature:** F147
>
> **Numbering note:** F100–F146 заняты (F146 = inference decomposition framework, активный эпик `sdplab-vrnw`). F147 — следующий свободный.
>
> **Scope:** `sdp_lab` — generic Stage-композитор + 3 micro-classifier'а + applicative integration в один real consumer (ws-verdict, F146-04). Не публикуется отдельно.
>
> **Parent context:** F144 confidence-checker ([internal/inference/confidence/](../../internal/inference/confidence/)) — оценка уверенности **после** инференса. F145 cascade — выбор tier'а LLM. F146 decomposition — разбиение одной LLM-задачи на стадии. F147 закрывает orthogonal gap: **«вообще не вызывать LLM, если micro-уровень даёт уверенный ответ»**.
>
> **Trigger:** челлендж День 10 «Micro-model first». Дух челленджа сохраняется (двухуровневый инференс, замеры fallback rate / latency / token saving), но реализация подгоняется под реальные точки SDP, где LLM-вызов сейчас обязателен и многие случаи тривиальны.

## 1. Why now

1. **F146 framework приземляется (F146-01 closed, F146-02 in-progress).** Появляется готовый `Stage[In, Out]` interface — micro-first можно выразить как Stage-композитор, без нового Pipeline-типа и без ломки текущего API.
2. **ws-verdict (F146-04) — критичный consumer.** Каждый завершённый workstream проходит через verdict. Эмпирически (по последним 30 закрытым beads issue'ам в sdplab) **большинство случаев тривиальны**: тесты зелёные + guard-paths без diff → `PASS`; тесты красные → `FAIL`. На LLM их гонять — чистая трата токенов и latency. F147 даёт штатный механизм short-circuit'а перед вызовом модели.
3. **Сейчас нет канонического места для «pre-LLM gate».** F144 работает после инференса (валидирует ответ), F145 выбирает дешёвую модель, но всё равно зовёт LLM. Промежуточный уровень «никакая модель не нужна» отсутствует. Каждый call-site решал бы это ad-hoc.
4. **Размеченные данные уже лежат.** `.beads/issues.jsonl` — корпус labeled bd issue'ов с type / priority. `cmd/sdp-ft-dataset` уже умеет его собирать (F133). Это бесплатный train/test set для embedding-classifier'ов без отдельной разметки.
5. **Ollama — стандартный runtime в SDP.** F145 уже инжектит Ollama как low-tier provider. Embedding-модели (`bge-small-en-v1.5`, `nomic-embed-text`) живут в той же runtime → один runtime, нулевая дополнительная зависимость.

## 2. Goals / Non-goals

**Goals:**
- Generic Stage-композитор `decompose.WithEscalation[In, Out](micro Stage[In, Out], llm Stage[In, Out], cfg EscalationConfig) Stage[In, Out]` — оборачивает (micro, llm) в обычный `Stage`, видимый Pipeline'у как один шаг.
- Реализовать **3 разных micro-classifier'а** для доказательства, что Stage-композитор реально generic (не one-shot wrapper):
  - **`WsVerdictMicro`** — детерминистский: парсит test report + guard diff → `{PASS, FAIL, UNSURE}`. Без ML, чистые правила. Применим как pre-stage в F146-04 ws-verdict pipeline.
  - **`BdSeverityMicro`** — embedding-based на title+description → `{P0, P1, P2, P3, UNSURE}` + score. Корпус: размеченные `.beads/issues.jsonl` (>50 закрытых issue). Backbone: `bge-small-en-v1.5` через Ollama.
  - **`BdTypeMicro`** — embedding-based на title+description → `{bug, task, feature, UNSURE}` + score. Тот же corpus, тот же backbone.
- Замеры по требованию челленджа: ≥30 запросов на каждый classifier (простые + edge + adversarial); fallback rate, latency p50/p95, accuracy против ground truth, total LLM call savings.
- Интеграция как opt-in в **один real consumer** (ws-verdict, F146-04) — proof of value без широкого blast radius.

**Non-goals:**
- Не делаем **новый Pipeline-тип**. `WithEscalation` — это `Stage[In, Out]`, существующий Pipeline ничего не знает про escalation. Семантика короткого замыкания инкапсулирована внутри Stage.
- Не трогаем F144 confidence API. F147 confidence — про "довериться micro vs escalate", это отдельная семантика. Возможна общая `Status`-shared (она уже в `decompose.Status`), но интерфейсы свои.
- Не реализуем **cross-classifier ensemble** (несколько micro моделей голосуют). Один micro per Stage — расширение в follow-up.
- Не делаем UI / dashboard. Метрики — JSON evidence + markdown report (как в F144).
- Не делаем **online learning** / fine-tune embedding'а. Embeddings берутся из off-the-shelf модели; classifier — k-NN / cosine threshold.
- Не делаем shadow-mode перед включением default=on в ws-verdict — после F147-09 evidence сразу default=on, kill-switch остаётся (`--no-microfirst`).
- ~~Не покрываем dispatch routing cold-start~~ — **включено в F147** как F147-06 (`RoutingColdStartMicro`).
- ~~Не пишем CLI `bd suggest`~~ — **включено в F147** как F147-08.

## 3. Approach per workstream

### F147-01 · Core composer `decompose.WithEscalation`

**Проблема:** Текущий `Pipeline` линейный — каждая стадия передаёт `Out` следующей. Нет встроенного «short-circuit, если результат уверенный». Надо выразить escalation так, чтобы Pipeline ничего не знал.

**Решение:** Stage-композитор. Внутри одной Stage сидит логика «попробовать micro → проверить confidence → escalate в llm если нужно».

```go
package decompose

// EscalationConfig controls when to escalate from micro to llm stage.
type EscalationConfig struct {
    // ConfidenceThreshold: micro Out is trusted iff its reported score >= threshold.
    // Score is read via the Confider interface below; if Out doesn't implement
    // Confider, the micro is treated as Unsure (always escalates).
    ConfidenceThreshold float64

    // EscalateOnError: if true, micro errors trigger llm fallback. If false,
    // micro errors propagate as Stage error (llm not invoked).
    EscalateOnError bool

    // RecordSkippedTrace: if true, llm StageTrace is recorded as zero-cost
    // skipped trace inside the composed Stage trace (for accurate "saved" metrics).
    RecordSkippedTrace bool
}

// Confider is implemented by types that report self-confidence.
// MicroClassifier outputs typically implement this; LLM Stage outputs can too.
type Confider interface {
    Confidence() float64       // [0, 1]
    Status() Status            // StatusOK / StatusUnsure / StatusFail
}

// WithEscalation composes (micro, llm) into a single Stage[In, Out].
// Run logic:
//   1. Run micro. If err and !EscalateOnError → propagate err.
//   2. If micro Out is Confider and Status == OK and Confidence >= threshold
//      → return micro Out + micro trace (escalated=false).
//   3. Else → run llm with the same In. Return llm Out + combined trace
//      (micro trace + llm trace; escalated=true).
//
// The combined StageTrace.Attempts encodes which path: 1 = micro-only,
// 2 = micro + llm escalation.
func WithEscalation[In, Out any](
    micro Stage[In, Out],
    llm Stage[In, Out],
    cfg EscalationConfig,
) Stage[In, Out]
```

**Trace contract:**
- Micro-only success: `Trace.Attempts=1`, `Trace.LatencyMs = micro.LatencyMs`, `Trace.TokensIn/Out = 0` (assuming embedding-only or pure-Go micro).
- Escalation: `Trace.Attempts=2`, `Trace.LatencyMs = micro + llm`, `Trace.TokensIn/Out = llm.TokensIn/Out`. Micro's zero-token contribution makes "saved tokens vs llm-only" computable as `(escalations * llm_avg_tokens)` — counterfactual baseline.

**Why a Stage-composer (γ), not a new Pipeline-type (α) or extended StageConfig (β):**
- (α) New `EscalationPipeline[Final]` duplicates Pipeline machinery and forces caller to choose between two Pipeline types per use case.
- (β) Extending `StageConfig.OnFailure` with `ShortCircuit` couples short-circuit semantics to the linear Pipeline. Out type of stage[i] must equal Final — fragile across heterogeneous pipelines (e.g., F146-04: extract→classify→aggregate, where extract Out ≠ Final).
- (γ) Stage-composer is opaque to Pipeline; per-stage independently. Composes naturally with `confidence.Wrap` (F144) and cascade (F145) — those are also Stage-level.

**Acceptance:**
- `WithEscalation` unit-tested: micro-OK → no llm call, micro-Unsure → llm called, micro-error + EscalateOnError=true → llm called, micro-error + EscalateOnError=false → error.
- Trace correctness: combined latency = sum, token counts only from llm path.
- ≥85% line coverage in package.

### F147-02 · `WsVerdictMicro` (rule-based, no ML)

**Проблема:** ws-verdict (F146-04) сейчас зовёт LLM на каждый workstream. Многие случаи тривиальны — тесты прошли + guard чистый = PASS, тесты упали = FAIL. Это можно решить парсером.

**Input:** `WsVerdictInput{TestReport, GuardDiff, WsName}` (типы уже определены в F146-04 spec).

**Output:** `WsVerdictMicroResult` с полями `{Verdict, Confidence, Status, Reasons}`. Реализует `Confider`.

**Правила (deterministic):**
1. `TestReport.Failed > 0` → `FAIL`, confidence=0.95.
2. `TestReport.Failed == 0 && TestReport.Errored == 0 && len(GuardDiff.OutOfScope) == 0` → `PASS`, confidence=0.90.
3. `len(GuardDiff.OutOfScope) > 0` → `UNSURE`, confidence=0.40 (escalate, потому что guard violation требует семантической оценки).
4. `TestReport.Skipped > threshold` → `UNSURE`, confidence=0.50 (skipped tests могут скрывать реальную проблему).
5. `TestReport.Coverage < ws.MinCoverage` → `UNSURE`, confidence=0.45 (numeric, но политика семантическая).
6. Иначе → `UNSURE`, confidence=0.30.

**Threshold для escalation:** `ConfidenceThreshold=0.85`. Только rules 1–2 проходят без escalation.

**Acceptance:**
- 30+ test cases (golden inputs из реальных SDP test reports — собираются из `internal/build/.sdp/evidence/` и архива).
- Expected fallback rate ≥40% (по типичной популяции workstream'ов; точное число определяется на этапе бенча).
- Accuracy на cases с явным PASS/FAIL = 100% (deterministic).
- Latency p99 < 5ms (regex/parse-only).

### F147-03 · `BdSeverityMicro` (embedding-based)

**Проблема:** `@issue` skill сейчас классифицирует severity через большую LLM. P0 vs P1 vs P2 — это semantic decision, но 80% случаев определяются ключевыми сигналами в title/description ("production down", "data loss" → P0; "typo", "stale doc" → P3).

**Backbone:** `bge-small-en-v1.5` или `nomic-embed-text` через Ollama (`POST /api/embeddings`). Оба ~30M параметров, локально, бесплатно.

**Алгоритм (k-NN over labeled corpus):**
1. Загрузить размеченный корпус из `.beads/issues.jsonl` (только closed, имеют `priority` поле). Параметризуется через `cmd/sdp-microfirst-train` (separate small command).
2. Для каждого labeled issue посчитать embedding(title + "\n" + description).
3. На запрос: посчитать embedding входа, найти top-k=5 ближайших по cosine.
4. Если top-1 cosine ≥ 0.85 **и** top-3 голосуют за один label → `OK`, confidence = top-1 cosine.
5. Если top-3 разнобой или top-1 < 0.85 → `UNSURE`, confidence = top-1 cosine.

**Output:** `BdSeverityMicroResult{Priority, Confidence, Status, Neighbors}` (Neighbors — для explainability в trace).

**Threshold для escalation:** `ConfidenceThreshold=0.85`.

**Train/eval split:**
- Train: все closed bd issues по состоянию на ветку, **исключая** последние 30 закрытых.
- Eval: 30 последних закрытых (hold-out по времени, чтобы избежать leakage).
- Metric: accuracy(Status=OK), fallback rate, escalation accuracy (когда escalated to LLM, насколько ответ совпадает с ground truth).

**Acceptance:**
- Eval accuracy on `Status=OK` subset ≥ 80% (если меньше — threshold ужесточается, fallback rate растёт).
- Fallback rate ≤ 50% (баланс — слишком высокий означает micro бесполезен).
- Latency p95 < 100ms (embedding call to Ollama + cosine over ~50 vectors).

### F147-04 · `BdTypeMicro` (embedding-based)

Тот же дизайн, что F147-03, но классы — `{bug, task, feature, UNSURE}`. Корпус — те же `.beads/issues.jsonl` с полем `type`.

**Уточнение:** `chore` маппится в `task`, `epic` исключается из train (сильно отличается по тексту).

**Acceptance:** идентичный F147-03, но threshold=0.80 (3-class задача легче 4-class).

### F147-06 · `RoutingColdStartMicro` (embedding-based)

**Проблема:** В `internal/dispatch/route.go:197` для unknown task types отдаётся neutral prior — никакой capability hint. Это блокирует разумный cold-start, особенно когда новый harness/agent ещё не имеет истории.

**Решение:** Embedding-classifier поверх существующих task-type → capability mappings. Корпус: `internal/dispatch/profiles_default.json` (F145-09) + закрытые beads issues с указанной capability.

**Алгоритм:** идентичен `BdSeverityMicro`/`BdTypeMicro` (k-NN over labeled corpus, cosine similarity, threshold 0.80).

**Output:** `RoutingColdStartResult{CapabilityHint, Confidence, Status, Neighbors}`. Реализует `Confider`.

**Интеграция:** `internal/dispatch/route.go` — заменяет neutral prior в `Router.Route()` (line 197 на main). Если `Status=OK` → используем capability hint в scoring; `UNSURE` → fallback к старому neutral prior (no regression).

**Acceptance:**
- Eval accuracy ≥ 75% (cold-start задача сложнее bd-classifier'ов).
- Fallback rate ≤ 60%.
- Latency p95 < 100ms.
- Backward-compat: при `Status=UNSURE` поведение Router идентично pre-F147.

### F147-07 · ws-verdict micro-first integration

**Проблема:** F147-01..04 — библиотечные. Без real consumer не доказан value.

**Решение:** В F146-04 ws-verdict pipeline'е первая стадия (`extract`) оборачивается в `WithEscalation(WsVerdictMicro, originalExtractStage, cfg)`.

**Code shape:**
```go
extractMicro := microfirst.NewWsVerdictMicro(rules.Default())
extractLLM   := llm.NewExtractStage(haiku, ...) // существующий из F146-04
extractStage := decompose.WithEscalation(extractMicro, extractLLM, decompose.EscalationConfig{
    ConfidenceThreshold: 0.85,
    EscalateOnError:     true,
    RecordSkippedTrace:  true,
})

p := decompose.New[FinalVerdict]("ws-verdict")
p = decompose.Then(p, extractStage, ...)
p = decompose.Then(p, classifyStage, ...)
p = decompose.Then(p, aggregateStage, ...)
```

**Default behaviour:** default=on в `cmd/sdp-ws-verdict-validate`. Kill-switch — `--no-microfirst` (отключает обёртку, восстанавливает поведение F146-04 байт-в-байт).

**Зависимость:** `BLOCKED-BY F146-04` (нужна готовая 3-stage pipeline). Если F146-04 ещё не вышел из in-progress, F147-07 встаёт в `blocked`.

**Acceptance:**
- E2E test: для тривиальных fixture'ов (clear-pass, clear-fail) extract stage не зовёт LLM (`StageTrace.TokensIn == 0`); для ambiguous — зовёт.
- `--no-microfirst` восстанавливает поведение F146-04 байт-в-байт (regression test).

### F147-08 · `bd suggest` CLI

**Проблема:** `BdSeverityMicro` и `BdTypeMicro` — библиотечные. Без CLI-обёртки они невидимы для пользователя.

**Решение:** Новая команда `bd suggest <title> [--description=<text>]` (плагин beads, реализуется как `cmd/sdp-bd-suggest/main.go` + интеграция через [.agents/skills/beads](../../.agents/skills/beads/) wrapper при необходимости).

**Output (JSON по умолчанию, human-readable при `--format=human`):**
```json
{
  "title": "...",
  "type":     {"value": "bug",  "confidence": 0.91, "status": "ok"},
  "priority": {"value": "P2",   "confidence": 0.62, "status": "unsure", "escalated": true, "fallback": "P2"},
  "neighbors": [{"id": "sdplab-...", "title": "...", "score": 0.93}, ...]
}
```

`escalated=true` означает, что при `--escalate` команда вызвала большую LLM (Anthropic). Без `--escalate` — только micro, `unsure` остаётся `unsure`.

**Не делаем (важно):** `bd suggest` не пишет в `.beads/issues.jsonl` — это **read-only suggester**. Применение — через копирование значений в `bd create`.

**Acceptance:**
- Команда работает на 5+ ручных тестовых тайтлах.
- `--format=json` возвращает stable schema; `--format=human` печатает human-readable summary.
- Документировано в `docs/reference/microfirst-tier.md` + `.agents/skills/beads/` README upd.

### F147-09 · Bench harness + report

**Проблема:** Челлендж требует замеров. Без harness'а — нет evidence.

**Решение:** `cmd/sdp-microfirst-bench` — отдельный CLI:
- Загружает корпус (`.beads/issues.jsonl` для bd-classifier'ов, fixture'ы из `internal/build/.sdp/fixtures/ws-verdict/` для ws-verdict, `internal/dispatch/profiles_default.json` + closed bd с capability для routing).
- Прогоняет каждый classifier в двух режимах: micro-only и WithEscalation(micro, llm).
- Эмулирует "llm" как mock с фиксированной latency 800ms + token cost (для бенча; реальные числа считаются при integration test против live LLM, но это вне F147-09 scope).
- Output: JSON в `internal/build/.sdp/evidence/f147/{classifier}.json` + markdown report `f147-bench-report.md` со сравнением: `total_requests, micro_handled, escalated, fallback_rate, p50/p95_latency, accuracy, llm_calls_saved, est_token_savings`.

**Acceptance:**
- Report содержит данные для всех **четырёх** classifier'ов (WsVerdict, BdSeverity, BdType, RoutingColdStart).
- Aggregate `est_token_savings` ≥ 30% (если меньше — переоценить thresholds).
- Aggregate p50 latency через micro-first меньше, чем llm-only baseline (для real cases, не worst-case).

## 4. Workstream DAG

```
F147-01 (composer + Confider)
   │
   ├─→ F147-02 (WsVerdictMicro · deterministic)
   │       │
   │       └─────────────────────────────┐
   │                                     │
   ├─→ F147-03 (Ollama embedding client + k-NN lib · shared infra)
   │       │
   │       ├─→ F147-04 (BdSeverityMicro)
   │       ├─→ F147-05 (BdTypeMicro)
   │       └─→ F147-06 (RoutingColdStartMicro · dispatch integration)
   │                                     │
   │                                     ├─→ F147-08 (bd suggest CLI · uses 04+05)
   │                                     │
   │                                     └─→ F147-09 (bench harness + report)
   │
   └─→ F147-07 (ws-verdict integration · uses 02; BLOCKED-BY F146-04)
```

**Dependency на F146:**
- F147-01 depends on F146-01 (closed) — `Stage[In, Out]`, `StageTrace`, `Status`. Готово.
- F147-07 blocked by F146-04 (open) — нужна готовая 3-stage pipeline. Остальные F147-* не блокируются.

**Параллелизуемое:** F147-02 и F147-03 независимы после F147-01. F147-04, F147-05, F147-06 параллелизуются после F147-03.

## 5. Rollout / Risks

**Rollout:**
1. F147-01 (composer) → F147-02 (WsVerdictMicro), F147-03 (Ollama+k-NN) — параллельно.
2. F147-04..06 — параллельно после F147-03.
3. F147-07 (ws-verdict integration) — стартует когда F146-04 closed; default=on c kill-switch `--no-microfirst`.
4. F147-08 (bd suggest CLI) — после F147-04 + F147-05.
5. F147-09 (bench + report) — финальный. Подтверждает evidence для exit criteria.

**Risks:**
- **R1: Embedding-based micro деградирует при corpus drift.** Размеченные bd issues могут устареть (старые соглашения по priority). Митигация: re-train periodic (`cmd/sdp-microfirst-train`), eval на hold-out тесте, fail CI если accuracy < threshold.
- **R2: Ollama unavailable → bd classifier'ы падают.** Митигация: `EscalateOnError=true` → fallback к LLM. F147-01 поведение покрывает кейс.
- **R3: WsVerdictMicro упускает edge case (false PASS).** Опасно — пропустит сломанный workstream. Митигация: правила консервативные (UNSURE по умолчанию), eval на 30+ adversarial fixtures (умышленно тонкие фейлы), opt-in перед массовым включением.
- **R4: Composer increases trace complexity.** Митигация: trace-tests в F147-01 покрывают combined trace shape; F146 telemetry не ломается (одна Stage из вне).
- **R5: Threshold tuning subjective.** Митигация: F147-06 бенч сравнивает три threshold'а (0.75/0.85/0.95) и фиксирует выбор evidence-based.

## 6. Success criteria (F147 epic exit)

1. `decompose.WithEscalation` merged + ≥85% coverage.
2. Все 4 micro-classifier'а (WsVerdict, BdSeverity, BdType, RoutingColdStart) merged + unit-tested.
3. F147-07 integration: ws-verdict default=on, kill-switch `--no-microfirst` работает.
4. F147-08 `bd suggest` CLI работает (`--format=json` + `--format=human`).
5. F147-09 bench report: aggregate `llm_calls_saved` ≥ 30%, no accuracy regression vs llm-only на hold-out eval.
6. Documentation: `docs/reference/microfirst-tier.md` (how-to + canonical example для добавления новой micro Stage).
7. Evidence в `internal/build/.sdp/evidence/f147/` коммитнут.

## 7. Resolved decisions

- **R1 (was OQ1):** Default=on для ws-verdict — **сразу после F147-09 evidence**, без shadow-mode. Kill-switch `--no-microfirst` остаётся.
- **R2 (was OQ2):** `bd suggest` CLI — **в F147** (F147-08), не follow-up.
- **R3 (was OQ3):** Routing cold-start — **в F147** (F147-06), не отдельный F148.

## 8. Челлендж compliance (Day 10)

- ✅ Двухуровневый инференс: micro → escalation в LLM.
- ✅ Уровень 1: маленькая модель (embedding `bge-small`), либо ML-классификатор (k-NN), либо детерминистский (WsVerdictMicro). Все три варианта в одном эпике.
- ✅ Возвращает structured result + confidence score + status (OK/UNSURE) — через `Confider` interface.
- ✅ Уровень 2: LLM fallback при UNSURE/low-confidence/error.
- ✅ ≥30 запросов на classifier (F147-06 acceptance).
- ✅ Метрики: micro_handled, escalated, llm_calls, p50/p95 latency.
- 🆕 Bonus: реальная интеграция в production-path (ws-verdict) — не just-PoC.
