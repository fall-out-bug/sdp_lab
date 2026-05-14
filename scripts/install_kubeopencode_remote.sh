#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
NAMESPACE="kubeopencode-system"
RELEASE="kubeopencode"

usage() {
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--namespace <ns>] [--release <name>]"
}

validate_host() {
  local host="$1"
  local user_re='[A-Za-z0-9._][A-Za-z0-9._-]*'
  local label_re='[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?'
  local hostname_re="${label_re}(\\.${label_re})*"
  local ipv4_re='([0-9]{1,3}\.){3}[0-9]{1,3}'

  if [[ -z "$host" || "$host" == -* ]]; then
    return 1
  fi
  if [[ "$host" =~ ^(${user_re}@)?(${hostname_re}|${ipv4_re})$ ]]; then
    return 0
  fi
  return 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --host" >&2
        usage
        exit 2
      fi
      HOST="$2"
      shift 2
      ;;
    --port)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --port" >&2
        usage
        exit 2
      fi
      PORT="$2"
      shift 2
      ;;
    --namespace)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --namespace" >&2
        usage
        exit 2
      fi
      NAMESPACE="$2"
      shift 2
      ;;
    --release)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --release" >&2
        usage
        exit 2
      fi
      RELEASE="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      usage
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  usage
  exit 2
fi
if ! validate_host "$HOST"; then
  echo "Invalid --host: $HOST" >&2
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
ssh -p "${PORT}" -- "${HOST}" bash -s -- "${NAMESPACE}" <<'REMOTE'
set -euo pipefail
namespace="$1"
kubectl get ns "$namespace" >/dev/null 2>&1 || kubectl create ns "$namespace"
REMOTE
ssh -p "${PORT}" -- "${HOST}" bash -s -- "${RELEASE}" "${NAMESPACE}" <<'REMOTE'
set -euo pipefail
release="$1"
namespace="$2"
helm upgrade --install "$release" oci://quay.io/kubeopencode/helm-charts/kubeopencode --namespace "$namespace" --set server.enabled=false
REMOTE
DEPLOY_NAME="$(ssh -p "${PORT}" -- "${HOST}" bash -s -- "${NAMESPACE}" "${RELEASE}" <<'REMOTE'
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
ssh -p "${PORT}" -- "${HOST}" bash -s -- "${NAMESPACE}" "${DEPLOY_NAME}" <<'REMOTE'
set -euo pipefail
namespace="$1"
deploy="$2"
kubectl -n "$namespace" rollout status "deploy/$deploy" --timeout=300s
REMOTE
ssh -p "${PORT}" -- "${HOST}" bash -s -- "${NAMESPACE}" <<'REMOTE'
set -euo pipefail
namespace="$1"
kubectl -n "$namespace" get deploy,pods
REMOTE
