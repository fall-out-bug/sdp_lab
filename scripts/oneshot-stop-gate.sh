#!/usr/bin/env bash
# Stop hook gate: blocks agent exit when oneshot CI phase is incomplete.
#
# Reads each checkpoint in .sdp/checkpoints/ and decides whether the agent
# should keep working (exit 2) or is allowed to stop (exit 0).
#
# F143-01: hardened against infinite loop. Previously, if a checkpoint had
# pr_number set but no matching run file, the gate exited 2 every Stop event,
# and the harness's stop_hook_active flag was sometimes not propagated, so
# the agent re-entered indefinitely.
#
# New rules (any one is enough to allow stop for a given checkpoint):
#   1. SDP_STOP_HOOK_BYPASS=1 in env (manual escape valve).
#   2. .stop file alongside the checkpoint (sticky bypass for one feature).
#   3. payload.stop_hook_active == true.
#   4. PR is merged or closed on GitHub (gh CLI, optional best-effort).
#   5. checkpoint.phase_status == "done" AND step ∈ {pr_created, ci_done}.
#      The local oneshot is finished; CI is on remote and the agent has no
#      pending local action — looping the agent here is pure waste.
#   6. run file exists and reports last_phase=ci, last_state=ok.

set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECKPOINT_DIR="${SDP_CHECKPOINT_DIR:-${ROOT}/.sdp/checkpoints}"
RUNS_DIR="${SDP_RUNS_DIR:-${ROOT}/.sdp/runs}"

# 1. Manual env-var bypass — one-shot escape for a stuck loop.
if [ "${SDP_STOP_HOOK_BYPASS:-}" = "1" ]; then
  exit 0
fi

# 3. Read payload from stdin and respect stop_hook_active to prevent loop.
PAYLOAD=""
if ! [ -t 0 ]; then
  PAYLOAD="$(cat 2>/dev/null || true)"
fi
if [ -n "$PAYLOAD" ]; then
  ACTIVE="$(echo "$PAYLOAD" | jq -r '.stop_hook_active // false' 2>/dev/null)"
  [ "$ACTIVE" = "true" ] && exit 0
fi

[ ! -d "$CHECKPOINT_DIR" ] && exit 0
for f in "$CHECKPOINT_DIR"/F*.json; do
  [ ! -f "$f" ] && continue

  # 2. Per-checkpoint sticky bypass.
  if [ -f "${f}.stop" ]; then
    continue
  fi

  PR="$(jq -r '.pr_number // empty' "$f")"
  [ -z "$PR" ] && continue
  FID="$(jq -r '.feature_id // empty' "$f")"
  [ -z "$FID" ] && continue

  # 5. Local oneshot finished — PR opened or CI step is done.
  PHASE_STATUS="$(jq -r '.phase_status // ""' "$f")"
  STEP="$(jq -r '.step // ""' "$f")"
  if [ "$PHASE_STATUS" = "done" ] && { [ "$STEP" = "pr_created" ] || [ "$STEP" = "ci_done" ]; }; then
    continue
  fi

  # 6. Run file says CI is ok.
  RUN_FILE="$(ls "$RUNS_DIR"/oneshot-"$FID"-*.json 2>/dev/null | sort -r | head -1)"
  if [ -n "$RUN_FILE" ]; then
    P="$(jq -r '.last_phase // ""' "$RUN_FILE")"
    S="$(jq -r '.last_state // ""' "$RUN_FILE")"
    [ "$P" = "ci" ] && [ "$S" = "ok" ] && continue
  fi

  # 4. PR is already closed/merged on the remote (best-effort, no hard fail).
  if command -v gh >/dev/null 2>&1; then
    PR_STATE="$(gh pr view "$PR" --json state --jq .state 2>/dev/null || echo "")"
    case "$PR_STATE" in
      MERGED|CLOSED) continue ;;
    esac
  fi

  cat >&2 <<EOF
oneshot-stop-gate: feature $FID has open PR #$PR but local CI loop has not
completed (phase_status=$PHASE_STATUS step=$STEP). To unblock:
  - run: sdp-ci-loop --pr $PR --feature $FID
  - or:  touch $f.stop          (skip this checkpoint)
  - or:  SDP_STOP_HOOK_BYPASS=1 (skip all checkpoints, this session)
EOF
  exit 2
done
exit 0
