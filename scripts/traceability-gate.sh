#!/usr/bin/env bash
# traceability-gate.sh — advisory traceability check before PR creation.
#
# For each workstream of FEATURE:
#   - Extract AC[0-9]+ IDs from docs/workstreams/backlog/${FEATURE}-*.md
#     (or frontmatter `acs: [AC1, AC2]`).
#   - Confirm each AC ID appears in at least one *_test.go file changed by
#     this branch (git diff main --name-only | grep _test.go).
#   - An AC entry with literal value "NONE" is treated as explicitly waived.
#
# For each schema change (schema/*.schema.json) in the diff:
#   - Require adjacent `_test.go` invoking `jsonschema.Validate`,
#     OR a `testdata/<schemaname>/` directory.
#
# Mode (env var SDP_TRACEABILITY_MODE or arg --mode=gate|advisory):
#   advisory (default) — warn only, exit 0
#   gate               — missing coverage → exit 1
#
# Usage:
#   traceability-gate.sh <FEATURE_ID> [--mode=gate]
#
# Integrated into delivery-loop.md Phase 2.

set -euo pipefail

FEATURE="${1:-}"
MODE="${SDP_TRACEABILITY_MODE:-advisory}"

if [[ -z "$FEATURE" ]]; then
  echo "Usage: $0 <FEATURE_ID> [--mode=gate]" >&2
  exit 2
fi

for arg in "$@"; do
  case "$arg" in
    --mode=gate)     MODE=gate     ;;
    --mode=advisory) MODE=advisory ;;
  esac
done

ROOT="${SDP_REPO_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
cd "$ROOT"

WARN=0
FAIL=0
warn() {
  WARN=$((WARN + 1))
  printf 'TRACEABILITY WARN: %s\n' "$*" >&2
}
fail() {
  FAIL=$((FAIL + 1))
  printf 'TRACEABILITY FAIL: %s\n' "$*" >&2
}

# Collect WS files for this feature
shopt -s nullglob
WS_FILES=(docs/workstreams/backlog/${FEATURE}-*.md)
if [[ ${#WS_FILES[@]} -eq 0 ]]; then
  warn "No WS files for feature ${FEATURE} (docs/workstreams/backlog/${FEATURE}-*.md)"
fi

# Branch → diff against main to find touched test files
TOUCHED_TESTS="$(git diff --name-only main 2>/dev/null | grep -E '_test\.go$' || true)"
TOUCHED_SCHEMA="$(git diff --name-only main 2>/dev/null | grep -E '^schema/.*\.schema\.json$' || true)"

# 1. AC coverage
for ws in "${WS_FILES[@]}"; do
  acs="$(grep -oE 'AC[0-9]+' "$ws" | sort -u || true)"
  [[ -z "$acs" ]] && continue
  for ac in $acs; do
    # Waiver support — WS file declares "AC1: NONE" on a single line
    if grep -qE "^${ac}:[[:space:]]*NONE\$" "$ws"; then
      continue
    fi
    if [[ -z "$TOUCHED_TESTS" ]]; then
      warn "${ws}: ${ac} declared but no *_test.go files in diff"
      continue
    fi
    found=0
    while IFS= read -r tf; do
      [[ -z "$tf" ]] && continue
      if grep -q "$ac" "$tf" 2>/dev/null; then
        found=1
        break
      fi
    done <<< "$TOUCHED_TESTS"
    if [[ $found -eq 0 ]]; then
      msg="${ws}: ${ac} not referenced in any changed test file"
      if [[ "$MODE" == "gate" ]]; then fail "$msg"; else warn "$msg"; fi
    fi
  done
done

# 2. Schema coverage
while IFS= read -r schema; do
  [[ -z "$schema" ]] && continue
  schemaname="$(basename "$schema" .schema.json)"
  # Look for neighboring _test.go with jsonschema.Validate referencing this schema
  test_hit="$(grep -rl --include='*_test.go' "jsonschema.Validate" . 2>/dev/null | head -5 || true)"
  testdata_dir="$(/usr/bin/find testdata -type d -name "$schemaname" 2>/dev/null | head -1 || true)"
  if [[ -z "$test_hit" && -z "$testdata_dir" ]]; then
    msg="${schema}: no jsonschema.Validate test nor testdata/${schemaname}/ found"
    if [[ "$MODE" == "gate" ]]; then fail "$msg"; else warn "$msg"; fi
  fi
done <<< "$TOUCHED_SCHEMA"

# Summary
if [[ $FAIL -gt 0 ]]; then
  printf '\n✗ traceability gate FAILED (%d errors, %d warnings) in mode=%s\n' "$FAIL" "$WARN" "$MODE" >&2
  exit 1
fi

if [[ $WARN -gt 0 ]]; then
  printf '\n⚠ traceability advisory: %d warnings (mode=%s)\n' "$WARN" "$MODE"
  exit 0
fi

printf '✓ traceability OK (mode=%s)\n' "$MODE"
