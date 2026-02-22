# Feature: AgentRun CRD + Controller (FR-002)

Priority: P0
Effort: 5-7 days
Depends on: FR-001, FR-003

## Problem

No CRD for orchestrating multi-role flow. Current multi-role probe is a script, not a K8s-native resource. AgentRun is needed to manage lifecycle: worker Tasks → reviewer Task → terminal.

## Scope

### CRD Schema

```yaml
apiVersion: sdp.dev/v1alpha1
kind: AgentRun
metadata:
  name: run-<issue-id>-<attempt>
  labels:
    beads.issue: <issue-id>
    sdp.project: <project-id>
spec:
  issueId: sdp_dev-xxx
  repo: .
  baseBranch: main
  model: glm-4.7
  workstream: builder
  timeoutSec: 1200
status:
  phase: Pending|Running|Succeeded|Failed
  conditions: [...]
  workerTask: run-xxx-worker
  reviewerTask: run-xxx-reviewer
  prUrl: https://github.com/.../pull/N
  lastError: ""
```

### Controller

1. Validate AgentRun spec
2. Create analyst+coder Tasks (parallel)
3. Wait for both to reach terminal
4. Collect outputs (task logs → ConfigMap)
5. Create reviewer Task with aggregated context
6. On reviewer terminal → publish AgentRun status
7. Update Beads issue via adapter

### RBAC

- ServiceAccount for controller
- Role: create/get/list/watch Tasks, AgentRuns
- RoleBinding in kubeopencode-system

## Acceptance Criteria

- `kubectl apply -f agentrun.yaml` → controller creates worker Tasks
- analyst+coder start in parallel
- reviewer starts after analyst+coder reach terminal
- AgentRun status reflects final result
- Beads issue transitions to done/blocked
