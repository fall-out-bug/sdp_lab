# Pre-existing Phantom CLI Analysis (2026-02-23)

## Summary

Skills and agents reference `sdp guard` subcommands that **do not exist** in the installed sdp CLI. The real sdp CLI (homebrew) provides only: `activate`, `check`, `deactivate`, `status`.

## What Exists vs What's Referenced

| Command | Exists? | Referenced In |
|---------|---------|---------------|
| `sdp guard activate` | ✅ Yes | build skill, guard skill |
| `sdp guard check` | ✅ Yes | guard skill |
| `sdp guard status` | ✅ Yes | guard skill |
| `sdp guard deactivate` | ✅ Yes | guard skill |
| `sdp guard context check` | ❌ No | build skill, orchestrator, hooks |
| `sdp guard context go $FEATURE_ID` | ❌ No | orchestrator |
| `sdp guard branch check --feature=$ID` | ❌ No | build skill |
| `sdp guard complete <ws-id>` | ❌ No | build skill |
| `sdp guard finding list` | ❌ No | sdp/CLAUDE.md |
| `sdp guard finding resolve <id>` | ❌ No | sdp/CLAUDE.md |

## Impact

### 1. Build Skill (`sdp/prompts/skills/build/SKILL.md`)

```bash
sdp guard context check 2>/dev/null || true
sdp guard branch check --feature=$FEATURE_ID 2>/dev/null || true
...
sdp guard complete 00-067-01 2>/dev/null || true
```

- **context check** — intended: verify agent is in correct feature context (branch/checkpoint)
- **branch check** — intended: verify current branch matches checkpoint
- **complete** — intended: mark workstream done, clear guard scope

All use `2>/dev/null || true` → **silent no-op**. No runtime failure, but agent never gets the intended validation.

### 2. Orchestrator Agent (`sdp/prompts/agents/orchestrator.md`)

```
2. Run: `sdp guard context check`
3. If check fails: Run: `sdp guard context go $FEATURE_ID`
```

- **context check** — intended: pre-git validation that CWD/branch are correct
- **context go** — intended: fix context (e.g. checkout correct branch)

Without these, orchestrator has no CLI-based context validation. The design doc (oneshot-autonomous-design.md) already says: "Defensive branch check через checkpoint (не через `sdp guard context go`)" — use checkpoint, not phantom command.

### 3. Hooks (`hooks/pre-build.sh`, `scripts/hooks/pre-build.sh`)

```bash
sdp guard context check 2>/dev/null || true
sdp guard activate "$WS_ID" 2>/dev/null || true
```

- **context check** — no-op
- **activate** — ✅ exists, works

### 4. sdp/CLAUDE.md

Documents `sdp guard finding list` and `sdp guard finding resolve`. Per **00-018-02** these were to be stripped (phantom). The grep validation in 00-018-02 only checked `collision|contract|memory|resolve|parse` — not `guard finding`. So CLAUDE.md was never updated.

## Root Cause

1. **Design evolution**: The oneshot design moved to checkpoint-based branch validation. `sdp guard context` and `sdp guard branch check` were never implemented — the design chose checkpoint instead.
2. **00-018-02 scope gap**: Phantom removal targeted `sdp guard finding` in the guard skill, but CLAUDE.md (full CLI reference) was out of scope.
3. **Skill/agent drift**: Build skill and orchestrator still reference the old, unimplemented commands. The `2>/dev/null || true` pattern was added to avoid failures, but the commands were never built.

## Alignment with Roadmap

**Роадмап Phase 0 явно указывает:**

- **Exit criteria** (ROADMAP.md:122): "Zero phantom CLI commands in any skill"
- **F018** (00-018-02): Remove phantom commands — `sdp guard finding` в scope, context/branch/complete — нет
- **Agent Loop Reliability**: "External enforcement" — flow control в CLI, не в промптах
- **Oneshot design** (840): "Defensive branch check через checkpoint (не через `sdp guard context go`)"
- **Oneshot design** (886): "bd verification query вместо `sdp guard finding list`"

**Вывод:** Роадмап не планирует реализацию context/branch/complete/finding. Планируется **замена на checkpoint + bd**. Вариант A согласован с роадмапом. Вариант B (реализовать) противоречит дизайну.

---

## Recommended Fixes

### Option A: Replace with existing mechanisms (minimal) — **РЕКОМЕНДУЕТСЯ**

| Phantom | Replacement |
|---------|-------------|
| `sdp guard context check` | Checkpoint branch check (already in build skill lines 39-44) |
| `sdp guard context go $FEATURE_ID` | `git checkout $(jq -r .branch .sdp/checkpoints/${FEATURE_ID}.json)` |
| `sdp guard branch check --feature=$ID` | Same as above — redundant with checkpoint check |
| `sdp guard complete` | Remove — `sdp guard deactivate` exists, or no-op (guard clears on next activate) |
| `sdp guard finding list/resolve` | Remove from CLAUDE.md; use `bd list` for findings |

### Option B: Implement missing commands in sdp CLI — **НЕ СОГЛАСОВАН С РОАДМАПОМ**

Add to sdp-plugin or sdp_dev:
- `sdp guard context check` — read checkpoint, verify branch
- `sdp guard context go <feature>` — checkout branch from checkpoint
- `sdp guard branch check --feature=<id>` — alias for context check
- `sdp guard complete <ws-id>` — deactivate + optional ws-verdict trigger

### Option C: Hybrid — **ЧАСТИЧНО A + CLAUDE.md**

- Remove phantom refs from skills/agents (Option A for context/branch/complete)
- Remove `sdp guard finding` from CLAUDE.md (00-018-02 completion)
- Document that `sdp-guard --ws` (F023) is the scope checker; `sdp guard` is the edit-time scope (different tools)

## Files to Update (Option A)

| File | Change |
|------|--------|
| `sdp/prompts/skills/build/SKILL.md` | Remove lines 46-47 (context check, branch check); replace `sdp guard complete` with `sdp guard deactivate` or remove |
| `sdp/prompts/agents/orchestrator.md` | Replace context check/go with checkpoint-based branch validation |
| `hooks/pre-build.sh` | Remove `sdp guard context check` (redundant with activate) |
| `scripts/hooks/pre-build.sh` | Same |
| `.opencode/skills/build/SKILL.md` | Mirror sdp/prompts changes |
| `.opencode/agents/orchestrator.md` | Mirror sdp/prompts changes |
| `sdp/CLAUDE.md` | Remove `sdp guard finding list/resolve` from Guard Commands table |

## Relation to F023

F023 added `sdp-guard --ws` (separate binary) for **post-commit scope checking** in orchestrate. The phantom commands are for **pre-edit / session context** — different purpose. F023 does not fix the phantoms; they require a separate workstream (e.g. F018 follow-up or new WS).
