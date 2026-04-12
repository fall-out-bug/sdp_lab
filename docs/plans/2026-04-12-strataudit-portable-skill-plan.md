# StratAudit Portable Skill Plan

**Date:** 2026-04-12  
**Status:** Proposed  
**Feature:** F111  
**Spec:** [2026-04-12-strataudit-portable-skill-design.md](2026-04-12-strataudit-portable-skill-design.md)

## WS-01: Provider-Neutral Engine and Runtime Resolution

Changes:

- add `ModelRuntime` interface to `internal/strataudit`;
- switch engine entrypoints from `*LLMClient` to `ModelRuntime`;
- add config-driven runtime section;
- resolve OpenRouter runtime from config instead of hardcoded env/baseURL;
- add tests for runtime resolution and CLI behavior.

Acceptance:

- engine can run with injected runtime implementation;
- CLI defaults to OpenRouter but no longer hardcodes `OPENROUTER_API_KEY`;
- host-only provider path fails with explicit message instead of implicit OpenRouter assumption.

## WS-02: Local Skill Surface and Discovery Docs

Changes:

- add local `skills/strataudit.md`;
- update `docs/reference/skills.md`;
- update `docs/reference/components.md`;
- document runtime selection order and fallback behavior.

Acceptance:

- repo-local harnesses have a clear StratAudit skill surface;
- docs say host-native first, OpenRouter as enhancer;
- components catalog no longer claims `internal/strataudit -> OpenRouter` as the only path.

## WS-03: Public `sdp` Skill Publication and Boundary Repair

Changes:

- restore real `sdp` submodule on branch if needed;
- publish public StratAudit skill/prompt docs into `sdp/`;
- sync wording with local skill surface.

Acceptance:

- public prompt surface can describe StratAudit as portable discovery capability;
- repo boundary is honest again on this branch;
- public docs do not frame StratAudit as one private CLI with one provider.
