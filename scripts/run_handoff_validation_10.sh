#!/usr/bin/env bash
# Run 10 consecutive orchestrate cycles for adapter handoff checklist validation (sdp_dev-oip).
# Usage: ./scripts/run_handoff_validation_10.sh --host <user@ip> [--port <port>] [--issues <id1,id2,...>]
set -euo pipefail

HOST=""
PORT="22"
ISSUES=""
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host) HOST="$2"; shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    --issues) ISSUES="$2"; shift 2 ;;
    *) echo "Unknown: $1"; exit 2 ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip> [--port <port>] [--issues <id1,id2,...>]"
  echo "  --issues: comma-separated beads IDs. If omitted, bd ready is used to pick 10."
  exit 2
fi

mkdir -p "${ROOT_DIR}/.sdp/runs"
RUN_LOG="${ROOT_DIR}/.sdp/runs/handoff-validation-10-$(date +%Y%m%dT%H%M%SZ).log"

if [[ -z "${ISSUES}" ]]; then
  echo "[validation] picking 10 issues from bd ready"
  ISSUES="$(cd "${ROOT_DIR}" && bd ready --json 2>/dev/null | python3 -c "
import json,sys
data=json.load(sys.stdin)
items=data if isinstance(data,list) else data.get('issues',[])
ids=[i['id'] for i in items if isinstance(i,dict) and i.get('id')]
print(','.join(ids[:10]))
" 2>/dev/null || echo "")"
  if [[ -z "${ISSUES}" ]]; then
    echo "[validation] no ready issues; create tasks or pass --issues id1,id2,..."
    exit 1
  fi
fi

count=0
for id in $(echo "${ISSUES}" | tr ',' ' '); do
  count=$((count + 1))
  echo "[validation] run ${count}/10: issue ${id}" | tee -a "${RUN_LOG}"
  if ! "${ROOT_DIR}/scripts/orchestrate_k8s_issue.sh" --host "${HOST}" --port "${PORT}" --issue "${id}" >> "${RUN_LOG}" 2>&1; then
    echo "[validation] run ${count} failed; see ${RUN_LOG}"
    exit 1
  fi
  echo "[validation] run ${count} ok" | tee -a "${RUN_LOG}"
done

echo "[validation] 10 consecutive runs completed successfully"
echo "[validation] log: ${RUN_LOG}"
