# F125 Migration Rollout Checklist

**Purpose:** Step-by-step guide for rolling out the intent model migration.
**Feature:** F125 (Toolkit UX — intent-routed skills over composable tools)
**Timeline:** 2026-04-17 → 2026-06-01

## Pre-Rollout (2026-04-17)

### Documentation ✅

- [x] Create migration guide (`docs/reference/migration-guide.md`)
- [x] Create deprecated aliases mapping (`.agents/skills/deprecated-aliases.md`)
- [x] Create deprecation implementation guide (`docs/reference/deprecation-implementation-guide.md`)
- [x] Update skills reference (`docs/reference/skills.md`) with intents first
- [x] Add deprecation warnings to legacy skill section

### Verification

- [x] Verify all 26 legacy skills have 1:1 intent mapping
- [ ] Test migration guide examples with actual skill invocations
- [x] Confirm deprecation warnings are clear and actionable
- [ ] Check all links in migration docs work

## Phase 1: Soft Launch (Week 1-2: 2026-04-17 to 2026-05-01)

### Goals

- Introduce intent model without breaking existing workflows
- Gather feedback on deprecation warnings
- Identify most-used legacy skills for targeted migration

### Actions

- [x] Deploy deprecation warnings to production
- [ ] Monitor analytics for legacy skill usage
- [ ] Collect user feedback on warning clarity
- [ ] Identify and document any edge cases

### Success Criteria

- Zero breaking changes to existing workflows
- Deprecation warnings shown for all legacy skills
- Migration guide accessible and helpful
- No increase in support requests related to skills

## Phase 2: Active Migration (Week 3-6: 2026-05-01 to 2026-05-31)

### Goals

- Drive adoption of intent-based skills
- Reduce legacy skill usage to <10%
- Fix any issues discovered in Phase 1

### Actions

- [ ] Feature migration guide prominently in docs
- [ ] Send migration announcement to users
- [ ] Update examples and tutorials to use intents
- [ ] Provide office hours or Q&A for migration questions
- [ ] Create migration scripts for common patterns

### Migration Scripts

Create helper scripts for common patterns:

```bash
# Find all legacy skill usage in docs
scripts/find-legacy-skills.sh docs/

# Replace legacy skill invocations
scripts/migrate-to-intents.sh docs/
```

### Success Criteria

- Legacy skill usage <10% of total skill invocations
- Migration guide viewed by >80% of active users
- No critical bugs in deprecation warning system
- Clear migration path for all documented workflows

## Phase 3: Hard Cutover (2026-06-01)

### Goals

- Remove legacy skill name support
- Complete migration to intent model
- Update all documentation to remove legacy references

### Actions

- [ ] Remove legacy skill routing from harness
- [ ] Update error messages for unknown skills
- [ ] Remove legacy skill examples from docs
- [ ] Archive legacy skill documentation
- [ ] Update all training materials

### Success Criteria

- Legacy skill names return clear error messages
- All documentation references intents only
- Zero legacy skill invocations in production
- User support requests stable or decreased

## Post-Rollout (2026-06-01+)

### Monitoring

- [ ] Monitor for any remaining legacy skill usage attempts
- [ ] Track user satisfaction with intent model
- [ ] Measure skill discoverability improvement
- [ ] Document lessons learned

### Continuous Improvement

- [ ] Update intent model based on usage patterns
- [ ] Refine mode auto-detection heuristics
- [ ] Add new intents/modes as needed
- [ ] Keep migration guide archived for historical reference

## Testing Checklist

### Before Each Phase

- [ ] Test all deprecation warnings with real invocations
- [ ] Verify routing preserves original behavior
- [ ] Check migration guide links and examples
- [ ] Confirm harness compatibility (Claude Code, Codex, OpenCode, Cursor)

### After Each Phase

- [ ] Review analytics for adoption metrics
- [ ] Collect and categorize user feedback
- [ ] Document and fix any issues discovered
- [ ] Update documentation based on learnings

## Rollback Plan

If critical issues arise:

1. **Phase 1/2:** Revert deprecation warnings, investigate issue
2. **Phase 3:** Restore legacy skill support, delay cutover

### Rollback Triggers

- >5% increase in skill-related support requests
- Critical bugs in intent routing
- Negative user feedback on migration experience
- Breaking changes to existing workflows

## Communication Plan

### Announcements

- **Phase 1:** "Introducing Intent-Based Skills" — soft launch announcement
- **Phase 2:** "Skill Migration Reminder" — active migration push
- **Phase 3:** "Legacy Skill Removal" — hard cutover notice

### Channels

- Documentation (migration guide, skills reference)
- Release notes / changelog
- Team meetings / office hours
- Support ticket templates

## Metrics to Track

| Metric | Phase 1 Target | Phase 2 Target | Phase 3 Target |
|--------|---------------|---------------|----------------|
| Legacy skill usage | Baseline | <50% | 0% |
| Intent skill usage | >10% | >90% | 100% |
| Migration guide views | >50% | >80% | N/A |
| Support requests (skills) | No increase | <baseline | <baseline |
| User satisfaction | Stable | Improved | Improved |

## References

- **Migration guide:** `docs/reference/migration-guide.md`
- **Deprecated aliases:** `.agents/skills/deprecated-aliases.md`
- **Implementation guide:** `docs/reference/deprecation-implementation-guide.md`
- **Skills reference:** `docs/reference/skills.md`
- **Intent design:** `docs/plans/2026-04-13-sdp-skill-architecture-design.md`

## Owner

**Feature:** F125 (Toolkit UX — intent-routed skills over composable tools)
**Workstream:** WS 00-125-04 (Migration Harness and Documentation Cutover)
**Status:** Phase 1 complete — all documentation updated, commands.json fixed, deprecation warnings implemented
