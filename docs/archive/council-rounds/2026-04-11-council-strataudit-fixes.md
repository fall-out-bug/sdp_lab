# LLM Council: StratAudit Fix Prioritization (Round 2)

**Date:** 2026-04-11
**Rounds:** 2
**Consensus:** REACHED on 6/8 fixes
**Models:** gemini-3.1-pro (Critic), deepseek-v3.2 (Technician), kimi-k2.5 (Philosopher), minimax-m2.7 (Pragmatist), mimo-v2-pro (Engineer)

## P0 — Ship Now

### F1: LLM fallback на reasoning-поле [CRITICAL]
- **Consensus:** 5/5 SHIPPED_NOW
- **Effort:** 1h
- **Fix:** В `llmclient.go` добавить fallback: `if content == "" && reasoning != "" { content = reasoning }`
- **Warning (Critic):** Sanitize reasoning output before passing to JSON parsers

### F3: outputDir абсолютный путь [HIGH]
- **Consensus:** 4/5 SHIPPED_NOW
- **Effort:** 0.5h
- **Fix:** В `main.go` — `filepath.Join(*dir, cfg.Output.Dir)` или `filepath.Abs()`

## P1 — Before Next Run

### F4: JSON entities+traces [HIGH]
- **Consensus:** 3/5 NEXT_RUN
- **Effort:** 2-3h
- **Fix:** Добавить `entities[]` и `traces[]` в `report/json.go`

## P2 — Next Iteration

### F2: Русификация findings [HIGH]
- **Consensus:** CONDITIONAL (Critic: DEFER, Pragmatist: DEFER, Engineer: NEXT_RUN)
- **Effort:** 1-2h
- **Fix:** Локализация шаблонов в `analyze.go`
- **Risk (Critic):** Downstream regex/filters may depend on English keywords

### F8: Полный набор документов [MEDIUM]
- **Consensus:** CONDITIONAL
- **Effort:** 2h
- **Fix:** Использовать все 50+ implementation-документов

## P3 — Defer

### F6: Mermaid-схема трассировки [MEDIUM]
- **Consensus:** DEFER (3/5)
- **Effort:** 8h
- **Condition:** Defer until F4 done + >10 traces exist

### F5: Go PPTX/DOC экстракторы [HIGH]
- **Consensus:** DROP (Pragmatist+Engineer) / DEFER (Technician)
- **Effort:** 20h
- **Decision:** Python-скрипты работают, не тратить время в v1

## DROP

### F7: Pre-extraction фильтрация [MEDIUM]
- **Consensus:** DROP (3/5)
- **Rationale:** Premature optimization. Low coverage may be ground truth.
