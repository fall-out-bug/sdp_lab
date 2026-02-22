# SDP Kubernetes Secrets

Unified `sdp-credentials` secret across all SDP namespaces. Provisioned by `scripts/provision_secrets.sh`.

## Secret structure

| Key | Description | Required |
|-----|-------------|----------|
| `github_token` | GitHub PAT for repo clone/push | yes |
| `z_ai_api_key` | ZhipuAI API key (Coding Plan) | yes |
| `openrouter_api_key` | OpenRouter API key | no (empty placeholder if unset) |
| `intake_api_key` | API key for intake-gateway (Bearer or X-API-Key) | yes (auto-generated if unset) |
| `registry_api_key` | API key for registry-agent (Bearer or X-API-Key) | yes (auto-generated if unset) |

## Namespace / consumer matrix

| Namespace | Secret | Consumers |
|-----------|--------|-----------|
| `sdp-workers` | `sdp-credentials` | opencode-agent |
| `sdp-control` | `sdp-credentials` | intake-gateway, registry-agent, swarm-orchestrator |
| `kubeopencode-system` | `sdp-credentials` | sdp-analyst, sdp-coder, sdp-reviewer |

## Provisioning

```bash
# All namespaces (default)
./scripts/provision_secrets.sh --host user@host

# Specific namespaces
./scripts/provision_secrets.sh --host user@host --namespaces sdp-workers,kubeopencode-system,sdp-control

# Custom SSH port
./scripts/provision_secrets.sh --host user@host --port 2222
```

Key source priority: env vars (`Z_AI_API_KEY`, `OPENROUTER_API_KEY`, `INTAKE_API_KEY`, `REGISTRY_API_KEY`) > `~/.config/opencode/opencode.json` > fail (or auto-generate for intake/registry).

GitHub token is obtained via `gh auth token` on the remote host.

## Environment variables consumed

| Env var | Secret key | Used by |
|---------|------------|---------|
| `GITHUB_TOKEN` | `github_token` | opencode-agent, orchestrator, agents |
| `GH_TOKEN` | `github_token` | init containers (clone) |
| `Z_AI_API_KEY` | `z_ai_api_key` | opencode-agent, agents |
| `OPENROUTER_API_KEY` | `openrouter_api_key` | opencode-agent, agents |
| `INTAKE_API_KEY` | `intake_api_key` | intake-gateway |
| `REGISTRY_API_KEY` | `registry_api_key` | registry-agent |

## Script integration

- `scripts/apply_worker_manifests.sh` — calls `provision_secrets.sh --namespaces sdp-workers`
- `scripts/run_kubeopencode_multi_role_probe.sh` — calls `provision_secrets.sh --namespaces kubeopencode-system`

For orchestrator deployment, provision `sdp-control`:

```bash
./scripts/provision_secrets.sh --host user@host --namespaces sdp-control
```
