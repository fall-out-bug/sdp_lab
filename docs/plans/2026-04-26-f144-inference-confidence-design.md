# Inference Confidence & Quality Control — Design

> **Status:** Design (2026-04-26) · **Owner:** Andrei · **Target feature:** F144
>
> **Numbering note:** F100–F143 заняты (последний — F143 closed CI bugs про oneshot-stop-gate и main consistency). F144 — следующий свободный.
>
> **Scope:** `sdp_lab` (core library + 3 in-tree адаптера). Не публикуется отдельно — фича внутренняя.
>
> **Parent context:** quorum-верификация ([internal/verify/quorum.go](../../internal/verify/quorum.go)) — multi-judge для артефактов SDP, не для вызовов LLM. Текущее покрытие confidence — точечное в [internal/architect/llm/](../../internal/architect/llm/) (per-item score внутри одного промпта). Этот дизайн закрывает gap «generic confidence wrapper над LLM-инференсом».
>
> **Trigger:** челлендж День 7 «Оценка уверенности и контроль качества инференса». Дух челленджа сохраняется (≥3 техники, замеры по корректным/edge/adversarial, метрики rejection/retry/latency/cost), но реализация подгоняется под реальные точки SDP, где ошибка инференса дорого стоит.

## 1. Why now

1. **LLM-вызовы в SDP принимают необратимые решения.** ws-verdict выносит `passed/failed` на workstream → или зашипили сломанное, или зарезали рабочее. architect classify задаёт стиль/паттерны на весь репо — кривая классификация портит весь downstream-анализ. dispatch classify маршрутизирует задачу в harness/agent — потеря работы при ошибке.
2. **Текущая защита — только формат.** [cmd/sdp-ws-verdict-validate/](../../cmd/sdp-ws-verdict-validate/) проверяет JSON schema verdict, но не семантическую уверенность. Низкая температура (0.2) детерминирует output, но **не валидирует** его.
3. **Confidence уже есть точечно.** Per-item `confidence: 0.0-1.0` живёт в [internal/architect/llm/prompt_patterns.go](../../internal/architect/llm/prompt_patterns.go) и аналогах. Но: (a) это аннотация модели на свои же ответы (single-pass, без cross-check), (b) нет агрегата по всему ответу, (c) нет gating UNSURE → human review.
4. **Quorum существует, но он про артефакты.** [internal/verify/quorum.go](../../internal/verify/quorum.go) — multi-verifier подход для SDP-артефактов (QA/Security/Policy ≥ MinApprovals). Концепции переиспользуемы, но выход (Verdict + Confidence) другой и API не подходит для inline-инференса.
5. **Нет gating на UNSURE.** Сейчас низкоуверенный ответ всё равно идёт в продакшн. Челлендж требует явного механизма «модель не уверена — не пропускаем».

## 2. Goals / Non-goals

**Goals:**
- Generic confidence wrapper `internal/inference/confidence/` — переиспользуемая библиотека поверх существующего `LLMClient`. Output: `Result{Answer, Status ∈ {OK, UNSURE, FAIL}, Score ∈ [0,1], Reasons, Trace}`.
- Реализовать **минимум 3 техники** из челленджа: **self-check** (critic-prompt с заменой роли), **redundancy** (N-sample voting через temperature jitter), **constraint-based** (JSON schema + invariants). Scoring выходит как агрегат по этим трём.
- Применить wrapper к **трём call-site'ам** с разными профилями cost/criticality:
  - **A. ws-verdict second-opinion** (critical / низкая частота / большой output): full set техник, UNSURE → `bd human <id>`.
  - **B. architect classify** (high-stakes / средняя частота / структурный JSON): full set техник, UNSURE → auto-retry с увеличенным N.
  - **C. dispatch classify** (hot path / высокая частота / маленький output): только constraint + lite self-check (single-pass), UNSURE → conservative fallback.
- Telemetry: latency overhead, cost overhead (token delta), rejection rate (FAIL), retry rate (UNSURE→retry succeeded), human-handoff rate (UNSURE→human).
- Test corpus + replay harness: корректные / edge / adversarial входы, golden ожидания, metric report в `internal/build/.sdp/evidence/`.

