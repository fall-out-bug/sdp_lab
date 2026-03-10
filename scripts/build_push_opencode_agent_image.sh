#!/usr/bin/env bash
set -euo pipefail

IMAGE="ghcr.io/fall-out-bug/sdp-dev-opencode-agent:latest"
BD_VERSION="v0.59.0"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_BIN_DIR="${ROOT_DIR}/.tmp/opencode-runtime"
BD_SRC_DIR="${ROOT_DIR}/.tmp/beads-src-${BD_VERSION}"

echo "[image] target: ${IMAGE}"

echo "[image] building linux runtime binaries"
mkdir -p "${RUNTIME_BIN_DIR}"
rm -f "${RUNTIME_BIN_DIR}"/*

if [[ ! -d "${BD_SRC_DIR}" ]]; then
  git clone --depth 1 --branch "${BD_VERSION}" https://github.com/steveyegge/beads.git "${BD_SRC_DIR}"
fi

cd "${ROOT_DIR}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/opencode-agent" ./cmd/opencode-agent
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/orchestrator" ./cmd/orchestrator
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/swarm-worker" ./cmd/swarm-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/swarm-reviewer" ./cmd/swarm-reviewer
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/autonomy-worker" ./cmd/autonomy-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/beads-fsm" ./cmd/beads-fsm
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/pr-gate" ./cmd/pr-gate
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/pr-publish" ./cmd/pr-publish
(cd "${BD_SRC_DIR}" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${RUNTIME_BIN_DIR}/bd" ./cmd/bd)

echo "[image] docker build with Dockerfile.runtime"
if [[ -n "${GITHUB_TOKEN:-}" ]]; then
  printf '%s' "${GITHUB_TOKEN}" | docker login ghcr.io -u fall-out-bug --password-stdin
else
  gh auth token | docker login ghcr.io -u fall-out-bug --password-stdin
fi

docker build -t "${IMAGE}" -f "${ROOT_DIR}/deploy/images/opencode-agent/Dockerfile.runtime" "${ROOT_DIR}"
docker push "${IMAGE}"

echo "[image] pushed ${IMAGE}"
