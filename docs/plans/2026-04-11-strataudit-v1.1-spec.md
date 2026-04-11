# StratAudit v1.1 — Спецификация доработок

**Дата:** 2026-04-11
**Источник:** LLM Council (2 раунда, 6 моделей) + тестовый прогон на документах УБРиР
**Цель:** Рабочий end-to-end пайплайн с покрытием >50% на реальных данных

---

## Контекст проблемы

Тестовый прогон на документах УБРиР (2 strategy, 1 architecture, 3 implementation) показал:
- **110 сущностей** извлечено (76 strategy, 30 architecture, 4 implementation)
- **1 трассировка** из 3 кандидатов — покрытие 0-25%
- **141 находка** (105 gaps, 32 orphans, 3 coverage, 1 weak_link)

Корневые причины: (1) link stage не использует LLM-верификацию, (2) reasoning-модели не поддерживаются, (3) отчёт не содержит полную модель.

---

## P0 — Критические фиксы (блокируют следующий прогон)

### FIX-01: Поддержка reasoning-моделей

**Файл:** `internal/strataudit/llmclient.go:134-149`
**Приоритет:** P0
**Трудоёмкость:** 1-2ч

**Проблема:** Структура `result.Choices[0].Message` десериализует только поле `Content`. Reasoning-модели (deepseek-v3.2-speciale, deepseek-r1) возвращают `content: null`, а текст — в поле `reasoning`. Пайплайн молча получает пустую строку.

**Текущий код:**
```go
var result struct {
    Choices []struct {
        Message struct{ Content string } `json:"message"`
    } `json:"choices"`
}
content := result.Choices[0].Message.Content
```

**Требование:**
1. Добавить поле `Reasoning string` в структуру message
2. Fallback chain: `content` → `reasoning` → error
3. Проверка `len(choices) == 0` — graceful error, не panic
4. Конфиг-параметр `llm.reasoning_fallback: true` (default: true)

**Приёмка:** Пайплайн работает с `deepseek/deepseek-v3.2-speciale` и `google/gemini-2.5-flash` без ошибок.

---

### FIX-02: Абсолютный путь для outputDir

**Файл:** `cmd/sdp-strataudit/main.go:91`, `internal/strataudit/pipeline.go:64`
**Приоритет:** P0
**Трудоёмкость:** 0.5ч

**Проблема:** `cfg.Output.Dir` (default: `.strataudit`) резолвится от CWD, а не от `--dir`. DB создаётся в `filepath.Join(*dir, cfg.Output.Dir)`, но отчёты пишутся в `cfg.Output.Dir` от CWD. Split-brain: DB в одном месте, отчёты — в другом.

**Текущий код (main.go:91):**
```go
dbPath := filepath.Join(*dir, cfg.Output.Dir, "strataudit.db")
```

**Текущий код (pipeline.go:64):**
```go
outputDir := cfg.Output.Dir  // голый относительный путь
```

**Требование:**
1. В `main.go` после загрузки конфига: `cfg.Output.Dir = filepath.Join(*dir, cfg.Output.Dir)` + `filepath.Abs()`
2. Или: передать `baseDir` в `RunPipeline()` и резолвить там
3. CLI выводит resolved путь: `"Output: /abs/path/.strataudit"`

**Приёмка:** `sdp-strataudit run --dir /tmp/test` создаёт DB и отчёты в `/tmp/test/.strataudit/`.

---

### FIX-03: LLM-верификация в link stage

**Файл:** `internal/strataudit/link.go:158-185`
**Приоритет:** P0
**Трудоёмкость:** 3-4ч

**Проблема:** `createTraces()` получает `*LLMClient` как аргумент, но никогда его не вызывает. Трассировки создаются чисто по embedding similarity >= `trace_confidence` (default: 0.6). При threshold 0.5 для candidates и 0.6 для traces — кандидат с similarity 0.67 проходит автоматически, без LLM-проверки контекста.

**Текущий код (link.go:158-185):**
```go
func createTraces(ctx, cfg, llm *LLMClient, candidates, lowerLevel, upperLevel) []model.Trace {
    for _, c := range candidates {
        confidence := c.sim  // просто similarity
        justification := fmt.Sprintf("Embedding similarity: %.2f", c.sim)
        if confidence >= threshold {
            traces = append(...)  // без LLM вызова
        }
    }
}
```

