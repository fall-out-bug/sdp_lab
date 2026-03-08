# SDP CLI UX Exploration Report

**Scope:** Public SDP CLI in `sdp/sdp-plugin/cmd/sdp`  
**Date:** 2026-03-08  
**Method:** Codebase search (read, glob, grep). No edits, no speculation beyond observed code/docs.

---

## 1. Command Clusters

### 1.1 Top-Level Commands (39 registered in `main.go`)

| Cluster | Commands | Purpose |
|--------|----------|---------|
| **Setup & Health** | `init`, `doctor`, `hooks`, `completion` | Project bootstrap, environment check |
| **Workstream Lifecycle** | `plan`, `apply`, `build`, `verify`, `parse`, `drift` | Create → Execute → Verify |
| **Guard & Scope** | `guard`, `collision`, `acceptance` | Edit scope, branch validation |
| **Evidence & Log** | `log`, `deploy`, `checkpoint` | Evidence chain, deployment approval |
| **Skills & Quality** | `skill`, `quality`, `tdd`, `watch` | Skill validation, quality gates |
| **Planning & Design** | `design`, `idea`, `prd`, `prototype` | Requirements, workstream design |
| **Task & Session** | `beads`, `task`, `session`, `resolve` | Beads integration, session state |
| **Observability** | `status`, `metrics`, `telemetry`, `decisions` | Status, metrics, telemetry |
| **Advanced** | `contract`, `orchestrate`, `memory`, `coordination`, `health`, `diagnose`, `next`, `git` | Contracts, orchestration, diagnostics |

### 1.2 Subcommand Depth

- **Shallow (1–2 subcommands):** `init`, `doctor`, `hooks`, `plan`, `apply`, `build`, `verify`, `parse`, `deploy`, `prd`, `drift`, `watch`, `completion`, `status`, `next`, `health`, `diagnose`
- **Deep (4+ subcommands):** `guard` (activate, check, status, deactivate, context, branch, finding), `log` (show, export, stats, trace), `skill` (validate, check-all, list, show, record), `session` (init, sync, repair, show, delete), `telemetry` (status, export, upload, disable, enable, analyze, consent), `checkpoint` (create, resume, list, clean), `contract` (synthesize, generate, lock, validate, verify), `decisions` (list, search, export, log), `metrics` (collect, classify, report), `memory` (index, search, stats), `coordination` (events, stats, verify)

---

## 2. Strongest UX Paths

### 2.1 Happy Path (Documented & Tested)

**Path:** `init` → `plan` → `apply` → `status` / `verify` → `log show`

- **Files:** `sdp/docs/QUICKSTART.md`, `sdp/docs/CLI_REFERENCE.md`, `sdp/sdp-plugin/templates/minimal-go/README.md`
- **Integration tests:** `e2e_integration_test.go` (plan → apply → trace), `workflow_integration_test.go` (plan, apply, log)
- **Help quality:** `plan`, `apply`, `log` have Long + Example blocks
- **Docs alignment:** QUICKSTART, CLI_REFERENCE, minimal-go template all reference this flow

### 2.2 First-Run Path

**Path:** `init --auto` or `init --guided` → `doctor` → `status`

- **Files:** `sdp/docs/QUICKSTART.md`, `sdp/sdp-plugin/README.md`, `sdp/sdp-plugin/docs/reference/2026-02-16-f068-ux-baseline.md`
- **Integration tests:** `cmd_integration_test.go` (TestInitCommand, TestDoctorCommand)
- **Help quality:** `init` has detailed Long (modes, preflight, env vars), `doctor` has environment checks

### 2.3 Guard + Build Path

**Path:** `guard activate <ws-id>` → `build <ws-id>` (or `@build` via skill)

- **Files:** `sdp/sdp-plugin/README.md`, `sdp/CLAUDE.md`, `build.go`, `guard.go`
- **Help quality:** `guard` and `build` have clear Use/Short; `build` Long references `@build` and `sdp-orchestrate`

---

## 3. Weakest UX Seams

### 3.1 Root Help Mismatch (Critical)

**File:** `sdp/sdp-plugin/cmd/sdp/main.go` lines 23–48

The root `Long` description lists only 6 commands:

```
init, doctor, hooks, watch, checkpoint, completion
```

But `main.go` registers **39** top-level commands. Users running `sdp --help` see a truncated mental model; commands like `plan`, `apply`, `status`, `guard`, `log` are not mentioned.

