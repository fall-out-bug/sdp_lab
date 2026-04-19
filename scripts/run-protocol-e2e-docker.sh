#!/usr/bin/env bash
# Run protocol E2E in Docker (from sdp_dev root).
# Usage: ./scripts/run-protocol-e2e-docker.sh
# Set GLM_API_KEY for Phase 5 (opencode LLM code generation).

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Dockerfile resolution: prefer sdp/ci/ (local checkout), fall back to native ci/
if [ -f "$ROOT/sdp/ci/Dockerfile.protocol-e2e" ]; then
  DOCKERFILE="$ROOT/sdp/ci/Dockerfile.protocol-e2e"
  BUILD_ARGS="--build-arg SDP_PLUGIN_PATH=sdp/sdp-plugin"
elif [ -f "$ROOT/ci/Dockerfile.protocol-e2e" ]; then
  DOCKERFILE="$ROOT/ci/Dockerfile.protocol-e2e"
  BUILD_ARGS=""
else
  echo "ERROR: Dockerfile not found in sdp/ci/ or ci/." >&2
  echo "Clone the public sdp repo: git clone https://github.com/fall-out-bug/sdp.git sdp" >&2
  exit 1
fi

echo "Building protocol-e2e image..."
docker build -f "$DOCKERFILE" $BUILD_ARGS -t sdp-protocol-e2e:latest "$ROOT"

echo "Running protocol-e2e (GLM_API_KEY=${GLM_API_KEY:+set})..."
docker run --rm -e GLM_API_KEY="${GLM_API_KEY}" sdp-protocol-e2e:latest
