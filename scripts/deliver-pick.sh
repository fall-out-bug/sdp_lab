#!/usr/bin/env bash
# Pick the next deliverable feature epic from `bd ready`.
#
# Usage: deliver-pick.sh
# Output (on stdout): "<EPIC_ID>\t<TITLE>"
#
# Exit codes:
#   0  picked one (id + title on stdout)
#   1  bd command failed
#   4  no deliverable feature in ready queue
#
# Selection rules (deterministic, no agent reasoning):
#   - issue_type ∈ {epic, feature}
#   - title does NOT start with "[bug]"
#   - title does NOT contain " ← F"   (workstream-leaf marker)
#   - labels does NOT contain any of: coordination, meta, program
#   - sort: priority ASC (0=critical first), then created_at ASC (oldest first)
#   - F142-04: candidate MUST have at least one ws file in
#     docs/workstreams/backlog/00-NNN[-MM].md; if its ws frontmatter
#     declares status=design-pending, skip.
#   - take first surviving candidate
#
# Coordination/meta/program epics MUST be tagged so the picker skips them:
#   bd label add <epic-id> coordination

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WS_DIR="${REPO_ROOT}/docs/workstreams/backlog"

# Extract feature ID (e.g. F141, F101-02) from a title like
# "F141: Multi-harness install" or "F101-02: Extend Write Plan ...".
extract_fid() {
  echo "$1" | grep -oE 'F[0-9]+(-[0-9]+)?' | head -1
}

# Translate a feature ID into the ws-file glob, e.g.
# F141    -> docs/workstreams/backlog/00-141-*.md
# F101-02 -> docs/workstreams/backlog/00-101-02.md
ws_glob_for_fid() {
  local fid="$1"
  if [[ "$fid" == *-* ]]; then
    local epic="${fid%-*}"
    local sub="${fid#*-}"
    local epic_num="${epic#F}"
    epic_num=$((10#$epic_num))
    sub=$((10#$sub))
    printf '%s/00-%03d-%02d.md' "$WS_DIR" "$epic_num" "$sub"
  else
    local epic_num="${fid#F}"
    epic_num=$((10#$epic_num))
    printf '%s/00-%03d-*.md' "$WS_DIR" "$epic_num"
  fi
}

# Read `status:` from a ws file frontmatter. Empty if no frontmatter
# or no status field (treat as "open" by default).
ws_status() {
  local file="$1"
  awk '
    BEGIN { in_fm=0; found=0 }
    /^---$/ { in_fm=!in_fm; next }
    in_fm && /^status:/ {
      sub(/^status:[[:space:]]*/, "")
      print
      found=1
      exit
    }
  ' "$file"
}

bd_out="$(bd ready --json -n 200 2>/dev/null)" || {
  echo "deliver-pick: bd ready failed" >&2
  exit 1
}

if [[ -z "${bd_out}" ]]; then
  echo "deliver-pick: bd ready returned no output" >&2
  exit 1
fi

# Sorted candidate list (id\ttitle per line) after coarse jq filter.
candidates="$(printf '%s' "${bd_out}" | jq -r '
  [.[]
    | select(.issue_type == "epic" or .issue_type == "feature")
    | select(.title | test("^\\[bug\\]") | not)
    | select(.title | test(" ← F") | not)
    | select(((.labels // []) | map(. == "coordination" or . == "meta" or . == "program") | any) | not)
  ]
  | sort_by(.priority, .created_at)
  | .[]
  | "\(.id)\t\(.title)"
')" || {
  echo "deliver-pick: jq filter failed" >&2
  exit 1
}

if [[ -z "${candidates}" ]]; then
  echo "deliver-pick: no deliverable feature in ready queue" >&2
  exit 4
fi

# Iterate candidates in priority order; require ws scaffold + non-design-pending.
while IFS=$'\t' read -r bead_id title; do
  [[ -z "${bead_id}" ]] && continue
  fid="$(extract_fid "${title}")"
  if [[ -z "${fid}" ]]; then
    # No F-id in title → not a workstream feature; skip.
    continue
  fi
  glob="$(ws_glob_for_fid "${fid}")"
  # shellcheck disable=SC2086
  ws_files=( $(ls $glob 2>/dev/null) )
  if [[ ${#ws_files[@]} -eq 0 ]]; then
    # F142-04: no ws scaffold → picker bait. Skip.
    echo "deliver-pick: skip ${bead_id} (${fid}) — no ws file matches ${glob}" >&2
    continue
  fi
  status="$(ws_status "${ws_files[0]}")"
  if [[ "${status}" == "design-pending" ]]; then
    echo "deliver-pick: skip ${bead_id} (${fid}) — ws status=design-pending" >&2
    continue
  fi
  printf '%s\t%s\n' "${bead_id}" "${title}"
  exit 0
done <<< "${candidates}"

echo "deliver-pick: no deliverable feature in ready queue (after ws scaffold filter)" >&2
exit 4
