#!/usr/bin/env bash
# Test oneshot-stop-gate.sh: before CI → blocked; after CI green → allowed
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATE="$ROOT/scripts/oneshot-stop-gate.sh"
TMP="$(mktemp -d)"
trap "rm -rf $TMP" EXIT

export SDP_CHECKPOINT_DIR="$TMP/cp"
export SDP_RUNS_DIR="$TMP/runs"
mkdir -p "$SDP_CHECKPOINT_DIR" "$SDP_RUNS_DIR"

# Test 1: No checkpoint → allow (exit 0)
bash "$GATE"
code=$?
[ $code -ne 0 ] && echo "FAIL: no checkpoint should exit 0, got $code" && exit 1

# Test 2: Checkpoint with pr_number, no run file → block (exit 2)
echo '{"feature_id":"F015","pr_number":42}' > "$SDP_CHECKPOINT_DIR/F015.json"
out=$(bash "$GATE" 2>&1) && code=0 || code=$?
[ $code -ne 2 ] && echo "FAIL: pr set, no run file should exit 2, got $code" && exit 1
echo "$out" | grep -q "sdp-ci-loop" || (echo "FAIL: expected sdp-ci-loop in output"; exit 1)

# Test 3: Checkpoint + run file with last_phase=ci, last_state=ok → allow (exit 0)
echo '{"run_id":"r1","feature_id":"F015","events":[],"last_phase":"ci","last_state":"ok"}' > "$SDP_RUNS_DIR/oneshot-F015-20260223T120000Z.json"
bash "$GATE"
code=$?
[ $code -ne 0 ] && echo "FAIL: CI complete should exit 0, got $code" && exit 1

# Test 4: stop_hook_active=true → allow (exit 0)
echo '{"feature_id":"F015","pr_number":42}' > "$SDP_CHECKPOINT_DIR/F015.json"
rm -f "$SDP_RUNS_DIR"/oneshot-F015-*.json
echo '{"stop_hook_active":true}' | bash "$GATE"
code=$?
[ $code -ne 0 ] && echo "FAIL: stop_hook_active should exit 0, got $code" && exit 1

echo "PASS: oneshot-stop-gate tests"
