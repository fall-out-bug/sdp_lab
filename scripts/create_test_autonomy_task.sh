#!/usr/bin/env bash
# Create a test Beads task for swarm Scenario 1 (autonomy from backlog).
# Labels: autonomy, strict-evidence, risk:low, lane:commit, model:glm-5, workstream:builder
# Usage: ./scripts/create_test_autonomy_task.sh [title]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

TITLE="${1:-Add a no-op function to internal/llm for test coverage}"
LABELS="autonomy,strict-evidence,risk:low,lane:commit,model:glm-5,workstream:builder"

echo "[create_test_autonomy_task] Creating task: $TITLE"
ID=$(bd create "$TITLE" \
  --type task \
  --labels "$LABELS" \
  --description "Scenario 1 test: minimal change for swarm validation." \
  --acceptance "Function exists and has test coverage." \
  --silent)

echo "[create_test_autonomy_task] Created: $ID"
echo "Run: bd show $ID"
echo "Run: ./scripts/run_local_swarm.sh"
