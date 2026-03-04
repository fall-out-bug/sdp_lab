# Idea: Phase 4 Remediation — sdp Review Findings

**Source:** [2026-02-25-beads-remediation-plan.md](../plans/2026-02-25-beads-remediation-plan.md) Phase 4  
**Scope:** sdp submodule (sdp-plugin Go, schemas, prompts)  
**Beads:** dg5t, 0cgy, ppha, zq8l, 2mu6

---

## Requirements

### 1. Verifier Interface Abstraction (sdp_dev-dg5t)

**Problem:** Verifier tight coupling to quality/security — no interface abstraction.

**Current:** `verifier.go` directly imports `internal/quality` and `internal/security`. Cannot mock for tests or swap implementations.

**Solution:** Introduce interfaces (CoverageChecker, PathValidator, CommandRunner). Inject via constructor. Production uses real impls; tests use mocks.

**Scope:** `sdp/sdp-plugin/internal/verify/`

---

### 2. Parser Frontmatter Bug (sdp_dev-0cgy)

**Problem:** `contentStr[4:frontmatterEnd+4]` assumes `---` at start — index bug.

**Scope:** `sdp/sdp-plugin/internal/verify/` (parser)

---

### 3. Intent Schema (sdp_dev-ppha)

**Problem:** `docs/intent/{task_id}.json` has no formal schema.

**Solution:** Verify intent.schema.json exists and is referenced. Extend if needed.

**Scope:** `sdp/schema/`

---

### 4. Schema Path Inconsistency (sdp_dev-zq8l)

**Problem:** build uses `.sdp/schema/`, review uses `schema/` — inconsistent.

**Solution:** Unify path convention in prompts/skills.

**Scope:** `sdp/prompts/skills/`

---

### 5. PromptOps Checks Formalization (sdp_dev-2mu6)

**Problem:** PromptOps checks not formalized — no check_id/status/note table for downstream tools.

**Solution:** Define structured output format for PromptOps reviewer (schema or doc).

**Scope:** `sdp/schema/`, `sdp/prompts/skills/review/`

---

## Out of Scope (sdp_dev)

- tjq8 — TDD artifact (@oneshot)
- vwgy — @vision output
- 8ds3 — @deploy
- mt0x — @oneshot advance
- l0i3 — already covered by sdp guard activate

---

## Dependencies

- WS-01 (Verifier interface) — no deps
- WS-02 (Parser) — no deps
- WS-03 (Intent schema) — no deps
- WS-04 (Schema path) — no deps
- WS-05 (PromptOps) — may depend on WS-04 for schema path
