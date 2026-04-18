# Deprecated Skill Aliases

**Purpose:** Compatibility layer for legacy skill names during F125 migration.
**Status:** Active deprecation (2026-04-17 → 2026-06-01)
**See:** `docs/reference/migration-guide.md` for full migration guide

## Mapping Format

Each legacy skill maps to:
- **New intent:** The intent-based skill that replaces it
- **Mode/dimension:** Specific mode or dimension to use
- **Deprecation warning:** Message shown when legacy name is used

## Understand Aliases

| Legacy Skill | Routes To | Intent Mode | Warning Message |
|--------------|-----------|-------------|-----------------|
| `@scout` | `@understand` | quick | `@scout` is deprecated. Use `@understand` (auto-detects quick mode) |
| `@architect` | `@understand` | standard | `@architect` is deprecated. Use `@understand --depth standard` |
| `@metrics` | `@understand` | standard | `@metrics` is deprecated. Use `@understand --depth standard` |
| `@spec` | `@understand` | deep | `@spec` is deprecated. Use `@understand --depth deep` |
| `@landscape` | `@understand` | standard/deep | `@landscape` is deprecated. Use `@understand` (auto-detects depth) |
| `@index query` | `@understand` | deep | `@index query` is deprecated. Use `@understand --depth deep` |

## Build Aliases

| Legacy Skill | Routes To | Intent Mode | Warning Message |
|--------------|-----------|-------------|-----------------|
| `@feature` | `@build` | feature | `@feature` is deprecated. Use `@build` (auto-detects feature mode) |
| `@idea` | `@build` | idea | `@idea` is deprecated. Use `@build --mode idea` |
| `@design` | `@build` | idea | `@design` is deprecated. Use `@build --mode idea` |
| `@ux` | `@build` | idea | `@ux` is deprecated. Use `@build --mode idea` |
| `@vision` | `@build` | idea | `@vision` is deprecated. Use `@build --mode idea` |
| `@oneshot` | `@build` | prototype | `@oneshot` is deprecated. Use `@build --mode prototype`. Note: Checkpoint/resume behavior is now available through `@operate --mode plan` for session management |
| `@prototype` | `@build` | prototype | `@prototype` is deprecated. Use `@build --mode prototype` |

## Fix Aliases

| Legacy Skill | Routes To | Intent Mode | Warning Message |
|--------------|-----------|-------------|-----------------|
| `@hotfix` | `@fix` | quick | `@hotfix` is deprecated. Use `@fix` (auto-detects quick mode) |
| `@bugfix` | `@fix` | systematic | `@bugfix` is deprecated. Use `@fix --mode systematic` |
| `@issue` | `@fix` | systematic | `@issue` is deprecated. Use `@fix --mode systematic` |
| `@debug` | `@fix` | investigate | `@debug` is deprecated. Use `@fix --mode investigate` |

## Review Aliases

| Legacy Skill | Routes To | Intent Dimension | Warning Message |
|--------------|-----------|------------------|-----------------|
| `@reality-check` | `@review` | reality | `@reality-check` is deprecated. Use `@review --dimension reality` |
| `@verify-workstream` | `@review` | readiness | `@verify-workstream` is deprecated. Use `@review --dimension readiness` |

**Note:** `@review` is **not deprecated** — it's the primary intent. Only dimension-specific aliases are deprecated.

## Operate Aliases

| Legacy Skill | Routes To | Intent Mode | Warning Message |
|--------------|-----------|-------------|-----------------|
| `@deploy` | `@operate` | deploy | `@deploy` is deprecated. Use `@operate --mode deploy` |
| `@ci-triage` | `@operate` | triage | `@ci-triage` is deprecated. Use `@operate --mode triage` |
| `@plan` | `@operate` | plan | `@plan` is deprecated. Use `@operate --mode plan` |

## Practices (Not Skills, No Direct Replacement)

These were never true skills — they're practices or tools:

| Practice | Status | Replacement |
|----------|--------|-------------|
| `@tdd` | **Embedded practice** | No replacement needed — test-first is default in @build and @fix |
| `@guard` | **Automatic hook** | No replacement needed — runs automatically pre-commit |
| `@go-modern` | **Style convention** | No replacement needed — applied automatically |
| `@think` | **Prompt technique** | No replacement needed — used throughout intents |
| `@beads` | **CLI tool** | Use `bd` commands directly (not an AI skill) |

## Implementation Notes

### For Harness Authors

When implementing deprecation warnings:

1. **Detect legacy skill name** from this mapping
2. **Route to new intent** with specified mode/dimension
3. **Emit warning message** before executing
4. **Link to migration guide** for details

Example warning format:

```
⚠️ DEPRECATION WARNING: @architect is deprecated
→ Use: @understand --depth standard
→ See: docs/reference/migration-guide.md

[Executing with legacy compatibility...]
```

### For Tool Authors

When building tools that dispatch skills:

1. **Check this mapping first** for legacy names
2. **Transform to new intent + mode** before dispatch
3. **Log deprecation** for analytics
4. **Prefer new intent names** in documentation

## Timeline

- **2026-04-17:** Soft launch — aliases work with warnings
- **2026-06-01:** Hard cutover — aliases removed, intents only

## References

- Migration guide: `docs/reference/migration-guide.md`
- Intent design: `docs/plans/2026-04-13-sdp-skill-architecture-design.md`
- Skills reference: `docs/reference/skills.md`
