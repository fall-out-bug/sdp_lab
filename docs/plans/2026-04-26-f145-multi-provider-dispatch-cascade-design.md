# F145 — Multi-Provider Dispatch Matrix & Confidence-Driven Cascade Routing

**Feature:** F145
**Status:** Design (2026-04-26)
**Owner:** Andrei
**Depends on:** F133 (Local model dispatch via Ollama, PR #127), F144 (Inference Confidence & Quality Control, PR #131)
**Branch (planned):** `feature/F145-multi-provider-cascade`

## 1. Problem

SDP `dispatch` слой имеет три задокументированных провала.

**1.1 Тихий bug в model-плюмбинге.** `Router.Route` выбирает `(harness, provider, model)` и записывает в `DispatchDecision`, но два из четырёх harness'ов **молча игнорируют выбранную модель**:
- [internal/dispatch/harness/cursor.go:38](../../internal/dispatch/harness/cursor.go) — `opts.Model` не передаётся в `cursor agent`. Cursor CLI поддерживает `--model` (~30 моделей: composer-1.5/2/2-fast + gpt-5.x codex lineup + sonnet-4 + gpt-5).
- [internal/dispatch/harness/opencode.go:40](../../internal/dispatch/harness/opencode.go) — `opts.Model` не передаётся в `opencode run`. OpenCode CLI поддерживает `-m provider/model` явно.

Router думает что выбрал `composer-2-fast`, харнесс запускается с дефолтом из конфига. Routing-решения не реализуются.

**1.2 Provider-каркас полупустой.** `Provider` interface ([harness.go:21](../../internal/dispatch/harness/harness.go)) задуман как абстракция над rate-limit-aware вендором, но **единственная реализация — `ZAIProvider`** с захардкоженными `glm-5/glm-4.7` и `CheckLimits` возвращающим stub. Anthropic/OpenAI/Cursor/Kimi/Ollama как `Provider` не существуют. Historical note: old `LocalConfig`/`local.go` was removed by the F145 cleanup; current local dispatch code lives in [internal/dispatch/ollama_client.go](../../internal/dispatch/ollama_client.go) and harness/provider files.

**1.3 Routing single-shot, без confidence loop.** `Router.Route` ранжирует модели **upfront** по prior'ам и выбирает одну. `DispatchingInvoker.Fallback` срабатывает **только на ошибках** (packet load fail, routing fail, invoker missing). Если cheap модель ответила, но плохо — никто не эскалирует.

В то же время F144 шипнул `internal/inference/confidence/` с `Status{OK,UNSURE,FAIL}` и стратегиями (`constraint`, `selfcheck`, `nsample`, `composer`). Эти сигналы **не используются** в dispatch-решениях.

## 2. Goals

1. Закрыть silent-bug: cursor/opencode передают выбранную модель в CLI.
2. Расширить provider matrix до 5 реальных вендоров: OpenAI, Anthropic, Cursor, Kimi (Moonshot), Ollama. Каждый — формальный `Provider` impl с `Models()` каталогом и не-stub `CheckLimits()`.
3. Добавить tier-aware routing: `tier_class` label на профилях (`fast | balanced | strong | local`) для разделения cheap/strong моделей.
4. Реализовать `CascadingInvoker`: cheap tier → confidence gate → escalate to strong tier при `UNSURE/FAIL`. Heuristic short-circuit (length/refusal-pattern) минимизирует cost для очевидных провалов.
5. Интегрировать F144 `confidence.Checker` как gate-policy с composer-default (`constraint+selfcheck`).
6. Day-8 acceptance: replay-corpus прогон 20+ prompts с разделением по tier'ам, отчёт `% stayed cheap / % escalated / false-OK rate`.

## 3. Non-Goals

- **Cursor model selection через config** (мимо `--model`). CLI флаг работает — этого достаточно. Раскопки cursor config-format'а — out of scope.
- **Bench-driven profile generator** (auto-bench всех моделей на коде). Tier-mapped seed достаточен для cold-start; bench — отдельный эпик F146 (не в этой работе).
- **Cost-tracking / billing** (USD per inference). Только token-counter и tier-label. Полноценный cost-router — F147 (parking lot).
- **Provider auth refactor** (storage of API keys in vault/keychain). В рамках F145 ключи остаются у harness CLIs (codex/cursor/opencode читают свои env vars сами). SDP `Provider` impls — metadata layer только.
- **Web UI / dashboard** для cascade-метрик. CSV-репорт через replay-corpus достаточен.

## 4. Architecture

### 4.1 Provider sub-package

Новый layout: `internal/dispatch/harness/providers/`.

```
internal/dispatch/harness/
├── harness.go             # Harness/Provider interfaces, Registry (existing)
├── claude.go              # ClaudeHarness (existing)
├── codex.go               # CodexHarness (existing, plumbing OK)
├── cursor.go              # CursorHarness (FIX: --model plumbing)
├── opencode.go            # OpenCodeHarness (FIX: -m provider/model plumbing)
├── zai.go                 # ZAIProvider (existing — оставляем как есть)
└── providers/             # NEW
    ├── providers.go       # Register/Get + ProviderRegistry
    ├── openai.go          # OpenAIProvider (codex lineup)
    ├── anthropic.go       # AnthropicProvider
    ├── cursor.go          # CursorProvider (~30 models via `cursor agent --list-models`)
    ├── kimi.go            # KimiProvider (Moonshot)
    └── ollama.go          # OllamaProvider — replaces LocalConfig
```

Каждый `Provider` impl:
- `Name() string` — канонический id (e.g. `"openai"`, `"anthropic"`)
- `Models() []string` — каталог. Для cursor берётся из `cursor agent --list-models`, кешируется.
- `CheckLimits(ctx) (*Limits, error)` — реальная проверка, см. §4.5.

### 4.2 Tier-class label

`CapabilityProfile` расширяется одним опциональным полем:

```go
type TierClass string

const (
    TierFast     TierClass = "fast"     // cheap, low-latency: composer-2-fast, gpt-5.3-codex-low, qwen2.5-coder, glm-4.7
    TierBalanced TierClass = "balanced" // medium: composer-2, gpt-5.3-codex, sonnet, gpt-5.2
    TierStrong   TierClass = "strong"   // top-tier: gpt-5.3-codex-xhigh, opus, gpt-5.3-codex-spark-preview-xhigh
    TierLocal    TierClass = "local"    // Ollama tier — no API cost, capability ≈ fast
)

type CapabilityProfile struct {
    // existing fields...
    Harness      string                     `json:"harness"`
    Provider     string                     `json:"provider"`
    Model        string                     `json:"model"`
    Capabilities map[string]*CapabilityScore `json:"capabilities"`
    // NEW
    TierClass    TierClass                  `json:"tier_class,omitempty"`
}
```

Router-сторона: `Route(...)` возвращает один winner как раньше (back-compat). Cascade-сторона: `cascade.SelectTiers(profiles, policy)` группирует по `TierClass`, выдаёт ordered tier-chain.

### 4.3 CascadingInvoker

Новый pkg `internal/dispatch/cascade/`.

```go
type Policy struct {
    MaxDepth        int            // default 2 (cheap → strong)
    MaxBudget       time.Duration  // soft-cap latency
    Tiers           []TierClass    // ordered: ["fast", "strong"] по умолчанию
    GateStrategy    confidence.Strategy[string]  // F144 — default composer(constraint+selfcheck)
    Heuristics      ShortCircuitConfig
    EscalateOn      []confidence.Status  // [UNSURE, FAIL]
}

type ShortCircuitConfig struct {
    MinLengthChars       int      // < N → escalate без confidence-call
    RefusalPatterns      []string // regex: "I cannot|I'm unable|sorry, I can't"
    EmptyOrWhitespace    bool     // any non-meaningful → escalate
}

type Invoker struct {
    Router    *dispatch.Router
    Policy    Policy
    Limits    map[string]*harness.Limits
    InvokeFor func(harness string) dispatch.LLMInvoker
    Fallback  dispatch.LLMInvoker  // если cascade полностью fail'ится
}

func (i *Invoker) Invoke(ctx context.Context, dir, agent, prompt string) (Result, error)
```

`Result` несёт metrics: `TierUsed`, `EscalationCount`, `TotalTokens`, `FinalStatus`, `ShortCircuitReason`.

**Алгоритм:**
1. Router ранжирует профили (как сейчас).
2. Cascade фильтрует по `Policy.Tiers` — собирает ordered tier-chain.
3. Для каждого tier:
   - invoke harness с моделью этого tier'а
   - проверить heuristics (length/refusal-pattern) — если срабатывает, escalate
   - иначе вызвать `confidence.Checker` (F144) — если `Status ∈ EscalateOn`, escalate
   - если `OK` → return result
4. Если все tier'ы exhausted → return last result + `EscalationCount=MaxDepth` + `FinalStatus=UNSURE`.

### 4.4 Tier-mapped profile seed

`internal/dispatch/profiles_default.go` (или `.json` embed) — встроенный seed по семействам моделей. Cold-start guarantee: ни одна модель не получает 0.5 prior; tier-mapping даёт осмысленный baseline.

| Tier | Examples | Default `coding/Go` prior |
|---|---|---|
| `fast` | composer-2-fast, gpt-5.3-codex-low, qwen2.5-coder:7b, glm-4.7 | 0.55 |
| `balanced` | composer-2, gpt-5.3-codex, sonnet-4, gpt-5.2 | 0.75 |
| `strong` | gpt-5.3-codex-xhigh, opus-4, gpt-5.3-codex-spark-preview-xhigh | 0.90 |
| `local` | qwen2.5-coder:7b (через ollama) | 0.50 (для low-complexity coding only) |

После bench (отдельный эпик) prior'ы перезаписываются реальными.

### 4.5 LimitsCache (CheckLimits hybrid)

`internal/dispatch/harness/limits_cache.go`:

```go
type LimitsCache struct {
    poller    *time.Ticker  // TTL=30s
    providers map[string]Provider
    cache     map[string]*Limits   // sync.RWMutex protected
    lastHTTP  map[string]*Limits   // populated from response headers
}

// Get returns limits with priority: header-driven cache > poller cache > stub.
func (c *LimitsCache) Get(provider string) *Limits

// UpdateFromHeaders called by harness adapters after each LLM-call.
// Provider can extract `x-ratelimit-remaining-*` headers.
func (c *LimitsCache) UpdateFromHeaders(provider string, hdrs http.Header)
```

Bootstrap flow: на старте poller вызывает `Provider.CheckLimits` для каждого зарегистрированного провайдера. После первого ответа от харнесса обновляем `lastHTTP` cache из response headers (если provider это поддерживает).

### 4.6 Migration: LocalConfig → OllamaProvider

`LocalConfig` в [route.go](../../internal/dispatch/route.go) текущей логикой инжектит local-профиль для `complexity=low + cap=coding`. Переезд:
1. `OllamaProvider` (§4.1) формализует Ollama как Provider с `Models()` (читает `ollama list`).
2. Profile `(opencode, ollama, qwen2.5-coder:7b)` с `tier_class=local` добавляется в `profiles_default.json`.
3. Special-case в `route.go` (`if r.LocalConfig != nil && task.Complexity == "low"`) удаляется. Логика поглощается общим ranker'ом.
4. Backwards: `LocalConfig` тип удаляется. Внутренний codebase — единственный consumer (нет downstream-зависимостей через сериализацию).

Чистый break — не deprecation alias. Один PR, миграция всех call-сайтов в той же WS.

## 5. Workstream Decomposition

### Phase 1 — Provider Layer (6 WS, parallel)

| WS | Title |
|---|---|
| 00-145-01 | providers/ sub-pkg scaffold + register pattern |
| 00-145-02 | OpenAIProvider (codex lineup, headers parsing) |
| 00-145-03 | AnthropicProvider |
| 00-145-04 | CursorProvider (~30 models) |
| 00-145-05 | KimiProvider (Moonshot) |
| 00-145-06 | OllamaProvider (replaces LocalConfig — chunked migration внутри WS) |

### Phase 2 — Routing Integration (3 WS)

| WS | Title |
|---|---|
| 00-145-07 | harness plumbing fix — cursor.go + opencode.go `--model` |
| 00-145-08 | tier_class label + Router scoring integration |
| 00-145-09 | profiles_default.json — tier-mapped seed (~40 entries) |

### Phase 3 — Cascade Layer (3 WS)

| WS | Title |
|---|---|
| 00-145-10 | cascade pkg + CascadingInvoker (composer gate, heuristics, MaxDepth/Budget) |
| 00-145-11 | F144 confidence.Checker injection + escalation policy (UNSURE/FAIL) |
| 00-145-12 | cascade-replay mode in F144 replay corpus + tier_used metrics |

### Phase 4 — Operations (2 WS)

| WS | Title |
|---|---|
| 00-145-13 | LimitsCache (background poller + response-header adapter) |
| 00-145-14 | smoke integration test + Day-8 acceptance demo |

### DAG

```
01 (scaffold)
 ├─→ 02 (OpenAI)
 ├─→ 03 (Anthropic)
 ├─→ 04 (Cursor)
 ├─→ 05 (Kimi)
 └─→ 06 (Ollama)         ─→ 09 (seed needs Ollama profile)
                          ↑
07 (plumbing) — independent of 01..06 (но мерджит после 04 для cursor profile testing)
08 (tier_class)         ─→ 09 (seed)
                            ↓
{02..06,08,09} ──────→ 10 (cascade) ─→ 11 (confidence) ─→ 12 (replay)
                                                            ↓
13 (limits) — параллельно с Phase 3                         ↓
                                                            ↓
                                                          14 (smoke + demo)
```

**Critical path:** 01 → {02,06} → 09 → 10 → 11 → 12 → 14 (≈ 7 sequential steps).

**Parallel start (после 01):** 02, 03, 04, 05, 06, 07, 08, 13 — 8 WS могут идти одновременно.

## 6. Acceptance Criteria (epic-level)

- [ ] 5 новых Provider impls (OpenAI, Anthropic, Cursor, Kimi, Ollama) с `Models()` non-empty + `CheckLimits()` non-stub (хотя бы один реальный сигнал, не `Source: "unavailable"`).
- [ ] cursor + opencode harness регрессия: e2e тест `Spawn(opts.Model="X")` подтверждает что CLI получил `--model X`.
- [ ] `tier_class` label корректно фильтрует cascade tier-chain (unit test).
- [ ] `CascadingInvoker.Invoke` эскалирует на `Status=UNSURE` и `Status=FAIL` (mock confidence.Checker, replay).
- [ ] Heuristic short-circuit срабатывает для: пустого ответа, ответа <50 символов, refusal-pattern (regex `I cannot|I'm unable|sorry, I can't`).
- [ ] F144 replay-corpus в cascade-mode выдаёт CSV-отчёт: `prompt_id, tier_used, escalation_count, final_status, tokens, latency_ms`.
- [ ] Day-8 demo: 20+ prompts, ≥40% staying-on-cheap для simple-coding-easy, ≥80% escalation для complex-coding-hard.
- [ ] LimitsCache не блокирует hot-path (assertion на p99 latency Route < 5ms vs current).
- [ ] Smoke integration job (`workflow_dispatch`) зелёный для всех 4 harness CLIs.
- [ ] `LocalConfig` тип удалён, всё опирается на `OllamaProvider`.
- [ ] go build, go test, go vet чистые на всём репо.

## 7. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Cursor `--model` флаг сломается в новой версии CLI | smoke integration ловит регрессию; CapabilityProfile поддерживает override на single-model fallback |
| OpenCode `-m provider/model` формат меняется | то же; интеграционный тест assert'ит конкретный CLI invocation |
| Кеш `LimitsCache` отдаёт stale данные → Router выбирает over-limit profile | TTL=30s; при `429` от harness — invalidate cache + retry |
| F144 `confidence.Checker` добавляет +1 LLM-call на каждый cheap-tier результат → cost overhead | constraint-only override per tier (zero-cost gate); composer применяется только если constraint passed |
| Profile seed (~40 entries) разрастается при добавлении новых моделей | seed генерируется build-time из tier-mapping table; ручных правок минимум |
| Heuristic short-circuit ловит false positives (короткий валидный ответ типа "yes") | `MinLengthChars` per agent-type override; default conservatively low (50) |
| Cascade depth=2 не хватает для очень сложных prompts | `MaxDepth` configurable; verify-агенты могут получить depth=3 через policy override |

## 8. Security & Operations

**Credentials.** Harness CLIs (codex/cursor/opencode/claude) хранят свои API keys в env vars или config-files как и раньше. SDP Provider impls — metadata-only: не читают и не передают credentials. `KimiProvider` — единственный кейс где может потребоваться `KIMI_API_KEY` в env (если CLI его не подхватывает сам); явно документируем.

**CI gates.**
- Unit + go vet + cascade-replay assertion → on PR (быстрые, deterministic).
- Smoke integration (real CLI invocations) → `workflow_dispatch` only, требует `OPENAI_API_KEY`/`ANTHROPIC_API_KEY`/etc. в repo secrets. Не блокирует мерж.

**Observability.** `CascadingInvoker.Invoke` логгирует через `slog` с полями: `tier_used`, `escalation_count`, `total_tokens`, `final_status`, `short_circuit_reason`. Совместим с F140 telemetry pipeline (если включён).

## 9. Future Work (out of scope)

- F146: Bench-driven profile generator (`cmd/sdp-bench-profiles`) — реальные prior'ы из measured task-pass-rate.
- F147: Cost-aware routing (USD per inference, budget guardrails).
- F148: Multi-region routing (latency-aware harness selection).

## 10. References

- F133 design: [PR #127](https://github.com/fall-out-bug/sdp_lab/pull/127) — Local model dispatch via Ollama
- F144 design: [docs/plans/2026-04-26-f144-inference-confidence-design.md](2026-04-26-f144-inference-confidence-design.md), [PR #131](https://github.com/fall-out-bug/sdp_lab/pull/131) merged
- Existing dispatch: [internal/dispatch/route.go](../../internal/dispatch/route.go), [invoker.go](../../internal/dispatch/invoker.go)
- F127 multi-harness modernization: [docs/plans/2026-04-16-f127-multi-harness-modernization-design.md](2026-04-16-f127-multi-harness-modernization-design.md)
