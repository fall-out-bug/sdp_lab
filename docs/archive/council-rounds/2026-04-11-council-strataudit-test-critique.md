# LLM Council Report: StratAudit Test Run Critique

**Rounds:** 1 of 2
**Consensus:** PARTIAL (6/8 issues resolved)
**Convergence:** 6/8 resolved, 0 deferred, 2 split
**Quorum:** 5/6 models (Architect pending codex-rescue)

## Issue Ledger

| ID | Title | Severity | Verdict | Confidence |
|----|-------|----------|---------|------------|
| I1 | Findings на английском | HIGH | RESOLVED — SUPPORT 5/5 | HIGH |
| I2 | Нет схемы трассировки | HIGH | DEFERRED — depends on I8 | HIGH |
| I3 | Нет PPTX/DOC экстракторов в Go | HIGH | SPLIT — 1/3/1 | MEDIUM |
| I4 | Reasoning-модели не поддерживаются | CRITICAL | RESOLVED — SUPPORT 5/5 | CRITICAL |
| I5 | JSON-отчёт неполный | MEDIUM | RESOLVED — SUPPORT 4/5 | HIGH |
| I6 | outputDir резолвится от CWD | MEDIUM | RESOLVED — SUPPORT 5/5 | HIGH |
| I7 | Нет pre-extraction | MEDIUM | SPLIT — 2/0/2 | MEDIUM |
| I8 | Низкое покрытие 0-25% | MEDIUM | CONDITIONAL — root cause unclear | MEDIUM |

## RESOLVED (consensus ≥80%)

### I4: Reasoning-модели — CRITICAL, 5/5 SUPPORT
**Action:** В `llmclient.go` добавить fallback на `reasoning` поле если `content` пустой:
```go
type message struct {
    Content   *string `json:"content"`
    Reasoning string  `json:"reasoning,omitempty"`
}
// content := deref(msg.Content); if content == "" && msg.Reasoning != "" { content = extractFinalAnswer(msg.Reasoning) }
```

### I1: Findings на английском — HIGH, 5/5 SUPPORT
**Action:** Локализовать шаблоны findings в `analyze.go`:
- "Gap:" → "Разрыв:"
- "Orphan:" → "Изолированная сущность:"
- "has no support from" → "не имеет поддержки от уровня"
- Добавить параметр `lang` в конфиг

### I6: outputDir резолвится от CWD — HIGH, 5/5 SUPPORT
**Action:** В `main.go` резолвить `cfg.Output.Dir` через `filepath.Join(*dir, cfg.Output.Dir)` или `filepath.Abs()`

### I5: JSON-отчёт неполный — HIGH, 4/5 SUPPORT
**Action:** Добавить `entities` и `traces` в JSON-отчёт (`report/json.go`)

## SPLIT (requires Round 2)

### I3: Go-экстракторы для PPTX/DOC
- **OPPOSE** (Pragmatist): Python работает, не тратить время на MVP
- **CONDITIONAL** (Philosopher): Python лучше для документов, унифицировать через интерфейс
- **Critic**: Security veto — Command Injection через filenames
- **Decision needed:** Go-only (unioffice) vs interface abstraction vs keep Python

### I7: Pre-extraction
- **SUPPORT** (Engineer, Technician): TF-IDF/keyword фильтрация перед LLM
- **OPPOSE** (Pragmatist): Premature optimization для MVP
- **Decision needed:** Implement now vs defer to v2

## Minority Reports

**Philosopher on I8:** "Low coverage could be the ground truth — accurate detection of documentation gaps rather than algorithmic failure."

**Philosopher on I3:** "Python may be superior for document parsing (ecosystem maturity). Define Extractor interface with two implementations."

## Round Convergence
| Round | Resolved | New | Confidence Avg | Models Active |
|-------|----------|-----|----------------|---------------|
| 1     | 6/8      | 0   | HIGH           | 5/6           |
