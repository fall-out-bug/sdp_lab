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

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFEST_DIR="${ROOT_DIR}/deploy/k8s/observability"

echo "[apply] copying observability manifests to ${HOST}:${PORT}"
ssh -p "${PORT}" "${HOST}" "mkdir -p /tmp/sdp-dev-observability"
scp -P "${PORT}" -r "${MANIFEST_DIR}/." "${HOST}:/tmp/sdp-dev-observability/"

echo "[apply] applying observability kustomization on remote"
ssh -p "${PORT}" "${HOST}" "kubectl apply -k /tmp/sdp-dev-observability"

echo "[apply] rollout status"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-observability rollout status deploy/loki --timeout=180s"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-observability rollout status deploy/tempo --timeout=180s"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-observability rollout status deploy/otel-collector --timeout=180s"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-observability rollout status deploy/prometheus --timeout=180s"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-observability rollout status deploy/grafana --timeout=180s"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-observability rollout status ds/promtail --timeout=180s"

ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-observability get deploy,ds,pod,svc"
