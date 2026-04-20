#!/usr/bin/env bash
# Local wrapper: docker build + docker run for protocol E2E
# Usage: GLM_API_KEY=... ./ci/run-protocol-e2e.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# sdp/ is a native directory (submodule retired in F128).
# Protocol E2E assets live in sdp/ci/ (published to the public sdp repo).
if [ -f "$REPO_ROOT/sdp/ci/Dockerfile.protocol-e2e" ]; then
  DOCKERFILE="$REPO_ROOT/sdp/ci/Dockerfile.protocol-e2e"
  BUILD_ARGS="--build-arg SDP_PLUGIN_PATH=sdp/sdp-plugin"
elif [ -f "$REPO_ROOT/ci/Dockerfile.protocol-e2e" ]; then
  DOCKERFILE="$REPO_ROOT/ci/Dockerfile.protocol-e2e"
  BUILD_ARGS=""
else
  echo "ERROR: Dockerfile not found in sdp/ci/ or ci/." >&2
  echo "Clone the public sdp repo: git clone https://github.com/fall-out-bug/sdp.git sdp" >&2
  exit 1
fi

echo "=== Protocol E2E (Docker) ==="
docker build -f "$DOCKERFILE" $BUILD_ARGS -t sdp-protocol-e2e:latest "$REPO_ROOT"

echo ""
echo "=== Running protocol E2E test ==="
docker run --rm \
  -e GLM_API_KEY="${GLM_API_KEY:-}" \
  sdp-protocol-e2e:latest

echo ""
echo "Protocol E2E passed"
