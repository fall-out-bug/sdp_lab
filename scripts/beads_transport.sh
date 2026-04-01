#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

REMOTE="${BEADS_BACKUP_REMOTE:-origin}"
BRANCH="${BEADS_BACKUP_BRANCH:-beads-backup}"
DOLT_REMOTE_NAME="${BEADS_DOLT_REMOTE_NAME:-origin}"

usage() {
  cat <<'EOF'
Usage: scripts/beads_transport.sh <fetch|export|status>

Modes:
  fetch   Pull Beads state from Dolt remote when configured, otherwise restore
          from the git-backed backup branch if it exists.
  export  Push Beads state to Dolt remote when configured, otherwise publish
          the current backup snapshot to the git-backed backup branch.
  status  Print which transport path will be used.
EOF
}

has_dolt_remote() {
  local output
  output="$(bd dolt remote list 2>/dev/null || true)"
  [[ -n "${output}" && "${output}" != "No remotes configured." ]] || return 1
  printf '%s\n' "${output}" | rg -q "(^|[[:space:]])${DOLT_REMOTE_NAME}([[:space:]]|$)"
}

has_backup_branch() {
  git ls-remote --exit-code --heads "${REMOTE}" "${BRANCH}" >/dev/null 2>&1
}

fetch_transport() {
  if has_dolt_remote; then
    echo "[beads_transport] pulling from Dolt remote ${DOLT_REMOTE_NAME}"
    bd dolt pull
    return 0
  fi

  if has_backup_branch; then
    echo "[beads_transport] restoring from git backup branch ${REMOTE}/${BRANCH}"
    bd backup fetch-git --remote "${REMOTE}" --branch "${BRANCH}"
    return 0
  fi

  echo "[beads_transport] no Dolt remote and no backup branch ${REMOTE}/${BRANCH}; skipping fetch"
}

export_transport() {
  if has_dolt_remote; then
    echo "[beads_transport] pushing to Dolt remote ${DOLT_REMOTE_NAME}"
    bd dolt push
    return 0
  fi

  echo "[beads_transport] publishing backup snapshot to git branch ${REMOTE}/${BRANCH}"
  bd backup export-git --remote "${REMOTE}" --branch "${BRANCH}"
}

status_transport() {
  if has_dolt_remote; then
    echo "mode=dolt-remote remote_name=${DOLT_REMOTE_NAME} remote_configured=true backup_branch=${REMOTE}/${BRANCH}"
    return 0
  fi

  if has_backup_branch; then
    echo "mode=git-backup remote_name=${DOLT_REMOTE_NAME} remote_configured=false backup_branch=${REMOTE}/${BRANCH}"
    return 0
  fi

  echo "mode=local-only remote_name=${DOLT_REMOTE_NAME} remote_configured=false backup_branch=${REMOTE}/${BRANCH}"
}

main() {
  local cmd="${1:-}"
  case "${cmd}" in
    fetch)
      fetch_transport
      ;;
    export)
      export_transport
      ;;
    status)
      status_transport
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
}

main "$@"
