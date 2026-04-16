#!/usr/bin/env bash
# bd_post_close.sh — Post-`bd close` auto-sync hook
#
# Moves completed workstream files from backlog/ to done/, updates frontmatter
# status to "done", and updates INDEX.md ranges.
#
# Usage:
#   bd close <id> 2>&1 | scripts/bd_post_close.sh
#   echo "closed sdplab-abc1" | scripts/bd_post_close.sh
#   scripts/bd_post_close.sh sdplab-abc1 sdplab-def2   # bead IDs as args
#
# Environment:
#   BD_POST_CLOSE_DRY_RUN=1   List intended changes without applying
#   REPO_ROOT                 Override repo root (auto-detected if unset)
set -uo pipefail

# ---------------------------------------------------------------------------
# Resolve repo root
# ---------------------------------------------------------------------------
if [ -z "${REPO_ROOT:-}" ]; then
  REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fi

WS_BACKLOG="${REPO_ROOT}/docs/workstreams/backlog"
WS_DONE="${REPO_ROOT}/docs/workstreams/done"
INDEX_FILE="${REPO_ROOT}/docs/workstreams/INDEX.md"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log()  { printf '[bd-post-close] %s\n' "$*"; }
warn() { printf '[bd-post-close] WARN: %s\n' "$*" >&2; }
dry()  { [ "${BD_POST_CLOSE_DRY_RUN:-0}" = "1" ]; }

