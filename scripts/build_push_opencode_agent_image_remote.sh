#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
IMAGE=""
REMOTE_DIR="/tmp/sdp_lab_image_build"
RETRIES="5"
BD_VERSION="v0.49.6"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_BIN_DIR="${ROOT_DIR}/.tmp/opencode-runtime"

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
    --retries)
      RETRIES="$2"
      shift 2
      ;;
    --bd-version)
      BD_VERSION="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--image <image>] [--retries <n>] [--bd-version <vX.Y.Z>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--image <image>] [--retries <n>] [--bd-version <vX.Y.Z>]"
  exit 2
fi

if [[ -z "${IMAGE}" ]]; then
  COMMIT_SHA="$(git -C "${ROOT_DIR}" rev-parse --short=12 HEAD)"
  IMAGE="sdp-dev-opencode-agent:git-${COMMIT_SHA}"
fi

BD_SRC_DIR="${ROOT_DIR}/.tmp/beads-src-${BD_VERSION}"

echo "[remote-image] building linux runtime binaries locally"
mkdir -p "${RUNTIME_BIN_DIR}"
rm -f "${RUNTIME_BIN_DIR}"/*

if [[ ! -d "${BD_SRC_DIR}" ]]; then
  git clone --depth 1 --branch "${BD_VERSION}" https://github.com/steveyegge/beads.git "${BD_SRC_DIR}"
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/opencode-agent" ./cmd/opencode-agent
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/orchestrator" ./cmd/orchestrator
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/swarm-worker" ./cmd/swarm-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/swarm-reviewer" ./cmd/swarm-reviewer
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/autonomy-worker" ./cmd/autonomy-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/beads-fsm" ./cmd/beads-fsm
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/pr-gate" ./cmd/pr-gate
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/pr-publish" ./cmd/pr-publish
(cd "${BD_SRC_DIR}" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/bd" ./cmd/bd)

echo "[remote-image] preparing remote build dir ${REMOTE_DIR}"
ssh -p "${PORT}" "${HOST}" "rm -rf ${REMOTE_DIR} && mkdir -p ${REMOTE_DIR}"

echo "[remote-image] syncing build context"
COPYFILE_DISABLE=1 tar \
  --exclude=.git \
  --exclude=.sdp \
  --exclude=.beads/bd.sock \
  --exclude=.beads/daemon.pid \
  --exclude=.beads/daemon.lock \
  --exclude=.beads/daemon.log \
  -czf - . | ssh -p "${PORT}" "${HOST}" "tar -xzf - -C ${REMOTE_DIR}"

echo "[remote-image] build local image ${IMAGE} on remote host"
ssh -p "${PORT}" "${HOST}" "RETRIES='${RETRIES}' IMAGE='${IMAGE}' REMOTE_DIR='${REMOTE_DIR}' bash -s" <<'EOF'
set -euo pipefail
for i in $(seq 1 "${RETRIES}"); do
  echo "[remote-image] build attempt ${i}/${RETRIES}"
  if docker build -t "${IMAGE}" -f "${REMOTE_DIR}/deploy/images/opencode-agent/Dockerfile.runtime" "${REMOTE_DIR}"; then
    exit 0
  fi
  if [ "${i}" -lt "${RETRIES}" ]; then
    sleep 5
  fi
done
echo "[remote-image] build failed after ${RETRIES} attempts" >&2
exit 1
EOF
ssh -p "${PORT}" "${HOST}" "docker image ls ${IMAGE}"

echo "[remote-image] done"
echo "[remote-image] image=${IMAGE}"
