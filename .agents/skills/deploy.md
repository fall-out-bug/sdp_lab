---
name: deploy
description: "DEPRECATED: Legacy deploy skill. Redirects to @operate intent with deploy mode for production deployments."
deprecated: true
redirect: operate
version: 0.0.0
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# @deploy → @operate

**Deprecated.** Use `@operate --mode deploy` instead.

See: `.agents/skills/operate.md`

Legacy safety mirror: before any deploy-like action, require compact APPROVED
review evidence, no open P0/P1 for the feature, green deterministic gates, and
explicit handling of PI review provider degradation. Do not commit raw
`.sdp/runs/pi-review/*` telemetry by default.
