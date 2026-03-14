# Failure Recovery Examples

This document shows common failure scenarios and how SDP guides recovery.

## Scenario 1: Missing Prerequisites

### Failure
```bash
$ sdp up --profile oss-combine
Error: Missing required tools:
  - bd: Beads issue tracker not found
  - opa: Open Policy Agent not found
```

### Recovery Guidance
```bash
$ sdp-ready --instructions --action check_status

Context: Action: check_status

Instructions:

1. Check current status
   Reason: Get a complete picture of ready, blocked, and in-progress work
   Command: sdp-ready
   Expected: Status summary with next action recommendation

2. Install missing tools
   Reason: bd is required for issue tracking
   Command: go install github.com/your-org/beads@latest
   Expected: bd binary available in PATH

3. Re-run bootstrap
   Reason: Verify all prerequisites are satisfied
   Command: sdp up --profile oss-combine --dry-run
   Expected: No missing tool errors
```

## Scenario 2: Guard Denial

### Failure
```bash
$ git push --force
Error: [sdp-guard] Command denied by policy
  Phase: build
  Reason: force push is not allowed on protected branches
  Gate: push-protection
```

### Recovery Guidance
```bash
$ sdp-ready --instructions --action resolve_blockers

Instructions:

1. Review blocking issues
   Reason: 1 issue(s) are blocked - identify root causes
   Command: bd blocked
   Expected: List of blocked issues and their blockers

2. Use safe alternative
   Reason: force push is blocked; use regular push or create new branch
   Command: git push origin HEAD
   Expected: Changes pushed without force flag
```

## Scenario 3: All Work Blocked

### Failure
```bash
$ sdp-ready
📋 Ready work (0 ready, 3 blocked):
  ○ [P2] sdplab-abc: Feature X (blocked by: sdplab-0)
  ○ [P2] sdplab-def: Feature Y (blocked by: sdplab-0)
  ○ [P2] sdplab-ghi: Feature Z (blocked by: sdplab-0)

Next action:
- Recommendation: Resolve blockers
- Reason: No ready work: 3 issue(s) are blocked
- Command: bd blocked
```

### Recovery Guidance
```bash
$ bd show sdplab-0
○ sdplab-0 · Blocking parent task

$ bd update sdplab-0 --status in_progress
$ # Complete the blocking task...
$ bd close sdplab-0 -r "Completed"

$ sdp-ready
📋 Ready work (3 ready, 0 blocked):
  ● [P2] sdplab-abc: Feature X
  ● [P2] sdplab-def: Feature Y
  ● [P2] sdplab-ghi: Feature Z
```

## Scenario 4: Invalid Configuration

### Failure
```bash
$ sdp up --profile oss-combine
Error: Invalid configuration:
  - beads.prefix is empty
  - github.findings_artifact is missing
```

### Recovery Guidance
```bash
$ sdp-ready --instructions

Instructions:

1. Edit configuration file
   Reason: Required fields are missing from .sdp/config.yaml
   Command: $EDITOR .sdp/config.yaml
   Expected: Configuration file with all required fields

2. Validate configuration
   Reason: Ensure all required fields are present and valid
   Command: sdp up --profile oss-combine --dry-run
   Expected: Configuration validated successfully

3. Proceed with bootstrap
   Reason: Configuration is valid, ready to provision
   Command: sdp up --profile oss-combine
   Expected: .sdp/ directory structure created
```

## JSON Format for Agents

All guidance is available in JSON format:

```bash
$ sdp-ready --format status-view --instructions --action resolve_blockers
```

Output:
```json
{
  "spec_version": "v1.0",
  "context": "Action: resolve_blockers",
  "instructions": [
    {
      "step": 1,
      "action": "Review blocking issues",
      "reason": "3 issue(s) are blocked - identify root causes",
      "command": "bd blocked",
      "expected_outcome": "List of blocked issues and their blockers"
    },
    {
      "step": 2,
      "action": "Resolve blocker: sdplab-0",
      "reason": "This issue blocks other work from proceeding",
      "command": "bd show sdplab-0",
      "expected_outcome": "Blocker details and path to resolution"
    }
  ]
}
```

## Walkthrough: Why Each Step?

Every recommendation includes a reason sourced from known state:

| Step | Why Recommended | Next Gate |
|------|-----------------|-----------|
| Continue in-progress | Work already started, finish first | Evidence collection |
| Start highest priority | Most important ready work | Guard validation |
| Resolve blockers | No work can proceed otherwise | Work becomes ready |
| Install missing tool | Bootstrap cannot complete | Profile validation |
| Fix configuration | Invalid state blocks all flows | Directory provisioning |
