# Inference Decomposition Framework — Design

> **Status:** Design (2026-04-26) · **Owner:** Andrei · **Target feature:** F146
>
> **Numbering note:** F144 (confidence) merged 2026-04-26 (PR #131). F145 (multi-provider cascade) in progress (sdplab-ldmq). F146 — следующий свободный.
>
> **Scope:** `sdp_lab` (core library + первый adapter `ws-verdict` + bench cmd). Не публикуется отдельно — фича внутренняя.
>
> **Parent context:**
> - **F144** ([internal/inference/confidence/](../../internal/inference/confidence/)) — gate качества **одного ответа** (self-check / N-sample / constraint). Per-prompt scoring.
> - **F145** ([internal/dispatch/cascade/](../../internal/dispatch/cascade/), in progress) — fallback chain между провайдерами для **одного запроса** (escalate on UNSURE/FAIL).
> - **F146** (this) — разбиение **одной сложной задачи** на цепочку коротких этапов со строгим форматом между ними. Ортогональная третья ось: F144 = quality, F145 = provider, F146 = work split.
>
> **Trigger:** челлендж День 9 «Декомпозиция инференса (multi-stage inference)». Дух челленджа сохраняется (monolithic vs multi-stage A/B, разные модели per stage, строгий формат), реализация подгоняется под реальные точки SDP, где сложный single-shot инференс плохо масштабируется по cost/latency/accuracy.

## 1. Why now

1. **Single-shot инференс ws-verdict сейчас монолитен.** [cmd/sdp-ws-verdict-validate](../../cmd/sdp-ws-verdict-validate/) проверяет JSON schema; F144 wrapper навешивает confidence — но семантическая работа (extract → classify gates → aggregate) всё равно делается как один большой промпт. Один failure pattern (формат сломался на любом из gate'ов) — потеряна вся работа.
2. **Cost/latency не оптимизированы по этапам.** Простой extract можно сделать на Haiku за 200ms, classify — на Sonnet (чувствителен к контексту), aggregate — снова на Haiku. Сейчас всё идёт на одной модели per call-site, цена доминируется самым требовательным шагом.
3. **F144 confidence per-stage не имеет структурного crutch'а.** Confidence wrapping применяется к одному вызову; нет паттерна «оборачивать каждый stage в свой confidence profile» (extract → constraint, classify → self-check + N-sample). F146 даёт каркас, в котором F144 натурально подключается per-stage.
4. **F145 cascade не имеет hierarchical control.** Cascade chain один на запрос. С F146 каждый stage может иметь свой cascade (extract: Haiku→Sonnet, classify: Sonnet→Opus, aggregate: Haiku) — гранулярная экономия.
5. **Нет benchmark'а monolithic vs decomposed.** Дух челленджа — измеримое сравнение. Без F146 нет infrastructure для запроса «дай мне A/B на golden corpus».

## 2. Goals / Non-goals

**Goals:**

- Generic decomposition library `internal/inference/decompose/` поверх существующих `llmclient`, `confidence` и (когда merge) `cascade`. Output: `Result[Final]{Answer, StageResults[], AggregateTrace, Status, Score}`.
- **Stage-as-a-type** через Go generics: `Stage[In, Out any]`, type-safe pipeline composition.
- **Three built-in stitchers** (контракт между этапами):
  - `EnumStitcher` — закрытый список значений (классификация решений).
  - `JSONStitcher` — JSON Schema validation (reuse F144 schema infra).
  - `TOONStitcher` — Token-Optimized Object Notation (compact табличный формат для aggregate stages, оптимизирован под token efficiency).
- **Composition с F144/F145** — каждый stage опционально оборачивается в `confidence.Checker` (per-stage policy) и/или `cascade.Invoker` (per-stage chain). Включается через `StageConfig`, не обязателен.
- **Per-stage failure policy:** `Abort` / `RetryOnce` / `Fallback` (mirror F144 `UnsureBehavior`).
- **First adapter — ws-verdict** как 3-stage pipeline: extract artifacts → per-gate classify → aggregate verdict.
- **Monolithic baseline + bench harness** — `cmd/sdp-decompose-bench` гонит monolithic vs decomposed на одном corpus'е, выдаёт A/B report с latency/tokens/accuracy delta.
- Telemetry: per-stage trace + aggregate roll-up. Reuse F144 `Trace` структуры с расширением.

**Non-goals:**

- Не делаем **runtime pipeline configuration** (YAML/JSON pipeline definitions). Сейчас — только Go-defined pipelines. Динамическая регистрация по имени — follow-up F1XX.
- Не покрываем **streaming pipelines** (Stage emits chunks → next Stage consumes streaming). Только request/response модель.
- Не реализуем **branching DAG** (один stage с несколькими out-edges). Только linear chain.
- Не трогаем `confidence.Checker` API — F146 использует его как dependency, не модифицирует.
- Не публикуем decomposition как первоклассный публичный SDP интерфейс (не `sdp pipeline ...` CLI). Только internal library + первый adapter + bench cmd.
- Не интегрируем decomposition в `architect classify` или `dispatch classify` — F146 ограничен ws-verdict как proof-of-concept; расширение — отдельные follow-up'ы.
- Не покрываем embedding-вызовы (`TaskClassEmbedding`) — то же ограничение что у F144.

## 3. Approach per workstream

### F146-01 · Core library `internal/inference/decompose/`

**Проблема:** Нет переиспользуемой обёртки `Pipeline` с output'ом `Result[Final]`. Каждый call-site писал бы свой ad-hoc chain.

**Решение:**

```go
package decompose

// Stage is a single step in the pipeline. In is the upstream output type;
// Out is the type emitted to the next stage (or the final type if last).
type Stage[In, Out any] interface {
    Name() string
    Run(ctx context.Context, in In) (Out, StageTrace, error)
}

// StageConfig wires optional cross-cutting concerns onto a Stage.
type StageConfig struct {
    Confidence  *confidence.Policy   // F144 wrap, optional
    Cascade     *cascade.Policy      // F145 wrap, optional (when F145 lands)
    Stitcher    Stitcher             // strict format guard for Run output
    OnFailure   FailurePolicy        // Abort | RetryOnce | Fallback
    FallbackOut any                  // used when OnFailure == Fallback
    Timeout     time.Duration
}

// Pipeline carries a chain of stages. Built via fluent helpers
// (Then[A,B,C], Then2, etc.) to keep generic types tractable.
type Pipeline[Final any] struct {
    stages   []anyStage // erased; type-checked at construction
    policies []StageConfig
    name     string
}

type Result[Final any] struct {
    Answer       Final
    Status       Status              // OK | UNSURE | FAIL (mirrors F144)
    Score        float64             // [0,1] aggregate across stages
    StageResults []StageResult       // per-stage outcome + trace
    Trace        AggregateTrace      // total latency, tokens, cost
    Reasons      []string
}

type StageResult struct {
    Name     string
    Status   Status
    SubScore float64
    Trace    StageTrace
    Output   any                     // raw, for debugging
    Retries  int
}

type StageTrace struct {
    LatencyMs    int64
    TokensIn     int
    TokensOut    int
    CostUSD      float64
    StitcherErr  string              // empty if format passed
    ConfidenceLog *confidence.Trace  // populated if F144 wrapped
    CascadeLog    *cascade.Trace     // populated if F145 wrapped
}

func New[Final any](name string) *Pipeline[Final]
func (p *Pipeline[Final]) Run(ctx context.Context, in any) (Result[Final], error)
```

**Construction (avoiding generic-chain explosion):**

```go
// Two-stage construction with explicit type binding.
pipe := decompose.New[FinalVerdict]("ws-verdict-pipeline")
s1 := decompose.NewStage[Diff, ExtractedFacts](...)
s2 := decompose.NewStage[ExtractedFacts, GateMatrix](...)
s3 := decompose.NewStage[GateMatrix, FinalVerdict](...)
pipe.Then(s1, cfg1).Then(s2, cfg2).Then(s3, cfg3)
```

**Ключевое:**
- Generic over `Final` для top-level result-типа. Per-stage `In/Out` зафиксированы при `NewStage`. Type-check на `Then(...)` — output prev stage = input next stage.
- `Stage` — узкий interface; реализация — обычная Go-функция через `NewStage(name, fn)`.
- `Result.Score` — взвешенное среднее `StageResult.SubScore`. Default weights: equal across stages; override через `Pipeline.SetWeights`.
- `StageTrace.StitcherErr` всегда заполнен (empty = ok). Persistence в evidence.
- `Pipeline.Run` идёт sequentially; если stage возвращает error, действует `StageConfig.OnFailure`.

**Acceptance:** unit tests на (a) happy path 3 stages, (b) stage 2 fail → Abort, (c) stage 2 fail → RetryOnce → ok, (d) stage 2 fail → Fallback (использован FallbackOut), (e) timeout per-stage, (f) generic type-mismatch caught at compile time (compile-fail negative test). ≥85% coverage. README пакета с canonical usage example.

### F146-02 · Stitchers (Enum / JSON / TOON)

**Проблема:** Output stage N идёт в input stage N+1. Без structural validation любой format-drift модели ломает цепь.

**Решение:**

```go
// Stitcher is invoked on the Out value of a stage before it becomes the In
// of the next stage. Pipeline calls Validate; on err, the StageConfig.OnFailure
// policy fires. Marshal renders Out into the prompt context for the next stage.
type Stitcher interface {
    Name() string
    Validate(out any) error       // semantic: enum membership, schema, TOON shape
    Marshal(out any) (string, error)
}
```

**Three implementations:**

1. **`EnumStitcher`** — `Validate` проверяет, что value входит в зарегистрированный набор (`pass|warn|fail`, `P0|P1|P2|P3`). `Marshal` — просто `string(value)`. Cheapest format. Используется на classify-stages с дискретным output.

2. **`JSONStitcher`** — обёртка над JSON Schema validator (reuse `confidence.NewJSONSchemaConstraint`-стиль). Принимает schema path или embedded schema. `Validate` гоняет schema; `Marshal` — `json.Marshal` с pretty-print под limit. Используется на extract-stages.

3. **`TOONStitcher`** — Token-Optimized Object Notation: компактный табличный формат для аггрегатов и листов структур.
   - **Layout:** заголовок `# field1 | field2 | field3`, далее `value1 | value2 | value3` per row.
   - **Validate:** парсит обратно в map/slice, проверяет column types против registered schema.
   - **Marshal:** рендерит структуру в TOON-grid; на single-record выдаёт inline `field1=v1, field2=v2`.
   - **Token cost vs JSON:** ожидаемая экономия ~40-60% на табличных aggregate output'ах (без quote'ов, без braces, без repeated keys per row).

**Acceptance:**
- Per-stitcher unit tests: valid input → no error + correct marshal; invalid → typed error.
- Cross-stitcher property test: `Marshal(x); Validate(parsed) == nil` для всех valid x.
- Bench `BenchmarkStitcher_Marshal_<JSON|TOON>` на одном corpus'е → token-count delta задокументирован в README.
- TOON edge cases: nested objects (запрещены в v1, error), null cells, empty rows.

### F146-03 · Confidence + cascade integration per stage

**Проблема:** F144 wrapper и (готовящийся) F145 cascade — отдельные обёртки над `LLMClient`. Pipeline должен композировать их per-stage без дублирования telemetry или owners.

**Решение:**

- `Pipeline.Run` для каждого stage:
  1. Берёт `StageConfig.Cascade` (если задан) — оборачивает базовый stage `Run` в `cascade.Invoker.Invoke`. Возвращает первый OK или эскалирует.
  2. Поверх результата — `StageConfig.Confidence` (если задан) — `confidence.Checker.Check`. UNSURE/FAIL транслируются в `Status` stage'а.
  3. Stitcher applies на финальный output (после confidence, до передачи next stage).
- **Order matters:** cascade — provider-level retry (escalation между моделями); confidence — semantic gate. Cascade сначала (получаем best provider answer), потом confidence (проверяем семантику best provider answer).
- **Trace merge:** `StageTrace.ConfidenceLog` берётся из F144, `CascadeLog` — из F145. `Pipeline` кладёт оба в `StageResult.Trace`.
- **Aggregate score:** `Result.Score = mean(StageResult.SubScore)`; при наличии Confidence используется его score, при отсутствии — `1.0` если stage прошёл, `0.0` если failed.
- **Status escalation:** одна stage в FAIL → весь Result FAIL (если OnFailure не вернул RetryOnce). Любая UNSURE → Result UNSURE (если все остальные OK). Все OK → OK.

**F145 readiness:** на момент F146-03 F145 в progress (sdplab-ldmq не merged). Реализация F146-03 должна быть **forward-compatible**: интерфейс `cascade.Invoker` объявлен в `decompose` как локальный narrow-interface, plug-in реальной F145 реализации — после её merge. Stub-implementation в тестах через mock.

**Acceptance:** integration test 3 случая: (a) stage только с Confidence (cascade nil), (b) stage только с Cascade (через mock), (c) stage с обоими. Каждый кейс проверяет: правильная последовательность вызовов, корректный merged Trace, status propagation в Result.

### F146-04 · ws-verdict pipeline adapter

**Проблема:** Текущий ws-verdict — single-shot LLM call с JSON schema validation post-hoc. Не использует F146 caркас — соответственно нет per-stage telemetry, нет stage-level retry, нет gate-by-gate scoring.

**Решение — 3-stage pipeline:**

```
Stage 1 — extract       (Haiku, JSONStitcher)
  In:  ws diff + git status + test report
  Out: ExtractedFacts {files_changed, tests_added, coverage_delta, lint_status, build_status}

Stage 2 — classify      (Sonnet, EnumStitcher per gate)
  In:  ExtractedFacts
  Out: GateMatrix {build, test, lint, coverage} → each ∈ {pass, warn, fail} + reason

Stage 3 — aggregate     (Haiku, TOONStitcher)
  In:  GateMatrix
  Out: FinalVerdict {verdict ∈ {passed, partial, failed}, blocking_gates[], score, summary}
```

**Per-stage configs:**

| Stage | Model | Confidence Profile | Stitcher | OnFailure |
|---|---|---|---|---|
| extract | Haiku | constraint-only (lite) | JSON | RetryOnce |
| classify | Sonnet | self-check + N=3 sample | Enum (per gate) | Abort |
| aggregate | Haiku | constraint-only (lite) | TOON | Fallback (default fail-safe verdict) |

**Adapter file:** `internal/inference/decompose/adapters/wsverdict/pipeline.go`. Точка входа — `NewWSVerdictPipeline(client llmclient.Client) *decompose.Pipeline[FinalVerdict]`. Используется из нового флага в `cmd/sdp-ws-verdict-validate` (`--decompose`) или из bench cmd напрямую.

**Schema reuse:** F144 schema'ы под `schema/inference/` extended. JSONStitcher для extract использует `schema/inference/ws-verdict-extracted.json` (новая). Aggregate схема — extension существующей `ws-verdict-result.json`.

**Acceptance:**
- 3-stage pipeline executes end-to-end on 3 fixture inputs (clean pass / mixed / fail).
- Stage 2 fail (e.g., classify timeout) корректно стопает и Result.Status == FAIL с правильным StageResult.Trace.
- Per-stage tokens/latency задокументированы в evidence; sum совпадает с aggregate Trace.

### F146-05 · Monolithic baseline для A/B

**Проблема:** Декомпозиция без сравнения — не evidence. Нужен baseline single-prompt, чтобы измерить delta.

**Решение:**

- **`internal/inference/decompose/adapters/wsverdict/monolithic.go`** — single-prompt версия: один большой промпт, на вход — тот же diff/test report, на выход — тот же `FinalVerdict` (полная JSON структура). Промпт сконструирован из объединения instruction'ов трёх stage'ов decomposed pipeline.
- **Confidence profile:** F144 full set (self-check + N=3 + constraint). Это дань честному сравнению — baseline должен быть «лучшей версией single-shot», иначе A/B нечестен.
- **Model:** Sonnet (consistent с classify-stage decomposed pipeline; Haiku был бы заведомо хуже на single-shot).
- Возвращает `Result[FinalVerdict]` совместимого формата (с `StageResults` длины 1 — псевдо-stage `monolithic`), чтобы bench harness обрабатывал оба пути одинаково.

**Acceptance:**
- Monolithic adapter возвращает корректный `FinalVerdict` на всех трёх fixture inputs.
- Trace содержит latency, tokens, cost (sanity-check vs LLM provider response).
- Output structure 1:1 совпадает с decomposed для одинаковых inputs (semantic equivalence — отдельный gate в bench).

### F146-06 · Bench harness + A/B report

**Проблема:** Без харнесcа A/B сравнение — это вручную сравнить evidence-логи. Нужен автоматический cmd для repeatable measurement.

**Решение:**

- **`cmd/sdp-decompose-bench/main.go`** — берёт corpus path, для каждого fixture гонит обе версии (monolithic + decomposed), пишет CSV/markdown report.
- **Corpus:** F144 ws-verdict corpus переиспользуется (`internal/inference/confidence/testdata/ws-verdict/{correct,edge,adversarial}/`). Это даёт нечестное преимущество F144-tuned данным, но **полное соответствие задаче пользователя** («не усечённый корпус»).
- **Helper extraction:** общие части replay (corpus loader, fixture format, evidence writer) выделяем в `internal/inference/replayutil/`. Используется и `cmd/sdp-confidence-replay` (рефакторинг — зашивка нового пакета без semantic change), и `cmd/sdp-decompose-bench`.
- **Output report** `docs/research/2026-04-26-f146-decomposition-replay-report.md`:
  - Per-fixture row: `id | category | monolithic_status | decomposed_status | latency_mono | latency_decomp | tokens_mono | tokens_decomp | cost_mono | cost_decomp`.
  - Aggregate sections: latency p50/p95, total tokens, accuracy (vs golden status), TOON token-saving %.
  - **Hypothesis verdict:** evidence-backed утверждение «decomposition даёт X% saving / Y% accuracy delta / Z% latency delta».

**Metrics:**

| Metric | Definition | Acceptance threshold |
|---|---|---|
| Accuracy | `% (status == golden)` | decomposed ≥ monolithic (no regression) |
| Total tokens | sum across all stages | decomposed ≤ monolithic × 0.85 (target: 15% saving) |
| Latency p50 | wall-time per pipeline | decomposed ≤ monolithic × 1.30 (overhead <30%, parallelism в follow-up) |
| Cost USD | tokens × pricing | decomposed ≤ monolithic × 0.70 (Haiku на extract/aggregate — expected 30%+ saving) |
| TOON saving | tokens(JSON marshal aggregate) vs tokens(TOON marshal aggregate) | TOON ≤ JSON × 0.60 (40% saving target) |

**Acceptance:**
- Bench cmd запускается локально и под `-tags=replay` в CI (не блокирует main CI).
- Markdown report сгенерирован, все 5 метрик заполнены, выводы зафиксированы. Порог не блокирует merge — это evidence для дальнейших решений, не gate.
- Replay-utility сохраняет behavior существующего `sdp-confidence-replay` (regression test).

## 4. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Generic chain composition взрывается по сложности типов (3+ stages) | API становится unwieldy | Construction через fluent `.Then()` с type-erased internal storage; type-check в `NewStage` создании, runtime check на `Pipeline.Run`. Альтернативный API через `decompose.Compose2/3/4(s1, s2, ...)` для типичных случаев. |
| TOON parsing хрупкий на edge cases (nested objects, escape) | False FAIL | v1 ограничен flat tables; nested → schema error. Документировано в README. |
| Decomposition увеличивает latency (3 sequential calls > 1 call) | Hot path не годится | Bench measures это явно. Optimization (parallel stages где possible, caching) — follow-up F1XX. F146 принимает overhead в обмен на cost saving + per-stage retry granularity. |
| Inter-stage prompt injection через user-controlled extracted fields | Security gap | Strict structured passing (Q5 ответ) — Stitcher принудительно конвертирует output в structured form (JSON/Enum/TOON). Текстовая инжекция не выходит за structural границы format'а. Не silver bullet, но baseline защита. |
| F145 не merged к моменту F146-03 → blocked work | Sequencing | F146-03 использует local interface для cascade (`cascade.Invoker` через type alias / narrow interface). Mock в тестах. Замена на реальный F145 import — после его merge, без API change. |
| Reuse F144 corpus даёт нечестное преимущество decomposed-варианту (corpus tuned под F144 confidence profile) | Метрики искажены | Документируется в report как known bias. Follow-up F1XX добавит fresh corpus, специально crafted под decomposed scenarios. |
| Adapter (F146-04) дублирует логику с существующим ws-verdict | Code drift | Adapter — opt-in через флаг `--decompose` в `sdp-ws-verdict-validate`. Default behavior не меняется. Migration plan — отдельный issue после bench evidence. |

## 5. Dependencies & Sequencing

```
F146-01 (core lib types)
  └── F146-02 (stitchers)
       └── F146-03 (F144/F145 integration)
            ├── F146-04 (ws-verdict adapter)         ← parallel after 03
            └── F146-05 (monolithic baseline)        ← parallel after 03
                 └── F146-06 (bench harness + report)
```

Critical path: 01 → 02 → 03 → {04 ‖ 05} → 06. После 03 — параллелизм 04+05.

**External dependencies:**
- F144 (`internal/inference/confidence/`) — already merged, hard dependency.
- F145 (`internal/dispatch/cascade/`) — soft dependency, F146 forward-compatible via local narrow-interface.
- `internal/llmclient` — already merged (F108).
- `internal/inference/confidence/testdata/ws-verdict/` — corpus reuse for F146-06.

## 6. Test strategy

- **Unit (TDD):** Per-package coverage ≥85%. Каждый stitcher, Pipeline.Run все ветки failure policy, Stage timeout.
- **Integration:** Real-LLM smoke под `-tags=integration`. Ws-verdict pipeline на 3 fixture inputs end-to-end. Не в default CI.
- **Replay (F146-06):** Bench harness против fixture corpus. Под `-tags=replay`, evidence-bound.
- **Adversarial corpus (от F144):** Reuse — adversarial inputs должны попадать в FAIL bucket в обоих режимах. Если decomposed FAIL'ит на adversarial, а monolithic пропускает — это эксплицитный win, документируется.
- **Compile-fail negative tests:** для generic type-mismatch (output stage N ≠ input stage N+1) — через `// +build go_compile_fail` или отдельный тестовый пакет.

## 7. Open questions (resolve during build)

1. **Schema location.** Куда `ws-verdict-extracted.json` — `schema/inference/` (новая поддиректория, F144 не создавала) или `schema/contracts/`? — Решим при F146-04.
2. **TOON parser library.** Писать свой минимальный (на flat tables хватит ~150 LOC) или искать существующий? — Default: писать свой. Внешний пакет не оправдан для v1 ограниченного scope.
3. **Per-stage parallelism.** Stages currently sequential; stages типа «classify gate A» и «classify gate B» независимы и могли бы идти параллельно. Включаем в v1 или follow-up? — Default: follow-up, F146 sequential для simplicity. Bench покажет, насколько overhead виден.
4. **Aggregate weight calibration.** `Result.Score = mean(SubScore)` — равные веса. Должны ли extract/aggregate weight'иться меньше classify (где основная семантика)? — Стартуем с equal, корректируем после bench evidence.
5. **Bench evidence корпус — как хранить.** Inline в `internal/inference/decompose/testdata/` (новый) или reuse F144 path? — Default: reuse F144 path для F146-06 (Q пользователь указал «не усечённый»), inline для unit tests F146-01..03.

## 8. Rollout

- **Branch:** `feature/F146-inference-decomposition` (already created). Worktree: `.worktrees/f146-inference-decomposition`.
- **Draft PR:** открывается после F146-01 (core types ready).
- **Adapter merging order:** F146-04 (ws-verdict) — opt-in через флаг. Default behavior `sdp ws-verdict-validate` не меняется. Migration to default — отдельный решение после F146-06 evidence.
- **Backwards-compat:** ни один существующий call-site не оборачивается без явного opt-in. F146 = library + первый adapter + bench. Никаких bd-policy / CI-gate изменений.
- **Bench report (F146-06):** первая итерация в этой PR. Дальше — manual re-run на change'ах prompts / pricing.

## 9. Success criteria

1. ≥3 stitcher'а реализованы (Enum / JSON / TOON), все unit-tested ≥85% coverage.
2. ws-verdict 3-stage pipeline end-to-end запускается на 3 fixture inputs.
3. Monolithic baseline даёт сопоставимый output на тех же inputs.
4. Bench cmd сгенерировал markdown report со всеми 5 метриками; A/B verdict зафиксирован.
5. **Token saving (decomposed vs monolithic):** ≥15% on average (target evidence).
6. **TOON token saving (vs JSON aggregate):** ≥40% (target evidence).
7. **Accuracy regression:** none (decomposed ≥ monolithic on golden corpus).
8. Документация: README пакета `internal/inference/decompose/` + cookbook example с ws-verdict adapter.

## 10. References

- Челлендж День 9 (формулировка задачи в conversation 2026-04-26).
- [docs/plans/2026-04-26-f144-inference-confidence-design.md](2026-04-26-f144-inference-confidence-design.md) — F144 design, parent confidence layer.
- F145 epic [sdplab-ldmq](https://github.com/) — multi-provider cascade (in progress).
- [internal/inference/confidence/](../../internal/inference/confidence/) — already-merged confidence wrapper, dependency.
- [internal/llmclient/](../../internal/llmclient/) — base LLM client.
- [cmd/sdp-ws-verdict-validate/](../../cmd/sdp-ws-verdict-validate/) — extension point for F146-04.
- [cmd/sdp-confidence-replay/](../../cmd/sdp-confidence-replay/) — sibling cmd, replayutil source.

---

**Next step:** decompose into beads epic F146 + 6 children (`bd create` × 7), generate workstream files (`docs/workstreams/backlog/00-146-XX.md`), update INDEX.md, open draft PR.
