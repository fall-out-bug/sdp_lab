# GitHub App v1 Flow Design

**Feature:** F151-05 (sdplab-hfk0.5)
**Internal namespace:** sdp-pr-gate
**Display name:** ChangePassport
**Status:** Design v1

## Overview

The GitHub App v1 flow implements the "GitHub PR Gate Loop" described in the ChangePassport manifesto v2 §"v1 Loop". The app operates as a GitHub App with least-privilege permissions, listening to webhook events and posting check results.

No implementation is provided — this document defines the flow for future implementation.

## App Installation and Permissions

### GitHub App permissions (least privilege)

| Permission | Access | Justification |
|---|---|---|
| Checks | Read + Write | Post and update PR check status |
| Pull requests | Read | Read PR metadata, comments, reviews, files |
| Commit statuses | Read | Read CI status |
| Contents | Read | Read changed files (scope detection) |
| Issues | Read | Read linked issue metadata |
| Metadata | Read | Repository metadata (automatic) |
| Members | Read | Verify user roles for override validation |

### Webhook events

| Event | Trigger | Action |
|---|---|---|
| `pull_request` | opened, synchronize | Generate/regenerate passport |
| `pull_request` | closed (merged) | Archive passport |
| `issue_comment` | created (on PR) | Parse for override trigger |
| `check_run` | completed | Collect CI evidence |
| `installation` | added, removed | Register/unregister repository |
| `pull_request_review` | submitted | Collect review evidence |

### Installation flow

```text
1. Org admin installs GitHub App
2. Selects repositories (or all)
3. App receives installation webhook
4. App registers repository in its config
5. App posts welcome comment on next open PR in registered repos
```

## Check Status Mapping

The GitHub Check status maps to passport decisions:

| Decision | Check status | Check conclusion | Visual |
|---|---|---|---|
| merge | completed | success | Green check |
| hold | completed | action_required | Yellow warning |
| rework | completed | failure | Red X |
| escalate | completed | neutral | Gray circle |
| override | completed | success | Green check + "override" label |

### Check output

Each check run includes:
- **Title**: `ChangePassport: {decision}` (e.g. "ChangePassport: ready")
- **Summary**: 2-3 sentence passport summary
- **Details**: Link to full passport (Markdown rendered)
- **Annotations**: Findings surfaced as check annotations on specific files/lines

```json
{
  "name": "ChangePassport",
  "head_sha": "abc123...",
  "status": "completed",
  "conclusion": "success",
  "output": {
    "title": "ChangePassport: ready",
    "summary": "All evidence collected. 6 events from 3 providers. 0 unresolved findings.",
    "text": "## Passport\n\n[full markdown passport content]"
  }
}
```

## Main Sequence: PR Open → Decision

```text
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ PR Open  │────>│  Scope   │────>│ Evidence │────>│ Passport │
│ Webhook  │     │  Seed    │     │ Collect  │     │ Generate │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
                                                          │
                                                          v
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ Decision │<────│  Check   │<────│ Findings │<────│  Risk    │
│ Finalize │     │  Post    │     │ Resolve  │     │ Assess   │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
```

### Step 1: PR Open Webhook

Trigger: `pull_request` event with action `opened` or `synchronize`

1. Verify app is installed for this repository
2. Extract PR metadata: number, title, author, labels, head SHA, base branch
3. Extract changed files via GitHub API
4. Extract linked issues (from PR body)

### Step 2: Scope Seed

1. **Auto-seed scope** from:
   - PR template fields (if present)
   - Linked issue title/body
   - Labels
   - Changed files (grouped by directory/module)

2. **Post scope confirmation comment**:
   ```text
   🛂 **ChangePassport — Scope Confirmation**

   Detected scope:
   - ✅ `internal/cache/` — Redis caching layer
   - ✅ `config/` — Cache configuration
   - ⬜ `monitoring/` — Dashboard update (out of scope?)

   React with 👍 to confirm, or comment corrections.
   ```

3. **Wait for confirmation** (timeout: 30 minutes):
   - 👍 reaction on the comment → confirmed
   - Corrective comment → update scope
   - Timeout → proceed with auto-detected scope (all `unknown`)

### Step 3: Evidence Collection

Collect evidence from registered providers:

| Provider | Trigger | Data collected |
|---|---|---|
| GitHub Actions | `check_run` completed event | CI results, test outcomes |
| Scanner (SonarQube/etc.) | Via Evidence Provider API | SAST findings |
| GitHub Reviews | `pull_request_review` submitted | Review status, comments |
| Git Commits | From PR diff | Commit metadata, author info |

Collection timeout: 10 minutes after last PR sync. If evidence is incomplete at timeout, proceed with what's available and mark providers as `degraded` or `pending`.

### Step 4: Passport Generation

