# WS-7: FIX-08 — Extractor interface + BridgeExtractor

**Статус:** PENDING
**Приоритет:** P1
**Трудоёмкость:** 4-6ч
**Зависимости:** WS-1 (council consensus)

## Проблема
Конвертация документов — полностью вне пайплайна. Пользователь вручную запускает Python-скрипт для конвертации PPTX, DOC и других форматов в TXT. Go-пайплайн поддерживает только `.txt/.md/.pdf/.docx`.

## Файлы
- `internal/strataudit/ingest.go:200-213` — extractText() switch
- `internal/strataudit/ingest.go:301-308` — isSupportedExt() switch
- `internal/strataudit/extract.go` — PDF extraction
- `internal/strataudit/extract_docx.go` — DOCX extraction
- Новые: `internal/strataudit/extractor.go`, `internal/strataudit/extractor_bridge.go`
- `internal/strataudit/config.go` — добавить ExtractorsConfig
- `internal/strataudit/ingest_test.go` — обновить тесты

## Изменения

### 1. Extractor interface (extractor.go)
```go
type Extractor interface {
    CanHandle(ext string) bool
    Extract(ctx context.Context, path string, data []byte) (string, error)
    Name() string
}

type ExtractorRegistry struct {
    extractors []Extractor
}

func NewExtractorRegistry(cfg *Config) *ExtractorRegistry {
    r := &ExtractorRegistry{}
    r.Register(&TextExtractor{})
    r.Register(&PDFExtractor{})
    r.Register(&DOCXExtractor{})
    if cfg.Extractors.ExternalCommand != "" {
        r.Register(NewBridgeExtractor(cfg.Extractors))
    }
    return r
}
```

### 2. Обёртки существующих экстракторов
- `TextExtractor`: `.txt/.md/.markdown` → `string(data)`
- `PDFExtractor`: `.pdf` → `extractPDFWithLedongthuc(data)`
- `DOCXExtractor`: `.docx` → `extractDOCXFromZIP(data)`

### 3. BridgeExtractor (extractor_bridge.go)
```go
type BridgeExtractor struct {
    command    string   // "textutil" или "libreoffice"
    extensions []string // ["pptx", "doc", "rtf", "xls", "xlsx"]
}

func (b *BridgeExtractor) Extract(ctx context.Context, path string, data []byte) (string, error) {
    // 1. Записать data во временный файл (нужен для textutil/libreoffice)
    // 2. Sanitize: filepath.Base() для имени
    // 3. exec.CommandContext(ctx, b.command, args...)
    // 4. Прочитать stdout (textutil -stdout) или выходной файл (libreoffice)
    // 5. Очистить временные файлы
}
```

Sanitize от command injection:
- `filepath.Base(path)` — только имя файла, без пути
- Временный файл в `os.TempDir()`
- Проверить что путь не содержит `..` или `|` или `;`

### 4. Конфиг
```go
type ExtractorsConfig struct {
    ExternalCommand string   `yaml:"external_command"` // "textutil" (default) или "libreoffice"
    UseExternal     []string `yaml:"use_external"`     // ["pptx", "doc", "rtf"]
}
```
Defaults: `ExternalCommand: "textutil"`, `UseExternal: ["pptx", "doc", "rtf", "xls", "xlsx"]`

### 5. Обновить ingest.go
- `isSupportedExt()` → `registry.CanHandle(ext)`
- `extractText()` → `registry.Extract(ctx, path, data)`

## Тесты
- `TestExtractorRegistry_Routing` — table-driven по extension
- `TestTextExtractor` — существующий
- `TestBridgeExtractor_TextutilAvailable` — интеграционный (skip если нет textutil)
- `TestBridgeExtractor_Sanitize` — пути с `..`, `|`, `;`

## Приёмка
- `sdp-strataudit run --dir /path/with/pptx` — извлекает текст без Python-скриптов
- Если `textutil` не установлен — warning + skip, не fatal
- Существующие форматы (.txt, .pdf, .docx) работают как раньше

## Commit
`feat(strataudit): Extractor interface with bridge for PPTX/DOC/RTF`
