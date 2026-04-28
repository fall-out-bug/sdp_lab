# Python Extractor - F105 AI Architect

Python dependency and framework detection using pragmatic regex/text parsing.

## Scope

This subdirectory implements the Python extractor for the F105 AI Architect feature, providing:
- Import statement extraction with 60-70% accuracy
- Framework detection (Flask, FastAPI, Django, Celery)
- Dependency manifest parsing (requirements.txt, pyproject.toml, setup.py, setup.cfg, Pipfile)

## Files

- **extractor.go** - Main `PythonExtractor` implementing `architect.Extractor`
- **extractor_test.go** - Comprehensive tests for extraction scenarios
- **imports.go** - Import parsing with regex patterns (supports `import X`, `from X import Y`, relative imports)
- **dependencies.go** - Dependency manifest parsers (requirements.txt, pyproject.toml, setup.py, setup.cfg, Pipfile)
- **framework.go** - Framework detection with confidence scoring
- **framework_test.go** - Framework detection tests
- **treesitter.go** - Tree-sitter query definitions for future CGo integration
- **constants.go** - Shared constants (skip directories, test directories)

## Acceptance Criteria Met

✅ **Import Extraction**
- Extracts top-level import statements with 60-70% accuracy
- Handles `import X` statements
- Handles `from X import Y` statements
- Handles relative imports (`from . import`, `from .. import`)
- Distinguishes stdlib from third-party packages

✅ **Import Resolution**
- `import X` - Absolute imports
- `from X import Y` - From imports with module name extraction
- Relative imports with dot counting (`from . import`, `from ..sub import`)

✅ **Dependency Manifests**
- **requirements.txt**: Pinned (`==`) and unpinned versions, comments, option flags
- **pyproject.toml**: PEP 621 `[project.dependencies]`, Poetry `[tool.poetry.dependencies]`
- **setup.py**: `install_requires` list with regex-based extraction
- **setup.cfg**: `[options]` section with `install_requires`
- **Pipfile**: `[packages]` and `[dev-packages]` sections

✅ **Framework Detection**
- **Flask**: `app = Flask()`, `@app.route()`, `Blueprint()`
- **FastAPI**: `app = FastAPI()`, `@app.get/post()`, `APIRouter()`
- **Django**: `INSTALLED_APPS`, `urlpatterns`, `class Model(models.Model)`, `AppConfig`
- **Celery**: `app = Celery()`

✅ **Known Blind Spots Documented**
- Dynamic imports (`importlib.import_module()`)
- `sys.path` manipulation
- Conditional imports (`if DEBUG: import foo`)
- Notebook imports (`.ipynb` files)
- String-based imports (`__import__("foo")`)

## Implementation Details

### Pragmatic Regex Approach

Uses regex-based parsing instead of tree-sitter to avoid heavy CGo dependencies:
- **Pros**: Fast, no CGo dependencies, good enough for common patterns
- **Cons**: 60-70% accuracy vs 90-95% with tree-sitter, misses complex patterns

### Tree-Sitter Migration Path

`treesitter.go` documents tree-sitter query definitions for future CGo integration:
- `TreeSitterImportQuery()` - Import extraction queries
- `TreeSitterFrameworkQuery()` - Framework detection queries
- Accuracy improvements expected: 60-70% → 90-95%

### Framework Detection

Framework patterns detected with confidence scores:
- Flask: 85-95% confidence (strong pattern matching)
- FastAPI: 85-95% confidence (distinctive decorators)
- Django: 85-95% confidence (strong conventions)
- Celery: 80-90% confidence (pattern varies)

## Usage

```go
import (
    "context"
    "github.com/fall-out-bug/sdp_lab/internal/architect/extract/python"
)

func main() {
    extractor := &python.PythonExtractor{}
    ctx := context.Background()
    result, err := extractor.Extract(ctx, "/path/to/project")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Language: %s\n", result.Language)
    fmt.Printf("Files: %d\n", result.FileCount)
    fmt.Printf("Dependencies: %d\n", len(result.Dependencies))
    fmt.Printf("Frameworks: %d\n", len(result.Frameworks))
}
```

## Testing

Run tests with:
```bash
go test ./internal/architect/extract/python/... -count=1 -v
```

Test coverage:
- Simple Flask projects with requirements.txt
- FastAPI projects with pyproject.toml
- Django projects with multiple files
- Mixed dependency sources (requirements.txt + setup.py)
- Pipfile projects
- Non-Python projects (empty result)

## Known Limitations

1. **Import Accuracy**: 60-70% due to regex limitations
2. **Relative Imports**: Simplified resolution without full path context
3. **Conditional Imports**: Not detected (inside `if` blocks)
4. **Dynamic Imports**: `importlib`, `__import__()` not handled
5. **Notebooks**: `.ipynb` files not parsed
6. **Complex Patterns**: Multi-line imports, nested imports not fully supported

## Future Enhancements

1. **Tree-Sitter Integration**: Enable CGo-based tree-sitter for 90-95% accuracy
2. **Import Graph**: Build C4 Level 3 clustering from import relationships
3. **Type Hints**: Extract type information from `typing` module usage
4. **Async Detection**: Identify async functions and coroutine patterns
5. **Test Detection**: Improved test file detection beyond directory names

## Accuracy Estimates

| Feature | Current (Regex) | With Tree-Sitter |
|---------|-----------------|------------------|
| Import extraction | 60-70% | 90-95% |
| Framework detection | 75-85% | 85-90% |
| Dependency parsing | 85-90% | 95-98% |

## Dependencies

No external dependencies required for basic functionality.
Tree-sitter integration would require:
- `github.com/tree-sitter/tree-sitter-python` (CGo)

## License

Part of the SDP Lab project.
