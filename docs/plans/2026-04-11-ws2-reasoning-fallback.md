# WS-2: FIX-01 — Reasoning fallback в LLM клиенте

**Статус:** APPROVED (Council R1+R2 consensus)
**Приоритет:** P0 (v1.0.1, Slice 1)
**Трудоёмкость:** 1-2ч
**Зависимости:** нет

## Проблема
`llmclient.go:134-148` — response struct десериализует только `Content`. Reasoning-модели (deepseek-v3.2-speciale, deepseek-r1) возвращают `content: null`, текст — в `reasoning`. Пайплайн получает пустую строку.

## Файлы
- `internal/strataudit/llmclient.go:134-148` — response struct + content extraction
- `internal/strataudit/config.go:33-47` — LLMConfig (добавить ReasoningFallback)
- `internal/strataudit/llmclient_test.go` — тесты

## Изменения

### 1. Response struct (llmclient.go:134-143) — Council correction: *string для обоих полей
```go
// Было:
Message struct{ Content string } `json:"message"`

// Стало (Council: оба поля *string — JSON null возможен для обоих):
Message struct {
    Content   *string `json:"content"`
    Reasoning *string `json:"reasoning"`
} `json:"message"`
```

### 2. Fallback chain (llmclient.go:148)
```go
// Было:
content := result.Choices[0].Message.Content

// Стало:
if len(result.Choices) == 0 {
    return nil, fmt.Errorf("llm: no choices in response")
}
content := ""
if result.Choices[0].Message.Content != nil {
    content = *result.Choices[0].Message.Content
}
// Council: минимальная длина 50 символов перед fallback
if content == "" && result.Choices[0].Message.Reasoning != nil {
    r := *result.Choices[0].Message.Reasoning
    if len(r) >= 50 {
        content = extractFinalAnswer(r)
    }
}
// Council: slog.Warn при fallback, не Info
if content == "" {
    return nil, fmt.Errorf("llm: empty content and reasoning in response")
}
```

### 3. extractFinalAnswer() — Council correction: последний непустой параграф, не "80%"
- Ищет `<answer>...</answer>` блок в reasoning — берёт содержимое
- Если тегов нет — берёт **последний непустой параграф** (Council: не "80% текста" — это fragile)
- Council: проверить что `<answer>` extraction не создаёт injection surface
- Sanitize: убрать trailing whitespace

### 4. Конфиг (config.go)
- Добавить `ReasoningFallback bool` в LLMConfig
- Default: `true`
- YAML: `llm.reasoning_fallback: true`

### 5. Тесты (llmclient_test.go)
Table-driven:
- `content only` — Content="hello", Reasoning="" → "hello"
- `reasoning only` — Content=nil, Reasoning="thinking...answer" → "answer"
- `both present` — Content="short", Reasoning="long" → "short" (content приоритет)
- `both empty` — Content=nil, Reasoning="" → error
- `no choices` — Choices=[] → error
- `answer tag in reasoning` — reasoning с `<answer>` → извлечь

## Приёмка
- `go test ./internal/strataudit/...` проходит
- Пайплайн работает с `deepseek/deepseek-v3.2-speciale` (reasoning модель)
- Пайплайн работает с `google/gemini-2.5-flash` (content модель)

## Commit
`fix(strataudit): reasoning model fallback in LLM client`
