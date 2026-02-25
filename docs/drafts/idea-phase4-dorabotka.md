# Idea: F053 Phase 4 Доработка — Product Review Continuation

**Source:** [F053-REVIEW-SUMMARY.md](../reviews/F053-REVIEW-SUMMARY.md) Product Review (2026-02-25)  
**Scope:** sdp_dev (internal/, cmd/, docs/) + sdp submodule  
**Feature:** F053 (extended)  
**Design:** `/design phase4-dorabotka`

---

## Context

F053 workstreams 00-053-01..37 exist. Product Review (2026-02-25) identified additional beads not yet covered by workstreams. All findings planned as continuation of F053.

---

## Remaining Findings → Workstreams

### P1 (blocking)

| Bead | Title | WS |
|------|-------|-----|
| sdp_dev-dqem | Naming inconsistency sdp_dev (resolved) | 00-053-38 |

### P2 (tracked)

| Bead | Title | WS |
|------|-------|-----|
| sdp_dev-rfh4 | Root directory 20+ binary files cleanup | 00-053-39 |
| sdp_dev-yi2i | Coverage files in root, gitignore | 00-053-40 |
| sdp_dev-ftrq | No unified logging strategy | 00-053-41 |
| sdp_dev-cugd | Path validation not used consistently | 00-053-42 |

### P3

| Bead | Title | WS |
|------|-------|-----|
| sdp_dev-3x92 | sdp/sdp-plugin vs sdp_dev/cmd boundary clarification | 00-053-43 |

---

## Already Covered (00-053-16..37)

- 00-053-16: Pipeline Hooks Shell Injection (sdp_dev-0ddg) — Done
- 00-053-17: sdp-evidence Path Validation (sdp_dev-qx8i)
- 00-053-18: Checkpoint Data Loss (sdp_dev-a2iw)
- 00-053-19: sdp-orchestrate Coverage (sdp_dev-4rpn)
- 00-053-20: internal/evidence Coverage (sdp_dev-c5fj)
- 00-053-21: F053 ws-verdict Files (sdp_dev-l572) — Done
- 00-053-22..37: Various Round 2 beads

---

## Dependencies

- 00-053-38..43 — no cross-deps; can run in parallel
- 00-053-42 (path validation) — may touch same files as 00-053-17, 00-053-28; coordinate if needed
