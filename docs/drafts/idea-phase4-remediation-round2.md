# Idea: Phase 4 Remediation Round 2 — F053 Review Findings

**Source:** [F053-REVIEW-SUMMARY.md](../reviews/F053-REVIEW-SUMMARY.md) Round 2  
**Scope:** sdp_dev (internal/, cmd/) + sdp submodule  
**Feature:** F053 (extended)

---

## P0 (blocking)

### 1. Pipeline Hooks Shell Injection (sdp_dev-0ddg) — DONE (00-053-16)

**Problem:** `internal/orchestrate/hooks.go:86` uses `exec.CommandContext(hookCtx, "sh", "-c", h.Command)`. Malicious PR can add hooks that run arbitrary code in CI.

**Solution:** Replace `sh -c` with allowlist of executables + args as list, or restricted DSL. No arbitrary shell.

**Scope:** `internal/orchestrate/hooks.go`, `.sdp/pipeline-hooks.yaml.example`

**Migration:** Legacy `command` field is rejected. Use `executable` + `args` array. See `.sdp/pipeline-hooks.yaml.example`.

---

## P1 (blocking)

### 2. sdp-evidence Arbitrary File Read (sdp_dev-qx8i)

**Problem:** `cmd/sdp-evidence/main.go` passes user path to `evidence.Inspect(path)` / `ValidateStrictFile(path)` without validation. `os.ReadFile(path)` can read any file (e.g. `../../../etc/passwd`).

**Solution:** Validate resolved path is under project root or CWD; reject `..`, absolute paths outside allowed dir.

**Scope:** `cmd/sdp-evidence/main.go`, `internal/evidence/inspect.go`

---

### 3. Checkpoint Data Loss (sdp_dev-a2iw) — DONE (00-053-18)

**Problem:** `orchestrate.Checkpoint` and `ciloop.Checkpoint` are different structs. ciloop loads + saves overwrites with smaller struct — Workstreams, Review, CreatedAt lost.

**Solution:** Shared checkpoint package with one struct, or ciloop merge logic preserving unknown fields, or separate checkpoint files.

**Scope:** `internal/orchestrate/checkpoint.go`, `internal/ciloop/checkpoint.go`, `cmd/sdp-ci-loop/main.go`

**Implemented:** Merge-safe save in ciloop.SaveCheckpoint — loads raw map, overlays ciloop fields, preserves Workstreams/Review/CreatedAt.

---

### 4. sdp-orchestrate 0% Coverage (sdp_dev-4rpn) — DONE (00-053-19)

**Problem:** `cmd/sdp-orchestrate/main.go`, `runAdvance`, `runHydrate`, `runNextAction` untested.

**Solution:** Add unit tests for advance/hydrate/nextaction with fakes; integration test for main flow.

**Scope:** `cmd/sdp-orchestrate/`

**Implemented:** Unit tests for runNextAction, runHydrate, runAdvance. Coverage 20.3%.

---

### 5. internal/evidence 40% Coverage (sdp_dev-c5fj)

**Problem:** `auto_attest.go`, `attestation.go`, `cmd/auto-attest` at 0% coverage.

**Solution:** Add tests for AutoAttest, NewStatement, auto-attest CLI paths.

**Scope:** `internal/evidence/`, `cmd/auto-attest/`

---

### 6. F053 Zero ws-verdict Files (sdp_dev-l572)

**Problem:** 15 Done F053 workstreams have no `docs/ws-verdicts/00-053-*.json` with ac_evidence.

**Solution:** Create ws-verdict JSON for each 00-053-01..15 with ac_evidence array.

**Scope:** `docs/ws-verdicts/`, `docs/workstreams/backlog/00-053-*.md`

---

## P2 (tracked)

### 7. Runfile AppendRunEvent Flock (sdp_dev-a9th)

**Scope:** `internal/ciloop/runfile.go` — add flock for inter-process safety.

---

### 8. SRE Context Propagation (sdp_dev-luyz, 06a4, 59z6, pmod)

- time.After leak in `internal/ciloop/loop.go` → use `time.NewTimer` + `defer timer.Stop()`
- Poller.GetChecks lacks ctx → add ctx, check ctx.Done() before retries
- exec.Command without Context in build, policy, auto_attest → use CommandContext
- sdp apply uses context.Background → signal.NotifyContext for SIGINT/SIGTERM

**Scope:** `internal/ciloop/`, `sdp/sdp-plugin/cmd/sdp/apply.go`, `internal/orchestrate/`, `internal/evidence/`

---

### 9. agent-constraints Path Traversal (sdp_dev-akwg)

**Scope:** `internal/orchestrate/constraints.go` — validate `c.Path` before filepath.Join.

---

### 10. Review Skill Language-Agnostic (sdp_dev-mk89, sdp_dev-0eba)

**Scope:** `sdp/prompts/skills/review/SKILL.md` — remove Cursor/Claude-specific references (Task tool, .claude/agents).

---

## Additional (00-053-26..35)

### 11. Verifier + Config + coverage_lang (00-053-26)

Beads: 1j9w, eix0, zd4q, gq7c, lo1j, te1h, 1581

### 12. Evidence Writer + Emitter (00-053-27)

Beads: ywv8, wp5a, 5xbv, gq9x

### 13. Config + Path Validation (00-053-28)

Beads: pxcg, ha5u, 6c73

### 14. coverage_lang Error Handling (00-053-29)

Beads: 9da2, 0azz, f3e0, 8adu

### 15. Executor + Retry (00-053-30)

Beads: 6h50, yedi, uxyj

### 16. Docs + Drift + ws-verdict (00-053-31)

Beads: 9kg0, m9i7, yno1, 5x6s

### 17. Evidence Event + Emit Errors (00-053-32)

Beads: s97b, c8rm

### 18. sdp Submodule (00-053-33)

Beads: yxcg, iid9, 85g4, as2n

### 19. TechLead exec + LLM Invoker (00-053-34)

Beads: 4736, p7xv

### 20. Emitter Flakiness + P3 (00-053-35)

Beads: o62u, qjv4, p77m, 5402

---

## Dependencies

- 00-053-16 (P0 hooks) — no deps
- 00-053-17 (evidence path) — no deps
- 00-053-18 (checkpoint) — no deps
- 00-053-19 (orchestrate coverage) — no deps
- 00-053-20 (evidence coverage) — no deps
- 00-053-21 (ws-verdict) — no deps
- 00-053-22..25 — can run in parallel after P0/P1
