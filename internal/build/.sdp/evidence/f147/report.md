# Microfirst Bench Report

Generated: 2026-04-26T22:59:17Z

## Classifier Metrics

| Classifier | Total | Micro Handled | Escalated | Fallback% | P50ms | P95ms | Accuracy% | LLM Saved% | Token Savings% |
|------------|-------|--------------|-----------|-----------|-------|-------|-----------|-----------|---------------|
| wsverdict | 35 | 32 | 3 | 8.6 | 0.000 | 0.000 | 93.8 | 91.4 | 91.4 |
| bdseverity | 30 | 25 | 5 | 16.7 | 0.010 | 0.012 | 100.0 | 83.3 | 83.3 |
| bdtype | 30 | 20 | 10 | 33.3 | 0.011 | 0.012 | 100.0 | 66.7 | 66.7 |
| routing | 30 | 25 | 5 | 16.7 | 0.011 | 0.016 | 100.0 | 83.3 | 83.3 |

## Summary

Total LLM calls saved: 81.2%

- **wsverdict**: 32/35 handled by micro (91.4% LLM savings)
- **bdseverity**: 25/30 handled by micro (83.3% LLM savings)
- **bdtype**: 20/30 handled by micro (66.7% LLM savings)
- **routing**: 25/30 handled by micro (83.3% LLM savings)

