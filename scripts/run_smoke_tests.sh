#!/usr/bin/env bash
# Run CLI smoke scenarios and emit a structured JSON report.
# Usage: ./scripts/run_smoke_tests.sh [--json]
# Exit: 0 = all pass, 1 = failures, 2 = build error

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
JSON_OUTPUT="${SDP_SMOKE_JSON:-0}"
if [[ "${1:-}" == "--json" ]]; then JSON_OUTPUT=1; fi

cd "$PROJECT_ROOT"

REPORT_FILE="$(mktemp /tmp/sdp-smoke-XXXXXX.json)"
trap 'rm -f "$REPORT_FILE"' EXIT

START_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Run smoke tests, capturing full output
if go test -tags=smoke ./test/smoke/... -v -json 2>&1 \
   | tee /tmp/sdp-smoke-raw.json \
   | go run golang.org/x/tools/cmd/godoc@latest 2>/dev/null \
   || true; then : ; fi

# Parse go test -json output into a structured report
go test -tags=smoke ./test/smoke/... -v 2>&1 | tee /tmp/sdp-smoke-stdout.txt
SMOKE_EXIT=${PIPESTATUS[0]:-$?}

END_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Extract pass/fail counts
PASS_COUNT=$(grep -c "^--- PASS:" /tmp/sdp-smoke-stdout.txt 2>/dev/null || echo 0)
FAIL_COUNT=$(grep -c "^--- FAIL:" /tmp/sdp-smoke-stdout.txt 2>/dev/null || echo 0)
SKIP_COUNT=$(grep -c "^--- SKIP:" /tmp/sdp-smoke-stdout.txt 2>/dev/null || echo 0)

# Build per-test entries
TESTS_JSON=""
while IFS= read -r line; do
  if [[ "$line" =~ ^---\ (PASS|FAIL|SKIP):\ (.+)\ \((.+)\)$ ]]; then
    status="${BASH_REMATCH[1]}"
    name="${BASH_REMATCH[2]}"
    duration="${BASH_REMATCH[3]}"
    entry="{\"name\":\"$name\",\"status\":\"$status\",\"duration\":\"$duration\"}"
    TESTS_JSON="${TESTS_JSON:+$TESTS_JSON,}$entry"
  fi
done < /tmp/sdp-smoke-stdout.txt

STATUS="pass"
if [[ "$FAIL_COUNT" -gt 0 ]]; then STATUS="fail"; fi

cat > "$REPORT_FILE" <<EOF
{
  "runner": "sdp-smoke",
  "started_at": "$START_TS",
  "finished_at": "$END_TS",
  "status": "$STATUS",
  "summary": {
    "total": $((PASS_COUNT + FAIL_COUNT + SKIP_COUNT)),
    "pass":  $PASS_COUNT,
    "fail":  $FAIL_COUNT,
    "skip":  $SKIP_COUNT
  },
  "tests": [$TESTS_JSON]
}
EOF

if [[ "$JSON_OUTPUT" == "1" ]]; then
  cat "$REPORT_FILE"
else
  echo "=== SDP Smoke Tests ==="
  echo "  Status:  $STATUS"
  echo "  Pass:    $PASS_COUNT"
  echo "  Fail:    $FAIL_COUNT"
  echo "  Skip:    $SKIP_COUNT"
  echo ""
  if [[ "$FAIL_COUNT" -gt 0 ]]; then
    echo "FAILURES:"
    grep "^--- FAIL:" /tmp/sdp-smoke-stdout.txt || true
  fi
fi

exit "$SMOKE_EXIT"
