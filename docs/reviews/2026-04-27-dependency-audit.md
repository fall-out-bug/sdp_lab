# Dependency Diet and Duplicate-Code Audit

> **Workstream:** 00-150-05 (F150-05, sdplab-hjl7)
> **Date:** 2026-04-27
> **Scope:** go.mod direct/indirect dependencies, duplicate helper patterns in internal/

## Summary

The project has 16 direct dependencies and 80 indirect dependencies (96 total in go.mod, 430 lines in go.sum). All direct dependencies are justified by at least one stable or operator-facing owner, or are correctly scoped to experimental code. No dependencies were removed during this audit -- `go mod tidy` produced no changes, confirming all declared dependencies are in use.

The heaviest dependency chain is the sigstore ecosystem (sigstore-go -> sigstore/rekor -> rekor-tiles -> timestamp-authority), which pulls approximately 725 transitive edges. This is justified by the stable evidence package.

Nine duplicate code patterns were identified, with fix/defer decisions documented below.

## 1. Direct Dependency Justification

| Direct Dependency | Surface | Used By | Verdict |
|---|---|---|---|
| `github.com/gomarkdown/markdown` | stable | `internal/architect/security` (sanitize.go) | **Keep** -- GA architect package |
| `github.com/google/uuid` | stable | `cmd/sdp` (main CLI) | **Keep** -- core product |
| `github.com/in-toto/in-toto-golang` | stable | `internal/evidence` (attestations) | **Keep** -- GA evidence package |
| `github.com/ledongthuc/pdf` | experimental | `internal/strataudit` (extract.go) | **Keep** -- strataudit is GA but experimental surface; note for build-tag isolation (F150-04) |
| `github.com/mark3labs/mcp-go` | experimental | `internal/mcp`, `cmd/sdp-mcp` | **Keep** -- MCP is Beta/tooling; note for build-tag isolation |
| `github.com/mattn/go-sqlite3` | mixed | `internal/agentloop` (experimental), `internal/beads` (stable), `internal/index` (stable), `internal/session` (stable via orchestrate), `internal/strataudit` (experimental) | **Keep** -- CGO SQLite is used by 4 stable packages |
| `github.com/microcosm-cc/bluemonday` | stable | `internal/architect/security` (sanitize.go) | **Keep** -- HTML sanitizer for GA architect |
| `github.com/santhosh-tekuri/jsonschema/v5` | stable | `internal/manifest`, `internal/adapters/sdk`, `internal/orchestrate`, `cmd/sdp-ws-verdict-validate`, `internal/evidence`, `internal/bridge` | **Keep** -- schema validation across 6+ stable packages |
| `github.com/sigstore/sigstore-go` | stable | `internal/evidence` (sigstore_signer.go) | **Keep** -- GA evidence package; heavy transitive chain |
| `github.com/stretchr/testify` | test | widespread (internal/agentloop.test and many others) | **Keep** -- test framework |
| `golang.org/x/sync` | stable | `internal/architect` (errgroup) | **Keep** -- GA architect package |
| `golang.org/x/sys` | stable | `internal/architect/security` (unix fd) | **Keep** -- GA architect, security-critical |
| `golang.org/x/time` | experimental | `internal/strataudit` (rate limiter), `internal/spec/testdata` (sample code) | **Defer** -- only experimental + test fixture; candidate for removal once build-tag isolation lands (F150-04) |
| `golang.org/x/tools` | stable | `internal/architect/extract` (go/packages) | **Keep** -- GA architect extraction |
| `google.golang.org/protobuf` | stable | `internal/evidence` (protojson) | **Keep** -- GA evidence, required by in-toto/sigstore |
| `gopkg.in/yaml.v3` | mixed | `cmd/sdp-strataudit` (experimental), also used transitively | **Keep** -- YAML config loading in multiple packages |

### Dependencies classified as experimental-only

These dependencies are used exclusively by experimental/lab-only code. They remain in go.mod because the code compiles, but they should be isolated behind build tags per F150-04:

- `golang.org/x/time` -- only `internal/strataudit` (experimental) and `internal/spec/testdata` (sample fixture)
- `github.com/ledongthuc/pdf` -- only `internal/strataudit` (experimental)
- `github.com/mark3labs/mcp-go` -- only `internal/mcp` and `cmd/sdp-mcp` (Beta/experimental)

## 2. Heavy Indirect Dependency Chains

### sigstore ecosystem (~725 transitive graph edges)

The `github.com/sigstore/sigstore-go` direct dependency pulls the heaviest chain:

| Heavy Indirect Dep | Transitive Children | Notes |
|---|---|---|
| `github.com/sigstore/rekor` | 222 | Transparency log client |
| `github.com/sigstore/rekor-tiles/v2` | 210 | Rekor v2 tiles |
| `github.com/sigstore/timestamp-authority/v2` | 160 | RFC 3161 TSA |
| `github.com/google/certificate-transparency-go` | 137 | CT log client |
| `github.com/google/go-containerregistry` | 49 | Container image registry |
| `github.com/go-openapi/runtime` | 37 | OpenAPI runtime (used by rekor) |

**Verdict:** This chain is justified by `internal/evidence` (GA, stable surface). The sigstore ecosystem is the industry standard for supply-chain signing and verification. No reduction possible without removing evidence functionality. The `go-openapi/*` sub-packages (14 modules) are all pulled transitively by rekor, not by sdp_lab directly.

### Other notable indirect chains

