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
  fetch   Pull Beads state from Dolt remote when configured. In git-backup
          mode this is intentionally a no-op.
  export  Push Beads state to Dolt remote when configured, otherwise publish
          the current portable export to the archival backup branch.
  status  Print which transport path will be used.
EOF
}

cleanup_worktree() {
  local dir="${1:-}"
  if [[ -n "${dir}" && -d "${dir}" ]]; then
    git worktree remove --force "${dir}" >/dev/null 2>&1 || rm -rf "${dir}"
  fi
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

prepare_backup_worktree() {
  local dir
  dir="$(mktemp -d "${TMPDIR:-/tmp}/sdplab-beads-backup-XXXXXX")"

  if has_backup_branch; then
    git fetch "${REMOTE}" "${BRANCH}" >/dev/null 2>&1
    git worktree add --detach "${dir}" "refs/remotes/${REMOTE}/${BRANCH}" >/dev/null 2>&1
  else
    git worktree add --detach "${dir}" HEAD >/dev/null 2>&1
    (
      cd "${dir}"
      git checkout --orphan "${BRANCH}" >/dev/null 2>&1
      git rm -rf . >/dev/null 2>&1 || true
    )
  fi

  printf '%s\n' "${dir}"
}

copy_export_into_worktree() {
  local dir="${1}"
  mkdir -p "${dir}/.beads"
  bd export -o "${dir}/.beads/issues.jsonl" >/dev/null
}

export_backup_branch() {
  local dir=""
  dir="$(prepare_backup_worktree)"

  copy_export_into_worktree "${dir}"

  (
    cd "${dir}"
    git add .beads/issues.jsonl
    if git diff --cached --quiet; then
      echo "[beads_transport] backup branch ${REMOTE}/${BRANCH} already up to date"
      exit 0
    fi

    git commit -m "beads backup snapshot" >/dev/null
    git push "${REMOTE}" "HEAD:refs/heads/${BRANCH}" >/dev/null
  )

  cleanup_worktree "${dir}"

  echo "Exported backup snapshot to git branch ${BRANCH}"
  echo "  Remote: ${REMOTE}"
  echo "  Path: .beads/issues.jsonl"
  echo "  Push: complete"
}

fetch_transport() {
  if has_dolt_remote; then
    echo "[beads_transport] pulling from Dolt remote ${DOLT_REMOTE_NAME}"
    bd dolt pull
    return 0
  fi

  if has_backup_branch; then
    echo "[beads_transport] archival backup branch ${REMOTE}/${BRANCH} exists, but git-backup mode does not hydrate local Dolt state"
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
  export_backup_branch
}

status_transport() {
  if has_dolt_remote; then
    echo "mode=dolt-remote remote_name=${DOLT_REMOTE_NAME} remote_configured=true backup_branch=${REMOTE}/${BRANCH}"
    return 0
  fi

  if has_backup_branch; then
    echo "mode=git-backup archival_only=true remote_name=${DOLT_REMOTE_NAME} remote_configured=false backup_branch=${REMOTE}/${BRANCH}"
    return 0
  fi

  echo "mode=local-only archival_only=false remote_name=${DOLT_REMOTE_NAME} remote_configured=false backup_branch=${REMOTE}/${BRANCH}"
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