**Non-goals:**
- Не делаем **fine-tuning** или дообучение моделей (явный запрет челленджа, плюс не оправдано для F144).
- Не трогаем `internal/verify/quorum.go` API — он живёт параллельно для artifact-уровня. Возможна общая абстракция `Verdict` через type alias, но без рефакторинга quorum.
- Не реализуем **cross-model consensus** (вызов нескольких провайдеров через `modelgateway`). Это естественное расширение, но требует отдельной cost/policy дискуссии — выносится в follow-up F1XX.
- Не пишем generic «retry до победного» — UNSURE retry имеет жёсткий бюджет (1 повтор, потом эскалация).
- Не делаем UI/dashboard. Метрики — JSON evidence + markdown report.
- Не покрываем embedding-вызовы (`TaskClassEmbedding` из `kernel`) — там confidence-понятие иное.

## 3. Approach per workstream

### F144-01 · Core library `internal/inference/confidence/`

**Проблема:** Нет переиспользуемой обёртки над `LLMClient` с output'ом `(answer, status, score)`. Каждый call-site делал бы свой ad-hoc check.

**Решение:**

```go
package confidence

type Status string
const (
    StatusOK     Status = "ok"      // score >= ok_threshold
    StatusUnsure Status = "unsure"  // ok_threshold > score >= fail_threshold
    StatusFail   Status = "fail"    // score < fail_threshold
)

type Result[T any] struct {
    Answer    T
    Status    Status
    Score     float64        // [0,1] aggregate
    SubScores map[string]float64 // per-strategy: "self_check", "consensus", "constraint"
    Reasons   []string       // human-readable why score is what it is
    Trace     Trace          // sample answers, timings, token usage
    Attempts  int
}

type Trace struct {
    Samples       []SampleTrace  // populated by NSample
    SelfCheckLog  *SelfCheckLog  // populated by SelfCheck
    ConstraintLog *ConstraintLog // populated by Constraint
    LatencyMs     int64
    TokensIn      int
    TokensOut     int
    CostUSD       float64
}

type Strategy[T any] interface {
    Run(ctx context.Context, req Request[T]) (subScore float64, log any, err error)
    Name() string
}

type Policy struct {
    OKThreshold     float64    // default 0.8
    FailThreshold   float64    // default 0.5
    Weights         map[string]float64 // strategy name → weight, normalized
    UnsureBehavior  UnsureBehavior     // RetryOnce | HumanHandoff | ConservativeFallback
    MaxLatencyMs    int64      // soft budget; strategies short-circuit if exceeded
    MaxCostUSD      float64    // hard budget; strategies short-circuit if exceeded
}

type Checker[T any] struct {
    client     llmclient.Client  // существующий клиент (architect/llm_client.go)
    parser     ResponseParser[T] // переиспользует parser из call-site
    strategies []Strategy[T]
    policy     Policy
    telemetry  TelemetrySink
}

func (c *Checker[T]) Check(ctx context.Context, req Request[T]) (Result[T], error)
```

**Ключевое:**
- Generic over `T` (Go 1.21+ generics) — answer-тип задаётся call-site'ом, scorer не привязан к одному формату.
- `Strategy` interface — extensible (можно добавить cross-model strategy позже без правки `Checker`).
- `Policy` строго конфигурируется per call-site, не глобально.
- `Trace` всегда сериализуется в evidence (даже на FAIL) — для replay-harness и debugging.
- Budget cutoff: если `MaxLatencyMs` исчерпан, оставшиеся стратегии возвращают `subScore = 0.5, log = "skipped: budget"`. Hard cost cap = ошибка.

**Acceptance:** unit tests на (a) успех всех стратегий, (b) одна fail, (c) budget cutoff, (d) пустой набор стратегий → ошибка валидации Policy. ≥85% coverage пакета. Документация `internal/inference/confidence/README.md` с canonical usage example.

### F144-02 · Strategy: self-check (critic-prompt)

**Проблема:** Single-pass CoT даёт reasoning, но модель не оспаривает свои выводы.

**Решение:**
- Two-prompt protocol: (1) основной inference (как сейчас), (2) **critic-prompt** с явной сменой роли: «Ты — рецензент. Вот вход X, вот предложенный ответ Y. Найди ошибки, противоречия, пропущенные кейсы. Верни JSON `{"verdict": "agree"|"disagree"|"unsure", "confidence": 0..1, "issues": [...]}`».
- Критик использует **тот же** провайдер/модель (cross-model — отдельная работа), но с другим system prompt и независимым context (без истории основного вызова). Temperature критика: 0.0 (детерминизм для оценки).
- `subScore` маппится: `agree → confidence`, `disagree → 1 - confidence` (инверсия), `unsure → 0.5`.
- Lite-mode (для горячих путей вроде dispatch): inline-инструкция в основной промпт «...затем самокритично проверь свой ответ и верни `self_score: 0..1`». Один round-trip, дешёвый, но слабее full critic.

