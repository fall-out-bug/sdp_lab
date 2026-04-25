#!/usr/bin/env bash
# Test suite for scripts/deliver-pick.sh
# Run: ./scripts/deliver-pick_test.sh
# Exit 0 on all pass, non-zero on any fail.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PICK="${REPO_ROOT}/scripts/deliver-pick.sh"

PASS=0
FAIL=0
FAILED_TESTS=()

assert_eq() {
  local label="$1"; local expected="$2"; local actual="$3"
  if [[ "${actual}" == "${expected}" ]]; then
    echo "  PASS: ${label}"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: ${label} — expected '${expected}', got '${actual}'"
    FAIL=$((FAIL + 1))
    FAILED_TESTS+=("${label}")
  fi
}

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

MOCK_DIR="$(mktemp -d)"
trap 'rm -rf "${MOCK_DIR}"' EXIT

# Mock bd by writing a fixture-driven stub.
make_bd_mock() {
  local fixture="$1"
  cat > "${MOCK_DIR}/bd" <<EOF
#!/usr/bin/env bash
if [[ "\${1:-}" == "ready" ]]; then
  cat <<'JSON'
${fixture}
JSON
  exit 0
fi
echo "mock bd: unhandled args: \$*" >&2
exit 99
EOF
  chmod +x "${MOCK_DIR}/bd"
}

run_pick() {
  PATH="${MOCK_DIR}:${PATH}" "${PICK}" 2>/dev/null
}

echo "Test 1: pick first epic by priority then created_at"
make_bd_mock '[
  {"id":"sdplab-aaa","title":"F100: feature one","issue_type":"epic","priority":1,"created_at":"2026-04-20T10:00:00Z","labels":[]},
  {"id":"sdplab-bbb","title":"F101: feature two","issue_type":"epic","priority":0,"created_at":"2026-04-21T10:00:00Z","labels":[]}
]'
out="$(run_pick)"; rc=$?
assert_exit "happy path" 0 "${rc}"
assert_eq "P0 wins over P1" "sdplab-bbb"$'\t'"F101: feature two" "${out}"

echo
echo "Test 2: tie-break by created_at (older first)"
make_bd_mock '[
  {"id":"sdplab-ccc","title":"F102: newer","issue_type":"epic","priority":1,"created_at":"2026-04-22T10:00:00Z","labels":[]},
  {"id":"sdplab-ddd","title":"F103: older","issue_type":"epic","priority":1,"created_at":"2026-04-20T10:00:00Z","labels":[]}
]'
out="$(run_pick)"; rc=$?
assert_eq "older wins on equal priority" "sdplab-ddd"$'\t'"F103: older" "${out}"

echo
echo "Test 3: skip type=task (workstream-leaf)"
make_bd_mock '[
  {"id":"sdplab-eee","title":"F140-01: trace schema","issue_type":"task","priority":1,"created_at":"2026-04-19T10:00:00Z","labels":[]},
  {"id":"sdplab-fff","title":"F104: real epic","issue_type":"epic","priority":1,"created_at":"2026-04-20T10:00:00Z","labels":[]}
]'
out="$(run_pick)"; rc=$?
assert_eq "task is skipped, epic picked" "sdplab-fff"$'\t'"F104: real epic" "${out}"

echo
echo "Test 4: skip type=bug"
make_bd_mock '[
  {"id":"sdplab-ggg","title":"[bug] something broken","issue_type":"bug","priority":0,"created_at":"2026-04-19T10:00:00Z","labels":[]},
  {"id":"sdplab-hhh","title":"F105: feature","issue_type":"epic","priority":1,"created_at":"2026-04-20T10:00:00Z","labels":[]}
]'
out="$(run_pick)"; rc=$?
assert_eq "bug skipped" "sdplab-hhh"$'\t'"F105: feature" "${out}"

