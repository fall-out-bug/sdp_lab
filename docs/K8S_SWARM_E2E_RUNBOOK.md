# K8s Swarm E2E Runbook

Status: implemented (2026-02-21), updated (2026-02-22)

## Goal

Verify the full E2E flow: NATS -> Bridge -> swarm-orchestrator (or feature-orchestrator) -> K8s exec / AgentRun -> opencode-agent -> swarm-worker -> swarm-reviewer -> PR.

## Prerequisites

- K8s cluster with control plane (NATS, intake-gateway, swarm-orchestrator) and workers (opencode-agent) deployed
- `sdp-credentials` secret with `github_token`, `z_ai_api_key`, `openrouter_api_key`
- Project workspace cloned in swarm-workspaces PVC or opencode-agent init container
- Beads initialized in workspace (`.beads/issues.jsonl`)

## Deployment Steps

1. **Apply control plane** (includes swarm-orchestrator with ServiceAccount):

   ```bash
   kubectl kustomize deploy/k8s/control/ | kubectl apply -f -
   ```

2. **Apply workers** (includes opencode-agent and swarm-orchestrator RBAC in sdp-workers):

   ```bash
   kubectl kustomize deploy/k8s/workers/ | kubectl apply -f -
   ```

3. **Provision secrets** (if not already done):

   ```bash
   ./scripts/provision_secrets.sh --host user@cluster --namespaces sdp-control,sdp-workers
   ```

4. **Build and load swarm-orchestrator image** (if using local build):

   ```bash
   docker build -t sdp/swarm-orchestrator:latest -f deploy/images/swarm-orchestrator/Dockerfile .
   # For minikube: minikube image load sdp/swarm-orchestrator:latest
   ```

## Verification Steps

1. **Check pods**:

   ```bash
   kubectl -n sdp-control get pods -l app=swarm-orchestrator
   kubectl -n sdp-workers get pods -l app=opencode-agent
   ```

2. **Create a test Beads issue** (in the workspace that Bridge polls):

   - Bridge polls workspaces from `WORKSPACE_BASE` (default `/workspaces`)
   - For `repo_url: "."`, workspace is `/workspaces/<project_id>`
   - Create issue with labels: `autonomy`, `strict-evidence`, `workstream:builder`

   Example (when exec'd into a pod with bd):

   ```bash
   bd create --title "Test swarm E2E" --label autonomy --label strict-evidence --label workstream:builder
   ```

3. **Observe flow**:

   - **Path A (swarm-orchestrator):** Bridge polls Beads every 5s, publishes `sdp.beads.<project>.ready`; swarm-orchestrator receives event, calls `dispatchK8s`; K8s exec: preflight (git sync, bd sync) -> claim (bd update in_progress) -> opencode-agent with `SDP_ISSUE`.
   - **Path B (feature-orchestrator):** Aggregator subscribes to `sdp.beads.*.ready`, maintains priority queue; feature-orchestrator poll loop creates AgentRun CRDs; adapter-controller reconciles AgentRun -> Tasks -> opencode-agent.
   - opencode-agent runs swarm-worker `--issue <id>`, then swarm-reviewer. Issue transitions to closed, PR created.

4. **Check logs**:

   ```bash
   kubectl -n sdp-control logs -l app=swarm-orchestrator -f
   kubectl -n sdp-workers logs -l app=opencode-agent -f
   ```

## Environment Variables

### swarm-orchestrator

| Variable | Default | Description |
|----------|---------|-------------|
| SDP_DISPATCH_MODE | local | Set to `k8s` for K8s exec dispatch |
| SDP_K8S_NAMESPACE | sdp-workers | Namespace for opencode-agent pods |
| NATS_URL | — | NATS server URL (required) |
| WORKSPACE_BASE | /workspaces | Base dir for project workspaces |

### opencode-agent / workers

| Variable | Description |
|----------|-------------|
| SDP_ISSUE | Beads issue ID for the current run (e.g. `sdp_dev-4pg`). Set by swarm-orchestrator or adapter when invoking the agent. |
| Model / GLM | Default model is from policy (e.g. `glm-4.7`, `glm-5`). Override via issue labels `model:<id>`; feature-orchestrator enforces policy allowlist. |

## Troubleshooting

- **"no opencode-agent pod found"**: Ensure opencode-agent deployment is running in sdp-workers
- **"in-cluster config" error**: swarm-orchestrator must run inside the cluster (not locally) for K8s mode
- **RBAC "forbidden"**: Verify swarm-orchestrator Role and RoleBinding exist in sdp-workers
- **Bridge not publishing ready**: Check workspace has Beads initialized and issue has correct labels
