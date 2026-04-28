# Override Protocol v1 — Design Document

**Feature:** F151-04 (sdplab-hfk0.4)
**Internal namespace:** sdp-pr-gate
**Status:** Design v1

## Overview

The override protocol allows a decision owner to explicitly override the system's readiness recommendation with a documented reason. Overrides are always auditable, never silent, and never unattributed.

## Override Roles

| Role | Can override | Scope |
|---|---|---|
| `decision_owner` | Any decision on their assigned PRs | Their PRs only |
| `reviewer` | Cannot override directly | Must request decision_owner |
| `admin` | Any decision on any PR in their scope | Organization/repository scope |

### Role assignment

- `decision_owner` is assigned per PR (typically the PR author's manager or the team lead)
- `admin` is a repository-level or organization-level role
- `reviewer` cannot override; they can only comment and approve/reject reviews

## Comment-Trigger Format

Overrides are triggered via GitHub PR comments. The format is:

```text
@changepassport override --reason="<reason text>"
```

### Parsing grammar

```
override_comment := trigger_whitespace "@changepassport" whitespace "override" reason_clause
reason_clause    := whitespace "--reason=" quoted_string
quoted_string    := '"' ( any_char_except_quote )* '"'
whitespace       := ( ' ' | '\t' )+
trigger_whitespace := (optional leading whitespace)
```

### Examples

**Example 1: Simple override**
```text
@changepassport override --reason="Client deadline accepted; scanner finding tracked in #350"
```

**Example 2: Override with context**
```text
Looks good overall. The SAST finding is a false positive — I've verified manually.

@changepassport override --reason="SAST finding S102 is false positive; verified by manual code review. Tracked in security-backlog."
```

### Parsing rules

1. The trigger `@changepassport override` must appear verbatim (case-insensitive for the trigger keyword, case-sensitive for the username)
2. `--reason=` is required; no override without a reason
3. The reason must be at least 10 characters (prevents low-effort overrides like "ok")
4. Multiple override comments in the same PR: only the most recent is active, but all are preserved in the audit trail
5. The comment author must have `decision_owner` or `admin` role for the PR

## Required Override Fields

Every override produces a decision record with:

| Field | Value | Rationale |
|---|---|---|
| `decision` | `"override"` | Fixed |
| `decided_by` | Comment author | Attribution |
| `reason` | From `--reason=` parameter | Accountability |
| `decided_at` | Timestamp of comment | When it happened |
| `evidence_snapshot_ref` | Current snapshot hash | Evidence state at override |
| `audit_ref` | New audit entry | Immutable trail |
| `override_detail.trigger` | `"comment"` | How triggered |
| `override_detail.original_decision` | Previous system rec | What was overridden |
| `override_detail.previous_decision_id` | Previous UUID | Chain reference |

## Audit Log Entry

Each override creates an immutable audit log entry:

```json
{
  "audit_id": "audit-uuid",
  "passport_id": "passport-uuid",
  "event_type": "override",
  "trigger": "comment",
  "actor": { "id": "bob", "type": "human", "name": "Bob Manager" },
  "comment_ref": "https://github.com/acme/webapp/pull/345#issuecomment-12345",
  "reason": "Client deadline accepted; scanner finding tracked in #350",
  "previous_decision": "hold",
  "evidence_snapshot_ref": "sha256:...",
  "timestamp": "2026-05-01T14:10:00Z",
  "signature": {
    "signer": "sdp-pr-gate-service",
    "algorithm": "ed25519",
    "value": "base64-signature"
  }
}
```

## Security Considerations

### Impersonation prevention

1. **GitHub identity verification**: The comment must come from an authenticated GitHub user. The webhook payload contains the verified user identity.
2. **Role check**: The service verifies the comment author has `decision_owner` or `admin` role for the PR's repository.
3. **No bot overrides**: Comments from bot accounts (GitHub bots, other GitHub Apps) are rejected unless explicitly allowlisted.

### Comment verification

1. The webhook delivery must have a valid GitHub signature (HMAC-SHA256)
2. The comment must be on an open PR (not closed/merged)
3. The comment must be a new comment (not edited). Edited comments are ignored for override triggers.
4. Rate limiting: max 5 override attempts per PR per hour

### Audit integrity

1. Audit entries are append-only: no API exists to modify or delete them
2. Each entry is signed by the service key
3. Entries are stored with content-addressable references
4. The audit trail is independently verifiable without the service running

## Override Flow (End to End)

```text
1. Human posts comment:
   "@changepassport override --reason="...""

2. GitHub delivers webhook to sdp-pr-gate

3. Service validates:
   a. Webhook signature (HMAC-SHA256)
   b. Comment author identity
   c. Author has decision_owner or admin role
   d. PR is still open
   e. Reason meets minimum length
   f. Rate limit not exceeded

4. Service creates:
   a. Decision record (decision: override)
   b. Audit log entry (immutable, signed)

5. Service updates:
   a. Passport (decision section updated)
   b. GitHub Check status → ready

6. Service posts confirmation comment:
   "Override recorded by @bob: 'Client deadline accepted...'. Audit ref: audit-uuid"

7. Override is visible in:
   a. GitHub Check status (ready)
   b. Passport (override block)
   c. Audit trail (immutable entry)
   d. Confirmation comment on PR
```

## Non-GitHub Override Triggers (v2 consideration)

v1 supports only GitHub PR comments. Future triggers:
- REST API: `POST /api/v1/decisions/{passport_id}/override`
- CLI: `sdp pr-gate override --reason="..." --pr=342`
- Slack integration: `/changepassport override reason="..."`

These are NOT part of v1 design but the protocol schema supports `trigger: "api"` and `trigger: "manual"` for forward compatibility.
