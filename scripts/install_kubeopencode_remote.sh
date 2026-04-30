#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
NAMESPACE="kubeopencode-system"
RELEASE="kubeopencode"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      HOST="$2"
      shift 2
      ;;
    --port)
      PORT="$2"
      shift 2
      ;;
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --release)
      RELEASE="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--namespace <ns>] [--release <name>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--namespace <ns>] [--release <name>]"
  exit 2
fi
if [[ ! "$PORT" =~ ^[0-9]+$ ]]; then
  echo "Invalid --port: $PORT" >&2
  exit 2
fi
if [[ ! "$NAMESPACE" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ ]]; then
  echo "Invalid Kubernetes namespace: $NAMESPACE" >&2
  exit 2
fi
if [[ ! "$RELEASE" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ ]]; then
  echo "Invalid Helm release name: $RELEASE" >&2
  exit 2
fi

echo "[kubeopencode] install/upgrade ${RELEASE} in ${NAMESPACE}"
ssh -p "${PORT}" "${HOST}" bash -s -- "${NAMESPACE}" <<'REMOTE'
set -euo pipefail
namespace="$1"
kubectl get ns "$namespace" >/dev/null 2>&1 || kubectl create ns "$namespace"
REMOTE
ssh -p "${PORT}" "${HOST}" bash -s -- "${RELEASE}" "${NAMESPACE}" <<'REMOTE'
set -euo pipefail
release="$1"
namespace="$2"
helm upgrade --install "$release" oci://quay.io/kubeopencode/helm-charts/kubeopencode --namespace "$namespace" --set server.enabled=false
REMOTE
DEPLOY_NAME="$(ssh -p "${PORT}" "${HOST}" bash -s -- "${NAMESPACE}" "${RELEASE}" <<'REMOTE'
set -euo pipefail
namespace="$1"
release="$2"
kubectl -n "$namespace" get deploy -l "app.kubernetes.io/instance=$release" -o jsonpath='{.items[0].metadata.name}'
REMOTE
)"
if [[ -z "${DEPLOY_NAME}" ]]; then
  echo "No deployment found for release ${RELEASE} in ${NAMESPACE}"
  exit 1
fi
ssh -p "${PORT}" "${HOST}" bash -s -- "${NAMESPACE}" "${DEPLOY_NAME}" <<'REMOTE'
set -euo pipefail
namespace="$1"
deploy="$2"
kubectl -n "$namespace" rollout status "deploy/$deploy" --timeout=300s
REMOTE
ssh -p "${PORT}" "${HOST}" bash -s -- "${NAMESPACE}" <<'REMOTE'
set -euo pipefail
namespace="$1"
kubectl -n "$namespace" get deploy,pods
REMOTE
