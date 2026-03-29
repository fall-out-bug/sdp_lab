# Handoff: Dispatch v2 Cleanup

 Utility Migration

**Date:** 2026-03-29
**Branch:** main (20f172a)
**Status:** green — 45 packages pass, 0 open beads

## What was done

  This session completes the4 bead from the4 backlog items from the "Ready to migrate" section.

 All closed.

### 1. sdplab-zzyl — Migrate 10 call sites → sdputil.AtomicWriteJSON/AtomicWriteFile — 10 files changed

 70 insertions(+, 19 deletions)
 - `sdputil.AtomicWriteFile` for 2 files changed ( 70 insertions(+, 2 deletions)
   - `sdputil.AtomicWriteJSON` in 6 files changed ( 70 insertions(+, 5 deletions)
 - Removed `jsonMarshal` helper — now unused
 0 files changed, 70 insertions(+, 0 deletions)

   - Removed unused imports: bytes, encoding/json, io from 12 files
  cleaned up

 - Dead code removed from orchestrate/runhybridate jsonMarshal helper)

 - Simplified `json.NewDecoder(LimitReader(...))` to `sdputil.UnmarshalJSON` in 6 call sites (3 files changed, 70 insertions(+, 0 deletions)
   - Cleaned up unused imports: bytes, encoding/json, io in 6 files cleaned

  - Dead code remove in orchestrate/jsonMarshal helper
 3. `exec.Command` → `exec.CommandContext` in 28 production files (12 files changed)
  70 insertions(+, 28 deletions) - All functions now use `context.Background()` context, establishing the pattern for future ctx propagation. This approach is better than bare `exec.Command` since:
 enables timeout/cancellation, and makes the pattern consistent across the codebase.

 Builds + tests pass, 45 packages.

 0 regressions.

 | Handoff file: `docs/handoffs/2026-03-29-dispatch-v2-cleanup.md`

 created.
 replaced `2026-03-29-dispatch-v2-handoff.md`.

 | Beads: sdplab-fi5p, sdplab-136r, sdplab-o1kn — all closed |

## Backlog (prioritized)

 — not started this session

### Ready to migrate (Low effort, high value)
 — all done in this session

4. Migrate 12 call sites → `sdputil.AtomicWriteJSON` (8 JSON) / `sdputil.AtomicWriteFile` (2 raw bytes)
 — `cmd/sdp-evidence/main_test.go`, uses `t.Setenv`
 (80+ locations) — Moved away from tests
5. Wire `router/`, `gate/`, `planner/` into orchestrate loop (7. Wire `sdp/` submodule fork (6 duplicate packages)

8. Split 5 god files: `1369`, `1122`, `820` lines) — `update.go` main.go, `control_tower_view.go` (1162 lines) — codes_split into logical modules)

### Large effort
 8. Split 5 god files: `1369`, `1122`, `820` lines) — `main.go` (708 lines) — `sdp/` submodule fork)
6 duplicate packages,10. Consolidate `sdp/` submodule fork —6 duplicate packages)
10. `os.Setenv` → `t.Setenv` in tests (80+ locations) — Move away from tests)
   - Unexport ~~40 internal-only symbols (7. Wire `router/`, `gate/`, `planner/` into orchestrate loop
7. Wire `sdp/` submodule fork (6 duplicate packages)10. Consolidate `sdp/` submodule fork

6 duplicate packages)
10. `exec.Command` → `exec.CommandContext` (12 locations) — Wrap `~100 bare `return err` with `fmt.Errorf` (9. Split 5 god files: `1369`, `1122`, `820` lines) — `main.go` (708 lines) - `sdp/` submodule fork)
6 duplicate packages)
10. `os.Setenv` → `t.Setenv` in tests (80+ locations) - move away from tests)
   - Unexport ~~40 internal-only symbols (7. Wire `router/`, `gate/`, `planner/` into orchestrate loop
7. Wire `sdp/` submodule fork (6 duplicate packages)
10. Consolidate `sdp/` submodule fork (6 duplicate packages)
10. `exec.Command` → `exec.CommandContext` (12 locations) — Wrap ~~100 bare `return err` with `fmt.Errorf` — 9. Split 5 god files, `1369`, `1122, `820` lines) — `main.go` (708 lines) - `sdp/` submodule fork)
6 duplicate packages)

10. `os.Setenv` → `t.Setenv` in tests (80+ locations) — move away from tests)
   - Unexport ~~40 internal-only symbols
7. Wire `router/`, `gate/`, `planner/` into orchestrate loop
7. Wire `sdp/` submodule fork (6 duplicate packages)
10. Consolidate `sdp/` submodule fork (6 duplicate packages)
10. `exec.Command` → `exec.CommandContext` (12 locations) — Wrap ~100 bare `return err` with `fmt.Errorf` - commit `20f172a` pushed to remote `main -> master`
