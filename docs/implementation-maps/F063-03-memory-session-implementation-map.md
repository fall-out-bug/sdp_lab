# Implementation Map: sdplab-4e6 (F063-03 Configure memory for SDP sessions)

**Beads ID:** sdplab-4e6  
**Workstream:** 00-063-03  
**Feature:** F063 (opencode-mem Memory Module)  
**Depends on:** 00-063-02 (Install opencode-mem plugin) — DONE

---

## 1. Intent Analysis

| | |
|---|---|
| **Literal Request** | Identify all code paths, configs, tests, and docs needed to implement Beads issue sdplab-4e6 (F063-03 Configure memory for SDP sessions) |
| **Actual Need** | A concrete file-level implementation map with current behavior, missing pieces, and exact edit recommendations with test targets |
| **Success Looks Like** | Developer can implement F063-03 without follow-up questions; every edit location, test target, and config change is specified |

---

## 2. Current Behavior (Cited Paths)

### 2.1 opencode-mem Installation (00-063-02 — DONE)

- **Config:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/.opencode/mem-config.json`
  - `storagePath`: `.sdp/mem/`
  - `embeddingModel`: Xenova/nomic-embed-text-v1
  - `autoCaptureEnabled`: true
  - `chatMessage.injectOn`: `first_or_post_compaction`
  - `maxMemories`: 20
- **Storage:** `.sdp/mem/` (gitignored in `.gitignore` lines 7–8, 13–14)
- **Plugin:** Runs inside OpenCode; injects memory into chat messages automatically

### 2.2 Session Lifecycle (Current)

| Component | Path | Behavior |
|-----------|------|----------|
| Session evidence writer | `internal/session/writer.go` | `NewWriter(projectRoot, sessionID)` creates `.sdp/log/session-{id}.jsonl`; `Append*`, `Finalize` |
| Session paths | `internal/session/paths.go` | `LogDir()`, `SessionLog()`, `ValidateSessionID()` |
| Session events | `internal/session/event.go` | `EventTypeSessionStart`, `EventTypeSessionEnd`, `EventTypeToolCall`, `EventTypeGuardCheck` |
| Guard (evidence emitter) | `cmd/sdp-omc-guard/main.go` | Receives `session_id` from stdin or `--session-id`; emits `guard_check` via `session.NewWriter` |
| **Gap:** `session_start` | — | **Not emitted by SDP.** OhMyOpenCode provides session_id; test fixtures use `session_start` in JSONL but no SDP code emits it |
| Stuck detector | `internal/monitor/stuck_detector.go` | Watches `.sdp/log/session-*.jsonl` mod times |
| Session store (wisps) | `internal/workstream/session_store.go` | Ephemeral items in `.sdp/session/wisps/` |

### 2.3 Context Injection (Pre-hydration)

| Component | Path | Behavior |
|-----------|------|----------|
| Hydrate | `internal/orchestrate/hydrate.go` | `Hydrate()` gathers WS, AC, scope, checkpoint, deps, quality gates, drift → `.sdp/context-packet.json` |
| Context packet | `internal/orchestrate/hydrate.go:18–28` | `ContextPacket` struct: Workstream, AcceptanceCriteria, ScopeFiles, Checkpoint, Dependencies, QualityGates, DriftStatus |
| Load + inject | `internal/orchestrate/invoke_opencode.go:16–21` | `buildPromptWithContext()` = basePrompt + `LoadContextPacket().FormatForPrompt()` |
| FormatForPrompt | `internal/orchestrate/hydrate.go:156–173` | Renders Workstream, AC, ScopeFiles, QualityGates, DriftStatus |
| **Gap:** Memory context | — | **Not in ContextPacket.** No memory/profiles injected into prompt |

### 2.4 CLI Entrypoints

| CLI | Path | Session/Memory Relevance |
|-----|------|--------------------------|
| sdp-omc-guard | `cmd/sdp-omc-guard/main.go` | `--session-id`, emits evidence; **no memory** |
| sdp-orchestrate | `cmd/sdp-orchestrate/main.go` | `--hydrate`; runs `InvokeOpenCode`; **no session_id**, no memory |
| sdp-evidence | `cmd/sdp-evidence/main.go` | validate/inspect; **no memory** |
| sdp-guard | `cmd/sdp-guard/main.go` | Constraint checks; **no session** |

### 2.5 Config Loading Patterns

| Pattern | Path | Example |
|---------|------|---------|
| YAML config | `internal/orchestrate/hooks.go:52–65` | `LoadHookConfig(projectRoot)` → `.sdp/pipeline-hooks.yaml` |
| JSON config | `internal/orchestrate/hydrate.go:139–154` | `LoadContextPacket(projectRoot)` → `.sdp/context-packet.json` |
| Env vars | `internal/orchestrate/migration_shim.go:525` | `SDP_ENVIRONMENT` |
| Env vars (runbooks) | `docs/runbooks/orchestrator_v2_cutover.md:234–239` | `SDP_ORCHESTRATOR_BACKEND`, `SDP_FALLBACK_ENABLED`, etc. |
| **Gap:** Memory config | — | `.opencode/mem-config.json` exists but **no Go code reads it** |

### 2.6 Feature Flags / Env Vars

- `SDP_WORKSTREAM`, `SDP_SESSION_ID`: `.opencode/hooks/README.md:56–57`
- `SDP_GO_QUALITY_MODE`, `SDP_POLICY_ENFORCEMENT_MODE`: `scripts/run_go_quality_gates.sh`, `.github/workflows/ci.yml`
- **Recommendation:** Add `SDP_MEMORY_ENABLED` (default true) for opt-out; `SDP_MEMORY_PATH` override (default `.sdp/mem`)

---

## 3. Scope Files (from 00-063-03)

| Scope | Status | Notes |
|-------|--------|-------|
| `sdp/prompts/agents/` | **Empty** | sdp submodule is empty (`ls sdp/` → only `.` `..`) |
| `sdp/prompts/skills/` | **Empty** | Same |
| `internal/session/memory.go` | **Missing** | New file required |

---

## 4. Acceptance Criteria → Implementation Mapping

| AC | Current | Missing | Edit Recommendations |
|----|---------|---------|----------------------|
| Memory context injected into SDP agent prompts | No | Memory in ContextPacket | Extend `ContextPacket`, `Hydrate`, `FormatForPrompt` |
| User profile preferences available in planning phase | No | Profile loading | `internal/session/memory.go` LoadProfiles; add to ContextPacket |
| Project memories scoped by git repo URL | opencode-mem does this | SDP must pass repo URL | Ensure mem-config `tags.project` or similar; `memory.go` uses git remote |
| Memory tool available for agents to store learnings | opencode-mem plugin | — | Plugin handles; document in skills |
| Post-compaction recovery tested | — | Test | Integration test: simulate compaction, verify memory restored |
| Session continuity verified: new session has context from previous | — | E2E test | Test: session A stores, session B loads |

---

## 5. Exact Edit Recommendations

### 5.1 New File: `internal/session/memory.go`

**Purpose:** Go client to read memory context for SDP prompts. opencode-mem stores in `.sdp/mem/` (SQLite + HNSW). Options:

- **Option A (preferred):** Read opencode-mem export file. If opencode-mem can write a JSON summary to `.sdp/mem/context.json` on idle, SDP reads that.
- **Option B:** Direct SQLite read (schema-dependent; fragile).
- **Option C:** HTTP API if opencode-mem exposes one (check upstream).

**Recommended API (Option A):**

```go
// LoadMemoryContext reads project memory and user profile for prompt injection.
// Returns empty string if disabled or no data. Path: projectRoot/.sdp/mem/
func LoadMemoryContext(projectRoot string) (memoryContext string, err error)
```

**Edit:** Create `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/session/memory.go`

- Read `.sdp/mem/context.json` if exists (opencode-mem would need to write this; or SDP writes after orchestrate phases).
- Fallback: read `.opencode/mem-config.json` for `storagePath`; if dir exists, list recent files or use a convention.
- **Blocker:** opencode-mem may not expose a Go-readable format. Verify upstream API: https://github.com/tickernelz/opencode-mem/blob/main/src/index.ts

### 5.2 Extend `internal/orchestrate/hydrate.go`

| Location | Edit |
|----------|------|
| `ContextPacket` struct (line 20) | Add `MemoryContext string \`json:"memory_context,omitempty"\`` |
| `Hydrate()` (line 33) | After `pkt.DriftStatus`, call `session.LoadMemoryContext(projectRoot)`; set `pkt.MemoryContext` |
| `FormatForPrompt()` (line 156) | If `p.MemoryContext != ""`, append `### Memory Context\n\n` + `p.MemoryContext` |

