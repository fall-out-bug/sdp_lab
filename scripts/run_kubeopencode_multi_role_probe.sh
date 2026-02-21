#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
NAMESPACE="kubeopencode-system"
RUN_ID="run-$(date +%Y%m%d-%H%M%S)"
TASK_TIMEOUT="600"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

wait_task_terminal() {
  local task_name="$1"
  local timeout_seconds="${2:-${TASK_TIMEOUT}}"
  local start
  start="$(date +%s)"
  while true; do
    local phase
    phase="$(ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} get task ${task_name} -o jsonpath='{.status.phase}' 2>/dev/null" || echo "Unknown")"
    if [[ "${phase}" == "Completed" ]]; then
      return 0
    fi
    if [[ "${phase}" == "Failed" ]]; then
      echo "[probe] task ${task_name} failed"
      ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} describe task ${task_name}"
      return 1
    fi
    if (( $(date +%s) - start > timeout_seconds )); then
      echo "[probe] timeout (${timeout_seconds}s) waiting for ${task_name}; deleting stuck task"
      ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} delete task ${task_name} --wait=false 2>/dev/null" || true
      ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} describe task ${task_name} 2>/dev/null" || true
      return 1
    fi
    sleep 5
  done
}

fetch_task_log() {
  local task_name="$1"
  ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} logs \$(kubectl -n ${NAMESPACE} get task ${task_name} -o jsonpath='{.status.podName}') --tail=200"
}

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
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --run-id)
      RUN_ID="$2"
      shift 2
      ;;
    --timeout)
      TASK_TIMEOUT="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--namespace <ns>] [--run-id <id>] [--timeout <seconds>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--namespace <ns>] [--run-id <id>] [--timeout <seconds>]"
  exit 2
fi

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

if [[ -z "${Z_AI_API_KEY}" ]]; then
  echo "Z_AI_API_KEY is required (env or ~/.config/opencode/opencode.json)"
  exit 2
fi

echo "[probe] ensure kubeopencode installed"
if ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} get deploy -l app.kubernetes.io/name=kubeopencode >/dev/null 2>&1"; then
  echo "[probe] kubeopencode deployment already present, skipping install"
else
  "${ROOT_DIR}/scripts/install_kubeopencode_remote.sh" --host "${HOST}" --port "${PORT}" --namespace "${NAMESPACE}"
fi

echo "[probe] create credentials secret"
Z_AI_API_KEY_B64="$(printf '%s' "${Z_AI_API_KEY}" | base64 | tr -d '\n')"
OPENROUTER_API_KEY_B64="$(printf '%s' "${OPENROUTER_API_KEY:-}" | base64 | tr -d '\n')"
ssh -p "${PORT}" "${HOST}" "NAMESPACE='${NAMESPACE}' Z_AI_API_KEY_B64='${Z_AI_API_KEY_B64}' OPENROUTER_API_KEY_B64='${OPENROUTER_API_KEY_B64}' bash -s" <<'EOF'
set -euo pipefail
GH_TOKEN="$(gh auth token)"
Z_AI_API_KEY="$(printf '%s' "${Z_AI_API_KEY_B64}" | base64 -d)"
# openrouter_api_key: required by agents.yaml; use placeholder when OPENROUTER_API_KEY not set
if [[ -n "${OPENROUTER_API_KEY_B64}" ]]; then
  OPENROUTER_VAL="$(printf '%s' "${OPENROUTER_API_KEY_B64}" | base64 -d)"
else
  OPENROUTER_VAL=""
fi
kubectl -n "${NAMESPACE}" create secret generic sdp-kubeopencode-credentials \
  --from-literal=github_token="${GH_TOKEN}" \
  --from-literal=z_ai_api_key="${Z_AI_API_KEY}" \
  --from-literal=openrouter_api_key="${OPENROUTER_VAL}" \
  --dry-run=client -o yaml | kubectl apply -f -
EOF

echo "[probe] apply multi-role agent definitions"
ssh -p "${PORT}" "${HOST}" "mkdir -p /tmp/sdp-kubeopencode"
scp -P "${PORT}" -r "${ROOT_DIR}/deploy/k8s/kubeopencode/." "${HOST}:/tmp/sdp-kubeopencode/"
ssh -p "${PORT}" "${HOST}" "kubectl apply -k /tmp/sdp-kubeopencode"

ANALYST_TASK="${RUN_ID}-analyst"
CODER_TASK="${RUN_ID}-coder"
REVIEWER_TASK="${RUN_ID}-reviewer"

