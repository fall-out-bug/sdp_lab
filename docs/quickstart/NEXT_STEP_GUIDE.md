# Next-Step Guide

SDP provides deterministic next-step guidance through the `sdp-ready` CLI.

## Quick Start

```bash
sdp-ready                        # Current status and next action
sdp-ready --format json          # JSON output for automation
sdp-ready --format status-view   # StatusView contract output
sdp-ready --instructions         # Step-by-step instructions
```

## StatusView Contract

```json
{
  "spec_version": "v1.0",
  "timestamp": "2026-03-14T15:30:00Z",
  "ready_count": 5,
  "blocked_count": 2,
  "in_progress_count": 1,
  "items": [
    {
      "id": "sdplab-abc",
      "title": "Task title",
      "status": "ready",
      "priority": 1,
      "blocked_by": [],
      "labels": ["F069", "protocol"]
    }
  ],
  "next_action": {
    "recommended": "Start sdplab-abc",
    "reason": "sdplab-abc has the highest priority among ready issues",
    "command": "bd update sdplab-abc --status in_progress"
  }
}
```

## InstructionPayload Contract

```json
{
  "spec_version": "v1.0",
  "context": "Action: start",
  "instructions": [
    {
      "step": 1,
      "action": "Claim the issue sdplab-abc",
      "reason": "sdplab-abc has the highest priority",
      "command": "bd update sdplab-abc --status in_progress"
    }
  ]
}
```

## Next-Action Logic

| Condition | Recommendation |
|-----------|----------------|
| In-progress items exist | Continue current work |
| Ready items exist | Start highest priority |
| All blocked | Resolve blockers |
| Empty queue | No action required |

## Action Types

| Action | Description |
|--------|-------------|
| `continue` | Resume in-progress work |
| `start` | Begin new work |
| `resolve_blockers` | Fix blocking issues |
| `check_status` | General status inquiry |

## Troubleshooting

### No Ready Work

```bash
bd blocked              # Check blockers
sdp-ready --no-cache    # Force refresh
```

### All Work Blocked

```bash
sdp-ready --instructions --action resolve_blockers
```

## Agent Integration

```bash
STATUS=$(sdp-ready --format status-view)
sdp-ready --format status-view --instructions
```
