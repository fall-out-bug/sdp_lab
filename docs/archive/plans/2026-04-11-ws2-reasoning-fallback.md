# WS-2: FIX-01 — Reasoning fallback в LLM клиенте

**Статус:** PENDING
**Приоритет:** P0
**Трудоёмкость:** 1-2ч
**Зависимости:** WS-1 (council consensus)

## Проблема
`llmclient.go:134-148` — response struct десериализует только `Content`. Reasoning-модели (deepseek-v3.2-speciale, deepseek-r1) возвращают `content: null`, текст — в `reasoning`. Пайплайн получает пустую строку.

## Файлы
- `internal/strataudit/llmclient.go:134-148` — response struct + content extraction
- `internal/strataudit/config.go:33-47` — LLMConfig (добавить ReasoningFallback)
- `internal/strataudit/llmclient_test.go` — тесты

## Изменения

### 1. Response struct (llmclient.go:134-143)
```go
// Было:
Message struct{ Content string } `json:"message"`

// Стало:
Message struct {
    Content   *string `json:"content"`
    Reasoning string  `json:"reasoning,omitempty"`
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
if content == "" && result.Choices[0].Message.Reasoning != "" {
    content = extractFinalAnswer(result.Choices[0].Message.Reasoning)
}
if content == "" {
    return nil, fmt.Errorf("llm: empty content and reasoning in response")
}
```

### 3. extractFinalAnswer() — новая функция
- Ищет `<answer>...</answer>` блок в reasoning — берёт содержимое
- Если тегов нет — берёт последние 80% reasoning текста (reasoning модели часто начинаются с chain-of-thought)
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
