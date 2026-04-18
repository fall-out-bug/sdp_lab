# F130 — Agent Code Quality Standards

> **Status:** In Progress
> **Created:** 2026-04-18
> **Author:** Andrei (session 2026-04-18, "Day 1 - Rules" challenge)
> **Depends on:** F127 (multi-harness modernization), F129 (autonomy/regression hardening)

---

## 1. Problem

Without explicit code style rules, AI agents adapt local patterns or miss project conventions entirely. Observed gaps from "Day 1 - Rules" challenge:

- **Constructor naming** — agents use `NewFoo()` when the project convention is `MakeFoo()` or vice versa; inconsistency only surfaces at review.
- **Module paths** — agents infer `go.mod` module paths incorrectly when the canonical path differs from the directory name.
- **Subprocess safety** — agents write `exec.Command("sh", "-c", untrustedInput)` without context-cancellation wiring or timeout guards.
- **Test guards** — integration tests missing `t.Skip()` for missing-binary or missing-env conditions; tests fail in CI but appear green locally.
- **File-level structure** — agents produce valid Go but deviate from the project's canonical file template (package doc, imports grouping, constructor placement).

These gaps cause review cycles, style drift across harnesses, and regressions that only appear when a second agent picks up the same codebase cold.

The core issue: each harness starts every session cold. Without a canonical cold-start reference, every agent re-derives conventions from whichever files it happens to read first, producing divergent style across sessions and harnesses.

---

## 2. Solution

Three complementary artifacts that together close the cold-start gap:

### 2.1 Canonical `go-patterns.md`

A single reference document (`docs/reference/go-patterns.md`) that every harness reads at session start. It contains:

- **Good patterns** (5+) — constructor naming, error wrapping, context propagation, table-driven tests, subprocess safety.
- **Antipatterns** (6+) — with explicit "do NOT do this" examples drawn from real gap discoveries.
- **Architecture patterns** (7+) — package layout, interface placement, dependency direction rules.
- **Canonical file template** — package doc comment, import grouping, constructor, public API, unexported helpers.

The document is written as a rules reference, not a tutorial. Short examples, no prose padding. Agents load it once; it replaces implicit convention derivation.

### 2.2 `sdp-healthcheck` CLI

A binary at `cmd/sdp-healthcheck` that checks the local dev environment and reports actionable pass/fail status. Checks include:

- Go toolchain version meets minimum requirement.
- Docker daemon reachable (required for quality gates).
- Git submodules initialized (required for `.claude/agents`, `.claude/hooks`, `sdp/docs/*` symlinks).
- `bd` (beads) CLI present and reachable.
- `rtk` (Rust Token Killer) present and reachable.
- Module path in `go.mod` matches expected canonical path.

Output format: structured table with PASS/FAIL/WARN per check, actionable remediation hint on failure, and a final summary exit code (0 = all pass, 1 = any failure).

This makes environment problems visible in seconds rather than buried in a failing build.

### 2.3 Rules-Iteration Protocol

A formal methodology for discovering and closing rule gaps:

1. **Write rules** — author `go-patterns.md` v1 from first principles and codebase review.
2. **Real task** — assign an agent a non-trivial implementation task using only the rules document as cold-start context.
3. **Gap analysis** — after the task, enumerate every place the agent deviated from intended conventions, even if the code is correct.
4. **Update rules** — close each identified gap in `go-patterns.md` with a concrete example.
5. **Repeat** — run a new task to validate that v2 closes the gaps found in v1.

This protocol is designed to be run as a recurring SDP skill, not a one-time exercise. Each harness that onboards to the project runs at least one rules-iteration cycle.

---

## 3. Workstreams

| ID | Title | Status | Description |
|----|-------|--------|-------------|
| 00-F130-01 | go-patterns.md canonical cold-start reference | done | Authored `docs/reference/go-patterns.md` with 5 good patterns, 6 antipatterns, 7 architecture patterns, and a canonical file template. Discovered 5 rule gaps during the session task. |
| 00-F130-02 | sdp-healthcheck CLI — dev env health checker | done | Implemented `cmd/sdp-healthcheck` with structured pass/fail output for toolchain, Docker, submodules, beads, rtk, and module path. |
| 00-F130-03 | .cursorrules for Cursor harness | backlog | Author `.cursorrules` at repo root referencing `go-patterns.md` and setting Cursor-specific cold-start behavior (auto-read patterns doc, no implicit convention derivation). |
| 00-F130-04 | CI integration — sdp-healthcheck as pre-push gate | backlog | Wire `sdp-healthcheck` into the pre-push hook and/or CI so environment problems are caught before a broken environment contaminates a build. |
| 00-F130-05 | Rules iteration protocol — challenge methodology as SDP skill | backlog | Codify the "Day 1 - Rules" challenge methodology as a reusable SDP skill (`/rules-iteration`): trigger, rules doc path, task assignment template, gap analysis checklist, update protocol. |

---

## 4. Done Status

Work completed in session 2026-04-18:

- **00-F130-01 done** — `docs/reference/go-patterns.md` authored. Contains good patterns, antipatterns, architecture patterns, and a canonical file template. Five rule gaps discovered and logged during the implementation task.
- **00-F130-02 done** — `cmd/sdp-healthcheck` implemented. Compiles, runs, and reports structured health output for the local dev environment.

Remaining workstreams (00-F130-03 through 00-F130-05) are backlog items with no blocking dependencies on each other.

---

## 5. Acceptance Criteria

- [ ] `docs/reference/go-patterns.md` exists, passes review by at least one additional harness cold-start session without producing the same gaps as session 2026-04-18.
- [ ] `sdp-healthcheck` binary builds via `./scripts/run_go_quality_gates.sh` without errors.
- [ ] `sdp-healthcheck` exits 0 on a correctly configured dev machine and exits 1 with clear remediation messages on a machine with known gaps.
- [ ] `.cursorrules` (00-F130-03) references `go-patterns.md` and is verified by at least one Cursor agent session.
- [ ] `sdp-healthcheck` is wired into the pre-push gate or CI pipeline (00-F130-04); a broken environment blocks push.
- [ ] Rules-iteration skill (00-F130-05) is documented and executable; at least one complete cycle (rules → task → gap analysis → v2) is recorded.
- [ ] All five rule gaps discovered in session 2026-04-18 are closed in `go-patterns.md` v2.

---

## 6. Dependencies

| Feature | Relation |
|---------|----------|
| **F127** — Multi-harness modernization | F130 inherits the multi-harness agent surface. `go-patterns.md` and `.cursorrules` are harness-specific cold-start artifacts introduced by F127's harness modernization lane. |
| **F129** — Autonomy and regression hardening | F130 extends F129's quality gate work. `sdp-healthcheck` is a peer to the regression guards shipped in F129; CI integration (00-F130-04) follows the same gate-wiring pattern. |

---

## 7. Non-Goals

The following are explicitly out of scope for F130:

- **Language-server integration** — F130 does not wire `gopls`, `golangci-lint`, or any LSP into the agent loop. Static analysis tooling is a separate concern.
- **Go linter configuration** — `.golangci.yml` authoring and linter rule tuning are not part of this feature. F130 produces human-readable pattern guidance, not machine-executable lint rules.
- **Security scanning** — no SAST, dependency vulnerability scanning, or supply-chain checks. Those belong to the trust lane (F064–F067, F078).
- **Python/TypeScript patterns** — F130 is Go-only. Other language pattern docs are future work and would be separate features.
- **Automated rule discovery** — no LLM-based codebase analysis to auto-generate pattern docs. The rules-iteration protocol is a human-in-the-loop process.
