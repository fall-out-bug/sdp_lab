# AI Architect -- Phase 1 Extractor Implementation Specification

**Date:** 2026-04-10
**Status:** Implementation-ready
**Depends on:** `docs/plans/2026-04-10-ai-architect-design.md` (v3)
**Affects:** `internal/architect/extract/` package

This document fills the gap between the design spec's high-level extractor descriptions and the concrete patterns a developer must implement. Every section is normative: a developer should be able to implement each extractor from this spec alone without making additional design decisions.

---

## 1. Tree-Sitter Import Queries

> **Phase marker:** All tree-sitter queries in this section are **Phase 2 implementation, Phase 1 contract definition**. In Phase 1, these queries define the interface contract that the tree-sitter integration must satisfy. The actual `.scm` files and `go-tree-sitter` bindings are implemented in Phase 2. Phase 1 uses the existing regex extractors with the fixes specified elsewhere in this document.

The current codebase uses regex for Python, Java, and TypeScript import extraction (see `internal/architect/extract/python_extract.go`, `java_extract.go`, `typescript_extract.go`). The design spec mandates tree-sitter as the default parser. This section specifies the tree-sitter queries and resolution algorithms that will replace or augment the regex implementations.

### 1.1 Go

Go uses `go/packages` natively (see `internal/architect/extract/go_extract.go`). Tree-sitter is NOT used for Go -- the native approach achieves 90-95% accuracy.

**Import path resolution algorithm:**
1. Read `go.mod` to extract module path.
2. `go/packages.Load(cfg, "./...")` with `NeedName | NeedImports`.
3. Filter edges to packages whose import path starts with module path (internal packages).
4. Skip packages where all `.go` files match `*.pb.go` or `*.gen.go` (generated code).
5. Cluster assignment: `pkg.PkgPath` minus module path prefix, up to last `/`.

**Known blind spots:**
- Build tags (`//go:build`) may cause packages to be skipped if the tags are not satisfied during analysis.
- CGo imports (`import "C"`) -- filtered out by the `isInternal` check.
- Dot imports (`import . "pkg"`) -- the import is recorded but the implicit namespace pollution is not tracked.
- `go generate` output -- partially handled by `isGenerated()` but only checks file suffixes.

**Accuracy estimate:** 90-95%.

### 1.2 Python

<!-- Phase 2 implementation, Phase 1 contract definition -->
<!-- go-tree-sitter does not support #eq? predicates; use Go code filtering -->

**Tree-sitter query (.scm) -- Phase 2 implementation, Phase 1 contract definition:**

```scheme
; --- Absolute imports ---
; import os
(import_statement
  (dotted_name) @import.module)

; import os.path
(import_statement
  (dotted_name) @import.module)

; import a, b  (multiple names via aliased_import)
(import_statement
  (aliased_import
    name: (dotted_name) @import.module))

; --- from-import (absolute module) ---
; from flask import Flask
(import_from_statement
  module_name: (dotted_name) @import.from_module)

; from flask.restful import Api
(import_from_statement
  module_name: (dotted_name) @import.from_module)

; --- from-import (relative module) ---
; from . import X
(import_from_statement
  module_name: (relative_import) @import.relative)

; from ..sub import Y
(import_from_statement
  module_name: (relative_import) @import.relative)

; --- Wildcard: from module import * ---
(import_from_statement
  module_name: (dotted_name) @import.from_module
  (wildcard_import))
```

**Note:** `relative_import` is a concrete node type in tree-sitter-python (not `import_prefix`, which is a field name). The `import_prefix` field is used internally by tree-sitter but is not a valid capture target. Relative import depth (number of leading dots) is parsed from the `relative_import` node text in Go code.

**Import path resolution algorithm:**

1. For `import X` statements:
   - Extract the first component of the dotted name (e.g., `flask` from `flask.restful`).
   - Check against `pythonStdlib` map (see `python_extract.go` lines 15-61). If found, classify as `stdlib`.
   - Check if a matching directory or `.py` file exists under the project root. If yes, classify as `relative`.
   - Otherwise, classify as `third-party`.

2. For `from X import Y` statements:
   - Resolve `X` using the same algorithm as step 1.
   - If `X` starts with `.` (relative import), count leading dots to determine parent levels:
     - `from . import X` (1 dot): same package as current file's directory.
     - `from .. import X` (2 dots): parent package.
     - `from .sub import X` (1 dot + name): sibling module.
   - Resolution uses the current file's relative path to compute the target module path (see `resolveImport()` in `python_extract.go` lines 305-359).

3. Manifest correlation:
   - Cross-reference discovered imports against `requirements.txt` and `pyproject.toml` dependencies.
   - Imports not found in manifests but not in stdlib are potential undeclared dependencies or local modules.

**Known blind spots:**
- `importlib.import_module("X")` -- dynamic imports are invisible to static analysis.
- `sys.path` manipulation at runtime -- changes the effective import resolution.
- Conditional imports (`try: import X except: import Y`) -- tree-sitter captures both branches, cannot determine which is active.
- Notebook (`.ipynb`) imports -- not scanned (different file format).
- `__init__.py` implicit re-exports -- importing a package does not imply importing its submodules.

**Accuracy estimate:** 60-70%.

### 1.3 Java/Kotlin

<!-- Phase 2 implementation, Phase 1 contract definition -->

**Tree-sitter query for Java (.scm) -- Phase 2 implementation, Phase 1 contract definition:**

```scheme
; Single import: import com.example.Service;
(import_declaration
  (scoped_identifier) @import.path)

; Wildcard import: import com.example.*;
(import_declaration
  (scoped_identifier) @import.path
  (asterisk) @import.wildcard)
```

**Static import handling:** `tree-sitter-java` does not have a `static_import` child node. Static imports (`import static X.Y`) are detected in Go code by checking the raw text of the `import_declaration` node for the `static` keyword prefix. The import path is extracted the same way as non-static imports, with a `static: true` flag set in the output.

**Tree-sitter query for Kotlin (.scm):**

```scheme
; import statement
(import_header
  (identifier) @import.path)

; import alias: import com.example.Service as Svc
(import_header
  (identifier) @import.path
  (import_alias
    (identifier) @import.alias))
```

**Import path resolution algorithm:**

1. Extract the fully-qualified package path from each `import_declaration`.
2. Determine if the import is internal or external:
   - Read the project's `package` declaration from the same file to get the base package.
   - If the import starts with the project's base package (inferred from the directory structure under `src/main/java/` or `src/main/kotlin/`), classify as `internal`.
   - Otherwise, cross-reference against dependencies in `pom.xml` or `build.gradle` to classify as `third-party` with source framework.
3. For wildcard imports (`import X.*`):
   - Record the wildcard but do not expand it (would require classpath resolution).
   - Mark as `wildcard: true` in the output; this is a known low-accuracy signal.
4. Map import paths to source files:
   - Convert `com.example.Service` to `src/main/java/com/example/Service.java`.
   - If the file does not exist, check for Kotlin: `src/main/kotlin/com/example/Service.kt`.

**Known blind spots:**
- Reflection (`Class.forName("com.example.Service")`) -- invisible to static analysis.
- Runtime DI wiring (Spring `@Autowired`, CDI) -- dependencies are declared in annotations/XML, not imports.
- Annotation processors -- generate code at compile time, invisible pre-compilation.
- Kotlin DSL configs (`build.gradle.kts`) -- may contain dependency declarations that are not standard Gradle syntax.
- Maven/Gradle multi-module resolution -- a module's `pom.xml` may reference sibling modules whose sources are in adjacent directories.

**Accuracy estimate:** 70-80%.

### 1.4 TypeScript/JavaScript

<!-- Phase 2 implementation, Phase 1 contract definition -->
<!-- go-tree-sitter does not support #eq? predicates; use Go code filtering -->

**Tree-sitter query (.scm) -- Phase 2 implementation, Phase 1 contract definition:**

> **IMPORTANT:** go-tree-sitter does not support `#eq?` predicates. All identifier
> filtering (e.g., distinguishing `require` from other function calls) must be done
> in Go code after the query returns results.

```scheme
; --- ES module imports ---
; import X from "Y"
(import_statement
  (import_clause
    (_) @import.name)
  (string) @import.source)

; ES module side-effect: import "Y"
(import_statement
  (string) @import.source)

; --- Re-export ---
; export { X } from "Y"
(export_statement
  (string) @import.source)

; --- Dynamic import: import("Y") ---
(call_expression
  function: (import) @import.dynamic
  arguments: (arguments
    (string) @import.source))

; --- CommonJS require (structural match only) ---
; Matches ANY call_expression with identifier function and string argument.
; Go code MUST filter by checking if the captured @_fn identifier text equals "require".
(call_expression
  function: (identifier) @import._fn
  arguments: (arguments
    (string) @import.source))

; --- Require with property access: const X = require("Y").Z ---
; Matches member_expression whose object is a call with identifier function.
; Go code MUST filter by checking if the captured @_fn identifier text equals "require".
(call_expression
  function: (member_expression
    object: (call_expression
      function: (identifier) @import._fn
      arguments: (arguments
        (string) @import.source))))
```

**Go-side filtering for CommonJS `require`:**

After executing the query, iterate captured nodes. For any capture named `import._fn`, check whether the node text equals `"require"`. Discard the match if it does not. This replaces the unsupported `#eq?` predicate.

**Import path resolution algorithm:**

1. Classify specifier type:
   - Starts with `./` or `../` -- relative import.
   - Starts with `@/` or other path alias -- check `tsconfig.json` `compilerOptions.paths` (see `parseTSConfig()` in `typescript_extract.go` lines 233-273).
   - Starts with `@` followed by scope name (e.g., `@nestjs/core`) -- scoped package.
   - Otherwise -- bare package name.

2. Relative import resolution:
   - Resolve relative to the importing file's directory.
   - Try extensions in order: `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`.
   - Try `/index.ts`, `/index.tsx`, `/index.js` if the path points to a directory.

3. Path alias resolution:
   - Read `tsconfig.json` `compilerOptions.paths` and `baseUrl`.
   - Apply prefix replacement: `@/components/Button` with alias `@/*` -> `src/*` becomes `src/components/Button`.
   - Resolve the resulting path as a relative import.

4. Package import resolution:
   - Look up `node_modules/<package>/package.json` to determine entry point.
   - Check `package.json` `exports` field first (Node.js resolution algorithm).
   - Fall back to `main` field.
   - If no `node_modules` (monorepo), check workspace packages.

5. Re-export handling:
   - Mark re-exports with `Kind: ImportReExport`.
   - Barrel files (`index.ts` that re-export from many modules) are a known accuracy reducer.
   - Do NOT follow re-export chains -- record the direct dependency only.

**Known blind spots:**
- Dynamic `import()` -- captured by tree-sitter but the argument may be a variable, not a string literal.
- `require` with variable argument (`require(variable)`) -- not captured.
- Path aliases defined in build tools (webpack `resolve.alias`, Vite `resolve.alias`) but NOT in `tsconfig.json`.
- Barrel re-exports create false transitivity -- `import {X} from './index'` appears to depend on the barrel, not the actual source of X.
- Webpack Module Federation -- runtime-loaded modules are invisible.
- CommonJS `module.exports = require('X')` patterns -- partially captured.

**Accuracy estimate:** 65-75%.

### 1.5 SQL

SQL is NOT analyzed for imports. It provides data architecture signals only. See the SQL Analysis section below and the existing implementation in `internal/architect/extract/sql_extract.go`.

---

## 2. Framework Detection Pattern Library

### 2.1 Spring Boot (Java)

**Detection signals (any ONE is sufficient):**

| Signal | Location | Confidence |
|--------|----------|------------|
| `@SpringBootApplication` annotation | Source file | 0.95 |
| `spring-boot-starter` in `<artifactId>` | `pom.xml` | 0.90 |
| `org.springframework.boot:spring-boot-starter` in dependencies | `build.gradle` | 0.90 |
| `import org.springframework.boot.*` | Source file | 0.70 |

