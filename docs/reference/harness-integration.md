# Harness Integration Reference

> **Policy source:** F127-05 (`docs/plans/2026-04-16-f127-multi-harness-modernization-design.md`).
> **Related:** `cmd/sdp-dispatch/` · `.agents/skills/README.md` · `AGENTS.md`.

## Overview

SDP is a multi-harness platform. The same agent instructions, skills, and workflows
must work regardless of which coding-agent harness the operator chooses.

`AGENTS.md` is still the policy source, but some harnesses need their own
entrypoint file format. In practice that means:

- `CLAUDE.md` is a thin Claude-specific wrapper over `AGENTS.md`
- `.cursorrules` is a self-contained Cursor entrypoint
- `.kilo/` and `kilo.jsonc` are Kilo Code entrypoints once F162 lands
- `.codex/AGENTS.md` is a Codex-facing entrypoint
- OpenCode uses `AGENTS.md` plus `.opencode/hooks/` for scope enforcement

The skill directory `.agents/skills/` is the canonical location for all SDP skills.
Claude Code, OpenCode, and Cursor scan this path natively. Other harnesses access
skills through symlinks or AGENTS.md references.

```
AGENTS.md  ────────────────────────────────────── universal rules
.agents/skills/  ────────────────────────────────── universal skills
   ┌──────────┬──────────┬──────────┬──────────┐
   │ Claude   │ OpenCode │ Cursor   │ Codex    │
   │ Code     │          │          │ CLI      │
   │          │          │          │          │
CLAUDE.md    AGENTS.md  .cursorrules  .codex/AGENTS.md
(wrapper)    (native)   (native)      (native)
```

## Supported Harnesses

