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

echo "[kubeopencode] install/upgrade ${RELEASE} in ${NAMESPACE}"
ssh -p "${PORT}" "${HOST}" "kubectl get ns ${NAMESPACE} >/dev/null 2>&1 || kubectl create ns ${NAMESPACE}"
ssh -p "${PORT}" "${HOST}" "helm upgrade --install ${RELEASE} oci://quay.io/kubeopencode/helm-charts/kubeopencode --namespace ${NAMESPACE} --set server.enabled=false"
DEPLOY_NAME="$(ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} get deploy -l app.kubernetes.io/instance=${RELEASE} -o jsonpath='{.items[0].metadata.name}'")"
if [[ -z "${DEPLOY_NAME}" ]]; then
  echo "No deployment found for release ${RELEASE} in ${NAMESPACE}"
  exit 1
fi
ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} rollout status deploy/${DEPLOY_NAME} --timeout=300s"
ssh -p "${PORT}" "${HOST}" "kubectl -n ${NAMESPACE} get deploy,pods"
