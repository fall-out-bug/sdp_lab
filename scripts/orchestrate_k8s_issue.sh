#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
ISSUE=""
TIMEOUT="300"
POLL="10"
RETRIES="3"
RETRY_DELAY="5"
RUN_ID=""
RUN_FILE=""

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
    --retries)
      RETRIES="$2"
      shift 2
      ;;
    --retry-delay)
      RETRY_DELAY="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> --issue <id> [--port <port>] [--timeout <seconds>] [--poll <seconds>] [--retries <count>] [--retry-delay <seconds>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" || -z "${ISSUE}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> --issue <id> [--port <port>] [--timeout <seconds>] [--poll <seconds>] [--retries <count>] [--retry-delay <seconds>]"
  exit 2
fi

remote() {
  ssh -p "${PORT}" "${HOST}" "$@"
}

append_run_phase() {
  local phase="$1"
  local state="$2"
  local message="$3"
  local pr_url="${4:-}"
  if [[ -z "${RUN_FILE}" ]]; then
    return
  fi
  RUN_FILE="${RUN_FILE}" PHASE="${phase}" STATE="${state}" MESSAGE="${message}" PR_URL="${pr_url}" python3 - <<'PY'
import json
import os
from datetime import datetime, timezone

path = os.environ["RUN_FILE"]
phase = os.environ["PHASE"]
state = os.environ["STATE"]
message = os.environ["MESSAGE"]
pr_url = os.environ.get("PR_URL", "")

with open(path, "r", encoding="utf-8") as f:
    doc = json.load(f)

event = {
    "at": datetime.now(timezone.utc).isoformat(),
    "phase": phase,
    "state": state,
    "message": message,
}
if pr_url:
    event["pr_url"] = pr_url

doc.setdefault("events", []).append(event)
doc["last_phase"] = phase
doc["last_state"] = state
if pr_url:
    doc["pr_url"] = pr_url

with open(path, "w", encoding="utf-8") as f:
    json.dump(doc, f, indent=2)
PY
}

remote_retry() {
  local phase="$1"
  local cmd="$2"
  local attempt=1
  local out=""
  while [[ "${attempt}" -le "${RETRIES}" ]]; do
    if out="$(remote "${cmd}" 2>&1)"; then
      printf "%s" "${out}"
      return 0
    fi
    append_run_phase "${phase}" "retry" "attempt ${attempt}/${RETRIES} failed: ${out}"
    if [[ "${attempt}" -eq "${RETRIES}" ]]; then
      echo "${out}" >&2
      return 1
    fi
    sleep "${RETRY_DELAY}"
    attempt=$((attempt + 1))
  done
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
raw = payload[start:]
try:
    obj = json.loads(raw)
except json.JSONDecodeError:
    first_line = raw.split("\n")[0]
    obj = json.loads(first_line) if first_line.strip() else {}
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
raw = payload[start:]
try:
    obj = json.loads(raw)
except json.JSONDecodeError:
    first_line = raw.split("\n")[0]
    obj = json.loads(first_line) if first_line.strip() else {}
if isinstance(obj, list):
    obj = obj[0] if obj else {}
notes = obj.get("notes", "")
match = re.search(r"https://github.com/[^\s]+/pull/\d+", notes)
print(match.group(0) if match else "")
PY
}

echo "[orchestrate] waiting for opencode-agent deployment"
mkdir -p .sdp/runs
RUN_ID="orchestrate-${ISSUE}-$(date -u +%Y%m%dT%H%M%SZ)"
RUN_FILE=".sdp/runs/${RUN_ID}.json"
RUN_ID="${RUN_ID}" ISSUE="${ISSUE}" HOST="${HOST}" POLL="${POLL}" TIMEOUT="${TIMEOUT}" RUN_FILE="${RUN_FILE}" python3 - <<'PY'
import json
import os
from datetime import datetime, timezone

doc = {
    "run_id": os.environ["RUN_ID"],
    "issue_id": os.environ["ISSUE"],
    "orchestrator": "scripts/orchestrate_k8s_issue.sh",
    "host": os.environ["HOST"],
    "poll_seconds": int(os.environ["POLL"]),
    "timeout_seconds": int(os.environ["TIMEOUT"]),
    "started_at": datetime.now(timezone.utc).isoformat(),
    "events": [],
}
with open(os.environ["RUN_FILE"], "w", encoding="utf-8") as f:
    json.dump(doc, f, indent=2)
