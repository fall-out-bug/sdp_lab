#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
IMAGE=""
BRANCH=""

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
    --image)
      IMAGE="$2"
      shift 2
      ;;
    --branch)
      BRANCH="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--image <image>] [--branch <branch>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--image <image>] [--branch <branch>]"
  exit 2
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFEST_DIR="${ROOT_DIR}/deploy/k8s/workers"

Z_AI_API_KEY="${Z_AI_API_KEY:-}"
OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-}"
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

echo "[apply] copying worker manifests to ${HOST}:${PORT}"
ssh -p "${PORT}" "${HOST}" "mkdir -p /tmp/sdp-dev-workers"
if [[ -n "${BRANCH}" ]]; then
  echo "[apply] patching SDP_REPO_BRANCH to ${BRANCH}"
  TMP_MANIFEST="$(mktemp -d)"
  trap "rm -rf '${TMP_MANIFEST}'" EXIT
  cp -r "${MANIFEST_DIR}/." "${TMP_MANIFEST}/"
  if sed --version >/dev/null 2>&1; then
    sed -i "s|value: feat/sdp_dev-[^[:space:]]*|value: ${BRANCH}|g" "${TMP_MANIFEST}/opencode-agent.yaml"
  else
    sed -i '' "s|value: feat/sdp_dev-[^[:space:]]*|value: ${BRANCH}|g" "${TMP_MANIFEST}/opencode-agent.yaml"
  fi
  scp -P "${PORT}" -r "${TMP_MANIFEST}/." "${HOST}:/tmp/sdp-dev-workers/"
else
  scp -P "${PORT}" -r "${MANIFEST_DIR}/." "${HOST}:/tmp/sdp-dev-workers/"
fi

echo "[apply] ensuring worker credentials secret"
Z_AI_API_KEY_B64="$(printf '%s' "${Z_AI_API_KEY}" | base64 | tr -d '\n')"
OPENROUTER_API_KEY_B64="$(printf '%s' "${OPENROUTER_API_KEY:-}" | base64 | tr -d '\n')"
ssh -p "${PORT}" "${HOST}" "Z_AI_API_KEY_B64='${Z_AI_API_KEY_B64}' OPENROUTER_API_KEY_B64='${OPENROUTER_API_KEY_B64}' bash -s" <<'EOF'
set -euo pipefail
GH_TOKEN="$(gh auth token)"
Z_AI_API_KEY="$(printf '%s' "${Z_AI_API_KEY_B64}" | base64 -d)"
OPENROUTER_VAL=""
[[ -n "${OPENROUTER_API_KEY_B64}" ]] && OPENROUTER_VAL="$(printf '%s' "${OPENROUTER_API_KEY_B64}" | base64 -d)"
kubectl -n sdp-workers create secret generic sdp-agent-credentials \
  --from-literal=github_token="${GH_TOKEN}" \
  --from-literal=z_ai_api_key="${Z_AI_API_KEY}" \
  --from-literal=openrouter_api_key="${OPENROUTER_VAL}" \
  --dry-run=client -o yaml | kubectl apply -f -
EOF

echo "[apply] applying worker kustomization on remote"
ssh -p "${PORT}" "${HOST}" "kubectl apply -k /tmp/sdp-dev-workers"

if [[ -n "${IMAGE}" ]]; then
  echo "[apply] pinning opencode-agent image ${IMAGE}"
  ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-workers set image deployment/opencode-agent opencode-agent=${IMAGE} init-workspace=${IMAGE}"
fi

ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-workers get deploy,pod"