### 3.2 Demo Discoverability Gap

**Files:** `sdp/sdp-plugin/cmd/sdp/demo.go`, `sdp/sdp-plugin/cmd/sdp/main.go`, `sdp/sdp-plugin/internal/ui/completion_*.go`

`demoCmd()` is implemented and now registered via `rootCmd.AddCommand(demoCmd())`, so the CLI entrypoint exists. The remaining gap is discoverability: shell completion and some planning notes lag the live command tree, which still makes `sdp demo` easy to miss.

### 3.3 Help Integration Test Incomplete

**File:** `sdp/sdp-plugin/cmd/sdp/help_integration_test.go` lines 95–121

`TestCommandsCoverage` expects 19 commands and omits 20+ that exist:

- **Missing:** `apply`, `plan`, `build`, `log`, `init`, `collision`, `contract`, `deploy`, `design`, `diagnose`, `idea`, `git`, `health`, `memory`, `metrics`, `next`, `prototype`, `coordination`, `resolve`, `session`, `task`
- **Incorrect:** `help` is not a top-level command (Cobra built-in)

### 3.4 Docs vs CLI Disconnect

| Doc | Commands Emphasized | CLI Reality |
|-----|---------------------|-------------|
| `sdp/README.md` | `@feature`, `@oneshot`, `@review`, `@deploy` | Skills, not CLI; CLI is "optional" |
| `sdp/docs/QUICKSTART.md` | `sdp init`, `sdp verify`, `sdp status`, `sdp log show` | Matches |
| `sdp/docs/CLI_REFERENCE.md` | 11 commands | Matches subset |
| `sdp/sdp-plugin/README.md` | doctor, status, init, guard, parse, verify, tdd, telemetry | Omits plan, apply, log, checkpoint |
| `sdp/sdp-plugin/docs/NAVIGATION.md` | Skills (@feature, @build, etc.) | No CLI command tree; skills ≠ CLI |
| `sdp/sdp-plugin/docs/TUTORIAL.md` | `sdp init`, `@feature`, `@build`, `@review` | Mixes CLI and skills |

Quickstart examples (`docs/examples/go/QUICKSTART.md`, etc.) focus on **skills** (`@feature`, `@design`, `@build`, `@review`), not CLI. The CLI `plan`/`apply` flow is documented in `templates/minimal-go/README.md` and `sdp/docs/CLI_REFERENCE.md` but not in language quickstarts.

### 3.5 Skill vs CLI Overlap

- **Skills:** `@feature`, `@design`, `@build`, `@oneshot`, `@review`, `@deploy` — invoked from IDE
- **CLI equivalents:** `plan`, `apply`, `build`, `orchestrate`, (no direct `review`/`deploy` CLI for full flow)
- **Gap:** `sdp plan` and `sdp apply` exist and are tested, but QUICKSTART and NAVIGATION emphasize skills. Users may not discover the CLI-only path.

### 3.6 `sdp git` and Session Coupling

**File:** `sdp/sdp-plugin/cmd/sdp/git_wrapper.go`

`sdp git` requires `sdp session init --feature=F###` first. Error message explains this, but there is no `sdp session` in the root Long or in most quickstart docs. Session is an advanced concept; `git` fails with a session error if not initialized.

---

## 4. Integration Test Coverage

| Test File | Commands Covered |
|-----------|------------------|
| `help_integration_test.go` | version, help, doctor, commands list (incomplete) |
| `cmd_integration_test.go` | init, checkpoint, beads |
| `e2e_integration_test.go` | plan (dry-run, JSON), apply (trace) |
| `workflow_integration_test.go` | plan, apply, log (help, JSON parse) |
| `telemetry_integration_test.go` | telemetry |
| `parse_integration_test.go` | parse |
| `decisions_integration_test.go` | decisions |
| `drift_integration_test.go` | drift |
| `log_show_export_integration_test.go` | log show, log export |
| `log_trace_stats_integration_test.go` | log trace, log stats |
| `quality_integration_test.go` | quality |
| `skill_tdd_integration_test.go` | skill, tdd |

**Gaps:** No integration tests for `guard`, `build`, `status`, `design`, `idea`, `prototype`, `next`, `diagnose`, `health`, `session`, `git`, `memory`, `coordination`, `contract`, `orchestrate`.

---

