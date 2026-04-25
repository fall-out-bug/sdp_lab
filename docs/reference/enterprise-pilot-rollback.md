# Rollback and Disable Playbook

> **Goal**: Safe rollback and emergency disable procedures for SDP pilot integrations.

## Overview

This playbook covers three scenarios:
1. **Immediate disable** — stop all SDP gates right now
2. **Safe rollback** — revert to pre-SDP state with audit trail preservation
3. **Incident response** — structured communication during SDP-related incidents

## Immediate Disable Path

Use when SDP gates are blocking merges and you need to restore normal flow immediately.

### Disable All Gates

```bash
# Option 1: Config disable (fastest)
sdp config set gates.enabled false

# Option 2: Remove gate jobs from CI workflow
# Edit .github/workflows/ci.yml — comment out or remove these jobs:
#   evidence-gate, scope-gate, consistency-gate, policy-gate, coverage-gate, ci-pass
```

### Disable Specific Gate

```bash
# Disable individual gates by name
sdp config set gates.evidence.enabled false
sdp config set gates.scope.enabled false
sdp config set gates.consistency.enabled false
sdp config set gates.policy.enabled false
sdp config set gates.coverage.enabled false
```

### Disable Contracted Runtime

```bash
# Revert to CI-only mode
sdp config set runtime.mode ci-only

# Or disable runtime entirely
sdp config set runtime.enabled false
```

### Verify Disable

```bash
# Check gates status
sdp config get gates

# Open a test PR to confirm gates are inactive
gh pr create --title "test: verify gates disabled" --body "Testing gate disable"
```

**Re-enable** when ready:
```bash
sdp config set gates.enabled true
```

## Safe Rollback Path

Use when you want to revert to pre-SDP state while preserving the audit trail.

### Step 1: Document Current State

```bash
# Capture current SDP state
sdp log show --last 100 > /tmp/sdp-rollback-audit.log

# Record evidence files present
find .sdp/ -name "*.json" -o -name "*.jsonl" -o -name "*.yml" | sort > /tmp/sdp-rollback-filelist.txt

# Record git state
git log --oneline -20 > /tmp/sdp-rollback-gitlog.txt
```

### Step 2: Disable Gates (Non-Destructive)

```bash
sdp config set gates.enabled false
git add .sdp/config.yml
git commit -m "chore: disable SDP gates for rollback"
```

### Step 3: Preserve Audit Trail

```bash
# Archive evidence (DO NOT DELETE)
mkdir -p .sdp/archive/rollback-$(date +%Y%m%d)
cp -r .sdp/log/ .sdp/archive/rollback-$(date +%Y%m%d)/log/
cp -r .sdp/evidence/ .sdp/archive/rollback-$(date +%Y%m%d)/evidence/ 2>/dev/null || true
cp .sdp/config.yml .sdp/archive/rollback-$(date +%Y%m%d)/config.yml

git add .sdp/archive/
git commit -m "chore: archive SDP audit trail before rollback"
```

### Step 4: Remove CI Gate Jobs

```bash
# Edit .github/workflows/ci.yml
# Remove or comment out: evidence-gate, scope-gate, consistency-gate,
#                         policy-gate, coverage-gate, ci-pass jobs
# Keep: build-test, snapshot-test, push-protection
git add .github/workflows/ci.yml
git commit -m "chore: remove SDP CI gates from workflow"
```

### Step 5: Clean Up (Optional)

```bash
# Remove SDP runtime files (config and archive are preserved above)
rm -rf .sdp/log/events.jsonl
rm -rf .sdp/checkpoints/
rm -rf .sdp/locks/
rm -rf .sdp/memory.db

# Keep .sdp/config.yml and .sdp/archive/ for reference
git add -A .sdp/
git commit -m "chore: clean up SDP runtime files"
```

### Rollback Evidence File

After rollback, a `deploy.rollback.json` is created with:

```json
{
  "rollback_id": "auto-generated-uuid",
  "timestamp": "2026-04-25T12:00:00Z",
  "reason": "pilot-integration-issues",
  "previous_state": "contracted-runtime",
  "target_state": "no-sdp",
  "audit_preserved": true,
  "gates_disabled": ["evidence-gate", "scope-gate", "consistency-gate", "policy-gate", "coverage-gate"],
  "files_archived": [".sdp/log/events.jsonl", ".sdp/evidence/"],
  "operator": "admin@example.com"
}
```

