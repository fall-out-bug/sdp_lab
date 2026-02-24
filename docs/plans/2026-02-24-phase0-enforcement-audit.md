# Phase 0 Enforcement Audit: Why Nothing Actually Works

> **Date:** 2026-02-24
> **Trigger:** P2/P3 remediation bypassed all Phase 0 controls — no beads, no evidence, no trace
> **Verdict:** 93% of Phase 0 is cleanup or potential; only 7% is real enforcement

---

## 1. Feature Classification (Charity Majors)

| # | Feature | Category | Rationale |
|---|---------|----------|-----------|
| F014 | CI Loop CLI | POTENTIAL | Works only when agent runs it; ad-hoc flows bypass |
| F015 | Stop Hook Gate | ENFORCEMENT | Blocks exit when checkpoint exists and CI incomplete — but only inside oneshot flow |
| F016 | Oneshot Outer Loop | POTENTIAL | `sdp orchestrate` exists but nothing forces its use |
| F017 | Skill Eval Suite | CLEANUP | Catches skill regressions; no runtime enforcement |
| F018 | Dead Code Purge | CLEANUP | Removed dead code; no enforcement |
| F019 | Skill Compression | CLEANUP | Improved clarity; no enforcement |
| F020 | Build Scope Fix | CLEANUP | Adjusted prompt; agent can still ignore it |
| F021 | Language-Agnostic Skills | CLEANUP | Improved portability; no enforcement |
| F022 | Context Pre-Hydration | POTENTIAL | Only runs when orchestrate runs |
| F023 | Scope Enforcement | POTENTIAL | Only in `--advance`; has `--skip-guard`; direct commits bypass |
| F024 | Phase Hooks | POTENTIAL | `.sdp/pipeline-hooks.yaml` doesn't exist; hooks are no-op |
| F025 | Prompt Consolidation | CLEANUP | Structure improvement; no enforcement |
| F026 | Prompt Provenance | POTENTIAL | Evidence only produced in K8s flows |
| F027 | CI Auto-Fixers | POTENTIAL | Same bypass as F014 |

**Summary:** 1 ENFORCEMENT, 6 CLEANUP, 7 POTENTIAL. ~7% real enforcement.

---

## 2. Architectural Error (Martin Kleppmann)

**Fundamental error: application-level validation instead of system-level enforcement.**

All controls are inside the orchestration pipeline. But the merge point (GitHub) has no controls:

- CI runs: build, test, lint, k8s-validate — **no evidence validation**
- Branch protection: **not configured** for evidence checks
- Any path that skips orchestrate bypasses **everything**

**Analogy:** A database with no constraints, relying on the application to validate. Any client that skips the app (direct SQL, another service, a bug) can violate invariants.

**The choke point is merge.** All changes must pass through merge. Enforcement must live there — not inside one specific workflow.

---

## 3. Self-Referential Paradox (John Ousterhout)

### Is this a design problem or implementation gap?

**Fundamental design problem.** The design assumes agents will follow instructions. Enforcement is implemented as prompts and skills, not as gates.

### What principle is violated?

**"Make the right thing easy and the wrong thing hard."** Currently:
- Wrong thing (direct commits, no beads, no evidence) → easy
- Right thing (beads → evidence → commit) → requires extra steps the agent can skip

Also violated: **separation of concerns** — the same entity (agent) is both executor and enforcer.

### Can agents enforce rules on themselves?

**No.** The analogy:
- OS enforces memory limits on processes; processes cannot opt out
- Database enforces constraints; applications cannot bypass them
- CI enforces checks; developers cannot merge without passing

Here, the agent is asked to enforce its own limits. That is like asking a process to enforce its own memory limits.

### Deep Module principle

The quality system (beads, evidence, guard) has:
- **Complex interface:** Many instructions across skills, AGENTS.md, and docs
- **Shallow behavior:** "The agent should do X." No hidden logic guarantees X happens

A proper deep module:
- **Interface:** `Commit(workstreamID, changes) → error`
- **Hidden behavior:** Checks beads, evidence, scope inside; returns error if any check fails

The current design is documentation, not a module with a mandatory interface.

---

## 4. Minimum Viable Enforcement (Stripe Minions)

### CI Gate (blocks merge)

Add `evidence-gate` job to `.github/workflows/ci.yml`:

```yaml
  evidence-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - name: Validate evidence files
        run: |
          set -e
          base="${GITHUB_BASE_REF:-master}"
          for f in $(git diff --name-only origin/$base...HEAD 2>/dev/null \
            | grep '^\.sdp/evidence/.*\.json$' || true); do
            [ -f "$f" ] && go run ./cmd/sdp-evidence validate --require-pr-url=false "$f"
          done
      - name: Validate review verdict
        run: |
          base="${GITHUB_BASE_REF:-master}"
          if git diff --name-only origin/$base...HEAD 2>/dev/null \
            | grep -q '^\.sdp/review_verdict\.json$'; then
            jq -e '.verdict | test("^(APPROVED|CHANGES_REQUESTED)$")' .sdp/review_verdict.json
            jq -e '.feature and (.finding_ids | type == "array")' .sdp/review_verdict.json
          fi
```

### Branch Protection (blocks bypass)

| Setting | Value |
|---------|-------|
| Branch name pattern | `master` |
| Require status checks | `build-test`, `k8s-validate`, `evidence-gate` |
| Require branches up to date | Yes |
| Do not allow bypassing | Yes |

### Pre-push Hook (catches early)

```bash
#!/bin/sh
# hooks/pre-push.sh — validate evidence before push
while read local_ref local_sha remote_ref remote_sha; do
  [ -z "$local_sha" ] && continue
  range="${remote_sha}..${local_sha}"
  for f in $(git diff --name-only "$range" 2>/dev/null \
    | grep '^\.sdp/evidence/.*\.json$' || true); do
    [ -f "$f" ] && go run ./cmd/sdp-evidence validate --require-pr-url=false "$f" || exit 1
  done
done
```

### Review Verification (prompt-level)

Add to review skill synthesizer: "Parse each subagent output for `FINDINGS_CREATED: id1 id2`. If a subagent reported P0-P3 findings but output lacks `FINDINGS_CREATED`, treat that role as FAIL."

---

## 5. Implementation Order

| Session | What | Prevents |
|---------|------|----------|
| 1 | CI `evidence-gate` job + `hooks/pre-push.sh` | Merge/push with invalid evidence |
| 2 | Branch protection in GitHub + review skill update | Merge without CI; synthesizer skipping findings |

---

## 6. Summary

| What Phase 0 built | What it actually does |
|---------------------|----------------------|
| pr-gate | Validates evidence — but only inside K8s swarm pipeline |
| sdp-evidence validate | CLI exists — but CI never calls it |
| sdp guard | Checks scope — but only inside `sdp orchestrate --advance`, with `--skip-guard` |
| beads tracking | Issue tracker — but agents didn't use it during remediation |
| Phase hooks | Framework exists — but `.sdp/pipeline-hooks.yaml` doesn't exist |
| Stop hook | Blocks premature exit — but only when checkpoint exists from orchestrate |
| Review skill | Says to `bd create` — but nothing verifies subagents did it |

**Core diagnosis:** Phase 0 built tools. It did not build enforcement. The tools work when used; nothing ensures they are used.

**Fix:** Move enforcement to the merge boundary (CI + branch protection). The only reliable enforcement point is where all paths converge: merge into master.
