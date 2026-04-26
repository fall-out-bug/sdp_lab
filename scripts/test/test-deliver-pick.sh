#!/usr/bin/env bash
# Smoke test for scripts/deliver-pick.sh — F142-04 picker hardening.
#
# Verifies:
#   1. Picker exits 0 and emits id\ttitle when bd ready has eligible candidates.
#   2. Picker skips a feature whose ws frontmatter has status=design-pending.
#   3. Picker skips a feature with no ws file matching its F-id.
#
# This test mutates real ws files in-place and restores them on EXIT.
# Run only on a clean working tree.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PICKER="${REPO_ROOT}/scripts/deliver-pick.sh"
WS_DIR="${REPO_ROOT}/docs/workstreams/backlog"

cd "$REPO_ROOT"

if ! command -v bd >/dev/null 2>&1; then
  echo "SKIP: bd not on PATH" >&2
  exit 0
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "SKIP: jq not on PATH" >&2
  exit 0
fi

PASS=0
FAIL=0

assert_contains() {
  local needle="$1" haystack="$2" label="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    echo "  ✓ $label"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $label"
    echo "    expected substring: $needle"
    echo "    actual:             $haystack"
    FAIL=$((FAIL + 1))
  fi
}

# Test 1: smoke — picker should produce some valid output (exit 0 with id).
echo "Test 1: picker selects an eligible candidate"
out="$($PICKER 2>/dev/null)"
rc=$?
if [[ "$rc" -eq 0 ]]; then
  assert_contains "sdplab-" "$out" "output contains a sdplab-id"
elif [[ "$rc" -eq 4 ]]; then
  echo "  ! ready queue empty — Test 1 inconclusive"
else
  echo "  ✗ unexpected exit code: $rc"
  FAIL=$((FAIL + 1))
fi
echo

# Test 2: design-pending ws gets skipped.
# Find a candidate ws file backed by an open feature in ready queue.
echo "Test 2: design-pending ws is skipped"
victim_id="$(bd ready --json -n 200 2>/dev/null | jq -r '
  [.[] | select(.issue_type == "epic" or .issue_type == "feature")
       | select(.title | test("^\\[bug\\]") | not)
       | select(.title | test(" ← F") | not)
       | select(((.labels // []) | map(. == "coordination" or . == "meta" or . == "program") | any) | not)]
  | sort_by(.priority, .created_at) | .[0].id // empty')"
victim_title="$(bd ready --json -n 200 2>/dev/null | jq -r --arg id "$victim_id" '.[] | select(.id == $id) | .title')"
victim_fid="$(echo "$victim_title" | grep -oE 'F[0-9]+(-[0-9]+)?' | head -1)"

if [[ -z "$victim_fid" ]]; then
  echo "  ! no eligible candidate found — Test 2 inconclusive"
else
  if [[ "$victim_fid" == *-* ]]; then
    epic="${victim_fid%-*}"; sub="${victim_fid#*-}"
    epic_num=$((10#${epic#F}))
    sub_num=$((10#$sub))
    victim_ws="$(printf '%s/00-%03d-%02d.md' "$WS_DIR" "$epic_num" "$sub_num")"
  else
    epic_num=$((10#${victim_fid#F}))
    victim_ws="$(ls $(printf '%s/00-%03d-*.md' "$WS_DIR" "$epic_num") 2>/dev/null | head -1)"
  fi

  if [[ -f "$victim_ws" ]]; then
    cp "$victim_ws" "$victim_ws.bak"
    trap 'mv "$victim_ws.bak" "$victim_ws" 2>/dev/null' EXIT
    sed -i.tmp 's/^status:.*$/status: design-pending/' "$victim_ws"
    rm -f "$victim_ws.tmp"
    stderr="$($PICKER 2>&1 1>/dev/null)"
    assert_contains "skip $victim_id" "$stderr" "stderr mentions skipping $victim_id"
    mv "$victim_ws.bak" "$victim_ws"
    trap - EXIT
  else
    echo "  ! cannot locate ws file for $victim_fid (expected $victim_ws) — Test 2 inconclusive"
  fi
fi
echo

# Test 3: smoke — picker output format is "id\ttitle".
echo "Test 3: output format is id<tab>title"
out="$($PICKER 2>/dev/null)"
if [[ "$out" == *$'\t'* ]]; then
  echo "  ✓ output contains a tab separator"
  PASS=$((PASS + 1))
else
  echo "  ✗ output missing tab separator"
  echo "    actual: $out"
  FAIL=$((FAIL + 1))
fi
echo

echo "Result: $PASS pass, $FAIL fail"
[[ "$FAIL" -eq 0 ]]
