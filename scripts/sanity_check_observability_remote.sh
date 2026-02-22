#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"

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
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>]"
  exit 2
fi

SSH=(ssh -p "${PORT}" "${HOST}")

echo "[sanity] target host: ${HOST}:${PORT}"
"${SSH[@]}" "kubectl -n sdp-observability get deploy,ds,pod,svc"

echo "[sanity] loki telemetry labels seen in last 15m"
"${SSH[@]}" "kubectl -n sdp-observability port-forward svc/loki 3100:3100 >/tmp/sdp-loki-port-forward.log 2>&1 & echo \$! > /tmp/sdp-loki-port-forward.pid"
sleep 2

cleanup() {
  "${SSH[@]}" "if [ -f /tmp/sdp-loki-port-forward.pid ]; then kill \$(cat /tmp/sdp-loki-port-forward.pid) >/dev/null 2>&1 || true; rm -f /tmp/sdp-loki-port-forward.pid; fi"
}
trap cleanup EXIT

"${SSH[@]}" "python3 - <<'PY'
import json
import urllib.parse
import urllib.request

query = '{namespace=~"sdp-(workers|control)",record_type=~"event|metric"}'
params = urllib.parse.urlencode({
    'query': query,
    'start': '0',
    'limit': '5',
    'direction': 'backward',
})
url = 'http://127.0.0.1:3100/loki/api/v1/query_range?' + params
with urllib.request.urlopen(url, timeout=10) as resp:
    payload = json.load(resp)
result = payload.get('data', {}).get('result', [])
print(f'loki_streams={len(result)}')
for stream in result[:5]:
    labels = stream.get('stream', {})
    print('stream', {
        'namespace': labels.get('namespace', ''),
        'component': labels.get('component', ''),
        'record_type': labels.get('record_type', ''),
        'issue_id': labels.get('issue_id', ''),
    })
PY"
