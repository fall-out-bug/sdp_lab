#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
TITLE=""
DESCRIPTION=""
WORKSTREAM="policy-k8s-risk-high"
PARENT=""
PRIORITY="1"

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
    --title)
      TITLE="$2"
      shift 2
      ;;
    --description)
      DESCRIPTION="$2"
      shift 2
      ;;
    --workstream)
      WORKSTREAM="$2"
      shift 2
      ;;
    --parent)
      PARENT="$2"
      shift 2
      ;;
    --priority)
      PRIORITY="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> --title <title> [--description <text>] [--workstream <model-chain-default-fallback|policy-k8s-risk-high>] [--parent <issue-id>] [--priority <0-4>] [--port <port>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" || -z "${TITLE}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> --title <title> [--description <text>] [--workstream <model-chain-default-fallback|policy-k8s-risk-high>] [--parent <issue-id>] [--priority <0-4>] [--port <port>]"
  exit 2
fi

case "${WORKSTREAM}" in
  model-chain-default-fallback)
    SPEC_ID="docs/MODEL_POLICY.md"
    MODEL_LABEL="model:glm-4.7"
    ;;
  policy-k8s-risk-high)
    SPEC_ID="docs/RISK_POLICY.md"
    MODEL_LABEL="model:glm-5"
    ;;
  *)
    echo "Unsupported workstream: ${WORKSTREAM}"
    echo "Supported: model-chain-default-fallback, policy-k8s-risk-high"
    exit 2
    ;;
esac

if [[ -z "${DESCRIPTION}" ]]; then
  DESCRIPTION="Auto-created feature task from /feature workflow."
fi

extract_issue_id() {
  local payload="$1"
  PAYLOAD="${payload}" python3 - <<'PY'
import json
import os

payload = os.environ.get("PAYLOAD", "")
start = -1
for i, ch in enumerate(payload):
    if ch in "[{":
        start = i
        break
if start == -1:
    raise SystemExit("missing json payload")
obj = json.loads(payload[start:])
if isinstance(obj, list):
    obj = obj[0] if obj else {}
issue_id = obj.get("id", "")
if not issue_id:
    raise SystemExit("missing issue id")
print(issue_id)
PY
}

remote_create_issue() {
  local host="$1"
  local port="$2"
  local title="$3"
  local description="$4"
  local parent="$5"
  local priority="$6"
  local spec_id="$7"
  local workstream="$8"
  local model_label="$9"

  local title_b64 description_b64 spec_b64 parent_b64 workstream_b64 model_b64
  title_b64="$(printf '%s' "${title}" | base64 | tr -d '\n')"
  description_b64="$(printf '%s' "${description}" | base64 | tr -d '\n')"
  spec_b64="$(printf '%s' "${spec_id}" | base64 | tr -d '\n')"
  parent_b64="$(printf '%s' "${parent}" | base64 | tr -d '\n')"
  workstream_b64="$(printf '%s' "${workstream}" | base64 | tr -d '\n')"
  model_b64="$(printf '%s' "${model_label}" | base64 | tr -d '\n')"

  ssh -p "${port}" "${host}" "TITLE_B64='${title_b64}' DESC_B64='${description_b64}' SPEC_B64='${spec_b64}' PARENT_B64='${parent_b64}' WORKSTREAM_B64='${workstream_b64}' MODEL_B64='${model_b64}' PRIORITY='${priority}' bash -s" <<'EOF'
set -euo pipefail
title="$(printf '%s' "${TITLE_B64}" | base64 -d)"
description="$(printf '%s' "${DESC_B64}" | base64 -d)"
spec_id="$(printf '%s' "${SPEC_B64}" | base64 -d)"
parent="$(printf '%s' "${PARENT_B64}" | base64 -d)"
workstream="$(printf '%s' "${WORKSTREAM_B64}" | base64 -d)"
model_label="$(printf '%s' "${MODEL_B64}" | base64 -d)"
labels="autonomy,strict-evidence,${model_label},lane:commit,workstream:${workstream}"
bd_args=("$title" -t task -p "$PRIORITY")
[[ -n "${parent}" ]] && bd_args+=(--parent "$parent")
bd_args+=(--spec-id "$spec_id" --description "$description" --labels "$labels" --json)
bd_cmd="bd create $(printf ' %q' "${bd_args[@]}")"
kubectl -n sdp-workers exec deploy/opencode-agent -- bash -lc "cd /workspace && bd sync --import-only >/dev/null && ${bd_cmd}"
EOF
}

remote_show_issue() {
  local host="$1"
  local port="$2"
  local issue_id="$3"
  ssh -p "${port}" "${host}" "kubectl -n sdp-workers exec deploy/opencode-agent -- sh -lc 'cd /workspace && bd show ${issue_id} --json'"
}

extract_pr_url() {
  local payload="$1"
  PAYLOAD="${payload}" python3 - <<'PY'
import json
import os
import re

payload = os.environ.get("PAYLOAD", "")
start = -1
for i, ch in enumerate(payload):
    if ch in "[{":
        start = i
        break
if start == -1:
    print("")
    raise SystemExit(0)
obj = json.loads(payload[start:])
if isinstance(obj, list):
    obj = obj[0] if obj else {}
notes = obj.get("notes", "")
match = re.search(r"https://github.com/[^\s]+/pull/\d+", notes)
print(match.group(0) if match else "")
PY
}

echo "[feature] creating task in beads"
create_out="$(remote_create_issue "${HOST}" "${PORT}" "${TITLE}" "${DESCRIPTION}" "${PARENT}" "${PRIORITY}" "${SPEC_ID}" "${WORKSTREAM}" "${MODEL_LABEL}")"
issue_id="$(extract_issue_id "${create_out}")"
echo "[feature] issue=${issue_id}"

echo "[feature] orchestrating worker/reviewer run"
"$(dirname "$0")/orchestrate_k8s_issue.sh" --host "${HOST}" --port "${PORT}" --issue "${issue_id}" --timeout 600 --poll 8

final_out="$(remote_show_issue "${HOST}" "${PORT}" "${issue_id}")"
pr_url="$(extract_pr_url "${final_out}")"

if [[ -z "${pr_url}" ]]; then
  echo "[feature] completed issue=${issue_id} (no PR URL found in notes)"
  exit 1
fi

echo "[feature] issue=${issue_id}"
echo "[feature] pr=${pr_url}"
