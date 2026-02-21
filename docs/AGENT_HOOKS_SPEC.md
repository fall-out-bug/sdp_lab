# Agent Hooks Specification

Status: reference  
Source: `specs/agent-hooks.yaml`

## Overview

Agent hooks run at lifecycle points per role. They enforce boundaries, clean workspaces, run tests, and route feedback.

## Lifecycle Points

| Point | When | Typical use |
|-------|------|-------------|
| `pre_execute` | Before task execution | boundary-check, workspace-clean |
| `post_execute` | After task execution | boundary-revalidate, go-test |
| `pre_publish` | Before publishing artifacts | evidence-finalize |
| `post_review` | After review consensus | feedback-route |

## Built-in Hooks

- **boundary-check** — Validates task stays within allowed paths (workstream)
- **workspace-clean** — Ensures clean workspace before execution
- **boundary-revalidate** — Re-checks boundaries after execution
- **go-test** — Runs `go test ./...` (coder role)
- **evidence-finalize** — Finalizes evidence before publish
- **feedback-route** — Routes review feedback to intake/NATS

## Role Configuration

See `specs/agent-hooks.yaml` for per-role hook lists. Custom hooks can be registered via `HookRegistry.Register()`.