1. Build passport from collected evidence using `passport.schema.json`
2. Compute integrity hash
3. Store passport (content-addressable)

### Step 5: Findings Resolution

1. Surface findings from evidence
2. Auto-resolve where possible (e.g., finding fixed in later commit)
3. Mark remaining as `open`
4. Add risk items for unresolved findings

### Step 6: Check Post

Post GitHub Check with appropriate status (see Check Status Mapping above).

### Step 7: Decision Finalize

If auto-merge is not enabled:
- Wait for human decision_owner to finalize
- System recommendation is shown in check status

If auto-merge is enabled (opt-in per repo):
- `merge` decisions are auto-applied
- `hold`/`rework`/`escalate` wait for human

## Override Sequence: Comment → Audit

```text
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Comment      │────>│ Validate     │────>│ Create Audit │
│ Posted       │     │ Author+Role  │     │ Entry        │
└──────────────┘     └──────────────┘     └──────────────┘
                                                  │
                                                  v
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Confirmation │<────│ Update Check │<────│ Update       │
│ Comment      │     │ Status       │     │ Passport     │
└──────────────┘     └──────────────┘     └──────────────┘
```

### Step 1: Comment Posted

Trigger: `issue_comment` event on a PR (not an issue)

1. Parse comment for `@changepassport override --reason="..."` pattern
2. If no match, ignore (not an override attempt)

### Step 2: Validate

Per override protocol (00-151-04):
1. Verify webhook signature
2. Verify comment author has `decision_owner` or `admin` role
3. Verify PR is open
4. Verify reason meets minimum length (10 chars)
5. Verify rate limit (5 per PR per hour)

### Step 3: Create Audit Entry

Create immutable, signed audit entry (see 00-151-03).

### Step 4: Update Passport + Check

1. Update passport decision section with override
2. Post updated GitHub Check (status: success, label: "override")

### Step 5: Confirmation Comment

```text
🛂 **Override recorded**

- **By:** @bob (decision_owner)
- **Reason:** Client deadline accepted; scanner finding tracked in #350
- **Audit ref:** audit-uuid-here
- **Previous recommendation:** hold

The override is logged in the permanent audit trail.
```

## Error Handling

### Webhook delivery failure

- GitHub retries webhooks up to 3 times over 20 hours
- If all retries fail, the PR remains without a ChangePassport check
- The app exposes a manual re-trigger API: `POST /api/v1/passports/generate?pr=342&repo=acme/webapp`

### Timeout during evidence collection

- Collection timeout: 10 minutes after last PR sync
- Partial passports are generated with `collection_status: "degraded"`
- Missing providers are listed with status `unavailable`
- Check status: `hold` (action required)

### Rate limiting

- GitHub API rate limit: handled with conditional requests (ETag/If-Modified-Since)
- Internal rate limit: 5 override attempts per PR per hour
- Webhook processing: max 100 concurrent webhook handlers per installation

### Partial evidence

If some providers return errors:
1. Generate passport with available evidence
2. Mark degraded providers in evidence section
3. Add `missing_evidence` risk items
4. Check status: `hold` (unless all critical providers succeeded)

### Race conditions

Multiple events for the same PR (e.g., rapid pushes):
1. Events are serialized per PR (no parallel processing for same PR)
2. Latest event wins for evidence collection (previous in-flight collection is cancelled)
3. Passport regeneration replaces previous passport (linked via `previous_passport_hash`)

## No-Egress Security Posture

The app operates with no-egress-by-default:

1. **Inbound only**: The app receives webhooks and API calls. It does NOT make outbound calls to external services.
2. **Evidence collection**: The app reads GitHub data via GitHub API (same origin). It does NOT call external CI systems, scanners, or APIs.
3. **Provider push model**: External providers push evidence events TO the app via the Evidence Provider API. The app does not pull from providers.
4. **No telemetry egress**: The app does not send analytics, metrics, or logs to external services.
5. **Configuration**: All configuration is stored locally. No external config services.

Exceptions (require explicit opt-in per repository):
- Outbound webhook notifications (e.g., Slack integration) — future, not v1
- External passport storage (e.g., S3) — future, not v1

## App Configuration

Per-repository configuration (stored in `.changepassport.yaml` or via app settings API):

```yaml
# .changepassport.yaml
version: 1

# Required providers (passport blocked if these are degraded/unavailable)
required_providers:
  - github-actions
  - github-review

# Optional providers (missing = warning, not block)
optional_providers:
  - sonarqube
  - snyk

# Decision owner assignment
decision_owner:
  # Map GitHub teams or users to decision owner role
  default: @engineering-leads
  paths:
    "internal/auth/*": @security-team

# Auto-merge (disabled by default)
auto_merge: false

# Scope confirmation timeout (minutes)
scope_timeout: 30
```