**Acceptance:** golden tests на 5+ кейсов: модель должна disagree на adversarial inputs (inject conflict), agree на корректных, unsure на ambiguous. Lite vs full mode: оба возвращают валидный subScore, full mode даёт более разделимый сигнал на тестовом корпусе (документировано числом).

### F144-03 · Strategy: N-sample redundancy (temperature jitter)

**Проблема:** Один LLM вызов — это сэмпл из распределения. Низкая температура снижает разброс, но не устраняет; и не даёт сигнала «эта задача стабильна / нестабильна».

**Решение:**
- N параллельных вызовов того же промпта с разными temperature: `[0.0, 0.3, 0.7]` для N=3 (default) или `[0.2, 0.5]` для N=2 (cheap mode).
- Параллелизм через `errgroup` с общим context-deadline.
- **Agreement metric** — задаётся parser'ом call-site'а (не universal): equality после нормализации, semantic-similarity для свободного текста, structural-match для JSON. Возвращает `[0,1]`.
- `subScore = mean_pairwise_agreement`. Если N=3: 3 пары → агрегируем как среднее. На полном consensus (`agreement = 1.0` для всех пар) — `subScore = 1.0`.
- **Tie-break для финального answer:** majority vote, на ничьей — sample с минимальной temperature (наиболее детерминированный).
- Token budget: каждый call засчитывается; total cost = N × baseline cost + critic. Метрика `cost_overhead_ratio` = `total / baseline`.

