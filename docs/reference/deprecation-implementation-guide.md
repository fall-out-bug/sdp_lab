# Deprecation Warning Implementation Guide

**Purpose:** Guide for harness authors implementing F125 deprecation warnings.
**Feature:** F125 (Toolkit UX — intent-routed skills over composable tools)
**Status:** Implementation reference

## Overview

When users invoke legacy skill names during the deprecation period (2026-04-17 → 2026-06-01), the harness should:

1. **Detect** the legacy skill name
2. **Route** to the new intent with appropriate mode/dimension
3. **Emit** a clear, actionable deprecation warning
4. **Execute** the intent using legacy compatibility

## Warning Format

Deprecation warnings should be:
- **Visible:** Use warning symbols (⚠️) and clear formatting
- **Actionable:** Show exactly what to use instead
- **Helpful:** Link to migration guide for details
- **Non-blocking:** Execute anyway during deprecation period

### Standard Warning Template

```
⚠️ DEPRECATION WARNING
@{legacy_skill} is deprecated and will be removed on 2026-06-01.

→ Use: @{new_intent} {mode_flag}
→ Example: {example_command}
→ See: docs/reference/migration-guide.md

[Executing with legacy compatibility...]
```

### Example Warnings

```
⚠️ DEPRECATION WARNING
@scout is deprecated and will be removed on 2026-06-01.

→ Use: @understand
→ Example: @understand this repo
→ See: docs/reference/migration-guide.md

[Executing @understand in quick mode...]
```

```
⚠️ DEPRECATION WARNING
@feature is deprecated and will be removed on 2026-06-01.

→ Use: @build
→ Example: @build add user authentication
→ See: docs/reference/migration-guide.md

[Executing @build in feature mode...]
```

```
⚠️ DEPRECATION WARNING
@design is deprecated and will be removed on 2026-06-01.

→ Use: @build --mode idea
→ Example: @build --mode design payment flow
→ See: docs/reference/migration-guide.md

[Executing @build in idea mode...]
```

## Implementation Algorithm

### Pseudocode

```python
def invoke_skill(skill_name, args):
    # Check if skill_name is a legacy alias
    if skill_name in DEPRECATED_ALIASES:
        mapping = DEPRECATED_ALIASES[skill_name]

        # Emit deprecation warning
        emit_warning(
            legacy_skill=skill_name,
            new_intent=mapping.intent,
            mode=mapping.mode,
            example=generate_example(mapping),
            migration_guide="docs/reference/migration-guide.md"
        )

        # Route to new intent
        skill_name = mapping.intent
        args = apply_mode(args, mapping.mode)

    # Execute the skill
    return execute_intent(skill_name, args)
```

### Data Structure

Use `.agents/skills/deprecated-aliases.md` as the source of truth:

```json
{
  "@scout": {
    "intent": "@understand",
    "mode": "quick",
    "warning": "@scout is deprecated. Use @understand (auto-detects quick mode)"
  },
  "@architect": {
    "intent": "@understand",
    "mode": "standard",
    "warning": "@architect is deprecated. Use @understand --mode standard"
  },
  "@feature": {
    "intent": "@build",
    "mode": "feature",
    "warning": "@feature is deprecated. Use @build (auto-detects feature mode)"
  }
  // ... see deprecated-aliases.md for full mapping
}
```

## Compatibility Behavior

### During Deprecation Period (2026-04-17 → 2026-06-01)

1. **Legacy names work** — no breaking changes
2. **Warnings shown** — educate users about new pattern
3. **Auto-routing** — translate to new intent + mode
4. **Execution continues** — don't block work

### After Hard Cutover (2026-06-01+)

1. **Legacy names fail** — clear error message
2. **No warnings** — only intent names accepted
3. **Direct execution** — no translation layer

### Error Message After Cutover

```
❌ UNKNOWN SKILL: @{legacy_skill}

This skill was deprecated and removed on 2026-06-01.

→ Use: @{new_intent} {mode_flag}
→ See: docs/reference/migration-guide.md
```

## Testing Checklist

Test deprecation warnings with:

- [ ] All 26 legacy skill names
- [ ] Each intent (understand, build, fix, review, operate)
- [ ] Each mode/dimension combination
- [ ] Warning message formatting
- [ ] Link to migration guide works
- [ ] Legacy compatibility still executes correctly
- [ ] Auto-routing preserves original behavior
- [ ] Warnings don't break automation/scripts

## Analytics (Optional)

Track deprecation metrics:

```json
{
  "event": "legacy_skill_invocation",
  "skill": "@scout",
  "routed_to": "@understand",
  "mode": "quick",
  "timestamp": "2026-04-17T10:30:00Z"
}
```

Use metrics to:
- Monitor adoption of new intent names
- Identify most-used legacy skills for targeted migration
- Verify cutover readiness

## Harness-Specific Notes

### Claude Code

- Hook into skill invocation before Skill tool dispatch
- Use markdown formatting for warnings
- Link to migration guide with absolute path

### Codex CLI

- Integrate with CLI argument parser
- Use terminal colors for visibility (yellow for warnings)
- Support both `/@skill` and `@skill` invocation patterns

### OpenCode/Cursor

- Add deprecation detection to skill routing layer
- Show warnings in agent output channel
- Preserve compatibility with existing workflows

## Rollout Strategy

1. **Week 1-2 (2026-04-17 to 2026-05-01):** Soft launch, monitor feedback
2. **Week 3-6 (2026-05-01 to 2026-05-31):** Active migration, warnings on all legacy invocations
3. **2026-06-01:** Hard cutover, remove legacy name support

## References

- **Full mapping:** `.agents/skills/deprecated-aliases.md`
- **User migration guide:** `docs/reference/migration-guide.md`
- **Intent design:** `docs/plans/2026-04-13-sdp-skill-architecture-design.md`
- **Skills reference:** `docs/reference/skills.md`
