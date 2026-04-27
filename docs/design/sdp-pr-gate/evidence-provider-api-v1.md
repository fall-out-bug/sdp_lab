# Evidence Provider API v1

> **Workstream:** 00-151-02
> **Product:** sdp-pr-gate (ChangePassport)
> **Status:** Design v1
> **Schema:** `schema/sdp-pr-gate/evidence-event.schema.json`

## 1. API Contract Overview

The Evidence Provider API defines how external systems feed evidence events into the ChangePassport merge-readiness pipeline. An evidence event is an immutable observed fact about a PR: a CI run passing, a security scanner finding, a reviewer approval, and so on.

Every provider must emit events conforming to the [evidence-event JSON Schema](../../../schema/sdp-pr-gate/evidence-event.schema.json). The API is provider-agnostic and does not prescribe any specific CI platform, scanner, or review tool.

**Design principles:**

1. **Observed facts are immutable.** Once accepted, evidence cannot be overwritten. Providers emit new events to supersede previous ones.
2. **Missing evidence is visible.** Degraded or absent evidence surfaces in the passport as a visible warning, never a silent pass.
3. **Idempotent ingestion.** Identical events (same dedup key) are silently accepted on replay.
4. **Provider-agnostic.** The API does not encode GitHub, GitLab, or any platform-specific concepts in the core contract.

## 2. Field Reference

14 required top-level fields from the manifesto contract. Two optional fields (`artifact_uri`, `error_state`) extend the base contract when applicable.

| # | Field | Type | Required | Description |
|---|-------|------|----------|-------------|
| 1 | `schema_version` | string (const `"1"`) | Yes | Schema version. Must be `"1"` for this API version. |
| 2 | `source` | string | Yes | Provider identifier. Reverse-DNS pattern: `^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)*$`. Max 128 chars. Part of dedup key. |
| 3 | `external_ref` | string | Yes | Provider-unique reference ID (CI run URL, finding ID, comment ID). Max 512 chars. Part of dedup key. |
| 4 | `repository` | object | Yes | Repository identity. Sub-fields: `id` (required), `url` (required), `provider`, `branch`. |
| 5 | `pull_request` | object | Yes | PR identity. Sub-fields: `id` (required), `url`, `title`. |
| 6 | `commit_sha` | string | Yes | Full 40-char hex SHA. Pattern: `^[0-9a-f]{40}$`. Part of dedup key. |
| 7 | `observed_at` | string (date-time) | Yes | When the evidence was originally observed by the provider. ISO 8601. |
| 8 | `collected_at` | string (date-time) | Yes | When the evidence was collected/submitted. Must be >= `observed_at`. ISO 8601. |
| 9 | `actor` | object | Yes | Entity that produced the evidence. Sub-fields: `type` (required), `id` (required), `name`. |
| 10 | `event_type` | string (enum) | Yes | Category: `commit`, `ci_run`, `test_result`, `scan_finding`, `review_comment`, `approval`, `merge`, `deployment`, `custom`. |
| 11 | `status` | string (enum) | Yes | Outcome: `success`, `failure`, `warning`, `skipped`, `degraded`, `pending`. |
| 12 | `summary` | string | Yes | Human-readable summary, max 2048 chars. Self-contained for passport display. |
| 13 | `artifact_uri` | string (uri) or null | No | URI to the primary artifact. Omit or null if no artifact exists. |
| 14 | `artifact_hash` | object or null | No | Cryptographic hash. Sub-fields: `algorithm` (required), `value` (required). Omit if no artifact. |
| 15 | `error_state` | object or null | No | Error details: `code` (required), `message` (required), `retry_possible`. Omit if no error. |

### 2.1 Actor Object

```json
{
  "type": "human | agent | system | tool",
  "id": "user@example.com",
  "name": "Jane Doe"
}
```

- `type`: Who/what produced the evidence. `human` for people, `agent` for AI agents, `system` for CI/CD platforms, `tool` for CLI tools.
- `id`: Unique within the source system.
- `name`: Optional display name.

### 2.2 Error State Object

```json
{
  "code": "PROVIDER_TIMEOUT",
  "message": "Connection to scanner timed out after 30s",
  "retry_possible": true
}
```

