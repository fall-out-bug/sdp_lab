# WS-4b: FIX-03b — LLM-верификация в link stage (P1)

**Статус:** APPROVED (Council R2 consensus, 6/6 SUPPORT)
**Приоритет:** P1 (v1.1.0, Slice 3)
**Трудоёмкость:** 3-4ч
**Зависимости:** WS-4a (FIX-03a — нужна диагностика similarity для принятия решения)

## Решение (на основе данных FIX-03a)
Реализация зависит от результатов диагностики:
- Если распределение бимодальное → настройка порогов (без LLM)
- Если распределение плоское → LLM verification с budget controls

## Файлы
- `internal/strataudit/link.go:158-185` — createTraces()
- `internal/strataudit/config.go` — ThresholdConfig (добавить AutoVerifySimilarity, LLMVerifyBudget)

## Изменения (если LLM verify нужен)

### 1. Двухуровневая верификация
- similarity >= 0.85 → auto-verified
- similarity в [trace_confidence, 0.85) → LLM verification

### 2. Budget control (Council requirement)
- `link.llm_verify_budget: 50` — hard stop, не soft warning
- При budget exhaustion: `slog.Warn("LLM verify budget exhausted")`

### 3. Concurrency control (Council requirement)
- Semaphore matching `cfg.LLM.MaxConcurrent`
- Rate limiter через существующий `c.limiter.Wait(ctx)`

### 4. Fail-closed (Council requirement)
- LLM verification failure = pair rejected (не accepted)

### 5. Config
```yaml
thresholds:
  auto_verify_similarity: 0.85
link:
  llm_verify_budget: 50
```

## Приёмка
- Покрытие >10% (vs текущие 0-25%)
- LLM verify budget respected (не более 50 вызовов)
- `slog.Info("trace verification", "auto", N, "llm", M)`

## Commit
`feat(strataudit): LLM verification in link stage with budget controls`
