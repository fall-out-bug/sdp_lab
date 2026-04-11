# WS-4a: FIX-03a — Логирование распределения similarity (P0)

**Статус:** APPROVED (Council R2 consensus, 6/6 SUPPORT)
**Приоритет:** P0 (v1.0.1, Slice 2)
**Трудоёмкость:** 1-2ч
**Зависимости:** WS-2 (FIX-01), WS-3 (FIX-02)

## Проблема
110 сущностей дали только 1 трассировку. Распределение similarity неизвестно — невозможно определить, проблема в пороге, качестве embeddings или структурном несоответствии.

## Файлы
- `internal/strataudit/link.go:101-135` — computeSimilarity() — добавить accumulation
- `internal/strataudit/config.go:48-55` — ThresholdConfig (добавить EmitDistribution)

## Изменения

### 1. В computeSimilarity() — накопление статистики
```go
type SimilarityStats struct {
    LevelPair     string         `json:"level_pair"`
    TotalPairs    int            `json:"total_pairs"`
    AboveThreshold int           `json:"above_threshold"`
    Distribution  struct {
        Min    float64 `json:"min"`
        Max    float64 `json:"max"`
        Mean   float64 `json:"mean"`
        Median float64 `json:"median"`
        P95    float64 `json:"p95"`
    } `json:"distribution"`
    Histogram []HistogramBucket `json:"histogram"`
}
```

### 2. В LinkEntities() — запись JSON файла
- Сохранить в `{output.dir}/similarity_distribution.json`
- Schema: run_id, generated_at, level_pairs[{stats, histogram}], current_threshold, recommendation
- Recommendation heuristic: если above_threshold < total_pairs * 0.02 → "threshold_may_be_too_high"

### 3. Конфиг
- `thresholds.emit_distribution: true` (default)
- YAML: `thresholds.emit_distribution: true`

## Приёмка
- После прогона: `similarity_distribution.json` существует в output dir
- Файл содержит stats по каждой паре уровней (min, max, mean, median, p95, histogram)
- Текущее покрытие трассировок не меняется (только диагностика)

## Commit
`feat(strataudit): log similarity distribution for link stage diagnostics`