## Incident Communication Template

### Template: SDP Gate Incident

```
Subject: [SDP] Incident Report — {gate_name} blocking merges

## Summary
{1-2 sentence description of what happened}

## Impact
- **Duration**: {start_time} to {end_time} (or "ongoing")
- **Affected teams**: {list teams}
- **Blocked PRs**: {count} PRs unable to merge
- **Root cause**: {gate_name} {failed/succeeded incorrectly} because {reason}

## Timeline
- {HH:MM} — First report of {symptom}
- {HH:MM} — Investigation started
- {HH:MM} — Root cause identified: {cause}
- {HH:MM} — {Action taken} (disable gate / fix config / rollback)
- {HH:MM} — Normal operations restored

## Resolution
{What was done to fix the issue}

## Prevention
{What changes will prevent recurrence}

## Artifacts
- Audit log: .sdp/archive/rollback-{date}/
- Rollback evidence: .sdp/deploy.rollback.json
- CI run: {link to failed GitHub Actions run}
```

### Template: Planned SDP Disable

```
Subject: [SDP] Planned maintenance — gates disabled {date} {time}-{time}

## Window
- **Start**: {date} {time} UTC
- **End**: {date} {time} UTC
- **Duration**: approximately {X} minutes

## Scope
SDP CI gates will be disabled for all PRs to {branches}.

## Reason
{Why gates need to be disabled}

## Action Required
No action required from development teams. PRs will merge without SDP gate checks during this window.

## Rollback Plan
If maintenance overruns, gates will remain disabled until {hard deadline}.

## Contact
{Who to contact for questions}
```

## Fail-Open vs Fail-Closed per Gate Type

| Gate | Default Mode | Fail-Open Behavior | Fail-Closed Behavior | Recommendation |
|------|-------------|-------------------|---------------------|----------------|
| evidence-gate | Fail-closed | Missing evidence → warning | Missing evidence → block | **Fail-closed** (audit integrity) |
| scope-gate | Fail-closed | Scope drift → warning | Scope drift → block | **Fail-closed** (prevent scope creep) |
| consistency-gate | Fail-closed | Protocol violations → warning | Protocol violations → block | **Fail-closed** (protocol integrity) |
| policy-gate | Fail-closed | Policy violations → warning | Policy violations → block | **Fail-closed** (governance) |
| coverage-gate | Fail-open | Below minimum → warning | Below minimum → block | **Fail-open** during pilot, closed after |
| secret-scan | Fail-closed | Secrets found → warning | Secrets found → block (hard gate) | **Fail-closed** always (security) |

### Configuring Gate Behavior

```yaml
# .sdp/guard-rules.yml
gates:
  evidence:
    mode: fail-closed  # block on missing evidence
  scope:
    mode: fail-closed  # block on scope drift
  consistency:
    mode: fail-closed  # block on protocol violations
  policy:
    mode: fail-closed  # block on policy violations
  coverage:
    mode: fail-open    # warn on low coverage (during pilot)
    minimum: 60        # threshold
  secretscan:
    mode: fail-closed  # always block on secrets (hard gate)
```

## Quick Reference Card

```
EMERGENCY: Disable all gates NOW
  sdp config set gates.enabled false

DISABLE ONE GATE
  sdp config set gates.<name>.enabled false

ROLLBACK DEPLOY
  sdp deploy rollback <previous-tag>

ARCHIVE AUDIT TRAIL
  cp -r .sdp/log/ .sdp/archive/rollback-$(date +%Y%m%d)/log/

CHECK GATE STATUS
  sdp config get gates

RE-ENABLE GATES
  sdp config set gates.enabled true
```

## Reference

- [CI-gate-only pilot](./enterprise-pilot-ci-gate-only.md)
- [Contracted runtime pilot](./enterprise-pilot-contracted-runtime.md)
- [CI gates map](./ci-gates-map.md)
- [Evidence coverage](./EVIDENCE-COVERAGE.md)
