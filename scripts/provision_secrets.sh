#!/usr/bin/env bash
# Provision sdp-credentials secret across SDP k8s namespaces.
# Key source priority: env vars > ~/.config/opencode/opencode.json > fail
set -euo pipefail

HOST=""
PORT="22"
NAMESPACES="sdp-workers,sdp-control,kubeopencode-system"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      HOST="$2"
      shift 2
      ;;
    --port)
      PORT="$2"
      shift 2
      ;;
    --namespaces)
      NAMESPACES="$2"
      shift 2
      ;;
    --help|-h)
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--namespaces <ns1,ns2,...>]"
      echo ""
      echo "Provisions sdp-credentials secret (github_token, z_ai_api_key, openrouter_api_key, intake_api_key, registry_api_key)"
      echo "in each specified namespace. Keys from env or ~/.config/opencode/opencode.json."
      echo "GitHub token obtained via 'gh auth token' on the remote host."
      echo ""
      echo "Default namespaces: sdp-workers,sdp-control,kubeopencode-system"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--namespaces <ns1,ns2,...>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--namespaces <ns1,ns2,...>]"
  exit 2
fi

# Key source: env > opencode.json > generate (for intake/registry)
Z_AI_API_KEY="${Z_AI_API_KEY:-}"
OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-}"
INTAKE_API_KEY="${INTAKE_API_KEY:-}"
REGISTRY_API_KEY="${REGISTRY_API_KEY:-}"

if [[ -z "${Z_AI_API_KEY}" && -f "${HOME}/.config/opencode/opencode.json" ]]; then
  Z_AI_API_KEY="$(python3 - <<'PY'
import json
import os
path = os.path.expanduser('~/.config/opencode/opencode.json')
try:
    with open(path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    env = data.get('mcp', {}).get('zai-mcp-server', {}).get('environment', {})
    print(env.get('Z_AI_API_KEY', ''))
except Exception:
    print('')
PY
)"
fi
if [[ -z "${OPENROUTER_API_KEY}" && -f "${HOME}/.config/opencode/opencode.json" ]]; then
  OPENROUTER_API_KEY="$(python3 - <<'PY'
import json
import os
path = os.path.expanduser('~/.config/opencode/opencode.json')
try:
    with open(path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    env = data.get('mcp', {}).get('zai-mcp-server', {}).get('environment', {})
    print(env.get('OPENROUTER_API_KEY', ''))
except Exception:
    print('')
PY
)"
fi

if [[ -z "${Z_AI_API_KEY}" ]]; then
  echo "Z_AI_API_KEY is required (env or ~/.config/opencode/opencode.json)"
  exit 2
fi

# intake_api_key and registry_api_key: env > opencode.json > generate random
if [[ -z "${INTAKE_API_KEY}" && -f "${HOME}/.config/opencode/opencode.json" ]]; then
  INTAKE_API_KEY="$(python3 -c "
import json, os
p = os.path.expanduser('~/.config/opencode/opencode.json')
try:
    with open(p) as f: d = json.load(f)
    env = d.get('mcp', {}).get('zai-mcp-server', {}).get('environment', {})
    print(env.get('INTAKE_API_KEY', ''))
except Exception: print('')
")"
fi
if [[ -z "${REGISTRY_API_KEY}" && -f "${HOME}/.config/opencode/opencode.json" ]]; then
  REGISTRY_API_KEY="$(python3 -c "
import json, os
p = os.path.expanduser('~/.config/opencode/opencode.json')
try:
    with open(p) as f: d = json.load(f)
    env = d.get('mcp', {}).get('zai-mcp-server', {}).get('environment', {})
    print(env.get('REGISTRY_API_KEY', ''))
except Exception: print('')
")"
fi
[[ -z "${INTAKE_API_KEY}" ]] && INTAKE_API_KEY="$(openssl rand -hex 32)"
[[ -z "${REGISTRY_API_KEY}" ]] && REGISTRY_API_KEY="$(openssl rand -hex 32)"

Z_AI_API_KEY_B64="$(printf '%s' "${Z_AI_API_KEY}" | base64 | tr -d '\n')"
OPENROUTER_API_KEY_B64="$(printf '%s' "${OPENROUTER_API_KEY:-}" | base64 | tr -d '\n')"
INTAKE_API_KEY_B64="$(printf '%s' "${INTAKE_API_KEY}" | base64 | tr -d '\n')"
REGISTRY_API_KEY_B64="$(printf '%s' "${REGISTRY_API_KEY}" | base64 | tr -d '\n')"

echo "[provision] creating sdp-credentials in namespaces: ${NAMESPACES}"
ssh -p "${PORT}" "${HOST}" "Z_AI_API_KEY_B64='${Z_AI_API_KEY_B64}' OPENROUTER_API_KEY_B64='${OPENROUTER_API_KEY_B64}' INTAKE_API_KEY_B64='${INTAKE_API_KEY_B64}' REGISTRY_API_KEY_B64='${REGISTRY_API_KEY_B64}' NAMESPACES='${NAMESPACES}' bash -s" <<'EOF'
set -euo pipefail
GH_TOKEN="$(gh auth token)"
Z_AI_API_KEY="$(printf '%s' "${Z_AI_API_KEY_B64}" | base64 -d)"
OPENROUTER_VAL=""
[[ -n "${OPENROUTER_API_KEY_B64}" ]] && OPENROUTER_VAL="$(printf '%s' "${OPENROUTER_API_KEY_B64}" | base64 -d)"
INTAKE_API_KEY="$(printf '%s' "${INTAKE_API_KEY_B64}" | base64 -d)"
REGISTRY_API_KEY="$(printf '%s' "${REGISTRY_API_KEY_B64}" | base64 -d)"

IFS=',' read -ra NSS <<< "${NAMESPACES}"
for ns in "${NSS[@]}"; do
  ns="$(echo "${ns}" | tr -d ' ')"
  [[ -z "${ns}" ]] && continue
  echo "[provision] applying sdp-credentials in ${ns}"
  kubectl -n "${ns}" create secret generic sdp-credentials \
    --from-literal=github_token="${GH_TOKEN}" \
    --from-literal=z_ai_api_key="${Z_AI_API_KEY}" \
    --from-literal=openrouter_api_key="${OPENROUTER_VAL}" \
    --from-literal=intake_api_key="${INTAKE_API_KEY}" \
    --from-literal=registry_api_key="${REGISTRY_API_KEY}" \
    --dry-run=client -o yaml | kubectl apply -f -
done
EOF

echo "[provision] done"
