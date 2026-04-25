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
#   - take first
#
# Coordination/meta/program epics MUST be tagged so the picker skips them:
#   bd label add <epic-id> coordination

set -uo pipefail

bd_out="$(bd ready --json -n 200 2>/dev/null)" || {
  echo "deliver-pick: bd ready failed" >&2
  exit 1
}

# Empty or unparseable JSON → bd error (exit 1) only if non-empty + invalid.
# Empty array is valid → exit 4.
if [[ -z "${bd_out}" ]]; then
  echo "deliver-pick: bd ready returned no output" >&2
  exit 1
fi

picked="$(printf '%s' "${bd_out}" | jq -r '
  [.[]
    | select(.issue_type == "epic" or .issue_type == "feature")
    | select(.title | test("^\\[bug\\]") | not)
    | select(.title | test(" ← F") | not)
    | select(((.labels // []) | map(. == "coordination" or . == "meta" or . == "program") | any) | not)
  ]
  | sort_by(.priority, .created_at)
  | .[0]
  | if . == null then empty else "\(.id)\t\(.title)" end
')" || {
  echo "deliver-pick: jq filter failed" >&2
  exit 1
}

if [[ -z "${picked}" ]]; then
  echo "deliver-pick: no deliverable feature in ready queue" >&2
  exit 4
fi

printf '%s\n' "${picked}"
exit 0