| Indirect Dep | Pulled By | Transitive Children | Notes |
|---|---|---|---|
| `google.golang.org/grpc` | sigstore, in-toto | 35 | Required for protobuf-based APIs |
| `go.opentelemetry.io/otel` | grpc (transitive) | 12 | OpenTelemetry SDK, transitive only |
| `go.mongodb.org/mongo-driver` | sigstore/rekor (transitive) | 12 | MongoDB driver for rekor internals |
| `github.com/go-openapi/*` (14 pkgs) | sigstore/rekor | varies | OpenAPI toolkit for rekor API client |

**Verdict:** All heavy indirect chains trace back to `sigstore-go` -> evidence (GA). No action needed.

## 3. Duplicate Code Patterns

### P1: safePath (3 implementations, security-relevant)

Three packages implement path-traversal protection independently:

| Location | Signature | Method |
|---|---|---|
| `internal/agentloop/tools_live.go:651` | `safePath(root, relPath string) (string, error)` | filepath.Clean + EvalSymlinks + prefix check |
| `internal/skills/augment.go:73` | `safePath(baseDir, untrusted string) (string, error)` | filepath.Abs + EvalSymlinks + prefix check |
| `internal/architect/security/path.go` | `PathValidator.ValidatePath(rawPath string)` | unix.Open + dirfd-based (most rigorous) |

**Decision: File bead (sdplab-hjl7 scope).** The `architect/security` implementation is the most rigorous (uses dirfd). A future refactor should consolidate `agentloop` and `skills` to use `security.PathValidator` or a shared `internal/common/safepath` utility. **Defer to post-F150** -- the implementations serve different packages and refactoring them during release prep introduces risk.

### P2: fileExists / dirExists (3 implementations)

| Location | Signature | Identical? |
|---|---|---|
| `internal/bootstrap/collector.go:165` | `fileExists(path string) bool` | Yes (same os.Stat pattern) |
| `internal/architect/extract/ts_extract.go:1174` | `fileExists(path string) bool` | Yes |
| `internal/architect/extract/typescript/extractor.go:222` | `fileExists(path string) bool` | Exact copy of ts_extract |

Additionally, `dirExists` is duplicated identically in:
- `internal/architect/extract/ts_extract.go:1180`
- `internal/architect/extract/typescript/extractor.go:228`

**Decision: Defer.** Low-risk duplication in extraction code. A shared `internal/common/fsutil` package could host these, but the functions are trivial (2 lines each) and the duplication is contained within the architect subsystem.

### P3: writeFile test helpers (4 implementations)

| Location | Signature |
|---|---|
| `internal/architect/extract/java_extract_test.go:28` | `writeFile(t *testing.T, path, content string)` |
| `internal/architect/extract/java/spring_test.go:21` | `writeFile(t *testing.T, path, content string)` -- exact copy |
| `internal/backlog/reference_check_test.go:17` | `mustWriteFile(t *testing.T, path string, data []byte)` |
| `internal/inference/microfirst/bdseverity/classifier_test.go:18` | `writeFile(path, content string) error` -- no *testing.T |

**Decision: Defer.** Test-only helpers, no production impact. The `java_extract_test.go` and `spring_test.go` versions are identical and could share a helper, but this is cosmetic.

### P4: jsonschema.NewCompiler() (5 call sites)

| Location | Package Surface |
|---|---|
| `internal/manifest/load.go:59` | stable |
| `internal/adapters/sdk/validation.go:227` | stable |
| `internal/adapters/sdk/validation.go:266` | stable |
| `internal/inference/confidence/adapters/wsverdict/wsverdict.go:172` | experimental |
| `internal/inference/decompose/stitcher_json.go:21` | experimental |
| `internal/orchestrate/checkpoint.go:25` | stable |

**Decision: Defer.** Each site compiles a different schema. A shared schema registry would reduce boilerplate but would be an architectural change beyond this audit's scope.

### P5: path validation patterns (26 filepath.Clean, 19 filepath.EvalSymlinks)

Path sanitization is widespread across the codebase (26 Clean, 19 EvalSymlinks sites). Not all are duplicates -- many serve different contexts (config loading, tool execution, file reading).

**Decision: Note only.** The `architect/security.PathValidator` provides the most rigorous implementation. Other packages use lighter-weight checks appropriate to their threat model.

## 4. Removals Made

**None.** `go mod tidy` produced no changes -- all declared dependencies are actively used by compiled code. No dependency was safe to remove without breaking the build or removing functional code.

## 5. Build Evidence

```
$ go mod tidy -v
(no output -- no changes needed)

$ go mod verify
all modules verified

$ go build ./...
(success -- no output)
```

## 6. Recommendations for Post-F150

| Priority | Item | Rationale |
|---|---|---|
| Medium | Build-tag isolation for experimental deps (F150-04) | Isolate `ledongthuc/pdf`, `mark3labs/mcp-go`, `golang.org/x/time` behind build tags so they don't compile into production binaries |
| Low | Consolidate `safePath` into `internal/common/safepath` | 3 implementations; `architect/security` version is the gold standard |
| Low | Extract `fileExists`/`dirExists` to `internal/common/fsutil` | 3 identical implementations |
| Low | Share `writeFile` test helper across architect tests | 4 near-identical implementations |
| Info | Monitor sigstore ecosystem releases for tree-shaking opportunities | 725 transitive edges is heavy but justified; future sigstore versions may reduce |

## Change Log

| Date | Change |
|---|---|
| 2026-04-27 | Initial audit report (F150-05, sdplab-hjl7) |
