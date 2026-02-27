# Feature Review: F053 — sdp Repository Comprehensive Audit

**Date:** 2026-02-25  
**Scope:** Entire sdp repo as cohesive product (sdp/ submodule + sdp_dev internal/, cmd/, docs/)  
**Verdict:** CHANGES_REQUESTED (Round 2)

**Product Review:** All findings recorded in beads with `review-finding` label. See Round 2 Beads below.

---

## Summary

**Round 1 (F053 WS 01–11):** Addressed critical gaps. Verdict was APPROVED with tracked P2/P3.

**Round 2 (Full repo audit):** 7-role review found P0/P1 blocking findings. All findings tracked in beads with `review-finding,F053,round-2,{role}`.

---

## Round 2 Reviewers

| Role | Verdict | Findings |
|------|---------|----------|
| QA | FAIL | P1: orchestrate 0% coverage; evidence 40% coverage. P2: emitter flakiness |
| Security | FAIL | P0: pipeline-hooks shell injection (sh -c). P1: sdp-evidence arbitrary file read. P2: path traversal |
| DevOps | PASS | P2: runfile AppendRunEvent lacks flock |
| SRE | FAIL | P1: defer cancel in loop; ctx==nil. P2: time.After leak; poller no ctx; exec without ctx |
| TechLead | FAIL | P1: checkpoint data loss (ciloop overwrites orchestrate). P2: LLM invoker; exec coupling |
| Docs | FAIL | P1: F053 zero ws-verdict files. P2: drift detect; sdp/docs minimal |
| PromptOps | FAIL | P2: review skill Cursor/Claude-specific references |

---

## Round 1 Reviewers (historical)

| Role | Verdict | Findings |
|------|---------|----------|
| QA | PASS | VerifyCoverage wired; Verifier has tests; config.Load calls Validate |
| Security | PASS | Path traversal fixed (guard.rules_file, evidence.log_path); SafeCommand used |
| DevOps | PASS | Evidence singleton + flock; Emit logs errors |
| SRE | PASS | Context propagation in Verifier, Executor; retry respects ctx |
| TechLead | PASS | WorkstreamRunner interface; mock in test; no production mock in retry |
| Docs | PASS | ws-verdict, review-verdict schemas exist; coding-workflow-predicate exists |
| PromptOps | PASS | Oneshot documents sdp-orchestrate; no phantom CLI; skills language-agnostic |

---

## F053 Workstreams Status

| WS | Title | Status | Notes |
|----|-------|--------|-------|
| 00-053-01 | Verifier Interface Abstraction | Done | CoverageChecker, PathValidator, CommandRunner |
| 00-053-02 | Parser Frontmatter Bug | Done | — |
| 00-053-03 | Intent Schema | Done | — |
| 00-053-04 | Schema Path Consistency | Done | — |
| 00-053-05 | PromptOps Checks Formalization | Done | — |
| 00-053-06 | Emit Goroutine — Evidence Not Lost | Done | slog.Error on emission failure |
| 00-053-07 | Writer Hash Chain Atomicity | Done | flock, singleton, re-read under lock |
| 00-053-08 | Coverage_lang Error Handling | Done | — |
| 00-053-09 | Config guard.rules_file Path Validation | Done | — |
| 00-053-10 | Executor ParseDependencies + Context | Done | Safe fallback; ctx propagation |
| 00-053-11 | Oneshot sdp-orchestrate Reference | Done | PATH + go run documented |

---

## Beads Findings (F053, sdp-repo)

### New (this review)

- **sdp_dev-85g4** (P2): Executor has no production CLIRunner — caller must implement WorkstreamRunner
- **sdp_dev-iid9** (P2): Schema coding-workflow-statement $ref uses relative path — may fail when resolved from different dirs
- **sdp_dev-yxcg** (P2): Evals test_cases.jsonl — ensure schema paths resolve

### Existing (sdp .beads) — F053-addressed (closed)

