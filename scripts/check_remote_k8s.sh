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

echo "[health] target host: ${HOST}:${PORT}"
"${SSH[@]}" "kubectl -n sdp-control get deploy,pod"
"${SSH[@]}" "kubectl -n sdp-workers get deploy,pod"
"${SSH[@]}" "kubectl -n sdp-observability get deploy,pod"
