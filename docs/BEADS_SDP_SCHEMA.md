# Beads SDP Schema

> **Status:** Proposed
> **Date:** 2026-03-24
> **Artifact:** C — Beads Mapping / Operational Graph

---

## Issue Types

| Type | Description | Beads label |
|---|---|---|
| `feature` | User-facing feature or deliverable | `sdp:feature` |
| `clarify` | Clarification task (waiting for human input) | `sdp:clarify` |
| `contract` | Task contract generation | `sdp:contract` |
| `review` | Code review task | `sdp:review` |
| `qa` | QA/validation task | `sdp:qa` |
| `release` | Release/deployment task | `sdp:release` |
| `gate:human` | Human approval gate | `sdp:gate:human` |
| `gate:ci` | CI pipeline gate | `sdp:gate:ci` |
| `gate:pr` | PR merge gate | `sdp:gate:pr` |
| `gate:timer` | Timer-based gate | `sdp:gate:timer` |

## Status Values

| Status | Meaning | Transitions From |
|---|---|---|
| `backlog` | Not yet prioritized | (initial) |
| `ready` | All dependencies met, can be claimed | `backlog`, `blocked` |
| `in_progress` | Currently being worked on | `ready` |
| `blocked` | Cannot proceed (dependency or gate) | `ready`, `in_progress` |
| `needs_clarification` | Waiting for human input | `in_progress`, `blocked` |
| `pending_review` | Work done, awaiting review | `in_progress` |
| `pending_qa` | Review passed, awaiting QA | `pending_review` |
| `done` | Completed and verified | `pending_qa` |
| `released` | Deployed to target environment | `done` |
| `failed` | Execution failed | `in_progress` |
| `escalated` | Failed threshold exceeded, needs intervention | `failed` |

## Gate Taxonomy

### Gate Types

| Gate | Trigger | Resolution |
|---|---|---|
| `human` | Phase transition requires human approval | Human approves/rejects via A2A or CLI |
| `ci` | Phase transition requires CI pass | CI pipeline callback |
| `pr` | Phase transition requires merged PR | PR webhook |
| `timer` | Phase transition requires time elapsed | Timer check on dispatch cycle |

### Gate Lifecycle

1. Created when phase transition is attempted
2. `pending` → `passed` or `failed`
3. If `failed` → associated issue becomes `blocked`
4. Gates are modeled as issues of type `gate:*` with dependency on the parent issue

## Metadata Schema

Every SDP issue carries metadata under `metadata.sdp`:

```json
{
  "sdp": {
    "card_id": "F069",
    "phase": "build",
    "contract": { "id": "CTR-001", "hash": "sha256:..." },
    "executor": {
      "role": "omo-implementation",
      "session_id": "ses_abc",
      "state": "running",
      "started_at": "2026-03-24T12:00:00Z"
    },
    "review": { "state": "pending", "attempts": 0, "last_attempt_at": null },
    "delivery": {
      "target": "staging",
      "state": "pending",
      "rollback_count": 0
    },
    "provenance": {
      "packet_hash": "sha256:...",
      "prompt_hash": "sha256:...",
      "artifact_hashes": {}
    },
    "scope_in": ["internal/auth/**"],
    "scope_out": ["deploy/**"]
  }
}
```

## Relationship Model

### Parent-child graph

```
F069 (feature)
├── F069-clarify (clarify)
├── CTR-001 (contract)
├── F069-build (feature, phase=build)
│   └── F069-review (review)
│       └── F069-qa (qa)
└── F069-release (release)
    └── G-F069-release-human (gate:human)
```

### Dependency rules

- Build depends on contract (CTR-001)
- Review depends on build
- QA depends on review
- Release depends on QA + human gate
- Each phase can only start when all dependencies are `done`

## Query Patterns

### What is next?
```
Issues where type=feature AND status=ready, ordered by priority
```

### Why blocked?
```
For issue X: find dependencies where status != done
```

### What needs my approval?
```
Issues where type=gate:human AND status=pending
```

### What lacks evidence?
```
Issues where type in (review, qa) AND metadata.sdp.phase matches
  AND metadata.sdp.provenance.artifact_hashes is empty
```

### Full trace for feature X?
```
Issue X → all descendants (phase, review, qa, gates)
  → join with external artifacts (evidence, provenance files)
```
