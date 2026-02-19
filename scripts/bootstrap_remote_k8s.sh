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

echo "[bootstrap] target host: ${HOST}:${PORT}"
"${SSH[@]}" "kubectl create ns sdp-control --dry-run=client -o yaml | kubectl apply -f -"
"${SSH[@]}" "kubectl create ns sdp-workers --dry-run=client -o yaml | kubectl apply -f -"
"${SSH[@]}" "kubectl create ns sdp-observability --dry-run=client -o yaml | kubectl apply -f -"
"${SSH[@]}" "kubectl create ns sdp-openclaw --dry-run=client -o yaml | kubectl apply -f -"

echo "[bootstrap] namespaces ensured"
"${SSH[@]}" "kubectl get ns sdp-control sdp-workers sdp-observability sdp-openclaw"
