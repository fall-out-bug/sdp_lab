# Roadmap / INDEX / Beads Mismatch Report

> **Generated:** 2026-03-03  
> **Scope:** Inconsistencies between `docs/roadmap/ROADMAP.md`, `docs/workstreams/INDEX.md`, backlog workstream frontmatter, and Beads mapping.

---

## 1. Executive Summary

| Artifact | Updated | Findings |
|----------|---------|----------|
| ROADMAP.md | 2026-03-01 | References F053/F054 workstreams that do not exist; otherwise aligned with INDEX for feature-level status |
| INDEX.md | 2026-02-23 | Stale (8 days behind ROADMAP); workstream tables conflict with backlog frontmatter |
| Backlog files | Mixed | 20+ workstreams have `status` frontmatter contradicting INDEX; F059 workstreams have status/AC contradicting INDEX |
| .beads-sdp-mapping.jsonl | 2026-03-01 | 107 entries; count matches backlog file count (107) |

**Beads open issues:** `bd ready --json` was not available in this environment; open Beads status could not be verified.

---

## 2. Phantom Workstreams (ROADMAP → INDEX)

**Source:** `docs/roadmap/ROADMAP.md` lines 276–277

| Feature | ROADMAP Reference | Exists in INDEX? | Exists in backlog? | In mapping? |
|---------|-------------------|------------------|--------------------|-------------|
| **F053** | Phase 4 Beads Remediation (00-053-01 … 00-053-46) | No | No | No |
| **F054** | Continuous Protocol Improvement (00-054-01 … 00-054-06) | No | No | No |

**Impact:** ROADMAP says "See [INDEX](workstreams/INDEX.md)" for F053, but INDEX has no F053/F054 sections. No backlog files `00-053-*.md` or `00-054-*.md` exist.

**File references:**
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/docs/roadmap/ROADMAP.md` (lines 276–277)
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/docs/workstreams/INDEX.md` (no F053/F054)

---

## 3. Backlog Frontmatter vs INDEX (Status Conflicts)

### 3.1 Backlog says `done`, INDEX says `Backlog` (False-positive done)

These workstream files have `status: done` in frontmatter but INDEX lists them as Backlog. Acceptance criteria are **unchecked** in the files inspected (00-068-01, 00-077-01), indicating work is not complete.

| WS ID | Feature | Backlog status | INDEX status | File path |
|-------|---------|----------------|--------------|-----------|
| 00-068-01 | F068 | done | Backlog | `docs/workstreams/backlog/00-068-01.md` |
| 00-068-02 | F068 | done | Backlog | `docs/workstreams/backlog/00-068-02.md` |
| 00-068-03 | F068 | done | Backlog | `docs/workstreams/backlog/00-068-03.md` |
| 00-069-01 | F069 | done | Backlog | `docs/workstreams/backlog/00-069-01.md` |
| 00-069-02 | F069 | done | Backlog | `docs/workstreams/backlog/00-069-02.md` |
| 00-071-01 | F071 | done | Backlog | `docs/workstreams/backlog/00-071-01.md` |
| 00-071-02 | F071 | done | Backlog | `docs/workstreams/backlog/00-071-02.md` |
| 00-071-03 | F071 | done | Backlog | `docs/workstreams/backlog/00-071-03.md` |
| 00-072-01 | F072 | done | Backlog | `docs/workstreams/backlog/00-072-01.md` |
| 00-072-02 | F072 | done | Backlog | `docs/workstreams/backlog/00-072-02.md` |
| 00-072-03 | F072 | done | Backlog | `docs/workstreams/backlog/00-072-03.md` |
| 00-072-04 | F072 | done | Backlog | `docs/workstreams/backlog/00-072-04.md` |
| 00-073-01 | F073 | done | Backlog | `docs/workstreams/backlog/00-073-01.md` |
| 00-073-02 | F073 | done | Backlog | `docs/workstreams/backlog/00-073-02.md` |
| 00-073-03 | F073 | done | Backlog | `docs/workstreams/backlog/00-073-03.md` |
| 00-074-01 | F074 | done | Backlog | `docs/workstreams/backlog/00-074-01.md` |
| 00-077-01 | F077 | done | Backlog | `docs/workstreams/backlog/00-077-01.md` |
| 00-077-02 | F077 | done | Backlog | `docs/workstreams/backlog/00-077-02.md` |

**Note:** 00-071-01 has all AC checked in the file; INDEX still says Backlog. Either INDEX is stale (work done) or feature-level status overrides.

### 3.2 Backlog says `backlog`, INDEX says `Done` (Stale backlog)

ROADMAP and INDEX agree F059 is Done. Backlog files for 00-059-01 and 00-059-02 have `status: backlog` and unchecked AC.

| WS ID | Feature | Backlog status | INDEX status | File path |
|-------|---------|----------------|--------------|-----------|
| 00-059-01 | F059 | backlog | Done | `docs/workstreams/backlog/00-059-01.md` |
| 00-059-02 | F059 | backlog | Done | `docs/workstreams/backlog/00-059-02.md` |

