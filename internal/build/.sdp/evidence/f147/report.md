# Microfirst Bench Report

Generated: 2026-04-26T22:29:05Z

## Classifier Metrics

| Classifier | Total | Micro Handled | Escalated | Fallback% | P50ms | P95ms | Accuracy% | LLM Saved% | Token Savings% |
|------------|-------|--------------|-----------|-----------|-------|-------|-----------|-----------|---------------|
| wsverdict | 35 | 32 | 3 | 8.6 | 0.000 | 0.000 | 93.8 | 91.4 | 91.4 |
| bdseverity | 18 | 0 | 18 | 100.0 | 0.016 | 0.018 | 0.0 | 0.0 | 0.0 |
| bdtype | 18 | 0 | 18 | 100.0 | 0.016 | 0.018 | 0.0 | 0.0 | 0.0 |
| routing | 18 | 2 | 16 | 88.9 | 0.019 | 0.021 | 0.0 | 11.1 | 11.1 |

## Summary

Total LLM calls saved: 25.6%

- **wsverdict**: 32/35 handled by micro (91.4% LLM savings)
- **bdseverity**: 0/18 handled by micro (0.0% LLM savings)
- **bdtype**: 0/18 handled by micro (0.0% LLM savings)
- **routing**: 2/18 handled by micro (11.1% LLM savings)

