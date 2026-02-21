# OpenRouter API Key

OpenRouter is the API gateway. Model IDs use `provider/model` format (e.g. `openai/gpt-4o`, `anthropic/claude-sonnet-4.6`). See https://openrouter.ai/models for current models.

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

## Allowed models (via OpenRouter API)

- `openai/gpt-5.2-codex`
- `anthropic/claude-sonnet-4.6`, `anthropic/claude-opus-4.6`
- `minimax/minimax-m2.5`
- `moonshotai/kimi-k2.5`

GLM is used via `zhipuai-coding-plan/glm-5` (coding plan), not OpenRouter.

To use: set `model` in Agent config or task label to e.g. `openai/gpt-5.2-codex`.
