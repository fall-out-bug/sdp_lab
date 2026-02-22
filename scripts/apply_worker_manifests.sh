#!/usr/bin/env bash
set -euo pipefail

HOST=""
PORT="22"
IMAGE=""
BRANCH=""

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
    --image)
      IMAGE="$2"
      shift 2
      ;;
    --branch)
      BRANCH="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--image <image>] [--branch <branch>]"
      exit 2
      ;;
  esac
done

if [[ -z "${HOST}" ]]; then
  echo "Usage: $0 --host <user@ip-or-host> [--port <port>] [--image <image>] [--branch <branch>]"
  exit 2
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFEST_DIR="${ROOT_DIR}/deploy/k8s/workers"

echo "[apply] provisioning sdp-credentials in sdp-workers"
"${ROOT_DIR}/scripts/provision_secrets.sh" --host "${HOST}" --port "${PORT}" --namespaces "sdp-workers"

echo "[apply] copying worker manifests to ${HOST}:${PORT}"
ssh -p "${PORT}" "${HOST}" "mkdir -p /tmp/sdp-dev-workers"
if [[ -n "${BRANCH}" ]]; then
  echo "[apply] patching SDP_REPO_BRANCH to ${BRANCH}"
  TMP_MANIFEST="$(mktemp -d)"
  trap "rm -rf '${TMP_MANIFEST}'" EXIT
  cp -r "${MANIFEST_DIR}/." "${TMP_MANIFEST}/"
  if sed --version >/dev/null 2>&1; then
    sed -i "s|value: feat/sdp_dev-[^[:space:]]*|value: ${BRANCH}|g" "${TMP_MANIFEST}/opencode-agent.yaml"
  else
    sed -i '' "s|value: feat/sdp_dev-[^[:space:]]*|value: ${BRANCH}|g" "${TMP_MANIFEST}/opencode-agent.yaml"
  fi
  scp -P "${PORT}" -r "${TMP_MANIFEST}/." "${HOST}:/tmp/sdp-dev-workers/"
else
  scp -P "${PORT}" -r "${MANIFEST_DIR}/." "${HOST}:/tmp/sdp-dev-workers/"
fi

echo "[apply] applying worker kustomization on remote"
ssh -p "${PORT}" "${HOST}" "kubectl apply -k /tmp/sdp-dev-workers"

if [[ -n "${IMAGE}" ]]; then
  echo "[apply] pinning opencode-agent image ${IMAGE}"
  ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-workers set image deployment/opencode-agent opencode-agent=${IMAGE} init-workspace=${IMAGE}"
fi

ssh -p "${PORT}" "${HOST}" "kubectl -n sdp-workers get deploy,pod"
