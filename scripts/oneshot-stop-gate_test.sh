#!/usr/bin/env bash
# Test oneshot-stop-gate.sh: before CI → blocked; after CI green → allowed
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATE="$ROOT/scripts/oneshot-stop-gate.sh"
TMP="$(mktemp -d)"
trap "rm -rf $TMP" EXIT

export CHECKPOINT_DIR="$TMP/cp"
export RUNS_DIR="$TMP/runs"
mkdir -p "$CHECKPOINT_DIR" "$RUNS_DIR"

# Override dirs in script - we need to patch or pass env. Simpler: run from TMP and symlink.
# Actually the script uses ROOT derived from its path. We need to run with different dirs.
# Patch: use SDP_CHECKPOINT_DIR and SDP_RUNS_DIR env if set.
# Let me update the script to support env override for testing.
# Simpler: create .sdp in TMP, cd there, and run script with modified path.
# The script uses dirname/.. to get ROOT. If we run from $TMP, ROOT would be $TMP's parent.
# Better: add env var support to the script.
# Actually: create a temp project structure:
#   $TMP/.sdp/checkpoints/
#   $TMP/.sdp/runs/
#   $TMP/scripts/oneshot-stop-gate.sh (copy)
# Then cd $TMP and run scripts/oneshot-stop-gate.sh
# The script does ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" - so ROOT=$TMP. Good.
mkdir -p "$TMP/.sdp/checkpoints" "$TMP/.sdp/runs" "$TMP/scripts"
cp "$GATE" "$TMP/scripts/oneshot-stop-gate.sh"
chmod +x "$TMP/scripts/oneshot-stop-gate.sh"

# Test 1: No checkpoint → allow (exit 0)
(cd "$TMP" && bash scripts/oneshot-stop-gate.sh)
code=$?
[ $code -ne 0 ] && echo "FAIL: no checkpoint should exit 0, got $code" && exit 1

# Test 2: Checkpoint with pr_number, no run file → block (exit 2)
echo '{"feature_id":"F015","pr_number":42}' > "$TMP/.sdp/checkpoints/F015.json"
out=$(cd "$TMP" && bash scripts/oneshot-stop-gate.sh 2>&1) && code=0 || code=$?
[ $code -ne 2 ] && echo "FAIL: pr set, no run file should exit 2, got $code" && exit 1
echo "$out" | grep -q "sdp-ci-loop" || (echo "FAIL: expected sdp-ci-loop in output"; exit 1)

# Test 3: Checkpoint + run file with last_phase=ci, last_state=ok → allow (exit 0)
echo '{"run_id":"r1","feature_id":"F015","events":[],"last_phase":"ci","last_state":"ok"}' > "$TMP/.sdp/runs/oneshot-F015-20260223T120000Z.json"
(cd "$TMP" && bash scripts/oneshot-stop-gate.sh)
code=$?
[ $code -ne 0 ] && echo "FAIL: CI complete should exit 0, got $code" && exit 1

# Test 4: stop_hook_active=true → allow (exit 0)
echo '{"feature_id":"F015","pr_number":42}' > "$TMP/.sdp/checkpoints/F015.json"
rm -f "$TMP/.sdp/runs"/oneshot-F015-*.json
(cd "$TMP" && echo '{"stop_hook_active":true}' | bash scripts/oneshot-stop-gate.sh)
code=$?
[ $code -ne 0 ] && echo "FAIL: stop_hook_active should exit 0, got $code" && exit 1

echo "PASS: oneshot-stop-gate tests"
