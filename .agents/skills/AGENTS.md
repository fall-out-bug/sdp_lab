# .agents/skills — Agent Contract

## Scope

This subtree owns runtime skill discovery files for harnesses that read
`.agents/skills` directly or through generated adapters.

## Contract

Flat `.md` files and `<name>/SKILL.md` directories are adapter/runtime surfaces.
They must route to the same canonical skill behavior described in
`docs/reference/skills.md` and `prompts/skills`.

## Dependencies

Use `sdp.manifest.yaml`, `docs/reference/skills.md`, and
`docs/reference/skill-authoring.md` as the normalization sources. Do not invent a
third skill taxonomy here.

## Runtime Assumptions

Different harnesses read different shapes:

- OpenCode, Cursor, and Kimi can read flat `.md` files.
- Pi-compatible loaders need `<name>/SKILL.md` directories.
- Codex-style runtimes may rely on explicit paths from root instructions.

## Local Rules

- Deprecated aliases must redirect to the canonical intent or replacement skill.
- Intent skills must not diverge from published structured skills.
- If a runtime file is only an adapter, keep it thin and point to the canonical
  owner.
- Runtime skills must preserve completion discipline from canonical skills:
  scoped staging, commit, push or exact blocker. Never add `git add .` to skill
  instructions.
- Keep `README.md` aligned with the actual filesystem and manifest state.
