#!/usr/bin/env bash
# E2E integration test: AgentRun on minikube
# WS-002-03: Validate full flow: create AgentRun -> Tasks -> terminal status
#
# Prerequisites:
#   - kubectl pointing to minikube (or any K8s cluster)
#   - adapter-controller image built and loaded (minikube: minikube image load sdp/adapter-controller:latest)
#
# Usage:
#   ./scripts/e2e_agentrun_minikube.sh [--build-image] [--skip-deploy] [--workspace-pvc]
#
# --workspace-pvc: Use PVC with init container for .beads/.sdp (WS-002-03 deferred AC)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
NAMESPACE="${SDP_ADAPTER_NS:-sdp-adapter}"
ISSUE_ID="${E2E_ISSUE_ID:-e2e-test-$(date +%s)}"
RUN_ID="e2e-run-$(date +%s)"
TIMEOUT=90

BUILD_IMAGE=false
SKIP_DEPLOY=false
WORKSPACE_PVC=false
for arg in "$@"; do
  case "$arg" in
    --build-image) BUILD_IMAGE=true ;;
    --skip-deploy) SKIP_DEPLOY=true ;;
    --workspace-pvc) WORKSPACE_PVC=true ;;
  esac
done

log() { echo "[e2e] $*"; }
fail() { log "FAIL: $*"; exit 1; }

# Ensure namespace exists
ensure_namespace() {
  if ! kubectl get namespace "$NAMESPACE" &>/dev/null; then
    kubectl create namespace "$NAMESPACE"
  fi
}

# Deploy adapter-controller
# Uses deploy/k8s/adapter/overlays/dev (emptyDir) for minimal E2E.
# With --workspace-pvc: uses overlays/e2e-pvc (PVC + init container for .beads/.sdp)
deploy_adapter() {
  local overlay="dev"
  if [ "$WORKSPACE_PVC" = true ]; then
    overlay="e2e-pvc"
    log "Deploying adapter-controller with PVC (workspace init)..."
  else
    log "Deploying adapter-controller..."
  fi
  kubectl kustomize "$ROOT_DIR/deploy/k8s/adapter/overlays/$overlay" | kubectl apply -f -
  if [ "$WORKSPACE_PVC" = true ]; then
    kubectl -n "$NAMESPACE" rollout status deployment/nats --timeout=60s
  fi
  kubectl -n "$NAMESPACE" rollout status deployment/adapter-controller --timeout=120s
  log "Adapter deployed"
}

# Build and load image for minikube
# Set ADAPTER_IMAGE to override (default matches deploy/k8s/adapter/base/deployment.yaml)
build_and_load_image() {
  local image="${ADAPTER_IMAGE:-ghcr.io/fall-out-bug/sdp-dev-adapter-controller:latest}"
  if command -v minikube &>/dev/null; then
    log "Building adapter-controller image..."
    docker build -t "$image" -f "$ROOT_DIR/deploy/images/adapter-controller/Dockerfile" "$ROOT_DIR"
    log "Loading image into minikube..."
    minikube image load "$image"
  else
    log "minikube not found, skipping image load (use existing image)"
  fi
}

# Create beads issue when using PVC (required for Beads transition AC)
# Sets ISSUE_ID to the created issue's ID for AgentRun
create_beads_issue() {
  log "Creating beads issue in workspace..."
  local out
  out=$(kubectl -n "$NAMESPACE" exec deploy/adapter-controller -- \
    sh -c "cd /workspaces/default && bd create 'E2E test' -t task -p 1 --labels 'autonomy,strict-evidence' --json" 2>/dev/null) || true
  if [[ -n "$out" ]]; then
    ISSUE_ID=$(echo "$out" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('id','') or (d[0].get('id','') if isinstance(d,list) else ''))" 2>/dev/null || echo "$ISSUE_ID")
    log "Created beads issue $ISSUE_ID"
  fi
}

# Create test AgentRun
create_agentrun() {
  log "Creating AgentRun $RUN_ID with issue $ISSUE_ID"
  kubectl apply -f - <<EOF
apiVersion: sdp.dev/v1alpha1
kind: AgentRun
metadata:
  name: $RUN_ID
  namespace: $NAMESPACE
spec:
  issueId: $ISSUE_ID
  model: glm-4.7
EOF
}

