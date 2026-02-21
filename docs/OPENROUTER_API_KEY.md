# OpenRouter API Key

OpenRouter support allows agents to use models like `openrouter/gpt-4o`, `openrouter/claude-sonnet-4` via the OpenRouter API.

## Where to Add the Key

### Option 1: Environment variable (local / probe)

```bash
export OPENROUTER_API_KEY="sk-or-v1-..."
./scripts/run_kubeopencode_multi_role_probe.sh --host user@host ...
```

### Option 2: OpenCode config (local)

Add to `~/.config/opencode/opencode.json`:

```json
{
  "mcp": {
    "zai-mcp-server": {
      "environment": {
        "Z_AI_API_KEY": "...",
        "OPENROUTER_API_KEY": "sk-or-v1-..."
      }
    }
  }
}
```

The probe script reads both keys from this file when env vars are unset.

### Option 3: Kubernetes secret (k8s agents)

Secret `sdp-kubeopencode-credentials` in the kubeopencode namespace. Add key `openrouter_api_key`:

```bash
kubectl -n kubeopencode-system patch secret sdp-kubeopencode-credentials \
  -p '{"stringData":{"openrouter_api_key":"sk-or-v1-..."}}'
```

Or create/update the secret with the probe script (it adds `openrouter_api_key` when `OPENROUTER_API_KEY` is set).

### Option 4: Worker manifests (deploy/k8s/workers)

For `opencode-agent` and similar workers, add to the secret referenced in the deployment and add env:

```yaml
- name: OPENROUTER_API_KEY
  valueFrom:
    secretKeyRef:
      name: sdp-credentials
      key: openrouter_api_key
```

## Allowed OpenRouter models

- `openrouter/gpt-4o`
- `openrouter/gpt-4o-mini`
- `openrouter/claude-sonnet-4`
- `openrouter/claude-3.5-sonnet`

To use: set `model` in Agent config or task label to e.g. `openrouter/gpt-4o`.
