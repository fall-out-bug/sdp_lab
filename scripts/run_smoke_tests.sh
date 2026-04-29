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
RAW_FILE="$(mktemp /tmp/sdp-smoke-raw-XXXXXX.json)"
trap 'rm -f "$REPORT_FILE" "$RAW_FILE"' EXIT

START_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Run smoke tests once and parse the JSON event stream.
set +e
go test -tags=smoke ./test/smoke/... -v -json > "$RAW_FILE" 2>&1
SMOKE_EXIT=$?
set -e
if [[ "$JSON_OUTPUT" != "1" ]]; then
  cat "$RAW_FILE"
fi

END_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)

START_TS="$START_TS" END_TS="$END_TS" RAW_FILE="$RAW_FILE" python3 - <<'PY' > "$REPORT_FILE"
import json
import os

tests = []
counts = {"pass": 0, "fail": 0, "skip": 0}
with open(os.environ["RAW_FILE"], "r", encoding="utf-8") as f:
    for line in f:
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        action = event.get("Action")
        name = event.get("Test")
        if action not in {"pass", "fail", "skip"} or not name:
            continue
        counts[action] += 1
        tests.append({
            "name": name,
            "status": action.upper(),
            "duration": event.get("Elapsed", 0),
            "package": event.get("Package", ""),
        })

status = "fail" if counts["fail"] else "pass"
json.dump({
    "runner": "sdp-smoke",
    "started_at": os.environ["START_TS"],
    "finished_at": os.environ["END_TS"],
    "status": status,
    "summary": {
        "total": sum(counts.values()),
        "pass": counts["pass"],
        "fail": counts["fail"],
        "skip": counts["skip"],
    },
    "tests": tests,
}, fp=os.sys.stdout, indent=2)
print()
PY

PASS_COUNT=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["summary"]["pass"])' "$REPORT_FILE")
FAIL_COUNT=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["summary"]["fail"])' "$REPORT_FILE")
SKIP_COUNT=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["summary"]["skip"])' "$REPORT_FILE")
STATUS=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["status"])' "$REPORT_FILE")

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
    python3 -c 'import json,sys; [print(t["name"]) for t in json.load(open(sys.argv[1]))["tests"] if t["status"] == "FAIL"]' "$REPORT_FILE"
  fi
fi

exit "$SMOKE_EXIT"
