#!/usr/bin/env bash
# flaky-detect.sh — Run tests multiple times and detect flaky ones.
# Advisory-only (F129-10). Always exits 0.
#
# Usage:
#   ./scripts/flaky-detect.sh           # default: 3 runs
#   RUNS=5 ./scripts/flaky-detect.sh    # custom run count
#   ./scripts/flaky-detect.sh 5         # custom run count (positional)

set -uo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

RUNS="${RUNS:-${1:-3}}"
TAGS="${GO_TAGS:-sqlite_fts5}"
PACKAGES="${FLAKY_PACKAGES:-./...}"

echo "==> Flaky detection: ${RUNS} run(s), packages=${PACKAGES}"
echo "    Advisory-only. Always exits 0."
echo ""

# Accumulate all JSON output across runs.
COMBINED=""
FAIL_COUNT=0

for i in $(seq 1 "$RUNS"); do
  echo "--- Run ${i}/${RUNS} ---"
  if OUTPUT=$(go test -tags "$TAGS" $PACKAGES -json -count=1 2>&1); then
    COMBINED="${COMBINED}${OUTPUT}"$'\n'
  else
    COMBINED="${COMBINED}${OUTPUT}"$'\n'
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi
done

# Flaky detection using fully qualified Package/Test identifiers.
# Using Package+Test avoids false positives when the same test name exists
# in different packages.
echo ""
echo "==> Analyzing results..."
PASS_TESTS=$(echo "$COMBINED" | grep '"Action":"pass"' | grep -v '"Test":""' | sed 's/.*"Package":"\([^"]*\)".*"Test":"\([^"]*\)".*/\1\/\2/' | sort -u)
FAIL_TESTS=$(echo "$COMBINED" | grep '"Action":"fail"' | grep -v '"Test":""' | sed 's/.*"Package":"\([^"]*\)".*"Test":"\([^"]*\)".*/\1\/\2/' | sort -u)

FLAKY=$(comm -12 <(echo "$PASS_TESTS") <(echo "$FAIL_TESTS") || true)
if [ -n "$FLAKY" ]; then
  echo "Potentially flaky tests (appeared in both pass and fail):"
  echo "$FLAKY" | while read -r t; do
    echo "  - $t"
  done
  echo ""
  echo "Advisory: investigate flaky tests above. Not a CI gate."
else
  echo "No flaky tests detected across ${RUNS} run(s)."
fi

# Mutation advisory (purely informational).
COV_PCT="0.0"
TEST_COUNT="0"
if COV_RAW=$(go test -tags "$TAGS" -coverprofile=/tmp/flaky-cov.out $PACKAGES 2>/dev/null); then
  if COV_LINE=$(echo "$COV_RAW" | grep '^total:' | awk '{print $NF}' | tr -d '%'); then
    COV_PCT="${COV_LINE}"
  fi
fi
# Rough test count from JSON output.
TEST_COUNT=$(echo "$COMBINED" | grep '"Action":"pass"' | grep -v '"Test":""' | sed 's/.*"Package":"\([^"]*\)".*"Test":"\([^"]*\)".*/\1\/\2/' | sort -u | wc -l | tr -d ' ')

echo ""
# Use the Go mutation advisory package for consistent guidance.
TMPFILE=$(mktemp /tmp/mutation-advisory-XXXXXX.go)
cat > "$TMPFILE" <<'GOSRC'
package main

import (
	"fmt"
	"os"
	"strconv"

	"sdp_dev/internal/mutation"
)

func main() {
	cov, _ := strconv.ParseFloat(os.Args[1], 64)
	tests, _ := strconv.Atoi(os.Args[2])
	fmt.Print(mutation.GenerateAdvisory(cov, tests))
}
GOSRC
if go run -tags "$TAGS" "$TMPFILE" "$COV_PCT" "$TEST_COUNT" 2>/dev/null; then
  : # advisory printed by Go package
else
  echo "==> Mutation Testing Advisory (F129-10)"
  echo "    Coverage: ${COV_PCT}% | Unique passing tests: ${TEST_COUNT}"
  echo "    Advisory-only. Not a CI gate."
fi
rm -f "$TMPFILE"
echo ""

echo "==> Flaky detection complete. Runs with failures: ${FAIL_COUNT}/${RUNS}"
echo "==> Exit 0 (advisory-only, never blocking)"
exit 0
