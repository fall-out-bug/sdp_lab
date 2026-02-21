#!/usr/bin/env bash
# Deploy SDP Swarm Platform to k8s.
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Deploying SDP Swarm Platform from $REPO_ROOT"

# Create namespaces if not exist
kubectl create namespace sdp-control --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace sdp-workers --dry-run=client -o yaml | kubectl apply -f -

# Apply control plane (NATS, intake-gateway, swarm-orchestrator)
kubectl apply -k "$REPO_ROOT/deploy/k8s/control/"

# Apply workers (role agents)
kubectl apply -f "$REPO_ROOT/deploy/k8s/workers/role-agent-coder.yaml" 2>/dev/null || true
kubectl apply -f "$REPO_ROOT/deploy/k8s/workers/role-agent-analyst.yaml" 2>/dev/null || true

# KEDA scalers (namespace sdp-workers, requires KEDA operator)
kubectl apply -f "$REPO_ROOT/deploy/k8s/control/keda-scalers.yaml" 2>/dev/null || true

echo "Deploy complete. Check: kubectl get pods -n sdp-control"
echo "NATS: kubectl port-forward -n sdp-control svc/nats 4222:4222 8222:8222"
