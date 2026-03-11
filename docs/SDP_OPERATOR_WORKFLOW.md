# SDP Operator Workflow

Status: reference
Scope: Beads + orchestrate + quality gates + evidence for operator tasks

## Overview

This document describes the SDP protocol workflow for conducting operator-related work: UP-001 (kubeopencode upstream), O2 AgentRun implementation, and related tasks.

## Workflow Diagram

```mermaid
flowchart TD
    subgraph beads [Beads]
        Ready[bd ready --label autonomy]
        Show[bd show id]
        Claim[bd update id --status in_progress]
        Close[bd close id --reason]
        Export[./scripts/beads_export.sh]
    end

    subgraph orchestrate [Orchestrate]
        Preflight[git fetch/rebase + ./scripts/beads_import_only.sh]
        Dispatch[orchestrate_k8s_issue.sh --host --issue]
        Worker[opencode-agent runs swarm-worker]
        Reviewer[swarm-reviewer]
    end

    subgraph quality [Quality Gates]
        SDP[sdp quality all]
        Tests[go test ./...]
        Lint[golangci-lint]
    end

    subgraph evidence [Evidence]
        FSM[cmd/beads-fsm]
        PRGate[cmd/pr-gate]
    end

    Ready --> Show --> Claim
    Claim --> Preflight --> Dispatch
    Dispatch --> Worker --> SDP --> Reviewer
    Reviewer --> PRGate --> FSM --> Close --> Export
```

## NATS Flow (Swarm Platform)

When the swarm platform is deployed, intake and lifecycle flow through NATS:

```mermaid
flowchart LR
    Intake[POST /api/v1/intake] --> Gateway[intake-gateway]
    Gateway -->|publish| NATS[(NATS JetStream)]
    NATS -->|sdp.intake.{project}| Orchestrator[swarm-orchestrator]
    Orchestrator -->|dispatch| Coder[role-agent-coder]
    Orchestrator -->|dispatch| Analyst[role-agent-analyst]
    Coder -->|sdp.lifecycle.>| NATS
    Analyst -->|sdp.lifecycle.>| NATS
```

- **Subjects:** `sdp.intake.{projectID}` (intake), `sdp.lifecycle.>` (lifecycle events)
- **Stream:** `SDP_INTAKE` (JetStream)
- **KEDA:** Scales coder/analyst agents by consumer lag

## Sequence

1. **Find work:** `bd ready --label autonomy --label workstream:kubeopencode-upstream` (or `workstream:agentrun-operator`)
2. **Get context:** `bd show <id>`
3. **Claim:** `bd update <id> --status in_progress`
4. **Dispatch:** Locally or via `scripts/orchestrate_k8s_issue.sh --host <user@ip> --issue <id>`
5. **In pod:** git fetch/rebase onto `$SDP_REPO_BRANCH`, run `./scripts/beads_import_only.sh`, then swarm-worker executes task
6. **Quality:** `sdp quality all`, `go test ./...`
7. **Evidence:** strict evidence, `cmd/pr-gate`, `cmd/beads-fsm`
8. **Complete:** `bd close <id> --reason "..."`, `./scripts/beads_export.sh`

## Workstreams

| Workstream | Paths | Use case |
|------------|-------|----------|
| `workstream:kubeopencode-upstream` | docs/, specs/, scripts/, internal/adapter/ | UP-001 fixes, adapter changes |
| `workstream:agentrun-operator` | internal/controller/, api/, deploy/k8s/, docs/, specs/, scripts/ | O2 AgentRun CRD, controller, RBAC |

## Orchestrate Script

```bash
scripts/orchestrate_k8s_issue.sh --host <user@ip> --issue <beads-id> [--timeout 300] [--retries 3]
```

Preflight: `git fetch origin "$SDP_REPO_BRANCH"` + `git rebase FETCH_HEAD`, then `./scripts/beads_import_only.sh`, then exec into opencode-agent pod to run swarm-worker.
In `sdp_lab` manifests, `SDP_REPO_BRANCH` defaults to `dev`; public `sdp` work keeps `main`.

## Quality Gates

- `make quality` or `sdp quality all` — SDP plugin checks
- `go test ./...` — unit and integration tests
- `golangci-lint run` — lint

## Evidence

- Strict evidence: `specs/strict-evidence-template.json` sections
- `cmd/pr-gate` — validates trace before PR
- `cmd/beads-fsm` — protocol flow state transitions

## References

- [BEADS_AUTONOMY_SPEC.md](BEADS_AUTONOMY_SPEC.md)
- [BEADS_SDP_REQUIREMENTS.md](BEADS_SDP_REQUIREMENTS.md)
- [K8S_OPERATOR_BACKLOG_PLAN.md](K8S_OPERATOR_BACKLOG_PLAN.md)
- [AGENT_HOOKS_SPEC.md](AGENT_HOOKS_SPEC.md)
- [AGENT_SKILLS_SPEC.md](AGENT_SKILLS_SPEC.md)
- [PROJECT_REGISTRY_SPEC.md](PROJECT_REGISTRY_SPEC.md)
- [docs/UP_001_WORK_PLAN.md](UP_001_WORK_PLAN.md)
