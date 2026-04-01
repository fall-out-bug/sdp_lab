#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

REMOTE_NAME="${BEADS_DOLT_REMOTE_NAME:-origin}"
REMOTE_URL="${BEADS_DOLT_REMOTE_URL:-}"
REPLACE=false
PUSH=false

usage() {
  cat <<'EOF'
Usage: scripts/beads_dolt_remote_bootstrap.sh --url <dolt-remote-url> [--push] [--replace]

Examples:
  scripts/beads_dolt_remote_bootstrap.sh --url https://doltremoteapi.dolthub.com/org/sdplab-beads --push
  scripts/beads_dolt_remote_bootstrap.sh --url file:///tmp/sdplab-beads-remote --push
  BEADS_DOLT_REMOTE_URL=https://doltremoteapi.dolthub.com/org/sdplab-beads \
    scripts/beads_dolt_remote_bootstrap.sh --push

Notes:
  - The remote is configured under the Dolt remote name "origin" by default.
  - Hosted Dolt usually requires DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD.
  - Use an explicit URI scheme. Bare filesystem paths are rejected on purpose.
  - Use --push after the remote is created and authenticated to publish local state.
EOF
}

require_remote_url() {
  if [[ -z "${REMOTE_URL}" ]]; then
    echo "error: missing Dolt remote URL; pass --url or set BEADS_DOLT_REMOTE_URL" >&2
    exit 2
  fi

  if [[ "${REMOTE_URL}" != *"://"* ]]; then
    echo "error: remote URL must use an explicit scheme (for example https://... or file:///...)" >&2
    exit 2
  fi
}

remote_exists() {
  local output
  output="$(bd dolt remote list 2>/dev/null || true)"
  [[ -n "${output}" && "${output}" != "No remotes configured." ]] || return 1
  printf '%s\n' "${output}" | rg -q "(^|[[:space:]])${REMOTE_NAME}([[:space:]]|$)"
}

warn_if_hosted_without_creds() {
  if [[ "${REMOTE_URL}" == https://doltremoteapi.dolthub.com/* ]]; then
    if [[ -z "${DOLT_REMOTE_USER:-}" || -z "${DOLT_REMOTE_PASSWORD:-}" ]]; then
      echo "warning: hosted Dolt remote detected but DOLT_REMOTE_USER/DOLT_REMOTE_PASSWORD are not set" >&2
    fi
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --url)
        [[ $# -ge 2 ]] || { echo "error: --url requires a value" >&2; exit 2; }
        REMOTE_URL="$2"
        shift 2
        ;;
      --push)
        PUSH=true
        shift
        ;;
      --replace)
        REPLACE=true
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "error: unknown arg: $1" >&2
        usage >&2
        exit 2
        ;;
    esac
  done
}

main() {
  parse_args "$@"
  require_remote_url
  warn_if_hosted_without_creds

  if remote_exists; then
    if [[ "${REPLACE}" == true ]]; then
      echo "[beads_dolt_bootstrap] removing existing Dolt remote ${REMOTE_NAME}"
      bd dolt remote remove "${REMOTE_NAME}"
    else
      echo "error: Dolt remote ${REMOTE_NAME} already exists; use --replace to reconfigure it" >&2
      exit 2
    fi
  fi

  echo "[beads_dolt_bootstrap] adding Dolt remote ${REMOTE_NAME}"
  bd dolt remote add "${REMOTE_NAME}" "${REMOTE_URL}"
  bd dolt remote list

  if [[ "${PUSH}" == true ]]; then
    echo "[beads_dolt_bootstrap] pushing local Beads state to ${REMOTE_NAME}"
    bd dolt push
  fi

  "${PROJECT_ROOT}/scripts/beads_transport.sh" status
}

main "$@"
