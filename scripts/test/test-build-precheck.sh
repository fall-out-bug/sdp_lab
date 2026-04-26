#!/usr/bin/env bash
# Smoke tests for scripts/hooks/build-precheck.sh — F142-07.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HOOK="${REPO_ROOT}/scripts/hooks/build-precheck.sh"
WS_DIR="${REPO_ROOT}/docs/workstreams/backlog"

cd "$REPO_ROOT"

PASS=0
FAIL=0

assert_exit() {
  local expected="$1" actual="$2" label="$3"
  if [[ "$actual" -eq "$expected" ]]; then
    echo "  ✓ $label (exit=$actual)"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $label (expected exit=$expected, got $actual)"
    FAIL=$((FAIL + 1))
  fi
}

# Test 1: missing ws → exit 1
echo "Test 1: missing ws-id refused"
"$HOOK" 00-999-99 >/dev/null 2>&1
assert_exit 1 $? "00-999-99 refused"
echo

# Test 2: design-pending ws → exit 1
echo "Test 2: design-pending refused"
"$HOOK" 00-082-01 >/dev/null 2>&1
assert_exit 1 $? "00-082-01 (status=design-pending) refused"
echo

# Test 3: open ws → exit 0
echo "Test 3: open ws accepted"
"$HOOK" 00-101-02 >/dev/null 2>&1
assert_exit 0 $? "00-101-02 (status=open) accepted"
echo

# Test 4: bad usage → exit 2
echo "Test 4: missing arg → exit 2"
"$HOOK" >/dev/null 2>&1
assert_exit 2 $? "no argument → exit 2"
echo

# Test 5: error message names the rule (F142-07)
echo "Test 5: error message references F142-07"
err="$("$HOOK" 00-999-99 2>&1)"
if [[ "$err" == *"F142-07"* ]]; then
  echo "  ✓ error mentions F142-07"
  PASS=$((PASS + 1))
else
  echo "  ✗ error missing F142-07 reference"
  echo "    actual: $err"
  FAIL=$((FAIL + 1))
fi
echo

echo "Result: $PASS pass, $FAIL fail"
[[ "$FAIL" -eq 0 ]]
