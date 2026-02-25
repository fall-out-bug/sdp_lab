# AGENTS.md / CLAUDE.md Sync Rules

**Purpose:** Keep agent instructions consistent between `sdp_dev` (AGENTS.md) and `sdp` submodule (CLAUDE.md).

## Shared Sections

When updating these, update **both** files:

| Section | AGENTS.md | sdp/CLAUDE.md |
|---------|-----------|---------------|
| Artifact placement | What Goes Where → Artifact Placement | Shared Conventions |
| "Продолжай" convention | sdp-orchestrate section | Shared Conventions |
| Command decision tree | Command Decision Tree | (optional in sdp) |

## Sync Process (Manual)

1. Edit `AGENTS.md` in sdp_dev root (source of truth for sdp_dev workflows)
2. Copy shared content to `sdp/CLAUDE.md` under "Shared Conventions"
3. Add Sync note to both files
4. Commit: sdp_dev changes + sdp submodule (if CLAUDE.md changed)

## What to Sync

- **Placement rules:** docs/reviews/, docs/workstreams/backlog/, docs/drafts/idea-*
- **"Продолжай" convention:** "продолжай {feature}" = sdp-orchestrate --feature {feature} --next-action
- **Status command:** sdp-orchestrate --feature FXXX --status

## What NOT to Sync

- sdp_dev-specific: repo boundary, beads mapping, feature delivery flow
- sdp-specific: CLI reference, protocol flow, skill list