**Acceptance:** unit tests с моками: identical samples → score 1.0, all-different → score 0 (для дискретного parser'а), partial agreement → промежуточный. Real-LLM smoke test (опционально, под `-tags=integration`).

### F144-04 · Strategy: constraint-based + scoring composer

**Проблема:** Часть проверок дешёвые (формат, диапазоны, инварианты) — их нужно делать первыми.

**Решение:**
- Constraint strategy получает callback'и от call-site:
  - `SchemaValidator` — JSON-Schema (переиспользует существующие схемы из `schema/`).
  - `Invariants` — список predicate-функций над answer'ом (`func(T) (bool, string)`).
- `subScore = 1.0` если все прошли, `0.0` если schema fail, частичный score = доля прошедших invariants.
- **Scoring composer** агрегирует: `score = Σ(weight_i × subScore_i) / Σ(weight_i)`. Default: `self_check=0.4, consensus=0.4, constraint=0.2`. Per-call-site override.
- Status mapping: `score >= 0.8 → OK`, `[0.5, 0.8) → UNSURE`, `< 0.5 → FAIL`. Constraint hard-fail (schema invalid) форсит `FAIL` независимо от score других стратегий — semantic-уверенность не имеет смысла на сломанном формате.

**Acceptance:** golden table-driven tests на матрице (constraint pass/fail × consensus high/low × self-check agree/disagree) → ожидаемый Status. Документация веса+пороги в [internal/inference/confidence/README.md](../../internal/inference/confidence/) с обоснованием стартовых значений.

### F144-05 · Adapter: ws-verdict second-opinion

**Проблема:** [cmd/sdp-ws-verdict-validate/main.go](../../cmd/sdp-ws-verdict-validate/main.go) валидирует только JSON schema. Семантическая корректность verdict (passed/failed обоснован) — нет.

**Решение:**
- Новая команда `sdp ws-verdict-confidence-check` (или флаг `--confidence` к существующей).
- Profile: full set (constraint + nsample N=3 + self-check critic).
- UNSURE → `bd human <id> --reason="ws-verdict confidence below threshold"`. Human pull даёт явный merge/reject.
- FAIL → блокирует merge (CI gate), требует пересборки verdict.
- Evidence path: `internal/build/.sdp/evidence/<id>/ws-verdict-confidence.json`.

**Acceptance:** integration test на 3 cases (clean OK, ambiguous → UNSURE → human handoff invoked, broken → FAIL blocks). Telemetry рекордит все три категории.

### F144-06 · Adapter: architect classify

**Проблема:** [internal/architect/classify/](../../internal/architect/classify/) возвращает per-item confidence, нет агрегата по всей классификации.

**Решение:**
- Wrap `Hypothesizer` через `confidence.Checker[ClassificationResult]`.
- Profile: full set, но N=3 фиксировано (классификация — критичная для downstream).
- UNSURE → auto-retry с N=5 и temperature spread `[0.0, 0.2, 0.4, 0.6, 0.8]`. Если всё ещё UNSURE → mark result `partial` и продолжаем без блока (architect — анализ, не gate).
- Aggregate score: средний confidence по items × consensus on top-1 классификации.

**Acceptance:** test на синтетическом репо (clear layered → OK, mixed style → UNSURE → retry → OK or `partial`, garbage input → FAIL).

### F144-07 · Adapter: dispatch classify (lite)

**Проблема:** [internal/dispatch/classify.go](../../internal/dispatch/classify.go) — hot path, full confidence overhead убьёт latency.

**Решение:**
- Lite profile: только constraint + lite self-check (single-pass with self_score). N-sample выключен.
- UNSURE → conservative fallback: задача идёт в default-harness (Claude Code) с label `confidence:unsure`.
- Cost overhead < 10% (constraint бесплатна, lite self-check добавляет ~50 tokens output).

**Acceptance:** latency benchmark: `BenchmarkDispatchClassify` до и после, регресс < 15%. Functional test: на shadowed inputs lite-mode даёт корректный fallback.

### F144-08 · Test corpus + replay harness + metrics report

**Проблема:** Без замеров (дух челленджа) фича не закрыта.

**Решение:**
- `internal/inference/confidence/testdata/` структура:
  ```
  ws-verdict/{correct,edge,adversarial}/*.json
  architect/{correct,edge,adversarial}/*.json
  dispatch/{correct,edge,adversarial}/*.json
  ```
- Replay harness: `cmd/sdp-confidence-replay` — берёт fixture, гонит через `Checker`, выдаёт `{input, expected_status, actual_status, score, latency_ms, tokens, cost_usd}`.
- Aggregator: суммирует по category × call-site → markdown report `docs/research/2026-04-26-f144-confidence-replay-report.md`.
- Метрики на report:
  - **Rejection rate** (FAIL): % adversarial → FAIL (хорошо ≥80%), % correct → FAIL (плохо, должно быть ≤2%).
  - **Retry rate** (UNSURE→retry в architect): % UNSURE, % UNSURE→OK после retry.
  - **Human handoff rate** (UNSURE в ws-verdict): % всех вызовов.
  - **Latency overhead**: p50/p95 wrapped vs raw, по call-site'у.
  - **Cost overhead**: total tokens × price вrapped vs raw, по call-site'у.

**Acceptance:** report сгенерирован, все 5 метрик заполнены, есть baseline-numbers (raw без wrapper'а) и delta-numbers (wrapped). Replay harness запускается локально и в CI (под `-tags=replay`, не блокирует main CI).

## 4. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Self-check критик соглашается с любым ответом основного вызова (sycophancy) | False high confidence | Adversarial test cases в corpus специально подбираются на sycophancy; metric `disagree_rate_on_adversarial` ≥ 60% — гейт для merge F144 |
| Temperature jitter не даёт реального разброса на детерминированных задачах | Consensus всегда 1.0 → no signal | Fallback: при `agreement = 1.0` для всех пар проверяем variance в reasoning text, не только в structured output. Если рассуждения тоже identical, признаём задачу легко-стабильной (это OK signal) |
| Cost overhead 2-3× ломает бюджет на горячих путях | Бюджет проекта | Per-call-site policy: lite mode для dispatch (N=1), full только для critical. Hard `MaxCostUSD` cap в `Policy` |
| UNSURE → human handoff заваливает inbox | Operator burnout | UNSURE rate измеряется в reporting; если > 10% на ws-verdict, поднимаем threshold до калибровки |
| Generic library добавляет dependency на architect/llm_client.go | Cross-package coupling | `confidence` зависит только от `llmclient` (общий низкоуровневый клиент), не от architect. Архитектурное правило фиксируется в README пакета |
| JSON schema constraint реализован неконсистентно в архитектурных промптах vs ws-verdict | False FAIL/OK | Re-export канонических схем из `schema/` через `confidence.NewJSONSchemaConstraint(path)` — single source |

## 5. Dependencies & Sequencing

```
F144-01 (core lib types)
  ├── F144-02 (self-check strategy)
  ├── F144-03 (n-sample strategy)
  └── F144-04 (constraint + composer)
        ├── F144-05 (ws-verdict adapter)         ← critical, ship first
        ├── F144-06 (architect adapter)
        └── F144-07 (dispatch adapter, lite)
F144-08 (replay + metrics) ← depends on at least one adapter, runs parallel after F144-05
```

Critical path: 01 → 04 → 05 → 08. Параллелизм возможен после 04: 05/06/07 идут одновременно.

## 6. Test strategy

- **Unit (TDD):** каждая стратегия + scorer + policy edge cases. Coverage ≥ 85% пакета.
- **Integration:** real-LLM smoke под `-tags=integration` (опт-ин, требует `OPENROUTER_API_KEY`). Не в default CI.
- **Replay (F144-08):** против fixture corpus. Запускается под `-tags=replay`, метрики evidence-bound.
- **Adversarial corpus:** минимум 5 кейсов на call-site, специально crafted на (a) sycophancy, (b) prompt injection в input, (c) inconsistent gold answer, (d) format-only correct (semantically wrong), (e) noisy/truncated input.

## 7. Open questions (resolve during build)

1. **Schema location.** Класть ws-verdict confidence schema в `schema/inference/confidence-result.json` (новое поддерево) или extend `schema/contracts/`? — Решим при F144-01 при генерации первой схемы.
2. **Где хранить `confidence.Policy` per call-site.** Inline literal в коде adapter'а (сейчас) или config-файл `configs/inference-confidence.yaml` (testable, но overhead)? — Default: inline + комментарий с обоснованием. Migrate to config если call-site'ов > 5.
3. **`bd human` integration shape.** Прямой shell-out `bd human <id>` или Go-binding через `internal/beads/`? — Зависит от состояния `internal/beads/` API на момент F144-05.
4. **Replay corpus как репозиторий.** В sdp_lab inline (testdata/) или отдельный репо `sdp-eval-corpus`? — Default: inline, выносим если > 50MB.

## 8. Rollout

- Branch: `feature/F144-inference-confidence`. Базируется на `main`. Worktree: `.worktrees/f144-inference-confidence`.
- Draft PR открывается после F144-01 (core types).
- Adapter merging: F144-05 (ws-verdict) первым в production, остальные адаптеры — отдельными commits в той же PR. Feature-flag не нужен — wrapping включается per-call-site по решению автора call-site'а.
- Backwards-compat: ни один существующий вызов не оборачивается без явного opt-in (через переход call-site'а на `confidence.Checker`). До этого — старое поведение.
- Replay-report (F144-08) — первая итерация в этой PR; дальше — auto-update через CI на nightly.

## 9. Success criteria

1. ≥3 техники реализованы и интегрированы в три call-site (Self-check + Redundancy + Constraint, четвёртый Scoring — композитный, фактически 4 закрыто).
2. Replay report содержит все 5 метрик из F144-08 с baseline + delta.
3. Adversarial rejection rate ≥ 80% на test corpus, false-FAIL rate ≤ 2% на correct corpus.
4. Latency overhead на dispatch call-site < 15% (hot path); на ws-verdict < 200% acceptable (rare path).
5. Минимум один UNSURE случай в replay поднял `bd human` (демонстрация end-to-end gating).
6. Документация: README пакета + один cookbook-скрипт `examples/confidence-usage.go`.

## 10. References

- Челлендж День 7 (формулировка задачи в conversation 2026-04-26).
- [internal/verify/quorum.go](../../internal/verify/quorum.go) — concept inspiration, не реюз.
- [internal/architect/llm/prompt_patterns.go](../../internal/architect/llm/prompt_patterns.go) — текущий per-item confidence, мигрируем на агрегат через wrapper.
- [cmd/sdp-ws-verdict-validate/main.go](../../cmd/sdp-ws-verdict-validate/main.go) — extension point для F144-05.
- [internal/dispatch/classify.go](../../internal/dispatch/classify.go) — extension point для F144-07.
- [docs/plans/2026-04-25-f141-multi-harness-install-bootstrap-design.md](2026-04-25-f141-multi-harness-install-bootstrap-design.md) — design template style.

---

**Next step:** decompose into beads epic F144 + 8 children (`bd create` × 9), generate workstream files for each leaf (`docs/workstreams/backlog/00-144-XX.md`), open draft PR.
