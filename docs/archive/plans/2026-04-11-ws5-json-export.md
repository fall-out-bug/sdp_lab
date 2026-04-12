# WS-5: FIX-04 — Полный JSON-экспорт

**Статус:** PENDING
**Приоритет:** P1
**Трудоёмкость:** 2-3ч
**Зависимости:** WS-4 (FIX-03 — link stage должен работать для traces)

## Проблема
`AuditReport` не содержит `entities` и `traces`. `BuildReport()` загружает только levels, findings, coverage. Невозможно восстановить полную модель из отчёта.

## Файлы
- `internal/strataudit/report/json.go:11-48` — AuditReport struct
- `internal/strataudit/report_builder.go:12-70` — BuildReport()
- `internal/strataudit/store.go` — добавить AllTraces()
- `internal/strataudit/report/json_test.go` — новый файл тестов

## Изменения

### 1. Новые типы (report/json.go)
```go
type EntityReport struct {
    ID          string `json:"id"`
    Type        string `json:"type"`
    Title       string `json:"title"`
    Description string `json:"description,omitempty"`
    LevelID     string `json:"level_id"`
    DocumentID  string `json:"document_id"`
}

type TraceReport struct {
    ID              string  `json:"id"`
    SourceEntityID  string  `json:"source_entity_id"`
    TargetEntityID  string  `json:"target_entity_id"`
    Relation        string  `json:"relation"`
    Confidence      float64 `json:"confidence"`
    Justification   string  `json:"justification"`
}
```

### 2. AuditReport (report/json.go)
```go
type AuditReport struct {
    // ... existing fields
    Entities []EntityReport `json:"entities,omitempty"`
    Traces   []TraceReport  `json:"traces,omitempty"`
}
```

### 3. Store — новый метод AllTraces() (store.go)
```go
func (s *SQLiteStore) AllTraces(ctx context.Context) ([]model.Trace, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, source_entity_id, target_entity_id, relation, confidence, justification, direction
        FROM traces ORDER BY confidence DESC`)
    // ... scan pattern как TracesForEntity
}
```

### 4. BuildReport() — загрузка entities + traces (report_builder.go)
```go
// После загрузки findings:
for _, l := range levels {
    entities, _ := store.EntitiesByLevel(ctx, l.ID, model.Page{Limit: 10000})
    for _, e := range entities {
        rpt.Entities = append(rpt.Entities, report.EntityReport{
            ID: e.ID, Type: string(e.Type), Title: e.Title,
            Description: e.Description, LevelID: e.LevelID, DocumentID: e.DocumentID,
        })
    }
}

traces, _ := store.AllTraces(ctx)
for _, t := range traces {
    rpt.Traces = append(rpt.Traces, report.TraceReport{
        ID: t.ID, SourceEntityID: t.SourceEntityID, TargetEntityID: t.TargetEntityID,
        Relation: string(t.Relation), Confidence: t.Confidence, Justification: t.Justification,
    })
}
```

## Приёмка
- `report.json` содержит `"entities": [...]` с id, type, title, level_id
- `report.json` содержит `"traces": [...]` с source/target entity IDs
- HTML отчёт не сломан

## Commit
`feat(strataudit): full entities and traces in JSON report`
