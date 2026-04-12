# WS-4: FIX-03 — LLM-верификация в link stage

**Статус:** PENDING
**Приоритет:** P0
**Трудоёмкость:** 3-4ч
**Зависимости:** WS-2 (FIX-01 — reasoning fallback нужен для verify LLM calls)

## Проблема
`link.go:158-185` — `createTraces()` получает `*LLMClient`, но никогда его не вызывает. Трассировки создаются чисто по embedding similarity >= threshold. При similarity 0.67 кандидат проходит без LLM-проверки контекста.

## Файлы
- `internal/strataudit/link.go:158-185` — createTraces()
- `internal/strataudit/config.go:48-55` — ThresholdConfig (добавить AutoVerifySimilarity)
- `internal/strataudit/link_test.go` — тесты

## Изменения

### 1. createTraces() — двухуровневая верификация
```go
func createTraces(ctx context.Context, cfg *Config, llm *LLMClient, candidates []candidate, lowerLevel, upperLevel model.Level) ([]model.Trace, error) {
    threshold := cfg.Thresholds.TraceConfidence
    autoThreshold := cfg.Thresholds.AutoVerifySimilarity // default 0.85
    var traces []model.Trace
    autoCount := 0
    llmCount := 0

    for _, c := range candidates {
        if ctx.Err() != nil { return traces, ctx.Err() }

        if c.sim >= autoThreshold {
            // Auto-verified: high similarity
            traces = append(traces, makeTrace(c, fmt.Sprintf("Auto-verified (similarity: %.2f)", c.sim)))
            autoCount++
        } else if c.sim >= threshold {
            // LLM verification needed
            verified, err := verifyTrace(ctx, llm, c, lowerLevel, upperLevel)
            if err != nil {
                slog.Warn("trace verification failed", "error", err)
                continue
            }
            if verified.Related {
                traces = append(traces, makeTrace(c, verified.Justification))
                llmCount++
            }
        }
    }
    slog.Info("trace verification complete", "auto_verified", autoCount, "llm_verified", llmCount)
    return traces, nil
}
```

### 2. verifyTrace() — LLM вызов
```go
type verificationResult struct {
    Related       bool    `json:"related"`
    Confidence    float64 `json:"confidence"`
    Relation      string  `json:"relation"`
    Justification string  `json:"justification"`
}

func verifyTrace(ctx context.Context, llm *LLMClient, c candidate, lower, upper model.Level) (*verificationResult, error) {
    prompt := fmt.Sprintf(`Given two strategic entities:
A: %q (%s) at level %s
B: %q (%s) at level %s

Is there a meaningful strategic relationship between them?
Return JSON: {"related": bool, "confidence": 0.0-1.0, "relation": "contributes_to|enables|decomposes_into|depends_on|none", "justification": "..."}`,
        c.source.Title, c.source.Type, lower.Name,
        c.target.Title, c.target.Type, upper.Name)

    resp, err := llm.Chat(ctx, LLMRequest{
        System:    "You are a strategy analyst verifying entity relationships. Return valid JSON only.",
        User:      prompt,
        Model:     "",  // use default
        JSONMode:  true,
        MaxTokens: 500,
    })
    if err != nil { return nil, err }

    var result verificationResult
    if err := json.Unmarshal([]byte(parseLLMJSON(resp.Content)), &result); err != nil {
        return nil, fmt.Errorf("parse verification: %w", err)
    }
    return &result, nil
}
```

### 3. Конфиг
- `ThresholdConfig.AutoVerifySimilarity float64` — default 0.85
- YAML: `thresholds.auto_verify_similarity: 0.85`

### 4. Логирование
- `slog.Info("trace verification", "auto", N, "llm", M, "total_candidates", K)`

## Тесты
Table-driven в `link_test.go`:
- `sim=0.90` → auto-verified, no LLM call
- `sim=0.70 + LLM yes` → trace created
- `sim=0.70 + LLM no` → trace NOT created
- `sim=0.40` → skipped entirely
- `LLM error` → warning logged, trace not created

Mock LLMClient: interface с предопределёнными ответами.

## Приёмка
- При `trace_confidence: 0.6` кандидаты 0.60-0.84 проходят LLM-проверку
- Кандидаты >= 0.85 auto-verified
- Покрытие >10% (vs текущие 0-25%)
- `go test ./internal/strataudit/...` проходит

## Commit
`feat(strataudit): LLM verification in link stage`