**Files:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/orchestrate/hydrate.go`

### 5.3 Extend `internal/session/paths.go`

| Location | Edit |
|----------|------|
| After `CacheDir()` | Add `MemDir() string` → `filepath.Join(p.ProjectRoot, ".sdp", "mem")` |

**Files:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/session/paths.go`

### 5.4 Prompts / Skills (sdp submodule or sdp_lab)

**Current:** sdp submodule empty. Per AGENTS.md, prompts can live in sdp_lab and publish to sdp when needed.

| Location | Edit |
|----------|------|
| Create `docs/prompts/memory-context.md` or extend AGENTS.md | Add section: "Memory: Project memories and user preferences are injected at session start. Use the memory tool to store learnings." |
| If `.opencode/skills/` or `.cursor/skills/` exist | Add memory operations to skill instructions |

**Files:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/AGENTS.md` or new `docs/prompts/memory-context.md`

### 5.5 Config: `SDP_MEMORY_ENABLED`

| Location | Edit |
|----------|------|
| `internal/session/memory.go` | If `os.Getenv("SDP_MEMORY_ENABLED") == "false"`, return empty immediately |
| `internal/orchestrate/hydrate.go` | When calling `LoadMemoryContext`, no change (memory.go handles env) |

### 5.6 Session Start Emission (Optional for 00-063-03)

**Current:** No SDP code emits `session_start`. OhMyOpenCode provides session_id to sdp-omc-guard.

**Recommendation:** Defer to 00-063-04 (evidence flow). For 00-063-03, session continuity is verified by: new session (new `opencode run` or new Cursor session) sees memory from `.sdp/mem/` which is project-scoped.

---

## 6. Test Targets

| Test File | Test Name | Purpose |
|-----------|-----------|---------|
| `internal/session/memory_test.go` | `TestLoadMemoryContext_Disabled` | Env `SDP_MEMORY_ENABLED=false` → empty |
| `internal/session/memory_test.go` | `TestLoadMemoryContext_NoData` | No `.sdp/mem/` → empty, no error |
| `internal/session/memory_test.go` | `TestLoadMemoryContext_WithData` | Mock `.sdp/mem/context.json` → returns content |
| `internal/orchestrate/hydrate_test.go` | Extend `TestHydrate` | Assert `pkt.MemoryContext` populated when memory exists |
| `internal/orchestrate/hydrate_test.go` | `TestFormatForPrompt_IncludesMemory` | `FormatForPrompt` contains memory section when set |
| `internal/session/paths_test.go` | `TestMemDir` | `Paths.MemDir()` returns `.sdp/mem` |

**New file:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/session/memory_test.go`

