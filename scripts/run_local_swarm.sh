#!/usr/bin/env bash
# Run one swarm cycle locally. Starts docker compose stack, then runs swarm-worker.
# Prerequisites: bd onboard, at least one autonomy task in backlog.
# Usage: ./scripts/run_local_swarm.sh [--no-compose]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

NO_COMPOSE=false
for arg in "$@"; do
  case "$arg" in
    --no-compose)
      NO_COMPOSE=true
      shift
      ;;
    *)
      ;;
  esac
done

if [[ "$NO_COMPOSE" == "false" ]]; then
  echo "[run_local_swarm] Starting docker compose stack..."
  docker compose up -d
  echo "[run_local_swarm] Waiting for NATS..."
  sleep 5
  echo "[run_local_swarm] Stack ready. Intake: http://localhost:8081, Registry: http://localhost:8080"
fi

echo "[run_local_swarm] Syncing Beads..."
"${PROJECT_ROOT}/scripts/beads_transport.sh" fetch >/dev/null 2>&1 || true

echo "[run_local_swarm] Running one swarm cycle..."
export SDP_MODEL="${SDP_MODEL:-glm-4.7}"
go run ./cmd/swarm-worker
