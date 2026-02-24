#!/usr/bin/env bash
# Stop hook gate: blocks agent exit when oneshot CI phase is incomplete.
# Reads checkpoint; if pr_number set and last_phase!=ci or last_state!=ok, exit 2.
# Handles stop_hook_active to prevent infinite block loop.

set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECKPOINT_DIR="${SDP_CHECKPOINT_DIR:-${ROOT}/.sdp/checkpoints}"
RUNS_DIR="${SDP_RUNS_DIR:-${ROOT}/.sdp/runs}"

# If stop_hook_active, allow stop (prevent infinite loop)
PAYLOAD=""
if [ -p /dev/stdin ] 2>/dev/null; then
  PAYLOAD="$(cat 2>/dev/null || true)"
fi
if [ -n "$PAYLOAD" ]; then
  ACTIVE="$(echo "$PAYLOAD" | jq -r '.stop_hook_active // false' 2>/dev/null)"
  [ "$ACTIVE" = "true" ] && exit 0
fi

[ ! -d "$CHECKPOINT_DIR" ] && exit 0
for f in "$CHECKPOINT_DIR"/F*.json; do
  [ ! -f "$f" ] && continue
  PR="$(jq -r '.pr_number // empty' "$f")"
  [ -z "$PR" ] && continue
  FID="$(jq -r '.feature_id // empty' "$f")"
  [ -z "$FID" ] && continue
  # Check run file: last_phase=ci and last_state=ok means CI complete
  RUN_FILE="$(ls "$RUNS_DIR"/oneshot-"$FID"-*.json 2>/dev/null | sort -r | head -1)"
  if [ -n "$RUN_FILE" ]; then
    PHASE="$(jq -r '.last_phase // ""' "$RUN_FILE")"
    STATE="$(jq -r '.last_state // ""' "$RUN_FILE")"
    [ "$PHASE" = "ci" ] && [ "$STATE" = "ok" ] && continue
  fi
  echo "CI phase incomplete. Run: sdp-ci-loop --pr $PR --feature $FID"
  exit 2
done
exit 0
