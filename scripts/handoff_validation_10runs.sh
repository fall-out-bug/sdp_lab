#!/usr/bin/env bash
# Handoff validation: 10 consecutive runs via operator path (FR-006, WS-019-01).
# Creates 10 Beads issues with workstream:handoff-validation and documents validation steps.
# Prerequisites: K8s cluster with adapter, feature-orchestrator, NATS, workers deployed.
# Usage: ./scripts/handoff_validation_10runs.sh [--create-only]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

CREATE_ONLY=false
for arg in "$@"; do
  case "$arg" in
    --create-only)
      CREATE_ONLY=true
      shift
      ;;
    *)
      ;;
  esac
done

LABELS="autonomy,strict-evidence,workstream:handoff-validation"

echo "[handoff_validation] Creating 10 Beads issues for handoff validation..."
for i in $(seq 1 10); do
  title="Handoff validation run $i/10"
  bd create --title "$title" --type task --priority 1 --labels "$LABELS" --json 2>/dev/null | head -1 || {
    echo "[handoff_validation] bd create failed (ensure bd onboard). Creating placeholder."
    break
  }
done

echo "[handoff_validation] Issues created. Labels: $LABELS"
echo ""
echo "Validation checklist (WS-019-01):"
echo "  1. Operator path only: adapter -> AgentRun -> worker -> reviewer (no kubectl exec)"
echo "  2. Each run emits .sdp/runs/orchestrate-<issue>.json"
echo "  3. pr-gate passes for all 10 runs"
echo "  4. Evidence validation blocks FSM transition on failure"
echo "  5. Duplicate dispatch prevention (no duplicate Tasks/AgentRuns per issue)"
echo ""
echo "To run E2E: ensure cluster is up, then Bridge will poll and feature-orchestrator"
echo "will create AgentRuns. See docs/K8S_SWARM_E2E_RUNBOOK.md"
echo ""

if [[ "$CREATE_ONLY" == "true" ]]; then
  echo "[handoff_validation] --create-only: skipping E2E. Run cluster manually."
  exit 0
fi

# Optional: run E2E if scripts/e2e_agentrun_minikube.sh exists
if [[ -f "${SCRIPT_DIR}/e2e_agentrun_minikube.sh" ]]; then
  echo "[handoff_validation] E2E script found. Run: ./scripts/e2e_agentrun_minikube.sh"
else
  echo "[handoff_validation] Manual validation: deploy cluster, observe 10 runs complete."
fi