- `code`: Machine-readable, dot-separated hierarchical format. Pattern: `^[A-Z][A-Z0-9_]*(\.[A-Z][A-Z0-9_]*)*$`.
- `message`: Human-readable description of the failure.
- `retry_possible`: Whether the provider believes a retry may succeed. Defaults to `false`.

### 2.3 Artifact Hash Object

```json
{
  "algorithm": "sha-256",
  "value": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```

- `algorithm`: One of `sha-256`, `sha-384`, `sha-512`, `blake2b-256`, `blake2b-512`. SHA-256 is required for v1.
- `value`: Hex-encoded digest. 64 chars for SHA-256/BLAKE2b-256, 96 for SHA-384, 128 for SHA-512/BLAKE2b-512.

## 3. Idempotent Ingestion

### 3.1 Dedup Key

The dedup key is the tuple `(source, external_ref, commit_sha)`.

An event with a dedup key matching a previously accepted event is an **idempotent replay**. The API returns `200 OK` with the original response body. No new event is stored.

### 3.2 Replay Handling

| Scenario | Behavior |
|----------|----------|
| Exact duplicate (all fields match) | Return `200 OK`, no new storage |
| Same dedup key, different field values | Return `409 Conflict`. The original event is immutable. Provider must emit a new event with a different `external_ref`. |
| New dedup key | Normal ingestion: validate, store, return `201 Created` |

### 3.3 Dedup Key Constraints

- `source` and `external_ref` are provider-controlled. Providers must ensure `external_ref` is globally unique within their namespace.
- `commit_sha` is 40-char hex. Providers should use the HEAD commit of the PR at the time of evidence collection.

## 4. Provider States and Transitions

Evidence events enter the system in one of three provider states:

```
                    +-----------+
                    |           |
        +---------->  accepted  |
        |           |           |
        |           +-----+-----+
        |                 |
        |   idempotent    |  validation
        |   replay        |  failure
        |                 |
        |                 v
        |           +-----+-----+
        |           |           |
        +-----------+ rejected  |
                    |           |
                    +-----------+

                    +-----------+
                    |           |
                    | degraded  |
                    |           |
                    +-----------+
```

### 4.1 State Definitions

**accepted** -- The event passed validation and is stored in the evidence store. This is the normal path.

**rejected** -- The event failed schema validation, authentication, or quota checks. The event is not stored. The API returns a `4xx` error with details.

**degraded** -- The event was accepted but contains partial or low-confidence data. Indicated by `status: "degraded"` in the event body. The passport must surface this as a visible warning.

### 4.2 State Transitions

| From | To | Trigger | Behavior |
|------|----|---------|----------|
| (new) | accepted | Valid event, new dedup key | Store event, return `201 Created` |
| accepted | accepted | Same dedup key, exact match | No-op, return `200 OK` |
| (new) | rejected | Schema validation failure | Return `400 Bad Request` |
| (new) | rejected | Auth failure | Return `401 Unauthorized` |
| (new) | rejected | Quota exceeded | Return `429 Too Many Requests` |
| (new) | degraded | Valid event with `status: "degraded"` | Store event, return `201 Created`. Passport surfaces warning. |

Note: Once an event is `accepted`, it cannot transition to `rejected` or `degraded`. Immutability is enforced at the storage layer.

## 5. Error Handling: Missing Evidence Propagation

### 5.1 Core Rule

**Missing evidence must not be silently converted into a pass.** If evidence is expected for a category (e.g., CI run, scan finding) but is absent or degraded, the passport must reflect this as a visible gap.

### 5.2 Degraded Evidence Flow

1. Provider submits event with `status: "degraded"` and optionally populates `error_state`.
2. API accepts the event (state: `accepted` in storage, but `degraded` semantically).
3. Passport generator sees the degraded event and includes a **warning entry** with:
   - The provider `source`
   - The `error_state.code` and `error_state.message`
   - A note that this evidence category is incomplete
4. The passport does **not** grant a pass for the affected category.

### 5.3 Missing Evidence

If no event is received for an expected category within the configured timeout:
1. The passport includes a **missing evidence entry** naming the expected provider and category.
2. The passport does **not** grant a pass for the affected category.
3. The passport does **not** assume success from silence.

