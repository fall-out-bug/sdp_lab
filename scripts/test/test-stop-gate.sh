#!/usr/bin/env bash
# Smoke tests for scripts/oneshot-stop-gate.sh — F143-01 infinite-loop fix.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
GATE="${REPO_ROOT}/scripts/oneshot-stop-gate.sh"

PASS=0
FAIL=0

assert_exit() {
  local expected="$1" actual="$2" label="$3"
  if [[ "$actual" -eq "$expected" ]]; then
    echo "  ✓ $label (exit=$actual)"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $label (expected=$expected got=$actual)"
    FAIL=$((FAIL + 1))
  fi
}

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/checkpoints" "$scratch/runs"

# Test 1: empty checkpoint dir → exit 0
echo "Test 1: empty checkpoint dir → allow stop"
SDP_CHECKPOINT_DIR="$scratch/checkpoints" SDP_RUNS_DIR="$scratch/runs" \
  "$GATE" </dev/null
assert_exit 0 $? "no checkpoints"
echo

# Test 2: checkpoint without pr_number → exit 0
echo "Test 2: checkpoint without pr_number → allow stop"
cat > "$scratch/checkpoints/F100.json" <<'EOF'
{"feature_id":"F100","phase":1,"step":"build","phase_status":"in_progress"}
EOF
SDP_CHECKPOINT_DIR="$scratch/checkpoints" SDP_RUNS_DIR="$scratch/runs" \
  "$GATE" </dev/null
assert_exit 0 $? "no PR opened"
rm "$scratch/checkpoints/F100.json"
echo

# Test 3: phase_status=done step=pr_created → allow (F143-01 rule 5)
echo "Test 3: phase_status=done step=pr_created → allow stop (rule 5)"
cat > "$scratch/checkpoints/F200.json" <<'EOF'
{"feature_id":"F200","pr_number":200,"phase":2,"step":"pr_created","phase_status":"done"}
EOF
SDP_CHECKPOINT_DIR="$scratch/checkpoints" SDP_RUNS_DIR="$scratch/runs" \
  "$GATE" </dev/null
assert_exit 0 $? "PR opened, local oneshot done"
echo

# Test 4: in-progress checkpoint → block
echo "Test 4: phase_status=in_progress → block stop"
cat > "$scratch/checkpoints/F300.json" <<'EOF'
{"feature_id":"F300","pr_number":300,"phase":1,"step":"build","phase_status":"in_progress"}
EOF
SDP_CHECKPOINT_DIR="$scratch/checkpoints" SDP_RUNS_DIR="$scratch/runs" \
  "$GATE" </dev/null 2>/dev/null
assert_exit 2 $? "in-progress blocks"
echo

# Test 5: stop_hook_active payload → allow even if blocking checkpoint exists
echo "Test 5: stop_hook_active=true → allow stop"
echo '{"stop_hook_active":true}' | \
  SDP_CHECKPOINT_DIR="$scratch/checkpoints" SDP_RUNS_DIR="$scratch/runs" "$GATE"
assert_exit 0 $? "loop-break flag respected"
echo

# Test 6: SDP_STOP_HOOK_BYPASS=1 → allow even if blocking checkpoint exists
echo "Test 6: SDP_STOP_HOOK_BYPASS=1 → allow stop"
SDP_STOP_HOOK_BYPASS=1 SDP_CHECKPOINT_DIR="$scratch/checkpoints" SDP_RUNS_DIR="$scratch/runs" \
  "$GATE" </dev/null
assert_exit 0 $? "manual env-var bypass"
echo

# Test 7: per-checkpoint .stop sticky bypass → allow
echo "Test 7: <checkpoint>.stop sticky bypass → allow stop"
touch "$scratch/checkpoints/F300.json.stop"
SDP_CHECKPOINT_DIR="$scratch/checkpoints" SDP_RUNS_DIR="$scratch/runs" \
  "$GATE" </dev/null
assert_exit 0 $? "per-checkpoint sticky bypass"
echo

echo "Result: $PASS pass, $FAIL fail"
[[ "$FAIL" -eq 0 ]]
