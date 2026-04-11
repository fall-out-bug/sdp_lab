# R6 Fix Plan (Council-Approved) — 5/5 Reviewed

## Council Verdict: APPROVE WITH REVISIONS
All 5 roles (Critic, Technician, Philosopher, Pragmatist, Engineer) reviewed.
Target: 3.5/5 on ground-truth validation.

## Implementation Order

| Step | Fix | Lines | Dependencies |
|------|-----|-------|-------------|
| 1 | Scala source support | ~80 | None |
| 2 | Scala source roots in javaPackageName | ~5 | Step 1 |
| 3 | EdgeSync constant + import-graph direction | ~75 | Steps 1-2 |
| 4 | Phantom container filtering | ~35 | Step 3 |
| 5 | Python path prefix stripping | ~40 | None (parallel with 1-4) |
| | **Total** | **~235** | |

## Step 1: Scala Source Support (~80 lines)

**File**: `java_extract.go`

1a. `isScalaFile(rel string) bool` — `.scala` suffix check (3 lines)

1b. Scala regex patterns:
```go
var (
    scalaImportRe = regexp.MustCompile(`^import\s+(.+)`)
    scalaPackageRe = regexp.MustCompile(`^package\s+([a-zA-Z0-9_.]+)`)
)
```

1c. `scanScalaFromReader(r io.Reader, relPath string) (string, []string)` (~40 lines):
- Parse package: use `scalaPackageRe` (no semicolon needed)
- Parse imports: use `scalaImportRe` to capture full import line
- **Brace expansion**: detect `{...}` in import, extract entries
  - `import org.apache.spark.sql.{DataFrame, Dataset}` → `org.apache.spark.sql.DataFrame`, `org.apache.spark.sql.Dataset`
  - `import org.apache.spark.sql.{DataFrame => DF}` → `org.apache.spark.sql.DataFrame` (strip ` => alias`)
  - `import org.apache.spark.sql.{_}` → `org.apache.spark.sql.*` (brace wildcard = `._`)
- **Underscore wildcard**: `import foo._` → `foo.*`
- **Multiline brace**: accumulate lines until closing `}` found

1d. Wire into `Extract()` file walker: `case isScalaFile(rel):` alongside Java/Kotlin

1e. Update language label: `"java/kotlin/scala"`

## Step 2: Scala Source Roots in javaPackageName (~5 lines)

**File**: `adapters.go`

Add `"src/main/scala/"` and `"src/test/scala/"` to the source root markers in `javaPackageName()`:
```go
for _, marker := range []string{
    "src/main/java/", "src/test/java/",
    "src/main/kotlin/", "src/test/kotlin/",
    "src/main/scala/", "src/test/scala/",  // NEW
} {
```

This ensures Scala packages are correctly extracted and mapped to Maven modules.

## Step 3: Import-Graph Relationship Direction (~75 lines)

**Files**: `model.go`, `adapters.go`, `pipeline.go`

3a. Add `EdgeSync EdgeKind = "sync"` constant to `model.go` (1 line)

3b. In `convertJavaResult()` (adapters.go), emit directed module edges (~40 lines):
- For each cluster pair (A imports B) in the import graph
- Map A and B to Maven module slugs using `buildModulePrefixMap`
- If A and B map to different modules → emit `StructuralEdge{Source: moduleA, Target: moduleB, Kind: EdgeSync}`
- Dedup by canonical module slug pair
- Use `sortStrings` for deterministic ordering

3c. In `pipeline.go`, replace sibling-pair heuristic (lines 435-476) (~35 lines):
- Remove the `parentPrefix`-based module sibling iteration
- Instead, iterate `profile.Edges` where `Kind == EdgeSync`
- Map source/target to container IDs
- Create `C4Relationship{From, To, Type: "sync", Description: "maven module dependency"}`

## Step 4: Phantom Container Filtering (~35 lines)

**File**: `pipeline.go`, `adapters.go`

4a. In `pipeline.go`, after container creation, filter (~20 lines):
- Keep container if ANY of:
  - Has import-graph cluster with packages
  - Has StructuralEdge pointing to/from it
  - Is Dockerfile-derived
  - **Is in the Maven module list** (from JavaExtractionResult.Modules)
- Remove hard-coded `'spark-runtime'` target

4b. In `adapters.go`, replace `'spark-runtime'` in runtime coupling edges (~10 lines):
- Resolve target from file path → container mapping
- Py4J couplings → target `"jvm"` (external system, not container)
- Spark RPC couplings → target `"spark-rpc"` (keep as external system)

4c. Add `spark-rpc` internal classification (5 lines):
- Mark spark-rpc as internal (not external) in pipeline.go depSystemMap

## Step 5: Python Path Prefix Stripping (~40 lines)

**File**: `python_extract.go`

5a. `findPythonPackageRoots(rootDir string) []string` (~20 lines):
- Walk rootDir looking for directories with `__init__.py`
- Compute longest common prefix of all top-level `__init__.py` parent paths
- For Spark: returns `["spark-3.5.7/python/"]` (prefix to strip)
- For normal repos: returns `[""]` (no stripping)

5b. Strip prefix in `BuildPythonImportGraph()` (~10 lines):
- Pre-scan `findPythonPackageRoots(rootDir)` once before walker
- In walker, strip prefix from `rel` before calling `pyPathToModule`

5c. Update `normalizePythonModulePath()` (~10 lines):
- After stripping, resolved imports should match directly
- Remove the prefix-matching hack that was needed before

## Verification Protocol

1. `go build ./...` + `go vet ./...` + `go test ./...` — clean
2. Small Maven fixture (3 modules, includes .scala files)
3. Run on Spark:
   - `sql/core` has 50+ packages (currently 1)
   - `hive → catalyst` (not `catalyst → hive`)
   - `hive → core` (not `core → hive`)
   - No phantom containers (spark-rm, network-yarn gone)
   - `pyspark.sql`, `pyspark.ml` as separate clusters
4. Council R6 ground-truth validation
