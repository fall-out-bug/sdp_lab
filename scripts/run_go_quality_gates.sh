#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
GO="${ROOT}/scripts/go_with_project_toolchain.sh"
MODE="${SDP_GO_QUALITY_MODE:-container}"
DOCKER_IMAGE="${SDP_GO_QUALITY_DOCKER_IMAGE:-golang:1.26-bookworm}"

cd "$ROOT"

run_host_quality_gates() {
  if [[ ! -x "$GO" ]]; then
    echo "missing executable: $GO" >&2
    echo "run: chmod +x scripts/go_with_project_toolchain.sh" >&2
    exit 1
  fi

  echo "==> Host toolchain mode"
  echo "==> Go toolchain"
  "$GO" version

  echo "==> go build ./..."
  "$GO" build ./...

  echo "==> go test ./... -count=1"
  "$GO" test ./... -count=1

  echo "==> go vet ./..."
  "$GO" vet ./...
}

run_container_quality_gates() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker is required for container mode." >&2
    echo "Install Docker or run with SDP_GO_QUALITY_MODE=host." >&2
    exit 1
  fi

  mkdir -p "$ROOT/.cache/go-build" "$ROOT/.cache/go-mod" "$ROOT/.cache/go"

  echo "==> Container toolchain mode"
  echo "==> Docker image: ${DOCKER_IMAGE}"

  docker run --rm \
    --user "$(id -u):$(id -g)" \
    -e PATH=/usr/local/go/bin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    -e GOCACHE=/workspace/.cache/go-build \
    -e GOMODCACHE=/workspace/.cache/go-mod \
    -e GOPATH=/workspace/.cache/go \
    -v "$ROOT:/workspace" \
    -w /workspace \
    "$DOCKER_IMAGE" \
    sh -c 'set -eu; go version; go build ./...; go test ./... -count=1; go vet ./...'
}

case "$MODE" in
  container)
    run_container_quality_gates
    ;;
  host)
    run_host_quality_gates
    ;;
  *)
    echo "invalid SDP_GO_QUALITY_MODE: $MODE (expected: container|host)" >&2
    exit 1
    ;;
esac

echo "All Go quality gates passed."