**Требование:**
1. Для candidates с similarity в диапазоне `[trace_confidence, 0.85)` — LLM-верификация
2. Для candidates с similarity >= 0.85 — auto-verified (текущее поведение)
3. LLM verification prompt: "Is there a strategic relationship between [entity A] and [entity B]?"
4. Ответ: `{relation, confidence, justification}`
5. Сохранять `verified: true/false` в trace_candidates
6. Логировать: `"auto-verified N traces, LLM-verified M traces"`

**Приёмка:** При проставлении `trace_confidence: 0.6` кандидаты 0.60-0.84 проходят LLM-проверку. Покрытие >10%.

---

## P1 — Перед следующим прогоном

### FIX-04: Полный JSON-экспорт

**Файл:** `internal/strataudit/report/json.go:11-17`, `report_builder.go:12-70`
**Приоритет:** P1
**Трудоёмкость:** 2-3ч

**Проблема:** `AuditReport` не содержит `entities` и `traces`. `BuildReport()` загружает только levels, findings, coverage. Невозможно восстановить полную модель из отчёта или отладить трассировку.

**Требование:**
1. Добавить в `AuditReport`:
   ```go
   Entities []EntityReport `json:"entities,omitempty"`
   Traces   []TraceReport  `json:"traces,omitempty"`
   ```
2. `EntityReport`: id, type, title, description, level_id, document_id
3. `TraceReport`: id, source_entity_id, target_entity_id, relation, confidence, justification
4. `BuildReport()` загружает entities и traces из store
5. JSON и HTML отчёты используют одну модель

**Приёмка:** `report.json` содержит `entities: [...]` и `traces: [...]`.

---

### FIX-05: Русификация findings

**Файл:** `internal/strataudit/analyze.go:52-61, 100-101, 126-127, 159-160`
**Приоритет:** P1
**Трудоёмкость:** 1-2ч

**Проблема:** Все шаблоны findings на английском при русскоязычных документах:
- `"Gap: %q has no support from %s"` → `"Разрыв: %q не имеет поддержки от уровня %s"`
- `"Orphan: %q has no link to %s"` → `"Сирота: %q не связана с уровнем %s"`
- `"Weak link: %.0f%% confidence to %q"` → `"Слабая связь: %.0f%% уверенности к %q"`

**Требование:**
1. Конфиг-параметр `output.lang: "ru"` (default: "ru")
2. Шаблоны локализуются через map:
   ```go
   var findingTemplates = map[string]map[string]struct{ Title, Desc string }{
       "gap":    {"ru": {"Разрыв: %q не имеет поддержки от уровня %s", "..."}, "en": {"Gap: ...", "..."}},
       "orphan": {"ru": {"Сирота: ...", "..."}, ...},
   }
   ```
3. HTML report: кнопки/заголовки тоже локализуются
4. Entity types в отчёте: `goal` → `цель`, `objective` → `задача`, `initiative` → `инициатива`

**Риск (Critic):** Downstream regex/filters могут зависеть от английских ключевых слов. Решение: `finding.type` остаётся английским (machine-readable), локализуется только `title` и `description`.

**Приёмка:** `report.json` содержит `"title": "Разрыв: \"Платежный хаб\" не имеет поддержки от уровня architecture"`.

---

### FIX-08: Нативная конвертация документов (Extractor interface)

**Файл:** `internal/strataudit/ingest.go`, новый `internal/strataudit/extractor.go`, `extract_pptx.go`
**Приоритет:** P1 (повышен с P2 по результатам ревью)
**Трудоёмкость:** 4-6ч (урезанный scope — только interface + PPTX)

**Проблема:** Конвертация документов — полностью вне пайплайна. Пользователь вручную запускает Python-скрипт (`/tmp/strataudit-convert/convert.py`) для конвертации `.pptx`, `.doc` и других форматов в `.txt`. Только после этого запускает StratAudit. Это violates end-to-end pipeline principle.

Go-пайплайн поддерживает только `.txt/.md/.pdf/.docx`. Остальные форматы (`.pptx`, `.doc`, `.xls`, `.xlsx`, `.rtf`, `.odt`) — молча пропускаются через `isSupportedExt()`.

**Требование (вариант B — interface abstraction, council recommendation):**

1. **Extractor interface** (`extractor.go`):
   ```go
   type Extractor interface {
       CanHandle(ext string) bool
       Extract(ctx context.Context, path string, data []byte) (string, error)
   }
   ```

2. **Go-реализации** (существующие, обёрнутые в interface):
   - `TextExtractor`: `.txt`, `.md`, `.markdown`
   - `PDFExtractor`: `.pdf` (ledongthuc/pdf)
   - `DOCXExtractor`: `.docx` (ZIP/XML парсинг)