## 5. Improvement Themes (Grounded in File Paths)

### Theme 1: Align Root Help with Reality

**Files:** `sdp/sdp-plugin/cmd/sdp/main.go`

- Update root `Long` to reflect all command clusters or at least the primary paths (init, doctor, plan, apply, status, log, guard, checkpoint, completion).
- Consider grouping in help output (e.g., "Setup:", "Execution:", "Evidence:") or adding a "See also" section for advanced commands.

### Theme 2: Register and Promote Demo

**Files:** `sdp/sdp-plugin/cmd/sdp/main.go`, `sdp/sdp-plugin/cmd/sdp/demo.go`

- Keep `rootCmd.AddCommand(demoCmd())` in place and protect it with help/completion coverage.
- Ensure QUICKSTART and NAVIGATION reference `sdp demo` as the "first success" path (F068-04).

### Theme 3: Fix and Extend Help Integration Test

**Files:** `sdp/sdp-plugin/cmd/sdp/help_integration_test.go`

- Sync `expectedCommands` with the full `main.go` AddCommand list (or derive from Cobra).
- Remove `help` from expected top-level commands.
- Optionally add subcommand presence checks for high-value commands (e.g., `log show`, `guard activate`).

### Theme 4: Unify CLI vs Skill Documentation

**Files:** `sdp/README.md`, `sdp/docs/QUICKSTART.md`, `sdp/sdp-plugin/docs/NAVIGATION.md`, `sdp/sdp-plugin/docs/examples/*/QUICKSTART.md`

- Add a "CLI-only path" section: `sdp init --auto` → `sdp plan "feature"` → `sdp apply` → `sdp status`.
- In NAVIGATION decision trees, add CLI equivalents next to skills (e.g., "`@build 00-001-01` or `sdp build 00-001-01`").
- Ensure `sdp/sdp-plugin/README.md` lists `plan`, `apply`, `log` alongside init/doctor/guard.

### Theme 5: Progressive Disclosure in Help

**Files:** `sdp/sdp-plugin/cmd/sdp/main.go`, individual command files

- Root help is overwhelming (39 commands). Options:
  - Add `sdp --help-short` or `sdp quickstart` that shows only init, plan, apply, status, log.
  - Use Cobra's `SuggestFor` or annotations to surface "essential" vs "advanced" in help.
- Ensure `sdp next` is discoverable (it recommends next action; good for onboarding).

### Theme 6: Session/Git Onboarding

**Files:** `sdp/sdp-plugin/cmd/sdp/git_wrapper.go`, `sdp/sdp-plugin/cmd/sdp/session.go`, docs

- When `sdp git` fails due to missing session, consider suggesting `sdp session init` with a concrete example.
- Document `session` in a "Git workflow" or "Advanced" section so users know when it applies.

---

## 6. File Reference Summary

| Category | Key Files |
|----------|-----------|
| **CLI entry** | `sdp/sdp-plugin/cmd/sdp/main.go` |
| **Root help** | `main.go` lines 19–48 |
| **Command defs** | `sdp/sdp-plugin/cmd/sdp/*.go` (135 files) |
| **Integration tests** | `*_integration_test.go`, `*_test.go` in `cmd/sdp/` |
| **Docs** | `sdp/README.md`, `sdp/docs/QUICKSTART.md`, `sdp/docs/CLI_REFERENCE.md`, `sdp/sdp-plugin/README.md`, `sdp/sdp-plugin/docs/NAVIGATION.md`, `sdp/sdp-plugin/docs/TUTORIAL.md` |
| **Quickstarts** | `sdp/sdp-plugin/docs/examples/{go,python,java}/QUICKSTART.md` |
| **Plugin manifest** | `sdp/sdp-plugin/plugin.json` |
| **Template** | `sdp/sdp-plugin/templates/minimal-go/README.md` |

---

## 7. Conclusion

The SDP CLI has a solid core path (`init` → `plan` → `apply` → `status`/`log`) with good help text and integration tests. The main UX issues are:

1. **Root help lies** — lists 6 commands, registers 39.
2. **Demo is under-promoted** — implemented and registered, but still easy to miss.
3. **Help test is stale** — misses half the commands.
4. **Docs favor skills** — CLI `plan`/`apply` path is under-promoted.
5. **Session/git** — advanced flow with unclear onboarding.

A cohesive CLI improvement pack should address these five areas before adding new features.
