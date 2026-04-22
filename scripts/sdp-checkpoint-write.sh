#!/usr/bin/env bash
# sdp-checkpoint-write.sh — atomic writer for delivery-loop checkpoint.
#
# Reads a merge-delta JSON from stdin and applies it to the feature checkpoint
# using tmp+rename for atomicity. Only the orchestrator calls this;
# subagents emit structured output that the orchestrator folds in.
#
# Usage:
#   echo '{"phase":2,"step":"impact-review"}' | sdp-checkpoint-write.sh F134
#
# Behavior:
#   - Reads existing .sdp/checkpoints/${FEATURE}.json (or seeds an empty {})
#   - Deep-merges stdin JSON on top (right wins)
#   - Updates last_heartbeat to now
#   - Validates schema_version == 2
#   - Writes to .tmp + rename (atomic on POSIX)
#
# Requires: jq

set -euo pipefail

FEATURE="${1:-}"
if [[ -z "$FEATURE" ]]; then
  echo "Usage: $0 <feature_id>" >&2
  exit 2
fi

CHECKPOINT_DIR="${SDP_CHECKPOINT_DIR:-.sdp/checkpoints}"
mkdir -p "$CHECKPOINT_DIR"
CP_FILE="${CHECKPOINT_DIR}/${FEATURE}.json"

# Read existing (or seed)
if [[ -f "$CP_FILE" ]]; then
  EXISTING="$(cat "$CP_FILE")"
else
  EXISTING='{"schema_version":2}'
fi

# Read delta from stdin
DELTA="$(cat)"

# Deep merge; update heartbeat; validate
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
MERGED="$(jq -n --argjson a "$EXISTING" --argjson b "$DELTA" --arg now "$NOW" '
  ($a * $b)
  | .last_heartbeat = $now
  | .schema_version = 2
')"

# Validate
VER="$(echo "$MERGED" | jq -r '.schema_version // 0')"
if [[ "$VER" != "2" ]]; then
  echo "sdp-checkpoint-write: schema_version must be 2 (got $VER)" >&2
  exit 3
fi

# Atomic write
TMP="${CP_FILE}.tmp.$$"
echo "$MERGED" | jq '.' > "$TMP"
mv "$TMP" "$CP_FILE"

echo "$CP_FILE"