3. **Bridge-реализация** — вызов внешнего конвертера:
   - `BridgeExtractor`: вызывает `exec.Command("textutil", ...)` на macOS или `libreoffice --headless` на Linux
   - Поддержка: `.pptx`, `.doc`, `.rtf`, `.xls`, `.xlsx`
   - Sanitize filename от command injection (Critic warning из council)
   - Fallback: если внешний инструмент не найден — warning + skip, не fatal

4. **Конфиг:**
   ```yaml
   extractors:
     external_command: "textutil"  # или "libreoffice"
     use_external: ["pptx", "doc", "rtf", "xls", "xlsx"]
   ```

5. **Интеграция:** `ingest.go` использует `ExtractorRegistry` вместо прямого switch в `extractText()`

6. **Логирование:** `"Extracted %s (%d bytes) via %s extractor"` — видно, какой экстрактор сработал

**Приёмка:**
- `sdp-strataudit run --dir /path/with/pptx` — конвертирует и извлекает текст без Python-скриптов
- `report.json` содержит entities из PPTX-документов
- Если `textutil` не установлен — warning, не fatal

---

## P2 — Следующая итерация

### FIX-06: Полный набор документов

**Приоритет:** P2
**Трудоёмкость:** 2ч

**Проблема:** Тест использовал 3 implementation-документа (736 строк) из 50+ доступных (640K строк). Недостаточно данных для трассировки.

**Требование:**
1. Использовать все 50 implementation-документов
2. При 640K строк — увеличить `chunk_token_limit` или добавить budget control (max chunks per doc)
3. Контроль стоимости: `max_total_chunks` в конфиге
4. Предупреждение при >1000 чанков: `"Large corpus: N chunks estimated. Cost: ~$M. Continue? [y/N]"`

**Приёмка:** Прогон на 50+ документах завершается за <15 мин с покрытием >10%.

---

### FIX-07: Mermaid-схема трассировки в HTML

**Файл:** `internal/strataudit/report/html.go`
**Приоритет:** P2
**Трудоёмкость:** 4-6ч
**Зависимость:** FIX-04 (нужны traces в модели)

**Проблема:** HTML-отчёт — только flat-список findings. Нет визуализации графа связей между уровнями.

**Требование:**
1. Mermaid DAG: узлы по уровням, рёбра по traces (цвет по confidence)
2. Фильтр: только connected nodes + 1-hop neighbors (не все 110 isolated)
3. Summary выше графа: `"76 strategy | 30 architecture | 4 implementation | 1 trace"`
4. Интерактив: клик по узлу → подсветить traces

**Приёмка:** В HTML-отчёте есть `<div class="mermaid">` с рабочим графом.

---

## P3 — Отложено

### FIX-09: Pre-extraction фильтрация

**Приоритет:** P3 (DROP per council)
**Трудоёмкость:** 10-12ч

**Council verdict:** Premature optimization. Philosopher: "Low coverage may be ground truth." Сначала отладить link stage (FIX-03), потом оценивать необходимость.

Если потребуется: двухэтапный pipeline — deterministic document reduction (headings, bullet lists, tables, boilerplate stripping) → LLM extraction только на отобранных фрагментах.

---

## Новые проблемы (Architect N1, N2)

### N1: Link stage контракт расходится с реализацией

**Файл:** `internal/strataudit/link.go:22`
**Приоритет:** P0 (устраняется FIX-03)

Комментарий: `"optionally verifies with LLM"` — но LLM никогда не вызывается. После FIX-03 контракт будет соответствовать коду.

### N2: AuditReport смешивает данные и представление

**Файл:** `internal/strataudit/report/json.go:11-17`, `report/html.go`
**Приоритет:** P2

`AuditReport` — одновременно JSON-модель и HTML-backing. Из-за этого модель усечена (нет entities/traces —不方便 для HTML cards). После FIX-04 проблема частично решается. Полное решение: разделить `AuditSnapshot` (полные данные) и `AuditPresentation` (для рендеринга).

---

## Порядок реализации

```
FIX-01 (reasoning fallback)  ──┐
FIX-02 (outputDir)            ──┤─ P0: ship now (~4h)
FIX-03 (LLM link verify)      ──┘
         │
FIX-04 (JSON entities+traces) ──┐
FIX-05 (русификация)           ──┤─ P1: before next run (~8h)
FIX-08 (Extractor interface)   ──┘
         │
FIX-06 (полный набор docs)    ─── P2: validate pipeline
FIX-07 (Mermaid diagram)      ─── P2: after FIX-04
```

**Итого P0+P1:** ~12h engineering time.