# Extract bead IDs from stdin or args matching sdplab-<alphanum>
# Returns one ID per line, deduplicated.  Returns empty (no error) if none.
extract_bead_ids() {
  local raw
  if [ $# -gt 0 ]; then
    raw=$(printf '%s\n' "$@")
  else
    raw=$(cat)
  fi
  # grep returns 1 on no match — tolerate that
  { printf '%s\n' "$raw" | grep -oE 'sdplab-[a-z0-9]+' || true; } | sort -u
}

# Given a bead ID, find workstream files in backlog/ whose ## Beads section
# references that ID.  Prints matching filenames (basename only), one per line.
find_ws_for_bead() {
  local bead_id="$1"
  [ -d "$WS_BACKLOG" ] || return 0

  local f
  for f in "$WS_BACKLOG"/*.md; do
    [ -f "$f" ] || continue
    # Use flag-based awk to capture ## Beads section through EOF
    if awk "
      /^## Beads/ { in_beads=1; next }
      /^## / && in_beads { in_beads=0 }
      in_beads { print }
    " "$f" 2>/dev/null | grep -qF "$bead_id"; then
      basename "$f"
    fi
  done | sort -u
}

# Update frontmatter status field from current value to "done"
update_frontmatter_status() {
  local filepath="$1"
  # Use awk for portability (macOS sed -i differs from GNU)
  local tmp
  tmp=$(mktemp)
  awk '
    NR==1 && /^---$/ { in_fm=1 }
    in_fm && /^status:/ { print "status: done"; next }
    /^---$/ && NR>1 && in_fm { in_fm=0 }
    { print }
  ' "$filepath" > "$tmp" && mv "$tmp" "$filepath"
}

# Update INDEX.md for a moved workstream:
#   1. In per-feature detail tables, change the specific WS row from Backlog → Done
#   2. In the feature-level summary table, update the status column if all WS done
update_index() {
  local ws_file="$1"   # e.g. "00-124-05.md"
  local ws_id="${ws_file%.md}"  # e.g. "00-124-05"
  local feature_id

  # Extract feature_id from the moved workstream file's frontmatter
  feature_id=$(grep -m1 '^feature_id:' "${WS_DONE}/${ws_file}" 2>/dev/null | sed 's/feature_id:[[:space:]]*//' || true)
  if [ -z "$feature_id" ]; then
    warn "Cannot determine feature_id for ${ws_id}; skipping INDEX.md update"
    return 0
  fi

  local feature_num="${feature_id#F}"
  local ws_prefix
  ws_prefix=$(printf '%03d' "$feature_num")

  if ! [ -f "$INDEX_FILE" ]; then
    warn "INDEX.md not found at ${INDEX_FILE}; skipping update"
    return 0
  fi

  # --- Update per-feature detail table row ---
  # Expected INDEX.md detail-table format (columns 1-4 are positional):
  #   | WS          | Feature | Title     | Status  |
  #   |-------------|---------|-----------|---------|
  #   | 00-124-05   | F124    | Some Title| Backlog |
  # The regex matches the ws_id in column 1, feature_id in column 2, captures
  # the title column(s), and replaces the status column.  Extra whitespace
  # around pipes is tolerated so both "| F124 |" and "|  F124  |" work.
  if dry; then
    log "DRY-RUN: would update ${ws_id} row in INDEX.md detail table: Backlog → Done"
  else
    local tmp_index
    tmp_index=$(mktemp)
    sed -E "s/\|[[:space:]]*${ws_id}[[:space:]]*\|[[:space:]]*${feature_id}[[:space:]]*\|[[:space:]]*(.*)[[:space:]]+\|[[:space:]]*Backlog[[:space:]]*\|/| ${ws_id} | ${feature_id} | \1 | Done |/" \
      "$INDEX_FILE" > "$tmp_index" && mv "$tmp_index" "$INDEX_FILE"
    log "Updated ${ws_id} in INDEX.md detail table: Backlog → Done"
  fi

  # --- Update feature-level summary status ---
  # Find all workstream files for this feature in both backlog and done
  local all_ws="" done_ws=""
  local f

  for f in "${WS_BACKLOG}"/*.md "${WS_DONE}"/*.md; do
    [ -f "$f" ] || continue
    local bname
    bname=$(basename "$f")
    case "$bname" in
      00-${ws_prefix}-[0-9][0-9].md|00-${ws_prefix}-[0-9].md)
        all_ws="${all_ws} ${bname}"
        if grep -q '^status: done' "$f" 2>/dev/null; then
          done_ws="${done_ws} ${bname}"
        fi
        ;;
    esac
  done

  # Count unique items
  local total_ws done_count
  total_ws=$(echo "$all_ws" | tr ' ' '\n' | sort -u | grep -c '.')
  done_count=$(echo "$done_ws" | tr ' ' '\n' | sort -u | grep -c '.' || true)

  local new_status="Backlog"
  if [ "$total_ws" -gt 0 ] && [ "$done_count" -ge "$total_ws" ]; then
    new_status="Done"
  fi

  # Update the feature summary line:
  # Expected format:
  #   | Feature      | Description | Workstreams              | Status  | Priority |
  #   |--------------|-------------|--------------------------|---------|----------|
  #   | **F124**     | ...         | 00-124-01 ... 00-124-05  | Backlog | P1       |
  # The last status-ish column (word-only between pipes) on the matching row is
  # replaced.  Extra whitespace is tolerated around pipes.
  if dry; then
    log "DRY-RUN: would update ${feature_id} summary status in INDEX.md to ${new_status}"
  else
    local tmp_idx
    tmp_idx=$(mktemp)
    # Replace status column on the feature summary row
    sed "/|[[:space:]]*\*\*${feature_id}\*\*[[:space:]]*|/s/|[[:space:]]*[A-Za-z]*[[:space:]]*|/| ${new_status} |/" \
      "$INDEX_FILE" > "$tmp_idx" && mv "$tmp_idx" "$INDEX_FILE"
    log "Updated ${feature_id} summary status in INDEX.md to ${new_status}"
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  local bead_ids moved=0

  # Collect bead IDs from args or stdin
  bead_ids=$(extract_bead_ids "$@")
  if [ -z "$bead_ids" ]; then
    log "No bead IDs found in input; nothing to do"
    return 0
  fi

  log "Processing bead IDs: $(echo "$bead_ids" | tr '\n' ' ')"

  # For each bead ID, find matching workstream files
  local ws_files=""
  local bid
  for bid in $bead_ids; do
    local matches
    matches=$(find_ws_for_bead "$bid")
    if [ -n "$matches" ]; then
      ws_files="${ws_files} ${matches}"
      log "Bead ${bid} → matched: $(echo "$matches" | tr '\n' ' ')"
    else
      log "Bead ${bid} → no matching workstream"
    fi
  done

  if [ -z "$ws_files" ]; then
    log "No workstream files matched; nothing to move"
    return 0
  fi

  # Deduplicate
  ws_files=$(echo "$ws_files" | tr ' ' '\n' | sort -u | grep -v '^$')

  # Create done/ directory if needed
  if ! dry; then
    mkdir -p "$WS_DONE"
  fi

  # Move each matched workstream file
  local ws
  for ws in $ws_files; do
    local src="${WS_BACKLOG}/${ws}"
    local dst="${WS_DONE}/${ws}"

    if [ ! -f "$src" ]; then
      warn "Source file not found: ${src} (already moved?)"
      continue
    fi

    if [ -f "$dst" ]; then
      warn "Destination already exists: ${dst} (skipping to avoid overwrite)"
      continue
    fi

    if dry; then
      log "DRY-RUN: would move ${ws} backlog/ → done/"
    else
      # Update frontmatter status before moving
      update_frontmatter_status "$src"
      mv "$src" "$dst"
      log "Moved ${ws} → done/"
    fi

    # Update INDEX.md (works in both dry-run and normal mode)
    update_index "$ws"
    moved=$((moved + 1))
  done

  if dry; then
    log "DRY-RUN complete: ${moved} file(s) would be moved"
  else
    log "Done: ${moved} file(s) moved to done/"
  fi
}

main "$@"