### 5.4 Error State Propagation

When `error_state` is populated:

| `retry_possible` | Passport behavior |
|-------------------|-------------------|
| `true` | Warning: "Evidence collection failed (transient). Retry may succeed." |
| `false` | Warning: "Evidence collection failed (permanent). Manual intervention required." |
| `null` (no error_state) | Normal processing |

## 6. Authentication Model

### 6.1 Provider Registration

Before submitting events, a provider must register with the ChangePassport service. Registration yields:

- **Provider ID** -- matches the `source` field in events.
- **API Key** -- a secret key for authentication.
- **Allowed event types** -- the set of `event_type` values this provider is authorized to emit.

Registration is an out-of-band process (admin CLI, config file, or management API) in v1.

### 6.2 Authentication Methods

Two methods are supported for v1:

**Method 1: API Key (required)**

```
Authorization: Bearer sdp_evp_<provider_id>_<random_secret>
```

- Every request must include the API key in the `Authorization` header.
- The API key is bound to the provider ID. Events with a `source` that does not match the authenticated provider are rejected (`403 Forbidden`).
- API keys can be rotated by the admin without changing the provider ID.

**Method 2: Signed JWT (optional, for high-security deployments)**

```
Authorization: Bearer <JWT>
```

- The JWT is signed with the provider's private key (RS256 or ES256).
- Claims: `iss` (provider ID), `sub` (provider ID), `iat`, `exp` (max 5 minutes).
- The service validates the signature against the provider's registered public key.
- If both API key and JWT are present, JWT takes precedence.

### 6.3 Source Validation

On every request, the service verifies:

1. The `source` field in the event body matches the authenticated provider ID.
2. The `event_type` is within the provider's allowed set.
3. The API key or JWT has not been revoked.

If any check fails, the event is rejected (`403 Forbidden`).

## 7. Rate Limits and Quotas

### 7.1 Per-Provider Limits

| Limit | Default | Description |
|-------|---------|-------------|
| Events per minute | 100 | Sliding window, per provider |
| Events per PR per commit | 50 | Prevents flooding a single PR |
| Batch size | 25 | Max events in a single batch request |
| Request size | 1 MB | Max request body size |

### 7.2 Rate Limit Headers

Every response includes:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1745800000
```

When rate limited, the API returns `429 Too Many Requests` with:

```json
{
  "error": "RATE_LIMITED",
  "message": "Provider 'github-actions' exceeded 100 events/minute limit",
  "retry_after": 12
}
```

### 7.3 Quota Override

Admins can configure per-provider overrides for rate limits and quotas via the management API (v2 scope).

## 8. Versioning

### 8.1 Schema Version Field

The `schema_version` field in every event indicates which schema version the event conforms to. For v1, this is always `"1"`.

### 8.2 Backwards Compatibility Rules

Within a major version (v1), the following changes are allowed:

- **Adding optional fields** to existing objects.
- **Adding values** to `event_type` and `status` enums.
- **Relaxing constraints** (e.g., increasing `maxLength`).

The following changes are **not** allowed within v1:

- **Removing fields.**
- **Making optional fields required.**
- **Removing enum values.**
- **Tightening constraints.**

### 8.3 Major Version Bumps

Breaking changes require a new major version (v2). The `$id` URI will change:

- v1: `https://sdp.dev/schema/sdp-pr-gate/evidence-event/v1`
- v2: `https://sdp.dev/schema/sdp-pr-gate/evidence-event/v2`

The API endpoint will also version:

- v1: `POST /api/v1/evidence`
- v2: `POST /api/v2/evidence`

### 8.4 Provider Negotiation

Providers declare their supported version via the `schema_version` field. The API validates against the declared version. If the version is unsupported, the API returns:

```json
{
  "error": "UNSUPPORTED_VERSION",
  "message": "schema_version '0' is not supported. Supported versions: 1",
  "supported_versions": [1]
}
```

## 9. Batch vs Single Event Ingestion

### 9.1 Single Event

```
POST /api/v1/evidence
Content-Type: application/json
Authorization: Bearer <key>

{ ... single evidence event ... }
```

