---
name: research
description: Codebase investigation — search first, read actual code, produce structured answer with file:line refs.
version: 1.0.0
tags: [research, exploration, documentation]
requires_cli: []
compatibility: [claude-code, opencode, cursor, codex]
---

# Research

## Purpose

Given a question about the codebase (architecture, behavior, coverage gaps, data flow), produce a
structured answer grounded in the actual source — not docs, not assumptions.

**Outcome:** precise answer with file:line references, relationship map, and open questions.

## Use When

- "How does X work?" — architecture, data flow, lifecycle
- "What calls Y?" — dependency tracing
- "Which parts aren't tested?" — coverage gaps
- "Where is Z configured?" — config / feature flags

**Do not use when:** the task requires writing or modifying code (use `bug-fix` or `feature-delivery`).

## MUST DO

1. **Search before reading** — grep for symbols, types, function names. Don't guess file paths.
2. **Search in named package first** — if the question names a package (e.g., "agentloop FSM"), search `internal/<pkg>/` before adjacent packages. Document all implementations found, not just the first.
3. **Read actual code** — not just docs or comments; docs can be stale.
4. **Trace relationships** — follow call chains: A → B → C. Don't stop at the first layer.
5. **If multiple implementations exist** — document ALL of them; note which is active/legacy.
6. **Cite every claim** — `file.go:42` format for every behavior statement.
7. **Note uncertainty** — if a path is unclear, say so explicitly.

## MUST NOT DO

- Write or modify any file.
- Answer "it probably works like…" without checking.
- Stop at the first file found — check call sites and callers.
- Cite only docs — verify against source.

## Response Format

```
## Summary
[2-3 sentences: direct answer to the question]

## How It Works
[Numbered steps, each with file:line reference]
  1. Entrypoint: cmd/sdp-foo/main.go:34 calls internal/foo.Run()
  2. ...

## Key Types / Interfaces
| Name | File | Role |
|------|------|------|

## Relationships
[Component → component arrows, or ASCII diagram if helpful]

## Coverage / Gaps
[Only if question is about testing or coverage]

## Open Questions
[What's unclear, where to dig further if needed]
```

## References

- `docs/reference/project-map.md` — canonical SOT split, where to look for what
- `docs/ARCHITECTURE.md` — C4 context and component map