PY
echo "[orchestrate] run_id=${RUN_ID}"
echo "[orchestrate] run_file=${RUN_FILE}"

append_run_phase "deployment" "running" "waiting for opencode-agent rollout"
remote_retry "deployment" "kubectl -n sdp-workers rollout status deployment/opencode-agent --timeout=240s >/dev/null"
append_run_phase "deployment" "ok" "opencode-agent deployment ready"

echo "[orchestrate] running preflight sync inside pod"
append_run_phase "preflight" "running" "starting workspace and bd sync preflight"
remote_retry "preflight" "kubectl -n sdp-workers exec deploy/opencode-agent -- sh -lc 'cd /workspace && git rev-parse --is-inside-work-tree >/dev/null && branch=\"\${SDP_REPO_BRANCH:-master}\" && git fetch origin \"\$branch\" && git rebase FETCH_HEAD && bd sync --import-only >/dev/null'"
append_run_phase "preflight" "ok" "preflight checks passed"

raw_initial="$(remote_retry "status" "kubectl -n sdp-workers exec deploy/opencode-agent -- sh -lc 'cd /workspace && bd show ${ISSUE} --json'" 2>/dev/null || true)"
status_initial="$(extract_field "${raw_initial}" "status")"
if [[ "${status_initial}" == "closed" ]]; then
  pr_initial="$(extract_pr_url "${raw_initial}")"
  append_run_phase "terminal" "ok" "issue already closed before dispatch" "${pr_initial}"
  echo "[orchestrate] issue ${ISSUE} already closed"
  if [[ -n "${pr_initial}" ]]; then
    echo "[orchestrate] pr=${pr_initial}"
  fi
  exit 0
fi

echo "[orchestrate] triggering one explicit agent cycle"
append_run_phase "dispatch" "running" "triggering one explicit agent cycle"
dispatch_output="$(remote "kubectl -n sdp-workers exec deploy/opencode-agent -- sh -lc 'set -e; lock=/tmp/sdp-orchestrate-${ISSUE}.lock; if mkdir \"\$lock\" 2>/dev/null; then trap \"rmdir \\\"\$lock\\\"\" EXIT; opencode-agent; else echo lock-exists; fi'" 2>&1 || true)"
if [[ "${dispatch_output}" == *"lock-exists"* ]]; then
  append_run_phase "dispatch" "ok" "dispatch already in progress for this issue; entering poll mode"
else
  append_run_phase "dispatch" "ok" "explicit cycle command finished"
fi

deadline=$(( $(date +%s) + TIMEOUT ))
last_status=""

while true; do
  now=$(date +%s)
  if [[ "${now}" -ge "${deadline}" ]]; then
    append_run_phase "terminal" "failed" "timeout while waiting for terminal issue status"
    echo "[orchestrate] timeout waiting for issue ${ISSUE}"
    remote "kubectl -n sdp-workers logs deploy/opencode-agent --tail=120"
    exit 1
  fi

  raw="$(remote_retry "status" "kubectl -n sdp-workers exec deploy/opencode-agent -- sh -lc 'cd /workspace && bd show ${ISSUE} --json'" 2>/dev/null || true)"
  status="$(extract_field "${raw}" "status")"
  pr_url="$(extract_pr_url "${raw}")"

  if [[ "${status}" != "${last_status}" ]]; then
    echo "[orchestrate] issue=${ISSUE} status=${status}"
    last_status="${status}"
  fi

  if [[ "${status}" == "closed" ]]; then
    append_run_phase "terminal" "ok" "issue closed" "${pr_url}"
    echo "[orchestrate] completed issue ${ISSUE}"
    if [[ -n "${pr_url}" ]]; then
      echo "[orchestrate] pr=${pr_url}"
    fi
    exit 0
  fi

  if [[ "${status}" == "blocked" ]]; then
    append_run_phase "terminal" "failed" "issue entered blocked state"
    echo "[orchestrate] issue ${ISSUE} blocked"
    remote "kubectl -n sdp-workers logs deploy/opencode-agent --tail=120"
    exit 1
  fi

  sleep "${POLL}"
done
