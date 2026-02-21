#!/usr/bin/env bash
# SSH tunnel to remote minikube API server.
# Run in background, then use KUBECONFIG=... kubectl locally.
#
# Usage:
#   ./scripts/remote_minikube_tunnel.sh --host user@host [--port 2222] [--local-port 8443]
#   # In another terminal:
#   export KUBECONFIG=$PWD/.kube/remote-minikube.yaml
#   kubectl get pods -n kubeopencode-system
set -euo pipefail

HOST=""
PORT="22"
LOCAL_PORT="8443"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KUBECONFIG_DIR="${ROOT_DIR}/.kube"
KUBECONFIG_FILE="${KUBECONFIG_DIR}/remote-minikube.yaml"

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
    --local-port)
      LOCAL_PORT="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--local-port <port>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--local-port <port>]"
  echo ""
  echo "Example:"
  echo "  $0 --host fall_out_bug@192.168.50.219 --port 2222"
  echo ""
  echo "Then in another terminal:"
  echo "  export KUBECONFIG=$KUBECONFIG_FILE"
  echo "  kubectl get pods -n kubeopencode-system"
  exit 2
fi

echo "[tunnel] Fetching kubeconfig from remote..."
mkdir -p "${KUBECONFIG_DIR}"
RAW_CONFIG="$(ssh -p "${PORT}" "${HOST}" "kubectl config view --minify --flatten" 2>/dev/null)"
REMOTE_API="$(echo "${RAW_CONFIG}" | grep -E '^\s+server:' | head -1 | sed -E 's|.*https://([^/]+).*|\1|')"
if [[ -z "${REMOTE_API}" ]]; then
  echo "[tunnel] Could not detect API server address"
  exit 1
fi
echo "${RAW_CONFIG}" | sed "s|https://${REMOTE_API}|https://127.0.0.1:${LOCAL_PORT}|g" > "${KUBECONFIG_FILE}"

echo "[tunnel] Starting SSH tunnel: localhost:${LOCAL_PORT} -> ${HOST}:${REMOTE_API}"
echo "[tunnel] KUBECONFIG=${KUBECONFIG_FILE}"
echo "[tunnel] Press Ctrl+C to stop"
exec ssh -p "${PORT}" -L "${LOCAL_PORT}:${REMOTE_API}" -N "${HOST}"