**File references:**
- ROADMAP line 271: "F059 … **Status: DONE** (sdp-omc-guard, sdp-ready implemented)"
- INDEX line 60: F059 Done
- INDEX lines 222–225: 00-059-01 … 00-059-04 all Done

### 3.3 F076: Backlog says `backlog`, INDEX says `In Progress`

| WS ID | Feature | Backlog status | INDEX status | File path |
|-------|---------|----------------|--------------|-----------|
| 00-076-01 | F076 | backlog | In Progress | `docs/workstreams/backlog/00-076-01.md` |

Backlog file has partial AC checked (5/6). INDEX is likely correct.

---

## 4. Feature-Level Status Summary

| Feature | ROADMAP | INDEX | Backlog aggregate | Conflict? |
|---------|---------|-------|-------------------|-----------|
| F059 | Done | Done | 00-059-01,02: backlog; 03,04: done | Yes (01,02 stale) |
| F062 | Not in Done | Backlog | 00-062-01 done, 02–03 backlog | No |
| F063 | Not in Done | Backlog | 00-063-01,02 done, 03,04 backlog | No |
| F068 | Not in Done | Backlog | All done (frontmatter) | Yes (frontmatter wrong) |
| F069 | Not in Done | Backlog | 01,02 done (frontmatter) | Yes (frontmatter wrong) |
| F071 | Not in Done | Backlog | All done (frontmatter) | Yes (frontmatter wrong) |
| F072 | Not in Done | Backlog | All done (frontmatter) | Yes (frontmatter wrong) |
| F073 | Not in Done | Backlog | All done (frontmatter) | Yes (frontmatter wrong) |
| F074 | Not in Done | Backlog | 00-074-01 done (frontmatter) | Yes (frontmatter wrong) |
| F076 | Not in Done | In Progress | backlog | Yes (frontmatter wrong) |
| F077 | Not in Done | Backlog | 01,02 done (frontmatter) | Yes (frontmatter wrong) |

---

## 5. Beads Mapping Validation

- **Rule (AGENTS.md):** `wc -l .beads-sdp-mapping.jsonl` must equal `ls docs/workstreams/backlog/*.md | wc -l`
- **Result:** 107 = 107 ✓
- **Missing mappings:** None. F053/F054 workstreams do not exist, so no mapping expected.

---

## 6. Suggested Canonical Source-of-Truth Rules

| Rule | Rationale |
|------|-----------|
| **1. INDEX.md is the workstream status authority** | INDEX is the single table of record for WS status. Backlog frontmatter should be derived from or synced to INDEX. |
| **2. ROADMAP.md is the feature-level and phase authority** | ROADMAP defines feature status (Done/Backlog/In Progress) and phase boundaries. INDEX should reflect ROADMAP for feature-level status. |
| **3. Backlog frontmatter `status` must match INDEX** | `sdp-protocol-check` or `sdp-doc-sync` should validate and optionally fix backlog `status` from INDEX. |
| **4. ROADMAP must not reference non-existent workstreams** | F053/F054 either need backlog files + INDEX entries, or ROADMAP should be updated to remove/qualify the references. |
| **5. Beads status flows from INDEX** | When `bd sync` runs, Beads should reflect INDEX status. Open Beads issues should align with INDEX "Backlog" and "In Progress" workstreams. |
| **6. Acceptance criteria drive done** | A workstream is Done only when all AC are checked. Frontmatter `status: done` with unchecked AC is invalid. |

---

## 7. Recommended Actions (Read-Only; No Edits)

1. **Fix F053/F054:** Either create `00-053-*.md` and `00-054-*.md` in backlog and add INDEX entries, or update ROADMAP to remove "See INDEX" and clarify these are planned (not yet workstreamed).
2. **Sync backlog frontmatter to INDEX:** For all 20 conflicting workstreams, set `status` in backlog files to match INDEX.
3. **Correct false-positive `done`:** For 00-068, 00-069, 00-071, 00-072, 00-073, 00-074-01, 00-077-01, 00-077-02, set `status: backlog` in frontmatter (or verify work and update INDEX if truly done).
4. **Correct stale F059 backlog:** Set `status: done` in 00-059-01.md and 00-059-02.md.
5. **Correct F076:** Set `status: in_progress` in 00-076-01.md.
6. **Update INDEX date:** Set "Updated" to 2026-03-01 or later to match ROADMAP.
7. **Add validation to sdp-protocol-check:** Check backlog `status` frontmatter vs INDEX; flag mismatches.

---

## 8. File Reference Index

| File | Purpose |
|------|---------|
| `/home/fall_out_bug/projects/vibe_coding/sdp_lab/docs/roadmap/ROADMAP.md` | Feature/phase status, F053/F054 references |
| `/home/fall_out_bug/projects/vibe_coding/sdp_lab/docs/workstreams/INDEX.md` | Workstream tables, feature status |
| `/home/fall_out_bug/projects/vibe_coding/sdp_lab/docs/workstreams/backlog/*.md` | Per-WS frontmatter, AC |
| `/home/fall_out_bug/projects/vibe_coding/sdp_lab/.beads-sdp-mapping.jsonl` | WS ID ↔ Beads ID mapping |
