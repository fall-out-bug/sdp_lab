---
name: oneshot
description: "DEPRECATED: Legacy oneshot skill. Redirects to @build intent with prototype mode for single-session prototypes."
deprecated: true
redirect: build
version: 0.0.0
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# @oneshot → @build

**Deprecated.** Use `@build --mode prototype` instead.

Note: Checkpoint/resume behavior is now available through `@operate --mode plan` for session management.

See: `.agents/skills/build.md` | `docs/reference/migration-guide.md`
