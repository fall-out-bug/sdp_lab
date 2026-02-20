#!/usr/bin/env bash
set -euo pipefail

IMAGE="ghcr.io/fall-out-bug/sdp-dev-opencode-agent:latest"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "[image] target: ${IMAGE}"
gh auth token | docker login ghcr.io -u fall-out-bug --password-stdin

docker build -t "${IMAGE}" -f "${ROOT_DIR}/deploy/images/opencode-agent/Dockerfile" "${ROOT_DIR}"
docker push "${IMAGE}"

echo "[image] pushed ${IMAGE}"
