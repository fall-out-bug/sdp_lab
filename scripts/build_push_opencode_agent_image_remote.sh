#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
IMAGE="ghcr.io/fall-out-bug/sdp-dev-opencode-agent:latest"
REMOTE_DIR="/tmp/sdp_dev_image_build"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCAL_BIN="${ROOT_DIR}/.tmp/opencode-agent-linux-amd64"

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

TOKEN="$(gh auth token)"

echo "[remote-image] building linux/amd64 binary locally"
mkdir -p "${ROOT_DIR}/.tmp"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${LOCAL_BIN}" "${ROOT_DIR}/cmd/opencode-agent"

echo "[remote-image] preparing remote build dir ${REMOTE_DIR}"
ssh -p "${PORT}" "${HOST}" "rm -rf ${REMOTE_DIR} && mkdir -p ${REMOTE_DIR}"
scp -P "${PORT}" "${LOCAL_BIN}" "${HOST}:${REMOTE_DIR}/opencode-agent"
scp -P "${PORT}" "${ROOT_DIR}/deploy/images/opencode-agent/Dockerfile.scratch" "${HOST}:${REMOTE_DIR}/Dockerfile"

echo "[remote-image] docker login ghcr.io"
ssh -p "${PORT}" "${HOST}" "echo '${TOKEN}' | docker login ghcr.io -u fall-out-bug --password-stdin"

echo "[remote-image] build and push ${IMAGE}"
ssh -p "${PORT}" "${HOST}" "docker build -t ${IMAGE} -f ${REMOTE_DIR}/Dockerfile ${REMOTE_DIR} && docker push ${IMAGE}"

echo "[remote-image] done"
