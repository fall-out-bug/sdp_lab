#!/usr/bin/env bash
# verify_skill_anchors.sh — contract test for skill cross-references.
#
# Skills reference each other by prose anchors ("see 'Session Bootstrap' in
# build.md"). A rename silently breaks the caller. This script greps for the
# required anchors and fails the build if any are missing.
#
# Adding an expectation:
#   EXPECTATIONS+=( "path/to/file.md:^## Required Heading$" )
#
# Exit codes:
#   0 — all anchors present
#   1 — one or more anchors missing (summary printed)
#   2 — script misuse (bad format, file not readable)

set -euo pipefail

ROOT="${SDP_REPO_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
cd "$ROOT"

# FILE:REGEX pairs — regex is ERE, anchors with ^ $ for whole-line match.
EXPECTATIONS=(
  ".agents/skills/build.md|^## Session Bootstrap"
  ".agents/skills/review.md|^## Dimensions"
  ".agents/skills/delivery-loop.md|^## Loop structure"
  ".agents/skills/delivery-loop.md|^## Checkpoint schema v2"
  ".agents/skills/delivery-loop.md|^## Abort & rollback"
  "prompts/commands/deliver.md|^# /deliver"
  "scripts/run_go_quality_gates.sh|^#!"
  "scripts/sdp-dispatch.sh|^dispatch_subagent"
  "scripts/sdp-checkpoint-write.sh|^FEATURE="
  "scripts/traceability-gate.sh|^FEATURE="
  ".agents/skills/delivery-loop.md|^## Phases \(declarative\)"
)

missing=0
checked=0
for spec in "${EXPECTATIONS[@]}"; do
  file="${spec%%|*}"
  pattern="${spec#*|}"
  checked=$((checked + 1))

  if [[ ! -r "$file" ]]; then
    printf 'MISSING FILE: %s (expected anchor: %s)\n' "$file" "$pattern" >&2
    missing=$((missing + 1))
    continue
  fi

  if ! grep -qE "$pattern" "$file"; then
    printf 'MISSING ANCHOR: %s in %s\n' "$pattern" "$file" >&2
    missing=$((missing + 1))
  fi
done

if [[ $missing -gt 0 ]]; then
  printf '\n%d/%d anchor expectations failed.\n' "$missing" "$checked" >&2
  printf 'Update the referenced file, or edit scripts/verify_skill_anchors.sh if the anchor intentionally moved.\n' >&2
  exit 1
fi

printf '✓ all %d skill anchors present\n' "$checked"
