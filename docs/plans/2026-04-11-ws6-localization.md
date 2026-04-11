# WS-6: FIX-05 — Русификация findings

**Статус:** APPROVED (Council R2)
**Приоритет:** P1
**Трудоёмкость:** 1-2ч
**Зависимости:** WS-4 (FIX-03 — link stage)

## Проблема
Все шаблоны findings на английском при русскоязычных документах. 7 finding types с hardcoded английскими строками.

## Файлы
- `internal/strataudit/analyze.go:52-166` — finding templates
- `internal/strataudit/report/html.go:19-103` — HTML hardcoded strings
- `internal/strataudit/config.go:57-60` — OutputConfig (добавить Lang)

## Изменения

### 1. Конфиг (config.go)
```go
type OutputConfig struct {
    Dir     string   `yaml:"dir"`
    Formats []string `yaml:"formats"`
    Lang    string   `yaml:"lang"` // "ru" (default) или "en"
}
```
Default в Validate(): `if c.Output.Lang == "" { c.Output.Lang = "ru" }`

### 2. Локализация findings (analyze.go)
```go
var findingTemplates = map[string]map[string]struct{ Title, Desc, Rec string }{
    "gap": {
        "ru": {
            Title: "Разрыв: %q не имеет поддержки от уровня %s",
            Desc:  "Сущность %q на уровне %s (ранг %d) не имеет трассированных связей с уровнем %s.",
            Rec:   "Добавить операционные сущности на уровне %s, которые поддерживают эту цель.",
        },
        "en": {
            Title: "Gap: %q has no support from %s",
            Desc:  "Entity %q at level %s (rank %d) has no traced contributions from level %s.",
            Rec:   "Add operational entities at %s level that contribute to this goal.",
        },
    },
    "orphan": { /* ... */ },
    "weak_link": { /* ... */ },
    "coverage": { /* ... */ },
    "alignment": { /* ... */ },
    "strong_trace": { /* ... */ },
    "ambiguous_trace": { /* ... */ },
}

var entityTypeLabels = map[string]map[string]string{
    "goal":        {"ru": "цель", "en": "goal"},
    "objective":   {"ru": "задача", "en": "objective"},
    "initiative":  {"ru": "инициатива", "en": "initiative"},
    // ...
}
```

### 3. Использование в Analyze()
```go
lang := cfg.Output.Lang
tpl := findingTemplates["gap"][lang]
title := fmt.Sprintf(tpl.Title, e.Title, lower.Name)
```

### 4. HTML report (report/html.go)
- Вынести все строки в map
- Рендерить в зависимости от lang

### Важно
`finding.type` остаётся английским (machine-readable). Локализуется только `title` и `description`.

## Приёмка
- `report.json`: `"title": "Разрыв: \"Платежный хаб\" не имеет поддержки от уровня architecture"`
- `report.json`: `"type": "gap"` (английский, machine-readable)
- HTML отчёт: русские заголовки и кнопки
- `output.lang: "en"` — английский вариант

## Commit
`feat(strataudit): localize findings to Russian (output.lang)`