echo
echo "Test 5: skip title containing ' ← F' (workstream marker)"
make_bd_mock '[
  {"id":"sdplab-iii","title":"F106-01: design ← F106","issue_type":"epic","priority":0,"created_at":"2026-04-19T10:00:00Z","labels":[]},
  {"id":"sdplab-jjj","title":"F106: feature root","issue_type":"epic","priority":1,"created_at":"2026-04-20T10:00:00Z","labels":[]}
]'
out="$(run_pick)"; rc=$?
assert_eq "WS-marker skipped" "sdplab-jjj"$'\t'"F106: feature root" "${out}"

echo
echo "Test 6: skip label=coordination (meta epics)"
make_bd_mock '[
  {"id":"sdplab-kkk","title":"F1.0 release readiness","issue_type":"epic","priority":0,"created_at":"2026-04-19T10:00:00Z","labels":["coordination"]},
  {"id":"sdplab-lll","title":"F107: feature","issue_type":"epic","priority":1,"created_at":"2026-04-20T10:00:00Z","labels":[]}
]'
out="$(run_pick)"; rc=$?
assert_eq "coordination skipped" "sdplab-lll"$'\t'"F107: feature" "${out}"

echo
echo "Test 7: skip label=meta and program too"
make_bd_mock '[
  {"id":"sdplab-mmm","title":"meta thing","issue_type":"epic","priority":0,"created_at":"2026-04-19T10:00:00Z","labels":["meta"]},
  {"id":"sdplab-nnn","title":"program thing","issue_type":"epic","priority":0,"created_at":"2026-04-19T11:00:00Z","labels":["program"]},
  {"id":"sdplab-ooo","title":"F108: feature","issue_type":"epic","priority":1,"created_at":"2026-04-20T10:00:00Z","labels":[]}
]'
out="$(run_pick)"; rc=$?
assert_eq "meta+program skipped" "sdplab-ooo"$'\t'"F108: feature" "${out}"

echo
echo "Test 8: include type=feature (not just epic)"
make_bd_mock '[
  {"id":"sdplab-ppp","title":"F109: typed-feature","issue_type":"feature","priority":1,"created_at":"2026-04-20T10:00:00Z","labels":[]}
]'
out="$(run_pick)"; rc=$?
assert_exit "feature type accepted" 0 "${rc}"
assert_eq "feature picked" "sdplab-ppp"$'\t'"F109: typed-feature" "${out}"

echo
echo "Test 9: empty input → exit 4"
make_bd_mock '[]'
out="$(run_pick)"; rc=$?
assert_exit "empty ready list" 4 "${rc}"

echo
echo "Test 10: only filtered-out items → exit 4"
make_bd_mock '[
  {"id":"sdplab-qqq","title":"[bug] x","issue_type":"bug","priority":0,"created_at":"2026-04-19T10:00:00Z","labels":[]},
  {"id":"sdplab-rrr","title":"F1.0","issue_type":"epic","priority":0,"created_at":"2026-04-19T11:00:00Z","labels":["coordination"]}
]'
out="$(run_pick)"; rc=$?
assert_exit "all filtered out" 4 "${rc}"

echo
echo "Test 11: missing labels field treated as empty"
make_bd_mock '[
  {"id":"sdplab-sss","title":"F110: no labels field","issue_type":"epic","priority":1,"created_at":"2026-04-20T10:00:00Z"}
]'
out="$(run_pick)"; rc=$?
assert_eq "missing labels OK" "sdplab-sss"$'\t'"F110: no labels field" "${out}"

echo
echo "Test 12: bd error → exit 1"
cat > "${MOCK_DIR}/bd" <<'EOF'
#!/usr/bin/env bash
echo "bd: database locked" >&2
exit 5
EOF
chmod +x "${MOCK_DIR}/bd"
run_pick; rc=$?
assert_exit "bd failure" 1 "${rc}"

echo
echo "==== Results: ${PASS} passed, ${FAIL} failed ===="
if [[ ${FAIL} -gt 0 ]]; then
  printf 'Failed: %s\n' "${FAILED_TESTS[@]}"
  exit 1
fi
exit 0
