# FeatureCard → Beads Mapping

> **Status:** Proposed
> **Date:** 2026-03-24
> **Artifact:** B — Feature / Intent Spec Layer

---

## Position

FeatureCard stays as a **semantic artifact**. It is not a competing source of truth for lifecycle state.

The canonical lifecycle state lives in Beads. FeatureCard provides:
- Intake/spec semantics
- Links to bead ID, contract ID, provenance ID
- No independent lifecycle engine

## Mapping

### FeatureCard fields → Beads placement

| FeatureCard Field | Target | Notes |
|---|---|---|
| `id` | Beads Issue `key` | Primary identifier |
| `title` | Issue `title` | Direct mapping |
| `status` | Issue `status` | **Beads is truth**. FeatureCard status is derived. |
| `priority` | Issue `priority` | Direct |
| `phase` | Issue `metadata.sdp.phase` | `clarify\|spec\|build\|verify\|review\|release` |
| `dependencies` | Issue `dependencies` | Beads native |
| `blockers` | Issue `dependencies` (blocked type) | Beads native |
| `executor_role` | Issue `metadata.sdp.executor.role` | `omo-implementation`, `review`, `clarification`, `human-admin` |
| `executor_session_id` | Issue `metadata.sdp.executor.session_id` | Runtime metadata |
| `executor_runtime_state` | Issue `metadata.sdp.executor.state` | Derived from execution evidence |
| `dispatched_packet_path` | External artifact | File-based, referenced from metadata |
| `executor_result` | External artifact | File-based, referenced from metadata |
| `scope_in` | Issue `metadata.sdp.scope_in` | File patterns allowed |
| `scope_out` | Issue `metadata.sdp.scope_out` | File patterns denied |
| `constraints` | Issue `metadata.sdp.constraints` | Free-form constraints |
| `objective` | Issue `description` or Issue `metadata.sdp.objective` | Rich text |
| `created_at` | Issue `time.created` | Timestamp |
| `updated_at` | Issue `time.updated` | Timestamp |

### Beads metadata schema for SDP

```json
{
  "sdp": {
    "card_id": "F069",
    "phase": "build",
    "contract": {
      "id": "CTR-001",
      "hash": "sha256:abc123..."
    },
    "executor": {
      "role": "omo-implementation",
      "session_id": "ses_xyz",
      "state": "running"
    },
    "review": {
      "state": "pending",
      "attempts": 0
    },
    "delivery": {
      "target": "staging",
      "state": "pending",
      "rollback_count": 0
    },
    "provenance": {
      "packet_hash": "sha256:def456...",
      "prompt_hash": "sha256:ghi789..."
    },
    "scope_in": ["internal/auth/**", "cmd/**"],
    "scope_out": ["deploy/**", "docs/**"]
  }
}
```

### What stays as files

- Dispatch packets (`.sdp/dispatches/*.json`)
- Result packets (`.sdp/executor-results/*.json`)
- Provenance files (`.sdp/prompt-provenance.json`)
- Evidence envelopes (`.sdp/evidence/{featureID}/{phase}.json`)
- Intake markdown (`docs/workstreams/backlog/*.md`)
- Contract artifacts (`contracts/*.json`)

### What moves to Beads-backed truth

- Readiness (is this item ready to be worked on?)
- Blockers (what is blocking this item?)
- Current work status (what phase is this item in?)
- Assignment / claim (who is working on this?)
- Follow-up routing references

## Implementation Notes

1. **FeatureCard model simplification**: Remove lifecycle engine methods. Card becomes a read-only view of Beads state + external artifact references.
2. **Dual-read during migration**: Card can hydrate from both file store and Beads adapter. Priority: Beads if available, file fallback.
3. **Write path**: SDP writes lifecycle state to Beads. External artifacts (evidence, provenance) remain file-based.
