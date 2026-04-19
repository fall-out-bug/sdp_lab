# Reference Documentation

Use this directory for stable reference docs, not for historical planning archaeology.

Start here:

1. [project-map.md](project-map.md) — what `sdp_lab` is, what the current direction is, and which docs are canonical
2. [canonical-happy-path.md](canonical-happy-path.md) — one stable description of Local Mode, Operator Mode, and board-to-delivery flow
3. [agent-catalog.md](agent-catalog.md) — default agent ownership across the canonical loop
4. [skills.md](skills.md) — public and internal skill surface
5. [commands.md](commands.md) — command reference

## Quick Find

| Need | Reference |
|---|---|
| Project identity and read order | [project-map.md](project-map.md) |
| Canonical happy path and mode split | [canonical-happy-path.md](canonical-happy-path.md) |
| Adopt SDP in another repo | [../../sdp/docs/QUICKSTART.md](../../sdp/docs/QUICKSTART.md) (local path, also published to sdp repo) |
| Canonical loop and default agents | [canonical-happy-path.md](canonical-happy-path.md), [agent-catalog.md](agent-catalog.md) |
| Canonical skill surface | [skills.md](skills.md) |
| Commands | [commands.md](commands.md) |
| Quality gates | [quality-gates.md](quality-gates.md) |
| Configuration | [configuration.md](configuration.md) |
| Runbooks index | [runbooks.md](runbooks.md) |
| Glossary | [GLOSSARY.md](GLOSSARY.md) |
| Principles | [PRINCIPLES.md](PRINCIPLES.md) |

## Rules

- prefer reference docs over dated plans when both exist
- prefer `project-map.md` for orientation, then follow links into deeper docs
- prefer `canonical-happy-path.md` for the stable system story and use plans only for rationale or rollout detail
- treat stale mentions of `sdp_dev`, `dev`, `master`, or `bd sync` as historical unless a live doc still says they are current

## Scope

This directory should answer stable questions such as:

- what the canonical loop is
- which agent or skill owns which stage
- which command or config surface is current
- which terms have precise meaning
