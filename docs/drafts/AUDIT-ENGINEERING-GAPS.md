# Аудит: Инженерные лакуны SDP

> **Дата:** 2026-02-25
> **Формат:** Addendum к AUDIT-ZHUKOV-2026.md
> **Фокус:** Что должен увидеть человек, который зайдёт в репо и решит — "этот чувак знает, что делает" или "это ChatGPT нагенерил"

---

## TL;DR

Репо посылает два сигнала одновременно:
- **"Этот человек ДУМАЕТ как архитектор"** — evidence envelope, in-toto predicate, boundary compliance, hash chain, 9 CI workflows
- **"Это написал агент без присмотра"** — mock-код в продакшне, проглоченные ошибки, race conditions в самом evidence layer, ноль промпт-инженерии в промптах

Проблема не в том, что 99% написали агенты. Проблема в том, что **видно**, где ты не review'ил. Умный человек зайдёт в `internal/evidence/emitter.go`, увидит `go func() { ... return }` и подумает: "evidence layer, который ТЕРЯЕТ evidence? Серьёзно?"

Ниже — 7 конкретных лакун, ранжированных по тому, насколько каждая бьёт по credibility.

---

## Лакуна 1: Mock-код в продакшне (Credibility Kill)

**Файл:** `sdp-plugin/internal/executor/retry.go:64-79`

```go
// executeWorkstreamMock is a mock executor for testing
// In production, this would call the actual workstream execution logic
func (e *Executor) executeWorkstreamMock(ctx context.Context, wsID string, attemptCount int) error {
    // Mock: 00-054-02 fails on first attempt, succeeds on retry
    if wsID == "00-054-02" {
        if attemptCount == 1 {
            return fmt.Errorf("mock execution failure for %s", wsID)
        }
        return nil
    }
    return nil
}
```

Это ПРОДАКШН-КОД с хардкоженным workstream ID `00-054-02`. Комментарий буквально говорит "in production, this would call the actual workstream execution logic." То есть execution engine — **ненастоящий**. Ретраи работают, но ретраят mock.

**Почему это убивает credibility:** Человек, который зашёл посмотреть, как SDP выполняет workstreams, находит заглушку. Это главный вопрос для любого оценщика: "а оно вообще что-то реальное делает?"

**Фикс:** Вынести mock в `*_test.go`. В продакшн-коде — интерфейс `WorkstreamRunner` с реальной реализацией (вызов `@build` через CLI или субпроцесс). Даже если реализация тривиальная — она должна быть РЕАЛЬНОЙ, не mock.

**Приоритет:** P0. Удалить за 1 час.

---

## Лакуна 2: Evidence Layer теряет evidence (Irony Kill)

**Файл:** `sdp-plugin/internal/evidence/emitter.go:12-28`

```go
// Emit appends an event to the evidence log. Non-blocking; errors are ignored.
func Emit(ev *Event) {
    // ...
    go func() {
        if err := emitSync(&ev2); err != nil {
            return  // ← ошибка проглочена
        }
    }()
}
```

Evidence layer — центральная идея SDP: "prove what agents did." И центральная функция этого layer'а МОЛЧА ТЕРЯЕТ данные. Нет логирования. Нет метрики. Нет канала ошибок. Если диск полный, если config сломан, если path невалидный — evidence просто исчезает. И никто не узнает.

**Почему это убивает credibility:** Это как firewall, который тихо отключается при перегрузке. Весь value proposition SDP — "proof, not vibes." И proof-система сама built on vibes.

**Фикс:**
```go
func Emit(ev *Event) {
    // ...
    go func() {
        if err := emitSync(&ev2); err != nil {
            slog.Error("evidence emission failed", "event_id", ev2.ID, "error", err)
            emitErrorCount.Add(1) // prometheus/expvar counter
        }
    }()
}
```

Минимум — structured logging. Лучше — error channel + metrics. Best — `EmitSync` по умолчанию, async только с explicit opt-in.

**Приоритет:** P0. Это идеологический баг.

---

## Лакуна 3: Race condition в hash chain (Integrity Kill)

**Файл:** `sdp-plugin/internal/evidence/writer.go:23-38`

```go
func NewWriter(path string) (*Writer, error) {
    // ...
    w := &Writer{path: path, lastHash: genesisHash}
    if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
        lastLine := lastLineBytes(b)
        if len(lastLine) > 0 {
            w.lastHash = hashLine(lastLine)
        }
    }
    return w, nil
}
```

