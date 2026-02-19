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
MANIFEST_DIR="${ROOT_DIR}/deploy/k8s/workers"

echo "[apply] copying worker manifests to ${HOST}:${PORT}"
ssh -p "${PORT}" "${HOST}" "mkdir -p /tmp/sdp-dev-workers"
scp -P "${PORT}" -r "${MANIFEST_DIR}/." "${HOST}:/tmp/sdp-dev-workers/"

echo "[apply] applying worker kustomization on remote"
ssh -p "${PORT}" "${HOST}" "kubectl apply -k /tmp/sdp-dev-workers"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-workers get deploy,pod"