**Component classification patterns:**

| Role | Annotations | How to extract |
|------|-------------|----------------|
| Controller | `@RestController`, `@Controller` | Scan for these annotations on class declarations |
| HTTP endpoint | `@RequestMapping`, `@GetMapping`, `@PostMapping`, `@PutMapping`, `@DeleteMapping`, `@PatchMapping` | Extract from method-level annotations. Parse the string argument as the path. If class has `@RequestMapping("/api")` and method has `@GetMapping("/users")`, the full path is `/api/users`. |
| Service | `@Service`, `@Component` | Scan class-level annotations |
| Repository | `@Repository` | Scan class-level annotations |
| Data entity | `@Entity`, `@Table` | Scan class-level annotations |
| Configuration | `@Configuration`, `@Bean` | `@Configuration` on class, `@Bean` on method |

**HTTP endpoint extraction algorithm:**

1. Find class-level `@RequestMapping` (or `@RestController` with path prefix). Extract the path prefix.
2. For each method with `@XMapping` annotation:
   - Extract HTTP method from annotation type: `GetMapping` -> GET, `PostMapping` -> POST, etc.
   - Extract path from annotation string argument. Multiple paths in one annotation are treated as aliases (record all).
   - Concatenate class prefix + method path.
   - If no path argument, the method responds to the class prefix only.
3. Output: `[{method: "GET", path: "/api/users/{id}", handler: "UserController.getUser"}]`

**Detection regex patterns (current implementation uses import-based detection, see `java_extract.go` lines 298-369):**

```go
// Annotation patterns to scan for in source files (not just imports)
var springAnnotationPatterns = map[string]string{
    `@SpringBootApplication`:   "spring_app",
    `@RestController`:          "spring_controller",
    `@Controller`:              "spring_controller",
    `@RequestMapping`:          "spring_endpoint",
    `@GetMapping`:              "spring_endpoint_get",
    `@PostMapping`:             "spring_endpoint_post",
    `@PutMapping`:              "spring_endpoint_put",
    `@DeleteMapping`:           "spring_endpoint_delete",
    `@PatchMapping`:            "spring_endpoint_patch",
    `@Service`:                 "spring_service",
    `@Component`:               "spring_component",
    `@Repository`:              "spring_repository",
    `@Entity`:                  "spring_entity",
    `@Table`:                   "spring_entity",
    `@Configuration`:           "spring_config",
    `@Bean`:                    "spring_config_bean",
}
```

### 2.2 Flask (Python)

**Detection signals:**

| Signal | Location | Confidence |
|--------|----------|------------|
| `flask` in `requirements.txt` or `pyproject.toml` | Manifest | 0.85 |
| `from flask import` in source | Source file | 0.90 |
| `import flask` in source | Source file | 0.80 |
| `@app.route` decorator | Source file | 0.95 |

**Route extraction patterns:**

