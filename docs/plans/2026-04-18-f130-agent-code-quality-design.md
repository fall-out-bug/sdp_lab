# F130 — AI Harness Config Auto-Provisioning

> **Status:** In Progress
> **Created:** 2026-04-18
> **Author:** Andrei
> **Depends on:** F120, F122, F123, F124, F127

---

## 1. Problem

AI harnesses generate code that drifts from project conventions because rules files are hand-written, incomplete, and not grounded in the actual codebase. The root causes:

- **Manual authoring doesn't scale.** A rules file written by one developer at one point in time reflects what that person noticed, not what the codebase actually enforces. Gaps are discovered at review, not at write time.
- **Each harness needs its own format.** Claude Code reads CLAUDE.md sections; Cursor reads `.cursorrules`; Codex CLI reads `codex.yaml`; OpenCode reads AGENTS.md. Maintaining four files in sync by hand is impractical.
- **No feedback loop between code and rules.** When the codebase evolves, rules files don't. Patterns extracted manually go stale. Agents following a stale rules file drift from actual project conventions.
- **Cold start is the worst case.** Every session starts from zero. Without rules grounded in the live codebase, each agent re-derives conventions from whatever files it reads first, producing divergent style across sessions and harnesses.

Result: recurring review cycles, style inconsistency across agents, and agents that adapt to what they see rather than what the project enforces. The problem scales with the number of harnesses and the maturity of the repo.

`go-patterns.md` (created 2026-04-18) is a manually-authored reference example of what the generator should produce. `sdp-healthcheck` is an example of a tool the bootstrap process can generate or invoke. These are the target output format, not the feature deliverables.

---

## 2. Solution

SDP analyzes the repo, extracts real patterns from code and git history, and generates harness-ready config files automatically. One source of truth — the actual codebase — produces N harness-specific files.

**Key properties:**

- **Grounded in code, not opinion.** Naming conventions come from identifier analysis. Architecture patterns come from interface/struct relationships. Good/bad examples come from real code and git history.
- **One extraction, N outputs.** A single pattern report is the intermediate representation. Harness adapters transform it into the target format for each harness.
- **Lifecycle-aware.** The pipeline handles greenfield repos (generate from scratch), brownfield repos with recent AI activity (generate + respect existing), and mature repos (diff and update only deltas — no full overwrites).
- **Bootstrap-integrated.** Pattern extraction and file generation are wired into `sdp bootstrap` (F124 flow), so provisioning happens at the same moment the repo is onboarded to SDP.

`go-patterns.md` (created 2026-04-18) is the reference output format for the rules generator. Any generated rules file for a Go repo should match that format: good patterns with short examples, antipatterns with explicit "do NOT" callouts, architecture patterns, and a canonical file template.

---

## 3. Pipeline

```
Repo
 │
 ├── F120 Scout        ─┐
 ├── F122 Index        ─┼─► Pattern Extractor (00-F130-02)
 └── F123 Spec Recovery─┘
                          │
                          ▼
                   Pattern Report (structured JSON)
                          │
                          ▼
                   Rules Generator (00-F130-03)
                          │
                          ▼
                   Language Rules File (e.g. go-patterns.md)
                          │
                          ▼
                   Harness Adapter (00-F130-04)
                    │         │         │         │
                    ▼         ▼         ▼         ▼
               CLAUDE.md  .cursorrules AGENTS.md codex.yaml
               (section)  (Cursor)  (lang section) (rules block)
                    └─────────┴─────────┴─────────┘
                                   │
                                   ▼
                         sdp bootstrap (00-F130-05, F124)
```

Each stage is independently testable. The pattern report JSON is the contract between extraction and generation. The rules file is the contract between generation and adaptation.

---

## 4. Workstreams

| ID | Title | Description |
|----|-------|-------------|
| 00-F130-01 | Config provisioning spec | Define "complete harness config": what files (CLAUDE.md sections, .cursorrules, AGENTS.md language section, codex.yaml), what harnesses (Claude Code, Cursor, Codex, OpenCode), what lifecycle stages (greenfield / brownfield-new / brownfield-mature). Produce JSON schema for the config manifest. |
| 00-F130-02 | Pattern extraction engine | Extend sdp scout / sdp index to extract: naming conventions (from identifiers), architecture patterns (from interface/struct relationships), good/bad code examples (from code + git history), typical file structure. Output: structured pattern report. |
| 00-F130-03 | Rules file generator | Generate language-specific patterns doc (go-patterns.md is the reference output format) from the extracted pattern report. Validate examples against actual code. Output: versioned rules file per language. |
| 00-F130-04 | Harness adapter layer | Transform the generated rules file into harness-specific formats: .cursorrules (Cursor), CLAUDE.md code-standards section (Claude Code), AGENTS.md language section, codex.yaml rules block. One source → N harness files. |
| 00-F130-05 | Bootstrap integration | Wire pattern extraction + file generation into sdp bootstrap (F124 flow). Support re-run on mature repos (drift detection: re-extract and diff against current rules). |

---

## 5. Lifecycle Stages

| Stage | Trigger | Behavior |
|-------|---------|----------|
| **greenfield** | New repo, no existing harness config files | Generate all files from scratch. No existing content to preserve. |
| **brownfield-new** | Repo has recent AI activity but no established config files | Generate and merge. Respect any existing CLAUDE.md sections; add generated language section without overwriting manual content. |
| **brownfield-mature** | Repo has existing harness config files (hand-written or previously generated) | Diff mode only. Re-extract patterns, diff against current rules, write only the delta. Never overwrite the full file. |

The lifecycle stage is detected automatically from the presence and age of existing config files and git history depth.

---

## 6. Acceptance Criteria

- [ ] `sdp bootstrap` generates harness configs from codebase analysis (not templates)
- [ ] Supports Claude Code, Cursor, Codex CLI harnesses at minimum
- [ ] Generated `go-patterns.md` matches format of manually-created reference (`docs/reference/go-patterns.md`, created 2026-04-18)
- [ ] Re-run on mature repo produces a diff, not a full overwrite
- [ ] Integration tests use `testing.Short()` guard

---

## 7. Dependencies

| Feature | Rationale |
|---------|-----------|
| **F120** — Scout | Provides the initial repo card and file inventory that seeds the pattern extractor. |
| **F122** — Index | Provides the persistent `.sdp/index.db` used by the pattern extractor to query identifiers and relationships without re-scanning. |
| **F123** — Spec Recovery | Provides extracted contracts, rules, and invariants that the pattern extractor uses as additional signal beyond raw identifier analysis. |
| **F124** — Bootstrap | The integration point: pattern extraction and harness file generation are wired into the `sdp bootstrap` flow as the "AI rules" provisioning step. |
| **F127** — Multi-harness modernization | Provides the harness format knowledge (what each harness reads, how it reads it, naming conventions per harness) that the adapter layer requires. |

---

## 8. Non-Goals

The following are explicitly out of scope for F130:

- **Runtime code review** — F130 generates config files at bootstrap time, not during live coding sessions.
- **Linter configuration** — `.golangci.yml` authoring and linter rule tuning are not part of this feature. F130 produces human-readable pattern guidance, not machine-executable lint rules.
- **Security scanning** — no SAST, dependency vulnerability scanning, or supply-chain checks. Those belong to the trust lane (F064–F067, F078).
- **Non-Go languages for MVP** — F130 ships Go pattern extraction and generation first. Other languages are future work.
- **CI enforcement** — wiring generated rules into CI quality gates is a follow-on concern (00-F130-05+). The MVP generates and installs files; it does not gate on them.