`sync.Mutex` защищает от concurrent writes в рамках одного процесса. Но `NewWriter` можно вызвать из двух процессов одновременно (два CI job'а, два терминала). Оба прочитают один `lastHash`, оба запишут event с одним `PrevHash`, и hash chain сломается. А `emitSync` создаёт NEW Writer на КАЖДЫЙ вызов (`w, err := NewWriter(path)`) — то есть mutex бесполезен для межпроцессной защиты.

**Почему это убивает credibility:** Hash chain — это криптографическая гарантия. Если она ломается при двух параллельных записях, это не гарантия. Для человека из Sigstore/in-toto community это красный флаг.

**Фикс:** File lock (`flock`/`fcntl`), или singleton Writer с lifecycle management, или advisory lock через `.sdp/log/events.lock`.

**Приоритет:** P0. Хеш-цепочка — это фундамент.

---

## Лакуна 4: Промпты без промпт-инженерии (AI Engineering Kill)

Это самая большая лакуна для позиционирования как **AI-инженера** (а не просто software-инженера).

**Количественная картина по 23 SKILL.md:**

| Техника | Присутствие | Ожидание |
|---------|------------|----------|
| Chain-of-thought | 0 из 23 | В каждом reasoning-heavy скилле |
| Few-shot examples | 1 из 23 (@tdd — один пример) | 3-5 примеров на скилл |
| Output validation schema | 2 из 23 (partial) | В каждом structured output |
| Hallucination detection | 0 из 23 | В review, build, reality |
| Confidence thresholds | 0 из 23 | В decision-making скиллах |
| Retry/fallback strategies | 0 из 23 | В каждом execution-скилле |
| Token budget management | 0 из 23 | В long-context скиллах |
| Model-agnostic patterns | ~5 из 23 (заявлено) | Все |

**Конкретный пример — @review:**

```
You are the {ROLE} expert for feature F{XX}. Review your domain. 
For each finding: bd create --silent --labels "review-finding,F{XX},round-1,{role}" --priority={0-3} --type=bug. 
Output: FINDINGS_CREATED: id1 id2. Rule: PASS if all P2/P3; FAIL if any P0/P1. 
Output verdict: PASS or FAIL
```

Это instruction list, не prompt. Нет:
- **Reasoning structure:** "First, analyze X. Then, evaluate Y. Finally, decide Z."
- **Evaluation criteria:** Что значит P0 vs P1 для SRE vs для Security? Нет rubric.
- **Output format validation:** Если LLM выведет "APPROVED" вместо "PASS"?
- **Adversarial dynamics:** 7 агентов работают параллельно, но не взаимодействуют. Нет disagreement tracking, нет consensus, нет devil's advocate. Это "7 монологов, склеенных в отчёт."
- **Few-shot:** Ни одного примера "вот хороший finding, вот плохой finding."

**Почему это убивает credibility:** Ты позиционируешь себя как AI-визионера, строящего протокол для AI-агентов. И промпты — это самый видимый артефакт твоей AI-экспертизы. Любой, кто откроет `prompts/skills/review/SKILL.md`, увидит инструкцию, а не инженерию. Карпати-уровень — это когда ты понимаешь failure modes LLM и ПРОЕКТИРУЕШЬ промпты вокруг них.

**Фикс (для 3-5 ключевых скиллов):**
- Добавить structured reasoning ("Analyze in this order: 1... 2... 3...")
- Добавить few-shot examples (2-3 на скилл)
- Добавить output validation (JSON schema reference в промпте)
- Для @review: добавить adversarial round ("Agent X challenges Agent Y's findings")
- Добавить confidence: "Rate your confidence 1-5. If < 3, flag for human review."

**Приоритет:** P1. Это то, что отделяет "software engineer who uses AI" от "AI engineer."

---

## Лакуна 5: Ноль Evaluation Framework (Science Kill)

В репо 36K LOC Go, 55K LOC тестов, 23 skill'а, 8 JSON Schema. И НИ ОДНОГО файла для оценки того, работают ли промпты.

**Что отсутствует:**

1. **Evals directory.** `evals/` — папка с test cases для промптов. "Дай этот промпт модели, ожидай такой output, измерь quality." У OpenAI есть evals. У Anthropic есть evals. У SDP — ноль.

2. **Prompt versioning.** Skills имеют version numbers в frontmatter (`version: 14.0.0` для @review). Но нет A/B comparison. Нет "v13 давала 80% quality, v14 даёт 85%." Версии есть, метрик к версиям — нет.

3. **Benchmark data.** Сколько раз @review поймала реальный баг? Сколько раз @build уложилась в scope? Сколько раз evidence envelope был валидным с первого раза? Нет данных.

4. **Failure taxonomy для ПРОМПТОВ.** Есть `schema/failure-taxonomy.schema.json` для кода. Нет failure taxonomy для prompt failures: "модель вышла за scope", "модель не следовала format", "модель hallucinated CLI command."

**Почему это убивает credibility:** AI engineering — это science. Science без measurement — это не science. Ты учишь в МГУ. Студент спросит: "а как вы измеряете quality ваших промптов?" И правильный ответ — не "мы eyeball'им."

**Фикс:**
```
evals/
├── review/
│   ├── test_cases.jsonl       # input → expected output
│   ├── rubric.md              # scoring criteria
│   └── results/
│       ├── v13.0.0.json       # scores for v13
│       └── v14.0.0.json       # scores for v14
├── build/
│   └── ...
└── README.md                  # how to run evals
```

Даже 10 test cases для 3 ключевых скиллов — это уже signal: "этот человек измеряет, а не надеется."

**Приоритет:** P1. Это investment в credibility и в реальное качество.

---

## Лакуна 6: Software Engineering gaps (Production Kill)

Эти вещи видны любому senior Go-инженеру за 10 минут чтения кода:

### 6a. Structured logging — отсутствует

```go
// cmd/sdp/main.go — типичный паттерн
fmt.Fprintf(os.Stderr, "Error: %v\n", err)
```

Один пакет (`internal/orchestrator`) использует `slog`. Остальные 29 — `fmt.Printf/Fprintf`. Для CLI это терпимо. Для evidence system, которая позиционируется как audit trail — нет. Logging IS evidence.

### 6b. Context propagation — обрывается

```go
// internal/executor/execution.go — принимает context
func (e *Executor) Execute(ctx context.Context, ...) { ... }

// internal/verify/verifier.go — создаёт НОВЫЙ context (!)
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
```

Context принимается на входе и игнорируется внутри. Cancel от верхнего уровня не пробрасывается. Graceful shutdown невозможен.

### 6c. Hardcoded values

```go
time.Sleep(100 * time.Millisecond)  // retry.go:57
context.WithTimeout(context.Background(), 60*time.Second)  // verifier.go:76
```

Нет конфигурируемых таймаутов. Для CI environments с разной latency — проблема.

### 6d. File open/close на каждый write

```go
// evidence/writer.go:84-103 — appendToFile
func appendToFile(path string, data []byte) error {
    f, err := os.OpenFile(path, ...)
    // write, sync, close — на КАЖДЫЙ event
}
```

При активном evidence emission (десятки events за билд) — это десятки open/sync/close. Для evidence layer с hash chain — buffered writer с periodic flush был бы правильнее.

**Фикс:** Один рефакторинг-пас с чеклистом: slog everywhere, context propagation, config-driven timeouts, buffered writer. Это weekend project, не month project.

**Приоритет:** P2 (не блокирует adoption, но блокирует respect от Go engineers).

---

## Лакуна 7: in-toto conformance gaps (Standards Kill)

У тебя `coding-workflow-predicate.schema.json` ссылается на in-toto Statement v0.1. Текущая версия — **v1.1**. Проблемы:

### 7a. Statement version

```json
"_type": "https://in-toto.io/Statement/v0.1"
```

v0.1 — deprecated. Текущий стандарт — v1 / v1.1. In-toto guidelines (`docs/new_predicate_guidelines.md`) рекомендуют v1.

### 7b. Naming convention

in-toto рекомендует `lowerCamelCase` для field names. У тебя:
- `issue_id` (snake_case) — должно быть `issueId`
- `risk_class` (snake_case) — должно быть `riskClass`
- `changed_files` — должно быть `changedFiles`
- `hash_prev` — должно быть `hashPrev`

Evidence envelope (legacy) использует snake_case. Coding workflow predicate (in-toto) тоже использует snake_case. Для регистрации в in-toto registry нужен lowerCamelCase.

### 7c. Predicate type URL

```json
"$id": "https://sdp.dev/attestation/coding-workflow/v1"
```

Домен `sdp.dev` — он зарегистрирован? Резолвится? in-toto guidelines требуют, чтобы predicate type URL либо резолвился, либо использовался namespace `https://in-toto.io/attestation/`.

### 7d. Missing `digestSet` pattern

SLSA/in-toto используют `digestSet` для subject:
```json
"digest": { "sha256": "abc123" }
```

У тебя `coverage.value` — число, не digest. `hash` — строка, не digestSet. Для interoperability с SLSA tooling (cosign verify-attestation) нужно следовать формату.

**Почему это важно:** Ты хочешь зарегистрировать predicate type в in-toto registry. Reviewers проверят conformance. Текущее состояние не пройдёт review без правок.

**Фикс:** 
1. Обновить Statement version до v1
2. Переименовать fields в lowerCamelCase (или обосновать snake_case в PR description)
3. Зарегистрировать/зарезервировать `sdp.dev` или использовать github URL
4. Добавить digestSet pattern для crypto fields

**Приоритет:** P1 (блокирует путь к стандартизации).

---

## Матрица: Что видит каждый тип посетителя

| Посетитель | Что видит хорошего | Что видит плохого | Вердикт |
|-----------|-------------------|-------------------|---------|
| **Go engineer** | 80%+ coverage, clean packages, CI | Mock в production, no slog, race conditions | "Агент писал, а хозяин не проверял" |
| **AI/ML engineer** | Evidence concept, boundary compliance | Промпты-инструкции, нет evals, нет CoT | "Software engineer, не AI engineer" |
| **Security engineer** | Hash chain, boundary enforcement | Race condition в writer, silent error drop | "Концепт есть, implementation не trusted" |
| **in-toto/SLSA person** | Predicate schema, Sigstore mention | v0.1, snake_case, unresolvable URL | "Не дочитал spec до конца" |
| **CTO evaluating for adoption** | Comprehensive protocol, good docs | 16 stars, 0 users, overengineered | "Impressive solo project, not ready for my team" |

---

## Top 10 фиксов по impact/effort

| # | Фикс | Impact | Effort | Credibility signal |
|---|------|--------|--------|-------------------|
| 1 | Удалить mock из retry.go, добавить реальный interface | 🔴 Critical | 2h | "Production code is real" |
| 2 | Добавить slog.Error в evidence Emit() | 🔴 Critical | 30min | "Evidence never lost silently" |
| 3 | File lock для evidence writer | 🔴 Critical | 4h | "Hash chain is actually reliable" |
| 4 | Создать `evals/` с 10 test cases для @review | 🟡 High | 1 day | "I measure my prompts" |
| 5 | Добавить few-shot examples в @review, @build | 🟡 High | 4h | "I understand prompt engineering" |
| 6 | in-toto v1 + lowerCamelCase | 🟡 High | 1 day | "I respect the standard" |
| 7 | Structured logging (slog) throughout | 🟢 Medium | 1 day | "Production-ready code" |
| 8 | Context propagation fix | 🟢 Medium | 4h | "Proper Go engineering" |
| 9 | Adversarial round в @review | 🟢 Medium | 4h | "Multi-agent is not multi-monologue" |
| 10 | Hero GIF в README | 🟢 Medium | 2h | "I care about users" |

**Total estimated effort: ~5 working days.**

Пять дней работы — и репо перестанет посылать сигнал "агент написал, хозяин не проверил" и начнёт посылать "архитектор спроектировал, агент реализовал, архитектор отрецензировал."

---

## Мета-наблюдение

Все 7 лакун имеют общий корень: **ты review'ишь ДИЗАЙН, но не review'ишь ИМПЛЕМЕНТАЦИЮ.** Архитектура evidence envelope — блестящая. Hash chain — правильная идея. in-toto predicate — правильный выбор. 9-section envelope — оригинальный и полезный дизайн.

Но на уровне строчки кода — mock в production, silent errors, race conditions, hardcoded timeouts. Это типичный паттерн "визионер + AI agents": vision отличный, execution — на автопилоте.

**Решение:** Для SDP (как для визитной карточки) — ты ЛИЧНО должен прочитать каждый файл в `internal/evidence/` и `internal/executor/`. Не весь репо. Но critical path — своими глазами. Это 10-15 файлов. Это вечер.

Когда ты этот вечер проведёшь, ты сможешь честно сказать: "Я спроектировал протокол, агенты реализовали, и я лично отрецензировал каждую строку critical path." Это НАМНОГО сильнее, чем "99% написано агентами."

---

*"В архитектуре ты — визионер. В коде — ты отсутствующий reviewer. Пять дней фиксов превратят второе в первое."*