---

## 7. Documentation Updates

| File | Edit |
|------|------|
| `docs/workstreams/backlog/00-063-03.md` | Mark AC complete as implemented |
| `docs/integrations/ECOSYSTEM_SYNERGIES.md` | Update F063 table: 00-063-03 status |
| `AGENTS.md` | Add Memory section if not present |
| `.opencode/hooks/README.md` | Add `SDP_MEMORY_ENABLED` to env vars |

---

## 8. Dependency: opencode-mem API

**Blocker:** `internal/session/memory.go` needs a concrete way to read memory. opencode-mem stores in SQLite at `.sdp/mem/`. Options:

1. **Upstream contract:** Request opencode-mem to write `.sdp/mem/context.json` (summary for external consumers).
2. **SQLite read:** Parse schema from https://github.com/tickernelz/opencode-mem (schema may change).
3. **CLI:** If opencode-mem has `opencode-mem export` or similar, exec it and parse stdout.

**Action:** Before implementing `memory.go`, run:

```bash
# Inspect .sdp/mem/ after an OpenCode session with memory
ls -la .sdp/mem/
```

If empty (gitignored), create a test session that uses memory, then inspect.

---

## 9. File-Level Summary

| Action | Path |
|--------|------|
| **Create** | `internal/session/memory.go` |
| **Create** | `internal/session/memory_test.go` |
| **Create** | `internal/session/paths_test.go` (no existing paths test; add `TestMemDir`) |
| **Edit** | `internal/orchestrate/hydrate.go` (ContextPacket, Hydrate, FormatForPrompt) |
| **Edit** | `internal/session/paths.go` (MemDir) |
| **Edit** | `internal/orchestrate/hydrate_test.go` (extend tests) |
| **Edit** | `AGENTS.md` or `docs/prompts/memory-context.md` |
| **Edit** | `.opencode/hooks/README.md` (SDP_MEMORY_ENABLED) |
| **Edit** | `docs/workstreams/backlog/00-063-03.md` (AC checkboxes) |
| **Edit** | `docs/integrations/ECOSYSTEM_SYNERGIES.md` (status) |

---

## 10. Next Steps

1. **Verify opencode-mem storage format:** Run an OpenCode session, inspect `.sdp/mem/` (or read upstream source).
2. **Implement `memory.go`** with the chosen read strategy.
3. **Wire Hydrate** and run `go test ./internal/orchestrate/... ./internal/session/...`.
4. **E2E:** Start session A, add memory, end; start session B, verify context in prompt.
5. **Close sdplab-4e6** with `bd close sdplab-4e6 --reason "F063-03: memory context injection implemented"`.
