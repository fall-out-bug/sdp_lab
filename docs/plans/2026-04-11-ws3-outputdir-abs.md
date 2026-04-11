# WS-3: FIX-02 — Абсолютный путь outputDir

**Статус:** PENDING
**Приоритет:** P0
**Трудоёмкость:** 0.5ч
**Зависимости:** WS-1 (council consensus)

## Проблема
`cfg.Output.Dir` (default: `.strataudit`) резолвится от CWD, а не от `--dir`. DB создаётся через `filepath.Join(*dir, ...)`, но отчёты пишутся в `cfg.Output.Dir` от CWD. Split-brain: DB в одном месте, отчёты — в другом.

## Файлы
- `cmd/sdp-strataudit/main.go:89-95` — dbPath + config loading
- `internal/strataudit/pipeline.go:64` — outputDir usage

## Изменения

### 1. main.go — после cfg.Validate()
```go
// После cfg.Validate() (строка ~87):
absDir, err := filepath.Abs(*dir)
if err != nil {
    fmt.Fprintf(os.Stderr, "error resolving dir: %v\n", err)
    os.Exit(1)
}
cfg.Output.Dir = filepath.Join(absDir, cfg.Output.Dir)
```

### 2. main.go — упростить dbPath
```go
// Было:
dbPath := filepath.Join(*dir, cfg.Output.Dir, "strataudit.db")

// Стало:
dbPath := filepath.Join(cfg.Output.Dir, "strataudit.db")
```

### 3. main.go — вывод resolved пути
```go
fmt.Printf("Output:   %s\n", cfg.Output.Dir)
```

### 4. pipeline.go — без изменений
`cfg.Output.Dir` уже абсолютный, `outputDir := cfg.Output.Dir` работает корректно.

## Приёмка
- `sdp-strataudit run --dir /tmp/test` создаёт DB и отчёты в `/tmp/test/.strataudit/`
- `sdp-strataudit run --dir .` создаёт в `$PWD/.strataudit/`
- CLI выводит: `Output: /abs/path/.strataudit`

## Commit
`fix(strataudit): resolve outputDir as absolute path from --dir`
