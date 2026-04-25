#!/usr/bin/env bash
# Test suite for scripts/deliver-acquire.sh
# Run: ./scripts/deliver-acquire_test.sh
# Exit 0 on all pass, non-zero on any fail.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ACQUIRE="${REPO_ROOT}/scripts/deliver-acquire.sh"

PASS=0
FAIL=0
FAILED_TESTS=()

assert_exit() {
  local label="$1"; local expected="$2"; local actual="$3"
  if [[ "${actual}" == "${expected}" ]]; then
    echo "  PASS: ${label} (exit=${actual})"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: ${label} — expected exit=${expected}, got exit=${actual}"
    FAIL=$((FAIL + 1))
    FAILED_TESTS+=("${label}")
  fi
}

require_file() {
  if [[ ! -f "$1" ]]; then
    echo "  FAIL: expected file ${1} to exist"
    FAIL=$((FAIL + 1))
    FAILED_TESTS+=("file_exists:${1}")
  else
    echo "  PASS: file ${1} exists"
    PASS=$((PASS + 1))
  fi
}

# Mocks: bd binary stubbed via PATH override.
MOCK_DIR="$(mktemp -d)"
TEST_LOCK="${REPO_ROOT}/.sdp/locks/deliver-TEST.lock"
trap 'rm -rf "${MOCK_DIR}"; rm -f "${TEST_LOCK}" "${TEST_LOCK}.flock"' EXIT

make_bd_mock() {
  # $1 = exit code, $2 = stderr message
  cat > "${MOCK_DIR}/bd" <<EOF
#!/usr/bin/env bash
if [[ "\${1:-}" == "update" && "\${3:-}" == "--claim" ]]; then
  echo "${2:-}" >&2
  exit ${1:-0}
fi
echo "mock bd: unhandled args: \$*" >&2
exit 99
EOF
  chmod +x "${MOCK_DIR}/bd"
}

cd "${REPO_ROOT}"
mkdir -p .sdp/locks

echo "Test 1: missing args → exit 64"
"${ACQUIRE}" 2>/dev/null; rc=$?
assert_exit "no args" 64 "${rc}"
"${ACQUIRE}" ONLY_FEATURE 2>/dev/null; rc=$?
assert_exit "one arg" 64 "${rc}"

echo
echo "Test 2: successful claim → exit 0, lock file written"
rm -f "${TEST_LOCK}"
make_bd_mock 0 ""
PATH="${MOCK_DIR}:${PATH}" "${ACQUIRE}" TEST sdplab-test 2>/dev/null; rc=$?
assert_exit "clean acquire" 0 "${rc}"
require_file "${TEST_LOCK}"

echo
echo "Test 3: lock file contains PID, host, user, ts, feature, epic"
if [[ -f "${TEST_LOCK}" ]]; then
  content="$(cat "${TEST_LOCK}")"
  for token in "pid=" "host=" "user=" "ts=" "feature=TEST" "epic=sdplab-test"; do
    if [[ "${content}" == *"${token}"* ]]; then
      echo "  PASS: lock contains ${token}"
      PASS=$((PASS + 1))
    else
      echo "  FAIL: lock missing ${token}"
      FAIL=$((FAIL + 1))
      FAILED_TESTS+=("lock_token:${token}")
    fi
  done
fi

echo
echo "Test 4: foreign claim (bd exits 1 with 'already claimed by') → exit 2, no lock file"
rm -f "${TEST_LOCK}"
make_bd_mock 1 "Error claiming sdplab-test: issue already claimed by other-user"
PATH="${MOCK_DIR}:${PATH}" "${ACQUIRE}" TEST sdplab-test 2>/dev/null; rc=$?
assert_exit "foreign claim" 2 "${rc}"
if [[ -f "${TEST_LOCK}" ]]; then
  echo "  FAIL: lock should NOT exist after foreign claim"
  FAIL=$((FAIL + 1))
  FAILED_TESTS+=("lock_leak_on_foreign_claim")
else
  echo "  PASS: no lock leak on foreign claim"
  PASS=$((PASS + 1))
fi

echo
echo "Test 5: generic bd error → exit 1"
rm -f "${TEST_LOCK}"
make_bd_mock 1 "Error: database locked"
PATH="${MOCK_DIR}:${PATH}" "${ACQUIRE}" TEST sdplab-test 2>/dev/null; rc=$?
assert_exit "bd generic error" 1 "${rc}"

echo
echo "Test 6: re-acquire from same PID (idempotent) → exit 0"
rm -f "${TEST_LOCK}"
make_bd_mock 0 ""
PATH="${MOCK_DIR}:${PATH}" "${ACQUIRE}" TEST sdplab-test 2>/dev/null; rc=$?
assert_exit "first acquire" 0 "${rc}"
# Re-acquire from the same shell PID
PATH="${MOCK_DIR}:${PATH}" "${ACQUIRE}" TEST sdplab-test 2>/dev/null; rc=$?
assert_exit "re-acquire same shell pid" 0 "${rc}"

echo
echo "Test 7: existing lock with live foreign PID on same host → exit 3"
rm -f "${TEST_LOCK}"
# Spawn a live sleep, plant lock with that PID
sleep 30 &
LIVE_PID=$!
HOSTNAME_VAL="$(hostname)"
cat > "${TEST_LOCK}" <<EOF
pid=${LIVE_PID}
host=${HOSTNAME_VAL}
user=${USER:-$(whoami)}
ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
feature=TEST
epic=sdplab-test
EOF
make_bd_mock 0 ""
PATH="${MOCK_DIR}:${PATH}" "${ACQUIRE}" TEST sdplab-test 2>/dev/null; rc=$?
assert_exit "lock held by live foreign pid" 3 "${rc}"
kill "${LIVE_PID}" 2>/dev/null
wait "${LIVE_PID}" 2>/dev/null

echo
echo "Test 8: existing lock with dead PID → take over, exit 0"
rm -f "${TEST_LOCK}"
# Find a definitely-dead PID
DEAD_PID=99999
while kill -0 "${DEAD_PID}" 2>/dev/null; do DEAD_PID=$((DEAD_PID + 1)); done
cat > "${TEST_LOCK}" <<EOF
pid=${DEAD_PID}
host=$(hostname)
user=${USER:-$(whoami)}
ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
feature=TEST
epic=sdplab-test
EOF
make_bd_mock 0 ""
PATH="${MOCK_DIR}:${PATH}" "${ACQUIRE}" TEST sdplab-test 2>/dev/null; rc=$?
assert_exit "stale lock with dead pid" 0 "${rc}"

echo
echo "==== Results: ${PASS} passed, ${FAIL} failed ===="
if [[ ${FAIL} -gt 0 ]]; then
  printf 'Failed: %s\n' "${FAILED_TESTS[@]}"
  exit 1
fi
exit 0
