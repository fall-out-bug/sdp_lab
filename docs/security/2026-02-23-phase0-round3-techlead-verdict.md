# Phase 0 (F014–F021) Tech Lead Review — Round 3 Verdict

**Reviewer:** Tech Lead Expert (Sam Newman + Martin Fowler + Kent C. Dodds + Theo Browne)  
**Date:** 2026-02-23  
**Scope:** Agent Loop Reliability — LOC, SOLID, duplication, error handling, naming

---

## 1. LOC Compliance

| File | LOC | Limit | Status |
|------|-----|-------|--------|
| `internal/orchestrate/cli.go` | **207** | 200 | ❌ OVER |
| `cmd/sdp-ci-loop/main.go` | 195 | 200 | ✅ |
| `internal/ciloop/classifier.go` | 41 | 200 | ✅ |
| `internal/ciloop/fixer.go` | 188 | 200 | ✅ |

**P2:** `internal/orchestrate/cli.go` exceeds 200 LOC by 7 lines. Test files exempted.

---

## 2. SOLID Principles

> Analyzing as **Sam Newman** (architecture) because Phase 0 is foundational infrastructure.

**Principles from 3 experts:**
1. **Martin Fowler:** "Extract till you drop — small, focused modules"
2. **Theo Browne:** "Fail fast, explicit errors — never return success with invalid data"
3. **Kent C. Dodds:** "Test behavior; interfaces enable testability"

### Single Responsibility (S)

**`internal/orchestrate/cli.go`** mixes:
- Git helpers: `CurrentBranch`, `RunPRPhase`, `GetPRInfo`
- Checkpoint/runfile: `EnsureRunFile`
- CI orchestration: `RunCILoop`
- Main loop: `RunOpenCodeLoop` (build/review/pr/ci-loop/done state machine)

**P2:** SRP violation — one file has 4+ distinct responsibilities.

### Dependency Inversion (D)

- `RunCILoop`, `RunPRPhase`, `GetPRInfo` use `exec.Command` directly — no interfaces.
- `cmd/sdp-ci-loop/main.go` correctly uses `CommandRunner` for `ghLogFetcher` and `Poller` (v9w3 fix).
- **P3:** orchestrate package could inject a runner for testability; acceptable for Phase 0.

### classifier.go / fixer.go

- **FixType** shared between `Classify` and `Fixer.applyFix` — DRY, single source of truth.
- `fixer.go` uses `LogFetcher` and `Committer` interfaces — good DIP.

---

## 3. Code Duplication

### CurrentBranch vs currentBranch (P2)

| Location | Signature | Error handling |
|----------|-----------|----------------|
| `internal/orchestrate/cli.go:22` | `CurrentBranch() (string, error)` | Returns error |
| `cmd/sdp-ci-loop/main.go:189` | `currentBranch() string` | Returns `""` on error, swallows |

Same logic (`git branch --show-current`), different implementations. `sdp-ci-loop` should reuse `orchestrate.CurrentBranch()` or a shared helper.

### RunOpenCodeLoop error pattern (P3)

The pattern `fmt.Fprintf(os.Stderr, "error: ...\n", err); os.Exit(1)` appears ~11 times. Could extract:

```go
func fatal(format string, args ...any) {
    fmt.Fprintf(os.Stderr, format, args...)
    os.Exit(1)
}
```

### SaveCheckpoint + error (P3)

Repeated in build, review, pr, ci-loop cases. Minor; extracting would reduce ~4 lines.

---

## 4. Error Handling Quality

### P1: GetPRInfo silent failure

**Location:** `internal/orchestrate/cli.go` lines 83–90

```go
if err != nil || len(out) == 0 {
    return 0, "", err   // When len(out)==0 and err==nil → returns (0, "", nil)
}
// ...
if err := json.Unmarshal(out, &arr); err != nil || len(arr) == 0 {
    return 0, "", err   // When len(arr)==0 and err==nil → returns (0, "", nil)
}
```

**Bug:** When no PR exists for the branch, `gh pr list` returns `[]`, `Unmarshal` succeeds, `len(arr)==0`. We return `(0, "", nil)` — **success with invalid data**.

**Impact:** Callers (e.g. `RunOpenCodeLoop` ci-loop case, `cmd/sdp-orchestrate/main.go` advance flow) receive `prNum=0` with `err=nil`. They may skip `RunCILoop` but still set `Phase = PhaseDone`, advancing state incorrectly.

**Fix:** Return explicit error when no PR found:

```go
if len(arr) == 0 {
    return 0, "", fmt.Errorf("no PR found for branch %s", branch)
}
```

And for `len(out)==0`:

```go
if len(out) == 0 {
    return 0, "", fmt.Errorf("gh pr list returned empty output")
}
```

### Other error handling

- All other errors properly wrapped with `%w`.
- No swallowed errors (round 2 fixes verified).
- `fixer.go` sanitization and commit message handling correct.

---

## 5. Naming Conventions

- Go conventions followed: PascalCase exported, camelCase unexported.
- `FixType`, `Classify`, `Classification` — clear.
- `ghLogFetcher`, `execRunner`, `gitCommitter` — descriptive.
- **No P0–P2 naming issues.**

---

## 6. Summary of Findings

| Severity | Count | Findings |
|----------|-------|----------|
| P0 | 0 | — |
| P1 | 1 | GetPRInfo returns (0, "", nil) when no PR — silent failure |
| P2 | 3 | cli.go LOC 207 > 200; SRP violation; CurrentBranch/currentBranch duplication |
| P3 | 2 | Repeated error pattern in RunOpenCodeLoop; SaveCheckpoint repetition |

---

## Verdict: **FAIL**

**Rule:** PASS if all findings are P2 or P3. FAIL for P0 or P1.

One **P1** finding (GetPRInfo silent failure) → **FAIL**.

---

## Recommended Remediation

1. **P1 (required):** Fix `GetPRInfo` to return explicit error when `len(out)==0` or `len(arr)==0` instead of `(0, "", nil)`.

2. **P2 (recommended):**
   - Split `cli.go` or extract helpers to bring it under 200 LOC (e.g. move `RunOpenCodeLoop` to `loop.go` or extract git helpers).
   - Replace `currentBranch()` in `cmd/sdp-ci-loop/main.go` with `orchestrate.CurrentBranch()` (or shared pkg).

3. **P3 (optional):** Extract `fatal()` helper and reduce SaveCheckpoint repetition in `RunOpenCodeLoop`.