```go
var flaskRoutePatterns = []*regexp.Regexp{
    // @app.route("/path", methods=["GET", "POST"])
    regexp.MustCompile(`@app\.route\s*\(\s*["']([^"']+)["']\s*(?:,\s*methods\s*=\s*\[([^\]]+)\])?`),
    // @app.get("/path")
    regexp.MustCompile(`@app\.(get|post|put|delete|patch)\s*\(\s*["']([^"']+)["']`),
    // Blueprint routes: @blueprint.route("/path")
    regexp.MustCompile(`@\w+\.route\s*\(\s*["']([^"']+)["']`),
    // Blueprint method routes: @bp.get("/path")
    regexp.MustCompile(`@\w+\.(get|post|put|delete|patch)\s*\(\s*["']([^"']+)["']`),
}
```

**Route extraction algorithm:**

1. Scan each `.py` file for decorator patterns matching `flaskRoutePatterns`.
2. For `@app.route("/path", methods=["GET"])`:
   - Extract path from group 1.
   - Extract methods from group 2 (if present). Default to `["GET"]` if no methods specified.
3. For `@app.get("/path")`:
   - Extract method from group 1 (the HTTP verb).
   - Extract path from group 2.
4. Blueprint detection:
   - Track variable assignments like `bp = Blueprint("name", __name__)`.
   - Routes on blueprint objects use the blueprint's `url_prefix`.
   - Full path = blueprint prefix + route path.

**Current implementation note:** The existing `reFlaskRoute` in `python_extract.go` (line 82) only detects `@app.route(`. It does not extract the path or method. The upgrade adds path/method extraction and blueprint support.

**Blueprint instance tracking algorithm:**

Blueprint routes (`@bp.route("/path")`) require knowing which variable names hold `Blueprint` instances and what `url_prefix` each blueprint was created with. The algorithm:

1. **Scan for Blueprint assignments.** In each `.py` file, match:
   ```go
   var blueprintAssignPattern = regexp.MustCompile(
       `(\w+)\s*=\s*Blueprint\s*\(\s*["']([^"']+)["']\s*(?:,\s*__name__)?\s*(?:,\s*url_prefix\s*=\s*["']([^"']+)["'])?`)
   ```
   - Group 1: variable name (e.g., `bp`, `admin_bp`)
   - Group 2: blueprint name (e.g., `"admin"`)
   - Group 3: url_prefix (e.g., `"/admin"`) -- may be absent

2. **Build a variable-to-prefix map per file.** Store `{varName -> urlPrefix}` for the current file. If `url_prefix` was not in the `Blueprint()` constructor, check for a subsequent `app.register_blueprint(bp, url_prefix="/X")` call:
   ```go
   var registerBlueprintPattern = regexp.MustCompile(
       `app\.register_blueprint\s*\(\s*(\w+)\s*(?:,\s*url_prefix\s*=\s*["']([^"']+)["'])?`)
   ```

3. **Match decorator patterns against tracked names.** When a decorator like `@bp.route("/settings")` is found, extract the variable name `bp` from the decorator. Look it up in the variable-to-prefix map. If found, the full route path is `prefix + route_path` (e.g., `/admin/settings`).

4. **Fallback for unrecognized variable names.** If the decorator's variable name is not in the tracked map, still record the route but mark `prefix_unknown: true`. This handles cases where the Blueprint is defined in a different file and imported.

### 2.3 FastAPI (Python)

**Detection signals:**

| Signal | Location | Confidence |
|--------|----------|------------|
| `fastapi` in `requirements.txt` or `pyproject.toml` | Manifest | 0.90 |
| `from fastapi import` in source | Source file | 0.95 |
| `@app.get`, `@router.get` decorator | Source file | 0.90 |

**Route extraction patterns:**

```go
var fastAPIRoutePatterns = []*regexp.Regexp{
    // @app.get("/path")
    regexp.MustCompile(`@(?:app|router)\.(get|post|put|delete|patch)\s*\(\s*["']([^"']+)["']`),
    // @app.api_route("/path", methods=["GET"])
    regexp.MustCompile(`@(?:app|router)\.api_route\s*\(\s*["']([^"']+)["']\s*(?:,\s*methods\s*=\s*\[([^\]]+)\])?`),
}
```

**Route extraction algorithm:**

1. Scan for decorator patterns. Same algorithm as Flask for path/method extraction.
2. For `api_route`, extract both path and methods list.
3. Router prefix detection:
   - Track `app.include_router(router, prefix="/api/v1")`.
   - Full path = router prefix + route path.
   - If `include_router` is not found, the router's routes are relative to root.

### 2.4 Django (Python)

**Detection signals:**

| Signal | Location | Confidence |
|--------|----------|------------|
| `django` in `requirements.txt` or `pyproject.toml` | Manifest | 0.90 |
| `from django import` in source | Source file | 0.85 |
| `INSTALLED_APPS` assignment | `settings.py` | 0.90 |
| `models.Model` subclass | `models.py` | 0.85 |

**Route extraction patterns:**

Django routes are NOT decorator-based. They are declared in `urls.py` files as `urlpatterns` lists:

```python
urlpatterns = [
    path('admin/', admin.site.urls),
    path('api/users/', views.UserList.as_view()),
    re_path(r'^api/(?P<id>\d+)/$', views.user_detail),
]
```

**Extraction patterns:**

```go
var djangoRoutePatterns = []*regexp.Regexp{
    // path("route/", view)
    regexp.MustCompile(`path\s*\(\s*["']([^"']+)["']\s*,`),
    // re_path(r"^route/", view)
    regexp.MustCompile(`re_path\s*\(\s*r?["']([^"']+)["']\s*,`),
    // url(r"^route/", view) -- deprecated but still common
    regexp.MustCompile(`url\s*\(\s*r?["']([^"']+)["']\s*,`),
}
```

**Route extraction algorithm:**

1. Find all `urls.py` files by walking the directory tree.
2. In each `urls.py`, look for `urlpatterns = [`.
3. Extract each `path()` and `re_path()` call, taking the first string argument as the route pattern.
4. For `include("other_app.urls")`, note the relationship but do not recurse (the included module may not be in the same project).

**Model detection:**

```go
var djangoModelPattern = regexp.MustCompile(`class\s+(\w+)\s*\(\s*models\.Model\s*\)`)
```

Extract class name from group 1. The module path is derived from the file's location.

### 2.5 Express/Fastify (JavaScript/TypeScript)

**Detection signals:**

| Signal | Location | Confidence |
|--------|----------|------------|
| `express` in `package.json` dependencies | Manifest | 0.85 |
| `fastify` in `package.json` dependencies | Manifest | 0.90 |
| `app.get(`, `app.post(` in source | Source file | 0.80 |
| `fastify.register(` in source | Source file | 0.85 |

**Route extraction patterns:**

```go
var expressRoutePatterns = []*regexp.Regexp{
    // app.get("/path", handler)
    // NOTE: \x60 is the backtick character; Go raw strings cannot contain backticks.
    regexp.MustCompile(`(?:app|router|server)\.(get|post|put|delete|patch|use)\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`),
    // app.route("/path").get(handler).post(handler)
    regexp.MustCompile(`\.route\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]\s*)`),
    // fastify.get("/path", handler)
    regexp.MustCompile(`fastify\.(get|post|put|delete|patch)\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`),
}
```

**Route extraction algorithm:**

1. Find the Express/Fastify app initialization:
   - `const app = express()` or `const app = require('express')()`.
   - `const fastify = require('fastify')({})`.
2. Track variable name used for the app instance.
3. Scan for method calls on that variable: `<var>.(get|post|put|delete|patch)("path", ...)`.
4. For `app.use("/prefix", router)`:
   - Record the prefix.
   - Routes defined on the router are relative to this prefix.
5. Full path = middleware prefix + route path.

**Fastify plugin pattern:**

```go
// fastify.register(require("./routes"), { prefix: "/api" })
var fastifyRegisterPattern = regexp.MustCompile(
    `fastify\.register\s*\(\s*(?:require\s*\(\s*["']([^"']+)["']\s*\)|\w+)\s*(?:,\s*\{\s*prefix\s*:\s*["']([^"']+)["'])?`)
```

### 2.6 NestJS (TypeScript)

**Detection signals:**

| Signal | Location | Confidence |
|--------|----------|------------|
| `@nestjs/core` in `package.json` dependencies | Manifest | 0.90 |
| `@Module(` decorator in source | Source file | 0.85 |
| `@Controller(` decorator in source | Source file | 0.90 |

**Decorator extraction patterns:**

```go
var nestPatterns = struct {
    module     *regexp.Regexp
    controller *regexp.Regexp
    injectable *regexp.Regexp
    get        *regexp.Regexp
    post       *regexp.Regexp
    put        *regexp.Regexp
    delete     *regexp.Regexp
    patch      *regexp.Regexp
}{
    module:     regexp.MustCompile(`@Module\s*\(\s*\{`),
    controller: regexp.MustCompile(`@Controller\s*\(\s*["']([^"']+)["']`),
    injectable: regexp.MustCompile(`@Injectable\s*\(\s*\)`),
    get:        regexp.MustCompile(`@Get\s*\(\s*["']([^"']+)["']`),
    post:       regexp.MustCompile(`@Post\s*\(\s*["']([^"']+)["']`),
    put:        regexp.MustCompile(`@Put\s*\(\s*["']([^"']+)["']`),
    delete:     regexp.MustCompile(`@Delete\s*\(\s*["']([^"']+)["']`),
    patch:      regexp.MustCompile(`@Patch\s*\(\s*["']([^"']+)["']`),
}
```

**Route extraction algorithm:**

1. Find `@Controller("prefix")` decorators. Extract the path prefix.
2. For each method in the controller class, find `@Get("path")`, `@Post("path")`, etc.
3. HTTP method is determined by the decorator name.
4. Full path = controller prefix + method path.
5. Module graph:
   - Extract `@Module({ imports: [...], controllers: [...], providers: [...] })`.
   - `imports` = dependencies on other modules.
   - `controllers` = HTTP endpoint providers.
   - `providers` = injectable services.

### 2.7 React/Next.js (TypeScript/JavaScript)

**Detection signals:**

| Signal | Location | Confidence |
|--------|----------|------------|
| `next` in `package.json` dependencies | Manifest | 0.90 |
| `next.config.js` or `next.config.mjs` or `next.config.ts` | Root file | 0.95 |
| `pages/` directory exists | Directory | 0.85 |
| `app/` directory exists | Directory | 0.90 |
| `react` in `package.json` + `pages/` dir | Manifest + Directory | 0.80 |

**Page route extraction (file-based routing):**

Next.js uses file-system routing. No decorator or function-call patterns to parse.

**Pages Router extraction:**

1. Walk `pages/` directory.
2. For each file:
   - `pages/index.tsx` -> route `/`
   - `pages/about.tsx` -> route `/about`
   - `pages/users/[id].tsx` -> route `/users/:id` (dynamic segment)
   - `pages/api/health.tsx` -> API route `/api/health`
   - `pages/_app.tsx`, `pages/_document.tsx` -> skip (special Next.js files)
3. For each directory:
   - `pages/blog/` -> all files inside get prefix `/blog/`
4. Dynamic segments: `[param]` becomes `:param`. `[...slug]` becomes `:slug+` (catch-all).

**App Router extraction:**

1. Walk `app/` directory.
2. Look for `page.tsx` or `page.jsx` files (these are routes).
3. For each directory containing `page.tsx`:
   - `app/page.tsx` -> route `/`
   - `app/users/page.tsx` -> route `/users`
   - `app/users/[id]/page.tsx` -> route `/users/:id`
4. Special files (skip): `layout.tsx`, `loading.tsx`, `error.tsx`, `not-found.tsx`, `template.tsx`, `default.tsx`.
5. API routes: `route.ts` or `route.js` files under `app/`.
   - `app/api/health/route.ts` -> API route `/api/health`

**Output format:**

```
[
  {type: "page", path: "/", file: "app/page.tsx"},
  {type: "page", path: "/users/:id", file: "app/users/[id]/page.tsx"},
  {type: "api", path: "/api/health", file: "app/api/health/route.ts"},
  {type: "page", path: "/about", file: "pages/about.tsx"},
]
```

---

## 3. Module Boundary Detection Rules

### 3.1 Maven

**Algorithm:**

1. Read root `pom.xml`.
2. Check for `<modules>` section. Extract each `<module>X</module>` value.
3. For each module name `X`:
   - Read `X/pom.xml` (relative to root).
   - Check `<packaging>` element. If value is `pom`, this is an aggregator-only module -- skip it as a container candidate (but still traverse its sub-modules).
   - Otherwise, this is a C4 container candidate.
4. Module metadata:
   - **Name:** Last path segment of the module directory. E.g., `services/auth` -> `auth`.
   - **Display name:** `titleCase(lastSegment)`. E.g., `auth` -> `Auth`.
   - **Technology:** Read `<parent>` groupId/artifactId for framework hints. Check dependencies for Spring Boot, Micronaut, etc.
5. Inter-module relationships:
   - For each module's `pom.xml`, read `<dependencies>`.
   - If a `<dependency>` has `<groupId>` and `<artifactId>` matching another module's coordinates, record a dependency edge.
   - Maven coordinates are `{groupId}:{artifactId}`. Match against all modules' declared coordinates.

**Parsing (existing: see `parsePomXML()` and `parseModules()` in `java_extract.go`):**

```go
// Root pom.xml module extraction
func detectMavenModules(rootDir string) ([]ModuleBoundary, error) {
    rootPom := filepath.Join(rootDir, "pom.xml")
    if _, err := os.Stat(rootPom); err != nil {
        return nil, nil // not a Maven project
    }

    moduleNames := parseModules(rootPom)
    var boundaries []ModuleBoundary

    for _, name := range moduleNames {
        modulePom := filepath.Join(rootDir, name, "pom.xml")
        packaging := readPackaging(modulePom)
        if packaging == "pom" {
            // Aggregator: recurse into sub-modules
            subs, _ := detectMavenModules(filepath.Join(rootDir, name))
            boundaries = append(boundaries, subs...)
            continue
        }

        deps := readModuleDependencies(modulePom)
        boundaries = append(boundaries, ModuleBoundary{
            ID:           name,
            DisplayName:  titleCase(filepath.Base(name)),
            BuildSystem:  "maven",
            SourcePath:   name,
            Dependencies: deps,
        })
    }
    return boundaries, nil
}
```

### 3.2 Gradle

**Algorithm:**

1. Read `settings.gradle` or `settings.gradle.kts`.
2. Extract `include` directives:
   - Groovy: `include 'module-name'`, `include ":module:name"`, `include('module-name')`
   - Kotlin: `include("module-name")`, `include(":module:name")`
3. Convert Gradle project notation to file paths:
   - `:module` -> `module/`
   - `:module:sub` -> `module/sub/`
   - `module-name` (no colon) -> `module-name/`
4. Each included project = C4 container candidate.
5. Verify each path has a `build.gradle` or `build.gradle.kts` file.

**Parsing patterns:**

```go
var gradleIncludePatterns = []*regexp.Regexp{
    // Groovy: include 'module' or include ":module"
    regexp.MustCompile(`include\s+['"]([^'"]+)['"]`),
    // Kotlin DSL: include("module")
    regexp.MustCompile(`include\s*\(\s*["']([^"']+)["']\s*\)`),
}

// Convert Gradle project path to directory path
func gradlePathToDir(gradlePath string) string {
    // Remove leading colon if present
    path := strings.TrimPrefix(gradlePath, ":")
    // Replace colons with path separators
    return strings.ReplaceAll(path, ":", string(filepath.Separator))
}
```

### 3.3 npm/yarn/pnpm Workspaces

**Algorithm:**

1. Read root `package.json`.
2. Check for `workspaces` field:
   - Array form: `"workspaces": ["packages/*", "apps/*"]`
   - Object form: `"workspaces": { "packages": ["packages/*"] }`
3. Expand glob patterns:
   - For each workspace pattern, use `filepath.Glob()` to expand.
   - Verify each expanded directory contains its own `package.json`.
4. Check for pnpm: read `pnpm-workspace.yaml` if it exists:
   ```yaml
   packages:
     - 'packages/*'
     - 'apps/*'
   ```
5. Each workspace package = C4 container candidate.
6. Read each workspace's `package.json` for:
   - `name` field: used as the module ID.
   - `dependencies`: check if any dependency matches another workspace package name (internal dependency).

**Implementation note:** The existing `parseWorkspaces()` in `typescript_extract.go` (lines 308-329) handles array and object forms. It does NOT expand globs or check for pnpm-workspace.yaml. The upgrade adds glob expansion and pnpm support.

**Dependency note:** Parsing `pnpm-workspace.yaml` requires `gopkg.in/yaml.v3`. This dependency must be added to `go.mod` if not already present. The parsing function:

```go
import "gopkg.in/yaml.v3"

type pnpmWorkspace struct {
    Packages []string `yaml:"packages"`
}

func parsePnpmWorkspace(rootDir string) ([]string, error) {
    data, err := os.ReadFile(filepath.Join(rootDir, "pnpm-workspace.yaml"))
    if err != nil {
        return nil, err // not found is not an error for non-pnpm projects
    }
    var ws pnpmWorkspace
    if err := yaml.Unmarshal(data, &ws); err != nil {
        return nil, fmt.Errorf("parse pnpm-workspace.yaml: %w", err)
    }
    return ws.Packages, nil
}
```

### 3.4 Go

**Algorithm:**

1. Read `go.mod` to confirm this is a Go project and extract module path.
2. Walk the directory tree from the module root.
3. Classify directories by convention:
   - `cmd/` subdirectories: each is a deployable service. C4 container candidate.
     - `cmd/server/` -> container "server"
     - `cmd/api/` -> container "api"
   - `internal/` subdirectories: internal packages. C4 component candidates (not containers).
     - `internal/auth/` -> component "auth"
     - `internal/service/order/` -> component "order"
   - `pkg/` subdirectories: public packages. C4 component candidates.
     - `pkg/logger/` -> component "logger"
   - `api/` or `proto/`: API definitions (Protobuf, OpenAPI). Not containers.
4. Filter rules:
   - Skip directories named `test`, `tests`, `testdata`, `docs`, `scripts`, `examples`.
   - Require at least one `.go` file per directory to qualify as a module.
5. Container naming: use the directory name under `cmd/`. If there is only one directory under `cmd/`, use the module name instead.
6. Component naming: use the full path relative to `internal/` or `pkg/`, using `/` as separator.

**Implementation note:** The existing Go extractor (`go_extract.go`) builds the import graph via `go/packages`. Module boundary detection is a new capability that runs on top of the file tree, not the import graph.

### 3.5 Fallback (No Build System)

**Algorithm:**

1. If no `pom.xml`, `build.gradle`, `settings.gradle`, `package.json`, `go.mod`, `pnpm-workspace.yaml`, or `Cargo.toml` is found at the root, apply the fallback heuristic.
2. List top-level directories (depth 1).
3. For each directory:
   - Skip known non-code directories: `.git`, `node_modules`, `vendor`, `docs`, `doc`, `test`, `tests`, `__pycache__`, `.tox`, `dist`, `build`, `.next`, `target`, `bin`, `out`, `.idea`, `.vscode`.
   - Count source files (files with extensions matching `textExtensions` map from `filetree.go`).
   - Require at least 2 source files to qualify as a module boundary.
4. Each qualifying directory = C4 container candidate.
5. Naming: use directory name. If the name is generic (`src`, `lib`, `main`), look at sub-directories for more specific names.

**Confidence:** Low. Mark all fallback-detected modules with `confidence: 0.3` and `method: "heuristic"`.

---

## 4. API Surface Extraction

### 4.1 API Surface Types

Every API surface produces a record of this shape:

```go
type APISurface struct {
    Type        string   `json:"type"`        // "http_endpoint", "grpc_service", "graphql_resolver", "message_producer", "message_consumer", "exported_interface", "cli_command"
    Method      string   `json:"method,omitempty"` // HTTP method, gRPC method name, etc.
    Path        string   `json:"path,omitempty"`   // URL path, topic name, command name, etc.
    Handler     string   `json:"handler,omitempty"` // Function/method/class that handles it
    File        string   `json:"file"`             // Source file where defined
    ContainerID string   `json:"container_id,omitempty"` // Parent container (if known)
    Metadata    map[string]string `json:"metadata,omitempty"` // Framework-specific extras
}
```

### 4.2 HTTP Endpoint Extraction Per Framework

**Spring Boot:**

```
Input:  @GetMapping("/users/{id}") on method getUser() in class annotated @RequestMapping("/api")
Output: {method: "GET", path: "/api/users/{id}", handler: "UserController.getUser", file: "UserController.java"}
```

Algorithm:
1. Find class-level `@RequestMapping` or `@RestController` with optional path.
2. For each method annotated with `@XMapping`:
   - Method name from decorator type: Get -> GET, Post -> POST, Put -> PUT, Delete -> DELETE, Patch -> PATCH, Request -> ALL (or check method-level `method` attribute).
   - Path from annotation string argument.
   - Handler = `ClassName.methodName`.
3. Path parameters (`{id}`) are preserved as-is -- they are part of the API contract.

**Flask:**

```
Input:  @app.route("/api/users/<int:user_id>", methods=["GET"])
Output: {method: "GET", path: "/api/users/<int:user_id>", handler: "get_user", file: "routes.py"}
```

Algorithm:
1. Match decorator pattern. Extract path and methods.
2. Handler = the function name immediately following the decorator.
3. Flask path syntax (`<int:user_id>`) is preserved as-is.

**FastAPI:**

```
Input:  @app.get("/api/users/{user_id}")
Output: {method: "GET", path: "/api/users/{user_id}", handler: "get_user", file: "main.py"}
```

Algorithm: Same as Flask but path syntax uses `{param}` instead of `<type:param>`.

**Express:**

```
Input:  app.get("/api/users/:userId", getUser)
Output: {method: "GET", path: "/api/users/:userId", handler: "getUser", file: "routes.ts"}
```

Algorithm:
1. Match method call pattern. Extract HTTP method, path, and handler reference.
2. Express path syntax (`:param`) preserved as-is.

**NestJS:**

```
Input:  @Get(":id") on method findOne() in class @Controller("users")
Output: {method: "GET", path: "/users/:id", handler: "UsersController.findOne", file: "users.controller.ts"}
```

Algorithm: Same as Spring Boot (decorator-based).

**Next.js (App Router):**

```
Input:  File at app/api/users/[id]/route.ts
Output: {method: "ANY", path: "/api/users/[id]", handler: "route", file: "app/api/users/[id]/route.ts"}
```

Note: Next.js App Router exports named HTTP method handlers (`export async function GET(...)`). Scan the file for exported function names to determine which HTTP methods are supported.

**Next.js (Pages Router):**

```
Input:  File at pages/api/health.ts with export default function handler()
Output: {method: "ANY", path: "/api/health", handler: "default", file: "pages/api/health.ts"}
```

Note: Pages API routes handle all methods in a single handler. Method detection requires inspecting `req.method` inside the handler body, which is beyond static analysis scope.

### 4.3 Other API Surface Types

**gRPC service:**
- Parse `.proto` files.
- Extract `service Name { rpc Method(Request) returns (Response); }`.
- Output: `{type: "grpc_service", path: "ServiceName", method: "MethodName", file: "service.proto"}`.

**GraphQL:**
- Parse `.graphql` files for `type Query { ... }`, `type Mutation { ... }`, `type Subscription { ... }`.
- Output: `{type: "graphql_resolver", method: "Query.fieldName", path: "/graphql", file: "schema.graphql"}`.

**Message producer/consumer:**
- Kafka: scan for `kafka.produce(topic, ...)`, `@KafkaListener`, `producer.send(topic, ...)`.
- RabbitMQ: scan for `channel.publish(exchange, routingKey, ...)`, `@RabbitListener`.
- Output: `{type: "message_producer", path: "topic-name", file: "events.ts"}` or `{type: "message_consumer", path: "topic-name", file: "consumer.ts"}`.

**Exported interface (Go):**
- Extract from Go packages: any exported interface type with its method signatures.
- Output: `{type: "exported_interface", path: "pkg.Service", method: "DoSomething", file: "service.go"}`.

**CLI command:**
- Cobra (Go): `var rootCmd = &cobra.Command{Use: "app"}`.
- Click (Python): `@click.command()`, `@click.group()`.
- Output: `{type: "cli_command", path: "app subcommand", file: "cmd/root.go"}`.

---

## 5. Layer Detection Pattern Library

### 5.1 Layer Definitions

The following YAML defines the complete mapping of directory/file naming patterns to architectural layers. This replaces the simple `namingPatterns` map in `internal/architect/extract/filetree.go`.

```yaml
layers:
  presentation:
    dirs:
      - controller
      - controllers
      - handler
      - handlers
      - route
      - routes
      - web
      - api
      - rest
      - endpoint
      - endpoints
      - resource
      - resources
      - view
      - views
      - page
      - pages
      - screen
      - screens
      - presenter
      - presenters
      - action
      - actions
    files:
      - "*controller*"
      - "*handler*"
      - "*route*"
      - "*router*"
      - "*view*"
      - "*page*"
      - "*screen*"
      - "*presenter*"
      - "*action*"
      - "*resource*"
    annotations:
      - "@Controller"
      - "@RestController"
      - "@RequestMapping"
      - "@GetMapping"
      - "@PostMapping"
      - "@PutMapping"
      - "@DeleteMapping"
      - "@PatchMapping"
      - "@app.route"
      - "@app.get"
      - "@app.post"
      - "@blueprint.route"
      - "@router.get"
      - "@router.post"
    frameworks:
      django:
        - "views.py"
        - "urls.py"
        - "serializers.py"
      rails:
        - "app/controllers/"
      laravel:
        - "app/Http/Controllers/"

  business:
    dirs:
      - service
      - services
      - domain
      - business
      - logic
      - core
      - application
      - usecase
      - usecases
      - interactors
      - command
      - commands
      - query
      - queries
      - processor
      - processors
      - manager
      - managers
    files:
      - "*service*"
      - "*usecase*"
      - "*use_case*"
      - "*interactor*"
      - "*command*"
      - "*query*"
      - "*domain*"
      - "*processor*"
      - "*manager*"
      - "*logic*"
    annotations:
      - "@Service"
      - "@Component"
      - "@Injectable"
      - "@UseCase"
    frameworks:
      django:
        - "services.py"
        - "forms.py"
      rails:
        - "app/services/"

  data:
    dirs:
      - repository
      - repositories
      - data
      - db
      - database
      - persistence
      - dao
      - model
      - models
      - entity
      - entities
      - schema
      - schemas
      - store
      - stores
      - gateway
      - gateways
      - migration
      - migrations
    files:
      - "*repository*"
      - "*dao*"
      - "*entity*"
      - "*model*"
      - "*schema*"
      - "*store*"
      - "*gateway*"
      - "*persistence*"
      - "*migration*"
    annotations:
      - "@Repository"
      - "@Entity"
      - "@Table"
    frameworks:
      django:
        - "models.py"
      rails:
        - "app/models/"

  infrastructure:
    dirs:
      - config
      - configuration
      - infra
      - infrastructure
      - deploy
      - deployment
      - external
      - adapter
      - adapters
      - client
      - clients
      - messaging
      - queue
      - queues
      - cache
      - plugin
      - plugins
      - middleware
      - middlewares
      - util
      - utils
      - helper
      - helpers
      - common
      - shared
    files:
      - "*config*"
      - "*adapter*"
      - "*client*"
      - "*messaging*"
      - "*middleware*"
      - "*plugin*"
      - "*util*"
      - "*helper*"
      - "*cache*"
    annotations:
      - "@Configuration"
      - "@Bean"
      - "@Component" # ambiguous: could be business or infra
```

### 5.2 Layer Detection Algorithm

The algorithm classifies each source file into an architectural layer. It uses a weighted scoring system to resolve conflicts.

**Step 1: Directory name scoring (weight: 0.5)**

For each source file at path `p`, extract the directory components. For each directory component, check against all layer `dirs` patterns. If a directory component exactly matches (case-insensitive) a pattern in layer L, add `0.5` to L's score.

```
Example: src/main/java/com/example/controller/UserController.java
Directory components: ["src", "main", "java", "com", "example", "controller"]
"controller" matches presentation layer -> presentation += 0.5
```

**Step 2: File name scoring (weight: 0.3)**

For the file's basename (without extension), check against all layer `files` glob patterns. Glob matching uses `filepath.Match` semantics where `*` matches any sequence of non-separator characters.

```
Example: UserController.java
Basename (no ext): "UserController"
Matches "*controller*" in presentation -> presentation += 0.3
```

**Step 3: Annotation scoring (weight: 0.2)**

Scan the file content for annotation/decorator patterns. For each annotation found, check against all layer `annotations` patterns. This requires reading the file, so it is only performed if steps 1-2 did not produce a clear winner (score gap < 0.2).

```
Example: file contains "@RestController"
Matches presentation layer -> presentation += 0.2
```

**Step 4: Framework-specific scoring (weight: 0.2, additive)**

If a framework has been detected for this project, check the file path against framework-specific patterns.

```
Example: Django project, file at myapp/views.py
"views.py" matches presentation in django framework -> presentation += 0.2
```

**Step 5: Conflict resolution**

1. The layer with the highest score wins.
2. If two layers tie:
   - Directory name match wins over file name match.
   - If still tied, use the priority order: `presentation > business > data > infrastructure`.
3. If no layer scores > 0.0, classify as `unclassified` with confidence 0.3.
4. The confidence value is the winner's score, capped at 1.0.

**Language-specific overrides:**

The generic layer patterns above apply cross-language defaults, but some directory names have language-specific meanings that must be overridden:

| Directory | Generic classification | Go override | Rationale |
|-----------|----------------------|-------------|-----------|
| `api/` | `presentation` | `infrastructure` or `unclassified` | In Go projects, `api/` typically holds Protobuf/OpenAPI definitions or generated code, not HTTP handlers. Go HTTP handlers live under `internal/` with naming like `internal/handler/`. |
| `internal/` | (no generic match) | Scan sub-directories | `internal/` is a Go-specific convention; its sub-directories (`internal/handler/`, `internal/service/`, `internal/repository/`) carry the layer signal, not `internal/` itself. |
| `pkg/` | (no generic match) | `infrastructure` | `pkg/` in Go denotes public reusable packages, typically utility/infrastructure code. |

Override mechanism: when a Go project is detected (presence of `go.mod`), the layer detection function applies a `languageOverrides` map before Step 5. If an override matches, it replaces the generic classification for that directory component.

**Step 6: Output**

```go
type LayerAssignment struct {
    FilePath    string  `json:"file_path"`
    Layer       string  `json:"layer"`       // "presentation", "business", "data", "infrastructure", "unclassified"
    Score       float64 `json:"score"`       // winning score
    Confidence  float64 `json:"confidence"`  // score capped at 1.0
    Evidence    string  `json:"evidence"`    // human-readable reason: "directory 'controller' matched presentation"
}
```

### 5.3 Layer Pattern Implementation

```go
// layerPattern defines a single layer's detection patterns.
type layerPattern struct {
    Name        string
    DirSet      map[string]bool  // lowercased directory names
    FileGlobs   []string         // glob patterns for filenames (without extension)
    Annotations []string         // annotation/decorator strings to scan for
}

// allLayers is the ordered list of layers to check.
// Order matters for tie-breaking: first layer wins.
var allLayers = []layerPattern{
    {
        Name: "presentation",
        DirSet: stringSet([]string{
            "controller", "controllers", "handler", "handlers",
            "route", "routes", "web", "api", "rest",
            "endpoint", "endpoints", "resource", "resources",
            "view", "views", "page", "pages", "screen", "screens",
            "presenter", "presenters", "action", "actions",
        }),
        FileGlobs: []string{
            "*controller*", "*handler*", "*route*", "*router*",
            "*view*", "*page*", "*screen*", "*presenter*",
            "*action*", "*resource*",
        },
        Annotations: []string{
            "@Controller", "@RestController", "@RequestMapping",
            "@app.route", "@app.get", "@app.post",
        },
    },
    {
        Name: "business",
        DirSet: stringSet([]string{
            "service", "services", "domain", "business", "logic",
            "core", "application", "usecase", "usecases",
            "interactors", "command", "commands", "query", "queries",
            "processor", "processors", "manager", "managers",
        }),
        FileGlobs: []string{
            "*service*", "*usecase*", "*use_case*", "*interactor*",
            "*command*", "*query*", "*domain*", "*processor*", "*manager*",
        },
        Annotations: []string{
            "@Service", "@Injectable",
        },
    },
    {
        Name: "data",
        DirSet: stringSet([]string{
            "repository", "repositories", "data", "db", "database",
            "persistence", "dao", "model", "models", "entity", "entities",
            "schema", "schemas", "store", "stores", "gateway", "gateways",
            "migration", "migrations",
        }),
        FileGlobs: []string{
            "*repository*", "*dao*", "*entity*", "*model*",
            "*schema*", "*store*", "*gateway*", "*persistence*",
        },
        Annotations: []string{
            "@Repository", "@Entity", "@Table",
        },
    },
    {
        Name: "infrastructure",
        DirSet: stringSet([]string{
            "config", "configuration", "infra", "infrastructure",
            "deploy", "deployment", "external", "adapter", "adapters",
            "client", "clients", "messaging", "queue", "queues",
            "cache", "plugin", "plugins", "middleware", "middlewares",
            "util", "utils", "helper", "helpers", "common", "shared",
        }),
        FileGlobs: []string{
            "*config*", "*adapter*", "*client*", "*messaging*",
            "*middleware*", "*plugin*", "*util*", "*helper*", "*cache*",
        },
        Annotations: []string{
            "@Configuration", "@Bean",
        },
    },
}

func stringSet(items []string) map[string]bool {
    m := make(map[string]bool, len(items))
    for _, s := range items {
        m[s] = true
    }
    return m
}
```

### 5.4 Layer Detection Function

```go
// ClassifyLayer determines the architectural layer for a source file.
// Parameters:
//   - relPath: file path relative to repository root
//   - content: file contents (may be empty string to skip annotation scanning)
//   - detectedFramework: name of detected framework (e.g., "django", "rails") or ""
//
// Returns a LayerAssignment with layer name, score, confidence, and evidence.
func ClassifyLayer(relPath, content, detectedFramework string) LayerAssignment {
    scores := make(map[string]float64)
    evidence := make(map[string][]string)

    dir := filepath.Dir(relPath)
    base := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
    lowerBase := strings.ToLower(base)

    // Step 1: directory name scoring (weight 0.5)
    parts := strings.Split(filepath.ToSlash(dir), "/")
    for _, part := range parts {
        lower := strings.ToLower(part)
        for _, layer := range allLayers {
            if layer.DirSet[lower] {
                scores[layer.Name] += 0.5
                evidence[layer.Name] = append(evidence[layer.Name],
                    fmt.Sprintf("directory '%s' matched %s", part, layer.Name))
            }
        }
    }

    // Step 2: file name scoring (weight 0.3)
    for _, layer := range allLayers {
        for _, glob := range layer.FileGlobs {
            matched, _ := filepath.Match(glob, lowerBase)
            if matched {
                scores[layer.Name] += 0.3
                evidence[layer.Name] = append(evidence[layer.Name],
                    fmt.Sprintf("filename '%s' matched %s", base, layer.Name))
            }
        }
    }

    // Step 3: annotation scoring (weight 0.2) -- only if content provided
    // and no clear winner yet
    bestScore := 0.0
    for _, s := range scores {
        if s > bestScore {
            bestScore = s
        }
    }
    if content != "" && bestScore < 0.5 { // only scan annotations if uncertain
        for _, layer := range allLayers {
            for _, ann := range layer.Annotations {
                if strings.Contains(content, ann) {
                    scores[layer.Name] += 0.2
                    evidence[layer.Name] = append(evidence[layer.Name],
                        fmt.Sprintf("annotation '%s' matched %s", ann, layer.Name))
                }
            }
        }
    }

    // Step 4: framework-specific scoring (weight 0.2)
    if detectedFramework != "" {
        basename := filepath.Base(relPath)
        frameworkPatterns, ok := frameworkFilePatterns[detectedFramework]
        if ok {
            for layerName, files := range frameworkPatterns {
                for _, f := range files {
                    if basename == f {
                        scores[layerName] += 0.2
                        evidence[layerName] = append(evidence[layerName],
                            fmt.Sprintf("framework '%s' file '%s' matched %s", detectedFramework, f, layerName))
                    }
                }
            }
        }
    }

    // Step 5: determine winner
    winner := "unclassified"
    winnerScore := 0.0
    for _, layer := range allLayers {
        s := scores[layer.Name]
        if s > winnerScore {
            winnerScore = s
            winner = layer.Name
        }
    }

    confidence := winnerScore
    if confidence > 1.0 {
        confidence = 1.0
    }
    if winner == "unclassified" {
        confidence = 0.3
    }

    var evidenceStr string
    if ev := evidence[winner]; len(ev) > 0 {
        evidenceStr = ev[0] // primary evidence
    } else {
        evidenceStr = "no matching patterns"
    }

    return LayerAssignment{
        FilePath:   relPath,
        Layer:      winner,
        Score:      winnerScore,
        Confidence: confidence,
        Evidence:   evidenceStr,
    }
}

// frameworkFilePatterns maps framework names to their layer-specific file conventions.
var frameworkFilePatterns = map[string]map[string][]string{
    "django": {
        "presentation": {"views.py", "urls.py", "serializers.py"},
        "business":     {"services.py", "forms.py"},
        "data":         {"models.py"},
    },
    "rails": {
        "presentation": {"app/controllers/"},
        "business":     {"app/services/"},
        "data":         {"app/models/"},
    },
    "spring": {
        "presentation": {}, // handled by annotations
        "business":     {},
        "data":         {},
    },
}
```

---

## 6. Integration with Existing Code

### 6.1 File Changes Summary

The following existing files need modification to integrate the specifications above:

| File | Change |
|------|--------|
| `internal/architect/extract/python_extract.go` | Add tree-sitter query support alongside existing regex. Add route path extraction for Flask/FastAPI/Django. Add blueprint prefix tracking. |
| `internal/architect/extract/java_extract.go` | Add tree-sitter query support. Add annotation-level scanning (not just imports). Add HTTP endpoint extraction from `@XMapping` annotations. |
| `internal/architect/extract/typescript_extract.go` | Add tree-sitter query support. Add path alias resolution. Add Next.js file-based route extraction. Add Fastify plugin prefix tracking. |
| `internal/architect/extract/adapters.go` | Add `LayerAssignment` and `APISurface` to `ProfileFragment` conversion. |
| `internal/architect/extract/filetree.go` | Replace simple `namingPatterns` with full layer detection algorithm. |
| `internal/architect/extract/deps.go` | Add module boundary detection per build system. |
| `internal/architect/profile.go` | Add `Layers`, `APISurfaces`, `ModuleBoundaries` fields to `ProfileFragment` and `CodebaseProfile`. |
| `internal/architect/types.go` | Add `LayerAssignment`, `APISurface`, `ModuleBoundary` types. |

### 6.2 New Files

| File | Purpose |
|------|---------|
| `internal/architect/extract/layers.go` | Layer detection algorithm (Section 5). |
| `internal/architect/extract/api_surface.go` | API surface extraction per framework (Section 4). |
| `internal/architect/extract/modules.go` | Module boundary detection per build system (Section 3). |

### 6.3 Tree-Sitter Integration Path

The existing codebase uses regex-based extraction. The upgrade path is:

1. **Phase 1 (immediate -- fix existing regex extractors + define tree-sitter query contracts):**
   - Fix existing regex extractors for accuracy issues identified in code review.
   - Add route extraction, layer detection, module boundaries, and API surface extraction on top of existing regex import results.
   - Define tree-sitter query contracts (the `.scm` queries in Section 1) as interface specifications. These are NOT implemented in Phase 1 but serve as the contract that Phase 2 must satisfy.
   - All tree-sitter queries in Section 1 are marked "Phase 2 implementation, Phase 1 contract definition."

2. **Phase 2 (tree-sitter integration):** Add `go-tree-sitter` bindings. Create `.scm` query files under `internal/architect/extract/queries/` using the corrected queries from Section 1. Add a `TreeSitterExtractor` that wraps the query engine. Run both regex and tree-sitter, prefer tree-sitter results when available, fall back to regex.

3. **Phase 3 (regex removal):** Once tree-sitter covers all regex patterns, deprecate regex extractors.

This phased approach means Phase 1 is implementable immediately without adding the tree-sitter C dependency.

---

## 7. Performance Requirements

Extractors operate on potentially large repositories (tens of thousands of files). The following performance SLAs ensure the system remains usable as a CLI tool that runs on developer machines.

### 7.1 Per-File SLAs

| Metric | Limit | Rationale |
|--------|-------|-----------|
| Max parse time per file | **5 seconds** | Any single file taking longer indicates an exponential regex or unbounded lookahead. Fail fast and skip the file. |
| Max memory per file parse | **50 MB** | Prevents pathological files (e.g., minified JS bundles) from consuming all memory. |

### 7.2 Per-Repository SLAs

| Metric | Limit | Rationale |
|--------|-------|-----------|
| Max total memory for 10K-file repo | **1 GB** | Must fit comfortably on a developer laptop with 8 GB RAM alongside the IDE and browser. |
| Max total extraction time for 10K-file repo | **5 minutes** | CLI tools must return results fast enough for interactive use. |
| Scaling | **Linear** | Extraction time for 20K files must be at most 2x the time for 10K files. No quadratic or worse behavior. |

### 7.3 Per-Extractor SLAs

| Metric | Limit | Rationale |
|--------|-------|-----------|
| Timeout per extractor (per language) | **30 seconds** | If a single language extractor hangs, the overall run still completes. |
| Max concurrency | **`runtime.NumCPU()`** | CPU-bound parsing benefits from parallelism up to core count. No benefit beyond that; avoids thrashing. |

### 7.4 Enforcement Mechanism

```go
type ExtractorConfig struct {
    MaxFileParseTime   time.Duration // default: 5s
    MaxFileMemory      int64         // default: 50 MB
    MaxExtractorTimeout time.Duration // default: 30s
    MaxConcurrency      int           // default: runtime.NumCPU()
    MaxTotalTime        time.Duration // default: 5 minutes
}

// Each extractor call is wrapped with a context timeout:
func extractWithTimeout(ctx context.Context, cfg ExtractorConfig, fn func() error) error {
    ctx, cancel := context.WithTimeout(ctx, cfg.MaxFileParseTime)
    defer cancel()
    done := make(chan error, 1)
    go func() { done <- fn() }()
    select {
    case err := <-done:
        return err
    case <-ctx.Done():
        return fmt.Errorf("extractor timed out after %s", cfg.MaxFileParseTime)
    }
}
```

---

## 8. Accuracy Expectations

These are the target accuracy numbers for Phase 1 (regex-based, with the enhancements specified in this document):

| Capability | Go | Python | Java | TypeScript | SQL |
|------------|-----|--------|------|------------|-----|
| Import extraction | 90-95% (native) | 55-65% | 70-75% | 55-65% | N/A |
| Framework detection | 80-90% (Gin, Echo) | 85-90% (Flask/FastAPI/Django) | 85-90% (Spring Boot) | 85-90% (Express/NestJS/Next.js) | N/A |
| Route extraction | 75-85% (Gin, Echo) | 70-80% (Flask/FastAPI) | 75-85% (Spring Boot) | 70-80% (Express) / 90%+ (Next.js file-based) | N/A |
| Module boundary | 85-95% (cmd/internal/pkg) | 60-70% (no standard) | 85-90% (Maven modules) | 80-90% (workspaces) | N/A |
| Layer detection | 70-80% | 65-75% | 75-85% (Spring annotations) | 65-75% | N/A |

Numbers in parentheses indicate the framework or convention that achieves the high end of the range. Projects without clear conventions fall to the low end.

---

## 9. New Types (Exact Struct Definitions)

### 9.1 Types to add to `internal/architect/types.go`

```go
// ModuleBoundary represents a deployable unit within the repository.
type ModuleBoundary struct {
    ID           string   `json:"id"`
    DisplayName  string   `json:"display_name"`
    BuildSystem  string   `json:"build_system"` // "maven", "gradle", "npm", "go", "python", "fallback"
    SourcePath   string   `json:"source_path"`
    Dependencies []string `json:"dependencies,omitempty"` // IDs of other ModuleBoundaries this depends on
    Language     string   `json:"language,omitempty"`
    Confidence   float64  `json:"confidence"`
    Method       string   `json:"method,omitempty"` // "build_system", "convention", "heuristic"
}

// APISurface describes a single entry point exposed by a component.
type APISurface struct {
    Type        string            `json:"type"`         // "http_endpoint", "grpc_service", "graphql_resolver", "message_producer", "message_consumer", "exported_interface", "cli_command"
    Method      string            `json:"method,omitempty"`
    Path        string            `json:"path,omitempty"`
    Handler     string            `json:"handler,omitempty"`
    File        string            `json:"file"`
    ContainerID string            `json:"container_id,omitempty"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}

// LayerAssignment records the architectural layer classification of a source file.
type LayerAssignment struct {
    FilePath   string  `json:"file_path"`
    Layer      string  `json:"layer"`       // "presentation", "business", "data", "infrastructure", "unclassified"
    Score      float64 `json:"score"`
    Confidence float64 `json:"confidence"`
    Evidence   string  `json:"evidence"`
}

// CIInfo describes CI/CD pipeline metadata.
type CIInfo struct {
    Platform string   `json:"platform"` // "github_actions", "gitlab_ci", "jenkins", "circleci"
    Jobs     []string `json:"jobs,omitempty"`
    Triggers []string `json:"triggers,omitempty"` // "push", "pull_request", "schedule", "manual"
}

// PackageBoundary represents a Python package or namespace package.
type PackageBoundary struct {
    Type string `json:"type"` // "python_package", "namespace_package"
    Path string `json:"path"`
    Name string `json:"name"`
}
```

### 9.2 Fields to add to `ProfileFragment` in `internal/architect/profile.go`

```go
type ProfileFragment struct {
    // ... existing fields ...
    Layers          []LayerAssignment  `json:"layers,omitempty"`
    APISurfaces     []APISurface      `json:"api_surfaces,omitempty"`
    ModuleBoundaries []ModuleBoundary `json:"module_boundaries,omitempty"`
    CI              *CIInfo           `json:"ci,omitempty"`
}
```

### 9.3 Fields to add to `CodebaseProfile` in `internal/architect/profile.go`

```go
type CodebaseProfile struct {
    // ... existing fields ...
    Layers          []LayerAssignment  `json:"layers,omitempty"`
    APISurfaces     []APISurface      `json:"api_surfaces,omitempty"`
    ModuleBoundaries []ModuleBoundary `json:"module_boundaries,omitempty"`
}
```

---

## 10. SecurityFilter Enhancement (Phase A-1)

### Current state

`internal/architect/security.go` has 5 secret patterns and 2 PII rules.

### New secret patterns to add to `NewSecurityFilter()`

```go
{re: regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`), typ: "stripe_live_key"},
{re: regexp.MustCompile(`eyJ[A-Za-z0-9-_]{20,}\.eyJ[A-Za-z0-9-_]{20,}`), typ: "jwt_token"},
{re: regexp.MustCompile(`xox[baprs]-[0-9]{10,}-[0-9]{10,}-[0-9a-zA-Z]{24,}`), typ: "slack_token"},
{re: regexp.MustCompile(`//[^/@\s]+:[^/@\s]+@`), typ: "connection_string_credentials"},
```

### Enhanced PII rules

Replace the existing path-scrubbing regexes:

```go
var reUserPath = regexp.MustCompile(`(?:/Users/|/home/)[^/]+`)
var reWindowsPath = regexp.MustCompile(`C:\\Users\\[^\\]+`)
```

Update `sanitizeString()` to also apply `reWindowsPath`:

```go
func (sf *SecurityFilter) sanitizeString(s string) string {
    // 1. Redact secrets.
    for _, p := range sf.patterns {
        s = p.re.ReplaceAllStringFunc(s, func(match string) string {
            return "[REDACTED:" + p.typ + "]"
        })
    }
    // 2. Scrub user paths (Unix).
    s = reUserPath.ReplaceAllString(s, "/Users/[REDACTED]")
    // 3. Scrub user paths (Windows).
    s = reWindowsPath.ReplaceAllString(s, "C:\\Users\\[REDACTED]")
    // 4. Hash internal package names.
    s = reInternalPkg.ReplaceAllStringFunc(s, func(match string) string {
        parts := strings.Split(match, ".")
        if len(parts) < 3 { return match }
        return "pkg." + parts[len(parts)-1]
    })
    return s
}
```

### Changed signature

```go
// Sanitize returns (sanitized_copy, SecretsFound_report).
func (sf *SecurityFilter) Sanitize(profile *CodebaseProfile) (*CodebaseProfile, *SecretsFound)
```

The existing signature returns only `*CodebaseProfile`. Update callers.

### Testing

| Test case | Input | Expected |
|-----------|-------|----------|
| AWS key | `"key=AKIAIOSFODNN7EXAMPLE"` | Contains `[REDACTED:aws_key]` |
| JWT | `"Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.sig"` | Contains `[REDACTED:jwt_token]` |
| Stripe live key | `"sk_live_abcdefghijklmnopqrstuvwxyz123456"` | Contains `[REDACTED:stripe_live_key]` |
| Connection string | `"postgres://user:pass@host:5432/db"` | Contains `[REDACTED:connection_string_credentials]` |
| Unix user path | `"/Users/alice/projects/app"` | `/Users/[REDACTED]/projects/app` |
| Windows path | `"C:\\Users\\bob\\code\\app"` | `C:\Users\[REDACTED]\code\app` |
| No secrets | `"package main\nfunc main() {}"` | Unchanged, SecretsFound.Count == 0 |

---

## 11. FileTreeExtractor Enhancement (Phase A-2)

### Missing field population

The existing `FileTreeInfo` struct has `TopLevel []string` and `NamingPatterns map[string]int` fields that are never populated. Fix this.

### Algorithm for TopLevel

```
1. During WalkDir, when depth == 1 (rel has no path separator or exactly one):
   - If d.IsDir(): append d.Name() to topLevelDirs
   - Else: append d.Name() to topLevelFiles
2. Sort both slices.
3. TopLevel = topLevelDirs + topLevelFiles.
```

### Algorithm for NamingPatterns

```
1. Replace the "seen" map with a counter map[string]int.
2. On each naming pattern match, increment the counter.
3. Convert to map[string]int in the return value.
```

### Performance guard

```go
const maxLineCountFileSize = 2 * 1024 * 1024 // 2 MB

// Before calling countLines:
info, _ := d.Info()
if info != nil && info.Size() > maxLineCountFileSize {
    return nil // skip line counting for very large files
}
```

---

## 12. DependencyManifestParser Enhancement (Phase A-2)

### Enhancement: Parse go.mod properly

Replace the line-scanning approach with proper parsing:

```go
type GoModInfo struct {
    ModulePath string
    GoVersion  string
    Requires   []GoModRequire
}

type GoModRequire struct {
    Path    string
    Version string
    Indirect bool
}

func parseGoMod(path string) (*GoModInfo, error) {
    // Scan lines:
    // "module github.com/org/repo" -> ModulePath
    // "go 1.22" -> GoVersion
    // "require (" block or single require lines -> Requires
    // "// indirect" comment -> Indirect = true
}
```

### Enhancement: Parse package.json properly

Replace the fragile line-scanning in `countPackageJSON` with `encoding/json`:

```go
type packageJSONDeps struct {
    Dependencies    map[string]string `json:"dependencies"`
    DevDependencies map[string]string `json:"devDependencies"`
}

func parsePackageJSONDeps(path string) ([]TSDependency, error) {
    data, err := os.ReadFile(path)
    if err != nil { return nil, err }
    var pkg packageJSONDeps
    if err := json.Unmarshal(data, &pkg); err != nil { return nil, err }
    var deps []TSDependency
    for name, ver := range pkg.Dependencies {
        deps = append(deps, TSDependency{Name: name, Version: ver, Dev: false})
    }
    for name, ver := range pkg.DevDependencies {
        deps = append(deps, TSDependency{Name: name, Version: ver, Dev: true})
    }
    return deps, nil
}
```

### Cross-manifest correlation

After all manifests are parsed, correlate dependency names:

```
1. Collect all ParsedDep slices keyed by manifest file.
2. For each dep name, count how many manifests contain it -> FoundIn.
3. Map to NotableDep{Name, FoundIn, Signal}.
4. Use the existing notableSignals map in deps.go for signal classification.
```

### Error handling

| Error | Severity | Action |
|-------|----------|--------|
| Unparseable manifest | Non-fatal | Log warning, skip file |
| Missing manifest file | Non-fatal | Normal, skip silently |
| File permission error | Non-fatal | Log warning, skip |

---

## 13. InfraExtractor Enhancement (Phase A-3)

### New: Serverless detection

```go
var serverlessIndicators = []struct {
    Filename string
    Kind     string
}{
    {"serverless.yml", "serverless_framework"},
    {"serverless.yaml", "serverless_framework"},
    {"vercel.json", "vercel"},
    {"netlify.toml", "netlify"},
    {"sam.yaml", "aws_sam"},
    {"template.yaml", "aws_sam"},
}
```

Algorithm: during WalkDir, check basename against `serverlessIndicators`. If matched, set `DeploymentType` to `"serverless"`.

### New: Helm chart detection

```go
func isHelmChart(name string) bool {
    return name == "Chart.yaml" || name == "Chart.yml"
}
```

Parse `Chart.yaml` for `name`, `version`, `appVersion`. Create `ContainerInfo` with `Type="helm_chart"`.

### New: DeploymentType priority

```
Priority: kubernetes > serverless > helm > docker-compose > bare
```

### New: CI pipeline analysis

For GitHub Actions workflows, parse `on:` for triggers and `jobs:` for job names. Populate `CIInfo{Platform, Jobs, Triggers}` in the fragment.

### Testing

| Test case | Input | Expected |
|-----------|-------|----------|
| Multi-stage Dockerfile | 3 FROM lines | 3 BaseImages, ContainerInfo created |
| compose depends_on | api depends on db | ServiceDep{From:"api", To:"db"} |
| serverless.yml | functions section | DeploymentType="serverless" |
| Helm Chart.yaml | name, version | ContainerInfo Type="helm_chart" |

---

## 14. GeneratedCodeDetector Enhancement (Phase A-4)

### New filename patterns

```go
"*.pb.ts":          "protobuf_generated",
"*_pb2.py":         "protobuf_generated",
"*_pb2_grpc.py":    "protobuf_generated",
"*.graphql.ts":     "graphql_codegen",
"*.generated.go":   "autogenerated",
"*.generated.java": "autogenerated",
"*.min.js":         "minified",
"*.min.css":        "minified",
```

### New header markers

```go
"# Generated by",
"/* eslint-disable */",
"// @generated",
"<auto-generated>",
```

### Size-based heuristic

After all pattern/header checks, if a file is >500KB and the first 1KB contains no
function/class definitions (no `func `, `class `, `def `, `function ` keywords),
flag as `"likely_generated:large_file"`.

---

## 15. CodebaseProfile Assembly (Phase A-5)

### Algorithm

```
func AssembleProfile(fragments []*architect.ProfileFragment) *architect.CodebaseProfile

1. Initialize CodebaseProfile with zero values.
2. For each fragment:
   a. Languages: append and dedup.
   b. Dependencies: merge DependencyInfo slices.
   c. ImportGraph: merge nodes, edges, clusters, circular deps.
   d. Infra: merge containers, services, resources.
   e. FileTree: take the first non-nil FileTree.
   f. Specs: append and dedup by path.
   g. SQLAnalysis: merge tables, FKs, views, indexes, ORM models.
   h. GitAnalysis: take the first non-nil GitAnalysis.
   i. Metrics: sum/merge fields.
   j. Generated: append all generated files.
   k. Layers: append all layer assignments.
   l. APISurfaces: append all API surfaces.
   m. ModuleBoundaries: append and dedup by ID.
   n. CI: take the first non-nil CIInfo.
3. Compute derived metrics:
   a. TestRatio = test file LOC / total LOC
   b. LanguagesCount = len(unique primary languages)
   c. ContainersDetected = len(Infra.Containers)
   d. ComponentsDetected = len(ImportGraph.Clusters)
   e. ContractsDiscovered = len(Specs)
   f. GeneratedExcluded = len(Generated)
4. Return assembled profile.
```

### Merge rules for ImportGraph

```
- Nodes: sum all node counts.
- Edges: sum all edge counts.
- Clusters: append, dedup by ID.
- CircularDependencies: append, dedup by (A, B) pair.
- ExtractionMethod: comma-separated if multiple methods.
- AccuracyEstimate: weighted average by node count.
```

---

## 16. Evaluation Harness Framework (Phase A-6)

### Test fixture directory structure

```
testdata/
  go/
    simple-service/       # 3 packages, no cycles
    monorepo/             # go.workspace with 2 modules
    grpc-service/         # gRPC project with generated files
  python/
    flask-app/            # Flask with routes and blueprints
    fastapi-app/          # FastAPI with decorators
    django-project/       # Django with apps, models, urls
  java/
    spring-boot/          # Spring Boot with @RestController
    multi-module-maven/   # Multi-module Maven project
  typescript/
    nextjs-app/           # Next.js with app/ directory
    express-api/          # Express API with routes
    nestjs-app/           # NestJS with decorators
    monorepo/             # pnpm workspace with 3 packages
  sql/
    ecommerce-schema/     # 15+ tables with FKs, migrations, views
```

### Harness test case definition

```go
type ExtractorTest struct {
    Name       string
    FixtureDir string            // relative to testdata/
    Extractor  architect.Extractor
    Assertions []Assertion
}

type Assertion struct {
    Field    string      // dot-path into ProfileFragment
    Op       string      // "eq", "gte", "lte", "contains", "not_contains"
    Expected interface{}
}
```

Example:

```go
ExtractorTest{
    Name:       "go_simple_service_import_count",
    FixtureDir: "go/simple-service",
    Extractor:  extract.GoAdapter{},
    Assertions: []Assertion{
        {Field: "ImportGraph.Nodes", Op: "gte", Expected: 3},
        {Field: "ImportGraph.CircularDependencies", Op: "eq", Expected: 0},
    },
}
```

### Accuracy measurement with golden files

```go
type AccuracyReport struct {
    Extractor    string
    FixtureDir   string
    Fields       []FieldAccuracy
    Precision    float64 // TP / (TP + FP) -- of detected items, how many are correct
    Recall       float64 // TP / (TP + FN) -- of actual items, how many were detected
    F1           float64 // 2 * (Precision * Recall) / (Precision + Recall)
    OverallScore float64
}

type FieldAccuracy struct {
    Field    string
    Expected interface{}
    Actual   interface{}
    Score    float64 // 0.0-1.0
}
```

Where TP = true positive (correctly detected item), FP = false positive (detected but incorrect), FN = false negative (missed item).

**Test assertion format for precision/recall/F1:**

```go
// Example assertions using the new metrics:
ExtractorTest{
    Name:       "go_simple_service_accuracy",
    FixtureDir: "go/simple-service",
    Extractor:  extract.GoAdapter{},
    Assertions: []Assertion{
        {Field: "ImportGraph.Nodes", Op: "gte", Expected: 3},
        {Field: "ImportGraph.CircularDependencies", Op: "eq", Expected: 0},
    },
}

// Precision/recall/F1 are computed by the harness when golden files are present:
//   - Compare extractor output against golden file fields.
//   - Each field match = TP, extra field = FP, missing field = FN.
//   - Assert: Precision >= 0.8, Recall >= 0.8, F1 >= 0.8 (thresholds per extractor).
```

Golden files live at `testdata/<fixture>/<extractor>.golden.json`. Example golden file for `go/simple-service/go.golden.json`:

```json
{
  "import_graph": {
    "nodes": 4,
    "edges": 5,
    "circular_dependencies": [],
    "accuracy_estimate": 0.93
  },
  "languages": [{"primary": "go", "all": ["go"]}]
}
```

---

## 17. Go Extractor Enhancement (Phase A-7)

### Accuracy target: 90-95%

Already achieved via `go/packages`. Enhancements target edge cases.

### go.work support

```go
func detectGoWork(dir string) ([]string, error) {
    data, err := os.ReadFile(filepath.Join(dir, "go.work"))
    if err != nil { return nil, err }
    var modules []string
    for _, line := range strings.Split(string(data), "\n") {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "use ") {
            modDir := strings.TrimSpace(strings.TrimPrefix(line, "use "))
            modDir = strings.Trim(modDir, "()")
            modules = append(modules, modDir)
        }
    }
    return modules, nil
}
```

When `go.work` exists, run `GoExtractor` for each module directory and merge.

### cmd/ directory detection

```go
func DetectDeployUnits(dir string) []string {
    cmdDir := filepath.Join(dir, "cmd")
    entries, err := os.ReadDir(cmdDir)
    if err != nil { return nil }
    var units []string
    for _, e := range entries {
        if e.IsDir() { units = append(units, e.Name()) }
    }
    return units
}
```

### Framework detection from imports

```go
var goFrameworkSignals = map[string]architect.Framework{
    "github.com/gin-gonic/gin":       {Name: "Gin", Confidence: 0.95, Evidence: "gin import"},
    "github.com/labstack/echo/v4":    {Name: "Echo", Confidence: 0.95, Evidence: "echo import"},
    "github.com/go-chi/chi/v5":       {Name: "Chi", Confidence: 0.95, Evidence: "chi import"},
    "google.golang.org/grpc":         {Name: "gRPC", Confidence: 0.9, Evidence: "grpc import"},
    "github.com/gorilla/mux":         {Name: "Gorilla Mux", Confidence: 0.9, Evidence: "gorilla/mux import"},
    "github.com/go-kratos/kratos/v2": {Name: "Kratos", Confidence: 0.9, Evidence: "kratos import"},
    "github.com/gofiber/fiber/v2":    {Name: "Fiber", Confidence: 0.9, Evidence: "fiber import"},
}
```

After building the import graph, scan edges for known framework imports and add
Framework entries to the output.

### HTTP handler detection

```go
var reGoHTTPHandler = regexp.MustCompile(
    `func\s+\w+\s*\(\s*\w+\s+(?:\*?)?(?:\w+\.)?ResponseWriter\b`,
)
```

### Error handling

| Error | Severity | Action |
|-------|----------|--------|
| No go.mod | Non-fatal | Return empty fragment (not a Go project) |
| go/packages.Load fails | Fatal | Return error (build environment issue) |
| Individual file read error | Non-fatal | Skip file |

---

## 18. Python Extractor Enhancement (Phase A-8)

### Accuracy target: 60-70%

### Import resolution strategy

```
1. Absolute imports ("import flask"):
   - Top-level module -> classify as stdlib (pythonStdlib map), third-party, or local.
   - Local = directory with __init__.py exists under project root.

2. Relative imports ("from . import utils"):
   - Already implemented in resolveImport(). No changes needed.

3. Conditional imports (try/except):
   - Tree-sitter captures both branches.
   - Mark both as "conditional" in Kind field.

4. Dynamic imports (importlib.import_module(...)):
   - If argument is a string literal, capture it.
   - Otherwise mark as "dynamic_import" with low confidence.
```

### Blueprint instance tracking

See Section 2.2 for the full algorithm. Key function:

```go
var blueprintAssignPattern = regexp.MustCompile(
    `(\w+)\s*=\s*Blueprint\s*\(\s*["']([^"']+)["']\s*(?:,\s*__name__)?\s*(?:,\s*url_prefix\s*=\s*["']([^"']+)["'])?`)

var registerBlueprintPattern = regexp.MustCompile(
    `app\.register_blueprint\s*\(\s*(\w+)\s*(?:,\s*url_prefix\s*=\s*["']([^"']+)["'])?`)
```

### Error handling

| Error | Severity | Action |
|-------|----------|--------|
| Unparseable .py file | Non-fatal | Skip file |
| Missing requirements.txt | Non-fatal | Rely on source analysis |
| Unicode decode error | Non-fatal | Skip file |
| Tree-sitter parse error | Non-fatal | Fall back to regex |

### Testing

| Fixture | Assertion | Expected |
|---------|-----------|----------|
| `python/flask-app` | Framework Flask detected | Confidence >= 0.9 |
| `python/flask-app` | Route count | >= 2 |
| `python/fastapi-app` | Framework FastAPI detected | Confidence >= 0.9 |
| `python/django-project` | Framework Django detected | Confidence >= 0.8 |
| `python/django-project` | Dependencies from requirements.txt | >= 5 |

### Known limitations

- `importlib.import_module()` with dynamic names cannot be resolved.
- `sys.path` runtime modifications are not analyzed.
- Jupyter notebook (.ipynb) imports are not extracted.
- Conditional imports inside try/except are captured but not resolved to one branch.

---

## 19. Java/Kotlin Extractor Enhancement (Phase A-9)

### Accuracy target: 70-80%

### Annotation-level scanning

The current implementation detects Spring via imports only. Add direct annotation scanning:

```go
var springAnnotationPatterns = map[string]string{
    `@SpringBootApplication`: "spring_app",
    `@RestController`:        "spring_controller",
    `@Controller`:            "spring_controller",
    `@RequestMapping`:        "spring_endpoint",
    `@GetMapping`:            "spring_endpoint_get",
    `@PostMapping`:           "spring_endpoint_post",
    `@Service`:               "spring_service",
    `@Component`:             "spring_component",
    `@Repository`:            "spring_repository",
    `@Entity`:                "spring_entity",
    `@Configuration`:         "spring_config",
    `@Bean`:                  "spring_config_bean",
}
```

Algorithm: scan file content for each pattern. Record file path and class name.

### HTTP endpoint extraction

```go
var reSpringMapping = regexp.MustCompile(
    `@(?:Get|Post|Put|Delete|Patch|Request)Mapping\s*\(\s*(?:"([^"]*)")?`)
```

Extract URL path from annotation. Concatenate class-level `@RequestMapping` prefix with method-level path.

### Module boundary: Maven multi-module

See Section 3.1 for the full algorithm. Uses existing `parseModules()` function.

### Module boundary: Gradle subprojects

See Section 3.2 for the full algorithm. Parse `settings.gradle`/`settings.gradle.kts` for `include` directives.

### Import resolution strategy

```
1. Parse import declarations from each .java/.kt file.
2. Classify:
   a. java.*/javax.*/jakarta.* -> stdlib
   b. Matches known dependency from pom.xml/build.gradle -> third-party
   c. Matches a package within the project -> internal
   d. Everything else -> unresolved_external
3. For internal imports, build edges: (source_package, target_package).
```

### Error handling

| Error | Severity | Action |
|-------|----------|--------|
| No pom.xml or build.gradle | Non-fatal | May not be Java project |
| Malformed XML | Non-fatal | Log warning, skip |
| File >2MB | Non-fatal | Skip file |
| Missing package declaration | Non-fatal | Use directory path |

### Testing

| Fixture | Assertion | Expected |
|---------|-----------|----------|
| `java/spring-boot` | Spring Boot detected | Confidence >= 0.9 |
| `java/spring-boot` | REST controllers found | >= 1 |
| `java/multi-module-maven` | Module count | >= 2 |
| `java/spring-boot` | Import graph nodes | >= 5 |

### Known limitations

- Reflection (`Class.forName`) not tracked.
- Runtime DI wiring (Spring XML, Guice) not analyzed.
- Annotation processors (Lombok, MapStruct) generate invisible code.
- Kotlin DSL build scripts parsed with regex only.

---

## 20. TypeScript/JavaScript Extractor Enhancement (Phase A-10)

### Accuracy target: 65-75%

### Import resolution strategy

```
1. Bare specifiers ("react", "lodash"):
   - Check workspace package names -> internal workspace dep
   - Check node_modules -> third-party
   - Otherwise -> unresolved

2. Relative specifiers ("./utils", "../Button"):
   - Resolve against importing file's directory.
   - Try extensions: .ts, .tsx, .js, .jsx, .mjs, .cjs.
   - Try /index.ts, /index.tsx, /index.js.

3. Path aliases ("@/components"):
   - Apply tsconfig.json paths mappings (already implemented).
   - After alias resolution, treat as relative.

4. Dynamic imports (import("...")):
   - Capture string literal argument.
   - Classify same as static imports.
   - Mark Kind as "dynamic_import" with lower confidence.
```

### pnpm workspace detection

```go
func parsePnpmWorkspace(rootDir string) ([]string, error) {
    data, err := os.ReadFile(filepath.Join(rootDir, "pnpm-workspace.yaml"))
    if err != nil { return nil, err }
    var ws struct {
        Packages []string `yaml:"packages"`
    }
    if err := yaml.Unmarshal(data, &ws); err != nil {
        return nil, fmt.Errorf("parse pnpm-workspace.yaml: %w", err)
    }
    return ws.Packages, nil
}
```

### Turborepo detection

```go
func detectTurborepo(root string) bool {
    return fileExists(filepath.Join(root, "turbo.json"))
}
```

### Error handling

| Error | Severity | Action |
|-------|----------|--------|
| Malformed package.json | Non-fatal | Skip, log warning |
| Missing tsconfig.json | Non-fatal | No path alias resolution |
| Unparseable .ts/.tsx | Non-fatal | Skip file |
| Circular re-exports | Non-fatal | Detect and flag as CircularDep |

### Testing

| Fixture | Assertion | Expected |
|---------|-----------|----------|
| `typescript/express-api` | Express detected | Confidence >= 0.9 |
| `typescript/express-api` | Route count | >= 3 |
| `typescript/nestjs-app` | NestJS detected | Confidence >= 0.9 |
| `typescript/nextjs-app` | Next.js detected | Confidence >= 0.9 |
| `typescript/monorepo` | Workspace count | >= 2 |

### Known limitations

- Webpack module federation not analyzed.
- Barrel re-exports create false coupling.
- `require()` with variable arguments not resolved.
- Path aliases in jsconfig.json not parsed (only tsconfig.json).

---

## 21. SQL Extractor Enhancement (Phase A-11)

### Accuracy target: 80-90% schema, 50-60% queries

### Stored procedure detection

```go
var reCreateProc = regexp.MustCompile(
    `(?i)CREATE\s+(?:OR\s+REPLACE\s+)?(?:FUNCTION|PROCEDURE)\s+["\x60]?(\w+)["\x60]?\s*\(`)

var reCreateTrigger = regexp.MustCompile(
    `(?i)CREATE\s+(?:OR\s+REPLACE\s+)?TRIGGER\s+["\x60]?(\w+)["\x60]?`)
```

### ALTER TABLE tracking

```go
var reAlterAddColumn = regexp.MustCompile(
    `(?i)ALTER\s+TABLE\s+["\x60]?(\w+)["\x60]?\s+ADD\s+(?:COLUMN\s+)?["\x60]?(\w+)["\x60]?\s+(\w+)`)

var reAlterAddFK = regexp.MustCompile(
    `(?i)ALTER\s+TABLE\s+["\x60]?(\w+)["\x60]?\s+ADD\s+(?:CONSTRAINT\s+\w+\s+)?` +
    `FOREIGN\s+KEY\s*\(\s*["\x60]?(\w+)["\x60]?\s*\)\s*REFERENCES\s+["\x60]?(\w+)["\x60]?`)
```

### ORM correlation algorithm

```
1. For each ORM model (GORM, Django, SQLAlchemy, Prisma, JPA):
   a. Extract the model name.
   b. Search SQLAnalysis.Tables for a matching table name.
   c. If found: link ORM model to SQL table.
   d. If not found: table may be in a different service.

2. Cross-reference column names:
   a. For linked ORM-table pairs, compare column names.
   b. Flag mismatches (schema drift).
```

### Expanded PII detection

```go
var piiExactPatterns = []string{
    "email", "phone", "ssn", "birth_date", "date_of_birth",
    "address", "first_name", "last_name", "ip_address",
    "credit_card", "password", "salary", "iban",
    "passport", "national_id",
}
```

### Testing

| Fixture | Assertion | Expected |
|---------|-----------|----------|
| `sql/ecommerce-schema` | Table count | >= 10 |
| `sql/ecommerce-schema` | FK count | >= 5 |
| `sql/ecommerce-schema` | PII columns | >= 3 |
| `sql/ecommerce-schema` | Data domains | >= 2 |
| `sql/ecommerce-schema` | Migrations detected | Count > 0 |

### Known limitations

- Dynamic SQL (generated at runtime) invisible.
- ORM-generated queries not traced.
- Cross-database references not resolved.
- Database-specific dialects may not parse correctly.

---

## 22. Cross-Cutting Concerns

### 22.1 Concurrency model

```go
func RunExtractors(ctx context.Context, repoRoot string,
    extractors []architect.Extractor) ([]*architect.ProfileFragment, error) {

    g, gctx := errgroup.WithContext(ctx)
    g.SetLimit(runtime.GOMAXPROCS(0))

    var mu sync.Mutex
    var results []*architect.ProfileFragment
    var firstErr error

    for _, ext := range extractors {
        ext := ext
        g.Go(func() error {
            frag, err := ext.Extract(gctx, repoRoot)
            mu.Lock()
            defer mu.Unlock()
            if err != nil {
                if firstErr == nil {
                    firstErr = fmt.Errorf("extractor %s: %w", ext.Name(), err)
                }
                return nil // do not cancel other extractors
            }
            results = append(results, frag)
            return nil
        })
    }

    _ = g.Wait()
    return results, firstErr
}
```

### 22.2 Performance SLAs

> **Authoritative values** are defined in Section 7.3 and Section 7.2. The table below restates them for quick reference. In case of conflict, Section 7 wins.

| Scope | Timeout | Rationale |
|-------|---------|-----------|
| Per extractor CALL | **30s** | Section 7.3: single-language extractor timeout |
| Per round (all extractors) | **180s** | All extractors run concurrently; wall-clock bound |
| Per repository session | **300s (5 min)** | Section 7.2: CLI must return fast enough for interactive use |

**Scaling table (extraction phase only):**

| Repo size | Max extraction time | Max total time (extraction + LLM) |
|-----------|--------------------|----------------------------------|
| <1K files | <10s | <30s |
| <10K files | <60s | <2 min |
| <50K files | <5 min (300s) | <6 min |
| >50K files | Incremental | Variable |

### 22.3 Performance guard rails

| Concern | Limit | Implementation |
|---------|-------|----------------|
| Max file size for parsing | 2 MB | Check `os.Stat().Size()` before reading |
| Max memory per extractor | 256 MB | Streaming parsers, no full-file buffering |
| Concurrency | `runtime.NumCPU()` | `errgroup.Group` with `SetLimit` |
| Large repos (>50K files) | Sample first 10K files | Reservoir sampling for import extraction |
| Tree-sitter parse timeout | 5s per file | Context with timeout |

### 22.4 Registry update

After all enhancements, `DefaultExtractors()` remains unchanged in order:

```go
func DefaultExtractors() []architect.Extractor {
    return []architect.Extractor{
        FileTreeExtractor{},          // runs first: tree structure + naming
        DependencyManifestParser{},   // manifest deps + signals
        SpecInventoryScanner{},       // spec file inventory
        GeneratedCodeDetector{},      // flag generated files
        &InfraExtractor{},            // Docker, k8s, Terraform, CI, serverless
        GitHistoryExtractor{},        // git history, co-change, ownership
        GoAdapter{},                  // go/packages + framework detection
        PythonAdapter{},              // regex + tree-sitter fallback
        JavaAdapter{},                // regex + tree-sitter fallback
        TypeScriptAdapter{},          // regex + tree-sitter fallback
        SQLExtractor{},               // DDL parser + ORM + PII
    }
}
```

Order matters: `FileTreeExtractor` and `GeneratedCodeDetector` run before
language adapters so later extractors can reference the generated file list.

### 22.5 Tree-sitter dependency

```go
// go.mod additions:
// github.com/tree-sitter/go-tree-sitter v0.24
// github.com/tree-sitter/tree-sitter-python v0.23
// github.com/tree-sitter/tree-sitter-java v0.23
// github.com/tree-sitter/tree-sitter-typescript v0.23
// github.com/tree-sitter/tree-sitter-kotlin v0.4
```

Grammars compile into the binary via `go generate`. No external shared libraries.

---

## 23. Implementation Order

| Phase | Item | Estimated effort | Depends on |
|-------|------|-----------------|------------|
| 1 | SecurityFilter enhancement (Section 10) | 1 day | Nothing |
| 2 | FileTreeExtractor TopLevel + NamingPatterns (Section 11) | 0.5 day | Nothing |
| 3 | GeneratedCodeDetector patterns (Section 14) | 0.5 day | Nothing |
| 4 | DependencyManifestParser enhancement (Section 12) | 1 day | Nothing |
| 5 | InfraExtractor serverless + Helm + CI (Section 13) | 1 day | Nothing |
| 6 | Evaluation harness framework (Section 16) | 1 day | Nothing |
| 7 | Go extractor framework detection + go.work (Section 17) | 1 day | go/packages |
| 8 | Python extractor tree-sitter + frameworks (Section 18) | 2 days | tree-sitter |
| 9 | Java/Kotlin extractor tree-sitter + Spring (Section 19) | 2 days | tree-sitter |
| 10 | TypeScript extractor tree-sitter + frameworks (Section 20) | 2 days | tree-sitter |
| 11 | SQL extractor enhancements (Section 21) | 1 day | Nothing |
| 12 | CodebaseProfile assembly + merge (Section 15) | 1 day | Items 2-11 |
| 13 | Integration testing + accuracy validation | 2 days | All above |

Total estimate: ~15 working days for one developer.

---

## 24. Acceptance Criteria

Each extractor must pass:

1. **Zero crashes**: No panics, no unhandled errors on any fixture.
2. **Accuracy floor**: Each language meets its stated accuracy target on the evaluation harness (measured against golden files).
3. **Performance budget**: Full extraction on `sql/ecommerce-schema` completes in under 5 seconds. On a 10K-file repo, under 60 seconds.
4. **Security**: Running `SecurityFilter.Sanitize` on the assembled profile produces zero `SecretMatch` hits and scrubs all user paths.
5. **Determinism**: Running the same extractor twice on the same fixture produces byte-identical JSON output.
