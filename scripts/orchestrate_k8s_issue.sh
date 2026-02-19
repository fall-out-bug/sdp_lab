#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
ISSUE=""
TIMEOUT="300"
POLL="10"

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
    --issue)
      ISSUE="$2"
      shift 2
      ;;
    --timeout)
      TIMEOUT="$2"
      shift 2
      ;;
    --poll)
      POLL="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> --issue <id> [--port <port>] [--timeout <seconds>] [--poll <seconds>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" || -z "${ISSUE}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> --issue <id> [--port <port>] [--timeout <seconds>] [--poll <seconds>]"
  exit 2
fi

remote() {
  ssh -p "${PORT}" "${HOST}" "$@"
}

extract_field() {
  local payload="$1"
  local field="$2"
  PAYLOAD="${payload}" FIELD="${field}" python3 - <<'PY'
import json
import os

payload = os.environ.get("PAYLOAD", "")
field = os.environ.get("FIELD", "")
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
value = obj.get(field, "")
if isinstance(value, (dict, list)):
    print(json.dumps(value))
else:
    print(str(value))
PY
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

echo "[orchestrate] waiting for opencode-agent deployment"
remote "kubectl -n sdp-workers rollout status deployment/opencode-agent --timeout=240s >/dev/null"

echo "[orchestrate] running preflight sync inside pod"
remote "kubectl -n sdp-workers exec deploy/opencode-agent -- sh -lc 'cd /workspace && git rev-parse --is-inside-work-tree >/dev/null && bd sync --import-only >/dev/null'"

echo "[orchestrate] triggering one explicit agent cycle"
remote "kubectl -n sdp-workers exec deploy/opencode-agent -- opencode-agent" || true

deadline=$(( $(date +%s) + TIMEOUT ))
last_status=""

while true; do
  now=$(date +%s)
  if [[ "${now}" -ge "${deadline}" ]]; then
    echo "[orchestrate] timeout waiting for issue ${ISSUE}"
    remote "kubectl -n sdp-workers logs deploy/opencode-agent --tail=120"
    exit 1
  fi

  raw="$(remote "kubectl -n sdp-workers exec deploy/opencode-agent -- sh -lc 'cd /workspace && bd show ${ISSUE} --json'" 2>/dev/null || true)"
  status="$(extract_field "${raw}" "status")"
  pr_url="$(extract_pr_url "${raw}")"

  if [[ "${status}" != "${last_status}" ]]; then
    echo "[orchestrate] issue=${ISSUE} status=${status}"
    last_status="${status}"
  fi

  if [[ "${status}" == "closed" ]]; then
    echo "[orchestrate] completed issue ${ISSUE}"
    if [[ -n "${pr_url}" ]]; then
      echo "[orchestrate] pr=${pr_url}"
    fi
    exit 0
  fi

  if [[ "${status}" == "blocked" ]]; then
    echo "[orchestrate] issue ${ISSUE} blocked"
    remote "kubectl -n sdp-workers logs deploy/opencode-agent --tail=120"
    exit 1
  fi

  sleep "${POLL}"
done