| Bead | Original | F053 resolution |
|------|----------|-----------------|
| sdp-0mzg | Evidence slog.Error | 00-053-06: slog.Error added |
| sdp-gco1 | Writer singleton + flock | 00-053-07: getOrCreateWriter, flock |
| sdp-73th | Executor mock in prod | WorkstreamRunner interface, mock in test; CLIRunner = consumer impl |
| sdp-73sv | VerifyCoverage placeholder | qualityCoverageChecker adapter |
| sdp-9iex | VerifyCommands context.Background | VerifyCommands(ctx) uses parent |
| sdp-ixfh | Config Load no Validate | Load calls cfg.Validate() |
| sdp-r163 | coding-workflow-predicate missing | File exists |
| sdp-tsry | coding-workflow-predicate | File exists |
| sdp-v0g0 | config.schema guard/quality | Schema has both |
| sdp-vfg9 | Verifier no tests | verify_test.go exists |

### Still open (доработка → 00-053-12)

- **sdp-4jxa** (P1): Prompts few-shot examples for @review, @build, @idea
- **sdp-y7dg** (P1): Verifier context propagation — VerifyCommands uses ctx; defer cancel in loop fixed
- **sdp-yout** (P3): Evidence TestEmit flakiness

---

## Round 2 Beads (F053, round-2)

### P0 (blocking)

| ID | Role | Title |
|----|------|-------|
| sdp_dev-0ddg | Security | pipeline-hooks.yaml shell injection (sh -c) |

### P1 (blocking)

| ID | Role | Title |
|----|------|-------|
| sdp_dev-4rpn | QA | cmd/sdp-orchestrate 0% coverage |
| sdp_dev-c5fj | QA | internal/evidence 40% coverage |
| sdp_dev-qx8i | Security | sdp-evidence arbitrary file read |
| sdp_dev-a2iw | TechLead | Checkpoint data loss when ciloop saves |
| sdp_dev-l572 | Docs | F053 zero ws-verdict files |

### P2 (tracked)

sdp_dev-a9th, sdp_dev-akwg, sdp_dev-59z6, sdp_dev-pmod, sdp_dev-06a4, sdp_dev-luyz, sdp_dev-4736, sdp_dev-p7xv, sdp_dev-mk89, sdp_dev-0eba, sdp_dev-o62u, sdp_dev-9kg0, sdp_dev-m9i7, sdp_dev-ha5u, sdp_dev-c8rm, sdp_dev-6c73, sdp_dev-qjv4

**Product Review (2026-02-25) — additional P2:**

| ID | Title |
|----|-------|
| sdp_dev-dqem | Naming inconsistency sdp_dev vs sdp_dev (docs) |
| sdp_dev-rfh4 | Root directory 20+ binary files cleanup |
| sdp_dev-yi2i | Coverage files in root, gitignore |
| sdp_dev-ftrq | No unified logging strategy |
| sdp_dev-cugd | Path validation not used consistently |

### P3

sdp_dev-zd4q, sdp_dev-5x6s, sdp_dev-5402

**Product Review (2026-02-25) — additional P3:**

| ID | Title |
|----|-------|
| sdp_dev-3x92 | sdp/sdp-plugin vs sdp_dev/cmd boundary clarification |

**Note:** Some findings (sdp_dev-ywv8, sdp_dev-te1h, sdp_dev-1581, sdp_dev-1j9w, sdp_dev-eix0, sdp_dev-lo1j) may overlap with F053-addressed items; verify before remediation.

---

## Synthesis

- **Conflicts:** None
- **Rubber stamps:** None; all 7 roles provided evidence
- **Blocking:** P0 sdp_dev-0ddg (hooks shell injection); P1 sdp_dev-4rpn, sdp_dev-c5fj, sdp_dev-qx8i, sdp_dev-a2iw, sdp_dev-l572
- **Adversarial blind spots:** QA passed coverage thresholds but sdp quality reports 37.5%; Security hooks are repo-controlled but PR can add malicious hook before merge
