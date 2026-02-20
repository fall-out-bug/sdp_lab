#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
IMAGE=""
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

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
    --image)
      IMAGE="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--image <image>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--image <image>]"
  exit 2
fi

if [[ -z "${IMAGE}" ]]; then
  COMMIT_SHA="$(git -C "${ROOT_DIR}" rev-parse --short=12 HEAD)"
  IMAGE="sdp-dev-opencode-agent:git-${COMMIT_SHA}"
fi

echo "[promote] ensuring image exists on remote: ${IMAGE}"
ssh -p "${PORT}" "${HOST}" "docker image inspect ${IMAGE} >/dev/null"

echo "[promote] loading image into minikube cache"
ssh -p "${PORT}" "${HOST}" "minikube image load --overwrite=true ${IMAGE}"

echo "[promote] updating deployment image"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-workers set image deployment/opencode-agent opencode-agent=${IMAGE} init-workspace=${IMAGE}"

echo "[promote] waiting rollout"
ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-workers rollout status deployment/opencode-agent --timeout=240s"

echo "[promote] done image=${IMAGE}"
