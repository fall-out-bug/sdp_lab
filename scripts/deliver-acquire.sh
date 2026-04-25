#!/usr/bin/env bash
# Acquire delivery slot for a feature: atomic claim + runtime lock file.
#
# Usage: deliver-acquire.sh <FEATURE> <EPIC_ID>
#
# Exit codes:
#   0  acquired (epic claimed; lock file written)
#   1  generic bd error
#   2  foreign claim (epic claimed by a different user)
#   3  lock held by another live process on this host
#  64  usage error
#
# Synchronization model:
#   - `bd update --claim` is the primary CAS (atomic at beads/dolt level).
#   - LOCK_FILE is a side-channel for stale detection and for catching the
#     same-user-multi-shell case before we waste a bd call.
#
# Side effects on success:
#   - bd update <EPIC_ID> --claim has been called (atomic CAS)
#   - <repo>/.sdp/locks/deliver-<FEATURE>.lock contains pid+host+user+ts+feature+epic
#
# Lock release: caller's responsibility (Phase 4 closeout removes the file).
# Stale lock (dead PID, same host) is taken over silently.
# Lock from different host is informational only — bd CAS still gates correctness.

set -uo pipefail

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <FEATURE> <EPIC_ID>" >&2
  exit 64
fi

FEATURE="$1"
EPIC_ID="$2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
LOCK_DIR="${REPO_ROOT}/.sdp/locks"
LOCK_FILE="${LOCK_DIR}/deliver-${FEATURE}.lock"

mkdir -p "${LOCK_DIR}"

HOSTNAME_VAL="$(hostname)"
USER_VAL="${USER:-$(whoami)}"

# Inspect existing lock for live-foreign-pid case (same host only).
if [[ -f "${LOCK_FILE}" ]]; then
  existing_pid="$(awk -F= '$1=="pid"{print $2; exit}' "${LOCK_FILE}" 2>/dev/null || true)"
  existing_host="$(awk -F= '$1=="host"{print $2; exit}' "${LOCK_FILE}" 2>/dev/null || true)"

  if [[ -n "${existing_pid}" && "${existing_host}" == "${HOSTNAME_VAL}" ]] \
     && [[ "${existing_pid}" != "$$" ]] \
     && kill -0 "${existing_pid}" 2>/dev/null; then
    echo "Lock held by live PID ${existing_pid} on ${existing_host} (file: ${LOCK_FILE})" >&2
    exit 3
  fi
  # Otherwise: stale (dead pid), our own pid, or different host — proceed.
fi

# Atomic CAS via bd --claim. Idempotent for same actor.
bd_out="$(bd update "${EPIC_ID}" --claim 2>&1)"; bd_rc=$?

if [[ ${bd_rc} -ne 0 ]]; then
  if echo "${bd_out}" | grep -qi "already claimed by"; then
    echo "Foreign claim on ${EPIC_ID}: ${bd_out}" >&2
    exit 2
  fi
  echo "bd claim error (rc=${bd_rc}): ${bd_out}" >&2
  exit 1
fi

# Write lock metadata atomically (tmp + rename).
TMP_LOCK="${LOCK_FILE}.$$.tmp"
cat > "${TMP_LOCK}" <<EOF
pid=$$
host=${HOSTNAME_VAL}
user=${USER_VAL}
ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
feature=${FEATURE}
epic=${EPIC_ID}
EOF
mv "${TMP_LOCK}" "${LOCK_FILE}"

exit 0
