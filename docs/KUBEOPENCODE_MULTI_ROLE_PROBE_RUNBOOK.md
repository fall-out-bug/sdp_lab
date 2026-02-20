# KubeOpenCode Multi-Role Probe Runbook

Status: prototype (provider-mapped + gated)

## Goal

Validate operator-based spawning of multiple roles (`analyst`, `coder`, `reviewer`) and mediator-style communication via orchestrator context.

## Commands

Install/upgrade operator:

```bash
./scripts/install_kubeopencode_remote.sh --host fall_out_bug@192.168.50.219 --port 2222
```

Run probe:

```bash
./scripts/run_kubeopencode_multi_role_probe.sh --host fall_out_bug@192.168.50.219 --port 2222 --run-id run-roles-02
```

## What probe verifies

- kubeopencode operator is deployed and healthy
- role-specific Agent resources exist
- analyst/coder Tasks are spawned in parallel
- reviewer Task is spawned after prior roles complete
- task phases and logs are collected for trace

## Provider mapping used

- Agent model routing is split by responsibility:
  - `analyst` and `coder`: `zhipuai-coding-plan/glm-4.7`
  - `reviewer`: `zhipuai-coding-plan/glm-5`
- Credentials include `ZHIPU_API_KEY` (plus `ZAI_API_KEY`/`Z_AI_API_KEY` aliases).
- This avoids the prior `Model not found: zai/glm-5` resolution failure.

## SDP gate behavior

- Probe now runs `cmd/operator-gate` over role logs.
- Run is successful only when all role envelopes pass semantic checks.
- `Task.phase=Completed` alone is not considered success.

## Communication model in this prototype

- role pods do not communicate directly
- orchestrator captures analyst/coder logs and writes a run-scoped ConfigMap
- reviewer consumes `/workspace/role-artifacts/analyst.log` and `/workspace/role-artifacts/coder.log`
- orchestrator remains central mediator and audit point