| Harness | Binary | Config file | Skill directories | Status |
|---------|--------|-------------|-------------------|--------|
| Claude Code | `/opt/homebrew/bin/claude` | `CLAUDE.md` (imports `AGENTS.md`) | `.claude/skills/` (symlink to `prompts/skills/`) | Primary, high reliability |
| OpenCode | `/opt/homebrew/bin/opencode` | `AGENTS.md` (native) | `.agents/skills/` (native), `.claude/skills/` (fallback) | Experimental, see [OpenCode section](#opencode-integration) |
| Cursor | `~/.local/bin/agent` (CLI) or IDE | `.cursorrules` (self-contained) | `.agents/skills/` (native), `.cursor/rules/*.mdc` | Experimental, untested in dispatch |
| Codex CLI | `/opt/homebrew/bin/codex` | `.codex/AGENTS.md` + `AGENTS.md` | Via explicit prompt paths and `.codex/` docs | Secondary, high reliability for edits |
| Pi | `/opt/homebrew/bin/pi` (`pi --version`, verified 0.70.6) | `AGENTS.md`/`CLAUDE.md`, `.pi/prompts/*.md` | `.agents/skills/<name>/SKILL.md` or generated `.pi/skills/<name>/SKILL.md` | Experimental; skills/commands smoke green, autonomous dispatch gated by F162 |
| Kilo Code | `kilo` (CLI) or VS Code extension | `AGENTS.md`, `CLAUDE.md`, `kilo.jsonc`, `.kilocodemodes` | `.kilo/agents/*.md`, `.kilo/rules-*` | Planned F162 target |

F162 expands SDP prompt-bundle packaging to Claude Code, Codex, OpenCode,
Cursor, Pi, and Kilo. Pi has a green local resource smoke for skills and command
prompt templates, but dispatch remains experimental until bundle resolution and
runtime launch evidence land.

## OpenCode Integration

### Quick start

```bash
opencode run --dir "$REPO" --agent implementer "prompt text"
```

**Always use `--agent implementer`** for batch, CI, and non-interactive dispatch.
See [below](#opencode-sisyphus-deadlock) for why.

### OpenCode Sisyphus Deadlock

#### Symptom

You run `opencode run "implement F127-01"`. The command returns exit code 0 (success).
You check the files -- nothing changed. No edits were applied.

This was first observed during the SDP UX audit on 2026-04-05.

#### Root cause

OpenCode's default agent is **Sisyphus** -- a lead-orchestrator designed for
interactive TUI sessions. When given a task, Sisyphus delegates work to background
sub-agents (explore, librarian, etc.) and returns a "waiting for results" response.

In non-interactive `opencode run` mode, the session ends immediately after the
orchestrator's first response. The background sub-agents may still be running, but
their results never make it back into the session. The process exits with success,
but no edits were written.

```
opencode run "task"
   |
   +-- Sisyphus receives task
   +-- Sisyphus delegates to sub-agent (background)
   +-- Sisyphus returns "waiting for results"
   +-- opencode run closes session  <-- edits never happen
   +-- exit 0 (misleading success)
```

#### Fix: use --agent implementer

The `--agent implementer` flag bypasses Sisyphus entirely. The implementer agent
works directly in the parent session, editing files before returning control.

```bash
# CORRECT -- for any non-interactive dispatch
opencode run --dir "$REPO" --agent implementer "implement F127-01"
```

The implementer agent:
- Edits files directly (no background delegation)
- Returns only after all edits are applied
- Works reliably in `opencode run` (non-interactive) mode

#### When to use each mode

| Mode | Command | Use when |
|------|---------|----------|
| `--agent implementer` | `opencode run --agent implementer "task"` | Batch dispatch, CI, `sdp-dispatch`, any non-interactive use |
| Default (Sisyphus) | `opencode` (interactive TUI) | Interactive pair programming in the TUI, where the event loop stays open |

#### Troubleshooting

| Problem | Check | Fix |
|---------|-------|-----|
| `opencode run` returns 0 but no edits | Missing `--agent implementer` | Add `--agent implementer` to the command |
| Edits appear incomplete | Prompt too vague | Make the prompt specific: include file paths, function names, acceptance criteria |
| "Agent not found" error | OpenCode version too old | Update OpenCode; verify with `opencode --version` |
| Skills not loaded | `.agents/skills/` missing | Verify `.agents/skills/` exists with real files; run `sdp-publish.sh` if needed |

#### Do not

- **Do not** rely on the default Sisyphus agent in `opencode run` mode.
- **Do not** increase timeouts hoping to "wait it out" -- the parent session is already closed; there is nothing to wait for.
- **Do not** retry the same command without `--agent implementer` -- it will return success without edits every time.

#### Reference

- First observed: SDP UX audit, 2026-04-05
- MEMORY record (local): `feedback_opencode_dispatch.md`
- Long-term goal: build SDP's own coding agents that behave consistently across all harnesses (see `project_unified_coding_agents.md`)

## Claude Code Integration

Claude Code is the primary SDP harness. It does not read `AGENTS.md` natively --
instead, `CLAUDE.md` acts as a thin override that imports `AGENTS.md`.

### How it works

```
CLAUDE.md          <-- thin override (Claude-specific rules only)
  @AGENTS.md       <-- universal rules (imported)
  @RTK.md          <-- token optimization rules (imported)

.claude/skills/    <-- symlink -> prompts/skills/
.claude/hooks/     <-- native hook directory
.claude/agents/    <-- (reserved, not currently used)
```

### Invocation

```bash
# Non-interactive (pipe mode)
claude -p "implement F127-01" --output-format text

# Interactive
claude
```

### Sub-agent dispatch

Claude Code uses the native `Task` tool for sub-agent dispatch. SDP's dispatch
policy in `AGENTS.md` governs when to delegate vs. work inline.

### Troubleshooting

| Problem | Check | Fix |
|---------|-------|-----|
| Skills not found | `.claude/skills/` symlink broken | Verify `.claude/skills` points to `prompts/skills/`; check native path exists |
| AGENTS.md not loaded | `CLAUDE.md` missing `@AGENTS.md` | Ensure `@AGENTS.md` line exists in `CLAUDE.md` |
| Hooks not firing | `.claude/hooks/` symlink broken | Verify `.claude/hooks/` path exists; check symlink target |

## Cursor Integration

Cursor reads `.cursorrules` at the project root and `.cursor/rules/*.mdc` for
rule files. In SDP, `.cursorrules` is a concise self-contained entrypoint that
points back to the canonical prompt and doc sources.

### How it works

SDP ships `.cursorrules` natively at the project root. Cursor also scans
`.agents/skills/` natively for skill discovery.

```bash
# Cursor CLI (agent mode)
agent -p "implement F127-01"
```

### Status (April 2026)

Cursor Agent is **untested** in the SDP dispatch pipeline. Known facts:
- Reads `AGENTS.md` natively
- Has agent mode in IDE and CLI mode through `agent -p`
- Sub-agent dispatch through internal agent runtime (semantics differ from Claude Code Task tool)

**Recommendation:** Use Cursor as a secondary validator for independent edit
verification, not as a primary dispatch worker.

## Pi Integration

Pi here means `@mariozechner/pi-coding-agent`, not Inflection Pi.

Verified local profile:

- Runtime: `/opt/homebrew/bin/pi`, version `0.70.6`.
- Project instructions: Pi loads `AGENTS.md` or `CLAUDE.md` from ancestor dirs.
- Skills: Pi loads Agent Skills from `.agents/skills/<name>/SKILL.md` and generated `.pi/skills/<name>/SKILL.md`. Flat `.agents/skills/*.md` files are not enough for Pi auto-discovery.
- Commands: Pi slash commands are prompt templates in `.pi/prompts/*.md`. Templates must include `$ARGUMENTS`; otherwise `/build 00-123-01` loses `00-123-01` during expansion.
- Tools: default Pi tools are `read`, `bash`, `edit`, and `write`; optional read-only `grep`, `find`, and `ls` can be enabled.
- Write permissions: Pi has no built-in permission popups. Use containering or extension-level policy if a stricter write gate is required.

Smoke test:

```bash
pi --version
scripts/smoke/pi_resources.mjs
```

Dispatch status: resource packaging is green. Autonomous SDP dispatch to Pi stays
experimental until F162 records bundle id/hash evidence and a non-interactive
launch contract.

## Kilo Code Integration

Kilo Code is a planned F162 prompt-bundle target. Its instruction surface is
compatible with SDP's package model because it supports project instruction
files and project-specific agents/modes.

Expected F162 entrypoints:

- `AGENTS.md` and `CLAUDE.md` for project-level instructions
- `kilo.jsonc` for project configuration and instruction sources
- `.kilocodemodes` for project-specific modes
- `.kilo/agents/*.md` for project-specific agents
- `.kilo/rules-{mode}/` for mode-specific instruction files

F162 should generate Kilo adapters from `sdp.manifest.yaml` rather than copying
prompt source files by hand.

## Codex CLI Integration

Codex CLI is reliable for file edits, but many runtimes still need a
Codex-specific entrypoint file for the command surface and fallback references.
SDP ships those in `.codex/`.

### Invocation

```bash
codex exec --full-auto "implement F127-01"
```

### Known limitation: no git in sandbox

Codex runs in a sandbox that prohibits shell operations outside of workspace edits.
The `git` command is not available. If you ask Codex to "commit and push," it
will fail -- but the edits it already made are still in the working tree.

**Workflow:**

```bash
# 1. Dispatch Codex for edits only
codex exec --full-auto "fix the off-by-one error in internal/reconciler/loop.go"

# 2. Commit and push separately (from Claude Code or manually)
git add internal/reconciler/loop.go
git commit -m "fix: off-by-one in reconciler loop"
git push
```

Do not include "commit and push" in prompts sent to Codex.

## Adding a New Harness

To add support for a new coding-agent harness to SDP:

1. **Check AGENTS.md native support.** If the harness reads `AGENTS.md` natively,
   most of the work is already done. Skip to step 3.

2. **Create a config file.** If the harness uses its own config file (like
   `CLAUDE.md` or `.cursorrules`), create it as a thin wrapper:
   ```markdown
   # [Harness name] config
   @AGENTS.md
   ```
   Keep the config minimal. All rules belong in `AGENTS.md`.

3. **Verify skill discovery.** Check which directories the harness scans for
   skills. If it reads `.agents/skills/` natively, skills are already available.
   Otherwise, create a symlink:
   ```bash
   ln -s ../.agents/skills .<harness>/skills
   ```

4. **Test with a simple prompt.** Run the harness in non-interactive mode with
   a trivial task:
   ```bash
   <harness-cli> "create a file called /tmp/harness-test.txt with content 'hello'"
   ```
   Verify the file was created.

5. **Test with SDP dispatch.** Run a real SDP task:
   ```bash
   <harness-cli> "run bd ready and list the first issue"
   ```
   Verify the harness reads `AGENTS.md` and understands SDP commands.

6. **Document known issues.** Add a section to this file with:
   - Invocation command
   - Config file used
   - Known limitations or deadlocks
   - Troubleshooting table

## Capability-based Routing

`cmd/sdp-dispatch` maintains `CapabilityProfile` files in `.sdp/dispatch/profiles/*.json`:

- `TestPassRate` per `task-type:language` per harness+model.
- ColdStartStrategy: `capability-heuristic` (default), `round-robin`, `fallback-chain`.
- Profile is considered stale after 30 days (warning on dispatch).

Full design: [OpenCode harness gates telemetry spec](../OPENCODE_HARNESS_GATES_TELEMETRY_SPEC.md).

## Common Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `opencode run` returns 0 but no edits | Sisyphus deadlock | Add `--agent implementer` |
| Codex returns "git: command not found" | Sandbox restriction | Do not ask Codex to commit; commit externally |
| Claude Code does not see a skill | Symlink broken | Verify `.claude/skills/` points to `prompts/skills/`; check files exist |
| Cursor Agent hangs | Untested surface (April 2026) | Fallback to Claude Code |
| Skills not found by any harness | `.agents/skills/` missing | Check that `.agents/skills/` exists with Pi-compatible `<name>/SKILL.md` directories or harness-native equivalents |
| Pi command drops its argument | Prompt template lacks `$ARGUMENTS` | Regenerate `.pi/prompts/*.md`; run `scripts/smoke/pi_resources.mjs` |
| Dispatch routes target Pi before F162 launch evidence | Runtime dispatch profile incomplete | Disable autonomous Pi routing until bundle id/hash evidence and launch contract are recorded |

## References

- [F127 design doc](../plans/2026-04-16-f127-multi-harness-modernization-design.md)
- [AGENTS.md](../../AGENTS.md) -- universal agent rules
- [Skill authoring guide](skill-authoring.md) -- SKILL.md frontmatter and body template
- [Multi-agent patterns](multi-agent-patterns.md) -- when to use each orchestration pattern
- [Skill directory README](../../.agents/skills/README.md) -- why `.agents/skills/` is canonical
- MEMORY records (local, not in repo): `feedback_opencode_dispatch.md`, `reference_harness_clis.md`, `project_unified_coding_agents.md`