# Wait for at least N tasks to appear
wait_for_tasks() {
  local expected=$1
  local elapsed=0
  while [ $elapsed -lt $TIMEOUT ]; do
    local count
    count=$(kubectl -n "$NAMESPACE" get tasks -l "agentrun=$RUN_ID" --no-headers 2>/dev/null | wc -l | tr -d ' ')
    if [ "${count:-0}" -ge "$expected" ]; then
      log "Found $count Tasks (expected >= $expected)"
      return 0
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done
  fail "Timeout waiting for $expected Tasks (got ${count:-0})"
}

# Patch all Tasks to Succeeded (simulates completion)
# Sequential pipeline (F004): analyst -> coder -> reviewer; each phase creates next task.
patch_tasks_succeeded() {
  for task in $(kubectl -n "$NAMESPACE" get tasks -l "agentrun=$RUN_ID" -o name 2>/dev/null); do
    kubectl -n "$NAMESPACE" patch "$task" --subresource=status --type=merge -p '{"phase":"Succeeded"}' 2>/dev/null || \
    kubectl -n "$NAMESPACE" patch "$task" --type=merge -p '{"status":{"phase":"Succeeded"}}' 2>/dev/null || true
  done
}

# Drive sequential pipeline: patch tasks as they appear until AgentRun Succeeded.
# F004: analyst (1) -> coder (2) -> reviewer (3); each patch triggers next phase.
drive_sequential_pipeline() {
  local elapsed=0
  local last_count=0
  while [ $elapsed -lt $TIMEOUT ]; do
    local phase
    phase=$(kubectl -n "$NAMESPACE" get agentrun "$RUN_ID" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [ "$phase" = "Succeeded" ]; then
      return 0
    fi
    if [ "$phase" = "Failed" ]; then
      local err
      err=$(kubectl -n "$NAMESPACE" get agentrun "$RUN_ID" -o jsonpath='{.status.lastError}' 2>/dev/null || echo "unknown")
      fail "AgentRun Failed: $err"
    fi
    local count
    count=$(kubectl -n "$NAMESPACE" get tasks -l "agentrun=$RUN_ID" --no-headers 2>/dev/null | wc -l | tr -d ' ')
    if [ "${count:-0}" -gt "$last_count" ]; then
      log "Found $count Tasks, patching to Succeeded..."
      patch_tasks_succeeded
      last_count=$count
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done
  fail "Timeout driving sequential pipeline (phase=$phase)"
}

# Wait for AgentRun to reach Succeeded
wait_for_agentrun_succeeded() {
  local elapsed=0
  while [ $elapsed -lt $TIMEOUT ]; do
    local phase
    phase=$(kubectl -n "$NAMESPACE" get agentrun "$RUN_ID" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [ "$phase" = "Succeeded" ]; then
      log "AgentRun reached Succeeded"
      return 0
    fi
    if [ "$phase" = "Failed" ]; then
      local err
      err=$(kubectl -n "$NAMESPACE" get agentrun "$RUN_ID" -o jsonpath='{.status.lastError}' 2>/dev/null || echo "unknown")
      fail "AgentRun Failed: $err"
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done
  fail "Timeout waiting for AgentRun Succeeded (phase=$phase)"
}

# Main
main() {
  log "E2E AgentRun test: namespace=$NAMESPACE, issue=$ISSUE_ID, run=$RUN_ID"
  cd "$ROOT_DIR"

  if [ "$BUILD_IMAGE" = true ]; then
    build_and_load_image
  fi

  ensure_namespace

  if [ "$SKIP_DEPLOY" != true ]; then
    deploy_adapter
  fi

  if [ "$WORKSPACE_PVC" = true ]; then
    create_beads_issue
  fi

  create_agentrun

  # AC: AgentRun created -> at least 1 Task (analyst) appears within 30s
  wait_for_tasks 1

  # F004 sequential pipeline: analyst -> coder -> reviewer. Patch each as it appears until Succeeded.
  drive_sequential_pipeline

  log "E2E PASS: AgentRun $RUN_ID completed successfully"
}

main "$@"