echo "[probe] spawn analyst and coder tasks in parallel"
ssh -p "${PORT}" "${HOST}" "NAMESPACE='${NAMESPACE}' ANALYST_TASK='${ANALYST_TASK}' CODER_TASK='${CODER_TASK}' RUN_ID='${RUN_ID}' bash -s" <<'EOF'
set -euo pipefail
cat <<YAML | kubectl apply -f -
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: ${ANALYST_TASK}
  namespace: ${NAMESPACE}
spec:
  agentRef:
    name: sdp-analyst
  description: |
    run_id=${RUN_ID}
    Analyze requirements for orchestrating multi-role SDP agents with operator execution.
    Provide risks, assumptions, and a concise step plan.
---
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: ${CODER_TASK}
  namespace: ${NAMESPACE}
spec:
  agentRef:
    name: sdp-coder
  description: |
    run_id=${RUN_ID}
    Propose implementation details for a multi-role orchestrator adapter.
    Include interfaces, data flow, and failure handling.
YAML
EOF

echo "[probe] wait for analyst/coder completion (timeout=${TASK_TIMEOUT}s)"
wait_task_terminal "${ANALYST_TASK}" "${TASK_TIMEOUT}"
wait_task_terminal "${CODER_TASK}" "${TASK_TIMEOUT}"

echo "[probe] capture analyst/coder logs into run artifact configmap"
ANALYST_LOG="$(fetch_task_log "${ANALYST_TASK}")"
CODER_LOG="$(fetch_task_log "${CODER_TASK}")"
ANALYST_B64="$(printf '%s' "${ANALYST_LOG}" | base64 | tr -d '\n')"
CODER_B64="$(printf '%s' "${CODER_LOG}" | base64 | tr -d '\n')"
ARTIFACTS_CM="${RUN_ID}-artifacts"
ssh -p "${PORT}" "${HOST}" "NAMESPACE='${NAMESPACE}' ARTIFACTS_CM='${ARTIFACTS_CM}' ANALYST_B64='${ANALYST_B64}' CODER_B64='${CODER_B64}' bash -s" <<'EOF'
set -euo pipefail
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
printf '%s' "${ANALYST_B64}" | base64 -d > "${tmpdir}/analyst.log"
printf '%s' "${CODER_B64}" | base64 -d > "${tmpdir}/coder.log"
kubectl -n "${NAMESPACE}" create configmap "${ARTIFACTS_CM}" \
  --from-file=analyst.log="${tmpdir}/analyst.log" \
  --from-file=coder.log="${tmpdir}/coder.log" \
  --dry-run=client -o yaml | kubectl apply -f -
EOF

echo "[probe] spawn reviewer with aggregated context"
ssh -p "${PORT}" "${HOST}" "NAMESPACE='${NAMESPACE}' REVIEWER_TASK='${REVIEWER_TASK}' RUN_ID='${RUN_ID}' ARTIFACTS_CM='${ARTIFACTS_CM}' bash -s" <<'EOF'
set -euo pipefail
cat <<YAML | kubectl apply -f -
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: ${REVIEWER_TASK}
  namespace: ${NAMESPACE}
spec:
  agentRef:
    name: sdp-reviewer
  contexts:
    - type: ConfigMap
      configMap:
        name: ${ARTIFACTS_CM}
      mountPath: role-artifacts
  description: |
    run_id=${RUN_ID}
    Review analyst and coder outputs and return verdict (approve|needs_changes).
    Use role artifacts from files:
    - /workspace/role-artifacts/analyst.log
    - /workspace/role-artifacts/coder.log
    Produce synthesis, risks, and a final JSON envelope.
YAML
EOF

echo "[probe] wait for reviewer completion (timeout=${TASK_TIMEOUT}s)"
wait_task_terminal "${REVIEWER_TASK}" "${TASK_TIMEOUT}"

echo "[probe] run summary"
ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} get task/${ANALYST_TASK} task/${CODER_TASK} task/${REVIEWER_TASK} -o wide"
ANALYST_LOG="$(fetch_task_log "${ANALYST_TASK}")"
CODER_LOG="$(fetch_task_log "${CODER_TASK}")"
REVIEW_LOG="$(fetch_task_log "${REVIEWER_TASK}")"
printf '%s\n' "${REVIEW_LOG}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT
printf '%s\n' "${ANALYST_LOG}" > "${TMP_DIR}/analyst.log"
printf '%s\n' "${CODER_LOG}" > "${TMP_DIR}/coder.log"
printf '%s\n' "${REVIEW_LOG}" > "${TMP_DIR}/reviewer.log"

echo "[probe] applying SDP operator gate"
go run ./cmd/operator-gate --run-id "${RUN_ID}" --analyst-log "${TMP_DIR}/analyst.log" --coder-log "${TMP_DIR}/coder.log" --reviewer-log "${TMP_DIR}/reviewer.log"

echo "[probe] completed run_id=${RUN_ID}"
