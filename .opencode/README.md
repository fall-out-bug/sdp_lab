# OpenCode Integration

This directory contains SDP integration for OpenCode.

## Prompt Surface

- Skills: `.agents/skills/` (native), with `prompts/skills/` as canonical structured source
- Commands: `prompts/commands/`
- Agents: `prompts/agents/`
- Canonical command map: `prompts/commands.yml`

## Hook Surface

OpenCode scope enforcement lives in:

- `.opencode/hooks/pre-tool-use.json`
- `.opencode/hooks/README.md`

The current hook implementation uses `sdp-omc-guard`, which is a stronger
wrapper around SDP guard semantics for edit and write operations.

## Usage

Use `@` commands with Opencode dispatch. These are harness commands, not
`./.sdp/bin/sdp` CLI calls:

```text
@vision "product"
@feature "add feature"
@build 00-XXX-YY
@review FXXX
@operate "deploy task"
```

`00-XXX-YY` is a workstream ID placeholder; it must come from queue-backed
operator mode.

For local delivery without a queue, use:

```bash
./.sdp/bin/sdp build "what you want to change" --dry-run --format text
```

If you use `opencode run` for non-interactive dispatch, always pass
`--agent implementer`:

```bash
opencode run --dir "$PWD" --agent implementer "@build 00-XXX-YY"
```

**OpenCode command warning:** do not run non-interactive `opencode run` without
`--agent implementer`.

Without it, non-interactive `opencode run` can exit with code 0 before edits are
applied (interactive Sisyphus handoff deadlock).

## Fallback Mode

If your OpenCode runtime cannot spawn subagents, follow the manual checklists in
[`docs/reference/FALLBACK_MODE.md`](../docs/reference/FALLBACK_MODE.md).