Response:
- `201 Created` -- new event stored
- `200 OK` -- idempotent replay
- `4xx` -- validation/auth error

### 9.2 Batch Event

```
POST /api/v1/evidence/batch
Content-Type: application/json
Authorization: Bearer <key>

[
  { ... event 1 ... },
  { ... event 2 ... },
  { ... event N ... }
]
```

Response:
```json
{
  "total": 3,
  "created": 2,
  "idempotent": 1,
  "rejected": 0,
  "results": [
    { "index": 0, "status": "created", "event_id": "evt_abc123" },
    { "index": 1, "status": "created", "event_id": "evt_def456" },
    { "index": 2, "status": "idempotent", "event_id": "evt_ghi789" }
  ]
}
```

- Max batch size: 25 events.
- Each event is validated independently. A rejected event does not affect other events in the batch.
- Batch events share the same authentication and rate limit quota.
- Partial success: the batch response includes per-event status. HTTP status code is `207 Multi-Status` if any events were rejected.

### 9.3 Ordering

Events within a batch have no ordering guarantees. Each event is processed independently.

## 10. Provider Health and Heartbeat

### 10.1 Heartbeat Endpoint

Providers may send periodic heartbeat requests to signal liveness:

```
POST /api/v1/evidence/heartbeat
Content-Type: application/json
Authorization: Bearer <key>

{
  "source": "github-actions",
  "status": "healthy",
  "timestamp": "2026-04-27T10:30:00Z",
  "message": "All GitHub Actions runners operational"
}
```

### 10.2 Health Status

| Provider status | Description |
|-----------------|-------------|
| `healthy` | Provider is operating normally |
| `degraded` | Provider is experiencing partial outages |
| `down` | Provider is offline |

### 10.3 Health Monitoring

- If no heartbeat is received within the configured interval (default: 5 minutes), the provider is marked `unknown`.
- If a provider reports `degraded` or `down`, the passport surfaces this: "Provider 'github-actions' reports degraded health. Evidence may be incomplete."
- Heartbeat is optional. Providers that do not send heartbeats are not penalized, but the system cannot report their health status.

### 10.4 Health Check Response

```
GET /api/v1/evidence/health/{source}
Authorization: Bearer <key>
```

```json
{
  "source": "github-actions",
  "status": "healthy",
  "last_heartbeat": "2026-04-27T10:30:00Z",
  "last_event": "2026-04-27T10:28:15Z",
  "events_today": 342
}
```

## 11. HTTP API Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/evidence` | Submit a single evidence event |
| `POST` | `/api/v1/evidence/batch` | Submit a batch of evidence events |
| `POST` | `/api/v1/evidence/heartbeat` | Send provider heartbeat |
| `GET` | `/api/v1/evidence/health/{source}` | Get provider health status |
| `GET` | `/api/v1/evidence/{event_id}` | Retrieve a stored event by ID |

### Common HTTP Status Codes

| Code | Meaning |
|------|---------|
| `200 OK` | Idempotent replay (event already exists) |
| `201 Created` | New event stored |
| `207 Multi-Status` | Batch with partial failures |
| `400 Bad Request` | Schema validation failure |
| `401 Unauthorized` | Missing or invalid credentials |
| `403 Forbidden` | Source mismatch or unauthorized event_type |
| `404 Not Found` | Event or provider not found |
| `409 Conflict` | Same dedup key with different content |
| `429 Too Many Requests` | Rate limit exceeded |
| `500 Internal Server Error` | Server-side failure |
| `503 Service Unavailable` | Temporary service outage |

## 12. Evidence Integrity

Every passport ties evidence to externally verifiable references. The API enforces this by requiring:

1. **Repository URL + PR URL** -- Verifiable links to the change.
2. **Commit SHA** -- Exact commit the evidence was observed on.
3. **External reference** -- Link to the CI run, scanner finding, or review comment in the source system.
4. **Artifact hash** -- Cryptographic integrity check on referenced artifacts.
5. **Timestamps** -- Both `observed_at` (provider's clock) and `collected_at` (submission time) for temporal integrity.
6. **Actor identity** -- Who or what produced the evidence.

The passport generator uses these references to build a complete, auditable chain from evidence to decision.
